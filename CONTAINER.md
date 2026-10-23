# Podman release 1.8.2

Version 1.8.2 (2026-10-23) is a rendering patch that converts the remaining
Unicode math to LaTeX; the image was built and pushed to Zot, with no server
deployment. See [RELEASE-1.8.2.md](RELEASE-1.8.2.md). The `Containerfile` builds
for `linux/amd64`. The SQLite driver `modernc.org/sqlite` is pure Go, so the
build sets `CGO_ENABLED=0`. The runtime image is `debian:bookworm-slim` with
`tzdata`, because the application loads `Europe/Zurich` at runtime.

The app runs as UID/GID `10001`. Only `/app/data` is writable and must be
persisted in a named Podman volume. `content/` is copied into the image as
static content, including `content/pdfs/` when it exists. The 13 PDFs are not in
Git; place them in `content/pdfs/` before building, otherwise the image is built
without them and the app shows missing-source notices. The published `1.8.0`
and `1.8.2` images were built without PDFs. The frontend is already embedded in
the binary via `go:embed`. The runtime image contains only `cmd/server` with
login enabled, not the login-free development entry point `cmd/preview`. There
is no auth-disable switch in the production server.

Check the PDFs before building:

```sh
ls content/pdfs/*.pdf | wc -l    # expected: 13
```

PowerShell:

```powershell
$appVersion = node -p "require('./web/ui-state.js').APP_VERSION"
podman build --platform linux/amd64 --build-arg "APP_VERSION=$appVersion" -t "localhost/mldl-exam-trainer:$appVersion" -f Containerfile .
podman volume create mldl-exam-data
podman run --rm --name mldl-exam-trainer-182 -p 127.0.0.1:18080:8080 -v mldl-exam-data:/app/data localhost/mldl-exam-trainer:1.8.2
```

Linux shell (bash/zsh):

```sh
appVersion=$(node -p "require('./web/ui-state.js').APP_VERSION")
podman build --platform linux/amd64 --build-arg "APP_VERSION=$appVersion" -t "localhost/mldl-exam-trainer:$appVersion" -f Containerfile .
podman volume create mldl-exam-data
podman run --rm --name mldl-exam-trainer-182 -p 127.0.0.1:18080:8080 -v mldl-exam-data:/app/data localhost/mldl-exam-trainer:1.8.2
```

If the volume does not contain a user yet, pass your own credentials through
the process environment for the one-time bootstrap. They are not written into
the image and are no longer required after successful initialization:

```powershell
$env:ADMIN_USERNAME = Read-Host 'Initial admin username'
$bootstrapPassword = Read-Host 'Initial password (at least 15 characters)' -AsSecureString
$env:ADMIN_PASSWORD = ([pscredential]::new($env:ADMIN_USERNAME, $bootstrapPassword)).GetNetworkCredential().Password
try {
    podman run --rm --name mldl-exam-trainer-182 -p 127.0.0.1:18080:8080 -v mldl-exam-data:/app/data -e ADMIN_USERNAME -e ADMIN_PASSWORD localhost/mldl-exam-trainer:1.8.2
} finally {
    Remove-Item Env:ADMIN_PASSWORD, Env:ADMIN_USERNAME -ErrorAction SilentlyContinue
    $bootstrapPassword.Dispose()
}
```

Linux shell (the password is read without echo and is not stored in the shell
history):

```sh
read -r -p 'Initial admin username: ' ADMIN_USERNAME
read -r -s -p 'Initial password (at least 15 characters): ' ADMIN_PASSWORD; echo
export ADMIN_USERNAME ADMIN_PASSWORD
podman run --rm --name mldl-exam-trainer-182 -p 127.0.0.1:18080:8080 -v mldl-exam-data:/app/data -e ADMIN_USERNAME -e ADMIN_PASSWORD localhost/mldl-exam-trainer:1.8.2
unset ADMIN_USERNAME ADMIN_PASSWORD
```

The test address is `http://127.0.0.1:18080`; the health endpoint is
`/healthz`. Allow a few seconds for startup before the first request. For the
read-only test:

```sh
podman run --rm --name mldl-exam-trainer-182-ro --read-only --cap-drop=ALL --security-opt=no-new-privileges -p 127.0.0.1:18081:8080 -v mldl-exam-data:/app/data localhost/mldl-exam-trainer:1.8.2
```

Use a fresh test volume for release smoke tests, not existing learning
progress, and remove it afterwards (`podman volume rm <name>`). The footer and
OCI label take version 1.8.2 from `web/ui-state.js`; Node is only needed for the
build command above, not at app runtime. Check the label with:

```sh
podman image inspect localhost/mldl-exam-trainer:1.8.2 --format '{{index .Labels "org.opencontainers.image.version"}}'
```

SQLite stays at schema 7, no migration. Old image tags are kept.

For the existing server installation, leave the configuration unchanged:
PVC `mldl-data`, mount `/data`, `APP_DB_PATH=/data/app.db` and proxy CIDRs.
Do not create a new empty volume and do not switch to the generic default path.

Images reach Zot only through `podman push` from the build machine; there is
no CI or other tool that publishes them. The approved Zot push is below. First confirm that the tag does not exist yet,
so no published tag is overwritten:

```sh
podman search --list-tags registry.marbui.ch/learn/mldl-exam-trainer
podman login registry.marbui.ch
podman tag localhost/mldl-exam-trainer:1.8.2 registry.marbui.ch/learn/mldl-exam-trainer:1.8.2
podman push registry.marbui.ch/learn/mldl-exam-trainer:1.8.2
```
