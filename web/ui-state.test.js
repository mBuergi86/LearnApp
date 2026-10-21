const test = require('node:test');
const assert = require('node:assert/strict');
const state = require('./ui-state.js');

test('application version has one exported source', () => {
  assert.equal(state.APP_VERSION, '1.8.1');
});

test('auth starts unknown and resolves only from a server result', () => {
  assert.equal(state.authTransition(state.AUTH.UNAUTHENTICATED, 'check-started'), state.AUTH.UNKNOWN);
  assert.equal(state.authTransition(state.AUTH.UNKNOWN, 'session-invalid'), state.AUTH.UNAUTHENTICATED);
  assert.equal(state.authTransition(state.AUTH.UNKNOWN, 'session-valid'), state.AUTH.AUTHENTICATED);
  assert.equal(state.authTransition(state.AUTH.UNAUTHENTICATED, 'login-succeeded'), state.AUTH.AUTHENTICATED);
  assert.equal(state.authTransition(state.AUTH.AUTHENTICATED, 'logout-succeeded'), state.AUTH.UNAUTHENTICATED);
});

test('system follows the OS while explicit themes do not', () => {
  assert.equal(state.resolveTheme('system', false), 'light');
  assert.equal(state.resolveTheme('system', true), 'dark');
  assert.equal(state.resolveTheme('light', true), 'light');
  assert.equal(state.resolveTheme('dark', false), 'dark');
});

test('all persisted theme values remain valid', () => {
  for (const preference of ['system', 'light', 'dark']) assert.equal(state.normalizeTheme(preference), preference);
  assert.equal(state.normalizeTheme('invalid'), 'light');
});

const navigationTopics = [
  {id:'first', module_id:'module-a', exam_id:'ml', priority:'core'},
  {id:'foreign', module_id:'module-b', exam_id:'ml'},
  {id:'middle', module_id:'module-a', exam_id:'ml', priority:'support'},
  {id:'other-exam', module_id:'module-a', exam_id:'dl'},
  {id:'last', module_id:'module-a', exam_id:'ml', priority:'core'},
];

test('topic navigation counts first, middle and last without skipping support topics', () => {
  assert.deepEqual(state.topicNavigation(navigationTopics,'first'), {position:1,total:3,previousID:null,nextID:'middle'});
  assert.deepEqual(state.topicNavigation(navigationTopics,'middle'), {position:2,total:3,previousID:'first',nextID:'last'});
  assert.deepEqual(state.topicNavigation(navigationTopics,'last'), {position:3,total:3,previousID:'middle',nextID:null});
});

test('topic navigation never crosses module or exam boundaries', () => {
  assert.deepEqual(state.topicNavigation(navigationTopics,'foreign'), {position:1,total:1,previousID:null,nextID:null});
  assert.deepEqual(state.topicNavigation(navigationTopics,'other-exam'), {position:1,total:1,previousID:null,nextID:null});
});

test('unknown and empty topic navigation has no fabricated destination', () => {
  assert.equal(state.topicNavigation(navigationTopics,'missing'),null);
  assert.equal(state.topicNavigation([],'missing'),null);
});
