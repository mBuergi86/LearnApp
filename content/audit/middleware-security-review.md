# Go middleware security review

As of: 2026-10-12. The app uses `net/http`; Echo is neither in `go.mod` nor in
the routing. No framework switch was made. Echo can provide suitable
middleware, but the framework alone is no security guarantee. In particular,
proxy trust must match the concrete network topology; see the
[official Echo documentation](https://echo.labstack.com/guide/ip-address/).

## Existing protections

Server-side SQLite sessions with a SHA-256 of the random browser token,
Argon2id (64 MiB, 3 iterations, parallelism 2), HttpOnly/Secure/SameSite=Lax,
idle and absolute timeouts, DB-based role checks, CSRF token and same-origin
check, parameterized SQL queries, limited JSON bodies, restrictive CSP and
further security headers. APIs and course PDFs require a valid session. There
are no browser JWTs and no auth tokens in web storage.

## Weaknesses found and fixed

1. `X-Forwarded-For` was trusted without checking the network peer.
   By default only `RemoteAddr` is now used. Configured proxy CIDRs allow a
   forwarded chain checked from the right up to the first untrusted hop;
   arbitrary header prefixes are not accepted.
2. The dummy Argon2 hash for unknown users contained URL Base64 instead of
   standard Base64. This caused the decoder to abort early. The encoding is
   fixed and tested; unknown users now also get a full Argon2 check. Disabled
   users also go through the password check.
3. The login limiter only limited short-term delays, not the whole one-minute
   window. After five failures the current window stays locked, per IP and
   username. Entries are removed after inactivity and the table is capped at
   10,000 entries.
4. Concurrent Argon2 login checks are limited to two. Additional requests get a
   short 429 response before any additional hash memory is allocated. This
   does not replace rate limits at the upstream proxy.
5. HSTS and the external HTTPS origin are only recognized with real TLS or an
   explicitly trusted proxy. The origin check now also compares the protocol.
6. Logout no longer reports success when DB invalidation fails. The client can
   show the error message and stays consistent.
7. Expired sessions are removed at startup and every minute. The previously
   existing cleanup was not wired to a periodic call.
8. Header, read, write and idle timeouts and a maximum header size are set
   explicitly.
9. Go 1.26.4 reported reachable known standard library vulnerabilities.
   The container and the preferred local toolchain now use Go 1.26.9.
   Existing x/crypto and x/sys dependencies were updated to v0.56.0/v0.49.0,
   without introducing a new runtime dependency.

## Scan and limitations

`govulncheck v1.9.0` with Go 1.26.9: no reachable known vulnerabilities and no
further findings in imported packages. At module level, the notice about the
unmaintained `x/crypto/openpgp` remains; this package is not imported or linked
into the app. This was additionally verified with
`GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go list -deps ./cmd/server`; the source
scan for exactly this Linux build configuration also reports no reachable
findings. The separate scan of the binary stripped with `-s -w`, by contrast,
reports the OpenPGP notice generically with package wildcards. This is not a
fully clean binary scan, and that is stated openly: according to the
[govulncheck documentation](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck),
missing symbol information can lead to module-level findings.
The scanner does not replace a penetration test, an OS image check or an
assessment of the Traefik, TLS, Kubernetes or registry configuration actually
in use.

The app is still designed for a private, shared learning state, not as a
tenant-isolated multi-user platform. A compromised browser or a stolen active
session can pose risks despite HttpOnly. Volume protection, updates, backups
and upstream network boundaries remain necessary.

## Required proxy configuration

`APP_TRUSTED_PROXY_CIDRS` contains, comma-separated, only the actual backend
peer CIDRs of your own proxy. IPv4/IPv6 are supported; `0.0.0.0/0` and `::/0`
are rejected. No private networks are trusted automatically. The proxy must
sanitize incoming forwarded headers.

Without proxy configuration, local direct HTTP calls keep working. An external
HTTPS origin behind a TLS proxy connected internally over HTTP, however, is
rejected until the correct proxy peer is configured. This configuration must be
aligned with the real topology before a later deployment; a blanket Kubernetes
CIDR was not guessed.

**Update 1.7.1:** same-origin requests confirmed by the browser
(`Sec-Fetch-Site: same-origin` with a matching host) are now accepted even
without trusted proxy CIDRs. HSTS and correct client IPs for rate limiting
still require the correct proxy configuration. See
[RELEASE-1.7.1.md](../../RELEASE-1.7.1.md).
