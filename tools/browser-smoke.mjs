import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const baseURL = process.env.SMOKE_BASE_URL || 'http://127.0.0.1:18080/';
const chromePath = process.env.CHROME_PATH;
const username = process.env.SMOKE_USERNAME;
const password = process.env.SMOKE_PASSWORD;
const debugPort = 9224;

if (!chromePath || !username || !password) {
  throw new Error('CHROME_PATH, SMOKE_USERNAME and SMOKE_PASSWORD are required');
}

const profile = join(tmpdir(), `mldl-v170-chrome-${process.pid}`);
await mkdir(profile, { recursive: true });
const chrome = spawn(chromePath, [
  '--headless=new', '--disable-gpu', '--no-first-run', '--no-default-browser-check',
  `--remote-debugging-port=${debugPort}`, `--user-data-dir=${profile}`, '--window-size=1440,1000', 'about:blank',
], { stdio: 'ignore' });

const sleep = (milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds));

class CDP {
  constructor(url) {
    this.socket = new WebSocket(url);
    this.nextID = 1;
    this.pending = new Map();
  }
  async open() {
    if (this.socket.readyState === WebSocket.OPEN) return;
    await new Promise((resolve, reject) => {
      this.socket.addEventListener('open', resolve, { once: true });
      this.socket.addEventListener('error', reject, { once: true });
    });
    this.socket.addEventListener('message', (event) => {
      const message = JSON.parse(event.data);
      if (!message.id) return;
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      if (message.error) pending.reject(new Error(message.error.message));
      else pending.resolve(message.result);
    });
  }
  call(method, params = {}) {
    const id = this.nextID++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.socket.send(JSON.stringify({ id, method, params }));
    });
  }
  close() { this.socket.close(); }
}

async function waitForDebugging() {
  for (let attempt = 0; attempt < 100; attempt += 1) {
    try {
      const response = await fetch(`http://127.0.0.1:${debugPort}/json/version`);
      if (response.ok) return;
    } catch {}
    await sleep(50);
  }
  throw new Error('Chrome DevTools endpoint did not start');
}

async function evaluate(client, expression) {
  const result = await client.call('Runtime.evaluate', { expression, awaitPromise: true, returnByValue: true });
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text);
  return result.result.value;
}

async function waitFor(client, expression, description) {
  for (let attempt = 0; attempt < 200; attempt += 1) {
    if (await evaluate(client, expression)) return;
    await sleep(50);
  }
  throw new Error(`Timed out waiting for ${description}`);
}

async function selectTheme(client, value) {
  if (await evaluate(client, `document.documentElement.dataset.themePreference === '${value}'`)) return;
  await evaluate(client, `(() => { const input = document.querySelector('input[name="settings-theme"][value="${value}"]'); input.click(); return true; })()`);
  await waitFor(client, `document.documentElement.dataset.themePreference === '${value}' && document.querySelector('#global-status').textContent === 'Darstellung gespeichert.'`, `${value} theme persistence`);
  await waitFor(client, `(async()=> (await (await fetch('/api/settings/theme')).json()).theme === '${value}')()`, `${value} server-side theme persistence`);
}

async function footerState(client) {
  return evaluate(client, `(() => {
    const footer = document.querySelector('#app-footer');
    const version = document.querySelector('#app-version');
    const style = getComputedStyle(footer);
    const rect = version.getBoundingClientRect();
    return { text: version.textContent.trim(), visible: style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0, position: style.position, color: style.color, fontSize: style.fontSize, top: rect.top, bottom: rect.bottom };
  })()`);
}

