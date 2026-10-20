# Release Notes

## v1.8.0

- 53 calculation questions disabled; 10 typical ones kept and rewritten with
  real LaTeX in context, prompt and explanation
- 67 new exam-style questions (scenarios, pitfalls, comparisons) across all
  ML and DL topics
- Inline math (`$...$`) is typeset locally as MathML in practice, exam,
  results and topic checks; no CSP change
- Larger text for questions, options, given data and explanations
- Multi-Hot encoding corrected to the binary course definition

Pools (all/core): ML 139/122, DL/DS 131/105. The database schema stays at
version 7; no migration. Details are in [RELEASE-1.8.0.md](RELEASE-1.8.0.md).

## v1.7.1

- Fixed the login error `403 Fremde Herkunft abgelehnt.` (foreign origin
  rejected) behind the TLS proxy: same-origin requests confirmed by the browser
  (`Sec-Fetch-Site: same-origin` with a matching host) are no longer checked
  against the server-side detected scheme
- Foreign origins and cross-site requests are still rejected; forwarded headers
  are still only accepted from configured trusted proxies

CSRF protection, Argon2id, sessions, learning content, exam pools, scheduler,
mastery, PDFs and backup/restore are unchanged from v1.7.0. The database schema
stays at version 7; there is no new migration for v1.7.1. Details are in
[RELEASE-1.7.1.md](RELEASE-1.7.1.md).

## v1.7.0

- LaTeX math syntax is typeset locally with KaTeX 0.19.0 as native MathML;
  no CDNs, no external fonts and no relaxation of the CSP
- 49 formula cards with symbol explanations, numbered simple worked examples
  and the meaning of the result, also in the Today learning views
- Quick checks with true/false questions on all 58 learning pages (60 existing
  plus 8 additional questions), with no effect on reviews, mastery, scheduler
  or exam statistics; the additional questions do not enter the exam pools
- "Kannst du es?" (Can you do it?) and module training only on the last topic
  of a module sequence
- 17 question prompts now ask about subject knowledge instead of the lecture or
  slides; IDs, types, options and answers are unchanged
- Separate login-free development preview `go run ./cmd/preview`, loopback only
  and with a fresh temporary database; not part of the runtime image

Auth, theme, exam grading, scheduler, mastery and the PDF system are unchanged
from v1.6.0. The database schema stays at version 7; there is no new migration
for v1.7.0. Details are in [RELEASE-1.7.0.md](RELEASE-1.7.0.md).

## v1.6.0

- Topic position at top and bottom, for example `Thema 1 / 3` (Topic 1 / 3)
- At the end of the page: previous topic, next topic and `Training`
- On the first topic, Back leads to the module overview; after the last topic,
  Next leads to the exam area
- Order and count match the module overview including supplementary topics;
  Next changes neither mastery nor status
- Native links, visible keyboard focus and mobile touch targets; no sticky
  navigation over learning content

Auth, themes, learning content and learning logic are unchanged from v1.5.0.
The database schema stays at version 7; there is no new migration for v1.6.0.

## v1.5.0

- All 28 ML topics didactically revised: own explanations, step sequences,
  worked examples, common confusions and two local SVG illustrations each
- All 30 DL/DS topics deepened: five explanatory paragraphs, step sequence,
  example, confusions, two illustrations, plus prerequisites and help with
  reading formulas; 116 illustrations in total, also in the Today learning step
- Formulas explain their symbols
- All 104 generated DL/DS questions reworked; duplicate answer options and
  statements that were "wrong" only because of their topic assignment removed
- All 122 ML questions with new or deeper explanations; 29 ML and 34 DL
  calculation questions are tested with independent calculations
- The content loader rejects identical answer texts and matching targets
- Security hardening: forwarded headers only from configured trusted proxies
  (`APP_TRUSTED_PROXY_CIDRS`); the origin check also compares the scheme;
  Argon2 verification also for unknown users; stricter login rate limit;
  limit on parallel Argon2 verifications; periodic session cleanup; explicit
  server timeouts; Go 1.26.9

Topic and question IDs, priorities, sources and exam pools (ML 122/104,
DL/DS 134/110) are preserved. Historical scores are not recalculated
retroactively. Auth flow, logout, theme, scheduler, mastery, exam and PDFs are
functionally unchanged. The database schema stays at version 7; there is no new
migration for v1.5.0. Details are in the
[content review](content/audit/dl-ds-quality-review.md) and the
[middleware audit](content/audit/middleware-security-review.md).

## v1.4.0

Not released. v1.3.0 was followed directly by v1.5.0; the review build
`dl-ds-review-20261012` built in between still carried version 1.3.0 internally.

## v1.3.0

- Visible version display `ML/DL Exam Trainer · v1.3.0` in the shared footer
- Footer available on login and app pages, on mobile and desktop

Auth, logout, theme logic, learning content, scheduler, mastery, exam, PDFs and
backup/restore are unchanged from v1.2.0. The database schema stays at
version 7; there is no new migration for v1.3.0.

## v1.2.0

- Fixed initial auth/dashboard flicker on page load
- Added mobile logout
- Added theme selection System / Light / Dark
- Made the theme selection persistent
- Improved mobile and desktop UX for auth/theme

Learning content, questions, exam pools, scheduler, mastery, PDF and
backup/restore logic are unchanged from v1.1. The database schema stays at
version 7; there is no new migration for v1.2.0.
