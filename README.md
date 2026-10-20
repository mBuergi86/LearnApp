# ML/DL Exam Trainer

Web app for exam preparation in Machine Learning and Deep Learning & Data Science. The current version is 1.8.0 (2026-10-20): practice focuses on exam-style questions, formulas in questions are typeset with LaTeX, and the text is larger. Version 1.7.1 (2026-10-20) fixed login behind the TLS proxy. Version 1.7.0 (2026-10-13) introduced real LaTeX typesetting rendered as native MathML, simple worked examples in 49 formula cards, and per-topic quick checks. The navigation from 1.6.0 is unchanged.

The user interface and the learning content are in German.

## Requirements

- Go 1.26.9 or newer (patched toolchain)
- a modern browser with a native PDF viewer and `<dialog>` support
- no npm, Node.js or frontend build dependencies
- a persistent container volume mounted at `/app/data` and a secret for the one-time admin bootstrap

## Build

From the project directory:

```powershell
go build -o mldl-exam-trainer.exe ./cmd/server
```

The vanilla JS/CSS interface in `web/` is embedded in the binary. Course JSON and PDFs remain external files.

## Start

On the very first start, admin credentials must be provided. They are stored in
SQLite as an Argon2id hash and are not needed afterwards:

```powershell
$env:ADMIN_USERNAME = 'your-admin-name'
$env:ADMIN_PASSWORD = 'a-long-one-time-password'
.\mldl-exam-trainer.exe
```

The server listens on `0.0.0.0:8080`. In deployment it must only be reachable
through the TLS-terminating Traefik ingress.

The authentication cookies are deliberately `Secure`, so the deployment must use
HTTPS. Modern browsers treat the local loopback address as a secure context; the
documented UI smoke test uses `http://127.0.0.1` only and does not carry over to
remote plain-text connections.

## Local preview without an image build

For reviewing content **without login**:

```powershell
go run ./cmd/preview
```

Then open `http://127.0.0.1:18087`. This separate development entry point binds
to loopback only, always uses a fresh temporary database, and ignores
`APP_DB_PATH` and all admin/proxy environment variables. The existing database
is left untouched. No users, sessions or admin rights are created; logout and
user management are not available in this preview. Foreign hosts, remote
clients and cross-site requests are rejected. Stop with `Ctrl+C`. The temporary
preview data is not deleted automatically.

The container builds `cmd/server` only, never `cmd/preview`. There is **no**
auth-disable switch in the production server. The regular start with login
remains `go run ./cmd/server`; on first start it still requires bootstrap
credentials. Preview mode is not part of the runtime image.

### Optional: preview with real login

Start in a separate PowerShell in the project directory. The preview binds to
loopback only and uses a new temporary SQLite file. Existing learning progress
and container volumes are not touched. Enter a preview password of at least 15
characters at the hidden prompt; do not put it in the command or a chat:

```powershell
$previewDir = Join-Path ([IO.Path]::GetTempPath()) ('mldl-preview-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $previewDir | Out-Null
$env:APP_DB_PATH = Join-Path $previewDir 'exam.db'
$env:EXAM_TRAINER_ADDR = '127.0.0.1:18087'
$env:APP_TRUSTED_PROXY_CIDRS = ''
$env:ADMIN_USERNAME = 'preview'
$previewPassword = Read-Host 'Preview password (at least 15 characters)' -AsSecureString
$env:ADMIN_PASSWORD = ([pscredential]::new($env:ADMIN_USERNAME, $previewPassword)).GetNetworkCredential().Password
try {
    go run ./cmd/server
} finally {
    Remove-Item Env:ADMIN_PASSWORD, Env:ADMIN_USERNAME -ErrorAction SilentlyContinue
    $previewPassword.Dispose()
}
```

