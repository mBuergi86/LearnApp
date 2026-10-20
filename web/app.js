const root = document.documentElement;
const app = document.querySelector('#app');
const main = document.querySelector('#main-content');
const breadcrumb = document.querySelector('#breadcrumb');
const globalStatus = document.querySelector('#global-status');
const appVersion = document.querySelector('#app-version');
const themeButtons = [...document.querySelectorAll('[data-theme-choice]')];
const systemTheme = window.matchMedia('(prefers-color-scheme: dark)');
const { render: renderLearningVisual } = globalThis.ExamTrainerLearningVisuals;
const { render: renderFormula, inline: richText, plain: plainText } = globalThis.ExamTrainerFormulaRenderer;
const { APP_VERSION, AUTH, authTransition, normalizeTheme, resolveTheme, topicNavigation } = globalThis.ExamTrainerUIState;
let catalog;
let quizSession;
let examSession;
let authState = AUTH.UNKNOWN;

appVersion.textContent = `ML/DL Exam Trainer · v${APP_VERSION}`;

const escapeHTML = (value = '') => String(value).replace(/[&<>'"]/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' })[character]);
const findExam = (id) => catalog.exams.find((exam) => exam.id === id);
const findModule = (id) => catalog.modules.find((module) => module.id === id);
const findTopic = (id) => catalog.topics.find((topic) => topic.id === id);
const modulesFor = (examID) => catalog.modules.filter((module) => module.exam_id === examID);
const topicsFor = (moduleID) => catalog.topics.filter((topic) => topic.module_id === moduleID);

async function requestJSON(url, options = {}) {
  const method = (options.method || 'GET').toUpperCase();
  if (method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS' && url !== '/api/auth/login') {
    options.headers = { ...(options.headers || {}), 'X-CSRF-Token': csrfToken() };
  }
  let response;
  try {
    response = await fetch(url, options);
  } catch {
    throw new Error('Der lokale Server ist nicht erreichbar. Bitte prüfe, ob der Exam Trainer läuft.');
  }
  let data;
  try {
    data = await response.json();
  } catch {
    throw new Error('Der Server hat eine ungültige Antwort geliefert.');
  }
  if (!response.ok) {
    if (response.status === 401 && authState === AUTH.AUTHENTICATED) sessionExpired();
    throw new Error(data.error || `Anfrage fehlgeschlagen (${response.status}).`);
  }
  return data;
}

function csrfToken() {
  const prefix = 'exam_trainer_csrf=';
  return document.cookie.split('; ').find((item) => item.startsWith(prefix))?.slice(prefix.length) || '';
}
function setAuthState(nextState) {
  authState = nextState;
  document.body.dataset.authState = nextState;
}
function renderAuthLoading() {
  setAuthState(authTransition(authState, 'check-started'));
  breadcrumb.textContent = 'Sitzung wird geprüft';
  app.innerHTML = '<section class="auth-loading" role="status" aria-live="polite"><span class="loading-mark" aria-hidden="true"></span><p>Sitzung wird geprüft …</p></section>';
}
function renderLogin(message = '') {
  setAuthState(authTransition(authState, 'session-invalid')); breadcrumb.textContent = 'Anmelden';
  app.innerHTML = `<section class="login-panel" aria-labelledby="login-title"><span class="eyebrow">Geschützter Lernbereich</span><h1 id="login-title">Anmelden</h1><p>Bitte melde dich an, um auf deinen Lernstand und die Kursunterlagen zuzugreifen.</p><form id="login-form" class="login-form"><label for="login-username">Benutzername</label><input id="login-username" name="username" autocomplete="username" required maxlength="128"><label for="login-password">Passwort</label><input id="login-password" name="password" type="password" autocomplete="current-password" required><p id="login-error" class="login-error" role="alert">${escapeHTML(message)}</p><button class="primary-action" type="submit">Anmelden</button></form></section>`;
  document.querySelector('#login-form').addEventListener('submit', async (event) => {
    event.preventDefault(); const form = event.currentTarget; const button = form.querySelector('button'); const error = document.querySelector('#login-error'); button.disabled = true; error.textContent = '';
    let loginSucceeded = false;
    try {
      await requestJSON('/api/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ username: form.username.value, password: form.password.value }) });
      loginSucceeded = true; form.password.value = ''; await startApplication('login-succeeded');
    } catch (loginError) {
      if (loginSucceeded) { setAuthState(AUTH.AUTHENTICATED); errorView(loginError, '#/'); return; }
      error.textContent = loginError.message; form.password.focus();
    }
    finally { button.disabled = false; }
  });
  document.querySelector('#login-username').focus();
}
function sessionExpired() { if (authState === AUTH.AUTHENTICATED) renderLogin('Deine Sitzung ist abgelaufen. Bitte melde dich erneut an.'); }
async function startApplication(successEvent = 'session-valid') {
  renderAuthLoading();
  const [loadedCatalog, settings] = await Promise.all([requestJSON('/api/catalog'), requestJSON('/api/settings/theme')]);
  catalog = loadedCatalog; renderTheme(settings.theme); await route();
  setAuthState(authTransition(authState, successEvent));
}

function announce(message) {
  globalStatus.textContent = '';
  requestAnimationFrame(() => { globalStatus.textContent = message; });
}

function focusHeading() {
  const heading = app.querySelector('h1');
  if (!heading) { main.focus({ preventScroll: true }); return; }
  heading.setAttribute('tabindex', '-1');
  heading.focus({ preventScroll: true });
}

function errorView(error, retryHref = location.hash || '#/') {
  breadcrumb.textContent = 'Fehler';
  app.innerHTML = `<section class="empty-state" role="alert"><h1>Inhalt momentan nicht verfügbar</h1><p>${escapeHTML(error.message || 'Ein unerwarteter Fehler ist aufgetreten.')}</p><a class="primary-action" href="${escapeHTML(retryHref)}">Erneut versuchen</a></section>`;
  focusHeading();
}

function appliedTheme(preference) { return resolveTheme(preference, systemTheme.matches); }
function renderTheme(preference) {
  const normalized = normalizeTheme(preference);
  root.dataset.theme = appliedTheme(normalized);
  root.dataset.themePreference = normalized;
  document.querySelectorAll('[data-theme-choice]').forEach((control) => {
    const selected = control.dataset.themeChoice === normalized;
    if (control.matches('button')) control.setAttribute('aria-pressed', String(selected));
    if (control.matches('input')) control.checked = selected;
  });
}
async function saveTheme(theme) {
  renderTheme(theme);
  globalStatus.textContent = 'Darstellung wird gespeichert …';
  try {
    await requestJSON('/api/settings/theme', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ theme }) });
    announce('Darstellung gespeichert.');
  } catch (error) { announce(error.message); }
}

function themeSettingsControl() {
  const preference = normalizeTheme(root.dataset.themePreference);
  const labels = { system: 'System', light: 'Hell', dark: 'Dunkel' };
  return `<fieldset class="settings-theme"><legend>Darstellung</legend><div class="theme-options">${['system', 'light', 'dark'].map((value) => `<label><input type="radio" name="settings-theme" value="${value}" data-theme-choice="${value}" ${preference === value ? 'checked' : ''}><span>${labels[value]}</span></label>`).join('')}</div><p>System folgt automatisch der Darstellung deines Geräts.</p></fieldset>`;
}

