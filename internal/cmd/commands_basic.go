package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"jmcp/internal/client"
	"jmcp/internal/output"
)

func runHealth(ctx context.Context, c *client.Client, args []string) (*Health, json.RawMessage, error) {
	fs := newFlagSet("health")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	return callAndDecode[Health](ctx, c, "health", map[string]any{})
}

func runServices(ctx context.Context, c *client.Client, args []string) (*Services, json.RawMessage, error) {
	fs := newFlagSet("services")
	pattern := stringFlag(fs, "pattern", "", "regex filter")
	limit := intFlag(fs, "limit", 0, "max results")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	payload := map[string]any{}
	if *pattern != "" {
		payload["pattern"] = *pattern
	}
	if *limit > 0 {
		payload["limit"] = *limit
	}
	return callAndDecode[Services](ctx, c, "get_services", payload)
}

func runSpans(ctx context.Context, c *client.Client, args []string) (*SpanNames, json.RawMessage, error) {
	fs := newFlagSet("spans")
	kind := stringFlag(fs, "kind", "", "span kind")
	pattern := stringFlag(fs, "pattern", "", "regex filter")
	limit := intFlag(fs, "limit", 0, "max results")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	pos := fs.Args()
	service := ""
	if len(pos) > 0 {
		service = pos[0]
	} else {
		selected, err := interactiveServicePicker(ctx, c)
		if err != nil {
			return nil, nil, err
		}
		service = selected
	}
	payload := map[string]any{"service_name": service}
	if *kind != "" {
		payload["span_kind"] = *kind
	}
	if *pattern != "" {
		payload["pattern"] = *pattern
	}
	if *limit > 0 {
		payload["limit"] = *limit
	}
	return callAndDecode[SpanNames](ctx, c, "get_span_names", payload)
}

func runSearch(ctx context.Context, c *client.Client, args []string) (*SearchResult, json.RawMessage, error) {
	fs := newFlagSet("search")
	span := stringFlag(fs, "span", "", "span name")
	since := stringFlag(fs, "since", "1h", "lookback duration")
	until := stringFlag(fs, "until", "now", "end time")
	errorsOnly := boolFlag(fs, "errors", false, "only traces with errors")
	depth := intFlag(fs, "depth", 10, "search depth")
	minDuration := stringFlag(fs, "min-duration", "", "minimum duration")
	maxDuration := stringFlag(fs, "max-duration", "", "maximum duration")
	var attrs multiFlag
	fs.Var(&attrs, "attr", "attribute KEY=VALUE")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	pos := fs.Args()
	service := ""
	if len(pos) > 0 {
		service = pos[0]
	} else {
		selected, err := interactiveServicePicker(ctx, c)
		if err != nil {
			return nil, nil, err
		}
		service = selected
	}
	parsedAttrs, err := parseAttrs(attrs)
	if err != nil {
		return nil, nil, err
	}
	payload := map[string]any{
		"service_name":   service,
		"start_time_min": "-" + strings.TrimPrefix(*since, "-"),
		"start_time_max": *until,
		"with_errors":    *errorsOnly,
		"search_depth":   *depth,
	}
	if *span != "" {
		payload["span_name"] = *span
	}
	if *minDuration != "" {
		payload["duration_min"] = *minDuration
	}
	if *maxDuration != "" {
		payload["duration_max"] = *maxDuration
	}
	if len(parsedAttrs) > 0 {
		payload["attributes"] = parsedAttrs
	}
	result, raw, err := callAndDecode[SearchResult](ctx, c, "search_traces", payload)
	if err != nil {
		return nil, raw, err
	}
	_ = saveSearchCache(result)
	return result, raw, nil
}

func runTopology(ctx context.Context, c *client.Client, args []string) (*Topology, json.RawMessage, error) {
	fs := newFlagSet("topology")
	depth := intFlag(fs, "depth", 0, "max tree depth")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	traceID, _, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return nil, nil, err
	}
	payload := map[string]any{"trace_id": traceID}
	if *depth > 0 {
		payload["depth"] = *depth
	}
	return callAndDecode[Topology](ctx, c, "get_trace_topology", payload)
}

func runErrors(ctx context.Context, c *client.Client, args []string) (*TraceErrors, json.RawMessage, error) {
	fs := newFlagSet("errors")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	traceID, _, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return nil, nil, err
	}
	return callAndDecode[TraceErrors](ctx, c, "get_trace_errors", map[string]any{"trace_id": traceID})
}

