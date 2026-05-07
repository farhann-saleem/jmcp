package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectConfig(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.MkdirAll(".jmcp", 0o755)
	os.WriteFile(filepath.Join(".jmcp", "config.yaml"), []byte(
		"endpoint: http://jaeger:16687/mcp\ndefaults:\n  search_depth: 50\n  since: 30m\n  output: json\n",
	), 0o644)

	cfg := loadProjectConfig()
	if cfg.Endpoint != "http://jaeger:16687/mcp" {
		t.Errorf("endpoint = %q", cfg.Endpoint)
	}
	if cfg.SearchDepth != 50 {
		t.Errorf("search_depth = %d", cfg.SearchDepth)
	}
	if cfg.Since != "30m" {
		t.Errorf("since = %q", cfg.Since)
	}
	if cfg.Output != "json" {
		t.Errorf("output = %q", cfg.Output)
	}
}

func TestLoadProjectConfigMissing(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	cfg := loadProjectConfig()
	if cfg.Endpoint != "" || cfg.SearchDepth != 0 || cfg.Since != "" || cfg.Output != "" {
		t.Errorf("expected zero config, got %+v", cfg)
	}
}

func TestLoadProjectConfigPartial(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	os.MkdirAll(".jmcp", 0o755)
	os.WriteFile(filepath.Join(".jmcp", "config.yaml"), []byte(
		"endpoint: http://custom:8080/mcp\n",
	), 0o644)

	cfg := loadProjectConfig()
	if cfg.Endpoint != "http://custom:8080/mcp" {
		t.Errorf("endpoint = %q", cfg.Endpoint)
	}
	if cfg.SearchDepth != 0 {
		t.Errorf("search_depth should be 0, got %d", cfg.SearchDepth)
	}
}
