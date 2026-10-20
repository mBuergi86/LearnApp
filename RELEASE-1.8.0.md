# Release 1.8.0

As of: 2026-10-20. Content and readability release for exam practice.
No image build, no registry push and no deployment yet.

## Why

Practice was dominated by calculation questions that do not match the open-book
exam. Formulas in questions were shown as raw text, and the text size was hard
to read. This release shifts practice to conceptual exam-style questions and
renders math properly.

## Content

- 53 numeric-choice questions are disabled (`enabled: false`). Their IDs stay
  in the content files, so existing answer history remains valid.
- 10 typical calculation questions stay active and were rewritten with LaTeX
  context, prompt and explanation:
  `ml-confusion-metrics`, `ml-performance-f1`, `ml-loss-optimize`,
  `ml-tree-impurity`, `ml-scaling-standard`, `ml-logistic-probability`,
  `dl-cnn-output-width`, `dl-cnn-spatial-dimensions-detail`,
  `dl-cnn-architecture-detail`, `dl-batch-iteration-epoch-core`.
- New file `content/questions/phase9-exam-style.json` with 67 exam-style
  questions (44 single-choice, 16 true/false, 7 multiple-select). They cover
  scenarios, pitfalls and comparisons across all ML and DL days: data types,
  EDA, leakage, encoding, VIF, metrics, cross-validation, trees and ensembles,
  k-means, activation functions, BatchNorm/LayerNorm, CNN, RNN/LSTM, seq2seq,
  attention and Transformer.
- Multi-Hot encoding now follows the course definition (binary code over
  ⌈log₂ k⌉ columns). The topic page `feature-encoding` and the question
  `ml-encoding-multihot` were corrected. The topic page notes that outside the
  course "multi-hot" often means multi-label encoding.
- `dl-backprop-error-definition` no longer gives the answer away in the prompt;
  the options now describe meanings instead of formulas.

Pools (all/core): ML 139/122, DL/DS 131/105. Active exam questions: 270.
Quick checks: 84.

## Rendering and readability

- Inline math with `$...$` is typeset locally with KaTeX as MathML in practice,
  exam, results, answer comparison, ordering, matching and topic checks.
  No CDN and no CSP change. Invalid formulas fall back to escaped text.
- Places that cannot contain markup (aria labels, `<option>`, screen-reader
  legends) use a plain-text conversion of the formula.
- Larger text: body 18 px, questions, options, context and explanations
  19 px, higher line height around formulas. The given data (`context`) now
  has its own highlighted box.

## Unchanged

- Authentication, CSRF, sessions, proxy handling, scheduler, mastery,
  backup/restore.
- SQLite schema: 7. No new DB migration.

## Test results

| Check | Result |
|---|---|
| gofmt | PASS |
| go vet ./... | PASS |
| go build ./... | PASS |
| go test -count=1 ./... | PASS except `TestProjectPilotTopicsReferenceTheirCoursePDFs`: the PDF `DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf` was not present in the build environment (environment only, not a code issue) |
| node --test web/*.test.js | PASS, 23 tests |
| New test: every inline formula in the question pools compiles | PASS |
| Browser check (preview server): fractions render as MathML, no console errors | PASS |
| Container build, browser smoke scripts, go test -race | NOT TESTED for 1.8.0 |

## Changed files since 1.7.1

- `content/questions/phase3-sample.json`, `phase5-expansion.json`,
  `phase8-dl-generated.json`, `phase8-ml-foundations-preprocessing.json`,
  `phase8-ml-models-selection.json`, `phase8-ml-trees-coverage.json`
- `content/questions/phase9-exam-style.json` (new)
- `content/topics/feature-encoding.json`
- `web/app.js`, `web/formula-renderer.js`, `web/styles.css`, `web/index.html`,
  `web/ui-state.js`
- Tests: `web/*.test.js`, `web/embed_test.go`,
  `internal/content/*_test.go`, `internal/quiz/*_content_test.go`
- `tools/browser-smoke.mjs`, `tools/dl-browser-smoke.mjs` (version string)
