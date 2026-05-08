package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/farhann-saleem/jmcp/internal/client"
	"github.com/farhann-saleem/jmcp/internal/output"
)

const defaultEndpoint = "http://localhost:16687/mcp"

var errHelp = errors.New("help requested")

func Execute(args []string, build BuildInfo) int {
	opts, remaining, err := parseGlobal(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		return 3
	}
	if len(remaining) == 0 || remaining[0] == "help" || remaining[0] == "--help" || remaining[0] == "-h" {
		printUsage(os.Stdout, build)
		return 0
	}
	if remaining[0] == "--version" || remaining[0] == "version" {
		fmt.Printf("jmcp %s (commit: %s, built: %s)\n", build.Version, build.Commit, build.Date)
		return 0
	}

	colorizer := output.NewColorizer(opts.NoColor)

	c := client.New(opts.Endpoint,
		client.WithTimeout(opts.Timeout),
		client.WithVerbose(opts.Verbose, func(format string, values ...any) {
			fmt.Fprintf(os.Stderr, format, values...)
		}),
	)

	cmd := remaining[0]
	cmdArgs := remaining[1:]
	var ctx context.Context
	var cancel context.CancelFunc
	if cmd == "watch" {
		ctx, cancel = signal.NotifyContext(context.Background(), os.Interrupt)
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
	}
	defer cancel()

	var data any
	var raw json.RawMessage
	err = nil

	switch cmd {
	case "health":
		data, raw, err = runHealth(ctx, c, cmdArgs)
	case "services":
		data, raw, err = runServices(ctx, c, cmdArgs)
	case "spans":
		data, raw, err = runSpans(ctx, c, cmdArgs)
	case "search":
		data, raw, err = runSearch(ctx, c, cmdArgs)
	case "topology":
		data, raw, err = runTopology(ctx, c, cmdArgs)
	case "errors":
		data, raw, err = runErrors(ctx, c, cmdArgs)
	case "details":
		data, raw, err = runDetails(ctx, c, cmdArgs)
	case "critical-path":
		data, raw, err = runCriticalPath(ctx, c, cmdArgs)
	case "deps":
		data, raw, err = runDeps(ctx, c, cmdArgs)
	case "investigate":
		data, err = runInvestigate(ctx, c, cmdArgs, opts, colorizer)
	case "init":
		data, err = runInit(cmdArgs)
	case "report":
		data, err = runReport(ctx, c, cmdArgs, opts, build)
	case "snapshot":
		data, err = runSnapshot(ctx, c, cmdArgs, opts)
	case "diff":
		data, err = runDiff(cmdArgs, colorizer)
	case "watch":
		err = runWatch(ctx, c, cmdArgs, colorizer)
	case "blame":
		data, err = runBlame(ctx, c, cmdArgs, opts, colorizer)
	case "export":
		err = runExport(ctx, c, cmdArgs)
	case "check":
		return runCheck(ctx, c, cmdArgs, colorizer)
	case "replay":
		data, err = runReplay(ctx, c, cmdArgs)
	case "completion":
		err = runCompletion(cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %q\n\n", cmd)
		printUsage(os.Stderr, build)
		return 3
	}
	if err != nil {
		if errors.Is(err, errHelp) {
			return 0
		}
		fmt.Fprintln(os.Stderr, "Error:", err)
		if strings.Contains(err.Error(), "cannot connect") {
			return 2
		}
		return 1
	}

	if opts.Save != "" && data != nil {
		if err := output.SaveJSON(opts.Save, data); err != nil {
			fmt.Fprintln(os.Stderr, "Error: save output:", err)
			return 1
		}
	}
	if data != nil {
		if err := render(os.Stdout, opts.Output, data, raw, colorizer); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			return 1
		}
	}
	return 0
}

var projConfig projectConfig

