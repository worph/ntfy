package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-sidecar/internal/ntfy"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var sendMessageTool = mcp.Tool{
	Name:        "send_message",
	Description: "Send a push notification to an ntfy topic. Supports title, priority (1-5), emoji tags, click URL, and action buttons.",
	InputSchema: mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"topic":    map[string]any{"type": "string", "description": "Target topic name (defaults to configured default)"},
			"message":  map[string]any{"type": "string", "description": "Notification body text"},
			"title":    map[string]any{"type": "string", "description": "Notification title"},
			"priority": map[string]any{"type": "integer", "description": "Priority: 1 (min) to 5 (max), default 3", "minimum": 1, "maximum": 5},
			"tags":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Emoji tags (e.g. ['warning'] shows warning emoji)"},
			"click":    map[string]any{"type": "string", "description": "URL to open when notification is clicked"},
			"actions":  map[string]any{"type": "array", "description": "Action buttons (view, http, broadcast)"},
		},
		Required: []string{"message"},
	},
}

var sendPhotoTool = mcp.Tool{
	Name:        "send_photo",
	Description: "Send a push notification with an image attachment to an ntfy topic.",
	InputSchema: mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"topic":    map[string]any{"type": "string", "description": "Target topic name (defaults to configured default)"},
			"url":      map[string]any{"type": "string", "description": "URL of the image to attach"},
			"caption":  map[string]any{"type": "string", "description": "Message body alongside the image"},
			"title":    map[string]any{"type": "string", "description": "Notification title"},
			"priority": map[string]any{"type": "integer", "description": "Priority: 1 (min) to 5 (max)", "minimum": 1, "maximum": 5},
		},
		Required: []string{"url"},
	},
}

var listMessagesTool = mcp.Tool{
	Name:        "list_messages",
	Description: "Read cached messages from an ntfy topic. Returns recent notifications.",
	InputSchema: mcp.ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"topic": map[string]any{"type": "string", "description": "Topic to read from (defaults to configured default)"},
			"since": map[string]any{"type": "string", "description": "Time duration like '30m', '1h', '1d' or a message ID. Default: '1h'"},
			"limit": map[string]any{"type": "integer", "description": "Max messages to return. Default: 50"},
		},
	},
}

var serverInfoTool = mcp.Tool{
	Name:        "server_info",
	Description: "Get ntfy server health status and version information.",
	InputSchema: mcp.ToolInputSchema{
		Type:       "object",
		Properties: map[string]any{},
	},
}

// AllTools returns all tool definitions for registration.
func AllTools() []mcp.Tool {
	return []mcp.Tool{sendMessageTool, sendPhotoTool, listMessagesTool, serverInfoTool}
}

// GetToolDefinitions returns tool definitions as plain maps for the Beacon announce payload.
func GetToolDefinitions() []map[string]any {
	tools := AllTools()
	defs := make([]map[string]any, len(tools))
	for i, t := range tools {
		defs[i] = map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		}
	}
	return defs
}

func HandleSendMessage(client *ntfy.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := extractArgs(req)

		message, _ := args["message"].(string)
		if message == "" {
			return mcp.NewToolResultError("missing required parameter: message"), nil
		}

		pubReq := ntfy.PublishRequest{
			Message: message,
		}
		if v, ok := args["topic"].(string); ok && v != "" {
			pubReq.Topic = v
		}
		if v, ok := args["title"].(string); ok {
			pubReq.Title = v
		}
		if v, ok := args["priority"].(float64); ok {
			pubReq.Priority = int(v)
		}
		if v, ok := args["tags"].([]interface{}); ok {
			for _, tag := range v {
				if s, ok := tag.(string); ok {
					pubReq.Tags = append(pubReq.Tags, s)
				}
			}
		}
		if v, ok := args["click"].(string); ok {
			pubReq.Click = v
		}
		if v, ok := args["actions"].([]interface{}); ok {
			for _, a := range v {
				b, err := json.Marshal(a)
				if err == nil {
					pubReq.Actions = append(pubReq.Actions, json.RawMessage(b))
				}
			}
		}

		_, err := client.Publish(ctx, pubReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to publish: %v", err)), nil
		}

		topic := pubReq.Topic
		if topic == "" {
			topic = client.DefaultTopic()
		}
		msg := fmt.Sprintf("Notification sent to topic '%s'", topic)
		if url := client.TopicURL(topic); url != "" {
			msg += fmt.Sprintf("\nSubscribe: %s", url)
		}
		return mcp.NewToolResultText(msg), nil
	}
}

func HandleSendPhoto(client *ntfy.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := extractArgs(req)

		url, _ := args["url"].(string)
		if url == "" {
			return mcp.NewToolResultError("missing required parameter: url"), nil
		}

		pubReq := ntfy.PublishRequest{
			Attach: url,
		}
		if v, ok := args["topic"].(string); ok && v != "" {
			pubReq.Topic = v
		}
		if v, ok := args["caption"].(string); ok {
			pubReq.Message = v
		}
		if v, ok := args["title"].(string); ok {
			pubReq.Title = v
		}
		if v, ok := args["priority"].(float64); ok {
			pubReq.Priority = int(v)
		}

		_, err := client.Publish(ctx, pubReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to publish: %v", err)), nil
		}

		topic := pubReq.Topic
		if topic == "" {
			topic = client.DefaultTopic()
		}
		msg := fmt.Sprintf("Photo notification sent to topic '%s'", topic)
		if url := client.TopicURL(topic); url != "" {
			msg += fmt.Sprintf("\nSubscribe: %s", url)
		}
		return mcp.NewToolResultText(msg), nil
	}
}

func HandleListMessages(client *ntfy.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := extractArgs(req)

		topic, _ := args["topic"].(string)
		since, _ := args["since"].(string)
		limit := 0
		if v, ok := args["limit"].(float64); ok {
			limit = int(v)
		}

		messages, err := client.ListMessages(ctx, topic, since, limit)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list messages: %v", err)), nil
		}

		if len(messages) == 0 {
			return mcp.NewToolResultText("No messages found"), nil
		}

		data, err := json.MarshalIndent(messages, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format messages: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func HandleServerInfo(client *ntfy.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		health, healthErr := client.Health(ctx)
		info, infoErr := client.Info(ctx)

		result := map[string]any{}
		if healthErr == nil {
			result["health"] = health
		} else {
			result["health_error"] = healthErr.Error()
		}
		if infoErr == nil {
			for k, v := range info {
				result[k] = v
			}
		} else {
			result["info_error"] = infoErr.Error()
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format info: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func extractArgs(req mcp.CallToolRequest) map[string]any {
	if m, ok := req.Params.Arguments.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
