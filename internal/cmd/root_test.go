package cmd

import (
	"testing"
)

func TestExecuteExitCodes(t *testing.T) {
	build := BuildInfo{Version: "test", Commit: "abc", Date: "now"}

	tests := []struct {
		name string
		args []string
		want int
	}{
		{"unknown command", []string{"nonexistent"}, 3},
		{"invalid output", []string{"--output", "xml", "health"}, 3},
		{"help", []string{"--help"}, 0},
		{"version", []string{"--version"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Execute(tt.args, build)
			if got != tt.want {
				t.Errorf("Execute(%v) = %d, want %d", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseGlobal(t *testing.T) {
	// defaults
	opts, remaining, err := parseGlobal([]string{"health"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Endpoint == "" {
		t.Error("endpoint should have default")
	}
	if opts.Output != "table" {
		t.Errorf("output = %q, want table", opts.Output)
	}
	if len(remaining) != 1 || remaining[0] != "health" {
		t.Errorf("remaining = %v", remaining)
	}

	// endpoint flag
	opts, _, err = parseGlobal([]string{"--endpoint", "http://custom:8080", "services"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Endpoint != "http://custom:8080" {
		t.Errorf("endpoint = %q", opts.Endpoint)
	}

	// output flag
	opts, _, err = parseGlobal([]string{"--output", "json", "health"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.Output != "json" {
		t.Errorf("output = %q", opts.Output)
	}

	// invalid output
	_, _, err = parseGlobal([]string{"--output", "xml", "health"})
	if err == nil {
		t.Error("expected error for invalid output")
	}
}
