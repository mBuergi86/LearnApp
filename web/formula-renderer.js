(function initializeFormulaRenderer(scope) {
  const escape = (value = '') => String(value).replace(/[&<>"']/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character]);
  const OPTIONS = Object.freeze({ output: 'mathml', displayMode: true, throwOnError: true, strict: 'error', trust: false, maxExpand: 1000, maxSize: 10 });
  function equation(latex) {
    try {
      if (typeof latex !== 'string' || !latex.trim() || latex.length > 4096) throw new Error('Invalid formula');
      const renderer = scope.katex || (typeof module !== 'undefined' ? require('./vendor/katex/katex.min.js') : null);
      // Native MathML avoids the inline CSS in KaTeX's HTML output. It needs
      // neither unsafe-inline CSP nor a CDN/font download.
      return `<div class="formula-equation" tabindex="0" role="group" aria-label="Mathematische Formel, bei Bedarf horizontal scrollbar">${renderer.renderToString(latex, OPTIONS)}</div>`;
    } catch {
      return '<p class="formula-render-error" role="status">Diese Formel konnte nicht dargestellt werden.</p>';
    }
  }
  function render(formula) {
    if (!formula) return '';
    const variables = Array.isArray(formula.variables) ? formula.variables : [];
    const latex = Array.isArray(formula.latex) ? formula.latex : [];
    const definitions = latex.length ? latex.map(equation).join('') : `<p class="formula-fallback">${escape(formula.expression)}</p>`;
    const example = formula.worked_example;
    const worked = example ? `<section class="formula-worked-example" aria-label="Beispielablauf zur Formel"><h3>Beispielablauf: ${escape(example.title)}</h3><h4>Gegeben</h4><ul>${(Array.isArray(example.given) ? example.given : []).map((value) => `<li>${escape(value)}</li>`).join('')}</ul><h4>Schritt für Schritt</h4><ol class="formula-example-steps">${(Array.isArray(example.steps) ? example.steps : []).map((step) => `<li><p>${escape(step.text)}</p>${step.latex ? equation(step.latex) : ''}</li>`).join('')}</ol><div class="formula-result"><h4>Ergebnis und Bedeutung</h4><p>${escape(example.result)}</p>${example.result_latex ? equation(example.result_latex) : ''}</div></section>` : '';
    return `<section class="learning-section formula-section"><h2>Wichtige Formel</h2><div class="formula-card"><div class="formula-definitions">${definitions}</div><p>${escape(formula.meaning)}</p><p class="formula-help">Breite Formeln kannst du bei Bedarf seitlich scrollen.</p><dl>${variables.map((variable) => `<div><dt>${escape(variable.symbol)}</dt><dd>${escape(variable.meaning)}</dd></div>`).join('')}</dl>${worked}</div></section>`;
  }
  const api = Object.freeze({ render, equation, OPTIONS });
  scope.ExamTrainerFormulaRenderer = api;
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
})(globalThis);