Open `http://127.0.0.1:18087` and log in as `preview` with the password you just
chose. The formula cards "Künstliches Neuron", "Logistische Regression" and
"Gradientenabstieg" are good for comparison. Stop with `Ctrl+C`. The temporary
database is kept so you can resume later; reuse the same `APP_DB_PATH`. The
preview is a separate development entry point, not the container entrypoint.

## Local Podman container

A local Podman containerization targeting `linux/amd64` is provided in
`Containerfile` and [.containerignore](.containerignore). The full local build,
test and export guide is in [CONTAINER.md](CONTAINER.md). It covers neither
server access nor a Kubernetes or k3s rollout.

## Data directories

```text
content/catalog.json       exam areas, modules and PDF registry
content/topics/            58 learning pages with source metadata
content/questions/         256 active, objectively gradable questions
content/inventory/         phase 8A slide inventory
content/audit/             final audits
content/pdfs/              13 external original PDFs (not in Git)
/app/data/exam.db          learning progress, users and sessions (SQLite, schema 7)
/app/data/exam.db-wal      possible SQLite WAL file during operation
/app/data/exam.db-shm      possible SQLite SHM file during operation
/app/data/backups/         automatic safety copies taken before restores
```

The database directory is created automatically on first start. Content and PDFs are not included in learning-progress backups and are not modified by a restore.

The course PDFs are copyrighted lecture slides and are excluded from the repository via `.gitignore`. Place them in `content/pdfs/` locally.

## Backup

Under `Einstellungen → Daten` (Settings → Data), `Backup herunterladen` (Download backup) downloads a file such as `mldl-exam-trainer-backup-2026-09-06.json`. The format is versioned:

```json
{
  "format": "mldl-exam-trainer-backup",
  "version": 1,
  "created_at": "...",
  "database_schema": 7,
  "data": {}
}
```

It contains all personal learning-progress data from schema 7: theme setting, review history with answers, question progress, topic progress/mastery, scheduler state and exam attempts still in progress. Users, password hashes and sessions are deliberately excluded from learning-progress backups. Today sessions and completed exam answers have no separate aggregates; the actually stored answers are fully contained in `reviews`.

## Restore

1. Open `Einstellungen → Daten → Backup wiederherstellen` (Settings → Data → Restore backup).
2. Select a JSON backup file.
3. Confirm the restore, which is explicitly labeled as replacing existing data.

The server accepts at most 8 MiB and validates JSON structure, format, version, schema, timestamps and all content references. The restore replaces only personal state, inside a single SQLite transaction. Any error causes a full rollback. Before any change, the server also writes a local JSON safety copy to `data/backups/`.

The safety copies contain the same personal state as a normal export. They are not deleted automatically. For a manual emergency rollback, the matching file can be restored through the same restore UI.

## PDFs and content

The 13 registered course PDFs live in `content/pdfs/` and are not embedded in the binary. The `/pdf/<file>` endpoint serves only registered regular files from this directory, blocks traversal and symlinks, and supports range requests. Spaces and parentheses in file names are URL-encoded; `#page=N` opens the browser PDF viewer at the referenced start page.

All 58 learning pages reference verified pages of their original slides. Missing physical PDFs produce a warning at startup and a clear notice on the learning page; structurally invalid sources, by contrast, abort content loading. Source links are hidden while a quiz question is unanswered or an exam is in progress.

Current content:

- 58 learning pages: 28 Machine Learning and 30 Deep Learning & Data Science
- 45 CORE and 13 SUPPORT topics
- 256 active questions: 122 Machine Learning and 134 Deep Learning & Data Science
- single choice, multiple select, true/false, ordering, matching and numeric multiple choice
- 13 registered PDFs with a total of 621 original pages

## Authentication and database

SQLite uses schema 7, WAL, foreign keys and a busy timeout. Migration 7 adds:

```text
users(id, username, password_hash, role, created_at, updated_at, disabled)
sessions(id, session_hash, csrf_hash, user_id, created_at, last_seen_at, expires_at)
```

