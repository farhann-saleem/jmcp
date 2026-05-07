package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const protocolVersion = "2024-11-05"

type Client struct {
	endpoint   string
	sessionID  string
	requestID  int
	httpClient *http.Client
	verbose    bool
	logger     func(string, ...any)
	mu         sync.Mutex
}

type Option func(*Client)

func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

func WithVerbose(verbose bool, logger func(string, ...any)) Option {
	return func(c *Client) {
		c.verbose = verbose
		c.logger = logger
	}
}

func New(endpoint string, opts ...Option) *Client {
	c := &Client{
		endpoint:   endpoint,
		requestID:  0,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     func(string, ...any) {},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) Endpoint() string {
	return c.endpoint
}

func (c *Client) Initialize(ctx context.Context) error {
	c.mu.Lock()
	c.sessionID = ""
	c.requestID = 1
	id := c.requestID
	c.mu.Unlock()

	c.log("-> initialize (new session)")
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "jmcp",
				"version": "0.1.0",
			},
		},
	}

	httpResp, rpcResp, _, err := c.post(ctx, payload, "")
	if err != nil {
		return err
	}
	if rpcResp != nil && rpcResp.Error != nil {
		return rpcResp.Error
	}
	sessionID := httpResp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		return errors.New("initialize response missing Mcp-Session-Id header")
	}

	c.mu.Lock()
	c.sessionID = sessionID
	c.mu.Unlock()
	c.log("<- session: %s", sessionID)

	c.log("-> notifications/initialized")
	notify := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	}
	if _, _, _, err := c.post(ctx, notify, sessionID); err != nil {
		return fmt.Errorf("send initialized notification: %w", err)
	}
	return nil
}

func (c *Client) CallTool(ctx context.Context, tool string, args any) (*ToolResult, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}

	result, err := c.callToolOnce(ctx, tool, args)
	if err == nil {
		return result, nil
	}
	if !isSessionError(err) {
		return nil, err
	}

	c.log("session expired; re-initializing")
	if initErr := c.Initialize(ctx); initErr != nil {
		return nil, fmt.Errorf("session expired and re-initialization failed: %w", initErr)
	}
	return c.callToolOnce(ctx, tool, args)
}

func (c *Client) ensureSession(ctx context.Context) error {
	c.mu.Lock()
	hasSession := c.sessionID != ""
	c.mu.Unlock()
	if hasSession {
		return nil
	}
	return c.Initialize(ctx)
}

func (c *Client) callToolOnce(ctx context.Context, tool string, args any) (*ToolResult, error) {
	c.mu.Lock()
	c.requestID++
	id := c.requestID
	sessionID := c.sessionID
	c.mu.Unlock()

	if args == nil {
		args = map[string]any{}
	}
	c.log("-> tools/call %s %s", tool, compactJSON(args))
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      tool,
			"arguments": args,
		},
	}

	_, rpcResp, content, err := c.post(ctx, payload, sessionID)
	if err != nil {
		return nil, err
	}
	if rpcResp != nil && rpcResp.Error != nil {
		return nil, fmt.Errorf("%s: %w", tool, rpcResp.Error)
	}
	return &ToolResult{
		RawJSONRPC: mustMarshal(rpcResp),
		Content:    content,
	}, nil
}

func (c *Client) post(ctx context.Context, payload any, sessionID string) (*http.Response, *JSONRPCResponse, json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}

	start := time.Now()
	c.log("POST %s", c.endpoint)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("cannot connect to MCP server at %s - is Jaeger running? %w", c.endpoint, err)
	}
	defer resp.Body.Close()
	c.log("<- %s (%s)", resp.Status, time.Since(start).Round(time.Millisecond))

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = resp.Status
		}
		return resp, nil, nil, httpError{status: resp.StatusCode, message: msg}
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return resp, nil, nil, nil
	}

	rpcResp, content, err := ParseSSEResponse(bytes.NewReader(raw))
	if err != nil {
		return resp, nil, nil, err
	}
	return resp, rpcResp, content, nil
}

func (c *Client) log(format string, args ...any) {
	if c.verbose {
		c.logger("[jmcp] "+format+"\n", args...)
	}
}

type httpError struct {
	status  int
	message string
}

func (e httpError) Error() string {
	return e.message
}

func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "session") ||
		strings.Contains(msg, "mcp-session") ||
		strings.Contains(msg, "unauthorized") ||
		strings.Contains(msg, "invalid") && strings.Contains(msg, "session")
}

func compactJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
