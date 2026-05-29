#!/usr/bin/env bash
# Runs time-windowed signing scenarios for a 7-party (threshold-3) TSS setup.
# Usage: ./scripts/test-sign-rounds.sh [rounds] [max_scenarios]
# This harness assumes:
# - the repository contains the sign discovery timeout changes
# - party homes already exist at testing/.test1 ... testing/.test7
# - each party uses vault_name=default
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TSS_BIN="${TSS_BIN:-${ROOT}/tss}"
ROUNDS="${1:-1}"
MAX_SCENARIOS="${2:-0}"   # 0 = run all
PARTIES=7       # total signers
THRESHOLD=3     # BNB TSS threshold — needs threshold+1 = 4 to sign
MIN_SIGN=$((THRESHOLD + 1))
PASS=0
FAIL=0
RESULTS_LOG="${SCRIPT_DIR}/test-sign-rounds.log"
PARTY_TIMEOUT=30  # seconds before killing a stalled party
PASSWORD="${PASSWORD:-123456789}"
CHANNEL_PASSWORD="${CHANNEL_PASSWORD:-123456789}"
MESSAGE_DEC="${MESSAGE_DEC:-81878784180467215339657199009968320418167612298391878172028460193657732373998}"
DISCOVERY_TIMEOUT="${DISCOVERY_TIMEOUT:-5s}"
VAULT_NAME="${VAULT_NAME:-default}"
LOG_LEVEL="${LOG_LEVEL:-info}"

# Clear logs from previous run
> "$RESULTS_LOG"
for i in $(seq 1 "$PARTIES"); do
  > "${SCRIPT_DIR}/test-party${i}.log"
done