function bindSettingsThemeControls() {
  document.querySelectorAll('input[name="settings-theme"]').forEach((input) => input.addEventListener('change', () => {
    if (input.checked) saveTheme(input.value);
  }));
}
function priorityBadge(priority) {
  const labels = { core: '◆ KERN', support: '● ERGÄNZUNG', reference: '○ REFERENZ' };
  return `<span class="priority priority-${priority}">${labels[priority]}</span>`;
}
function topicSourceAction(topic, label = 'Originalfolie ansehen') {
  const sources = topic?.sources || [];
  const source = sources.length === 1 && sources[0].available && sources[0].url ? sources[0] : null;
  return source ? `<a class="source-action" href="${escapeHTML(source.url)}" target="_blank" rel="noopener">${escapeHTML(label)} ↗</a>` : '';
}
async function homeView() {
  breadcrumb.textContent = 'Heute';
  const data = await requestJSON('/api/progress');
  const dueEntries = await Promise.all(catalog.exams.map(async (exam) => [exam.id, await requestJSON(`/api/reviews/due?exam=${encodeURIComponent(exam.id)}`)]));
  const dueByExam = Object.fromEntries(dueEntries);
  app.innerHTML = `<section class="intro"><span class="eyebrow">Heute</span><h1>Was möchtest du heute lernen?</h1><p>Wähle deinen Prüfungsbereich. Der Lernstand bleibt für beide Prüfungen getrennt.</p></section><section class="exam-grid" aria-label="Prüfungsbereiche">${catalog.exams.map((exam) => {
    const current = data.exams.find((item) => item.exam_id === exam.id) || { percent: 0 };
    const due = dueByExam[exam.id];
    return `<article class="exam-card"><div class="card-heading"><span class="exam-code">${exam.id === 'machine-learning' ? 'ML' : 'DL'}</span><span class="status">${current.percent} % beherrscht</span></div><h2>${escapeHTML(exam.name)}</h2><p>${escapeHTML(exam.description)}</p><div class="progress-row"><span>Beherrschung</span><strong>${current.percent} %</strong></div><progress class="progress-track" max="100" value="${current.percent}" aria-label="${escapeHTML(exam.name)}: ${current.percent} Prozent beherrscht"></progress><dl class="due-summary"><div><dt>Überfällig</dt><dd>${due.overdue}</dd></div><div><dt>Heute</dt><dd>${due.today}</dd></div><div><dt>Diese Woche</dt><dd>${due.this_week}</dd></div></dl><div class="card-actions"><a class="primary-action" href="#/today/${exam.id}">Heute lernen</a><a href="#/exam/${exam.id}">Themen öffnen</a></div></article>`;
  }).join('')}</section>`;
}
function examView(examID) {
  const exam = findExam(examID); if (!exam) return notFoundView();
  breadcrumb.innerHTML = `<a href="#/">Heute</a><span>/</span>${escapeHTML(exam.name)}`;
  app.innerHTML = `<section class="page-heading"><a class="back-link" href="#/">← Beide Prüfungsbereiche</a><span class="eyebrow">Prüfungsbereich</span><h1>${escapeHTML(exam.name)}</h1><p>${escapeHTML(exam.description)}</p></section><section class="module-list" aria-label="Module">${modulesFor(examID).map((module, index) => {
    const topics = topicsFor(module.id);
    const content = `<span class="module-number">${String(index + 1).padStart(2, '0')}</span><span><strong>${escapeHTML(module.name)}</strong><small>${escapeHTML(module.description)}</small></span><span class="module-count">${topics.length ? `${topics.length} Thema${topics.length > 1 ? 'en' : ''}` : 'Content folgt'}</span><span aria-hidden="true">${topics.length ? '→' : '—'}</span>`;
    return topics.length ? `<a class="module-row" href="#/module/${module.id}">${content}</a>` : `<div class="module-row empty">${content}</div>`;
  }).join('')}</section>`;
}
function moduleView(moduleID) {
  const module = findModule(moduleID); if (!module) return notFoundView();
  const exam = findExam(module.exam_id); const topics = topicsFor(moduleID);
  breadcrumb.innerHTML = `<a href="#/exam/${exam.id}">${escapeHTML(exam.name)}</a><span>/</span>${escapeHTML(module.name)}`;
  app.innerHTML = `<section class="page-heading"><a class="back-link" href="#/exam/${exam.id}">← ${escapeHTML(exam.name)}</a><span class="eyebrow">Modul</span><h1>${escapeHTML(module.name)}</h1><p>${escapeHTML(module.description)}</p></section><section class="topic-list">${topics.map((topic, index) => `<a class="topic-row" href="#/topic/${topic.id}"><span class="topic-position">${index + 1} / ${topics.length}</span><span>${priorityBadge(topic.priority)}<strong>${escapeHTML(topic.title)}</strong><small>${escapeHTML(topic.summary)}</small></span><span aria-hidden="true">→</span></a>`).join('')}</section>`;
}
function topicView(topicID) {
  const topic = findTopic(topicID); if (!topic) return notFoundView();
  requestJSON(`/api/topics/${encodeURIComponent(topicID)}/view`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(() => {});
  const module = findModule(topic.module_id); const exam = findExam(topic.exam_id);
  const sequence = topicNavigation(catalog.topics, topicID);
  const topicPosition = `Thema ${sequence.position} / ${sequence.total}`;
  const previousHref = sequence.previousID ? `#/topic/${sequence.previousID}` : `#/module/${module.id}`;
  const nextHref = sequence.nextID ? `#/topic/${sequence.nextID}` : `#/exam/${exam.id}`;
  const moduleTraining = !sequence.nextID ? `<section class="test-placeholder"><span>Kannst du es?</span><p>Du bist beim letzten Thema dieses Moduls. Im Modultraining kommen Fragen zu allen ${sequence.total} Themen vor.</p><a class="test-action" href="#/quiz/${exam.id}/${module.id}">Modultraining starten</a></section>` : '';
  const navigation = `<nav class="topic-navigation" aria-label="Themennavigation"><h2>Weiterlernen</h2><p class="topic-sequence">${topicPosition} · ${escapeHTML(module.name)}</p><div class="topic-navigation-actions"><a data-topic-previous href="${previousHref}">${sequence.previousID ? '← Vorheriges Thema' : '← Zur Themenübersicht'}</a>${!sequence.nextID ? `<a data-topic-training href="#/quiz/${exam.id}/${module.id}">Modultraining</a>` : ''}<a class="topic-next" data-topic-next href="${nextHref}">${sequence.nextID ? 'Nächstes Thema →' : 'Zum Prüfungsbereich →'}</a></div></nav>`;
  const explanation = Array.isArray(topic.explanation) ? topic.explanation : [];
  const steps = Array.isArray(topic.steps) ? topic.steps : [];
  const confusions = Array.isArray(topic.confusions) ? topic.confusions : [];
  const sources = (Array.isArray(topic.sources) ? topic.sources : []).map((source) => {
    const pageRanges = source.page_ranges?.length ? source.page_ranges : [{ label: source.page_label, url: source.url }];
    const rangeRows = pageRanges.map((range) => `<div class="source-range"><span>${range.start === range.end ? 'Folie' : 'Folien'} ${escapeHTML(range.label)}</span>${source.available && range.url ? `<a href="${escapeHTML(range.url)}" target="_blank" rel="noopener">Originalfolien öffnen ↗</a>` : ''}</div>`).join('');
    return `<div class="source-row"><strong>${escapeHTML(source.title || source.file)}</strong><div class="source-range-list">${rangeRows}</div>${source.available ? '' : `<em>⚠ Originaldatei momentan nicht verfügbar: ${escapeHTML(source.file)}</em>`}</div>`;
  }).join('');
  const formula = renderFormula(topic.formula);
  breadcrumb.innerHTML = `<a href="#/exam/${exam.id}">${escapeHTML(exam.name)}</a><span>/</span><a href="#/module/${module.id}">${escapeHTML(module.name)}</a>`;
  app.innerHTML = `<article class="learning-page"><header class="learning-header"><a class="back-link" href="#/module/${module.id}">← ${escapeHTML(module.name)}</a>${priorityBadge(topic.priority)}<p class="topic-sequence">${topicPosition}</p><h1>${escapeHTML(topic.title)}</h1></header><section class="core-idea"><span>Kernidee</span><p>${escapeHTML(topic.summary)}</p></section><section class="learning-section"><h2>So funktioniert es</h2><div class="explanation-list">${explanation.map((paragraph) => `<p>${escapeHTML(paragraph)}</p>`).join('')}</div></section>${renderLearningVisual(topic, 0)}${steps.length ? `<section class="learning-section"><h2>Ablauf</h2><ol class="steps">${steps.map((step) => `<li>${escapeHTML(step)}</li>`).join('')}</ol></section>` : ''}<section class="learning-section"><h2>${topic.illustrations?.length ? 'Durchgerechnetes Beispiel' : 'Beispiel'}</h2><p class="learning-example">${escapeHTML(topic.example || '')}</p>${renderLearningVisual(topic, 1)}</section>${formula}<section class="learning-section"><h2>⚠ Nicht verwechseln</h2><div class="confusion-list">${confusions.map((item) => `<div><strong>${escapeHTML(item.term)}</strong><p>${escapeHTML(item.note)}</p></div>`).join('')}</div></section><section class="sources" aria-labelledby="original-material-heading"><h2 class="eyebrow" id="original-material-heading">Originalmaterial</h2><div class="source-list">${sources}</div></section><section class="learning-section topic-checks" aria-labelledby="topic-checks-heading" data-check-topic="${escapeHTML(topic.id)}"><h2 id="topic-checks-heading">Kurzcheck: dieses Thema</h2><p>Richtig oder falsch? Prüfe nur den gerade gelesenen Inhalt. Diese Antworten verändern weder Mastery noch Prüfungsstatistik.</p><div class="topic-check-list" role="region" aria-label="Verständnisfragen"><p role="status">Verständnisfragen werden geladen …</p></div></section>${moduleTraining}${navigation}</article>`;
  loadTopicChecks(topic.id, document.querySelector('.topic-check-list'));
}

async function loadTopicChecks(topicID, container) {
  try {
    const data = await requestJSON(`/api/comprehension?topic=${encodeURIComponent(topicID)}`);
    // A response for the previous page must never populate the new topic.
    if (!container.isConnected) return;
    if (data.topic_id !== topicID || data.questions.some((question) => question.topic_id !== topicID || question.type !== 'true-false')) throw new Error('Die Verständnisfragen passen nicht zu diesem Thema.');
    if (!data.questions.length) { container.innerHTML = '<p>Für dieses Thema sind noch keine Kurzfragen vorhanden.</p>'; return; }
    container.innerHTML = data.questions.map((question, index) => `<form class="topic-check-card" data-check-id="${escapeHTML(question.id)}"><fieldset><legend>Frage ${index + 1}: ${richText(question.prompt)}</legend>${question.context ? `<p>${richText(question.context)}</p>` : ''}<div class="topic-check-options">${question.options.map((option) => `<label><input type="radio" name="check-${escapeHTML(question.id)}" value="${escapeHTML(option.id)}" required><span>${richText(option.text)}</span></label>`).join('')}</div></fieldset><button type="submit" class="check-answer">Antwort prüfen</button><div class="topic-check-feedback" aria-live="polite"></div></form>`).join('');
    container.querySelectorAll('form').forEach((form, index) => {
      const question = data.questions[index];
      form.addEventListener('submit', async (event) => {
        event.preventDefault();
        const selected = form.querySelector('input:checked');
        if (!selected || form.dataset.busy === 'true' || form.dataset.answered === 'true') return;
        const button = form.querySelector('button[type="submit"]');
        const fieldset = form.querySelector('fieldset');
        const feedback = form.querySelector('.topic-check-feedback');
        form.dataset.busy = 'true'; button.disabled = true; fieldset.disabled = true;
        try {
          const result = await requestJSON('/api/comprehension/answers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ topic_id: topicID, question_id: question.id, response: { option_ids: [selected.value] } }) });
          if (!form.isConnected) return;
          form.dataset.answered = 'true';
          feedback.innerHTML = `<div class="answer-feedback ${result.correct ? 'correct' : 'wrong'}" tabindex="-1"><h3>Deine Antwort ist ${result.correct ? 'richtig' : 'falsch'}.</h3><p>Die Aussage ist: <strong>${richText(optionText(question, result.correct_option_ids[0]))}</strong>.</p><h4>Warum – und warum nicht anders?</h4><p>${richText(result.explanation)}</p><button type="button" class="check-retry">Nochmal beantworten</button></div>`;
          feedback.querySelector('.check-retry').addEventListener('click', () => {
            form.reset(); fieldset.disabled = false; button.disabled = false; delete form.dataset.answered;
            feedback.replaceChildren(); form.querySelector('input').focus({ preventScroll: true });
          });
          feedback.querySelector('.answer-feedback').focus({ preventScroll: true });
        } catch (error) {
          if (!form.isConnected) return;
          feedback.textContent = error.message; fieldset.disabled = false; button.disabled = false;
        } finally { delete form.dataset.busy; }
      });
    });
  } catch (error) {
    if (!container.isConnected) return;
    container.innerHTML = `<p role="alert">${escapeHTML(error.message)}</p><button type="button" class="check-load-retry">Erneut laden</button>`;
    container.querySelector('button').addEventListener('click', () => loadTopicChecks(topicID, container));
  }
}

function todaySetupView(examID) {
  const exam = findExam(examID); if (!exam) return notFoundView();
  breadcrumb.innerHTML = `<a href="#/">Heute</a><span>/</span>${escapeHTML(exam.name)}`;
  app.innerHTML = `<section class="page-heading"><a class="back-link" href="#/">← Prüfungsbereich wählen</a><span class="eyebrow">${escapeHTML(exam.name)}</span><h1>Wie viel Zeit hast du?</h1><p>Die Session priorisiert fälligen Kernstoff und Schwachstellen.</p></section><section class="time-options" aria-label="Lernzeit">${[10, 20, 30, 45].map((minutes) => `<a href="#/today/${exam.id}/${minutes}"><strong>${minutes} min</strong><small>${minutes === 10 ? 'Nur Wiederholungen' : minutes === 20 ? 'Wiederholungen und Schwachstellen' : minutes === 30 ? 'Mit einem neuen Thema' : 'Mit bis zu zwei neuen Themen'}</small></a>`).join('')}</section>`;
}

async function todayPlanView(examID, minutes) {
  const exam = findExam(examID); if (!exam) return notFoundView();
  const plan = await requestJSON(`/api/today?exam=${encodeURIComponent(examID)}&minutes=${minutes}`);
  breadcrumb.innerHTML = `<a href="#/">Heute</a><span>/</span>${escapeHTML(exam.name)}`;
  app.innerHTML = `<section class="session-plan"><span class="eyebrow">${escapeHTML(exam.name)}</span><h1>Heute · ca. ${plan.estimated_minutes} Minuten</h1><dl><div><dt>Wiederholungen</dt><dd>${plan.review_count}</dd></div><div><dt>Schwachstellen</dt><dd>${plan.weak_count}</dd></div><div><dt>Neue Konzepte</dt><dd>${plan.new_topic_count}</dd></div><div><dt>Fragen</dt><dd>${plan.question_count}</dd></div></dl>${plan.question_ids.length ? `<button class="primary-action" id="start-session">Session starten</button>` : '<p>Aktuell ist für diese Auswahl nichts fällig.</p>'}<a class="secondary-link" href="#/today/${exam.id}">Andere Zeit wählen</a></section>`;
  document.querySelector('#start-session')?.addEventListener('click', () => startTodaySession(exam, plan));
}

async function startTodaySession(exam, plan) {
  const data = await requestJSON(`/api/questions?exam=${encodeURIComponent(exam.id)}`); const byID = new Map(data.questions.map((q) => [q.id, q]));
  const questions = plan.question_ids.map((id) => byID.get(id)).filter(Boolean);
  quizSession = { exam, moduleID: 'today', questions, index: 0, response: null, result: null, correct: 0, wrong: 0, repeated: new Set(), topicResults: {}, newTopicIDs: plan.new_topic_ids || [], newTopicIndex: 0, today: true };
  renderTodayLearningStep();
}

function renderTodayLearningStep() {
  if (quizSession.newTopicIndex >= quizSession.newTopicIDs.length) { renderQuestion(); focusHeading(); return; }
  const topic = findTopic(quizSession.newTopicIDs[quizSession.newTopicIndex]);
  if (!topic) { quizSession.newTopicIndex += 1; renderTodayLearningStep(); return; }
  requestJSON(`/api/topics/${encodeURIComponent(topic.id)}/view`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' }).catch(() => {});
  breadcrumb.textContent = 'Heute · Neues Konzept';
  app.innerHTML = `<article class="session-learning"><span class="eyebrow">Neues Konzept ${quizSession.newTopicIndex + 1} von ${quizSession.newTopicIDs.length}</span>${priorityBadge(topic.priority)}<h1>${escapeHTML(topic.title)}</h1><section class="core-idea"><span>Kernidee</span><p>${escapeHTML(topic.summary)}</p></section><section class="learning-section"><h2>So funktioniert es</h2><div class="explanation-list">${topic.explanation.map((paragraph) => `<p>${escapeHTML(paragraph)}</p>`).join('')}</div></section>${renderLearningVisual(topic, 0)}<section class="learning-section"><h2>${topic.illustrations?.length ? 'Durchgerechnetes Beispiel' : 'Beispiel aus der Vorlesung'}</h2><p class="learning-example">${escapeHTML(topic.example)}</p>${renderLearningVisual(topic, 1)}</section>${renderFormula(topic.formula)}<button class="primary-action" id="continue-session">${quizSession.newTopicIndex + 1 === quizSession.newTopicIDs.length ? 'Weiter zu den Fragen' : 'Nächstes Konzept'}</button></article>`;
  document.querySelector('#continue-session').addEventListener('click', () => { quizSession.newTopicIndex += 1; renderTodayLearningStep(); });
  focusHeading();
}

const statusLabels = { mastered: 'Sicher', practicing: 'In Übung', weak: 'Unsicher', seen: 'Gesehen', unlearned: 'Ungelernt' };
async function progressView() {
  breadcrumb.textContent = 'Stand'; const data = await requestJSON('/api/progress');
  app.innerHTML = `<section class="page-heading"><span class="eyebrow">Stand</span><h1>Dein Lernfortschritt</h1><p>Beherrschung wird aus Quizresultaten berechnet. Referenzthemen zählen nicht zum Prozentwert.</p></section><section class="progress-exams">${data.exams.map((item) => { const exam = findExam(item.exam_id); const openCore = item.topics.filter((topic) => topic.priority === 'core' && topic.status !== 'mastered'); return `<article class="progress-panel"><h2>${escapeHTML(exam.name)}</h2><div class="large-progress"><strong>${item.percent} %</strong><span>beherrscht</span></div><progress class="progress-track" max="100" value="${item.percent}" aria-label="${escapeHTML(exam.name)}: ${item.percent} Prozent beherrscht"></progress><dl class="status-counts">${['mastered', 'practicing', 'weak', 'seen', 'unlearned'].map((status) => `<div><dt>${statusLabels[status]}</dt><dd>${item.counts[status] || 0}</dd></div>`).join('')}</dl><h3>Kernstoff offen</h3><ul class="core-open">${openCore.length ? openCore.map((topic) => `<li><span aria-hidden="true">${topic.status === 'weak' ? '⚠' : '◇'}</span><a href="#/topic/${topic.topic_id}">${escapeHTML(topic.title)}</a><small>${statusLabels[topic.status]}</small></li>`).join('') : '<li>Alle vorhandenen Kernthemen sind sicher.</li>'}</ul><h3>Module</h3><div class="module-progress">${modulesFor(exam.id).map((module) => `<div><span>${escapeHTML(module.name)}</span><strong>${item.modules[module.id] || 0} %</strong></div>`).join('')}</div></article>`; }).join('')}</section>`;
}
function unavailableView(section) {
  breadcrumb.textContent = section === 'practice' ? 'Üben' : 'Fortschritt';
  app.innerHTML = `<section class="empty-state"><h1>${section === 'practice' ? 'Üben' : 'Fortschritt'}</h1><p>Für diesen Bereich sind noch keine Daten verfügbar.</p><a class="primary-action" href="#/">Zur Startseite</a></section>`;
}

function practiceView() {
  breadcrumb.textContent = 'Üben';
  app.innerHTML = `<section class="page-heading"><span class="eyebrow">Üben</span><h1>Welchen Bereich möchtest du üben?</h1><p>Die Fragen beider Prüfungen bleiben vollständig getrennt.</p></section><section class="practice-exams">${catalog.exams.map((exam) => `<a href="#/practice/${exam.id}"><span class="exam-code">${exam.id === 'machine-learning' ? 'ML' : 'DL'}</span><span><strong>${escapeHTML(exam.name)}</strong><small>Fragen auswählen</small></span><span aria-hidden="true">→</span></a>`).join('')}</section><a class="simulation-callout" href="#/exam-simulation"><span><strong>Prüfungssimulation</strong><small>10, 20 oder 30 Fragen ohne sofortiges Feedback</small></span><span aria-hidden="true">→</span></a>`;
}

function examSimulationView(examID) {
  breadcrumb.textContent = 'Prüfungssimulation';
  if (!examID) {
    app.innerHTML = `<section class="page-heading"><span class="eyebrow">Prüfungssimulation</span><h1>Welche Prüfung möchtest du simulieren?</h1><p>Lösungen und Auswertung erscheinen erst nach der vollständigen Abgabe.</p></section><section class="practice-exams">${catalog.exams.map((exam) => `<a href="#/exam-simulation/${exam.id}"><span class="exam-code">${exam.id === 'machine-learning' ? 'ML' : 'DL'}</span><span><strong>${escapeHTML(exam.name)}</strong><small>Simulation konfigurieren</small></span><span aria-hidden="true">→</span></a>`).join('')}</section>`;
    return;
  }
  const exam = findExam(examID); if (!exam) return notFoundView();
  breadcrumb.innerHTML = `<a href="#/exam-simulation">Prüfung</a><span>/</span>${escapeHTML(exam.name)}`;
  app.innerHTML = `<section class="page-heading"><a class="back-link" href="#/exam-simulation">← Prüfungsbereich wählen</a><span class="eyebrow">${escapeHTML(exam.name)}</span><h1>Simulation konfigurieren</h1><p>Während der Prüfung werden weder Lösungen noch Erklärungen angezeigt.</p></section><form class="exam-config" id="exam-config"><fieldset><legend>Fragenzahl</legend><div class="config-options">${[10, 20, 30].map((count) => `<label><input type="radio" name="count" value="${count}" ${count === 10 ? 'checked' : ''}><span>${count} Fragen</span></label>`).join('')}</div></fieldset><fieldset><legend>Inhalt</legend><div class="config-options scope-options"><label><input type="radio" name="scope" value="all" checked><span>Alle Themen</span></label><label><input type="radio" name="scope" value="core"><span>Nur Kernstoff</span></label><label><input type="radio" name="scope" value="weak"><span>Nur Schwachstellen</span></label></div></fieldset><p class="availability-note">Alle Themen berücksichtigt Kernstoff und Ergänzungen. Nur Kernstoff beschränkt die Auswahl auf Kernthemen. Bei Schwachstellen werden nur tatsächlich als unsicher erkannte Fragen verwendet.</p><p id="exam-config-error" class="answer-error" role="alert"></p><button class="primary-action" type="submit">Prüfung starten</button></form>`;
  document.querySelector('#exam-config').addEventListener('submit', async (event) => {
    event.preventDefault(); const form = new FormData(event.currentTarget); const count = form.get('count'); const scope = form.get('scope');
    const button = event.currentTarget.querySelector('button'); button.disabled = true; button.textContent = 'Fragen werden vorbereitet …';
    let data;
    try {
      data = await requestJSON(`/api/exam/questions?exam=${encodeURIComponent(exam.id)}&count=${count}&scope=${scope}`);
    } catch (error) {
      button.disabled = false; button.textContent = 'Prüfung starten';
      document.querySelector('#exam-config-error').textContent = error.message;
      return;
    }
    if (!data.questions.length) { app.innerHTML = `<section class="empty-state"><h1>Keine passenden Fragen</h1><p>Für diese Auswahl sind derzeit keine Fragen vorhanden.</p><a class="primary-action" href="#/exam-simulation/${exam.id}">Auswahl ändern</a></section>`; focusHeading(); return; }
    examSession = { attemptID: data.attempt_id, exam, scope, requestedCount: Number(count), questions: data.questions, index: 0, responses: {}, result: null };
    renderExamQuestion(); focusHeading();
  });
}

function emptyResponse(question) {
  if (question.type === 'ordering') return { order: question.options.map((option) => option.id) };
  if (question.type === 'matching') return { matches: {} };
  return { option_ids: [] };
}

function renderExamQuestion() {
  const question = examSession.questions[examSession.index]; const topic = findTopic(question.topic_id); const module = findModule(topic.module_id);
  examSession.responses[question.id] ||= emptyResponse(question); const state = { response: examSession.responses[question.id] }; const position = examSession.index + 1;
  breadcrumb.innerHTML = `<span>Prüfung</span><span>/</span>${escapeHTML(examSession.exam.name)}`;
  const shortfall = examSession.questions.length < examSession.requestedCount ? `<p class="exam-shortfall" role="status">${examSession.requestedCount} Fragen gewählt. Für diesen Filter sind aktuell ${examSession.questions.length} Fragen verfügbar.</p>` : '';
  app.innerHTML = `<article class="quiz-page exam-mode"><header class="quiz-progress"><span>${escapeHTML(examSession.exam.name)} · Prüfung</span><strong>Frage ${position} von ${examSession.questions.length}</strong>${shortfall}<progress class="question-progress-bar" max="${examSession.questions.length}" value="${position}" aria-label="Frage ${position} von ${examSession.questions.length}"></progress></header><section class="question-card"><span class="question-module">◆ ${escapeHTML(module.name)}</span>${question.type === 'multiple-select' ? '<p class="multi-hint">Mehrere Antworten sind richtig</p>' : ''}${question.context ? `<p class="question-context">${richText(question.context)}</p>` : ''}<h1>${richText(question.prompt)}</h1>${questionControl(question, false, state)}<p id="answer-error" class="answer-error" role="alert"></p><div class="exam-navigation">${position > 1 ? '<button type="button" data-exam-back>Zurück</button>' : '<span></span>'}<button class="primary-action" type="button" data-exam-next>${position === examSession.questions.length ? 'Prüfung abgeben' : 'Weiter'}</button></div></section></article>`;
  bindExamEvents(question, state);
}

function bindExamEvents(question, state) {
  document.querySelectorAll('input[name="answer"]').forEach((input) => input.addEventListener('change', () => { state.response.option_ids = question.type === 'multiple-select' ? [...document.querySelectorAll('input[name="answer"]:checked')].map((item) => item.value) : [input.value]; }));
  document.querySelectorAll('[data-match]').forEach((select) => select.addEventListener('change', () => { state.response.matches[select.dataset.match] = select.value; }));
  document.querySelectorAll('[data-move]').forEach((button) => button.addEventListener('click', () => { const index = Number(button.dataset.index); const target = button.dataset.move === 'up' ? index - 1 : index + 1; const direction = button.dataset.move; [state.response.order[index], state.response.order[target]] = [state.response.order[target], state.response.order[index]]; renderExamQuestion(); focusOrderControl(target, direction); }));
  document.querySelector('[data-exam-back]')?.addEventListener('click', () => { examSession.index -= 1; renderExamQuestion(); focusHeading(); });
  document.querySelector('[data-exam-next]').addEventListener('click', () => {
    if (!responseComplete(question, state)) { document.querySelector('#answer-error').textContent = 'Bitte beantworte die Aufgabe vollständig.'; return; }
    if (examSession.index < examSession.questions.length - 1) { examSession.index += 1; renderExamQuestion(); focusHeading(); return; }
    finishExamSimulation();
  });
}

async function finishExamSimulation() {
  const button = document.querySelector('[data-exam-next]'); button.disabled = true; button.textContent = 'Prüfung wird ausgewertet …';
  const answers = examSession.questions.map((question) => ({ question_id: question.id, response: examSession.responses[question.id] }));
  let result;
  try {
    result = await requestJSON('/api/exam/finish', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ attempt_id: examSession.attemptID, exam_id: examSession.exam.id, answers }) });
  } catch (error) {
    button.disabled = false; button.textContent = 'Prüfung abgeben'; document.querySelector('#answer-error').textContent = error.message; return;
  }
  examSession.result = result; renderExamResult(); focusHeading(); announce(`Prüfung abgeschlossen: ${result.percent} Prozent richtig.`);
}

