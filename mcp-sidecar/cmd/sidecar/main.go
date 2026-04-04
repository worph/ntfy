package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"mcp-sidecar/internal/beacon"
	mcpserver "mcp-sidecar/internal/mcp"
	"mcp-sidecar/internal/ntfy"

	"golang.org/x/sync/errgroup"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	ntfyURL := envOr("NTFY_BASE_URL", "http://localhost:8080")
	publicURL := os.Getenv("PUBLIC_URL")
	mcpPort := envOrInt("MCP_PORT", 9099)
	defaultTopic := envOr("NTFY_DEFAULT_TOPIC", "general")
	authToken := os.Getenv("NTFY_AUTH_TOKEN")

	client := ntfy.NewClient(ntfyURL, publicURL, authToken, defaultTopic)
	toolDefs := mcpserver.GetToolDefinitions()

	log.Printf("Starting ntfy MCP sidecar (ntfy=%s, mcp_port=%d, default_topic=%s)", ntfyURL, mcpPort, defaultTopic)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return mcpserver.StartServer(ctx, client, mcpPort)
	})

	g.Go(func() error {
		return beacon.StartAnnouncer(ctx, toolDefs, mcpPort)
	})

	if err := g.Wait(); err != nil && ctx.Err() == nil {
		log.Fatalf("Fatal error: %v", err)
	}
	log.Println("Sidecar stopped")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
