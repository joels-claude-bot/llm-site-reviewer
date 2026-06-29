#!/usr/bin/env bash
# e2e-docs.sh — build the docs, serve them, and verify every page actually renders.
# Self-contained: serves on :9810 (reserved, not the :9811 dev server) so it can run
# alongside the cowork dev tab without a port clash.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS="$ROOT/docs-site"
PORT=9810
BASE="http://localhost:$PORT"
FAILED=0

note() { printf '  %s\n' "$*"; }
pass() { printf '  \033[32mPASS\033[0m %s\n' "$*"; }
fail() { printf '  \033[31mFAIL\033[0m %s\n' "$*"; FAILED=1; }

echo "==> build"
( cd "$DOCS" && npm run build ) >/tmp/e2e-build.log 2>&1 || { cat /tmp/e2e-build.log; echo "build failed"; exit 1; }
note "built ($(grep -c '\.html' /tmp/e2e-build.log 2>/dev/null || echo '?') html outputs)"

echo "==> serve on :$PORT"
( cd "$DOCS" && npx rspress preview --host 0.0.0.0 --port "$PORT" ) >/tmp/e2e-serve.log 2>&1 &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null' EXIT

# wait for the server to answer
for _ in $(seq 1 40); do
  curl -sf -o /dev/null "$BASE/" && break
  sleep 0.25
done
curl -sf -o /dev/null "$BASE/" || { echo "server never came up"; cat /tmp/e2e-serve.log; exit 1; }
note "server up"

echo "==> verify pages (HTTP 200 + expected content in SSR HTML)"
# path<TAB>string that must appear in the rendered HTML
check() {
  local path="$1" needle="$2"
  local code body
  code=$(curl -s -o /tmp/e2e-page.html -w '%{http_code}' "$BASE$path")
  body=$(cat /tmp/e2e-page.html)
  if [[ "$code" != "200" ]]; then
    fail "$path -> HTTP $code"
  elif ! grep -qF "$needle" <<<"$body"; then
    fail "$path -> 200 but missing expected text: \"$needle\""
  else
    pass "$path"
  fi
}

check "/"                          "llm-site-reviewer"
check "/concept/principle.html"    "never sent to a large language model"
check "/concept/passes.html"       "viewport-segmented"
check "/concept/exit-codes.html"   "Block the merge"
check "/landscape/existing-tools.html" "wrap, don't rebuild"
check "/spec/catch-categories.html"    "MUST NOT CATCH"
check "/spec/test-corpus.html"         "two-sided"

echo
if [[ "$FAILED" -eq 0 ]]; then
  echo "e2e: all checks passed"
else
  echo "e2e: FAILURES above"
fi
exit "$FAILED"