function renderExamResult() {
  const result = examSession.result; const questionBank = new Map(examSession.questions.map((question) => [question.id, question])); const wrong = result.results.filter((item) => !item.correct);
  breadcrumb.textContent = 'Prüfung · Auswertung';
  const breakdown = (items) => items.map((item) => `<div><span>${escapeHTML(item.name)}</span><strong>${item.correct} / ${item.total}</strong></div>`).join('');
  app.innerHTML = `<section class="exam-result"><span class="eyebrow">Prüfung abgeschlossen</span><h1>${result.percent} % richtig</h1><dl class="exam-score"><div><dt>Richtig</dt><dd>${result.correct}</dd></div><div><dt>Falsch</dt><dd>${result.wrong}</dd></div><div><dt>Gesamt</dt><dd>${result.total}</dd></div></dl><div class="result-breakdowns"><section><h2>Nach Topic</h2>${breakdown(result.topics)}</section><section><h2>Nach Modul</h2>${breakdown(result.modules)}</section></div>${wrong.length ? `<section class="error-analysis"><h2>Fehleranalyse</h2>${wrong.map((item) => { const question = questionBank.get(item.question_id); const topic = findTopic(item.topic_id); return `<details><summary>${richText(question.prompt)}</summary><div class="answer-comparison"><div><strong>Deine Antwort</strong>${selectedAnswerMarkup(question, { response: item.response })}</div><div><strong>Richtig</strong>${correctAnswerMarkup(question, item)}</div></div><h3>Warum?</h3><p>${richText(item.explanation)}</p>${topicSourceAction(topic)}</details>`; }).join('')}<button class="primary-action" id="practice-wrong">Falsche Fragen nochmals üben</button></section>` : '<p class="all-correct">✓ Alle Fragen richtig beantwortet.</p>'}<div class="result-actions"><a class="primary-action" href="#/exam-simulation/${examSession.exam.id}">Neue Simulation</a><a href="#/">Fertig</a></div></section>`;
  document.querySelector('#practice-wrong')?.addEventListener('click', () => { const questions = wrong.map((item) => ({ ...questionBank.get(item.question_id), id: item.practice_question_id })); quizSession = { exam: examSession.exam, moduleID: 'exam-errors', questions, index: 0, response: null, result: null, correct: 0, wrong: 0, repeated: new Set(), topicResults: {}, today: false }; renderQuestion(); focusHeading(); });
}

