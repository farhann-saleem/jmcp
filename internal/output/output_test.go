package output

import (
	"testing"
	"time"
)

func TestFormatDurationUS(t *testing.T) {
	tests := []struct {
		us   int64
		want string
	}{
		{0, "0us"},
		{999, "999us"},
		{1000, "1.0ms"},
		{999999, "1000.0ms"},
		{1000000, "1.0s"},
		{60000000, "1m00s"},
		{-1, "-"},
	}
	for _, tt := range tests {
		got := FormatDurationUS(tt.us)
		if got != tt.want {
			t.Errorf("FormatDurationUS(%d) = %q, want %q", tt.us, got, tt.want)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"empty", "", "-"},
		{"invalid", "not-a-time", "not-a-time"},
		{"past", now.Add(-5 * time.Minute).Format(time.RFC3339Nano), "5m ago"},
		{"future", now.Add(2 * time.Hour).Format(time.RFC3339Nano), "in 2h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RelativeTime(tt.value, now)
			if got != tt.want {
				t.Errorf("RelativeTime(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"short", 10, "short"},
		{"exact", 5, "exact"},
		{"longstring", 7, "long..."},
		{"ab", 1, "a"},
		{"hello", 0, "hello"},
	}
	for _, tt := range tests {
		got := Truncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestBoolYes(t *testing.T) {
	if BoolYes(true) != "yes" {
		t.Error("BoolYes(true) != yes")
	}
	if BoolYes(false) != "no" {
		t.Error("BoolYes(false) != no")
	}
}

func TestPercent(t *testing.T) {
	if Percent(50, 100) != "50.0%" {
		t.Errorf("Percent(50,100) = %q", Percent(50, 100))
	}
	if Percent(0, 0) != "0%" {
		t.Errorf("Percent(0,0) = %q", Percent(0, 0))
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		v    any
		want string
	}{
		{nil, ""},
		{"hello", "hello"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{true, "true"},
	}
	for _, tt := range tests {
		got := ToString(tt.v)
		if got != tt.want {
			t.Errorf("ToString(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
