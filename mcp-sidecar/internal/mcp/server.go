package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"mcp-sidecar/internal/ntfy"

	"github.com/mark3labs/mcp-go/server"
)

// StartServer starts the MCP streamable HTTP server on the given port.
func StartServer(ctx context.Context, ntfyClient *ntfy.Client, port int) error {
	s := server.NewMCPServer("ntfy", "1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(sendMessageTool, HandleSendMessage(ntfyClient))
	s.AddTool(sendPhotoTool, HandleSendPhoto(ntfyClient))
	s.AddTool(listMessagesTool, HandleListMessages(ntfyClient))
	s.AddTool(serverInfoTool, HandleServerInfo(ntfyClient))

	httpTransport := server.NewStreamableHTTPServer(s,
		server.WithEndpointPath("/mcp"),
		server.WithStateLess(true),
	)

	addr := fmt.Sprintf(":%d", port)
	httpServer := &http.Server{
		Addr:    addr,
		Handler: httpTransport,
	}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down MCP server...")
		httpServer.Close()
	}()

	log.Printf("MCP server listening on %s/mcp", addr)
	err := httpServer.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
