#!/bin/sh
set -e

NTFY_CONF=/etc/ntfy/server.yml
AUTH_FILE=/var/cache/ntfy/user.db
CACHE_FILE=/var/cache/ntfy/cache.db

# --- Generate ntfy config file ---
mkdir -p /etc/ntfy
cat > "$NTFY_CONF" <<EOF
base-url: "${NTFY_BASE_URL:-http://localhost:8080}"
listen-http: ":8080"
behind-proxy: true
cache-file: "$CACHE_FILE"
auth-file: "$AUTH_FILE"
auth-default-access: "deny-all"
EOF

# --- Start ntfy server ---
ntfy serve &
NTFY_PID=$!
sleep 1

# --- First boot: create admin user and auth token ---
if [ ! -f "$AUTH_FILE" ] || ! ntfy user list 2>/dev/null | grep -q "admin"; then
  echo "First boot: creating admin user..."
  # Wait for ntfy to initialize the auth database
  for i in 1 2 3 4 5; do
    NTFY_PASSWORD="${NTFY_ADMIN_PASSWORD:-admin}" \
      ntfy user add --role=admin --ignore-exists admin 2>/dev/null && break
    sleep 1
  done

  # Generate a token for the MCP sidecar
  TOKEN_OUTPUT=$(ntfy token add admin 2>/dev/null || true)
  GENERATED_TOKEN=$(echo "$TOKEN_OUTPUT" | grep -oE 'tk_[a-zA-Z0-9]+')
  if [ -n "$GENERATED_TOKEN" ]; then
    echo "Created sidecar token: ${GENERATED_TOKEN}"
    echo "$GENERATED_TOKEN" > /var/cache/ntfy/sidecar-token
  fi
fi

# --- Start nginx reverse proxy ---
nginx

trap "kill $NTFY_PID 2>/dev/null; nginx -s quit 2>/dev/null" EXIT TERM INT

# Brief pause for ntfy to bind
sleep 1

# --- Start MCP sidecar as foreground process ---
exec /usr/local/bin/mcp-sidecar
