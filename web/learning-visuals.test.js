const test = require('node:test');
const assert = require('node:assert/strict');
const { readdirSync, readFileSync } = require('node:fs');
const { join } = require('node:path');
const visuals = require('./learning-visuals.js');

test('every ML and DL figure renders a local accessible SVG without missing numbers', () => {
  let count = 0;
  const directory = join(__dirname, '..', 'content', 'topics');
  for (const file of readdirSync(directory)) {
    const topic = JSON.parse(readFileSync(join(directory, file), 'utf8'));
    for (let index = 0; index < topic.illustrations.length; index++) {
      const html = visuals.render(topic, index);
      assert.match(html, /<svg[^>]+role="img"/);
      assert.match(html, /<title>[^<]+<\/title>/);
      assert.match(html, /<desc>[^<]+<\/desc>/);
      assert.match(html, /<figcaption>[^<]+<\/figcaption>/);
      assert.doesNotMatch(html, /NaN|Infinity|undefined|<script|<image|<foreignObject/);
      count++;
    }
  }
  assert.equal(count, 116);
});

test('figure text is escaped in SVG and captions', () => {
  const html = visuals.renderFigure({kind:'flow', title:'"<script>alert(1)</script>', caption:'<img onerror="evil()">', labels:['<b>Input</b>','" onclick="evil()']});
  assert.doesNotMatch(html, /<script>|<img|<b>|\sonclick="/);
  assert.match(html, /&lt;script&gt;/);
});

test('legacy topics without figures still render safely', () => {
  assert.equal(visuals.render({}, 0), '');
  assert.equal(visuals.render({illustrations:null}, 1), '');
});