func runDetails(ctx context.Context, c *client.Client, args []string) (*SpanDetails, json.RawMessage, error) {
	fs := newFlagSet("details")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	traceID, rest, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return nil, nil, err
	}
	if len(rest) == 0 {
		return nil, nil, errors.New("span IDs required")
	}
	return callAndDecode[SpanDetails](ctx, c, "get_span_details", map[string]any{"trace_id": traceID, "span_ids": rest})
}

func runCriticalPath(ctx context.Context, c *client.Client, args []string) (*CriticalPath, json.RawMessage, error) {
	fs := newFlagSet("critical-path")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	traceID, _, err := resolveTraceID(ctx, c, fs.Args())
	if err != nil {
		return nil, nil, err
	}
	return callAndDecode[CriticalPath](ctx, c, "get_critical_path", map[string]any{"trace_id": traceID})
}

func runDeps(ctx context.Context, c *client.Client, args []string) (*Dependencies, json.RawMessage, error) {
	fs := newFlagSet("deps")
	since := stringFlag(fs, "since", "", "start time")
	until := stringFlag(fs, "until", "", "end time")
	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	payload := map[string]any{}
	if *since != "" {
		payload["start_time"] = *since
	}
	if *until != "" {
		payload["end_time"] = *until
	}
	return callAndDecode[Dependencies](ctx, c, "get_service_dependencies", payload)
}

func interactiveServicePicker(ctx context.Context, c *client.Client) (string, error) {
	if !isTerminal() {
		return "", errors.New("service name required when stdin is not a TTY")
	}
	services, _, err := callAndDecode[Services](ctx, c, "get_services", map[string]any{})
	if err != nil {
		return "", err
	}
	if len(services.Services) == 0 {
		return "", errors.New("no services returned by Jaeger")
	}
	sort.Strings(services.Services)
	for i, service := range services.Services {
		fmt.Fprintf(os.Stderr, "%d. %s\n", i+1, service)
	}
	fmt.Fprint(os.Stderr, "Select service: ")
	text, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	text = strings.TrimSpace(text)
	if n, err := strconvAtoi(text); err == nil && n >= 1 && n <= len(services.Services) {
		return services.Services[n-1], nil
	}
	for _, service := range services.Services {
		if service == text {
			return service, nil
		}
	}
	return "", fmt.Errorf("invalid service selection %q", text)
}

func interactiveTracePicker(ctx context.Context, c *client.Client) (string, error) {
	if !isTerminal() {
		return "", errors.New("trace ID required when stdin is not a TTY")
	}
	service, err := interactiveServicePicker(ctx, c)
	if err != nil {
		return "", err
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "Errors only? [y/N]: ")
	ans, _ := reader.ReadString('\n')
	errorsOnly := strings.EqualFold(strings.TrimSpace(ans), "y")
	fmt.Fprint(os.Stderr, "Time range [1h]: ")
	since, _ := reader.ReadString('\n')
	since = strings.TrimSpace(since)
	if since == "" {
		since = "1h"
	}
	searchArgs := []string{service, "--since", since, "--depth", "20"}
	if errorsOnly {
		searchArgs = append(searchArgs, "--errors")
	}
	result, _, err := runSearch(ctx, c, searchArgs)
	if err != nil {
		return "", err
	}
	if len(result.Traces) == 0 {
		return "", errors.New("no traces found")
	}
	renderSearch(os.Stderr, result)
	fmt.Fprintf(os.Stderr, "Select [1-%d]: ", len(result.Traces))
	text, _ := reader.ReadString('\n')
	n, err := strconvAtoi(strings.TrimSpace(text))
	if err != nil || n < 1 || n > len(result.Traces) {
		return "", fmt.Errorf("invalid trace selection %q", strings.TrimSpace(text))
	}
	return result.Traces[n-1].TraceID, nil
}

func strconvAtoi(v string) (int, error) {
	return strconv.Atoi(v)
}

func renderSearch(w io.Writer, result *SearchResult) {
	now := time.Now()
	rows := make([][]string, 0, len(result.Traces))
	for i, trace := range result.Traces {
		rows = append(rows, []string{
			fmt.Sprintf("%d", i+1),
			output.Truncate(trace.TraceID, 18),
			trace.RootSpanName,
			output.FormatDurationUS(trace.DurationUS),
			fmt.Sprintf("%d", trace.SpanCount),
			fmt.Sprintf("%d", trace.ServiceCount),
			output.BoolYes(trace.HasErrors),
			output.RelativeTime(trace.StartTime, now),
		})
	}
	output.TableRows(w, []string{"#", "TRACE ID", "ROOT SPAN", "DURATION", "SPANS", "SERVICES", "ERRORS", "TIME"}, rows)
}
