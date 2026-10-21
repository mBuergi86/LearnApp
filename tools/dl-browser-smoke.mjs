import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import { mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const baseURL = process.env.SMOKE_BASE_URL || 'http://127.0.0.1:18080/';
const chromePath = process.env.CHROME_PATH;
const username = process.env.SMOKE_USERNAME;
const password = process.env.SMOKE_PASSWORD;
const debugPort = 9225;
const localPreview = process.env.SMOKE_LOCAL_PREVIEW === '1';
const checksOnly = process.env.SMOKE_CHECKS_ONLY === '1';
const securityErrors=[],externalRequests=[];

if (!chromePath || (!localPreview && (!username || !password))) {
  throw new Error('CHROME_PATH, SMOKE_USERNAME and SMOKE_PASSWORD are required');
}

const profile = join(tmpdir(), `mldl-dl-review-chrome-${process.pid}`);
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
      if (!message.id) {
        if(message.method==='Log.entryAdded' && message.params.entry.source==='security') securityErrors.push(message.params.entry.text);
        if(message.method==='Network.requestWillBeSent') {
          const url=message.params.request.url;
          if(/^https?:/.test(url) && new URL(url).origin!==new URL(baseURL).origin) externalRequests.push(url);
        }
        return;
      }
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
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description || result.exceptionDetails.text);
  return result.result.value;
}

async function waitFor(client, expression, description) {
  for (let attempt = 0; attempt < 200; attempt += 1) {
    if (await evaluate(client, expression)) return;
    await sleep(50);
  }
  const state = await evaluate(client, "({route:location.hash,auth:document.body.dataset.authState,heading:document.querySelector('h1')?.textContent,message:document.querySelector('#main-content')?.textContent.slice(0,300),checkFeedback:document.querySelector('.topic-check-feedback')?.textContent,choices:[...document.querySelectorAll('.topic-check-card input')].map(i=>({value:i.value,checked:i.checked})),checkBusy:document.querySelector('.topic-check-card')?.dataset.busy})");
  throw new Error(`Timed out waiting for ${description}: ${JSON.stringify(state)}`);
}

