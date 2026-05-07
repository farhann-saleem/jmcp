package cmd

import (
	"bufio"
	"os"
	"strings"
)

type projectConfig struct {
	Endpoint    string
	SearchDepth int
	Since       string
	Output      string
}

func loadProjectConfig() projectConfig {
	var cfg projectConfig
	f, err := os.Open(".jmcp/config.yaml")
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// skip section headers like "defaults:"
		if strings.HasSuffix(line, ":") && !strings.Contains(line[:len(line)-1], ":") {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "endpoint":
			cfg.Endpoint = value
		case "search_depth":
			n := 0
			for _, ch := range value {
				if ch >= '0' && ch <= '9' {
					n = n*10 + int(ch-'0')
				} else {
					break
				}
			}
			if n > 0 {
				cfg.SearchDepth = n
			}
		case "since":
			cfg.Since = value
		case "output":
			cfg.Output = value
		}
	}
	return cfg
}
