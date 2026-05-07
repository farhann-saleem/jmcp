#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${JMCP_ENDPOINT:-http://localhost:16687/mcp}"
BIN="${JMCP_BIN:-./bin/jmcp}"

if [[ ! -x "$BIN" ]]; then
  go build -o ./bin/jmcp .
fi

"$BIN" --endpoint "$ENDPOINT" health --output json >/dev/null
"$BIN" --endpoint "$ENDPOINT" services --output json >/dev/null

service="$("$BIN" --endpoint "$ENDPOINT" services --output json | tr -d '\n' | sed -n 's/.*"services": *\[\([^]]*\)\].*/\1/p' | tr ',' '\n' | tr -d ' "[]' | head -n1)"
if [[ -z "$service" ]]; then
  echo "No services returned by Jaeger; generate traces before running integration tests." >&2
  exit 1
fi

"$BIN" --endpoint "$ENDPOINT" search "$service" --depth 1 --output json >/dev/null
echo "integration smoke passed for service: $service"
