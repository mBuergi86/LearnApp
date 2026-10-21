# Release 1.8.1

As of: 2026-10-21. Content-only patch on top of 1.8.0.

## Why

The exam does not test programming. Questions that required knowing pandas or
PyTorch syntax are removed or rewritten in plain words.

## Content

- Disabled `exam-ml-rolling-vs-resample` (pandas `rolling`/`resample`/`pd.cut`)
- Disabled `exam-dl-pytorch-graph` (PyTorch vs TensorFlow graph)
- `exam-ml-dropna-axis`: same idea (drop the column vs drop the rows), now
  without `dropna`/`fillna` code in prompt, options and explanation
- `exam-ml-hidden-missing`: `isna().sum()` replaced by a plain description
- `exam-ml-binning-type`: removed the `pd.cut` reference

IDs are kept, so answer history stays valid.

Pools (all/core): ML 138/122, DL/DS 130/104. Active exam questions: 268.
No code, rendering or schema changes. SQLite schema stays at 7.

## Test results

| Check | Result |
|---|---|
| gofmt, go vet | PASS |
| go test -count=1 ./... | PASS except the two tests that need `DeepLearning_Tag1_Einfuhrung_2026_ShaoJuWoo.pdf` (missing in the build environment) |
| node --test web/*.test.js | PASS, 23 tests |
