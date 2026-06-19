#!/usr/bin/env bash
#
# Manual end-to-end verification of the gcb data pipeline against a local
# OpenSearch cluster (http://localhost:9200, no auth, security plugin disabled).
#
# Usage:
#   BGG_USERNAME=you BGG_PASSWORD=secret \
#     testdata/full_data_test/verify_full_pipeline.sh <events-file.xlsx|.csv>
#
# All build artifacts, fetched data, and per-step logs are written to ./tmp/
# (gitignored). The script can be run from anywhere; it resolves the repo root
# from its own location.

set -euo pipefail

# --- Paths -----------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

TMP_DIR="$REPO_ROOT/tmp"
LOG_DIR="$TMP_DIR/logs"
BIN="$TMP_DIR/gcb_test"
RANKS_CSV="$TMP_DIR/ranks.csv"
MAPPING_JSON="$TMP_DIR/bgg_mapping.json"

# --- Shared OpenSearch config (read by viper via AutomaticEnv) --------------
export OS_ADDRESS="http://localhost:9200"
export BATCH_SIZE="1000"
# Default index names (event_index / change_log_index) and no OS password.

EVENTS="${1:-}"

usage() {
  cat >&2 <<'EOF'
Usage:
  BGG_USERNAME=<user> BGG_PASSWORD=<pass> \
    testdata/full_data_test/verify_full_pipeline.sh <events-file.xlsx|.csv>

Requires a local OpenSearch reachable at http://localhost:9200 (no auth).
EOF
}

# --- Preflight -------------------------------------------------------------
fail() { echo "ERROR: $*" >&2; }

if [[ -z "${BGG_USERNAME:-}" ]]; then fail "BGG_USERNAME not set"; usage; exit 1; fi
if [[ -z "${BGG_PASSWORD:-}" ]]; then fail "BGG_PASSWORD not set"; usage; exit 1; fi

if [[ -z "$EVENTS" ]]; then fail "events file argument required"; usage; exit 1; fi
if [[ ! -f "$EVENTS" ]]; then fail "events file not found: $EVENTS"; exit 1; fi
case "$EVENTS" in
  *.xlsx|*.csv) ;;
  *) fail "events file must end in .xlsx or .csv: $EVENTS"; exit 1 ;;
esac

if ! curl -sf --max-time 5 "$OS_ADDRESS" >/dev/null 2>&1; then
  fail "OpenSearch not reachable at $OS_ADDRESS"
  cat >&2 <<'EOF'
Start a local single-node cluster with the security plugin disabled:
  docker run -d --name gcb-os-docker \
    -p 9200:9200 -p 9600:9600 \
    -e "discovery.type=single-node" \
    -e "DISABLE_SECURITY_PLUGIN=true" \
    opensearchproject/opensearch:latest
EOF
  exit 1
fi

# --- Build -----------------------------------------------------------------
mkdir -p "$LOG_DIR"
echo "▶ build → $BIN"
go build -ldflags="-w -s" -o "$BIN" .

# --- Step runner -----------------------------------------------------------
STEP_RESULTS=()

strip_ansi() { sed -E $'s/\x1b\\[[0-9;]*m//g'; }

summary() {
  printf '\n=== Summary ===\n'
  local r
  for r in "${STEP_RESULTS[@]}"; do
    printf '  %s\n' "$r"
  done
  printf '  logs: %s\n' "$LOG_DIR"
}

run_step() {
  local name="$1" logfile="$2"
  shift 2
  printf '\n▶ %s\n' "$name"

  set +e
  "$@" >"$logfile" 2>&1
  local code=$?
  set -e

  local stripped err_lines warn_lines err_count warn_count
  stripped="$(strip_ansi <"$logfile")"
  err_lines="$(printf '%s\n' "$stripped" | grep -E ' (ERR|FTL|PNC) ' || true)"
  warn_lines="$(printf '%s\n' "$stripped" | grep -E ' WRN ' || true)"
  err_count="$(printf '%s' "$err_lines"  | grep -c . || true)"
  warn_count="$(printf '%s' "$warn_lines" | grep -c . || true)"

  if [[ "$code" -ne 0 ]]; then
    printf '✗ FAIL %s (exit %d) — see %s\n' "$name" "$code" "$logfile"
    [[ -n "$err_lines" ]] && printf '%s\n' "$err_lines" | sed 's/^/    /'
    STEP_RESULTS+=("✗ $name (exit $code, ${err_count} err, ${warn_count} warn)")
    summary
    exit 1
  fi

  if [[ "$err_count" -gt 0 ]]; then
    printf '✗ FAIL %s — exited 0 but log has %d error line(s) — see %s\n' \
      "$name" "$err_count" "$logfile"
    printf '%s\n' "$err_lines" | sed 's/^/    /'
    STEP_RESULTS+=("✗ $name (logged ${err_count} err, ${warn_count} warn)")
    summary
    exit 1
  fi

  printf '✓ %s — 0 errors, %d warnings — %s\n' "$name" "$warn_count" "$logfile"
  if [[ "$warn_count" -gt 0 ]]; then
    printf '%s\n' "$warn_lines" | sed 's/^/    /'
  fi
  STEP_RESULTS+=("✓ $name (0 err, ${warn_count} warn)")
}

# --- Pipeline --------------------------------------------------------------
run_step "init" "$LOG_DIR/01_init.log" \
  "$BIN" data init -c --filepath "$EVENTS"

run_step "fetch-bgg" "$LOG_DIR/02_fetch-bgg.log" \
  "$BIN" data fetch-bgg --output "$RANKS_CSV"

run_step "bgg" "$LOG_DIR/03_bgg.log" \
  "$BIN" data bgg --filepath "$RANKS_CSV" --output "$MAPPING_JSON"

run_step "update" "$LOG_DIR/04_update.log" \
  "$BIN" data update --filepath "$EVENTS" --bgg-mapping "$MAPPING_JSON"

summary
printf '\n✓ All steps passed.\n'
