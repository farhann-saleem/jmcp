#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${JMCP_ENDPOINT:-http://localhost:16687/mcp}"
BIN="${JMCP_BIN:-$(cd "$(dirname "$0")/.." && pwd)/bin/jmcp}"
PASS=0
FAIL=0
SKIP=0

pass() { echo "  PASS: $1"; PASS=$((PASS+1)); }
fail() { echo "  FAIL: $1 — $2"; FAIL=$((FAIL+1)); }
skip() { echo "  SKIP: $1 — $2"; SKIP=$((SKIP+1)); }

if [[ ! -x "$BIN" ]]; then
  echo "Building jmcp..."
  make build
fi

echo "jmcp integration tests"
echo "Endpoint: $ENDPOINT"
echo ""

# --- health ---
echo "[health]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json health 2>&1) || { fail "health" "exit $?"; }
if echo "$out" | grep -q '"status"'; then
  pass "health returns status"
else
  fail "health" "missing status field"
fi

# --- services ---
echo "[services]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json services 2>&1) || { fail "services" "exit $?"; }
if echo "$out" | grep -q '"services"'; then
  pass "services returns list"
else
  fail "services" "missing services field"
fi

# Prefer a non-jaeger service (jaeger internal service may lack searchable traces)
all_services=$(echo "$out" | tr -d '\n' | sed -n 's/.*"services":\s*\[\(.*\)\].*/\1/p' | tr ',' '\n' | tr -d '"  ')
service=""
for s in $all_services; do
  if [[ "$s" != "jaeger" ]]; then
    service="$s"
    break
  fi
done
# Fall back to jaeger if it's the only service
if [[ -z "$service" ]]; then
  service=$(echo "$all_services" | head -1)
fi
if [[ -z "$service" ]]; then
  echo "  No services found. Generate traces before running integration tests."
  exit 1
fi
echo "  Using service: $service"

# --- search ---
echo "[search]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json search --depth 5 "$service" 2>&1) || { fail "search" "exit $?"; }
if echo "$out" | grep -q '"traces"'; then
  pass "search returns traces"
else
  fail "search" "missing traces field"
fi

# --- topology ---
echo "[topology]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json topology @1 2>&1) || { fail "topology" "exit $?"; }
if echo "$out" | grep -q '"trace_id"'; then
  pass "topology returns trace_id"
else
  fail "topology" "missing trace_id"
fi

# --- errors ---
echo "[errors]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json errors @1 2>&1) || { fail "errors" "exit $?"; }
if echo "$out" | grep -q '"total_error_count"'; then
  pass "errors returns count"
else
  fail "errors" "missing total_error_count"
fi

# --- critical-path ---
echo "[critical-path]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json critical-path @1 2>&1) || { fail "critical-path" "exit $?"; }
if echo "$out" | grep -q '"segments"'; then
  pass "critical-path returns segments"
else
  fail "critical-path" "missing segments"
fi

# --- deps ---
echo "[deps]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json deps 2>&1) || { fail "deps" "exit $?"; }
if echo "$out" | grep -q '"dependencies"'; then
  pass "deps returns dependencies"
else
  fail "deps" "missing dependencies"
fi

# --- investigate ---
echo "[investigate]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json investigate @1 2>&1) || { fail "investigate" "exit $?"; }
if echo "$out" | grep -q '"topology"'; then
  pass "investigate returns combined output"
else
  fail "investigate" "missing topology in output"
fi

# --- blame ---
echo "[blame]"
out=$("$BIN" --endpoint "$ENDPOINT" --output json blame @1 2>&1) || { fail "blame" "exit $?"; }
if echo "$out" | grep -q '"primary_suspect"'; then
  pass "blame returns suspect"
else
  fail "blame" "missing primary_suspect"
fi