Passwords use Argon2id (64 MiB, 3 iterations, parallelism 2). Browsers only
receive a random 256-bit session cookie; SQLite stores only its SHA-256 hash.
Sessions expire after 30 minutes of inactivity or after 12 hours at the latest.
Changing the password deletes all of that user's sessions. All write requests
require the CSRF header `X-CSRF-Token`.

`ADMIN_USERNAME` and `ADMIN_PASSWORD` are only needed while the `users` table is
empty. They belong in `env` as a Kubernetes Secret, never in Git, the image or
the Dockerfile. `POST /api/auth/password` changes your own password and
invalidates all of that user's sessions. The admin endpoint
`GET /api/admin/users` checks the role entirely server-side; the `users` table
can later be extended with TOTP MFA metadata.

Behind a TLS proxy, `APP_TRUSTED_PROXY_CIDRS` must contain the actual trusted
backend peer networks (comma-separated CIDRs). Without it, forwarded IP and
protocol headers are ignored. The proxy must reliably overwrite incoming
forwarded headers or append a correct chain. Details and test results are in
the [middleware audit](content/audit/middleware-security-review.md).

All 58 topics (28 ML and 30 DL/DS) include detailed explanations, worked
examples, common confusions and two local SVG illustrations each. Formulas
explain their symbols. The 122 ML questions were revised; 29 ML and 34 DL
calculation questions are verified independently. The pools remain 122/104 for
ML and 134/110 for DL/DS (all/core). Details and limitations are in the
[content review](content/audit/dl-ds-quality-review.md).

### Release build 1.8.0

```powershell
$appVersion = node -p "require('./web/ui-state.js').APP_VERSION"
podman build --platform linux/amd64 --build-arg "APP_VERSION=$appVersion" -t "localhost/mldl-exam-trainer:$appVersion" -f Containerfile .
```

The image tag is explicitly `:1.8.0`; the shared login/app footer shows
`ML/DL Exam Trainer · v1.8.0`. The version source is `web/ui-state.js`; the
build also applies it as an OCI image label. The approved registry tag is
`<registry>/learn/mldl-exam-trainer:1.8.0`. Not built or deployed yet.
Local verification uses dedicated test volumes, not existing learning data.

### Release notes 1.8.0

- 53 calculation questions disabled, 10 typical ones kept with LaTeX.
- 67 new exam-style questions in `content/questions/phase9-exam-style.json`.
- Inline math `$...$` rendered as MathML; larger text.
- Multi-Hot encoding corrected. Details: [RELEASE-1.8.0.md](RELEASE-1.8.0.md).

### Release notes 1.7.1

- Login on the production instance failed with `403 Fremde Herkunft abgelehnt.`
  (foreign origin rejected) because, behind Traefik, the same-origin check
  compared the scheme of the `Origin` (https) with the internally detected
  scheme (http).
- When the browser sends `Sec-Fetch-Site: same-origin` and the origin host
  matches `Host`, the scheme comparison is now skipped. All other requests are
  checked as strictly as before; foreign origins are still rejected.
- No changes to content, authentication, CSRF or proxy header trust. SQLite
  stays at schema 7, no migration. Details: [RELEASE-1.7.1.md](RELEASE-1.7.1.md).

### Release notes 1.7.0

- Quick checks at the bottom of all 58 learning pages: true/false questions for
  the current topic only, with feedback, explained results and retry. They use
  60 existing questions plus 8 additional comprehension questions for four
  topics that previously had no true/false coverage. Nothing is stored in
  reviews, mastery, scheduler or exam statistics; no DB migration.
- "Kannst du es?" (Can you do it?) and module training only appear on the last
  topic of a module sequence, not on the earlier topics. The rest of the
  practice selection remains available.
- 17 quiz prompts now ask about subject knowledge instead of the lecture or
  slides. IDs, answers and the 256 active exam questions are unchanged.
- LaTeX math syntax instead of bold monospace text; fractions, sums, indices,
  roots and matrices are typeset.
