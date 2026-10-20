# Phase 8D – final V1 audit

As of: 2026-09-06. Checked exclusively with local content files, the 13
registered original PDFs and the existing application.

## Summary

| Area | Result |
|---|---:|
| Original PDFs | 13 / 13 present |
| Original pages | 621 |
| Topics | 58 (28 ML, 30 DL/DS) |
| Priorities | 45 CORE, 13 SUPPORT, 0 REFERENCE |
| Active questions | 256 (122 ML, 134 DL/DS) |
| Disabled questions | 1, unchanged |
| Database schema | 6 |

## Inventory and sources

The slide ranges in `content/inventory/phase8a-inventory.md` were checked
against `pdfinfo`. Each of the 13 files is represented; per file, the ranges
start at page 1, end at the actual last page and do not overlap. The total is
exactly 621.

| Inventory status | Pages |
|---|---:|
| KEEP | 202 |
| MERGE | 298 |
| EXAMPLE | 19 |
| REFERENCE | 66 |
| DROP_FROM_LEARNING_VIEW | 36 |

KEEP and MERGE are reflected in the 58 topic sources; EXAMPLE ranges are
bundled as examples or explanations in the related topics. REFERENCE and
DROP_FROM_LEARNING_VIEW were deliberately not turned into standalone learning
pages.

All 58 topics have at least one registered PDF source with positive page
numbers that lie within the PDF. The content loader and source validation
loaded these references successfully.

## Topics and questions

| Exam area | Topics | CORE | SUPPORT | Active questions |
|---|---:|---:|---:|---:|
| Machine Learning | 28 | 21 | 7 | 122 |
| Deep Learning & Data Science | 30 | 24 | 6 | 134 |
| Total | 58 | 45 | 13 | 256 |

| Question type | Active questions |
|---|---:|
| Single choice | 108 |
| Multiple select | 41 |
| True / false | 75 |
| Ordering | 10 |
| Matching | 10 |
| Numeric multiple choice | 12 |

All 45 CORE topics have at least four active questions. All 13 SUPPORT topics
have at least one active question; no topic is without an active question.
Questions point to a topic in the same exam area and inherit its validated
original source.

The technical duplicate check found 0 duplicate IDs and 0 identical normalized
active prompts. Similar recall patterns were kept as distinct key-idea,
comprehension or property questions; a later didactic review round can vary
them further if needed.

## Exam pools and filters

| Pool | Active questions |
|---|---:|
| ML – all topics | 122 |
| ML – core only | 104 |
| DL/DS – all topics | 134 |
| DL/DS – core only | 110 |

The earlier pilot notice saying that all topics and core material were
identical was wrong after the content expansion. It was corrected: **all
topics** covers CORE and SUPPORT, **core only** covers CORE exclusively. The
existing server test covers the scope separation; a frontend regression test
protects the updated notice text.

## Live PDF and security check

- `GET /healthz` on localhost and the LAN address → `200 {"status":"ok"}`
- PDF with spaces and range request → `206 Partial Content`
- Introduction PDF with range request → `206 Partial Content`
- URL-encoded traversal attempt → `404`

## Tests

```text
gofmt
go test -count=1 ./...   PASS
go test -race ./...      PASS
go vet ./...             PASS
go build                 PASS
node --check web/app.js  PASS
```

The tests include content/source validation, all six question types, exam
scopes, single submission, scheduler/mastery, backup/restore rollback, PDF
range/encoding/traversal and source visibility.

## Browser, keyboard and known open points

The current execution environment does not provide a controllable browser UI.
Therefore **no** new real viewport, keyboard or light/dark acceptance test is
claimed. The existing automated structural accessibility and responsive tests
are green. A final manual check at 390, 768 and 1440 px remains advisable
before personal use.

No external CDN, font, tracking or API access was added; the application stays
offline-capable on the local network.

## Bugs found and fixed

1. The 8A inventory contained a mistakenly duplicated 133-page block, so the
   audit counted 754 instead of 621 pages. The incorrect block was removed;
   the corrected inventory coverage is 621 with no gaps or overlaps.
2. The exam configuration still showed the pilot notice that all topics and
   core material were identical. The text was updated to the now actually
   different filters and secured with a test.

## Release status

**V1 RELEASE READY** for local learning use. No functional, source or data
integrity blocker was found. The missing repeated real browser acceptance test
is a documented open acceptance point, not a technical blocker.
