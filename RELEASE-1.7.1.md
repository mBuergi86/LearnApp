# Release 1.7.1

As of: 2026-10-20. Bugfix release for login behind the TLS proxy.
Image built and pushed to Zot. No server access and no deployment.

## Image

- Tag: `<registry>/learn/mldl-exam-trainer:1.7.1`
- Local tag: `localhost/mldl-exam-trainer:1.7.1`
- Platform: linux/amd64, CGO disabled.
- Image ID: `eef2e74acaaa771268d2eecb1137bd1f1a932a540fe397174e683706ae745097`
- Digest: `sha256:780554ab9c663992e30ddf7679596a5856f693e363991ff8f04cfbc6248756bb`
- A registry pull confirmed identical image ID, platform and OCI version label 1.7.1.
- Tag `1.7.1` did not exist before, either locally or in Zot. Earlier tags
  are kept. No latest tag replaced.
- Version source: `web/ui-state.js`; footer and OCI label take 1.7.1 from it.

## Fixed bug

Login on the production instance failed with `403 Forbidden` and
`{"error":"Fremde Herkunft abgelehnt."}` (foreign origin rejected), even though
the credentials were correct.

- Cause: the same-origin check in `withSameOrigin`
  (`internal/http/server.go`) compared the scheme of the `Origin` with the
  scheme detected server-side. Behind Traefik the request arrives over HTTP;
  `https` is only detected when the peer is within `APP_TRUSTED_PROXY_CIDRS`.
  In production that was not the case, so the site's own HTTPS origin was
  treated as foreign.
- Scope: the 403 response was the application's JSON, not a block by the
  upstream Skyhigh gateway. The missing referer was not involved; the
  application does not check the referer.
- Fix: when the browser sets `Sec-Fetch-Site: same-origin` and the origin host
  matches `Host`, the scheme comparison is skipped. The browser has thereby
  already confirmed scheme, host and port. All other requests are still checked
  strictly.

## Unchanged

- Foreign origins and `Sec-Fetch-Site: cross-site` are still rejected.
- Forwarded headers (`X-Forwarded-Proto`, `X-Forwarded-For`) are still only
  accepted from configured trusted proxy CIDRs. `X-Forwarded-Host` is not
  evaluated.
- CSRF token for all write requests after login, Argon2id, sessions, cookie
  flags, rate limits and logging without credentials.
- Learning content, questions, exam pools, scheduler, mastery, PDFs, backup/restore.
- SQLite schema: 7. No new DB migration.

## Test results

| Check | Result |
|---|---|
| gofmt | PASS |
| go vet ./... | PASS |
| go test -count=1 ./... | PASS |
| New test `TestSameOriginLoginBehindUntrustedTLSProxy` | PASS: same origin 200, foreign origin 403, still strict without `Sec-Fetch-Site`, no HSTS via untrusted `X-Forwarded-Proto` |
| node --test web/*.test.js | PASS, 21 tests |
| go build ./... | PASS |
| Container smoke (read-only, cap-drop=ALL, no-new-privileges, throwaway volume) | PASS: login same origin 200, foreign origin 403, footer version 1.7.1 |
| go test -race, browser smoke scripts, govulncheck | NOT TESTED for 1.7.1 |

The smoke container `mldl-171-smoke` and its volume were removed after the test.

## Deployment notes

- Only change the image tag to `:1.7.1`. Be sure to keep the existing server
  configuration: PVC `mldl-data`, mount `/data`, `APP_DB_PATH=/data/app.db`,
  existing trusted proxy CIDRs.
- `APP_TRUSTED_PROXY_CIDRS` in production probably does not match the actual
  Traefik peer address: HSTS is missing, and the login rate limit sees all
  clients under the Traefik IP. Check the value against the real Traefik pod
  network.
- If an upstream proxy strips `Sec-Fetch-Site`, the fix does not apply; the
  correct proxy CIDR configuration is then required.

## Changed files since 1.7.0

- `internal/http/server.go`, `internal/http/proxy_test.go`
- `web/ui-state.js`, `web/ui-state.test.js`, `web/index.html`, `web/embed_test.go`
- `tools/browser-smoke.mjs`, `tools/dl-browser-smoke.mjs`
- `RELEASE-1.7.1.md`

There was no Git repository in the workspace at the time; this list is the
documented change list for this working state.