- 49 cards with givens, numbered calculation steps and a result with its
  meaning, also in Today learning views. Focus: substitution, basic arithmetic,
  proportions, means, weight/bias/activation, thresholds and parameter counts.
- Derivatives or function values are given where necessary for simple
  examples; no long proofs or matrix inversions as example tasks.
- KaTeX 0.19.0 (MIT) is bundled locally, without CDN or external fonts. Output
  is native MathML only: no `unsafe-inline` and no relaxation of the existing
  CSP. Source and checksums: `web/vendor/katex/provenance.json`.
- Rendering uses normal math fonts instead of a bold code font. Wide formulas
  scroll horizontally within their card.
- Exam grading, auth, theme, scheduler, mastery and the PDF system are unchanged.
- Full subject coverage of the PDFs is not yet proven. The existing inventory
  summarizes content and also lists REFERENCE/DROP areas; source links alone do
  not prove a 1:1 transfer of content.
- The theme setting still uses the existing `settings` entry `theme`. Existing
  values `light` and `dark` are unchanged; `system` is additionally allowed. The
  default for new or unconfigured databases deliberately stays `light`, so no
  existing behavior changes unasked. No new migration is required; the content
  JSON only gains additive formula/example fields. SQLite stays at schema 7.

Technical background: [KaTeX options](https://katex.org/docs/options) and
[security options](https://katex.org/docs/security). KaTeX is a limited math
renderer, not a full LaTeX document compiler. Typst SVGs would be an
alternative; its [HTML export](https://typst.app/docs/reference/html/) is
currently experimental. No Typst build pipeline is added.

### Release notes 1.6.0

- Topic position at top and bottom, for example `Thema 1 / 3` (Topic 1 / 3).
- At the actual end of the page: previous topic, next topic and `Training`.
- On the first topic, Back leads to the module overview; after the last topic,
  Next leads to the exam area.
- Order and count match the existing module overview, including supplementary
  topics; Next does not change mastery or status automatically.
- Native links, visible keyboard focus and mobile touch targets; no sticky
  navigation over learning content. Auth, themes, content and learning logic
  unchanged.

No new data migration was needed before phase 7. Restore and startup perform additional consistency checks for content and personal references.

## Tests and quality checks

```powershell
Get-ChildItem .\cmd,.\internal,.\web -Recurse -Filter '*.go' | ForEach-Object { gofmt -w $_.FullName }
go test ./...
go vet ./...
go build ./cmd/server
```

If Node.js is available locally, it can run the JavaScript tests and a syntax
check without installing any packages:

```powershell
node --test "web/*.test.js"
node --check web/app.js
```

The automated tests cover, among other things, content integrity, all six question types, shuffling, mastery/scheduler, single-submission exams, PDF encoding/range/traversal, source visibility, backup/restore round trip and rollback, API body limits, methods, same-origin behavior, and structural accessibility/responsive features.

## Operations and failure cases

- `/healthz` returns `{"status":"ok"}` when the database is reachable.
- Internal errors are logged server-side; browser responses never contain SQL errors, panics or stack traces.
- Write JSON endpoints require `Content-Type: application/json` and only accept limited request sizes.
- Security headers include CSP, `nosniff`, Referrer-Policy, frame protection and a restrictive Permissions-Policy. There is no open CORS. HSTS is only set when HTTPS is detected, either directly or via an explicitly trusted proxy.
- The server must be started from the project directory so that `content/`, `content/pdfs/` and `data/` resolve correctly.

## Known V1 limitations

- no TOTP MFA and no password reset channel; both require a deliberately chosen second factor or recovery process
- no PDF.js component, OCR, annotations, upload UI or PDF full-text search
- no AI grading or free-text answers
- browser PDF page anchors depend on the native viewer of the browser in use
- a final visual check on the actual PC and phone is still advisable, especially for native PDF viewers
- the 13 course PDFs define the learning scope; subject-matter clarifications were additionally checked against the official scikit-learn/PyTorch documentation and the original Transformer paper
