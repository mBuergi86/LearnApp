# Release 1.8.2

As of: 2026-10-23. Rendering patch on top of 1.8.1. The image was built and
pushed to Zot on 2026-10-23; no server deployment.

## Why

Seven questions still wrote math with Unicode symbols instead of LaTeX, so they
did not render as MathML like the rest of the pools.

## Content and rendering

- Cluster union and intersection, the BP3/BP4 gradients, the neuron sum and the
  VIF feature now use LaTeX (`content/questions/phase5-expansion.json`,
  `phase8-dl-generated.json`, `phase9-exam-style.json`).
- The plain-text fallback for places without markup (aria labels, `<option>`)
  understands more symbols: φ, ∂, ⋃, ∩, ∅, ∞ (`web/formula-renderer.js`).

IDs are kept, so answer history stays valid.

Pools (all/core): ML 138/122, DL/DS 130/104. Active exam questions: 268.
No pool, authentication or schema changes. SQLite schema stays at 7, no
migration.

## Image

| Item | Value |
|---|---|
| Source commit | `3bd1564` (`main`) |
| Local tag | `localhost/mldl-exam-trainer:1.8.2` |
| Registry tag | `registry.marbui.ch/learn/mldl-exam-trainer:1.8.2` |
| Image ID | `5875e9ae9520` |
| Manifest digest | `sha256:1ef513810ada94fc5745f66d8c1bd5c3834d59c2c35bd218dd137d18c20108ff` |
| Platform | `linux/amd64`, OCI label `org.opencontainers.image.version=1.8.2` |

The tag did not exist in the registry before the push; no tag was overwritten.
There is no `1.8.1` image: the registry goes from `1.8.0` straight to `1.8.2`,
which contains all 1.8.1 changes.

The course PDFs were not present in the build environment, so this image (like
`1.8.0`) contains no `content/pdfs/`. The app starts with
`PDF momentan nicht verfügbar` warnings and the learning pages show the
missing-source notice. To ship the slides, place the 13 PDFs in
`content/pdfs/` and rebuild under the same version only if the tag has not been
deployed yet; otherwise release a new patch version.

## Test results

| Check | Result |
|---|---|
| gofmt, go vet | PASS |
| go test ./... | PASS except `TestProjectContentProvidesThirtyQuestionsPerExam` and `TestProjectPilotTopicsReferenceTheirCoursePDFs`: all 13 PDFs were missing in the build environment (environment only, not a code issue) |
| node --test web/*.test.js | PASS, 23 tests |
| Container smoke test (fresh volume, test admin): `/` and `/healthz` | PASS, HTTP 200; OCI label shows 1.8.2 |
| Browser smoke scripts, go test -race | NOT TESTED for 1.8.2 |
