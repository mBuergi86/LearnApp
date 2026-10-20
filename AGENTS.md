# AGENTS.md

Guidance for AI coding agents (Claude Code, Codex, etc.) working in this repository.
Full documentation: [README.md](README.md), [CONTAINER.md](CONTAINER.md),
[DEPLOYMENT.md](DEPLOYMENT.md), specification in
[ML_DL_Exam_Trainer_V1_Specification.md](ML_DL_Exam_Trainer_V1_Specification.md).

## Project

ML/DL Exam Trainer: a learning and exam web app for the modules Machine Learning
and Deep Learning & Data Science. Go server with an embedded vanilla JS frontend,
SQLite (pure Go via `modernc.org/sqlite`, no CGO), content stored as JSON files.

- **Language:** commit messages, README and new documentation are written in
  **English**. The user interface and learning content (`web/`, `content/`)
  stay in **German**; keep new UI strings and questions German.
- No npm/Node dependencies and no frontend build. Node.js is only used
  optionally for `node:test` tests and syntax checks.

## Layout

```text
cmd/server/            production server (the only container entrypoint)
cmd/preview/           local preview without login, loopback only, temporary DB
internal/auth/         Argon2id, sessions, CSRF, trusted proxies
internal/content/      loading and validating content/*.json, formulas, illustrations
internal/db/           SQLite schema (currently 7), migrations, backup/restore, exams
internal/http/         HTTP server, routes, security headers, same-origin check
internal/progress/     mastery and scheduler
internal/quiz/         question types, grading, shuffling
web/                   frontend (embedded in the binary via embed.go)
web/vendor/katex/      KaTeX bundled locally, MathML output only, origin in provenance.json
content/catalog.json   exam areas, modules, PDF registry
content/topics/        learning pages (one JSON file per topic)
content/questions/     question pools
content/audit/, content/inventory/  audits and slide inventory (docs, not code)
tools/                 browser smoke tests (require Chrome and credentials)
```

Not in the repository (see `.gitignore`): `data/` (local DB, caches),
`content/pdfs/` (copyrighted original course slides), the `Machine Learning/`
and `Tag …/` folders, and all build artifacts (`*.exe`, `*.tar`). Missing PDFs
only produce a warning at startup.

## Commands

```powershell
go test ./...
go vet ./...
go build ./cmd/server
gofmt -l ./cmd ./internal ./web     # must print nothing
go run ./cmd/preview                # http://127.0.0.1:18087, no login
node --test "web/*.test.js"         # JS tests, if Node is available
node --check web/app.js
```

Before finishing a change, run at least `go test ./...`, `go vet ./...` and
`gofmt`; for changes in `web/`, also run the Node tests.

## Rules

- **Do not weaken security.** No relaxing the CSP (no `unsafe-inline`, no CDN),
  no opening CORS, no auth-disable switch in the production server.
  `cmd/preview` must never end up in the container image. Cookies stay `Secure`.
- **No secrets in the repo.** `ADMIN_USERNAME`/`ADMIN_PASSWORD` only as
  environment variables or a Kubernetes Secret, never in code, docs examples
  with real values, the `Containerfile` or commits.
- **Database.** Schema changes only as a new, numbered migration in
  `internal/db`; never edit existing migrations. Keep the backup format and
  restore validation (`internal/db/backup.go`) in sync with schema changes.
  Never test against `data/exam.db`; tests and preview use temporary DBs.
- **Content.** Question and topic IDs are stable (progress and backups
  reference them) and are never renamed or reused. New content must pass the
  validation in `internal/content/loader.go` and carry valid sources
  (`content/pdfs/<file>`, page numbers). Base subject matter on the course
  material; do not invent facts.
- **Version.** The single version source is `APP_VERSION` in `web/ui-state.js`
  (keep the test in `web/ui-state.test.js` in sync). Release notes go in
  `RELEASE-x.y.z.md` and `RELEASE_NOTES.md`.
- **Deployment.** No image push, registry or server access, and no deployment
  without explicit instruction.
- Add dependencies only after asking; prefer the standard library.
- Follow the existing style: short functions, log errors server-side, never
  expose internal error details to the browser.