let client;
try {
  await waitForDebugging();
  const targetResponse = await fetch(`http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent('about:blank')}`, { method: 'PUT' });
  const target = await targetResponse.json();
  client = new CDP(target.webSocketDebuggerUrl);
  await client.open();
  await client.call('Page.enable');
  await client.call('Runtime.enable');
  await client.call('Page.addScriptToEvaluateOnNewDocument', { source: `
    globalThis.__authFrames = [];
    addEventListener('DOMContentLoaded', () => {
      const sample = () => {
        const state = document.body?.dataset.authState || 'missing';
        const shell = document.querySelector('.sidebar');
        const dashboard = document.querySelector('.exam-grid');
        globalThis.__authFrames.push({ state, shellVisible: !!shell && getComputedStyle(shell).display !== 'none', dashboardVisible: !!dashboard && getComputedStyle(dashboard).visibility !== 'hidden' });
        if (globalThis.__authFrames.length < 180 && state === 'unknown') requestAnimationFrame(sample);
      };
      sample();
    });
  ` });

  await client.call('Page.navigate', { url: baseURL });
  await waitFor(client, `document.body?.dataset.authState === 'unauthenticated' && !!document.querySelector('#login-form')`, 'stable unauthenticated login');
  const initial = await evaluate(client, `({ state: document.body.dataset.authState, dashboard: !!document.querySelector('.exam-grid'), frames: globalThis.__authFrames })`);
  assert.equal(initial.state, 'unauthenticated');
  assert.equal(initial.dashboard, false);
  assert.equal(initial.frames.some((frame) => frame.state === 'unknown' && (frame.shellVisible || frame.dashboardVisible)), false);
  const loginFooter = await footerState(client);
  assert.equal(loginFooter.text, 'ML/DL Exam Trainer · v1.8.2');
  assert.equal(loginFooter.visible, true);
  assert.equal(loginFooter.position, 'static');

  await evaluate(client, `(() => {
    document.querySelector('#login-username').value = ${JSON.stringify(username)};
    document.querySelector('#login-password').value = ${JSON.stringify(password)};
    document.querySelector('#login-form').requestSubmit();
    return true;
  })()`);
  await waitFor(client, `document.body.dataset.authState === 'authenticated' && !!document.querySelector('.exam-grid')`, 'dashboard after login');
  const appFooter = await footerState(client);
  assert.equal(appFooter.text, 'ML/DL Exam Trainer · v1.8.2');
  assert.equal(appFooter.visible, true);
  assert.equal(appFooter.position, 'static');

  await client.call('Page.reload', { ignoreCache: true });
  await waitFor(client, `document.body?.dataset.authState === 'authenticated' && !!document.querySelector('.exam-grid')`, 'dashboard after session reload');

  await evaluate(client, `location.hash = '#/settings'`);
  await waitFor(client, `!!document.querySelector('#mobile-logout-button')`, 'settings page');
  await client.call('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'light' }] });
  await selectTheme(client, 'system');
  await waitFor(client, `document.documentElement.dataset.theme === 'light'`, 'system light theme');
  await client.call('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'dark' }] });
  await waitFor(client, `document.documentElement.dataset.theme === 'dark'`, 'system theme OS change');
  await selectTheme(client, 'light');
  assert.equal((await footerState(client)).visible, true);
  await client.call('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'dark' }] });
  assert.equal(await evaluate(client, `document.documentElement.dataset.theme`), 'light');
  await selectTheme(client, 'dark');
  assert.equal((await footerState(client)).visible, true);
  await client.call('Emulation.setEmulatedMedia', { features: [{ name: 'prefers-color-scheme', value: 'light' }] });
  assert.equal(await evaluate(client, `document.documentElement.dataset.theme`), 'dark');
  await selectTheme(client, 'system');

  await client.call('Page.reload', { ignoreCache: true });
  await waitFor(client, `document.body?.dataset.authState === 'authenticated' && document.documentElement.dataset.themePreference === 'system'`, 'persisted system theme after reload');

  const desktopChecks = {};
  for (const width of [1024, 1440]) {
    await client.call('Emulation.setDeviceMetricsOverride', { width, height: 1000, deviceScaleFactor: 1, mobile: false });
    desktopChecks[width] = await evaluate(client, `(() => {
      const logout = document.querySelector('#logout-button');
      const themes = document.querySelector('.topbar .theme-switcher');
      const footer = document.querySelector('#app-footer');
      return { noOverflow: document.documentElement.scrollWidth <= innerWidth, logoutVisible: getComputedStyle(logout).display !== 'none', themesVisible: getComputedStyle(themes).display !== 'none', logoutHeight: logout.getBoundingClientRect().height, footerVisible: getComputedStyle(footer).display !== 'none' };
    })()`);
    assert.equal(desktopChecks[width].noOverflow, true);
    assert.equal(desktopChecks[width].logoutVisible, true);
    assert.equal(desktopChecks[width].themesVisible, true);
    assert.equal(desktopChecks[width].footerVisible, true);
    assert.ok(desktopChecks[width].logoutHeight >= 44);
  }

  const mobileChecks = {};
  for (const width of [320, 375, 390]) {
    await client.call('Emulation.setDeviceMetricsOverride', { width, height: 844, deviceScaleFactor: 1, mobile: true });
    await evaluate(client, `location.hash = '#/settings'`);
    await waitFor(client, `!!document.querySelector('#mobile-logout-button')`, `${width}px settings`);
    await evaluate(client, `scrollTo(0, document.documentElement.scrollHeight)`);
    mobileChecks[width] = await evaluate(client, `(() => {
      const logout = document.querySelector('#mobile-logout-button');
      const options = [...document.querySelectorAll('.theme-options span')];
      const version = document.querySelector('#app-version').getBoundingClientRect();
      const navigation = document.querySelector('.bottom-nav').getBoundingClientRect();
      return { noOverflow: document.documentElement.scrollWidth <= innerWidth, logoutVisible: getComputedStyle(logout).display !== 'none', logoutHeight: logout.getBoundingClientRect().height, optionHeights: options.map((item) => item.getBoundingClientRect().height), footerVisible: version.width > 0 && version.height > 0, footerAboveNavigation: version.bottom <= navigation.top };
    })()`);
    assert.equal(mobileChecks[width].noOverflow, true);
    assert.equal(mobileChecks[width].logoutVisible, true);
    assert.equal(mobileChecks[width].footerVisible, true);
    assert.equal(mobileChecks[width].footerAboveNavigation, true);
    assert.ok(mobileChecks[width].logoutHeight >= 44);
    assert.equal(mobileChecks[width].optionHeights.every((height) => height >= 44), true);
  }

  await client.call('Emulation.setDeviceMetricsOverride', { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
  const screenshot = await client.call('Page.captureScreenshot', { format: 'png', fromSurface: true });
  const screenshotPath = join(tmpdir(), 'mldl-v170-mobile-footer.png');
  await writeFile(screenshotPath, Buffer.from(screenshot.data, 'base64'));

  await evaluate(client, `document.querySelector('#mobile-logout-button').click()`);
  await waitFor(client, `document.body.dataset.authState === 'unauthenticated' && !!document.querySelector('#login-form')`, 'login after mobile logout');
  await client.call('Page.reload', { ignoreCache: true });
  await waitFor(client, `document.body?.dataset.authState === 'unauthenticated' && !!document.querySelector('#login-form')`, 'login after logout reload');

  await evaluate(client, `location.hash = '#/'`);
  await evaluate(client, `(() => {
    document.querySelector('#login-username').value = ${JSON.stringify(username)};
    document.querySelector('#login-password').value = ${JSON.stringify(password)};
    document.querySelector('#login-form').requestSubmit();
    return true;
  })()`);
  await waitFor(client, `document.body.dataset.authState === 'authenticated' && !!document.querySelector('.exam-grid')`, 'dashboard before desktop logout');
  await client.call('Emulation.setDeviceMetricsOverride', { width: 1440, height: 1000, deviceScaleFactor: 1, mobile: false });
  await evaluate(client, `document.querySelector('#logout-button').click()`);
  await waitFor(client, `document.body.dataset.authState === 'unauthenticated' && !!document.querySelector('#login-form')`, 'login after desktop logout');
  await client.call('Page.reload', { ignoreCache: true });
  await waitFor(client, `document.body?.dataset.authState === 'unauthenticated'`, 'login after desktop logout reload');

  console.log(JSON.stringify({ footerVersion: '1.8.2', footerLogin: true, footerApp: true, footerLight: true, footerDark: true, footerNormalFlow: true, initialAuthNoFlicker: true, login: true, sessionReload: true, themeSystem: true, themeLight: true, themeDark: true, themeReload: true, osThemeChange: true, desktop: desktopChecks, mobile: mobileChecks, mobileLogoutReload: true, desktopLogoutReload: true, screenshot: screenshotPath }));
} finally {
  if (client) client.close();
  chrome.kill();
  await sleep(100);
  await rm(profile, { recursive: true, force: true });
}
