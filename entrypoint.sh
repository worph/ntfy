#!/bin/sh
# Start ntfy server in background
ntfy serve --behind-proxy --cache-file /var/cache/ntfy/cache.db &
NTFY_PID=$!

# Forward signals to ntfy when sidecar exits
trap "kill $NTFY_PID 2>/dev/null" EXIT TERM INT

# Brief pause for ntfy to bind
sleep 1

# Start MCP sidecar as foreground process (exec replaces shell)
exec /usr/local/bin/mcp-sidecar
