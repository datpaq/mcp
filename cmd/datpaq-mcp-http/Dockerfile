# syntax=docker/dockerfile:1.7
#
# Hosted Datpaq MCP server. Build context is the repo root:
#
#   docker build -f cmd/datpaq-mcp-http/Dockerfile -t datpaq/mcp-http .
#
# Produces a ~25MB statically-linked binary on top of distroless
# (which bundles CA certs and runs as nonroot). Targets any
# container-friendly host: Fly.io, Cloudflare Containers, Railway,
# Render, Cloud Run, etc.

# ── Stage 1: build ────────────────────────────────────────────────
FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS builder

WORKDIR /src

# Pre-download modules with a separate layer so source edits don't
# re-fetch the whole graph on every rebuild.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

# Static, stripped, reproducible build. CGO_ENABLED=0 → no glibc dep
# (so we can land on scratch/distroless). -trimpath drops local
# paths from the binary; -ldflags strips symbol + DWARF tables.
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags='-s -w' \
      -o /out/datpaq-mcp-http \
      ./cmd/datpaq-mcp-http

# ── Stage 2: runtime ──────────────────────────────────────────────
# distroless/static: ca-certificates, /etc/passwd with nonroot user,
# no shell, no package manager. About 2MB before the app binary
# lands on top.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/datpaq-mcp-http /usr/local/bin/datpaq-mcp-http

# Most hosts set $PORT; the binary defaults to 8080 when unset.
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/datpaq-mcp-http"]
