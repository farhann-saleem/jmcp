package main

import (
	"os"

	"jmcp/internal/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	code := cmd.Execute(os.Args[1:], cmd.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
	if code != 0 {
		os.Exit(code)
	}
}
