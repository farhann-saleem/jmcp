package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type JSONRPCResponse struct {
	JSONRPC string           `json:"jsonrpc,omitempty"`
	ID      int              `json:"id,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *JSONRPCError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("json-rpc error %d: %s", e.Code, e.Message)
}

type ToolResult struct {
	RawJSONRPC json.RawMessage
	Content    json.RawMessage
}

// ParseSSEResponse parses Jaeger's JSON-RPC-over-SSE response and extracts the
// double-encoded result.content[0].text payload when present.
func ParseSSEResponse(body io.Reader) (*JSONRPCResponse, json.RawMessage, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var data bytes.Buffer
	parse := func() (*JSONRPCResponse, json.RawMessage, error) {
		if strings.TrimSpace(data.String()) == "" {
			return nil, nil, nil
		}

		raw := json.RawMessage(bytes.TrimSpace(data.Bytes()))
		var resp JSONRPCResponse
		if err := json.Unmarshal(raw, &resp); err != nil {
			return nil, nil, fmt.Errorf("parse sse data as json-rpc: %w", err)
		}
		content := raw
		if resp.Result != nil {
			extracted, err := extractToolContent(*resp.Result)
			if err != nil {
				return nil, nil, err
			}
			if len(extracted) > 0 {
				content = extracted
			}
		}
		return &resp, content, nil
	}

	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			resp, content, err := parse()
			if err != nil || resp != nil {
				return resp, content, err
			}
			data.Reset()
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		case strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") || strings.HasPrefix(line, "id:") || strings.HasPrefix(line, "retry:"):
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("read sse response: %w", err)
	}
	return parse()
}

func extractToolContent(result json.RawMessage) (json.RawMessage, error) {
	var envelope struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		return nil, fmt.Errorf("parse json-rpc result: %w", err)
	}
	if len(envelope.Content) == 0 || envelope.Content[0].Text == "" {
		return nil, nil
	}

	text := strings.TrimSpace(envelope.Content[0].Text)
	if text == "" {
		return nil, nil
	}
	if json.Valid([]byte(text)) {
		return json.RawMessage(text), nil
	}
	encoded, err := json.Marshal(text)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
