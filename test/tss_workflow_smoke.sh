#!/usr/bin/env bash
set -euo pipefail

# Local smoke test for:
# - init 5 parties
# - keygen with parties 1,2,3 at threshold 2
# - sign on the initial 3-party committee
# - first regroup from old 3 -> new 5 with new threshold 3
# - sign on the 5-party committee
# - second regroup from old 4 -> new 3 with final threshold 1 using parties 2,3,4,5,
#   where 3,4,5 remain in the new committee and party 2 is old-only
# - sign on the final 3-party committee

gen_channel_password() {
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -hex 32
	else
		od -An -N32 -tx1 /dev/urandom | tr -d ' \n'
	fi
}

BIN="${BIN:-./tss}"
BASE="${BASE:-$(mktemp -d /private/tmp/tss-workflow-smoke.XXXXXX)}"
PASS="${PASS:-1234567890}"
MESSAGE="${MESSAGE:-1234567890}"
CHPASS="${CHPASS:-$(gen_channel_password)}"
KEYGEN_CH="${KEYGEN_CH:-$(printf '515%08X' "$(($(date +%s)+2400))")}"
SIGN1_CH="${SIGN1_CH:-$(printf '611%08X' "$(($(date +%s)+2400))")}"
REGROUP1_CH="${REGROUP1_CH:-$(printf '761%08X' "$(($(date +%s)+2400))")}"
SIGN2_CH="${SIGN2_CH:-$(printf '612%08X' "$(($(date +%s)+2400))")}"
REGROUP2_CH="${REGROUP2_CH:-$(printf '762%08X' "$(($(date +%s)+2400))")}"
SIGN3_CH="${SIGN3_CH:-$(printf '613%08X' "$(($(date +%s)+2400))")}"

BASE_PORTS=(19131 19132 19133 19134 19135)
ROUND1_TMP_PORTS=(19231 19232 19233)
ROUND2_TMP_PORTS=(19333 19334 19335)
PIDS=()

cleanup() {
	local pid
	local pids=("${PIDS[@]:-}")
	PIDS=()
	for pid in "${pids[@]}"; do
		pkill -P "$pid" 2>/dev/null || true
		kill "$pid" 2>/dev/null || true
		wait "$pid" 2>/dev/null || true
	done
}
trap cleanup EXIT

require_command() {
	local cmd="$1"
	command -v "$cmd" >/dev/null 2>&1 || {
		echo "$cmd is required but not installed"
		exit 1
	}
}

port_available() {
	local port="$1"
	if command -v lsof >/dev/null 2>&1; then
		! lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1
		return
	fi
	if command -v nc >/dev/null 2>&1; then
		! nc -z 127.0.0.1 "$port" >/dev/null 2>&1
		return
	fi
	echo "warning: neither lsof nor nc found; skipping port availability check" >&2
	return 0
}

check_required_ports() {
	local port
	for port in "${BASE_PORTS[@]}" "${ROUND1_TMP_PORTS[@]}" "${ROUND2_TMP_PORTS[@]}"; do
		if ! port_available "$port"; then
			echo "required port is already in use: $port"
			exit 1
		fi
	done
}

party_home() {
	local idx="$1"
	printf '%s/party-%s/chain-103' "$BASE" "$idx"
}

base_addr() {
	local idx="$1"
	printf '/ip4/127.0.0.1/tcp/%s' "${BASE_PORTS[$((idx - 1))]}"
}

round1_tmp_addr() {
	local idx="$1"
	printf '/ip4/127.0.0.1/tcp/%s' "${ROUND1_TMP_PORTS[$((idx - 1))]}"
}

round2_tmp_addr() {
	local idx="$1"
	printf '/ip4/127.0.0.1/tcp/%s' "${ROUND2_TMP_PORTS[$((idx - 3))]}"
}

round1_committee_addr() {
	local idx="$1"
	case "$idx" in
		1) round1_tmp_addr 1 ;;
		2) round1_tmp_addr 2 ;;
		3) round1_tmp_addr 3 ;;
		4) base_addr 4 ;;
		5) base_addr 5 ;;
		*) return 1 ;;
	esac
}

fail_phase() {
	local _phase="$1"
	local message="$2"
	cleanup
	echo "$message"
	echo "logs: $BASE"
	exit 1
}

wait_for_pids() {
	local timeout="$1"
	local deadline=$((SECONDS + timeout))

	while [ "${#PIDS[@]}" -gt 0 ]; do
		local remaining=()
		local pid
		for pid in "${PIDS[@]}"; do
			if kill -0 "$pid" 2>/dev/null; then
				remaining+=("$pid")
			else
				if wait "$pid"; then
					:
				else
					return 1
				fi
			fi
		done
		PIDS=()
		if [ "${#remaining[@]}" -gt 0 ]; then
			PIDS=("${remaining[@]}")
		fi

		if [ "${#PIDS[@]}" -eq 0 ]; then
			return 0
		fi
		if [ "$SECONDS" -gt "$deadline" ]; then
			return 124
		fi
		sleep 1
	done
}

