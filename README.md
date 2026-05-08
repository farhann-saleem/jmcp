# jmcp

[![CI](https://github.com/farhann-saleem/jmcp/actions/workflows/ci.yml/badge.svg)](https://github.com/farhann-saleem/jmcp/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/farhann-saleem/jmcp)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/farhann-saleem/jmcp)](https://github.com/farhann-saleem/jmcp/releases)
[![License](https://img.shields.io/github/license/farhann-saleem/jmcp)](LICENSE)

Incident debugging CLI for Jaeger via MCP. Goes from "something broke" to documented root cause without leaving the terminal.

Single static binary. Zero external dependencies. No telemetry.

## Install

```bash
# Option 1: curl (no Go required)
curl -sSL https://raw.githubusercontent.com/farhann-saleem/jmcp/main/install.sh | bash

# Option 2: go install
go install github.com/farhann-saleem/jmcp@latest

# Option 3: build from source
git clone https://github.com/farhann-saleem/jmcp.git && cd jmcp && make build
```

## 60-Second Demo

Start Jaeger with sample traces and run your first investigation:

```bash
docker compose up -d          # starts Jaeger + tracegen
jmcp health                   # verify connection
jmcp search tracegen          # find traces
jmcp blame @1                 # who caused the problem?
```

## How It Works

jmcp talks to Jaeger's MCP endpoint (port 16687). It handles session handshakes, JSON-RPC, and SSE parsing behind simple commands.

```
You  --->  jmcp  --->  Jaeger MCP endpoint (localhost:16687/mcp)
                        |
                        +-- manages MCP sessions
                        +-- parses SSE streams
                        +-- caches search results (@1, @2, @last)
```

After running `jmcp search`, results are cached locally. Use `@1` for the first trace, `@2` for the second, `@last` for the most recent. This saves you from copy-pasting trace IDs.

## Common Workflows

### Incident Response

Something is broken. Find out what, why, and document it:

```bash
jmcp search --errors --since 15m frontend   # find failing traces
jmcp blame @1                                # identify root cause service
jmcp investigate @1                          # full breakdown (topology + errors + critical path)
jmcp report --title "Checkout 500s" @1       # generate incident report
```

### Pre/Post Deploy Comparison

Capture baseline, deploy, compare:

```bash
jmcp snapshot --label before-deploy frontend  # capture current state
# ... deploy ...
jmcp snapshot --label after-deploy frontend   # capture new state
jmcp diff .jmcp/snapshots/before-deploy.json .jmcp/snapshots/after-deploy.json
```

### CI/CD Health Gate

Block deploys when traces show problems:

```bash
jmcp check --error-rate 5 --p95 500ms --since 5m my-service
# exit 0 = healthy, exit 1 = unhealthy
```

GitHub Actions example:

```yaml
- name: Trace Health Gate
  run: jmcp check --error-rate 5 --p95 500ms --since 5m my-service
```

### Live Monitoring

Watch for new error traces in real time:

```bash
jmcp watch --errors --alert frontend    # terminal bell on new errors
```

### Export for Postmortems

```bash
jmcp export --format csv @1     # spreadsheet-friendly
jmcp export --format json @1    # machine-readable
jmcp export --format dot @1     # Graphviz diagram
```

## Flag Ordering

Flags must come **before** positional arguments:

```bash
# correct
jmcp search --errors --since 1h frontend
jmcp export --format csv @1
jmcp snapshot --label v2.0 frontend

# wrong (flags after positional args are ignored)
jmcp search frontend --errors          # --errors ignored
jmcp export @1 --format csv            # --format ignored
```

## Command Reference

### Discovery

| Command | What it does | Example |
|---------|-------------|---------|
| `health` | Check Jaeger connection | `jmcp health` |
| `services` | List traced services | `jmcp services` |
| `spans` | List span names for a service | `jmcp spans frontend` |
| `search` | Find traces by service | `jmcp search --errors --since 1h frontend` |
| `deps` | Show service dependency graph | `jmcp deps` |

### Investigation

| Command | What it does | Example |
|---------|-------------|---------|
| `topology` | Trace span tree | `jmcp topology @1` |
| `errors` | Error spans in a trace | `jmcp errors @1` |
| `details` | Single span details | `jmcp details @1 <span-id>` |
| `critical-path` | Slowest path through trace | `jmcp critical-path @1` |
| `investigate` | All of the above combined | `jmcp investigate @1` |
| `blame` | Root cause analysis | `jmcp blame @1` |

### Operations

| Command | What it does | Example |
|---------|-------------|---------|
| `report` | Generate investigation report | `jmcp report --title "Bug" @1` |
| `snapshot` | Capture service trace stats | `jmcp snapshot --label v2.0 frontend` |
| `diff` | Compare two snapshots | `jmcp diff before.json after.json` |
| `watch` | Live trace monitor | `jmcp watch --errors --alert frontend` |
| `check` | CI/CD health gate | `jmcp check --p95 500ms frontend` |
| `export` | Export trace data | `jmcp export --format csv @1` |

### Setup

| Command | What it does | Example |
|---------|-------------|---------|
| `init` | Create `.jmcp/` project dir | `jmcp init` |
| `replay` | Re-run investigation from report | `jmcp replay report.md` |
| `completion` | Shell completions | `jmcp completion bash` |

Run `jmcp <command> --help` for full flag details on any command.

## Global Flags

```
-e, --endpoint <url>     MCP endpoint (default http://localhost:16687/mcp)
-o, --output <format>    table, json, raw (default table)
-s, --save <file>        Save parsed JSON output to file
-t, --timeout <dur>      Request timeout (default 30s)
-v, --verbose            Log protocol details to stderr
    --no-color           Disable color output
```

## Configuration

```bash
jmcp init   # creates .jmcp/config.yaml
```

```yaml
endpoint: http://localhost:16687/mcp
defaults:
  search_depth: 20
  since: 1h
  output: table
```

Priority: CLI flags > environment variables > config.yaml > defaults.

Environment variables: `JMCP_ENDPOINT`, `NO_COLOR`.

## Shell Completions

```bash
# bash
eval "$(jmcp completion bash)"

# zsh
jmcp completion zsh > "${fpath[1]}/_jmcp"

# fish
jmcp completion fish | source
```

## Output Formats

```bash
jmcp search frontend                     # table (default, human-readable)
jmcp --output json search frontend       # JSON (for scripting/piping)
jmcp --output raw search frontend        # raw JSON-RPC response
jmcp --save traces.json search frontend  # save to file
```

## Troubleshooting

| Error | Cause | Fix |
|-------|-------|-----|
| `cannot connect to MCP server` | Jaeger not running or MCP not enabled | Start Jaeger: `docker compose up -d` or `go run ./cmd/jaeger`. MCP uses port 16687. |
| `no services returned` | No traces ingested | Send traces from your app or use tracegen: `docker compose up -d` |
| `session expired` | MCP session timed out (~5min idle) | jmcp auto-reconnects. If persistent, reduce `--timeout` |
| `no cached search results` | Running `@1` without prior search | Run `jmcp search <service>` first |

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success (or healthy for `check`) |
| `1` | Error (or unhealthy for `check`) |
| `2` | Connection failure |
| `3` | Invalid arguments / unknown command |

## Development

```bash
make build            # build to bin/jmcp
make test             # unit tests
make test-race        # race detector
make lint             # go vet + gofmt
make coverage         # coverage report
make integration-test # E2E tests (requires running Jaeger)
```

## Contributing

1. Fork and clone
2. `make build && make test`
3. Submit PR

## License

[Apache 2.0](LICENSE)
