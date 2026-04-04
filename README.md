# ntfy-mcp

Self-hosted [ntfy](https://ntfy.sh) push notification server with an integrated MCP sidecar, designed for the [Beacon](../beacon) service mesh.

Claude (or any MCP client) can send push notifications to your phone, desktop, or any ntfy subscriber — just by calling a tool.

## Architecture

```
┌──────────────────────────────────┐
│  ntfy-mcp container  (mcp-net)  │
│                                  │
│  ntfy serve        → :8080       │  push notification server (Go)
│  mcp-sidecar       → :9099 tcp  │  MCP streamable HTTP endpoint
│  beacon announce   → :9099 udp  │  auto-discovery on mcp-net
└──────────────────────────────────┘
         │                    │
    ntfy clients         Beacon aggregator
  (phone, browser)      (http://localhost:9099/mcp)
```

Single container, two static Go binaries, no runtime dependencies.

## Quick Start

### Prerequisites

- Docker
- Beacon running on `mcp-net` (see [Beacon README](../beacon/README.md))

### 1. Create the shared network (if not already done)

```bash
docker network create mcp-net
```

### 2. Build and run

```bash
cd ntfy
docker compose up -d --build
```

### 3. Verify

Check the Beacon web UI at `http://localhost:9300` — the `ntfy` server should appear with its tools.

Or test directly:

```bash
# Send a test notification via ntfy
curl -d "Hello from ntfy-mcp!" http://localhost:8080/test

# Check MCP health
curl http://localhost:9099/mcp
```

### 4. Subscribe to notifications

Install the ntfy app on your phone or use the web UI at `http://localhost:8080` and subscribe to a topic.

## MCP Tools

Once discovered by Beacon, these tools are available as `ntfy__<tool_name>`:

| Tool | Description |
|------|-------------|
| `send_message` | Publish a notification to a topic |
| `send_photo` | Publish a notification with an image attachment |
| `list_messages` | Read cached messages from a topic |
| `server_info` | Health check and server version |

### send_message

```json
{
  "topic": "alerts",
  "message": "Build failed on main",
  "title": "CI Alert",
  "priority": 4,
  "tags": ["warning", "build"],
  "click": "https://github.com/org/repo/actions",
  "actions": [
    { "action": "view", "label": "Open CI", "url": "https://ci.example.com" }
  ]
}
```

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `topic` | string | yes | Target topic name |
| `message` | string | yes | Notification body |
| `title` | string | no | Notification title |
| `priority` | int | no | 1 (min) to 5 (max), default 3 |
| `tags` | string[] | no | Emoji tags (e.g. `["warning"]` → ⚠️) |
| `click` | string | no | URL to open on notification click |
| `actions` | object[] | no | Action buttons (view, http, broadcast) |

### send_photo

```json
{
  "topic": "photos",
  "url": "https://example.com/screenshot.png",
  "caption": "Latest dashboard screenshot"
}
```

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `topic` | string | yes | Target topic name |
| `url` | string | yes | URL of the image to attach |
| `caption` | string | no | Message body alongside the image |
| `title` | string | no | Notification title |
| `priority` | int | no | 1-5, default 3 |

### list_messages

```json
{
  "topic": "alerts",
  "since": "1h",
  "limit": 10
}
```

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `topic` | string | yes | Topic to read from |
| `since` | string | no | Duration like `30m`, `1h`, `1d` or message ID. Default `1h` |
| `limit` | int | no | Max messages to return. Default 50 |

### server_info

No parameters. Returns ntfy server version, uptime, and topic count.

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `NTFY_BASE_URL` | `http://localhost:8080` | Internal ntfy server URL |
| `MCP_PORT` | `9099` | MCP sidecar HTTP + UDP port |
| `NTFY_DEFAULT_TOPIC` | `general` | Default topic when none specified |
| `NTFY_AUTH_TOKEN` | *(empty)* | Bearer token for ntfy API (if auth enabled) |

### Docker Compose

```yaml
services:
  ntfy:
    build: .
    hostname: ntfy
    ports:
      - "8080:8080"   # ntfy web UI + API (optional, for direct access)
    expose:
      - "9099/tcp"     # MCP endpoint (Beacon discovers this)
      - "9099/udp"     # Beacon discovery
    environment:
      - NTFY_DEFAULT_TOPIC=general
    volumes:
      - ntfy-cache:/var/cache/ntfy   # optional persistence
    networks:
      - mcp-net

volumes:
  ntfy-cache:

networks:
  mcp-net:
    external: true
```

## Production Deployment

For public-facing deployments behind [nsl.sh](https://nsl.sh):

```
Internet → nsl.sh (HTTPS) → nginx-hash-lock (:8080) → ntfy (:8080)
                                    │
                              hash-based auth
```

- **ntfy** stays unauthenticated inside the Docker network
- **nginx-hash-lock** handles public access control via query parameter hash
- **Beacon network** is never exposed — MCP tools remain local-only
- **ntfy clients** (phone app) connect through the public nsl.sh URL with the auth hash

See [Nginx-hash-lock](../Nginx-hash-lock) for proxy configuration.

## Connecting Claude Code

Once Beacon is running, ntfy tools are automatically available through the Beacon aggregator:

```json
{
  "mcpServers": {
    "beacon": {
      "type": "streamableHttp",
      "url": "http://localhost:9099/mcp"
    }
  }
}
```

Or via CLI:

```bash
claude mcp add beacon --transport http http://localhost:9099/mcp
```

Tools appear as `ntfy__send_message`, `ntfy__send_photo`, etc.

## License

MIT
