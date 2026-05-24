<div align="center">

# Datpaq MCP Server

**Hosted [Model Context Protocol](https://modelcontextprotocol.io) server for the [Datpaq API](https://datpaq.com).**

[![Go Reference](https://pkg.go.dev/badge/github.com/datpaq/mcp.svg)](https://pkg.go.dev/github.com/datpaq/mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/datpaq/mcp)](https://goreportcard.com/report/github.com/datpaq/mcp)
[![License](https://img.shields.io/badge/license-Apache_2.0-6b21a8.svg)](LICENSE)

[Install](#install) · [Deploy](#deploy) · [Configuration](#configuration) · [Dashboard ↗](https://datpaq.com)

</div>

---

`datpaq-mcp-http` exposes the Datpaq API as MCP tools over HTTP, so any MCP-compatible client (Claude, IDE plugins, agent frameworks) can call Datpaq endpoints with a single connection.

For the **local CLI** (and the stdio MCP that ships with it for Claude Desktop), see [`github.com/datpaq/cli`](https://github.com/datpaq/cli).

## Install

```bash
go install github.com/datpaq/mcp/cmd/datpaq-mcp-http@latest
```

Requires Go 1.22+.

Or pull the Docker image:

```bash
docker build -t datpaq-mcp-http -f cmd/datpaq-mcp-http/Dockerfile .
docker run -p 8080:8080 -e DATPAQ_API_KEY=... datpaq-mcp-http
```

## Quickstart

```bash
DATPAQ_API_KEY=your_key datpaq-mcp-http --addr :8080
```

Then point an MCP client at `http://localhost:8080/mcp`.

## Deploy

Ships with a [Fly.io](https://fly.io) config:

```bash
fly launch --config fly.toml
fly secrets set DATPAQ_API_KEY=your_key
fly deploy
```

## What's exposed

The server registers one MCP tool per **active** Datpaq API endpoint. The active set is curated in [`internal/cli/active-apis.json`](internal/cli/active-apis.json) — add a slug there when a new API ships on [datpaq.com](https://datpaq.com), redeploy, and the tool appears for every connected client.

Inactive APIs are **scrubbed entirely** from the MCP surface (unlike the CLI, which still lets you invoke them directly). This keeps the hosted tool list focused on production-ready endpoints.

Tools that depend on local state or shell-out to the user's machine (auth flows, config writers, etc.) are deliberately not registered — see [`internal/mcp/public_tools.go`](internal/mcp/public_tools.go).

## Configuration

| Variable | Purpose |
| --- | --- |
| `DATPAQ_API_KEY` | API credential used for all downstream calls — required |
| `DATPAQ_BASE_URL` | Override the API base URL (default: `https://datpaq.com/api/v1`) — handy for pointing at staging |
| `PORT` | HTTP listen port (default: `8080`, also configurable via `--addr`) |

## Managing active APIs

The single source of truth for which endpoints are exposed is [`internal/cli/active-apis.json`](internal/cli/active-apis.json):

```json
{
  "active": [
    "convert-time",
    "ip-geolocation",
    "..."
  ]
}
```

Edit the file, rebuild or redeploy. The same file lives in [`github.com/datpaq/cli`](https://github.com/datpaq/cli) — keep both in sync when adding a new API.

## Development

```bash
git clone https://github.com/datpaq/mcp && cd mcp
make build
./bin/datpaq-mcp-http --addr :8080
```

Run tests:

```bash
make test
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