func parseGlobal(args []string) (globalOptions, []string, error) {
	projConfig = loadProjectConfig()
	endpointDefault := defaultEndpoint
	if projConfig.Endpoint != "" {
		endpointDefault = projConfig.Endpoint
	}
	outputDefault := string(output.Table)
	if projConfig.Output != "" {
		outputDefault = projConfig.Output
	}
	opts := globalOptions{
		Endpoint: envDefault("JMCP_ENDPOINT", endpointDefault),
		Output:   envDefault("JMCP_OUTPUT", outputDefault),
		Timeout:  30 * time.Second,
		NoColor:  os.Getenv("NO_COLOR") != "",
	}

	remaining := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		readValue := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			i++
			return args[i], nil
		}
		switch {
		case arg == "--endpoint" || arg == "-e":
			v, err := readValue()
			if err != nil {
				return opts, nil, err
			}
			opts.Endpoint = v
		case strings.HasPrefix(arg, "--endpoint="):
			opts.Endpoint = strings.TrimPrefix(arg, "--endpoint=")
		case arg == "--output" || arg == "-o":
			v, err := readValue()
			if err != nil {
				return opts, nil, err
			}
			opts.Output = v
		case strings.HasPrefix(arg, "--output="):
			opts.Output = strings.TrimPrefix(arg, "--output=")
		case arg == "--save" || arg == "-s":
			v, err := readValue()
			if err != nil {
				return opts, nil, err
			}
			opts.Save = v
		case strings.HasPrefix(arg, "--save="):
			opts.Save = strings.TrimPrefix(arg, "--save=")
		case arg == "--timeout" || arg == "-t":
			v, err := readValue()
			if err != nil {
				return opts, nil, err
			}
			d, err := time.ParseDuration(v)
			if err != nil {
				return opts, nil, err
			}
			opts.Timeout = d
		case strings.HasPrefix(arg, "--timeout="):
			d, err := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
			if err != nil {
				return opts, nil, err
			}
			opts.Timeout = d
		case arg == "--no-color":
			opts.NoColor = true
		case arg == "--verbose" || arg == "-v":
			opts.Verbose = true
		default:
			remaining = append(remaining, arg)
		}
	}
	if opts.Output != string(output.Table) && opts.Output != string(output.JSON) && opts.Output != string(output.Raw) {
		return opts, nil, fmt.Errorf("invalid output %q", opts.Output)
	}
	return opts, remaining, nil
}

func render(w io.Writer, format string, data any, raw json.RawMessage, c *output.Colorizer) error {
	switch output.Format(format) {
	case output.JSON:
		return output.WriteJSON(w, data)
	case output.Raw:
		if len(raw) == 0 {
			return output.WriteJSON(w, data)
		}
		_, err := fmt.Fprintln(w, string(raw))
		return err
	default:
		switch v := data.(type) {
		case *Health:
			output.KeyValues(w, [][2]string{{"Status", v.Status}, {"Server", v.Server}, {"Version", v.Version}})
		case *Services:
			rows := make([][]string, 0, len(v.Services))
			for _, service := range v.Services {
				rows = append(rows, []string{c.Cyan(service)})
			}
			output.TableRows(w, []string{"SERVICE"}, rows)
		case *SpanNames:
			rows := make([][]string, 0, len(v.SpanNames))
			for _, span := range v.SpanNames {
				rows = append(rows, []string{span.Name, span.Kind})
			}
			output.TableRows(w, []string{"SPAN NAME", "KIND"}, rows)
		case *SearchResult:
			renderSearch(w, v, c)
		case *Topology:
			renderTopology(w, v, c)
		case *TraceErrors:
			renderErrors(w, v, c)
		case *SpanDetails:
			renderDetails(w, v, c)
		case *CriticalPath:
			renderCriticalPath(w, v, c)
		case *Dependencies:
			renderDeps(w, v, c)
		case *Snapshot:
			renderSnapshot(w, v, c)
		default:
			return output.WriteJSON(w, data)
		}
	}
	return nil
}