# --- export json ---
echo "[export json]"
out=$("$BIN" --endpoint "$ENDPOINT" export --format json @1 2>&1) || { fail "export json" "exit $?"; }
if echo "$out" | grep -q '"trace_id"'; then
  pass "export json valid"
else
  fail "export json" "invalid JSON output"
fi

# --- export csv ---
echo "[export csv]"
out=$("$BIN" --endpoint "$ENDPOINT" export --format csv @1 2>&1) || { fail "export csv" "exit $?"; }
if echo "$out" | head -1 | grep -q 'path,service,span_name'; then
  pass "export csv has headers"
else
  fail "export csv" "missing CSV headers"
fi

# --- export dot ---
echo "[export dot]"
out=$("$BIN" --endpoint "$ENDPOINT" export --format dot @1 2>&1) || { fail "export dot" "exit $?"; }
if echo "$out" | grep -q 'digraph\|trace'; then
  pass "export dot valid"
else
  fail "export dot" "missing digraph"
fi

# --- check ---
echo "[check]"
"$BIN" --endpoint "$ENDPOINT" check --error-rate 100 --p95 1h "$service" 2>&1 >/dev/null
rc=$?
if [[ $rc -eq 0 || $rc -eq 1 ]]; then
  pass "check runs (exit $rc)"
else
  fail "check" "unexpected exit code $rc"
fi

# --- init ---
echo "[init]"
tmpdir=$(mktemp -d)
(cd "$tmpdir" && "$BIN" init 2>&1 >/dev/null)
if [[ -f "$tmpdir/.jmcp/config.yaml" ]]; then
  pass "init creates config"
else
  fail "init" "missing config.yaml"
fi
rm -rf "$tmpdir"

# --- snapshot ---
echo "[snapshot]"
tmpdir=$(mktemp -d)
mkdir -p "$tmpdir/.jmcp/snapshots"
out=$("$BIN" --endpoint "$ENDPOINT" snapshot --label integ-test --dir "$tmpdir/.jmcp/snapshots" "$service" 2>&1) || { fail "snapshot" "exit $?"; }
if [[ -f "$tmpdir/.jmcp/snapshots/integ-test.json" ]]; then
  pass "snapshot creates file"
else
  fail "snapshot" "file not created"
fi
rm -rf "$tmpdir"

# --- completion ---
echo "[completion]"
out=$("$BIN" completion bash 2>&1) || { fail "completion bash" "exit $?"; }
if echo "$out" | grep -q 'complete'; then
  pass "completion bash"
else
  fail "completion bash" "invalid output"
fi

out=$("$BIN" completion zsh 2>&1) || { fail "completion zsh" "exit $?"; }
if echo "$out" | grep -q 'compdef'; then
  pass "completion zsh"
else
  fail "completion zsh" "invalid output"
fi

out=$("$BIN" completion fish 2>&1) || { fail "completion fish" "exit $?"; }
if echo "$out" | grep -q 'complete -c jmcp'; then
  pass "completion fish"
else
  fail "completion fish" "invalid output"
fi

# --- per-command help ---
echo "[help]"
for cmd in health services search topology errors critical-path deps investigate blame export check snapshot report watch replay init diff completion; do
  rc=0; out=$("$BIN" "$cmd" --help 2>&1) || rc=$?
  if [[ $rc -eq 0 ]]; then
    pass "$cmd --help"
  else
    fail "$cmd --help" "exit $rc"
  fi
done

# --- exit codes ---
echo "[exit-codes]"
rc=0; "$BIN" nonexistent 2>/dev/null || rc=$?
if [[ $rc -eq 3 ]]; then pass "unknown cmd = 3"; else fail "unknown cmd" "expected 3, got $rc"; fi

rc=0; "$BIN" --output xml health 2>/dev/null || rc=$?
if [[ $rc -eq 3 ]]; then pass "bad output = 3"; else fail "bad output" "expected 3, got $rc"; fi

# --- summary ---
echo ""
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
echo "All integration tests passed."
