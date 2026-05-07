package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/farhann-saleem/jmcp/internal/output"
)

func noColor() *output.Colorizer { return output.DisabledColorizer() }

func TestRenderTopology(t *testing.T) {
	topo := &Topology{
		TraceID: "abc123",
		Spans: []TopologySpan{
			{Path: "root", Service: "frontend", SpanName: "GET /api", DurationUS: 1500, Status: "ok"},
			{Path: "root/child", Service: "backend", SpanName: "query", DurationUS: 800, Status: "error", TruncatedChildren: 2},
		},
	}
	var buf bytes.Buffer
	renderTopology(&buf, topo, noColor())
	out := buf.String()
	if !strings.Contains(out, "abc123") {
		t.Error("missing trace ID")
	}
	if !strings.Contains(out, "frontend") {
		t.Error("missing service name")
	}
	if !strings.Contains(out, "+2 truncated") {
		t.Error("missing truncation warning")
	}
}

func TestRenderErrors(t *testing.T) {
	errs := &TraceErrors{
		TotalErrorCount: 5,
		Spans: []FullSpan{
			{SpanID: "span1", Service: "svc", SpanName: "op", DurationUS: 100, Status: SpanStatus{Code: "ERROR", Message: "timeout"}},
		},
	}
	var buf bytes.Buffer
	renderErrors(&buf, errs, noColor())
	out := buf.String()
	if !strings.Contains(out, "Total errors: 5") {
		t.Error("missing total count")
	}
	if !strings.Contains(out, "timeout") {
		t.Error("missing error message")
	}
	if !strings.Contains(out, "4 additional error spans") {
		t.Error("missing truncation warning")
	}
}

func TestRenderDetails(t *testing.T) {
	details := &SpanDetails{
		Spans: []FullSpan{
			{SpanID: "s1", Service: "frontend", SpanName: "GET", DurationUS: 500, Status: SpanStatus{Code: "OK"}},
		},
	}
	var buf bytes.Buffer
	renderDetails(&buf, details, noColor())
	out := buf.String()
	if !strings.Contains(out, "frontend") {
		t.Error("missing service")
	}
}

func TestRenderCriticalPath(t *testing.T) {
	cp := &CriticalPath{
		TraceID:                "t1",
		TotalDurationUS:        10000,
		CriticalPathDurationUS: 8000,
		Segments: []CriticalPathSegment{
			{SpanID: "s1", Service: "api", SpanName: "handle", SelfTimeUS: 5000, StartOffsetUS: 0, EndOffsetUS: 5000},
		},
	}
	var buf bytes.Buffer
	renderCriticalPath(&buf, cp, noColor())
	out := buf.String()
	if !strings.Contains(out, "api") {
		t.Error("missing service in critical path")
	}
}

func TestRenderDeps(t *testing.T) {
	deps := &Dependencies{
		Dependencies: []Dependency{
			{Caller: "frontend", Callee: "backend", CallCount: 42},
		},
	}
	var buf bytes.Buffer
	renderDeps(&buf, deps, noColor())
	out := buf.String()
	if !strings.Contains(out, "frontend") || !strings.Contains(out, "42") {
		t.Error("missing dependency data")
	}
}

func TestRenderSnapshot(t *testing.T) {
	snap := &Snapshot{
		Label:   "test",
		Service: "myservice",
	}
	var buf bytes.Buffer
	renderSnapshot(&buf, snap, noColor())
	out := buf.String()
	if !strings.Contains(out, "myservice") {
		t.Error("missing service in snapshot")
	}
}