func callAndDecode[T any](ctx context.Context, c *client.Client, tool string, args any) (*T, json.RawMessage, error) {
	result, err := c.CallTool(ctx, tool, args)
	if err != nil {
		return nil, nil, err
	}
	var data T
	if err := json.Unmarshal(result.Content, &data); err != nil {
		return nil, result.RawJSONRPC, fmt.Errorf("decode %s response: %w", tool, err)
	}
	return &data, result.RawJSONRPC, nil
}

func stringFlag(fs *flag.FlagSet, name, def, usage string) *string {
	return fs.String(name, def, usage)
}

func boolFlag(fs *flag.FlagSet, name string, def bool, usage string) *bool {
	return fs.Bool(name, def, usage)
}

func intFlag(fs *flag.FlagSet, name string, def int, usage string) *int {
	return fs.Int(name, def, usage)
}

func parseAttrs(values multiFlag) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	attrs := map[string]string{}
	for _, item := range values {
		key, value, ok := strings.Cut(item, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --attr %q, expected KEY=VALUE", item)
		}
		attrs[key] = value
	}
	return attrs, nil
}

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parseWithHelp(fs *flag.FlagSet, name, summary, usage string, args []string) error {
	err := fs.Parse(args)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "\n%s - %s\n\nUsage:\n  jmcp [global flags] %s\n\nFlags:\n", name, summary, usage)
		fs.SetOutput(os.Stderr)
		fs.PrintDefaults()
		return errHelp
	}
	return err
}

func envDefault(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func printUsage(w io.Writer, build BuildInfo) {
	fmt.Fprintf(w, `jmcp %s - Jaeger MCP debugging CLI

Usage:
  jmcp [global flags] <command> [args] [flags]

Global flags:
  -e, --endpoint <url>     MCP endpoint (default %s)
  -o, --output <format>    table, json, raw (default table)
  -s, --save <file>        Save parsed JSON output to file
  -t, --timeout <dur>      Request timeout (default 30s)
  -v, --verbose            Log protocol details to stderr
      --no-color           Disable color output

Commands:
  health, services, spans, search, topology, errors, details, critical-path, deps
  investigate, report, snapshot, diff, watch, blame, export, check, init, replay
`, build.Version, defaultEndpoint)
}

func resolveTraceID(ctx context.Context, c *client.Client, args []string) (string, []string, error) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "@") {
		return args[0], args[1:], nil
	}
	if len(args) > 0 && strings.HasPrefix(args[0], "@") {
		traceID, err := resolveFromCache(args[0])
		return traceID, args[1:], err
	}
	traceID, err := interactiveTracePicker(ctx, c)
	return traceID, args, err
}

func cachePath() string {
	if _, err := os.Stat(".jmcp"); err == nil {
		return filepath.Join(".jmcp", ".last_search.json")
	}
	return filepath.Join(os.TempDir(), "jmcp_last_search.json")
}

func saveSearchCache(result *SearchResult) error {
	path := cachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return output.SaveJSON(path, result)
}

func readSearchCache() (*SearchResult, error) {
	raw, err := os.ReadFile(cachePath())
	if err != nil {
		return nil, errors.New("no cached search results; run `jmcp search <service>` first")
	}
	var result SearchResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func resolveFromCache(ref string) (string, error) {
	result, err := readSearchCache()
	if err != nil {
		return "", err
	}
	idx := 1
	if ref != "@last" {
		n, err := strconv.Atoi(strings.TrimPrefix(ref, "@"))
		if err != nil || n < 1 {
			return "", fmt.Errorf("invalid trace reference %q", ref)
		}
		idx = n
	}
	if idx > len(result.Traces) {
		return "", fmt.Errorf("trace reference %s out of range; cached results contain %d traces", ref, len(result.Traces))
	}
	return result.Traces[idx-1].TraceID, nil
}

func isTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) != 0
}
