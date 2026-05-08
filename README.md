# jmcp

[![CI](https://github.com/farhann-saleem/jmcp/actions/workflows/ci.yml/badge.svg)](https://github.com/farhann-saleem/jmcp/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/farhann-saleem/jmcp)](https://go.dev)
[![Release](https://img.shields.io/github/v/release/farhann-saleem/jmcp)](https://github.com/farhann-saleem/jmcp/releases)
[![License](https://img.shields.io/github/license/farhann-saleem/jmcp)](LICENSE)

**Incident debugging CLI for Jaeger via MCP.** Zero-copy-paste workflow from "something broke" to documented root cause.

jmcp hides MCP session handshakes, `Mcp-Session-Id` management, JSON-RPC payloads, and SSE parsing behind incident-oriented commands. Zero external dependencies — single static binary.

## Why jmcp?

| Without jmcp | With jmcp |
|---|---|
| Open Jaeger UI, click through traces, copy trace IDs | `jmcp search frontend --errors` |
| Switch tabs, paste IDs, click spans | `jmcp investigate @1` |
| Manually compare before/after deploys | `jmcp diff before-deploy after-deploy` |
| No CLI health gates for CI/CD | `jmcp check frontend --p95 500ms` |
| Write incident reports manually | `jmcp report @1 --title "Checkout timeout"` |

## Install

### Curl (no Go required)

```bash
curl -sSL https://raw.githubusercontent.com/farhann-saleem/jmcp/main/install.sh | bash
```

### Go Install

```bash
go install github.com/farhann-saleem/jmcp@latest
```

### Build from Source

```bash
git clone https://github.com/farhann-saleem/jmcp.git
cd jmcp
make build
./bin/jmcp --version
```

## Quick Start

```bash
# 1. Check connection
jmcp health

# 2. Find problem traces
jmcp search frontend --errors --since 30m

# 3. Investigate (topology + errors + critical path)
jmcp investigate @1

# 4. Root cause
jmcp blame @1

# 5. Document it
jmcp report @1 --title "Checkout timeout"
```

## Demo Setup

Start Jaeger with sample traces in one command:

```bash
docker compose up -d
jmcp health
```

## Command Reference

| Command | Description | Example |
|---|---|---|
| `health` | Check Jaeger server health | `jmcp health` |
| `services` | List traced services | `jmcp services` |
| `spans` | List span names for a service | `jmcp spans frontend --kind server` |
| `search` | Search traces | `jmcp search frontend --errors --since 1h` |
| `topology` | Show trace topology tree | `jmcp topology @1` |
| `errors` | Show error spans | `jmcp errors @1` |
| `details` | Show span details | `jmcp details @1 <span-id>` |
| `critical-path` | Critical path analysis | `jmcp critical-path @1` |
| `deps` | Service dependencies | `jmcp deps` |
| `investigate` | Full trace investigation | `jmcp investigate @1` |
| `blame` | Root cause analysis | `jmcp blame @1` |
| `report` | Generate investigation report | `jmcp report @1 --title "Bug"` |
| `snapshot` | Capture service state | `jmcp snapshot frontend --label v2.0` |
| `diff` | Compare snapshots | `jmcp diff before-deploy after-deploy` |
| `watch` | Live trace monitor | `jmcp watch frontend --errors --alert` |
| `check` | CI/CD health gate | `jmcp check frontend --p95 500ms` |
| `export` | Export trace data | `jmcp export @1 --format dot` |
| `init` | Initialize project dir | `jmcp init` |
| `replay` | Re-validate from report | `jmcp replay report.md` |
| `completion` | Shell completions | `jmcp completion bash` |

Every command supports `--help` for detailed flag reference.

## Trace References

`jmcp search` caches results. Use `@N` to reference traces from the last search:

```bash
jmcp search frontend --errors    # finds traces
jmcp topology @1                 # first result
jmcp errors @2                   # second result
jmcp investigate @last           # most recent
```

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

Initialize a project directory:

```bash
jmcp init
```

Creates `.jmcp/config.yaml` with defaults:

```yaml
endpoint: http://localhost:16687/mcp
defaults:
  search_depth: 20
  since: 1h
  output: table
```

CLI flags override config. Environment variables (`JMCP_ENDPOINT`, `NO_COLOR`) override config.

## Shell Completions

```bash
# bash — add to ~/.bashrc
eval "$(jmcp completion bash)"

# zsh — add to fpath
jmcp completion zsh > "${fpath[1]}/_jmcp"

# fish
jmcp completion fish | source
```

## CI/CD Integration

Use `jmcp check` as a deployment gate:

```yaml
# GitHub Actions example
- name: Trace Health Gate
  run: |
    jmcp check my-service \
      --error-rate 5 \
      --p95 500ms \
      --since 5m
```

Exit codes: `0` = healthy, `1` = unhealthy, `2` = connection error, `3` = invalid args.

## Output Formats

```bash
jmcp search frontend                    # table (default)
jmcp search frontend --output json      # JSON for scripting
jmcp search frontend --output raw       # raw JSON-RPC response
jmcp search frontend --save traces.json # save to file
```

## Privacy

jmcp collects **no telemetry**. All data stays between your machine and your Jaeger instance.

## Development

```bash
make build           # build binary
make test            # run tests
make test-race       # run tests with race detector
make lint            # go vet + gofmt check
make coverage        # test coverage report
make integration-test # requires running Jaeger
```

## Troubleshooting

**"cannot connect to MCP server"** — Jaeger not running or MCP not enabled. Start with `go run ./cmd/jaeger` or `docker compose up`. MCP defaults to port 16687.

**"no services returned"** — Jaeger running but no traces ingested. Generate traces with tracegen or your application.

**"session expired"** — MCP sessions timeout after ~5 minutes of inactivity. jmcp auto-reconnects, but long `--timeout` values may cause this. Reduce timeout or retry.

## Contributing

1. Fork and clone
2. `make build && make test`
3. Submit PR

## License

Apache 2.0
