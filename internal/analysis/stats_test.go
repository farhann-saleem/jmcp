package analysis

import "testing"

type statTrace struct {
	duration int64
	err      bool
	services []string
}

func (s statTrace) Duration() int64        { return s.duration }
func (s statTrace) HasError() bool         { return s.err }
func (s statTrace) ServicesList() []string { return s.services }

func TestComputeStats(t *testing.T) {
	stats := ComputeStats([]statTrace{
		{duration: 10, services: []string{"frontend"}},
		{duration: 20, err: true, services: []string{"driver"}},
		{duration: 30, services: []string{"frontend", "route"}},
	})
	if stats.TotalTraces != 3 || stats.ErrorTraces != 1 {
		t.Fatalf("counts = %+v", stats)
	}
	if stats.AvgDuration != 20 || stats.P50Duration != 20 {
		t.Fatalf("durations = %+v", stats)
	}
	if len(stats.ServicesSeen) != 3 {
		t.Fatalf("services = %+v", stats.ServicesSeen)
	}
}

func TestPercentileSmallSample(t *testing.T) {
	tests := []struct {
		name   string
		values []int64
		p      float64
		want   int64
	}{
		{"empty", nil, 0.50, 0},
		{"single_p50", []int64{42}, 0.50, 42},
		{"single_p95", []int64{42}, 0.95, 42},
		{"single_p99", []int64{42}, 0.99, 42},
		{"two_p50", []int64{10, 20}, 0.50, 10},
		{"two_p95", []int64{10, 20}, 0.95, 20},
		{"two_p99", []int64{10, 20}, 0.99, 20},
		{"three_p50", []int64{10, 20, 30}, 0.50, 20},
		{"three_p95", []int64{10, 20, 30}, 0.95, 30},
		{"three_p99", []int64{10, 20, 30}, 0.99, 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.values, tt.p)
			if got != tt.want {
				t.Errorf("percentile(%v, %.2f) = %d, want %d", tt.values, tt.p, got, tt.want)
			}
		})
	}
}

func TestPercentile100Items(t *testing.T) {
	values := make([]int64, 100)
	for i := range values {
		values[i] = int64(i + 1) // 1..100
	}
	p50 := percentile(values, 0.50)
	if p50 != 50 {
		t.Errorf("p50 = %d, want 50", p50)
	}
	p95 := percentile(values, 0.95)
	if p95 != 95 {
		t.Errorf("p95 = %d, want 95", p95)
	}
	p99 := percentile(values, 0.99)
	if p99 != 99 {
		t.Errorf("p99 = %d, want 99", p99)
	}
}

func TestComputeStatsEmpty(t *testing.T) {
	stats := ComputeStats([]statTrace{})
	if stats.TotalTraces != 0 || stats.P50Duration != 0 {
		t.Fatalf("empty stats = %+v", stats)
	}
}