let client;
try {
  await waitForDebugging();
  const target = await (await fetch(`http://127.0.0.1:${debugPort}/json/new?${encodeURIComponent('about:blank')}`, {method:'PUT'})).json();
  client = new CDP(target.webSocketDebuggerUrl);
  await client.open();
  await client.call('Page.enable');
  await client.call('Runtime.enable');
  await client.call('Log.enable');
  await client.call('Network.enable');
  await client.call('Page.navigate',{url:baseURL});
  if (!localPreview) {
    await waitFor(client, "!!document.querySelector('#login-form')", 'login form');
    await evaluate(client, `(() => {
    document.querySelector('#login-username').value=${JSON.stringify(username)};
    document.querySelector('#login-password').value=${JSON.stringify(password)};
    document.querySelector('#login-form').requestSubmit(); return true;
    })()`);
  } else {
    assert.equal(new URL(baseURL).hostname,'127.0.0.1','preview smoke must stay local');
    const status=await(await fetch(new URL('/api/auth/status',baseURL))).json();
    assert.equal(status.local_preview,true,'only the explicit dev entrypoint may skip login');
  }
  await waitFor(client, "document.body.dataset.authState === 'authenticated'", 'authenticated');
  const topics = await evaluate(client, "(async()=> (await (await fetch('/api/catalog')).json()).topics.map(t=>({id:t.id,title:t.title,module_id:t.module_id,exam_id:t.exam_id,hasFormula:!!t.formula})))()");
  assert.equal(topics.length,58);
  const problems=[], screenshots=[];
  let pages=0, figures=0,formulaStates=0,mathExpressions=0;
  for (const width of checksOnly?[]:[320,375,390,1024,1440]) {
    await client.call('Emulation.setDeviceMetricsOverride',{width,height:1000,deviceScaleFactor:1,mobile:width<800});
    for (const theme of ['light','dark']) {
      await evaluate(client, `renderTheme('${theme}')`);
      for (const topic of topics) {
        const siblings=topics.filter(t=>t.module_id===topic.module_id && t.exam_id===topic.exam_id);
        const index=siblings.findIndex(t=>t.id===topic.id);
        // Drive navigation through the browser. Hundreds of renderer-initiated
        // hash assignments can trigger Chromium's IPC flood protection, which
        // silently ignores subsequent assignments during this stress test.
        await client.call('Page.navigate', {url: `${baseURL}#/topic/${topic.id}`});
        await waitFor(client, `document.querySelector('.learning-header h1')?.textContent === ${JSON.stringify(topic.title)}`,topic.id);
        await waitFor(client,"!!document.querySelector('.topic-check-card')",'topic comprehension checks');
        await evaluate(client, 'document.fonts.ready');
        const state=await evaluate(client, `(() => {
          const svgs=[...document.querySelectorAll('.learning-figure svg')];
          const outside=[];
          for(const svg of svgs) {
            const view=svg.viewBox.baseVal;
            for(const text of svg.querySelectorAll('text')) {
              const r=text.getBBox();
              if(r.x < -1 || r.x+r.width > view.width+1 || r.y < -1 || r.y+r.height > view.height+1) outside.push(text.textContent);
            }
          }
          const nav=document.querySelector('.topic-navigation');
          scrollTo({top:document.documentElement.scrollHeight,behavior:'instant'});
          const bottom=document.querySelector('.bottom-nav');
          return {figures:svgs.length,overflow:document.documentElement.scrollWidth>innerWidth,outside,paragraphs:document.querySelectorAll('.explanation-list p').length,example:!!document.querySelector('.learning-example')?.textContent.trim(),
            counter:document.querySelector('.learning-header .topic-sequence')?.textContent,
            previous:nav.querySelector('[data-topic-previous]').getAttribute('href'),
            next:nav.querySelector('[data-topic-next]').getAttribute('href'),
            training:nav.querySelector('[data-topic-training]')?.getAttribute('href') || null,
            checks:[...document.querySelectorAll('.topic-check-card')].map(f=>f.dataset.checkId),
            checkTargetHeights:[...document.querySelectorAll('.topic-check-options label')].map(l=>l.getBoundingClientRect().height),
            moduleTraining:!!document.querySelector('.test-placeholder'),
            targetHeights:[...nav.querySelectorAll('a')].map(a=>a.getBoundingClientRect().height),
            position:getComputedStyle(nav).position,
            afterSources:!!(document.querySelector('.sources').compareDocumentPosition(nav)&Node.DOCUMENT_POSITION_FOLLOWING),
            aboveBottomNav:getComputedStyle(bottom).display==='none'||nav.getBoundingClientRect().bottom<=bottom.getBoundingClientRect().top,
            formulaExamples:document.querySelectorAll('.formula-worked-example').length,
            formulaSteps:document.querySelectorAll('.formula-example-steps li').length,
            mathCount:document.querySelectorAll('.formula-card math').length,
            formulaErrors:document.querySelectorAll('.formula-render-error').length,
            inlineMathStyles:document.querySelectorAll('.formula-card [style]').length,
            mathNormalWeight:[...document.querySelectorAll('.formula-card math')].every(m=>getComputedStyle(m).fontWeight==='400'),
            nativeFractionLayout:[...document.querySelectorAll('.formula-card mfrac')].every(f=>f.children.length===2 && f.children[0].getBoundingClientRect().y<f.children[1].getBoundingClientRect().y),
            formulasNotVerticallyClipped:[...document.querySelectorAll('.formula-equation')].every(b=>b.scrollHeight<=b.clientHeight+2)};
        })()`);
        assert.equal(state.figures,2,topic.id); assert.equal(state.overflow,false,topic.id); assert.equal(state.example,true,topic.id); assert.ok(state.paragraphs>=5,topic.id);
        assert.equal(state.counter,`Thema ${index+1} / ${siblings.length}`,topic.id);
        assert.equal(state.previous,index>0?`#/topic/${siblings[index-1].id}`:`#/module/${topic.module_id}`,topic.id);
        assert.equal(state.next,index+1<siblings.length?`#/topic/${siblings[index+1].id}`:`#/exam/${topic.exam_id}`,topic.id);
        assert.equal(state.training,index+1===siblings.length?`#/quiz/${topic.exam_id}/${topic.module_id}`:null,topic.id);
        assert.equal(state.moduleTraining,index+1===siblings.length,topic.id);
        assert.ok(state.checks.length>0,topic.id);
        assert.ok(state.checkTargetHeights.every(h=>h>=44),topic.id);
        assert.ok(state.targetHeights.every(h=>h>=44),topic.id);
        assert.equal(state.position,'static',topic.id);
        assert.equal(state.afterSources,true,topic.id);
        assert.equal(state.aboveBottomNav,true,topic.id);
        assert.equal(state.formulaErrors,0,topic.id);
        assert.equal(state.inlineMathStyles,0,topic.id);
        if(topic.hasFormula) {
          assert.equal(state.formulaExamples,1,topic.id);
          assert.ok(state.formulaSteps>=3,topic.id);
          assert.ok(state.mathCount>=5,topic.id);
          assert.equal(state.mathNormalWeight,true,topic.id);
          assert.equal(state.nativeFractionLayout,true,topic.id);
          assert.equal(state.formulasNotVerticallyClipped,true,topic.id);
          formulaStates++;mathExpressions+=state.mathCount;
        } else assert.equal(state.mathCount,0,topic.id);
        if(state.outside.length) problems.push({topic:topic.id,width,theme,outside:state.outside});
        pages++; figures+=state.figures;
        if(width===390 && ((theme==='light' && topic.id==='classification-threshold') || (theme==='dark' && topic.id==='activation-functions'))) {
          const clip=await evaluate(client,"(() => {const r=document.querySelectorAll('.learning-figure')[1].getBoundingClientRect();return {x:r.x+scrollX,y:r.y+scrollY,width:r.width,height:r.height,scale:1}})()");
          const shot=await client.call('Page.captureScreenshot',{format:'png',captureBeyondViewport:true,clip});
          const path=join(tmpdir(),`dl-review-${topic.id}-${theme}.png`);
          await writeFile(path,Buffer.from(shot.data,'base64')); screenshots.push(path);
        }
        if(width===390 && ((theme==='light' && topic.id==='artificial-neuron') || (theme==='dark' && topic.id==='logistic-regression-model'))) {
          const clip=await evaluate(client,"(() => {const r=document.querySelector('.formula-card').getBoundingClientRect();return {x:r.x+scrollX,y:r.y+scrollY,width:r.width,height:r.height,scale:1}})()");
          const shot=await client.call('Page.captureScreenshot',{format:'png',captureBeyondViewport:true,clip});
          const path=join(tmpdir(),`formula-v170-${topic.id}-${theme}.png`);
          await writeFile(path,Buffer.from(shot.data,'base64'));screenshots.push(path);
        }
      }
    }
  }
  assert.deepEqual(problems,[],'SVG text extends outside viewBox');
  const navigationWalkthroughs=[];
  const pressKey=async(key,code,keyCode,modifiers=0)=>{
    await client.call('Input.dispatchKeyEvent',{type:'keyDown',key,code,windowsVirtualKeyCode:keyCode,modifiers,...(key==='Enter'?{text:'\r'}:{})});
    await client.call('Input.dispatchKeyEvent',{type:'keyUp',key,code,windowsVirtualKeyCode:keyCode,modifiers});
  };
  const clickBottom=async(selector)=>{
    await waitFor(client,"!!document.querySelector('.topic-check-card')",'checks loaded before bottom navigation click');
    const point=await evaluate(client,`(() => {scrollTo({top:document.documentElement.scrollHeight,behavior:'instant'});const r=document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect();return {x:r.x+r.width/2,y:r.y+r.height/2}})()`);
    await client.call('Input.dispatchMouseEvent',{type:'mousePressed',button:'left',clickCount:1,...point});
    await client.call('Input.dispatchMouseEvent',{type:'mouseReleased',button:'left',clickCount:1,...point});
  };
  for(const width of checksOnly?[]:[390,1440]) {
    await client.call('Emulation.setDeviceMetricsOverride',{width,height:1000,deviceScaleFactor:1,mobile:width<800});
    await evaluate(client,`renderTheme('${width===390?'light':'dark'}')`);
    for(const exam of ['machine-learning','deep-learning-data-science']) {
      const first=topics.find(t=>t.exam_id===exam && topics.filter(s=>s.module_id===t.module_id).length===3);
      assert.ok(first,'three-topic module exists in '+exam);
      const siblings=topics.filter(t=>t.module_id===first.module_id);
      await client.call('Page.navigate',{url:`${baseURL}#/topic/${siblings[0].id}`});
      await waitFor(client,`document.querySelector('.learning-header .topic-sequence')?.textContent==='Thema 1 / 3' && location.hash==='#/topic/${siblings[0].id}'`,'first topic');
      await clickBottom('[data-topic-next]');
      await waitFor(client,`document.querySelector('.learning-header .topic-sequence')?.textContent==='Thema 2 / 3'`,'second topic');
      await clickBottom('[data-topic-previous]');
      await waitFor(client,`document.querySelector('.learning-header .topic-sequence')?.textContent==='Thema 1 / 3'`,'back to first');
      await clickBottom('[data-topic-next]');
      await waitFor(client,`document.querySelector('.learning-header .topic-sequence')?.textContent==='Thema 2 / 3'`,'second again');
      await clickBottom('[data-topic-next]');
      await waitFor(client,`document.querySelector('.learning-header .topic-sequence')?.textContent==='Thema 3 / 3'`,'last topic');
      if(width===390 && exam==='machine-learning') {
        const clip=await evaluate(client,"(() => {const r=document.querySelector('.topic-navigation').getBoundingClientRect();return {x:r.x+scrollX,y:r.y+scrollY,width:r.width,height:r.height,scale:1}})()");
        const shot=await client.call('Page.captureScreenshot',{format:'png',captureBeyondViewport:true,clip});
        const path=join(tmpdir(),'topic-navigation-v170-mobile.png');
        await writeFile(path,Buffer.from(shot.data,'base64'));screenshots.push(path);
      }
      await clickBottom('[data-topic-next]');
      await waitFor(client,`location.hash==='#/exam/${exam}' && !!document.querySelector('.module-list')`,'exam area');
      await client.call('Page.navigate',{url:`${baseURL}#/topic/${siblings[2].id}`});
      await waitFor(client,`!!document.querySelector('[data-topic-training]')`,'training link');
      await evaluate(client,"document.querySelector('[data-topic-previous]').focus()");
      await pressKey('Tab','Tab',9);
      assert.equal(await evaluate(client,"document.activeElement.hasAttribute('data-topic-training')"),true);
      assert.equal(await evaluate(client,"document.activeElement.matches(':focus-visible') && getComputedStyle(document.activeElement).outlineStyle!=='none'"),true);
      await pressKey('Tab','Tab',9);
      assert.equal(await evaluate(client,"document.activeElement.hasAttribute('data-topic-next')"),true);
      await pressKey('Tab','Tab',9,8);
      assert.equal(await evaluate(client,"document.activeElement.hasAttribute('data-topic-training')"),true);
      await pressKey('Enter','Enter',13);
      await waitFor(client,`location.hash==='#/quiz/${exam}/${first.module_id}' && !!document.querySelector('.question-card')`,'module training via keyboard');
      navigationWalkthroughs.push({width,exam,first:true,middle:true,last:true,back:true,examReturn:true,training:true,keyboard:true});
    }
  }
  await client.call('Page.navigate',{url:`${baseURL}#/topic/artificial-neuron`});
  await waitFor(client,"!!document.querySelector('.topic-check-card[data-check-id=\"dl-artificial-neuron-fact\"]')",'neuron comprehension check');
  const beforeChecks=await evaluate(client,"(async()=> (await requestJSON('/api/backup')).data)()");
  for(const choice of ['false','true']) {
    await evaluate(client,`document.querySelector('.topic-check-card input[value="${choice}"]').focus()`);
    await pressKey(' ','Space',32);
    await evaluate(client,"document.querySelector('.topic-check-card button[type=submit]').focus()");
    await pressKey('Enter','Enter',13);
    await waitFor(client,`!!document.querySelector('.topic-check-feedback .answer-feedback.${choice==='true'?'correct':'wrong'}')`,'comprehension answer '+choice);
    const feedback=await evaluate(client,"({explanation:document.querySelector('.topic-check-feedback').textContent,focused:document.activeElement.matches('.topic-check-feedback .answer-feedback')})");
    assert.ok(feedback.explanation.includes('Warum – und warum nicht anders?'));
    assert.ok(feedback.explanation.includes('Die Aussage ist: Richtig'));
    assert.equal(feedback.focused,true);
    await evaluate(client,"document.querySelector('.check-retry').click()");
    assert.equal(await evaluate(client,"document.querySelector('.topic-check-card fieldset').disabled"),false);
  }
  const afterChecks=await evaluate(client,"(async()=> (await requestJSON('/api/backup')).data)()");
  assert.deepEqual(afterChecks,beforeChecks,'short checks must not change learning state');
  await evaluate(client,"location.hash='#/'");
  await waitFor(client,"!!document.querySelector('.exam-grid')",'home');
  await evaluate(client, `(async()=>{
    const plan=await requestJSON('/api/today?exam=deep-learning-data-science&minutes=45');
    await startTodaySession(findExam('deep-learning-data-science'),plan); return true;
  })()`);
  const todayFigures=await evaluate(client,"document.querySelectorAll('.session-learning .learning-figure').length");
  assert.equal(todayFigures,2,'Today introduces new topics with their figures');
  await evaluate(client,"quizSession.newTopicIDs=['artificial-neuron'];quizSession.newTopicIndex=0;renderTodayLearningStep()");
  const todayFormulaExample=await evaluate(client,"document.querySelectorAll('.session-learning .formula-worked-example').length");
  assert.equal(todayFormulaExample,1);
  assert.deepEqual(securityErrors,[],'CSP errors');
  assert.deepEqual(externalRequests,[],'external page runtime requests');
  console.log(JSON.stringify({version:'1.8.1',pages,figures,topics:58,formulaStates,mathExpressions,nativeMathML:true,normalMathWeight:true,noCSPWeakening:true,noExternalRuntimeRequests:true,mobile:true,desktop:true,light:true,dark:true,todayFigures,todayFormulaExample,shortCheckCorrectAndWrong:true,shortCheckKeyboard:true,shortCheckNoLearningWrites:true,navigationWalkthroughs,screenshots}));
} finally {
  if(client)client.close(); chrome.kill(); await sleep(100);
  await rm(profile,{recursive:true,force:true});
}
