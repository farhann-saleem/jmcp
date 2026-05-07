# jmcp

`jmcp` is a Go CLI for Jaeger MCP. It hides the Streamable HTTP session handshake, `Mcp-Session-Id` handling, JSON-RPC payloads, and SSE parsing behind incident-oriented commands.

## Build

```bash
make build
./bin/jmcp --version
```

## Core Usage

```bash
jmcp health
jmcp services
jmcp spans frontend --kind server
jmcp search frontend --errors --since 30m --depth 20
jmcp topology @1
jmcp errors @1
jmcp critical-path @1
jmcp investigate @1
```

Global flags:

```bash
jmcp --endpoint http://localhost:16687/mcp --output json health
jmcp -e http://localhost:16687/mcp -o table search frontend --errors
jmcp --save out.json search frontend
jmcp --verbose health
```

## Production Debugging Commands

```bash
jmcp init
jmcp report @1 --title "Checkout timeout"
jmcp snapshot frontend --label before-deploy
jmcp diff before-deploy after-deploy
jmcp blame @1
jmcp export @1 --format dot
jmcp check frontend --error-rate 5 --p95 500ms
jmcp watch frontend --errors --interval 10s
```

`jmcp search` caches results in `.jmcp/.last_search.json` when `.jmcp/` exists, otherwise in `/tmp/jmcp_last_search.json`. Trace-consuming commands accept explicit trace IDs, `@N`, `@last`, or no argument in an interactive terminal.

## Jaeger MCP Endpoint

The default endpoint is:

```text
http://localhost:16687/mcp
```

Override it with `--endpoint` or `JMCP_ENDPOINT`.

## Tests

```bash
make test
make lint
make integration-test
```

The integration test expects a running Jaeger MCP server with trace data.

