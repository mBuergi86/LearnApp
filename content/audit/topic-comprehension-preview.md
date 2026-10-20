# Topic comprehension – preview 1.7.0

As of: 2026-10-13. Approved for version 1.7.0 after local preview.
Image build and Zot push are approved, no server deployment.

## Root cause fixed

The earlier "Kannst du es?" (Can you do it?) card linked to
`#/quiz/<exam>/<module>`. As a result, questions about all topics of the
module were loaded already on the first learning page. It was not a test
limited to the topic just read.

## New separation

- On each of the 58 learning pages: short true/false questions for exactly this
  topic, directly on the page, with an explanation after answering and retry.
- 60 existing true/false questions are reused. Eight additional short questions
  cover convolution, confusion matrix, statistics and EDA.
- The additional questions are stored separately in `content/comprehension.json`
  and do not enter the 256 active exam questions or the exam pools.
- `GET /api/comprehension?topic=...` returns no answers or explanations.
  `POST /api/comprehension/answers` checks topic, question and answer
  server-side. The existing session/CSRF middleware remains responsible.
- Quick checks write no reviews, mastery, scheduler state or exam attempts.
  Visited topics are marked on opening as before.
- "Kannst du es?" and module training are only offered on the last topic of the
  module sequence. The separate practice selection remains available unchanged.
  This is not a new enforced reading lock and not proof that all texts were read.
- 17 quiz prompts now ask directly about subject knowledge, not about slide
  wording. Question IDs, types, options and answer keys stay the same.

## PDF coverage: still open

The existing inventory `content/inventory/phase8a-inventory.md` distinguishes
KEEP, MERGE, EXAMPLE, REFERENCE and DROP_FROM_LEARNING_VIEW. The older audit
lists 621 original pages, including 66 REFERENCE and 36 DROP pages. Therefore
the app so far is not a proven verbatim transfer of all PDFs. Registered
sources and working PDF links alone do not prove complete subject coverage
either. A new slide-by-slide comparison and a decision on "complete in
substance" versus "verbatim" are still pending.

## Verification

Automated content/HTTP/frontend tests check topic binding, both answer cases,
no answer leaks, retry, delayed stale answers, unchanged learning state,
session/CSRF protection and last-only module training. The running `go run`
preview must be restarted to pick up the changed embedded client. A visual
check of the new quick checks is still open.