function practiceExamView(examID) {
  const exam = findExam(examID); if (!exam) return notFoundView();
  const modules = modulesFor(examID).filter((module) => topicsFor(module.id).length > 0);
  breadcrumb.innerHTML = `<a href="#/practice">Üben</a><span>/</span>${escapeHTML(exam.name)}`;
  app.innerHTML = `<section class="page-heading"><a class="back-link" href="#/practice">← Prüfungsbereich wählen</a><span class="eyebrow">${escapeHTML(exam.name)}</span><h1>Fragen auswählen</h1><p>Du erhältst sofort Feedback. Die Antwortreihenfolge wird für jeden Durchlauf neu gemischt.</p></section><section class="practice-selection"><a class="selection-card primary-selection" href="#/quiz/${exam.id}/all"><span class="selection-copy"><strong>Alle vorhandenen Fragen</strong><small>Gemischt aus allen befüllten Modulen</small></span><span class="selection-arrow" aria-hidden="true">→</span></a><h2>Oder ein Modul auswählen</h2>${modules.map((module) => `<a class="selection-card" href="#/quiz/${exam.id}/${module.id}"><span class="selection-copy"><strong>${escapeHTML(module.name)}</strong><small>${topicsFor(module.id).map((topic) => topic.title).join(' · ')}</small></span><span class="selection-arrow" aria-hidden="true">→</span></a>`).join('')}</section>`;
}

