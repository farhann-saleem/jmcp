package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/farhann-saleem/jmcp/internal/analysis"
	"github.com/farhann-saleem/jmcp/internal/client"
	"github.com/farhann-saleem/jmcp/internal/output"
)

func runInvestigate(ctx context.Context, c *client.Client, args []string, opts globalOptions, clr *output.Colorizer) (any, error) {
	fs := newFlagSet("investigate")
	depth := intFlag(fs, "depth", 0, "max topology depth")
	if err := parseWithHelp(fs, "investigate", "Full trace investigation", "investigate <trace-id> [flags]", args); err != nil {
		return nil, err
	}
	traceID, _, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return nil, err
	}
	topologyArgs := []string{traceID}
	if *depth > 0 {
		topologyArgs = append(topologyArgs, "--depth", fmt.Sprintf("%d", *depth))
	}
	topo, _, err := runTopology(ctx, c, topologyArgs)
	if err != nil {
		return nil, err
	}
	errs, _, err := runErrors(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	cp, _, err := runCriticalPath(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	if opts.Output == string(output.JSON) {
		return map[string]any{"topology": topo, "errors": errs, "critical_path": cp}, nil
	}
	fmt.Println("=== Topology ===")
	renderTopology(os.Stdout, topo, clr)
	fmt.Println("\n=== Errors ===")
	renderErrors(os.Stdout, errs, clr)
	fmt.Println("\n=== Critical Path ===")
	renderCriticalPath(os.Stdout, cp, clr)
	return nil, nil
}

func runInit(args []string) (any, error) {
	fs := newFlagSet("init")
	if err := parseWithHelp(fs, "init", "Initialize .jmcp project directory", "init", args); err != nil {
		return nil, err
	}
	dirs := []string{".jmcp", ".jmcp/reports", ".jmcp/snapshots"}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	config := "endpoint: http://localhost:16687/mcp\ndefaults:\n  search_depth: 20\n  since: 1h\n  output: table\n"
	if err := os.WriteFile(".jmcp/config.yaml", []byte(config), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(".jmcp/.gitignore", []byte("snapshots/\n# Reports are committed - they are investigation artifacts\n"), 0o644); err != nil {
		return nil, err
	}
	fmt.Println("Created .jmcp/")
	fmt.Println("  .jmcp/reports/")
	fmt.Println("  .jmcp/snapshots/")
	fmt.Println("  .jmcp/config.yaml")
	fmt.Println("  .jmcp/.gitignore")
	return nil, nil
}

func runReport(ctx context.Context, c *client.Client, args []string, opts globalOptions, build BuildInfo) (any, error) {
	fs := newFlagSet("report")
	dir := stringFlag(fs, "dir", ".jmcp/reports", "report directory")
	title := stringFlag(fs, "title", "", "report title")
	format := stringFlag(fs, "format", "md", "md or json")
	var notes multiFlag
	fs.Var(&notes, "note", "investigator note")
	if err := parseWithHelp(fs, "report", "Generate trace investigation report", "report <trace-id> [flags]", args); err != nil {
		return nil, err
	}
	traceID, _, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return nil, err
	}
	topo, _, err := runTopology(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	errs, _, err := runErrors(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	cp, _, err := runCriticalPath(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	var details *SpanDetails
	if len(errs.Spans) > 0 {
		spanIDs := make([]string, 0, len(errs.Spans))
		for _, span := range errs.Spans {
			spanIDs = append(spanIDs, span.SpanID)
		}
		var detailsErr error
		details, _, detailsErr = runDetails(ctx, c, append([]string{traceID}, spanIDs...))
		if detailsErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch error span details: %v\n", detailsErr)
		}
	}
	reportTitle := *title
	if reportTitle == "" && len(topo.Spans) > 0 {
		reportTitle = topo.Spans[0].SpanName
	}
	if reportTitle == "" {
		reportTitle = "Trace " + traceID
	}
	report := &Report{
		Title:        reportTitle,
		TraceID:      traceID,
		Generated:    time.Now().UTC().Format(time.RFC3339),
		Endpoint:     c.Endpoint(),
		ToolVersion:  build.Version,
		Topology:     topo,
		Errors:       errs,
		CriticalPath: cp,
		ErrorDetails: details,
		Notes:        notes,
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s_%s.%s", time.Now().UTC().Format("2006-01-02T15-04-05Z"), output.Truncate(traceID, 12), *format)
	path := filepath.Join(*dir, name)
	switch *format {
	case "json":
		if err := output.SaveJSON(path, report); err != nil {
			return nil, err
		}
	case "md":
		if err := os.WriteFile(path, []byte(renderMarkdownReport(report)), 0o644); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid report format %q", *format)
	}
	fmt.Println(path)
	return nil, nil
}

func runSnapshot(ctx context.Context, c *client.Client, args []string, opts globalOptions) (*Snapshot, error) {
	fs := newFlagSet("snapshot")
	label := stringFlag(fs, "label", "", "snapshot label")
	errorsOnly := boolFlag(fs, "errors", false, "only error traces")
	since := stringFlag(fs, "since", "1h", "lookback duration")
	depth := intFlag(fs, "depth", 20, "number of traces")
	dir := stringFlag(fs, "dir", ".jmcp/snapshots", "snapshot directory")
	if err := parseWithHelp(fs, "snapshot", "Capture service trace snapshot", "snapshot <service> [flags]", args); err != nil {
		return nil, err
	}
	pos := fs.Args()
	if len(pos) == 0 {
		return nil, errors.New("service name required")
	}
	service := pos[0]
	searchArgs := []string{service, "--since", *since, "--depth", fmt.Sprintf("%d", *depth)}
	if *errorsOnly {
		searchArgs = append(searchArgs, "--errors")
	}
	result, _, err := runSearch(ctx, c, searchArgs)
	if err != nil {
		return nil, err
	}
	snapLabel := *label
	if snapLabel == "" {
		snapLabel = time.Now().UTC().Format("2006-01-02T15-04-05Z")
	}
	snap := &Snapshot{
		Label:      snapLabel,
		Service:    service,
		CapturedAt: time.Now().UTC().Format(time.RFC3339),
		Endpoint:   opts.Endpoint,
		Query: map[string]any{
			"service_name": service,
			"with_errors":  *errorsOnly,
			"search_depth": *depth,
			"since":        *since,
		},
		Traces: result.Traces,
		Stats:  analysis.ComputeStats(result.Traces),
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil {
		return nil, err
	}
	if err := output.SaveJSON(filepath.Join(*dir, snapLabel+".json"), snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func runDiff(args []string, clr *output.Colorizer) (any, error) {
	fs := newFlagSet("diff")
	if err := parseWithHelp(fs, "diff", "Compare two snapshots", "diff <snapshot-a> <snapshot-b>", args); err != nil {
		return nil, err
	}
	pos := fs.Args()
	if len(pos) != 2 {
		return nil, errors.New("usage: jmcp diff <snapshot-a> <snapshot-b>")
	}
	a, err := readSnapshot(pos[0])
	if err != nil {
		return nil, err
	}
	b, err := readSnapshot(pos[1])
	if err != nil {
		return nil, err
	}
	fmt.Printf("Comparing: %s -> %s\n\n", a.Label, b.Label)
	rows := [][]string{
		{"Total traces", fmt.Sprintf("%d", a.Stats.TotalTraces), fmt.Sprintf("%d", b.Stats.TotalTraces), change(float64(a.Stats.TotalTraces), float64(b.Stats.TotalTraces), false)},
		{"Error traces", fmt.Sprintf("%d", a.Stats.ErrorTraces), fmt.Sprintf("%d", b.Stats.ErrorTraces), change(float64(a.Stats.ErrorTraces), float64(b.Stats.ErrorTraces), true)},
		{"Avg duration", output.FormatDurationUS(a.Stats.AvgDuration), output.FormatDurationUS(b.Stats.AvgDuration), change(float64(a.Stats.AvgDuration), float64(b.Stats.AvgDuration), true)},
		{"P95 duration", output.FormatDurationUS(a.Stats.P95Duration), output.FormatDurationUS(b.Stats.P95Duration), change(float64(a.Stats.P95Duration), float64(b.Stats.P95Duration), true)},
		{"P99 duration", output.FormatDurationUS(a.Stats.P99Duration), output.FormatDurationUS(b.Stats.P99Duration), change(float64(a.Stats.P99Duration), float64(b.Stats.P99Duration), true)},
	}
	output.TableRows(os.Stdout, []string{"METRIC", "BEFORE", "AFTER", "CHANGE"}, rows)
	return nil, nil
}

func runBlame(ctx context.Context, c *client.Client, args []string, opts globalOptions, clr *output.Colorizer) (any, error) {
	traceID, _, err := resolveTraceID(ctx, c, args)
	if err != nil {
		return nil, err
	}
	cp, _, err := runCriticalPath(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	errs, _, err := runErrors(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	errorBySpan := map[string]FullSpan{}
	for _, span := range errs.Spans {
		errorBySpan[span.SpanID] = span
	}
	var suspect CriticalPathSegment
	reason := "highest self-time on critical path"
	for _, seg := range cp.Segments {
		if _, ok := errorBySpan[seg.SpanID]; ok {
			suspect = seg
			reason = "on critical path and has error status"
			break
		}
		if suspect.SelfTimeUS == 0 || seg.SelfTimeUS > suspect.SelfTimeUS {
			suspect = seg
		}
	}
	result := map[string]any{"trace_id": traceID, "primary_suspect": suspect, "reason": reason}
	if opts.Output == string(output.JSON) {
		return result, nil
	}
	fmt.Printf("Trace: %s\n\n", traceID)
	fmt.Println("Root Cause Analysis:")
	fmt.Printf("  Primary suspect: %s (%s)\n", clr.Cyan(suspect.Service), suspect.SpanName)
	fmt.Printf("  Reason: %s\n", reason)
	fmt.Printf("  Self-time: %s of %s total\n", output.FormatDurationUS(suspect.SelfTimeUS), output.FormatDurationUS(cp.TotalDurationUS))
	if errSpan, ok := errorBySpan[suspect.SpanID]; ok && errSpan.Status.Message != "" {
		fmt.Printf("  Error: %s\n", clr.Red(errSpan.Status.Message))
	}
	fmt.Printf("\nRecommendation: Investigate %s service's %s operation\n", clr.Cyan(suspect.Service), suspect.SpanName)
	return nil, nil
}

func runExport(ctx context.Context, c *client.Client, args []string) error {
	fs := newFlagSet("export")
	format := stringFlag(fs, "format", "json", "json, csv, dot")
	if err := parseWithHelp(fs, "export", "Export trace data", "export <trace-id> [flags]", args); err != nil {
		return err
	}
	traceID, _, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return err
	}
	topo, _, err := runTopology(ctx, c, []string{traceID})
	if err != nil {
		return err
	}
	switch *format {
	case "json":
		return output.WriteJSON(os.Stdout, topo)
	case "csv":
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"path", "service", "span_name", "start_time", "duration_us", "status", "truncated_children"})
		for _, span := range topo.Spans {
			_ = w.Write([]string{span.Path, span.Service, span.SpanName, span.StartTime, fmt.Sprintf("%d", span.DurationUS), span.Status, fmt.Sprintf("%d", span.TruncatedChildren)})
		}
		w.Flush()
		return w.Error()
	case "dot":
		fmt.Println("digraph trace {")
		fmt.Println("  rankdir=LR;")
		fmt.Println("  node [shape=box];")
		for _, span := range topo.Spans {
			parts := strings.Split(span.Path, "/")
			if len(parts) < 2 {
				continue
			}
			parent := parts[len(parts)-2]
			current := parts[len(parts)-1]
			color := ""
			if strings.EqualFold(span.Status, "error") {
				color = ", color=red"
			}
			fmt.Printf("  %q -> %q [label=%q%s];\n", parent, current, span.SpanName, color)
		}
		fmt.Println("}")
		return nil
	default:
		return fmt.Errorf("invalid export format %q", *format)
	}
}

func runReplay(ctx context.Context, c *client.Client, args []string) (any, error) {
	fs := newFlagSet("replay")
	if err := parseWithHelp(fs, "replay", "Replay investigation from report", "replay <report-file>", args); err != nil {
		return nil, err
	}
	if len(fs.Args()) != 1 {
		return nil, errors.New("usage: jmcp replay <report-file>")
	}
	raw, err := os.ReadFile(fs.Args()[0])
	if err != nil {
		return nil, err
	}
	traceID := extractTraceIDFromReport(string(raw))
	if traceID == "" {
		return nil, errors.New("could not find trace ID in report")
	}
	fmt.Printf("Original trace: %s\n\n", traceID)
	topo, _, err := runTopology(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	errs, _, err := runErrors(ctx, c, []string{traceID})
	if err != nil {
		return nil, err
	}
	fmt.Println("Checking trace still exists... YES")
	fmt.Printf("Re-fetching topology...\n  Span count: %d\n", len(topo.Spans))
	fmt.Printf("Re-fetching errors...\n  Error count: %d\n", errs.TotalErrorCount)
	return nil, nil
}

func renderMarkdownReport(report *Report) string {
	nc := output.DisabledColorizer()
	var b strings.Builder
	fmt.Fprintf(&b, "# Trace Report: %s\n\n", report.Title)
	fmt.Fprintf(&b, "**Trace ID**: %s\n", report.TraceID)
	fmt.Fprintf(&b, "**Generated**: %s\n", report.Generated)
	fmt.Fprintf(&b, "**Endpoint**: %s\n", report.Endpoint)
	fmt.Fprintf(&b, "**Tool Version**: jmcp %s\n\n", report.ToolVersion)
	fmt.Fprintln(&b, "## Topology")
	fmt.Fprintln(&b)
	renderTopology(&b, report.Topology, nc)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Errors")
	fmt.Fprintln(&b)
	renderErrors(&b, report.Errors, nc)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Critical Path")
	fmt.Fprintln(&b)
	renderCriticalPath(&b, report.CriticalPath, nc)
	if report.ErrorDetails != nil {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## Error Span Details")
		fmt.Fprintln(&b)
		renderDetails(&b, report.ErrorDetails, nc)
	}
	if len(report.Notes) > 0 {
		fmt.Fprintln(&b)
		fmt.Fprintln(&b, "## Notes")
		fmt.Fprintln(&b)
		for _, note := range report.Notes {
			fmt.Fprintf(&b, "- %s\n", note)
		}
	}
	return b.String()
}

func readSnapshot(ref string) (*Snapshot, error) {
	candidates := []string{ref}
	if !strings.HasSuffix(ref, ".json") {
		candidates = append(candidates, filepath.Join(".jmcp", "snapshots", ref+".json"))
	}
	for _, path := range candidates {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(raw, &snap); err != nil {
			return nil, err
		}
		return &snap, nil
	}
	return nil, fmt.Errorf("snapshot not found: %s", ref)
}

func change(before, after float64, regression bool) string {
	if before == 0 {
		if after == 0 {
			return "-"
		}
		return "+inf !!"
	}
	pct := (after - before) / before * 100
	mark := ""
	if regression && pct > 50 {
		mark = " !!"
	}
	return fmt.Sprintf("%+.1f%%%s", pct, mark)
}

func extractTraceIDFromReport(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, "Trace ID") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.Trim(strings.TrimSpace(parts[len(parts)-1]), "*` ")
			}
		}
	}
	return ""
}

func sortedErrorServices(traces []TraceSummary) []string {
	seen := map[string]struct{}{}
	for _, trace := range traces {
		if trace.HasErrors {
			for _, service := range trace.Services {
				seen[service] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for service := range seen {
		out = append(out, service)
	}
	sort.Strings(out)
	return out
}
