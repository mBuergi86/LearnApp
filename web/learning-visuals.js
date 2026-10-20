(function initializeLearningVisuals(scope) {
  const escape = (value = '') => String(value).replace(/[&<>"']/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character]);
  const text = (x, y, value, extra = '') => `<text x="${x}" y="${y}" ${extra}>${escape(value)}</text>`;
  const line = (x1, y1, x2, y2) => `<line x1="${x1}" y1="${y1}" x2="${x2}" y2="${y2}" class="visual-line"/>`;
  const arrow = (y) => `<path d="M180 ${y} v16 m-5 -5 l5 5 l5 -5" class="visual-line"/>`;
  const box = (y, label, width = 312, x = 24) => {
    const [heading, detail] = String(label).split('|');
    return `<rect x="${x}" y="${y}" width="${width}" height="58" rx="9" class="visual-box"/>${text(x + width / 2, y + 24, heading, 'text-anchor="middle"')}${detail ? text(x + width / 2, y + 45, detail, 'text-anchor="middle" class="visual-detail"') : ''}`;
  };
  function draw(figure) {
    const labels = Array.isArray(figure.labels) ? figure.labels : [];
    const values = Array.isArray(figure.values) ? figure.values : [];
    if (figure.kind === 'nested') {
      return { height: 240, body: labels.map((label, i) => `<rect x="${18 + i * 32}" y="${12 + i * 58}" width="${324 - i * 64}" height="${218 - i * 58}" rx="12" class="visual-box"/>${text(180, 40 + i * 58, label, 'text-anchor="middle"')}`).join('') };
    }
    if (figure.kind === 'matrix') {
      const rows = values.length, cols = values[0]?.length || 1;
      const cell = Math.min(64, 300 / cols);
      const start = (360 - cols * cell) / 2;
      return { height: rows * cell + 54, body: text(180, 24, labels[0] || '', 'text-anchor="middle"') + values.map((row, i) => row.map((value, j) => `<rect x="${start + j * cell}" y="${38 + i * cell}" width="${cell - 3}" height="${cell - 3}" rx="5" class="visual-cell"/>${text(start + j * cell + (cell - 3) / 2, 38 + i * cell + cell / 2 + 4, value, 'text-anchor="middle"')}`).join('')).join('') };
    }
    if (figure.kind === 'bars') {
      const numbers = values[0] || [];
      const max = Math.max(1, ...numbers.map(Math.abs));
      return { height: labels.length * 64 + 16, body: labels.map((label, i) => text(24, 24 + i * 64, `${label}: ${numbers[i]}`, 'class="visual-detail"') + `<rect x="24" y="${32 + i * 64}" width="${Math.abs(numbers[i] || 0) / max * 300}" height="20" rx="4" class="visual-bar"/>`).join('') };
    }
    if (figure.kind === 'plot') {
      const xs = values.map((point) => point[0]), ys = values.map((point) => point[1]);
      const xmin = Math.min(...xs), xmax = Math.max(...xs), ymin = Math.min(0, ...ys), ymax = Math.max(1, ...ys);
      const px = (x) => 50 + (x - xmin) / (xmax - xmin || 1) * 270;
      const py = (y) => 210 - (y - ymin) / (ymax - ymin || 1) * 160;
      return { height: 274, body: line(50, 40, 50, 210) + line(50, 210, 330, 210) + text(24, 25, labels[1]) + text(190, 264, labels[0], 'text-anchor="middle"') + text(50, 235, xmin, 'text-anchor="middle"') + text(320, 235, xmax, 'text-anchor="middle"') + text(40, 56, ymax, 'text-anchor="end"') + `<polyline points="${values.map(([x, y]) => `${px(x)},${py(y)}`).join(' ')}" class="visual-curve"/>` + values.map(([x, y]) => `<circle cx="${px(x)}" cy="${py(y)}" r="3" class="visual-point"/>`).join('') };
    }
    if (figure.kind === 'network') {
      const sizes = values[0] || [2, 3, 1];
      const xs = sizes.map((_, i) => 58 + i * 244 / Math.max(1, sizes.length - 1));
      const ys = sizes.map((size) => Array.from({ length: Math.min(4, size) }, (_, i) => 70 + i * 44));
      const connections = xs.slice(1).map((x, i) => ys[i].map((y) => ys[i + 1].map((nextY) => line(xs[i], y, x, nextY)).join('')).join('')).join('');
      return { height: 294, body: connections + xs.map((x, i) => ys[i].map((y) => `<circle cx="${x}" cy="${y}" r="15" class="visual-box"/>`).join('') + text(x, 28, labels[i], 'text-anchor="middle" class="visual-detail"') + text(x, 270, `${sizes[i]} Werte`, 'text-anchor="middle" class="visual-detail"')).join('') };
    }
    if (figure.kind === 'residual') {
      return { height: 260, body: box(12, labels[0]) + arrow(76) + box(100, labels[1]) + arrow(164) + box(188, labels[2]) + `<path d="M336 41 h12 v176 h-12" class="visual-line"/>` + text(344, 145, '+', 'text-anchor="middle"') };
    }
    return { height: labels.length * 84 + 8, body: labels.map((label, i) => box(8 + i * 84, label) + (i < labels.length - 1 && figure.kind !== 'comparison' ? arrow(70 + i * 84) : '')).join('') };
  }
  function renderFigure(figure) {
    if (!figure || !figure.title || !figure.caption) return '';
    const drawing = draw(figure);
    return `<figure class="learning-figure"><h3>${escape(figure.title)}</h3><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 360 ${drawing.height}" role="img" aria-label="${escape(figure.title + '. ' + figure.caption)}"><title>${escape(figure.title)}</title><desc>${escape(figure.caption)}</desc>${drawing.body}</svg><figcaption>${escape(figure.caption)}</figcaption></figure>`;
  }
  function render(topic, index) {
    return renderFigure(Array.isArray(topic.illustrations) ? topic.illustrations[index] : undefined);
  }
  scope.ExamTrainerLearningVisuals = Object.freeze({ render, renderFigure });
  if (typeof module !== 'undefined' && module.exports) module.exports = scope.ExamTrainerLearningVisuals;
})(globalThis);
