package ntfy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PublishRequest represents a notification to publish via ntfy.
type PublishRequest struct {
	Topic    string            `json:"topic"`
	Message  string            `json:"message"`
	Title    string            `json:"title,omitempty"`
	Priority int               `json:"priority,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Click    string            `json:"click,omitempty"`
	Actions  []json.RawMessage `json:"actions,omitempty"`
	Attach   string            `json:"attach,omitempty"`
}

// Client is an HTTP client for the ntfy server API.
type Client struct {
	baseURL      string
	publicURL    string
	authToken    string
	defaultTopic string
	http         *http.Client
}

// NewClient creates a new ntfy API client.
func NewClient(baseURL, publicURL, authToken, defaultTopic string) *Client {
	return &Client{
		baseURL:      baseURL,
		publicURL:    publicURL,
		authToken:    authToken,
		defaultTopic: defaultTopic,
		http:         &http.Client{Timeout: 10 * time.Second},
	}
}

// DefaultTopic returns the configured default topic.
func (c *Client) DefaultTopic() string {
	return c.defaultTopic
}

// TopicURL returns the public URL for a topic, or empty if no public URL is configured.
func (c *Client) TopicURL(topic string) string {
	if c.publicURL == "" {
		return ""
	}
	return c.publicURL + "/" + topic
}

// Publish sends a notification to the ntfy server.
func (c *Client) Publish(ctx context.Context, req PublishRequest) (map[string]any, error) {
	if req.Topic == "" {
		req.Topic = c.defaultTopic
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal publish request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("publish: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("publish failed (status %d): %s", resp.StatusCode, string(b))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

// ListMessages retrieves cached messages from a topic.
// Response is NDJSON (one JSON object per line).
func (c *Client) ListMessages(ctx context.Context, topic, since string, limit int) ([]map[string]any, error) {
	if topic == "" {
		topic = c.defaultTopic
	}
	if since == "" {
		since = "1h"
	}
	if limit <= 0 {
		limit = 50
	}

	url := fmt.Sprintf("%s/%s/json?poll=1&since=%s", c.baseURL, topic, since)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list messages failed (status %d): %s", resp.StatusCode, string(b))
	}

	var messages []map[string]any
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() && len(messages) < limit {
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read messages: %w", err)
	}
	return messages, nil
}

// Health checks the ntfy server health endpoint.
func (c *Client) Health(ctx context.Context) (map[string]any, error) {
	return c.getJSON(ctx, c.baseURL+"/v1/health")
}

// Info returns ntfy server information.
func (c *Client) Info(ctx context.Context) (map[string]any, error) {
	return c.getJSON(ctx, c.baseURL+"/v1/info")
}

func (c *Client) getJSON(ctx context.Context, url string) (map[string]any, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(b))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