async function quizView(examID, moduleID) {
  const exam = findExam(examID); if (!exam) return notFoundView();
  const query = new URLSearchParams({ exam: examID });
  if (moduleID && moduleID !== 'all') query.set('module', moduleID);
  app.innerHTML = '<p class="loading">Fragen werden gemischt …</p>';
  const data = await requestJSON(`/api/questions?${query}`);
  quizSession = { exam, moduleID, questions: data.questions, index: 0, response: null, result: null, correct: 0, wrong: 0, repeated: new Set(), topicResults: {}, today: false };
  if (!quizSession.questions.length) {
    app.innerHTML = '<section class="empty-state"><h1>Keine Fragen vorhanden</h1><p>Für diese Auswahl sind noch keine Fragen aktiviert.</p><a class="primary-action" href="#/practice">Zur Auswahl</a></section>';
    return;
  }
  renderQuestion();
}

function optionText(question, id) {
  return [...(question.options || []), ...(question.match_options || [])].find((option) => option.id === id)?.text || id;
}

function questionControl(question, locked, state = quizSession) {
  if (question.type === 'ordering') {
    const order = state.response.order;
    return `<ol class="order-list">${order.map((id, index) => `<li><span class="order-controls"><button type="button" data-move="up" data-index="${index}" aria-label="${escapeHTML(plainText(optionText(question, id)))} nach oben" ${locked || index === 0 ? 'disabled' : ''}>↑</button><button type="button" data-move="down" data-index="${index}" aria-label="${escapeHTML(plainText(optionText(question, id)))} nach unten" ${locked || index === order.length - 1 ? 'disabled' : ''}>↓</button></span><span>${richText(optionText(question, id))}</span></li>`).join('')}</ol>`;
  }
  if (question.type === 'matching') {
    return `<div class="matching-list">${question.match_items.map((item) => `<label><span>${richText(item.text)}</span><select data-match="${item.id}" ${locked ? 'disabled' : ''}><option value="">Bitte zuordnen</option>${question.match_options.map((option) => `<option value="${option.id}" ${state.response.matches[item.id] === option.id ? 'selected' : ''}>${escapeHTML(plainText(option.text))}</option>`).join('')}</select></label>`).join('')}</div>`;
  }
  const multiple = question.type === 'multiple-select';
  const optionMarkup = question.options.map((option) => `<label class="answer-option"><input type="${multiple ? 'checkbox' : 'radio'}" name="answer" value="${option.id}" ${state.response.option_ids.includes(option.id) ? 'checked' : ''} ${locked ? 'disabled' : ''}><span class="choice-indicator" aria-hidden="true"></span><span>${richText(option.text)}</span></label>`).join('');
  return `<fieldset class="answer-list ${question.type === 'true-false' ? 'true-false' : ''}"><legend class="sr-only">Antwortmöglichkeiten zu: ${escapeHTML(plainText(question.prompt))}</legend>${optionMarkup}</fieldset>`;
}

