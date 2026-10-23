# Release 1.7.0

As of: 2026-10-13. Approved by the user after preview; image built and pushed
to Zot. No server access and no deployment.

## Image

- Tag: `registry.marbui.ch/learn/mldl-exam-trainer:1.7.0`
- Local tag: `localhost/mldl-exam-trainer:1.7.0`
- Platform: linux/amd64, CGO disabled.
- Image ID: `f023b411c481c922f38643c9702d6dd38eff561b73e4fdb131e338049425702c`
- Registry digest: `sha256:a65b95c7421eb5ac57fc6a3df3bf282132caf5c2f0600390c3d6d08a56c921d8`
- Size reported after a fresh registry pull: 117770089 bytes (about 112.31 MiB).
- A registry pull confirmed identical image ID, platform and OCI version label 1.7.0.
- Earlier tags are kept. No latest tag replaced.
- Version source: `web/ui-state.js`; footer and OCI label take 1.7.0 from it.

## Changes

- Real LaTeX math syntax is typeset locally with KaTeX 0.19.0 as native MathML.
  No external fonts/CDNs and no CSP relaxation.
- 49 formula cards with symbol explanations, numbered simple worked examples
  and the meaning of the result. All 49 calculation results independently
  recalculated.
- Topic-specific quick checks on all 58 learning pages, explanation after each
  answer, retry possible, without changing reviews/mastery/scheduler/exam state.
- 60 existing true/false questions plus eight additional short questions; the
  additional questions do not enter the exam pools.
- Module training / "Kannst du es?" (Can you do it?) only on the last topic of
  a module sequence.
- 17 question prompts changed from recalling the lecture to subject knowledge;
  existing IDs, types, options and answer keys preserved.
- Separate login-free development preview `go run ./cmd/preview`, strictly
  loopback and a fresh temporary DB. Not in the runtime image; no auth-disable
  switch in the production server. Login in the image stays enabled.

## Test results