# Delays (seconds) for each party's startup: one value per party (P1..P7).
# Discovery window is ~5s — parties that start >5s after the session opens miss it.
# Min required to sign: threshold+1 = 4
SCENARIOS=(
  # KEY INSIGHT: behavior depends on early_count = number of parties with delay <= 5s
  #   early_count < threshold+1 (4) → bootstrap keeps waiting → late parties can join
  #   early_count >= threshold+1  → bootstrap closes at t=5s, signing starts
  #     └─ late parties arriving DURING signing  → 0/7 disruption (full failure)
  #     └─ late parties arriving AFTER signing   → safe partial miss

  # --- all within window (delay <=5s) — 7 early → 7/7 ---
  "0 0 0 0 0 0 0"       # all simultaneous
  "0 1 2 3 4 5 5"       # staggered 1s apart, last two at boundary
  "0 0 5 5 5 5 5"       # first two immediate, rest at boundary
  "0 1 0 2 0 3 0"       # alternating early/late within window
  "3 0 1 5 2 4 0"       # random spread within window
  "0 5 0 5 0 5 0"       # alternating 0 and boundary
  "1 3 5 0 2 4 1"       # fully mixed, all within window

  # --- high-indexed party far beyond — 6 early, signing fast → 6/7 ---
  "0 0 0 0 0 0 6"       # p7 misses, others done before p7 arrives
  "0 0 0 0 0 14 0"      # p6 far beyond, safe partial

  # --- p1 (lowest-indexed) beyond — disrupts ongoing signing → 0/7 ---
  # p1 arriving 1s after bootstrap-close causes full disruption (protocol-level)
  "6 0 0 0 0 0 0"       # p1 at 6s, 6 early sign, p1 arrives mid-signing → 0/7

  # --- 2 parties beyond window — 5 early, fast signing → safe partial ---
  "0 0 0 0 0 7 14"      # p6,p7 far beyond, 5/7
  "0 0 0 0 0 10 20"     # p6,p7 extreme, 5/7
  "8 12 0 0 0 0 0"      # p1,p2 arrive well after signing done → 5/7
  "0 0 6 0 0 17 0"      # p3 arrives ~1s after close (may land in/out), p6 far → 5-6/7
  "6 0 0 0 7 0 0"       # p1,p5 arrive after fast 5-party signing → 5/7
  "0 0 0 0 0 15 20"     # p6,p7 extreme, 5/7

  # --- 3 parties beyond window — only 4 early, bootstrap waits full timeout ---
  # All late parties arrive AFTER bootstrap-close; signing completes with the
  # first 4 that formed a group, rest miss cleanly → 4/7
  "0 0 0 0 6 12 18"     # 4 early sign; p5/p6/p7 arrive staggered after → 4/7
  "6 6 6 0 0 0 0"       # 4 early sign; p1/p2/p3 arrive staggered after → 4/7
  "0 6 0 6 0 6 0"       # 4 early sign; p2/p4/p6 staggered after → 4/7
  "0 0 0 0 8 14 20"     # 4 early, p5/p6/p7 extreme → 4/7
  "0 0 0 6 17 20 0"     # 4 early (p1-p3,p7), p4 just beyond, p5/p6 extreme → 4-5/7
  "0 0 0 0 7 14 20"     # 4 early, p5/p6/p7 varied delays → 4/7
  "0 0 6 8 10 0 0"      # 4 early (p1,p2,p6,p7); p3/p4/p5 staggered after → 4/7
  "6 8 0 0 0 0 17"      # 4 early (p3-p6); p1 arrives 1s into signing → 0/7 DISRUPTION

  # --- 4 parties exactly beyond (3 early < threshold+1) → bootstrap WAITS ---
  # Only 3 early parties — below threshold+1=4 — bootstrap keeps open.
  # All 4 late parties MUST arrive TOGETHER (same delay) to all join;
  # if staggered, the first late one triggers close and the rest disrupt → 0/7
  "6 6 6 6 0 0 0"       # 3 early wait; all 4 late arrive together at t=6 → 7/7
  "6 0 6 0 6 0 6"       # 3 early wait; all 4 late arrive together at t=6 → 7/7
  "0 0 0 6 8 12 17"     # 3 early wait; p4 at t=6 triggers close (4 total), p5-p7 disrupt → 0/7
  "6 7 8 9 0 0 0"       # 3 early wait; p1 at t=6 triggers close, p2-p4 disrupt → 0/7
  "0 0 0 6 7 17 20"     # 3 early wait; p4 at t=6 triggers close, p5 at t=7 disrupts → 0/7

  # --- 5+ beyond (2 early < threshold+1) → bootstrap WAITS for late arrivals ---
  "0 0 6 6 14 17 20"    # 2 early wait; p3,p4 arrive together at t=6 → close; p5-p7 disrupt? → 4/7
  "6 6 6 6 6 0 0"       # 2 early wait; all 5 late arrive together at t=6 → 7/7
  "0 2 4 6 10 14 20"    # 3 early by t=4, bootstrap still waiting; p4 at t=6 triggers, rest disrupt → 0/7
  "6 12 0 0 0 17 20"    # 3 early wait; p1 at t=6 triggers close, p1+3 sign; p2/p6/p7 miss → 4/7
  "0 6 8 12 14 17 20"   # 1 early; accumulates until signing triggered, rest disrupt → 0/7
)

log() {
  echo "$@" | tee -a "$RESULTS_LOG"
}

run_timed() {
  local secs=$1; shift
  "$@" &
  local pid=$!
  (sleep "$secs" && kill "$pid" 2>/dev/null) &
  local killer=$!
  wait "$pid" 2>/dev/null || true
  kill "$killer" 2>/dev/null
  wait "$killer" 2>/dev/null || true
}

run_party() {
  local party=$1
  local channel_id=$2

  run_timed "$PARTY_TIMEOUT" "$TSS_BIN" sign \
    --home "${ROOT}/testing/.test${party}" \
    --vault_name "$VAULT_NAME" \
    --password "$PASSWORD" \
    --channel_password "$CHANNEL_PASSWORD" \
    --channel_id "$channel_id" \
    --log_level "$LOG_LEVEL" \
    --message "$MESSAGE_DEC" \
    --sign_discovery_timeout "$DISCOVERY_TIMEOUT"
}