check_phase_result() {
	local phase="$1"
	local timeout="$2"
	local status

	set +e
	wait_for_pids "$timeout"
	status=$?
	set -e
	if [ "$status" -eq 124 ]; then
		fail_phase "$phase" "$phase timed out; logs in $BASE"
	fi
	if [ "$status" -ne 0 ]; then
		fail_phase "$phase" "$phase failed; logs in $BASE"
	fi
}

assert_key_material() {
	local idx
	for idx in "$@"; do
		test -s "$(party_home "$idx")/default/sk.json"
		test -s "$(party_home "$idx")/default/pk.json"
	done
}

signature_from_log() {
	local log_file="$1"
	sed -n 's/.*received signature: \([0-9A-F]*\).*/\1/p' "$log_file" | tail -n1
}

start_keygen() {
	local idx="$1"
	local peers="$2"

	"$BIN" keygen \
		--home "$(party_home "$idx")" \
		--vault_name default \
		--parties 3 \
		--threshold 2 \
		--password "$PASS" \
		--channel_password "$CHPASS" \
		--channel_id "$KEYGEN_CH" \
		--p2p.peer_addrs "$peers" \
		--log_level debug \
		> "$BASE/keygen-$idx.log" 2>&1 &
	PIDS+=("$!")
}

start_sign() {
	local phase="$1"
	local channel_id="$2"
	local idx="$3"

	"$BIN" sign \
		--home "$(party_home "$idx")" \
		--vault_name default \
		--password "$PASS" \
		--message "$MESSAGE" \
		--channel_password "$CHPASS" \
		--channel_id "$channel_id" \
		--log_level debug \
		> "$BASE/$phase-$idx.log" 2>&1 &
	PIDS+=("$!")
}

run_sign_phase() {
	local phase="$1"
	local channel_id="$2"
	local timeout="$3"
	shift 3

	local idx
	local sig
	local ref_sig=""

	PIDS=()
	for idx in "$@"; do
		start_sign "$phase" "$channel_id" "$idx"
	done
	check_phase_result "$phase" "$timeout"

	for idx in "$@"; do
		sig="$(signature_from_log "$BASE/$phase-$idx.log")"
		if [ -z "$sig" ]; then
			fail_phase "$phase" "$phase missing signature output for party $idx; logs in $BASE"
		fi
		if [ -z "$ref_sig" ]; then
			ref_sig="$sig"
		elif [ "$sig" != "$ref_sig" ]; then
			fail_phase "$phase" "$phase produced mismatched signatures; logs in $BASE"
		fi
	done
}

start_regroup1() {
	local idx="$1"
	local extra_flags="$2"
	local new_peer_addrs="$3"

	# shellcheck disable=SC2086
	"$BIN" regroup \
		--home "$(party_home "$idx")" \
		--vault_name default \
		--password "$PASS" \
		--log_level debug \
		--channel_id "$REGROUP1_CH" \
		--channel_password "$CHPASS" \
		--threshold 2 \
		--parties 3 \
		--new_threshold 3 \
		--new_parties 5 \
		--p2p.new_peer_addrs "$new_peer_addrs" \
		$extra_flags \
		> "$BASE/regroup1-$idx.log" 2>&1 &
	PIDS+=("$!")
}

start_regroup2_old_only() {
	local idx="$1"
	local new_peer_addrs="$2"

	expect <<EOF > "$BASE/regroup2-$idx.log" 2>&1 &
set timeout -1
spawn "$BIN" regroup \
	--home "$(party_home "$idx")" \
	--vault_name default \
	--password "$PASS" \
	--log_level debug \
	--channel_id "$REGROUP2_CH" \
	--channel_password "$CHPASS" \
	--threshold 3 \
	--parties 5 \
	--new_threshold 1 \
	--new_parties 3 \
	--p2p.new_peer_addrs "$new_peer_addrs"
expect "Participant as a old committee?*" { send "y\r" }
expect "Participant as a new committee?*" { send "n\r" }
expect eof
EOF
	PIDS+=("$!")
}

start_regroup2_old_new() {
	local idx="$1"
	local new_listen="$2"
	local new_peer_addrs="$3"

	"$BIN" regroup \
		--home "$(party_home "$idx")" \
		--vault_name default \
		--password "$PASS" \
		--log_level debug \
		--channel_id "$REGROUP2_CH" \
		--channel_password "$CHPASS" \
		--threshold 3 \
		--parties 5 \
		--new_threshold 1 \
		--new_parties 3 \
		--p2p.new_listen "$new_listen" \
		--p2p.new_peer_addrs "$new_peer_addrs" \
		--is_old \
		--is_new_member \
		> "$BASE/regroup2-$idx.log" 2>&1 &
	PIDS+=("$!")
}

