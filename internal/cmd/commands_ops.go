package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/farhann-saleem/jmcp/internal/analysis"
	"github.com/farhann-saleem/jmcp/internal/client"
	"github.com/farhann-saleem/jmcp/internal/output"
)

const seenTrackerMax = 10000

type seenTracker struct {
	m     map[string]TraceSummary
	order []string
}

func newSeenTracker() *seenTracker {
	return &seenTracker{m: make(map[string]TraceSummary)}
}

func (s *seenTracker) has(id string) bool {
	_, ok := s.m[id]
	return ok
}

func (s *seenTracker) add(trace TraceSummary) {
	if len(s.m) >= seenTrackerMax {
		evict := len(s.order) / 10
		if evict < 1 {
			evict = 1
		}
		for _, id := range s.order[:evict] {
			delete(s.m, id)
		}
		s.order = s.order[evict:]
	}
	s.m[trace.TraceID] = trace
	s.order = append(s.order, trace.TraceID)
}

func (s *seenTracker) all() []TraceSummary {
	out := make([]TraceSummary, 0, len(s.m))
	for _, t := range s.m {
		out = append(out, t)
	}
	return out
}

func (s *seenTracker) len() int { return len(s.m) }

func runWatch(ctx context.Context, c *client.Client, args []string, clr *output.Colorizer) error {
	fs := newFlagSet("watch")
	interval := stringFlag(fs, "interval", "5s", "poll interval")
	errorsOnly := boolFlag(fs, "errors", false, "only error traces")
	alert := boolFlag(fs, "alert", false, "terminal bell on new findings")
	since := stringFlag(fs, "since", "5m", "initial lookback")
	if err := parseWithHelp(fs, "watch", "Watch for new traces", "watch <service> [flags]", args); err != nil {
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
	seen := newSeenTracker()
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
			if seen.has(trace.TraceID) {
				continue
			}
			seen.add(trace)
			if trace.DurationUS > peak.DurationUS {
				peak = trace
			}
			status := clr.Green("OK")
			if trace.HasErrors {
				status = clr.Red("ERROR")
				if *alert {
					fmt.Print("\a")
				}
			}
			fmt.Printf("[%s] %s | %s | %s | %d spans | %s\n",
				time.Now().Format("15:04:05"), clr.Cyan(trace.RootService), trace.RootSpanName,
				output.FormatDurationUS(trace.DurationUS), trace.SpanCount, status)
		}
		time.Sleep(tick)
	}
}

func runCheck(ctx context.Context, c *client.Client, args []string, clr *output.Colorizer) int {
	fs := newFlagSet("check")
	maxErrorRate := intFlag(fs, "error-rate", 10, "max acceptable error percentage")
	p95Limit := stringFlag(fs, "p95", "1s", "max p95 latency")
	since := stringFlag(fs, "since", "5m", "lookback window")
	depth := intFlag(fs, "depth", 50, "sample size")
	if err := parseWithHelp(fs, "check", "Health gate check for CI/CD", "check <service> [flags]", args); err != nil {
		if errors.Is(err, errHelp) {
			return 0
		}
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
		{"Error rate", fmt.Sprintf("< %d%%", *maxErrorRate), fmt.Sprintf("%.1f%%", errorRate), clr.Status(errorPass, passFail(errorPass))},
		{"P95 latency", "< " + *p95Limit, output.FormatDurationUS(stats.P95Duration), clr.Status(p95Pass, passFail(p95Pass))},
	})
	if errorPass && p95Pass {
		fmt.Println("\nResult: " + clr.Green("HEALTHY"))
		return 0
	}
	fmt.Println("\nResult: " + clr.Red("UNHEALTHY"))
	return 1
}

func printWatchSummary(start time.Time, seen *seenTracker, peak TraceSummary) {
	stats := analysis.ComputeStats(seen.all())
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
