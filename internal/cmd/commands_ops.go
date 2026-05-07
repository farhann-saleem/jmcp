package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"jmcp/internal/analysis"
	"jmcp/internal/client"
	"jmcp/internal/output"
)

func runWatch(ctx context.Context, c *client.Client, args []string) error {
	fs := newFlagSet("watch")
	interval := stringFlag(fs, "interval", "5s", "poll interval")
	errorsOnly := boolFlag(fs, "errors", false, "only error traces")
	alert := boolFlag(fs, "alert", false, "terminal bell on new findings")
	since := stringFlag(fs, "since", "5m", "initial lookback")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) == 0 {
		return errors.New("service name required")
	}
	service := fs.Args()[0]
	tick, err := time.ParseDuration(*interval)
	if err != nil {
		return err
	}
	seen := map[string]TraceSummary{}
	start := time.Now()
	var peak TraceSummary

	for {
		select {
		case <-ctx.Done():
			printWatchSummary(start, seen, peak)
			return nil
		default:
		}
		searchArgs := []string{service, "--since", *since, "--depth", "100"}
		if *errorsOnly {
			searchArgs = append(searchArgs, "--errors")
		}
		result, _, err := runSearch(ctx, c, searchArgs)
		if err != nil {
			return err
		}
		for _, trace := range result.Traces {
			if _, ok := seen[trace.TraceID]; ok {
				continue
			}
			seen[trace.TraceID] = trace
			if trace.DurationUS > peak.DurationUS {
				peak = trace
			}
			status := "OK"
			if trace.HasErrors {
				status = "ERROR"
				if *alert {
					fmt.Print("\a")
				}
			}
			fmt.Printf("[%s] %s | %s | %s | %d spans | %s\n",
				time.Now().Format("15:04:05"), trace.RootService, trace.RootSpanName,
				output.FormatDurationUS(trace.DurationUS), trace.SpanCount, status)
		}
		time.Sleep(tick)
	}
}

func runCheck(ctx context.Context, c *client.Client, args []string) int {
	fs := newFlagSet("check")
	maxErrorRate := intFlag(fs, "error-rate", 10, "max acceptable error percentage")
	p95Limit := stringFlag(fs, "p95", "1s", "max p95 latency")
	since := stringFlag(fs, "since", "5m", "lookback window")
	depth := intFlag(fs, "depth", 50, "sample size")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 3
	}
	if len(fs.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Error: service name required")
		return 3
	}
	service := fs.Args()[0]
	result, _, err := runSearch(ctx, c, []string{service, "--since", *since, "--depth", fmt.Sprintf("%d", *depth)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 1
	}
	stats := analysis.ComputeStats(result.Traces)
	limitUS, err := output.ParseDurationToUS(*p95Limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 3
	}
	errorRate := 0.0
	if stats.TotalTraces > 0 {
		errorRate = float64(stats.ErrorTraces) / float64(stats.TotalTraces) * 100
	}
	errorPass := errorRate <= float64(*maxErrorRate)
	p95Pass := stats.P95Duration <= limitUS

	fmt.Printf("Service: %s (last %s, %d traces sampled)\n\n", service, *since, stats.TotalTraces)
	output.TableRows(os.Stdout, []string{"CHECK", "THRESHOLD", "ACTUAL", "STATUS"}, [][]string{
		{"Error rate", fmt.Sprintf("< %d%%", *maxErrorRate), fmt.Sprintf("%.1f%%", errorRate), passFail(errorPass)},
		{"P95 latency", "< " + *p95Limit, output.FormatDurationUS(stats.P95Duration), passFail(p95Pass)},
	})
	if errorPass && p95Pass {
		fmt.Println("\nResult: HEALTHY")
		return 0
	}
	fmt.Println("\nResult: UNHEALTHY")
	return 1
}

func printWatchSummary(start time.Time, seen map[string]TraceSummary, peak TraceSummary) {
	traces := make([]TraceSummary, 0, len(seen))
	for _, trace := range seen {
		traces = append(traces, trace)
	}
	stats := analysis.ComputeStats(traces)
	fmt.Printf("\nWatch summary (%s):\n", time.Since(start).Round(time.Second))
	fmt.Printf("  Traces seen: %d\n", stats.TotalTraces)
	if stats.TotalTraces > 0 {
		fmt.Printf("  Errors: %d (%.1f%%)\n", stats.ErrorTraces, float64(stats.ErrorTraces)/float64(stats.TotalTraces)*100)
	}
	fmt.Printf("  Avg duration: %s\n", output.FormatDurationUS(stats.AvgDuration))
	if peak.TraceID != "" {
		fmt.Printf("  Peak duration: %s at %s\n", output.FormatDurationUS(peak.DurationUS), peak.StartTime)
	}
}

func passFail(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}
