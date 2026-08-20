<div align="center">

# Datpaq MCP Server

**Hosted [Model Context Protocol](https://modelcontextprotocol.io) server for the [Datpaq API](https://datpaq.com).**

[![Go Reference](https://pkg.go.dev/badge/github.com/datpaq/mcp.svg)](https://pkg.go.dev/github.com/datpaq/mcp)
[![Go Report Card](https://goreportcard.com/badge/github.com/datpaq/mcp)](https://goreportcard.com/report/github.com/datpaq/mcp)
[![License](https://img.shields.io/badge/license-Apache_2.0-6b21a8.svg)](LICENSE)

[Connect](#connect) · [Configure your client](#configure-your-client) · [Self-host](#self-host) · [Dashboard ↗](https://datpaq.com)

</div>

---

Hosted at **<https://mcp.datpaq.com/>** — also installable locally and self-hostable.

`datpaq-mcp-http` exposes the Datpaq API as MCP tools over streamable HTTP, so any MCP-compatible client (Claude Desktop, Cursor, Codex, Cline, Continue, agent frameworks) can call Datpaq endpoints with a single connection.

For the **local CLI** (and the stdio MCP that ships with it for Claude Desktop), see [`github.com/datpaq/cli`](https://github.com/datpaq/cli).

## Connect

| | |
| --- | --- |
| **URL** | `https://mcp.datpaq.com/` |
| **Transport** | streamable HTTP |
| **Protocol** | `2026-07-28` and `2025-11-25` (and earlier) — see [Protocol versions](#protocol-versions) |
| **Auth** | `Authorization: Bearer YOUR_DATPAQ_API_KEY` (per request) |
| **Tools** | Hosted tools across 35 active APIs |

Verify with `curl` — modern (`2026-07-28`):

```bash
curl -s -X POST https://mcp.datpaq.com/ -H "Authorization: Bearer $YOUR_DATPAQ_API_KEY" -H "Content-Type: application/json" -H "Accept: application/json, text/event-stream" -H "MCP-Protocol-Version: 2026-07-28" -H "Mcp-Method: server/discover" -d '{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientInfo":{"name":"curl","version":"0"},"io.modelcontextprotocol/clientCapabilities":{}}}}'
```

Or legacy (`initialize` handshake), which keeps working unchanged:

```bash
curl -s -X POST https://mcp.datpaq.com/ -H "Authorization: Bearer $YOUR_DATPAQ_API_KEY" -H "Content-Type: application/json" -H "Accept: application/json, text/event-stream" -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"curl","version":"0"}}}'
```

## Protocol versions

The server is **dual-era**: it speaks both the stateless `2026-07-28` revision and the
handshake-based revisions (`2025-11-25` and earlier) on the same endpoint, and picks per
request. You do not need to configure anything — point your client at the URL and it will
use whichever it supports.

| Era | Versions | How it's served |
| --- | --- | --- |
| Modern | `2026-07-28` | Stateless. Every request carries its version, client info, and capabilities in `_meta`, mirrored into the `MCP-Protocol-Version` / `Mcp-Method` / `Mcp-Name` headers. Implements `server/discover`, `tools/list`, `tools/call`. |
| Legacy | `2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05` | The `initialize` handshake, exactly as before. Sessions, `GET` SSE streams, and `Mcp-Session-Id` all still work. |

Notes for the modern era:

- `tools/list` and `server/discover` return `ttlMs` and `cacheScope: "public"` caching
  hints. The tool catalog is identical for every tenant and only changes on redeploy, so
  clients can cache it and skip re-fetching every tool definition on each reconnect.
- `Mcp-Method` and `Mcp-Name` are validated against the request body. A header that
  **disagrees** with the body is rejected with `-32020 HeaderMismatch`; a **missing**
  header is served anyway, so clients that have not implemented header mirroring yet still
  work.
- An unsupported version gets `400` + `-32022` listing the versions we do support, so a
  client can retry rather than fail.
- Features the spec deprecated in `2026-07-28` — roots, sampling, logging, DCR, and the
  2024-11-05 HTTP+SSE transport — were never used by this server, so nothing changes for
  you there.

**Auth model:** every request must present a Datpaq API key as `Authorization: Bearer <key>` — the server holds no credentials of its own, so a single instance can serve many tenants, each billed against their own Datpaq account.

## Configure your client

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "datpaq": {
      "url": "https://mcp.datpaq.com/",
      "headers": {
        "Authorization": "Bearer YOUR_DATPAQ_API_KEY"
      }
    }
  }
}
```

**Cursor** (`~/.cursor/mcp.json` or project `.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "datpaq": {
      "url": "https://mcp.datpaq.com/",
      "headers": {
        "Authorization": "Bearer YOUR_DATPAQ_API_KEY"
      }
    }
  }
}
```

**Codex CLI:**

```bash
export DATPAQ_API_KEY="YOUR_DATPAQ_API_KEY"
codex mcp add datpaq --url https://mcp.datpaq.com/ --bearer-token-env-var DATPAQ_API_KEY
```

**Codex TOML:**

```toml
[mcp_servers.datpaq]
url = "https://mcp.datpaq.com/"
bearer_token_env_var = "YOUR_DATPAQ_API_KEY"
```

For full step-by-step instructions per client (including Cline and Continue), see the [Datpaq MCP docs](https://datpaq.com/docs/mcp).

## Self-host

Install via Go:

```bash
go install github.com/datpaq/mcp/cmd/datpaq-mcp-http@latest
```

Requires Go 1.26.3+.

Or pull the Docker image:

```bash
docker build -t datpaq-mcp-http -f cmd/datpaq-mcp-http/Dockerfile .
docker run -p 8080:8080 datpaq-mcp-http
```

Run it:

```bash
datpaq-mcp-http --addr :8080
```

Then point an MCP client at `http://localhost:8080/`.

Ships with a [Fly.io](https://fly.io) config for one-command deploys:

```bash
fly launch --config fly.toml
fly deploy
```

No secrets to configure — clients authenticate per request.

## What's exposed

The server registers one MCP tool per **active** Datpaq API endpoint. The active set is curated in [`internal/cli/active-apis.json`](internal/cli/active-apis.json) — add a slug there when a new API ships on [datpaq.com](https://datpaq.com), redeploy, and the tool appears for every connected client.

Inactive APIs are **scrubbed entirely** from the MCP surface (unlike the CLI, which still lets you invoke them directly). This keeps the hosted tool list focused on production-ready endpoints.

Tools that depend on local state or shell-out to the user's machine (auth flows, config writers, etc.) are deliberately not registered — see [`internal/mcp/public_tools.go`](internal/mcp/public_tools.go).

## Configuration

| Variable | Purpose |
| --- | --- |
| `DATPAQ_BASE_URL` | Override the API base URL (default: `https://datpaq.com/api/v1`) — handy for pointing at staging |
| `PORT` | HTTP listen port (default: `8080`, also configurable via `--addr`) |

The Datpaq API key is **not** a server-side variable. Each request must include it as `Authorization: Bearer <key>`. See the Quickstart auth model.

## Managing active APIs

The active set is curated in [`internal/cli/active-apis.json`](internal/cli/active-apis.json). The **CLI repo is the canonical place to regenerate** that manifest from the admin dashboard export (or, in Phase 2, `GET /api/v1/catalog/active`). This repo receives updates via sync — it does not ship its own fetch tooling.

```json
{
  "active": [
    "convert-time",
    "ip-geolocation",
    "..."
  ]
}
```

**Canonical workflow** (sibling checkout at `../CLI`):

```bash
# In the CLI repo
cd ../CLI
make fetch-active-apis CATALOG_INPUT=./export.json   # or CATALOG_URL=... after Phase 2
make sync-active-apis                                 # CLI → MCP

# Then here
make build && make test
# commit internal/cli/active-apis.json + code changes, deploy
```

If you edited `active-apis.json` in this repo by mistake, push it back to CLI with `./scripts/sync-active-apis.sh` (MCP → CLI only). Prefer the CLI-first flow above for normal updates.

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
