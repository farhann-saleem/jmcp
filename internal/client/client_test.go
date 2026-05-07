package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientInitializesAndCallsTool(t *testing.T) {
	var sawInitialized bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		switch payload["method"] {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "session-1")
			writeSSE(w, `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`)
		case "notifications/initialized":
			if r.Header.Get("Mcp-Session-Id") != "session-1" {
				t.Fatalf("notification session = %q", r.Header.Get("Mcp-Session-Id"))
			}
			sawInitialized = true
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			if !sawInitialized {
				t.Fatal("tool call happened before initialized notification")
			}
			if r.Header.Get("Mcp-Session-Id") != "session-1" {
				t.Fatalf("tool session = %q", r.Header.Get("Mcp-Session-Id"))
			}
			writeSSE(w, `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"status\":\"ok\"}"}]}}`)
		default:
			t.Fatalf("unexpected method %v", payload["method"])
		}
	}))
	defer server.Close()

	c := New(server.URL)
	result, err := c.CallTool(context.Background(), "health", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Content) != `{"status":"ok"}` {
		t.Fatalf("content = %s", result.Content)
	}
}

func TestClientReconnectsOnSessionError(t *testing.T) {
	session := "session-1"
	toolCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		switch payload["method"] {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", session)
			writeSSE(w, `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`)
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			toolCalls++
			if toolCalls == 1 {
				session = "session-2"
				writeSSE(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"invalid session"}}`)
				return
			}
			if r.Header.Get("Mcp-Session-Id") != "session-2" {
				t.Fatalf("retry session = %q", r.Header.Get("Mcp-Session-Id"))
			}
			writeSSE(w, `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"status\":\"ok\"}"}]}}`)
		}
	}))
	defer server.Close()

	c := New(server.URL)
	result, err := c.CallTool(context.Background(), "health", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Content) != `{"status":"ok"}` {
		t.Fatalf("content = %s", result.Content)
	}
}

func writeSSE(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "text/event-stream")
	_, _ = w.Write([]byte("event: message\n"))
	_, _ = w.Write([]byte("data: " + data + "\n\n"))
}
