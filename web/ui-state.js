(function initializeUIState(scope) {
  const APP_VERSION = '1.8.2';
  const AUTH = Object.freeze({ UNKNOWN: 'unknown', AUTHENTICATED: 'authenticated', UNAUTHENTICATED: 'unauthenticated' });
  const THEMES = Object.freeze(['system', 'light', 'dark']);

  function normalizeTheme(preference) {
    return THEMES.includes(preference) ? preference : 'light';
  }

  function resolveTheme(preference, systemPrefersDark) {
    const normalized = normalizeTheme(preference);
    return normalized === 'system' ? (systemPrefersDark ? 'dark' : 'light') : normalized;
  }

  function authTransition(current, event) {
    if (event === 'check-started') return AUTH.UNKNOWN;
    if (event === 'session-valid' || event === 'login-succeeded') return AUTH.AUTHENTICATED;
    if (event === 'session-invalid' || event === 'logout-succeeded') return AUTH.UNAUTHENTICATED;
    return current;
  }

  function topicNavigation(topics, topicID) {
    const topic = topics.find((item) => item.id === topicID);
    if (!topic) return null;
    // Use the same catalog order as the module overview. Include support
    // topics too; this navigation must not silently skip existing material.
    const siblings = topics.filter((item) => item.module_id === topic.module_id && item.exam_id === topic.exam_id);
    const index = siblings.findIndex((item) => item.id === topicID);
    return { position: index + 1, total: siblings.length, previousID: siblings[index - 1]?.id || null, nextID: siblings[index + 1]?.id || null };
  }

  const model = Object.freeze({ APP_VERSION, AUTH, THEMES, normalizeTheme, resolveTheme, authTransition, topicNavigation });
  scope.ExamTrainerUIState = model;
  if (typeof module !== 'undefined' && module.exports) module.exports = model;
})(globalThis);
