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