echo "logs: $BASE"
echo "binary: $BIN"
echo "channels: keygen=$KEYGEN_CH sign1=$SIGN1_CH regroup1=$REGROUP1_CH sign2=$SIGN2_CH regroup2=$REGROUP2_CH sign3=$SIGN3_CH"
require_command expect
if [ ! -x "$BIN" ]; then
	echo "tss binary not found or not executable: $BIN"
	exit 1
fi
check_required_ports
echo "phase: init 5 parties"

for idx in 1 2 3 4 5; do
	"$BIN" init \
		--home "$(party_home "$idx")" \
		--vault_name default \
		--moniker "party-$idx-chain-103" \
		--password "$PASS" \
		--p2p.listen "$(base_addr "$idx")" \
		--log_level debug \
		> "$BASE/init-$idx.log" 2>&1
done
echo "init completed"

PIDS=()
echo "phase: keygen on parties 1,2,3 with threshold 2"
start_keygen 1 "$(base_addr 2),$(base_addr 3)"
start_keygen 2 "$(base_addr 1),$(base_addr 3)"
start_keygen 3 "$(base_addr 1),$(base_addr 2)"
check_phase_result "keygen" 60
assert_key_material 1 2 3
echo "keygen completed"

echo "phase: sign on parties 1,2,3"
run_sign_phase "sign-after-keygen" "$SIGN1_CH" 60 1 2 3
echo "sign after keygen completed"

PIDS=()
echo "phase: first regroup old 3 -> new 5"
start_regroup1 1 "--p2p.new_listen /ip4/0.0.0.0/tcp/19231 --is_old --is_new_member" "$(base_addr 2),$(base_addr 3),$(round1_tmp_addr 1),$(round1_tmp_addr 2),$(round1_tmp_addr 3),$(base_addr 4),$(base_addr 5)"
start_regroup1 2 "--p2p.new_listen /ip4/0.0.0.0/tcp/19232 --is_old --is_new_member" "$(base_addr 1),$(base_addr 3),$(round1_tmp_addr 1),$(round1_tmp_addr 2),$(round1_tmp_addr 3),$(base_addr 4),$(base_addr 5)"
start_regroup1 3 "--p2p.new_listen /ip4/0.0.0.0/tcp/19233 --is_old --is_new_member" "$(base_addr 1),$(base_addr 2),$(round1_tmp_addr 1),$(round1_tmp_addr 2),$(round1_tmp_addr 3),$(base_addr 4),$(base_addr 5)"
start_regroup1 4 "--is_new_member" "$(base_addr 1),$(base_addr 2),$(base_addr 3),$(round1_tmp_addr 1),$(round1_tmp_addr 2),$(round1_tmp_addr 3),$(base_addr 5)"
start_regroup1 5 "--is_new_member" "$(base_addr 1),$(base_addr 2),$(base_addr 3),$(round1_tmp_addr 1),$(round1_tmp_addr 2),$(round1_tmp_addr 3),$(base_addr 4)"
check_phase_result "regroup1" 60
assert_key_material 1 2 3 4 5
echo "first regroup completed"

echo "phase: sign on parties 2,3,4,5"
run_sign_phase "sign-after-regroup1" "$SIGN2_CH" 60 2 3 4 5
echo "sign after first regroup completed"

PIDS=()
echo "phase: second regroup old 4 -> new 3 using parties 2,3,4,5"
# The second regroup uses new_threshold=1 intentionally for a minimal
# final 2-of-3 smoke-test committee.
start_regroup2_old_only 2 "$(round1_committee_addr 3),$(round1_committee_addr 4),$(round1_committee_addr 5),$(round2_tmp_addr 3),$(round2_tmp_addr 4),$(round2_tmp_addr 5)"
start_regroup2_old_new 3 "$(round2_tmp_addr 3)" "$(round1_committee_addr 2),$(round1_committee_addr 4),$(round1_committee_addr 5),$(round2_tmp_addr 3),$(round2_tmp_addr 4),$(round2_tmp_addr 5)"
start_regroup2_old_new 4 "$(round2_tmp_addr 4)" "$(round1_committee_addr 2),$(round1_committee_addr 3),$(round1_committee_addr 5),$(round2_tmp_addr 3),$(round2_tmp_addr 4),$(round2_tmp_addr 5)"
start_regroup2_old_new 5 "$(round2_tmp_addr 5)" "$(round1_committee_addr 2),$(round1_committee_addr 3),$(round1_committee_addr 4),$(round2_tmp_addr 3),$(round2_tmp_addr 4),$(round2_tmp_addr 5)"
check_phase_result "regroup2" 60
assert_key_material 3 4 5
echo "second regroup completed"

echo "phase: sign on parties 3,4"
run_sign_phase "sign-after-regroup2" "$SIGN3_CH" 60 3 4
echo "sign after second regroup completed"
