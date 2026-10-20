const test = require('node:test');
const assert = require('node:assert/strict');
const {readdirSync,readFileSync} = require('node:fs');
const {join} = require('node:path');
const {createHash} = require('node:crypto');
const renderer = require('./formula-renderer.js');
const katex = require('./vendor/katex/katex.min.js');
const topics = readdirSync(join(__dirname,'..','content','topics')).map(file=>JSON.parse(readFileSync(join(__dirname,'..','content','topics',file),'utf8')));

test('all 49 formula cards and every example expression compile as semantic MathML', () => {
  let count=0, equations=0;
  for(const topic of topics) {
    if(!topic.formula) continue;
    const f=topic.formula;
    const expressions=[...f.latex,...f.worked_example.steps.map(s=>s.latex).filter(Boolean),f.worked_example.result_latex].filter(Boolean);
    for(const expression of expressions) {
      const math=katex.renderToString(expression,renderer.OPTIONS);
      assert.match(math,/<math[^>]+display="block"/,topic.id);
      assert.match(math,/<semantics>/,topic.id);
      assert.match(math,/encoding="application\/x-tex"/,topic.id);
      assert.doesNotMatch(math,/ style=|<img\b|<script\b|<a\b/,topic.id);
      equations++;
    }
    const html=renderer.render(f);
    assert.doesNotMatch(html,/formula-render-error|<pre\b/,topic.id);
    assert.match(html,/Beispielablauf:/,topic.id);
    assert.match(html,/Schritt für Schritt/,topic.id);
    assert.match(html,/Ergebnis und Bedeutung/,topic.id);
    count++;
  }
  assert.equal(count,49);
  assert.ok(equations>=300);
});

test('MathML defines real fractions, subscripts, sums and matrix tables',()=>{
  assert.match(renderer.equation(String.raw`\frac{a}{b}`),/<mfrac>/);
  assert.match(renderer.equation(String.raw`x_i`),/<msub>/);
  assert.match(renderer.equation(String.raw`\sum_{i=1}^N x_i`),/<munderover>/);
  assert.match(renderer.equation(String.raw`\begin{pmatrix}1&2\\3&4\end{pmatrix}`),/<mtable/);
});

test('untrusted commands cannot introduce links, executable HTML or inline CSS',()=>{
  for(const input of [String.raw`\href{https://evil.invalid}{x}`,String.raw`\includegraphics{https://evil.invalid/x}`,String.raw`\htmlStyle{color:red}{x}`,String.raw`\badcommand{x}`]) {
    const html=renderer.equation(input);
    assert.doesNotMatch(html,/<a\b|<img\b|<script\b|<iframe\b| style=| src=| href=/);
  }
  const html=renderer.render({expression:'<script>bad()</script>',meaning:'<img onerror="bad()">',variables:[{symbol:'<b>x</b>',meaning:'<a href="bad">bad</a>'}],worked_example:{title:'<script>bad()</script>',given:['<img src=x>'],steps:[{text:'<b>bad</b>'}],result:'<iframe>'}});
  assert.doesNotMatch(html,/<script\b|<img\b|<b>|<a\b|<iframe\b/);
  assert.match(html,/&lt;script&gt;/);
});

test('errors and legacy formula data are handled without crashing the app',()=>{
  assert.equal(renderer.render(null),'');
  assert.match(renderer.render({expression:'x=y',variables:null}),/formula-fallback/);
  assert.match(renderer.equation(String.raw`\frac{a}`),/formula-render-error/);
  assert.match(renderer.equation('x'.repeat(4097)),/formula-render-error/);
  assert.equal(renderer.OPTIONS.trust,false);
  assert.equal(renderer.OPTIONS.output,'mathml');
  assert.equal(renderer.OPTIONS.maxExpand,1000);
});

test('vendored renderer is pinned and unchanged from the integrity-checked package',()=>{
  assert.equal(katex.version,'0.19.0');
  const hash=createHash('sha256').update(readFileSync(join(__dirname,'vendor','katex','katex.min.js'))).digest('hex');
  assert.equal(hash,'103a53763cc033bba8d175bf3f0ba597c3505c9b6747dd3f2c7bc2a6bfcc8ae7');
});

test('inline math in question text becomes MathML and plain text stays escaped', () => {
  const html = renderer.inline(String.raw`Recall <b> $\frac{TP}{TP+FN}=0.75$ ok`);
  assert.match(html, /^Recall &lt;b&gt; <span class="inline-math">/);
  assert.match(html, /<mfrac>/);
  assert.match(html, / ok$/);
  assert.doesNotMatch(html, / style=|<script\b/);
  assert.equal(renderer.inline('ohne Formel & <i>'), 'ohne Formel &amp; &lt;i&gt;');
  assert.equal(renderer.inline(String.raw`kaputt $\bad{$ <x>`), String.raw`kaputt $\bad{$ &lt;x&gt;`);
  assert.equal(renderer.plain(String.raw`$z=\frac{x-\mu}{\sigma}$ und $\eta$`), 'z=(x-μ)/(σ) und η');
});

test('every inline formula in the question pools compiles', () => {
  const dir = join(__dirname, '..', 'content', 'questions');
  let formulas = 0;
  for (const file of readdirSync(dir)) {
    for (const question of JSON.parse(readFileSync(join(dir, file), 'utf8')).questions) {
      const texts = [question.prompt, question.context, question.explanation, ...(question.options || []).map((option) => option.text)].filter(Boolean);
      for (const text of texts) {
        for (const match of text.matchAll(/\$([^$\n]{1,400})\$/g)) {
          assert.doesNotThrow(() => katex.renderToString(match[1], { ...renderer.OPTIONS, displayMode: false }), `${question.id}: ${match[1]}`);
          formulas++;
        }
      }
    }
  }
  assert.ok(formulas >= 40, `only ${formulas} inline formulas`);
});
