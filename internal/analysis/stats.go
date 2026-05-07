package analysis

import (
	"math"
	"sort"
)

type TraceLike interface {
	Duration() int64
	HasError() bool
	ServicesList() []string
}

type Stats struct {
	TotalTraces  int      `json:"total_traces"`
	ErrorTraces  int      `json:"error_traces"`
	AvgDuration  int64    `json:"avg_duration_us"`
	P50Duration  int64    `json:"p50_duration_us"`
	P95Duration  int64    `json:"p95_duration_us"`
	P99Duration  int64    `json:"p99_duration_us"`
	ServicesSeen []string `json:"services_seen"`
}

func ComputeStats[T TraceLike](traces []T) Stats {
	stats := Stats{TotalTraces: len(traces)}
	if len(traces) == 0 {
		return stats
	}
	durations := make([]int64, 0, len(traces))
	services := map[string]struct{}{}
	var sum int64
	for _, trace := range traces {
		d := trace.Duration()
		durations = append(durations, d)
		sum += d
		if trace.HasError() {
			stats.ErrorTraces++
		}
		for _, service := range trace.ServicesList() {
			services[service] = struct{}{}
		}
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	stats.AvgDuration = sum / int64(len(traces))
	stats.P50Duration = percentile(durations, 0.50)
	stats.P95Duration = percentile(durations, 0.95)
	stats.P99Duration = percentile(durations, 0.99)
	stats.ServicesSeen = make([]string, 0, len(services))
	for service := range services {
		stats.ServicesSeen = append(stats.ServicesSeen, service)
	}
	sort.Strings(stats.ServicesSeen)
	return stats
}

func percentile(sorted []int64, p float64) int64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(len(sorted))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