function focusOrderControl(index, direction) {
  const preferred = document.querySelector(`[data-index="${index}"][data-move="${direction}"]:not(:disabled)`);
  const fallback = document.querySelector(`[data-index="${index}"]:not(:disabled)`);
  if (preferred || fallback) (preferred || fallback).focus(); else focusHeading();
}

function correctAnswerMarkup(question, result) {
  if (question.type === 'ordering') return `<ol>${result.correct_order.map((id) => `<li>${richText(optionText(question, id))}</li>`).join('')}</ol>`;
  if (question.type === 'matching') return `<ul>${question.match_items.map((item) => `<li><strong>${richText(item.text)}:</strong> ${richText(optionText(question, result.correct_matches[item.id]))}</li>`).join('')}</ul>`;
  return `<ul>${result.correct_option_ids.map((id) => `<li>${richText(optionText(question, id))}</li>`).join('')}</ul>`;
}

function selectedAnswerMarkup(question, state = quizSession) {
  if (question.type === 'ordering') return `<ol>${state.response.order.map((id) => `<li>${richText(optionText(question, id))}</li>`).join('')}</ol>`;
  if (question.type === 'matching') return `<ul>${question.match_items.map((item) => `<li><strong>${richText(item.text)}:</strong> ${richText(optionText(question, state.response.matches[item.id]))}</li>`).join('')}</ul>`;
  return `<ul>${state.response.option_ids.map((id) => `<li>${richText(optionText(question, id))}</li>`).join('')}</ul>`;
}

