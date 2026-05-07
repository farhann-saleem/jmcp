package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/farhann-saleem/jmcp/internal/output"
)

func renderTopology(w io.Writer, topo *Topology) {
	fmt.Fprintln(w, topo.TraceID)
	nodes := append([]TopologySpan(nil), topo.Spans...)
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Path < nodes[j].Path
	})
	for i, span := range nodes {
		depth := strings.Count(span.Path, "/")
		prefix := strings.Repeat("  ", depth)
		branch := "└──"
		if i < len(nodes)-1 {
			branch = "├──"
		}
		status := strings.ToUpper(span.Status)
		if status == "" {
			status = "UNSET"
		}
		truncated := ""
		if span.TruncatedChildren > 0 {
			truncated = fmt.Sprintf(" [+%d truncated]", span.TruncatedChildren)
		}
		fmt.Fprintf(w, "%s%s [%s] %s (%s) %s%s\n",
			prefix, branch, span.Service, span.SpanName, output.FormatDurationUS(span.DurationUS), status, truncated)
	}
}

func renderErrors(w io.Writer, errs *TraceErrors) {
	fmt.Fprintf(w, "Total errors: %d (showing %d)\n\n", errs.TotalErrorCount, len(errs.Spans))
	rows := make([][]string, 0, len(errs.Spans))
	for _, span := range errs.Spans {
		msg := span.Status.Message
		if msg == "" {
			msg = span.Status.Code
		}
		rows = append(rows, []string{
			span.SpanID,
			span.Service,
			span.SpanName,
			output.FormatDurationUS(span.DurationUS),
			msg,
		})
	}
	output.TableRows(w, []string{"SPAN ID", "SERVICE", "SPAN NAME", "DURATION", "ERROR"}, rows)
	if errs.TotalErrorCount > len(errs.Spans) {
		fmt.Fprintf(w, "\nWarning: %d additional error spans were truncated by the server.\n", errs.TotalErrorCount-len(errs.Spans))
	}
}

func renderDetails(w io.Writer, details *SpanDetails) {
	for i, span := range details.Spans {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Span: %s\n", span.SpanID)
		output.KeyValues(w, [][2]string{
			{"Service", span.Service},
			{"Name", span.SpanName},
			{"Parent", span.ParentSpanID},
			{"Start", span.StartTime},
			{"Duration", output.FormatDurationUS(span.DurationUS)},
			{"Status", formatStatus(span.Status)},
		})
		if len(span.Attributes) > 0 {
			fmt.Fprintln(w, "\nAttributes:")
			for _, line := range output.StringMapLines(span.Attributes) {
				fmt.Fprintln(w, line)
			}
		}
		if len(span.Events) > 0 {
			fmt.Fprintln(w, "\nEvents:")
			for _, event := range span.Events {
				fmt.Fprintf(w, "  [%s] %s\n", event.Timestamp, event.Name)
				for _, line := range output.StringMapLines(event.Attributes) {
					fmt.Fprintln(w, line)
				}
			}
		}
	}
}

func renderCriticalPath(w io.Writer, cp *CriticalPath) {
	percent := output.Percent(float64(cp.CriticalPathDurationUS), float64(cp.TotalDurationUS))
	fmt.Fprintf(w, "Trace: %s\n", cp.TraceID)
	fmt.Fprintf(w, "Total Duration:         %s\n", output.FormatDurationUS(cp.TotalDurationUS))
	fmt.Fprintf(w, "Critical Path Duration: %s (%s)\n\n", output.FormatDurationUS(cp.CriticalPathDurationUS), percent)
	rows := make([][]string, 0, len(cp.Segments))
	for i, seg := range cp.Segments {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			seg.Service,
			seg.SpanName,
			output.FormatDurationUS(seg.SelfTimeUS),
			fmt.Sprintf("%s-%s", output.FormatDurationUS(seg.StartOffsetUS), output.FormatDurationUS(seg.EndOffsetUS)),
		})
	}
	output.TableRows(w, []string{"SEGMENT", "SERVICE", "SPAN NAME", "SELF TIME", "OFFSET"}, rows)
	if len(cp.Segments) > 0 {
		fmt.Fprintln(w)
		for _, seg := range cp.Segments {
			width := 1
			if cp.CriticalPathDurationUS > 0 {
				width = int(float64(seg.SelfTimeUS) / float64(cp.CriticalPathDurationUS) * 30)
			}
			if width < 1 {
				width = 1
			}
			fmt.Fprintf(w, "[%s %s] ", seg.Service, strings.Repeat("#", width))
		}
		fmt.Fprintln(w)
	}
}

func renderDeps(w io.Writer, deps *Dependencies) {
	rows := make([][]string, 0, len(deps.Dependencies))
	for _, dep := range deps.Dependencies {
		rows = append(rows, []string{dep.Caller, dep.Callee, fmt.Sprintf("%d", dep.CallCount)})
	}
	output.TableRows(w, []string{"CALLER", "CALLEE", "CALLS"}, rows)
}

func renderSnapshot(w io.Writer, snap *Snapshot) {
	output.KeyValues(w, [][2]string{
		{"Label", snap.Label},
		{"Service", snap.Service},
		{"Captured", snap.CapturedAt},
		{"Traces", fmt.Sprintf("%d", snap.Stats.TotalTraces)},
		{"Errors", fmt.Sprintf("%d", snap.Stats.ErrorTraces)},
		{"Avg Duration", output.FormatDurationUS(snap.Stats.AvgDuration)},
		{"P95 Duration", output.FormatDurationUS(snap.Stats.P95Duration)},
	})
}

func formatStatus(status SpanStatus) string {
	if status.Message != "" {
		return status.Code + " - " + status.Message
	}
	if status.Code != "" {
		return status.Code
	}
	return "Unset"
}
