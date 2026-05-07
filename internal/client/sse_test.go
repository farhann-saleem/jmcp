package client

import (
	"strings"
	"testing"
)

func TestParseSSEResponseExtractsDoubleEncodedToolContent(t *testing.T) {
	body := `event: message
data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"{\"status\":\"ok\",\"server\":\"jaeger\"}"}]}}

`
	resp, content, err := ParseSSEResponse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != 2 {
		t.Fatalf("ID = %d, want 2", resp.ID)
	}
	got := strings.TrimSpace(string(content))
	want := `{"status":"ok","server":"jaeger"}`
	if got != want {
		t.Fatalf("content = %s, want %s", got, want)
	}
}

func TestParseSSEResponseHandlesJSONRPCError(t *testing.T) {
	body := `event: message
data: {"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"invalid session"}}

`
	resp, _, err := ParseSSEResponse(strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Error == nil || resp.Error.Message != "invalid session" {
		t.Fatalf("error = %#v, want invalid session", resp.Error)
	}
}