function responseComplete(question, state = quizSession) {
  if (question.type === 'ordering') return state.response.order.length === question.options.length;
  if (question.type === 'matching') return question.match_items.every((item) => state.response.matches[item.id]);
  return state.response.option_ids.length > 0;
}

function renderQuestion() {
  const question = quizSession.questions[quizSession.index];
  const topic = findTopic(question.topic_id); const module = findModule(topic.module_id);
  if (!quizSession.response) quizSession.response = question.type === 'ordering' ? { order: question.options.map((option) => option.id) } : question.type === 'matching' ? { matches: {} } : { option_ids: [] };
  const locked = Boolean(quizSession.result); const position = quizSession.index + 1;
  breadcrumb.innerHTML = `<a href="#/practice/${quizSession.exam.id}">Üben</a><span>/</span>${escapeHTML(quizSession.exam.name)}`;
  app.innerHTML = `<article class="quiz-page"><header class="quiz-progress"><span>${escapeHTML(quizSession.exam.name)}</span><strong>Frage ${position} von ${quizSession.questions.length}</strong><progress class="question-progress-bar" max="${quizSession.questions.length}" value="${position}" aria-label="Frage ${position} von ${quizSession.questions.length}"></progress></header><section class="question-card"><span class="question-module">◆ ${escapeHTML(module.name)}</span>${question.type === 'multiple-select' ? '<p class="multi-hint">Mehrere Antworten sind richtig</p>' : ''}${question.context ? `<p class="question-context">${richText(question.context)}</p>` : ''}<h1>${richText(question.prompt)}</h1>${questionControl(question, locked)}${!locked ? '<p id="answer-error" class="answer-error" role="alert"></p><button class="check-answer" type="button">Antwort prüfen</button>' : ''}</section>${locked ? `<section class="answer-feedback ${quizSession.result.correct ? 'correct' : 'wrong'}" tabindex="-1" role="status" aria-live="polite"><h2><span aria-hidden="true">${quizSession.result.correct ? '✓' : '✕'}</span> ${quizSession.result.correct ? 'Richtig' : 'Falsch'}</h2>${quizSession.result.correct ? '' : `<div class="answer-comparison"><div><strong>Deine Antwort</strong>${selectedAnswerMarkup(question)}</div><div><strong>Richtig</strong>${correctAnswerMarkup(question, quizSession.result)}</div></div>`}<h3>Warum?</h3><p>${richText(quizSession.result.explanation)}</p><div class="feedback-actions">${topicSourceAction(topic)}${quizSession.result.correct ? '' : `<a href="#/topic/${topic.id}">Konzept ansehen</a>`}<button class="next-question" type="button">${position === quizSession.questions.length ? 'Übung abschließen' : 'Weiter'}</button></div></section>` : ''}</article>`;
  bindQuestionEvents(question, locked);
  if (locked) document.querySelector('.answer-feedback').focus();
}

function bindQuestionEvents(question, locked) {
  if (locked) {
    document.querySelector('.next-question').addEventListener('click', () => {
      if (quizSession.index === quizSession.questions.length - 1) return quizCompleteView();
      quizSession.index += 1; quizSession.response = null; quizSession.result = null; renderQuestion(); focusHeading();
    });
    return;
  }
  document.querySelectorAll('input[name="answer"]').forEach((input) => input.addEventListener('change', () => {
    if (question.type === 'multiple-select') quizSession.response.option_ids = [...document.querySelectorAll('input[name="answer"]:checked')].map((item) => item.value);
    else quizSession.response.option_ids = [input.value];
  }));
  document.querySelectorAll('[data-move]').forEach((button) => button.addEventListener('click', () => {
    const index = Number(button.dataset.index); const target = button.dataset.move === 'up' ? index - 1 : index + 1;
    const direction = button.dataset.move;
    [quizSession.response.order[index], quizSession.response.order[target]] = [quizSession.response.order[target], quizSession.response.order[index]]; renderQuestion(); focusOrderControl(target, direction);
  }));
  document.querySelectorAll('[data-match]').forEach((select) => select.addEventListener('change', () => { quizSession.response.matches[select.dataset.match] = select.value; }));
  document.querySelector('.check-answer').addEventListener('click', async () => {
    if (!responseComplete(question)) { document.querySelector('#answer-error').textContent = 'Bitte beantworte die Aufgabe vollständig.'; return; }
    const button = document.querySelector('.check-answer'); button.disabled = true; button.textContent = 'Wird geprüft …';
    const session = !question._remediation && !quizSession.repeated.has(question.id) ? { question_ids: quizSession.questions.map((item) => item.id), current_index: quizSession.index } : undefined;
    try {
      quizSession.result = await requestJSON('/api/answers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ question_id: question.id, response: quizSession.response, session }) });
    } catch (error) {
      button.disabled = false; button.textContent = 'Antwort prüfen'; document.querySelector('#answer-error').textContent = error.message; return;
    }
    quizSession.topicResults[question.topic_id] = quizSession.topicResults[question.topic_id] || { title: findTopic(question.topic_id).title, wrong: false };
    if (quizSession.result.correct) quizSession.correct += 1; else {
      quizSession.wrong += 1;
      quizSession.topicResults[question.topic_id].wrong = true;
      if (quizSession.result.retry_question_ids?.length) {
        const questionBank = new Map(quizSession.questions.map((item) => [item.id, item]));
        quizSession.result.retry_question_ids.forEach((id) => { const retry = questionBank.get(id); if (retry) quizSession.questions.push({ ...retry, _remediation: true }); });
        quizSession.repeated.add(question.id);
      }
    }
    renderQuestion();
  });
}

function quizCompleteView() {
  breadcrumb.textContent = 'Übung abgeschlossen';
  const topics = Object.values(quizSession.topicResults); const repeated = topics.filter((item) => !item.wrong); const weak = topics.filter((item) => item.wrong);
  app.innerHTML = `<section class="session-complete"><span class="eyebrow">${quizSession.today ? 'Session fertig' : 'Übung abgeschlossen'}</span><h1>${quizSession.correct + quizSession.wrong} Fragen beantwortet</h1><dl><div><dt>Richtig</dt><dd>${quizSession.correct}</dd></div><div><dt>Falsch</dt><dd>${quizSession.wrong}</dd></div></dl>${repeated.length ? `<h2>Wiederholt</h2><ul>${repeated.map((item) => `<li>✓ ${escapeHTML(item.title)}</li>`).join('')}</ul>` : ''}${weak.length ? `<h2>Weiter üben</h2><ul>${weak.map((item) => `<li>⚠ ${escapeHTML(item.title)}</li>`).join('')}</ul>` : ''}<p>Deine Antworten und die nächsten Wiederholungen wurden gespeichert.</p><a class="primary-action" href="${quizSession.today ? '#/' : `#/practice/${quizSession.exam.id}`}">Fertig</a></section>`;
  focusHeading();
  announce('Übung abgeschlossen. Dein Lernstand wurde gespeichert.');
}

function settingsView() {
  breadcrumb.textContent = 'Einstellungen';
  app.innerHTML = `<section class="page-heading"><span class="eyebrow">Einstellungen</span><h1>Daten und Darstellung</h1><p>Verwalte deinen lokalen Lernstand. Kursinhalte und PDFs bleiben von Backups und Restores unberührt.</p></section><section class="settings-grid" aria-label="Einstellungen"><section class="settings-card">${themeSettingsControl()}</section><section class="settings-card session-settings"><h2>Sitzung</h2><p>ML/DL Exam Trainer v${APP_VERSION}</p><button type="button" class="logout-button mobile-logout" id="mobile-logout-button">Abmelden</button></section><section class="settings-card"><h2>Daten</h2><p>Das JSON-Backup enthält Einstellungen, Reviews, Fortschritt, Mastery, offene Prüfungsversuche und den Scheduler-Zustand.</p><div class="data-actions"><a class="primary-action" href="/api/backup" download>Backup herunterladen</a><button type="button" id="restore-open">Backup wiederherstellen</button></div><p class="settings-status" id="settings-status" role="status" aria-live="polite" tabindex="-1"></p></section></section><dialog class="restore-dialog" id="restore-dialog" aria-labelledby="restore-title"><form id="restore-form"><h2 id="restore-title">Backup wiederherstellen</h2><p class="restore-warning"><strong>Achtung:</strong> Dieses Backup ersetzt deinen aktuellen Lernfortschritt, Prüfungsversuche und Einstellungen.</p><p>Vor dem Restore legt der Server automatisch eine lokale Sicherheitskopie an. Content und PDFs werden nicht verändert.</p><label class="file-field" for="restore-file"><span>Backup-Datei</span><input id="restore-file" name="backup" type="file" accept=".json,application/json" required></label><p class="dialog-error" id="restore-error" role="alert"></p><div class="dialog-actions"><button type="button" id="restore-cancel">Abbrechen</button><button class="danger-action" type="submit" id="restore-confirm">Aktuellen Lernstand ersetzen</button></div></form></dialog>`;
  bindSettingsThemeControls();
  document.querySelector('#mobile-logout-button').addEventListener('click', logout);
  const trigger = document.querySelector('#restore-open');
  const dialog = document.querySelector('#restore-dialog');
  const form = document.querySelector('#restore-form');
  const fileInput = document.querySelector('#restore-file');
  const cancel = document.querySelector('#restore-cancel');
  const confirm = document.querySelector('#restore-confirm');
  let restoring = false;
  trigger.addEventListener('click', () => { dialog.showModal(); fileInput.focus(); });
  cancel.addEventListener('click', () => dialog.close());
  dialog.addEventListener('cancel', (event) => { if (restoring) event.preventDefault(); });
  dialog.addEventListener('close', () => trigger.focus());
  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    const errorBox = document.querySelector('#restore-error');
    const file = fileInput.files[0];
    if (!file) { errorBox.textContent = 'Bitte wähle eine JSON-Backup-Datei aus.'; fileInput.focus(); return; }
    if (file.size > 8 * 1024 * 1024) { errorBox.textContent = 'Das Backup darf höchstens 8 MB gross sein.'; fileInput.focus(); return; }
    restoring = true; errorBox.textContent = ''; fileInput.disabled = true; cancel.disabled = true; confirm.disabled = true; confirm.textContent = 'Backup wird wiederhergestellt …';
    try {
      const backupJSON = await file.text();
      const result = await requestJSON('/api/restore', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: backupJSON });
      renderTheme(result.theme || 'light');
      dialog.close();
      const status = document.querySelector('#settings-status');
      status.textContent = '✓ Backup erfolgreich wiederhergestellt.';
      status.focus();
      announce('Backup erfolgreich wiederhergestellt.');
    } catch (error) {
      errorBox.textContent = error.message;
    } finally {
      restoring = false; fileInput.disabled = false; cancel.disabled = false; confirm.disabled = false; confirm.textContent = 'Aktuellen Lernstand ersetzen';
    }
  });
}

