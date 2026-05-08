package main

import (
	"fmt"
	"os"

	"github.com/farhann-saleem/jmcp/internal/cmd"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "jmcp: internal error: %v\n", r)
			fmt.Fprintln(os.Stderr, "Please report this at https://github.com/farhann-saleem/jmcp/issues")
			os.Exit(1)
		}
	}()
	code := cmd.Execute(os.Args[1:], cmd.BuildInfo{
		Version: version,
		Commit:  commit,
		Date:    date,
	})
	if code != 0 {
		os.Exit(code)
	}
}