run_scenario() {
  local round=$1; shift
  local delays=("$@")

  # Unique channel ID per scenario — prevents cross-scenario bootstrap interference.
  # Format: 3-char prefix (scenario index) + 8-char expiry hex (now + 30min), total 11 chars.
  # The TSS binary rejects channel IDs whose embedded expiry <= now.
  local expiry channel_id
  expiry=$(( $(date +%s) + 1800 ))
  channel_id=$(printf '%03X%08X' "$round" "$expiry")

  # Record current line offsets before this round
  local offsets=()
  for i in $(seq 1 "$PARTIES"); do
    offsets+=( "$(wc -l < "${SCRIPT_DIR}/test-party${i}.log")" )
    echo "--- Round ${round} ---" >> "${SCRIPT_DIR}/test-party${i}.log"
  done

  # Launch all parties with their respective delays
  local pids=()
  for i in $(seq 1 "$PARTIES"); do
    local d="${delays[$((i-1))]}"
    (sleep "$d"; run_party "$i" "$channel_id" 2>&1) >> "${SCRIPT_DIR}/test-party${i}.log" &
    pids+=( $! )
  done

  wait "${pids[@]}" || true

  # Count successful signers
  local ok=0 failed_parties=""
  for i in $(seq 1 "$PARTIES"); do
    local plog="${SCRIPT_DIR}/test-party${i}.log"
    local off="${offsets[$((i-1))]}"
    if tail -n +$((off+1)) "$plog" 2>/dev/null | grep -Eq "sign(ing)? finished!"; then
      ok=$((ok+1))
    else
      failed_parties="${failed_parties} P${i}"
    fi
  done

  if [ "$ok" -ge "$MIN_SIGN" ]; then
    log "  PASS ($ok/${PARTIES} signed)${failed_parties:+ — failed:${failed_parties}}"
    PASS=$((PASS+1))
  else
    log "  FAIL ($ok/${PARTIES} signed) — failed:${failed_parties}"
    FAIL=$((FAIL+1))
    for i in $(seq 1 "$PARTIES"); do
      local plog="${SCRIPT_DIR}/test-party${i}.log"
      local off="${offsets[$((i-1))]}"
      if ! tail -n +$((off+1)) "$plog" 2>/dev/null | grep -Eq "sign(ing)? finished!"; then
        log "    [P${i} last lines]:"
        tail -n +$((off+1)) "$plog" 2>/dev/null | tail -5 | sed 's/^/      /' | tee -a "$RESULTS_LOG"
      fi
    done
  fi
}

cd "$ROOT"

SEPARATOR="================================================================"
TIMESTAMP="$(date '+%Y-%m-%d %H:%M:%S')"

{
  echo "$SEPARATOR"
  echo "  TSS Sign Bootstrap Test — $TIMESTAMP"
  echo "$SEPARATOR"
} | tee -a "$RESULTS_LOG"

log "Parties: ${PARTIES}  Threshold: ${THRESHOLD}  Min-sign: ${MIN_SIGN}"
log "Party timeout: ${PARTY_TIMEOUT}s"
log "Discovery timeout: ${DISCOVERY_TIMEOUT}"
log ""

ROUND_COUNTER=0
SCENARIO_COUNTER=0

for scenario in "${SCENARIOS[@]}"; do
  if [ "$MAX_SCENARIOS" -gt 0 ] && [ "$SCENARIO_COUNTER" -ge "$MAX_SCENARIOS" ]; then
    break
  fi
  SCENARIO_COUNTER=$((SCENARIO_COUNTER+1))
  read -r -a delays <<< "$scenario"

  # Build header line: "P1=Xs P2=Ys ..."
  header=""
  for i in $(seq 1 "$PARTIES"); do
    header="${header} P${i}=${delays[$((i-1))]}s"
  done

  {
    echo ""
    echo "----------------------------------------------------------------"
    echo "  Scenario:${header}"
    echo "----------------------------------------------------------------"
  } | tee -a "$RESULTS_LOG"

  for r in $(seq 1 "$ROUNDS"); do
    ROUND_COUNTER=$((ROUND_COUNTER+1))
    run_scenario "$ROUND_COUNTER" "${delays[@]}"
  done
  log ""
done

{
  echo ""
  echo "$SEPARATOR"
  echo "  Results: PASS=$PASS  FAIL=$FAIL  total=$((PASS+FAIL))"
  echo "$SEPARATOR"
} | tee -a "$RESULTS_LOG"