function notFoundView() { breadcrumb.textContent = 'Nicht gefunden'; app.innerHTML = '<section class="empty-state"><h1>Seite nicht gefunden</h1><p>Diese Lernseite existiert nicht.</p><a class="primary-action" href="#/">Zur Startseite</a></section>'; }
function updateNavigation(parts) {
  const active = parts[0] === 'exam' ? parts[1] : parts[0] === 'module' ? findModule(parts[1])?.exam_id : parts[0] === 'topic' ? findTopic(parts[1])?.exam_id : parts[0] === 'practice' || parts[0] === 'quiz' ? 'practice' : parts[0] === 'exam-simulation' ? 'simulation' : parts[0] === 'progress' ? 'progress' : parts[0] === 'settings' ? 'settings' : 'home';
  document.querySelectorAll('[data-nav]').forEach((link) => { const selected = link.dataset.nav === active; link.classList.toggle('active', selected); if (selected) link.setAttribute('aria-current', 'page'); else link.removeAttribute('aria-current'); });
  const mobileActive = active === 'home' ? 'home' : active === 'practice' || active === 'simulation' ? 'practice' : active === 'progress' ? 'progress' : active === 'settings' ? 'settings' : 'topics';
  document.querySelectorAll('[data-mobile-nav]').forEach((link) => { const selected = link.dataset.mobileNav === mobileActive; link.classList.toggle('active', selected); if (selected) link.setAttribute('aria-current', 'page'); else link.removeAttribute('aria-current'); });
}
async function route() {
  const parts = location.hash.replace(/^#\/?/, '').split('/').filter(Boolean);
  if (!parts.length) await homeView(); else if (parts[0] === 'exam') examView(parts[1]); else if (parts[0] === 'module') moduleView(parts[1]); else if (parts[0] === 'topic') topicView(parts[1]); else if (parts[0] === 'practice' && parts[1]) practiceExamView(parts[1]); else if (parts[0] === 'practice') practiceView(); else if (parts[0] === 'quiz') await quizView(parts[1], parts[2]); else if (parts[0] === 'exam-simulation') examSimulationView(parts[1]); else if (parts[0] === 'today' && parts[2]) await todayPlanView(parts[1], Number(parts[2])); else if (parts[0] === 'today') todaySetupView(parts[1]); else if (parts[0] === 'progress') await progressView(); else if (parts[0] === 'settings') settingsView(); else notFoundView();
  updateNavigation(parts); window.scrollTo({ top: 0, behavior: 'instant' }); focusHeading();
}
themeButtons.forEach((button) => button.addEventListener('click', () => saveTheme(button.dataset.themeChoice)));
systemTheme.addEventListener('change', () => { if (root.dataset.themePreference === 'system') renderTheme('system'); });
window.addEventListener('hashchange', () => {
  if (authState === AUTH.AUTHENTICATED) route().catch((error) => errorView(error));
});

async function logout(event) {
  const trigger = event?.currentTarget;
  if (trigger) trigger.disabled = true;
  try {
    await requestJSON('/api/auth/logout', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: '{}' });
    catalog = undefined; quizSession = undefined; examSession = undefined;
    setAuthState(authTransition(authState, 'logout-succeeded'));
    renderLogin('Du wurdest abgemeldet.');
  } catch (error) {
    if (trigger) trigger.disabled = false;
    announce(error.message);
  }
}

document.querySelector('#logout-button').addEventListener('click', logout);

async function initializeApplication() {
  renderAuthLoading();
  let status;
  try { status = await requestJSON('/api/auth/status'); }
  catch { renderLogin('Der lokale Server ist nicht erreichbar.'); return; }
  if (!status.authenticated) { renderLogin(); return; }
  try { await startApplication('session-valid'); }
  catch (error) { setAuthState(AUTH.AUTHENTICATED); errorView(error, '#/'); }
}

initializeApplication();
