package cmd

import (
	"encoding/json"
	"time"

	"github.com/farhann-saleem/jmcp/internal/analysis"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

type globalOptions struct {
	Endpoint string
	Output   string
	Save     string
	Timeout  time.Duration
	NoColor  bool
	Verbose  bool
}

type Health struct {
	Status  string `json:"status"`
	Server  string `json:"server"`
	Version string `json:"version"`
}

type Services struct {
	Services []string `json:"services"`
}

type SpanNames struct {
	SpanNames []SpanName `json:"span_names"`
}

type SpanName struct {
	Name string `json:"name"`
	Kind string `json:"span_kind"`
}

type SearchResult struct {
	TraceCount int            `json:"trace_count"`
	Traces     []TraceSummary `json:"traces"`
}

type TraceSummary struct {
	TraceID      string   `json:"trace_id"`
	RootService  string   `json:"root_service"`
	RootSpanName string   `json:"root_span_name"`
	StartTime    string   `json:"start_time"`
	DurationUS   int64    `json:"duration_us"`
	SpanCount    int      `json:"span_count"`
	ServiceCount int      `json:"service_count"`
	Services     []string `json:"services"`
	HasErrors    bool     `json:"has_errors"`
}

func (t TraceSummary) Duration() int64        { return t.DurationUS }
func (t TraceSummary) HasError() bool         { return t.HasErrors }
func (t TraceSummary) ServicesList() []string { return t.Services }

type Topology struct {
	TraceID string         `json:"trace_id"`
	Spans   []TopologySpan `json:"spans"`
}

type TopologySpan struct {
	Path              string `json:"path"`
	Service           string `json:"service"`
	SpanName          string `json:"span_name"`
	StartTime         string `json:"start_time"`
	DurationUS        int64  `json:"duration_us"`
	Status            string `json:"status"`
	TruncatedChildren int    `json:"truncated_children"`
}

type TraceErrors struct {
	TraceID         string     `json:"trace_id"`
	TotalErrorCount int        `json:"total_error_count"`
	Spans           []FullSpan `json:"spans"`
}

type SpanDetails struct {
	TraceID string     `json:"trace_id"`
	Spans   []FullSpan `json:"spans"`
}

type FullSpan struct {
	SpanID       string            `json:"span_id"`
	TraceID      string            `json:"trace_id"`
	ParentSpanID string            `json:"parent_span_id"`
	Service      string            `json:"service"`
	SpanName     string            `json:"span_name"`
	StartTime    string            `json:"start_time"`
	DurationUS   int64             `json:"duration_us"`
	Status       SpanStatus        `json:"status"`
	Attributes   map[string]any    `json:"attributes"`
	Events       []SpanEvent       `json:"events"`
	Links        []json.RawMessage `json:"links"`
}

type SpanStatus struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type SpanEvent struct {
	Name       string         `json:"name"`
	Timestamp  string         `json:"timestamp"`
	Attributes map[string]any `json:"attributes"`
}

type CriticalPath struct {
	TraceID                string                `json:"trace_id"`
	TotalDurationUS        int64                 `json:"total_duration_us"`
	CriticalPathDurationUS int64                 `json:"critical_path_duration_us"`
	Segments               []CriticalPathSegment `json:"segments"`
}

type CriticalPathSegment struct {
	SpanID        string `json:"span_id"`
	Service       string `json:"service"`
	SpanName      string `json:"span_name"`
	SelfTimeUS    int64  `json:"self_time_us"`
	StartOffsetUS int64  `json:"start_offset_us"`
	EndOffsetUS   int64  `json:"end_offset_us"`
}

type Dependencies struct {
	Dependencies []Dependency `json:"dependencies"`
}

type Dependency struct {
	Caller    string `json:"caller"`
	Callee    string `json:"callee"`
	CallCount int    `json:"call_count"`
}

type Snapshot struct {
	Label      string         `json:"label"`
	Service    string         `json:"service"`
	CapturedAt string         `json:"captured_at"`
	Endpoint   string         `json:"endpoint"`
	Query      map[string]any `json:"query"`
	Traces     []TraceSummary `json:"traces"`
	Stats      analysis.Stats `json:"stats"`
}

type Report struct {
	Title        string        `json:"title"`
	TraceID      string        `json:"trace_id"`
	Generated    string        `json:"generated"`
	Endpoint     string        `json:"endpoint"`
	ToolVersion  string        `json:"tool_version"`
	Summary      *TraceSummary `json:"summary,omitempty"`
	Topology     *Topology     `json:"topology,omitempty"`
	Errors       *TraceErrors  `json:"errors,omitempty"`
	CriticalPath *CriticalPath `json:"critical_path,omitempty"`
	ErrorDetails *SpanDetails  `json:"error_details,omitempty"`
	Notes        []string      `json:"notes,omitempty"`
}
