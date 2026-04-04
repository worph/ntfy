# Stage 1: Build the MCP sidecar
FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY mcp-sidecar/go.mod mcp-sidecar/go.sum* ./
RUN go mod download 2>/dev/null || true
COPY mcp-sidecar/ .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mcp-sidecar ./cmd/sidecar

# Stage 2: Final image based on official ntfy
FROM binwiederhier/ntfy:latest

COPY --from=builder /mcp-sidecar /usr/local/bin/mcp-sidecar
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /usr/local/bin/mcp-sidecar

EXPOSE 8080/tcp 9099/tcp 9099/udp

ENTRYPOINT ["/entrypoint.sh"]
