# syntax=docker/dockerfile:1
# modernc.org/sqlite is a pure-Go driver, so the Linux/amd64 binary needs no
# C compiler or CGO runtime libraries.
FROM golang:1.26.9-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/mldl-exam-trainer ./cmd/server

FROM debian:bookworm-slim

ARG APP_VERSION
LABEL org.opencontainers.image.title="ML/DL Exam Trainer" \
      org.opencontainers.image.version="${APP_VERSION}"

# Europe/Zurich is loaded at runtime. tzdata is therefore required at runtime.
RUN apt-get update \
    && apt-get install --no-install-recommends -y tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --gid 10001 app \
    && useradd --uid 10001 --gid 10001 --no-create-home --shell /usr/sbin/nologin app \
    && install -d --owner=10001 --group=10001 --mode=0750 /app/data

WORKDIR /app

COPY --from=build /out/mldl-exam-trainer /app/mldl-exam-trainer
COPY --from=build /src/content /app/content

USER 10001:10001
EXPOSE 8080

ENTRYPOINT ["/app/mldl-exam-trainer"]
