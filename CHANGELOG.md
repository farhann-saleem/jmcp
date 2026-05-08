# Changelog

All notable changes to jmcp are documented here. Format based on [Keep a Changelog](https://keepachangelog.com/).

## [0.3.0] - Unreleased

### Added
- Shell completions for bash, zsh, and fish (`jmcp completion bash|zsh|fish`)
- Panic recovery with clean error messages and issue reporting link
- Comprehensive integration test suite (35+ tests covering all commands)
- Docker Compose for one-command demo setup
- CHANGELOG.md
- `make test-race` target for race condition detection
- `make coverage` target for test coverage reports
- `make lint` now checks gofmt formatting

### Changed
- Professional README with badges, command reference, CI/CD examples
- Integration test expanded from 3 to 35+ validations

## [0.2.0] - 2026-05-08

### Fixed
- Percentile calculation bug for small sample sizes (used nearest-rank method)
- Watch command unbounded memory leak (capped at 10,000 traces with batch eviction)
- Per-command `--help` printing help text twice
- Search cache errors silently swallowed (now warns on stderr)
- Report silently skipping span details on fetch failure (now warns)

### Added
- Color output with ANSI codes (respects `--no-color` and `NO_COLOR` env var)
- Per-command `--help` for all 19 commands
- `.jmcp/config.yaml` reading (endpoint, search_depth, since, output defaults)
- `TraceCount` display when server truncates search results
- Interactive picker improvements: padded numbers, 3 retries, case-insensitive matching
- Test coverage for `cmd/`, `output/`, `analysis/` packages (30+ tests)
- GoReleaser config for cross-platform releases
- GitHub Actions CI + release workflows
- Install script (`curl | bash`)

## [0.1.0] - 2026-05-07

### Added
- Initial release with 19 commands: health, services, spans, search, topology, errors, details, critical-path, deps, investigate, report, snapshot, diff, watch, blame, export, check, init, replay
- MCP client with session handshake, reconnect, SSE parsing
- Trace cache with `@N` / `@last` references
- Table, JSON, and raw output formats
- Interactive service and trace pickers
- Snapshot comparison with `diff`
- Markdown report generation
- CI health gate with `check`