| Check | Result |
|---|---|
| gofmt | PASS |
| go test -count=1 ./... | PASS, 130 top-level tests / 374 cases including subtests |
| go test -race ./... | PASS |
| go vet ./... | PASS |
| go build ./... | PASS |
| node --check web/app.js and browser test script | PASS |
| node --test web/*.test.js | PASS, 21 tests |
| Initial auth state without flicker | PASS in container/Chrome |
| Login, session after reload, desktop/mobile logout | PASS in container/Chrome |
| Footer login/app, light/dark, normal document position | PASS |
| Theme system/light/dark, persistence on reload, live OS change | PASS |
| Topic layout at 320/375/390/1024/1440 px in light/dark | PASS, 580 views |
| SVG illustrations | PASS, 1160 rendered illustrations |
| Math typesetting | PASS, 490 card views / 3090 MathML expressions |
| Quick checks correct/wrong, keyboard, retry, learning state unchanged | PASS in real Chrome |
| Topic navigation first/middle/last, module training only at the end | PASS, mobile/desktop for ML and DL/DS |
| CSP violations / external browser runtime requests | none |
| UID/GID | PASS, 10001:10001 |
| Read-only root FS | PASS; writes to /app rejected |
| cap-drop=ALL | PASS, CapEff=0 |
| no-new-privileges | PASS, NoNewPrivs=1 |
| seccomp | PASS, Seccomp=2 |
| Production login instead of preview in the image | PASS; hardened container tested with regular login |
| Backup/restore, learning features, scheduler/mastery/exam regression suite | PASS, Go suite |
| Fully network-isolated container/restart test | NOT TESTED, execution policy blocked the extra command |

The browser test needed two fixes, both only in the test driver: wait for
asynchronously loaded quick checks before clicking navigation, and also deliver
the native character event on CDP Enter. The complete final run passed. The
MathML screenshots for a neuron in light mode and logistic regression in dark
mode were actually looked at. No purely visual PASS based on structural tests.

## Data and unchanged pools

- SQLite schema: 7. No new DB migration.
- ML: 28 topics, 122 questions total / 104 core.
- DL/DS: 30 topics, 134 questions total / 110 core.
- Total: 58 topics, 256 active exam questions, 13 PDFs.
- Production data/volumes were neither used nor changed.
- Be sure to keep the existing server configuration: PVC `mldl-data`, mount
  `/data`, `APP_DB_PATH=/data/app.db`, existing trusted proxy CIDRs.

## Open points

- Full subject or verbatim 1:1 PDF coverage is not confirmed. The older
  inventory also uses MERGE, REFERENCE and DROP. A new slide-by-slide
  comparison is still pending; this release does not claim it.
- The extra network-isolated test and container restart were not performed
  because of the execution policy. Based on observed requests, the browser
  runtime needs no external services; existing offline structural tests pass.
- Modern browsers with native MathML support are required. Wide formulas
  deliberately scroll horizontally within their card.
- The local test container `mldl-release-170-smoke` and its test volume
  `mldl-release-170-smoke-data` are kept for now, loopback port 18080. They
  contain only throwaway test data, no production learning progress. Optional
  cleanup by the user:

```powershell
podman rm -f mldl-release-170-smoke
podman volume rm mldl-release-170-smoke-data
```

These commands only affect the release test resources named above, not
`mldl-exam-data`, production PVCs or the running user preview on port 18087.

Addendum 2026-10-20: container `mldl-release-170-smoke` and volume
`mldl-release-170-smoke-data` were removed. `mldl-exam-data` was left unchanged.

## New/changed files since 1.6.0

- `Containerfile`, `README.md`, `CONTAINER.md`, `DEPLOYMENT.md`, `RELEASE-1.7.0.md`
- `cmd/preview/main.go`, `cmd/preview/main_test.go`
- `internal/content/model.go`, `internal/content/loader.go`
- `internal/content/formula.go`, `internal/content/formula_test.go`
- `internal/content/comprehension.go`, `internal/content/comprehension_test.go`
- `internal/http/server.go`, `internal/http/comprehension.go`, `internal/http/comprehension_test.go`
- `web/ui-state.js`, `web/ui-state.test.js`, `web/app.js`, `web/styles.css`, `web/index.html`
- `web/embed.go`, `web/embed_test.go`
- `web/formula-renderer.js`, `web/formula-renderer.test.js`, `web/formula-calculations.test.js`
- `web/topic-checks.test.js`
- `web/vendor/katex/katex.min.js`, `web/vendor/katex/LICENSE`, `web/vendor/katex/provenance.json`
- `tools/browser-smoke.mjs`, `tools/dl-browser-smoke.mjs`
- `content/comprehension.json`, `content/audit/topic-comprehension-preview.md`
- `content/questions/phase3-sample.json`, `phase5-expansion.json`,
  `phase8-ml-foundations-preprocessing.json`, `phase8-ml-models-selection.json`,
  `phase8-ml-trees-coverage.json` (text only; IDs/answers preserved).

Formula/example additions in these 49 topic files:

- `content/topics/activation-functions.json`
- `content/topics/artificial-neuron.json`
- `content/topics/backpropagation.json`
- `content/topics/batch-iteration-epoch.json`
- `content/topics/batch-normalization.json`
- `content/topics/classic-attention.json`
- `content/topics/classification-threshold.json`
- `content/topics/cnn-convolution.json`
- `content/topics/cnn-spatial-dimensions.json`
- `content/topics/collinearity.json`
- `content/topics/confusion-matrix.json`
- `content/topics/cross-validation.json`
- `content/topics/data-cleaning-noise.json`
- `content/topics/decision-trees-model.json`
- `content/topics/dl-loss-functions.json`
- `content/topics/dl-regularization.json`
- `content/topics/eda-correlation-outliers.json`
- `content/topics/eda-statistics.json`
- `content/topics/eda-workflow.json`
- `content/topics/ensemble-basics.json`
- `content/topics/feature-encoding.json`
- `content/topics/feature-scaling.json`
- `content/topics/features-datatypes.json`
- `content/topics/gradient-descent.json`
- `content/topics/hyperparameters-bias-variance.json`
- `content/topics/k-means.json`
- `content/topics/linear-regression-model.json`
- `content/topics/logistic-regression-model.json`
- `content/topics/loss-functions-ml.json`
- `content/topics/lstm.json`
- `content/topics/missing-values.json`
- `content/topics/ml-lifecycle.json`
- `content/topics/model-training-gradient.json`
- `content/topics/multi-head-attention.json`
- `content/topics/neural-network-architecture.json`
- `content/topics/overfitting-underfitting.json`
- `content/topics/performance-metrics.json`
- `content/topics/pooling-resnet.json`
- `content/topics/positional-information.json`
- `content/topics/query-key-value.json`
- `content/topics/random-forest.json`
- `content/topics/regression-bias-term.json`
- `content/topics/regression-optimization.json`
- `content/topics/rnn-basics.json`
- `content/topics/rnn-bptt.json`
- `content/topics/self-attention.json`
- `content/topics/supervised-unsupervised.json`
- `content/topics/train-validation-test.json`
- `content/topics/vanishing-gradients.json`

There was no Git repository in the workspace at the time; this list is not the
output of a Git diff but the documented change list for this working state.
