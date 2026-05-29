# Time-Windowed Discovery for `tss` Sign Bootstrap

## Purpose

This document is the detailed PR-support note for the upstream `tss` changes
that make sign bootstrap work with any signer set `k` where `t+1 <= k <= n`.

It is written in native `tss` terms so it can be used directly in upstream
review.

Related upstream discussion:
- [`bnb-chain/tss` issue #34](https://github.com/bnb-chain/tss/issues/34)

## Problem

The original sign bootstrap path could complete independently on different
nodes as soon as each node observed the minimum signer count. That worked for
some `t+1` cases, but it broke when more than `t+1` signers tried to join the
same session.

The failure mode was committee inconsistency:

- one node could begin signing with one signer set
- another node could begin signing with a different signer set
- late arrivals could still be accepted during a narrow handoff gap
- sign traffic could land on the wrong protocol handler during bootstrap/sign
  overlap

Once those conditions happened, the round no longer had one shared participant
set and the session could stall or fail.

## Behavior After the Change

In `SignMode`, bootstrap becomes a time-windowed discovery phase.

When the first valid sign peer is discovered, the session starts a discovery
deadline:

```text
deadline = first_peer_time + sign_discovery_timeout
```

Bootstrap completes when either:

- all expected sign peers have connected, or
- the discovery timeout has elapsed and at least `threshold` remote peers are
  present

This changes sign bootstrap from "finish as soon as the minimum is seen" to
"finish with one consistent signer set for this session".

The intended result is:

- if all parties arrive within the window, all of them sign
- if only `k` parties arrive within the window and `k >= t+1`, those `k` sign
- if fewer than `t+1` parties are available, signing does not proceed
- parties arriving after the session is committed do not disrupt a session that
  has already formed

## Detailed Changes

### 1. Configurable sign discovery timeout

The sign command gains a new duration flag:

```bash
./tss sign ... --sign_discovery_timeout 5s
```

`0` means "require all parties". A positive duration means "allow a flexible
signer set to form within this window".

### 2. Shared session deadline

The bootstrapper starts a discovery deadline on the first peer connection and
propagates that deadline through bootstrap messages. This is important for late
starters: a late party must inherit the existing session deadline, not create a
fresh local window.

Without deadline propagation, a late party could collect more peers than the
early parties and form a larger committee for the same sign session, which
would break session consistency.

### 3. Explicit bootstrap commit

There is a difference between:

- the logical point where bootstrap is considered "finished", and
- the moment the bootstrap host has actually stopped accepting new peers

The change introduces an explicit committed state after the bootstrap polling
loop exits. Late peers are rejected after commit instead of being accepted
during the handoff gap.

This prevents different parties from finalizing different signer sets for the
same session.

### 4. Protocol separation between bootstrap and signing

Bootstrap and signing use different stream handlers and must not accept each
other's traffic. The change ensures:

- bootstrap hosts accept bootstrap streams only
- signing hosts accept signing streams only

This avoids a failure mode where sign traffic reaches a still-running bootstrap
host and is silently discarded.

### 5. Retry and backoff handling

Late starters are sensitive to dial timing. The change improves transport
reliability by:

- clearing libp2p dial backoff before reconnect attempts
- retrying stream-negotiation failures instead of treating them as fatal

This makes the observed signer set better match the actual online set during
the discovery window.

## Why This Is Safe

This PR changes session formation in `tss`; it does not change the signing
math in `tss-lib`.

`tss-lib` already supports signing with any subset size `k` as long as:

- `k >= t+1`
- every signer uses the same subset

The purpose of the discovery-window changes is to guarantee that all signers in
one session converge on the same subset before the round begins.

## Files Affected in `tss`

The upstream-facing changes are concentrated in these areas:

- `common/config.go`
- `common/messages.go`
- `common/bootstrapper.go`
- `cmd/root.go`
- `p2p/p2p_transporter.go`

If included in the PR, `client/client.go` may also contain a small defensive
guard depending on the exact final patch set, but it is not the core of the
feature.

## Native Command Examples

Require every party to join before signing:

```bash
./tss sign \
  --home ~/.tss \
  --vault_name default \
  --password 123456789 \
  --channel_id <SIGN_CH> \
  --channel_password 123456789 \
  --message <DECIMAL_MESSAGE> \
  --sign_discovery_timeout 0
```

Allow a flexible signer set to form within a 5-second window:

```bash
./tss sign \
  --home ~/.tss \
  --vault_name default \
  --password 123456789 \
  --channel_id <SIGN_CH> \
  --channel_password 123456789 \
  --message <DECIMAL_MESSAGE> \
  --sign_discovery_timeout 5s
```

## Validation Harness

The change was validated with a 7-party sign harness that repeatedly starts
native `tss sign` processes with controlled per-party startup delays.

The validation setup uses:

- `parties = 7`
- `threshold = 3`
- minimum signer set `t+1 = 4`
- multiple startup-delay scenarios to simulate early joiners, boundary joiners,
  and late joiners

Each scenario assigns one startup delay per party and then launches the same
native sign command on all seven parties against one shared channel. In shell
terms, each party run has the form:

```bash
./tss sign \
  --home <party-home> \
  --vault_name default \
  --password 1234567890 \
  --channel_id <unique-channel> \
  --channel_password 1234567890 \
  --message <DECIMAL_MESSAGE> \
  --sign_discovery_timeout 5s
```

The harness evaluates whether each party completed signing successfully and
records the final signed-party count for the scenario.

The scenario set covers these cases:

- all seven parties start inside the discovery window
- exactly `k` parties start inside the discovery window, for multiple values of
  `k` where `4 <= k < 7`
- fewer than `t+1` parties start early, so bootstrap must remain open until
  enough participants arrive
- late arrivals after the session has already committed

## Validation Results

The recorded run reports:

- `PASS=34`
- `FAIL=0`
- total scenarios: `34`

The result set covers these important regimes:

- all parties inside the window: full-committee signing succeeds
- only a subset `k` inside the window with `k >= t+1`: that subset signs
  successfully
- fewer than `t+1` early parties: bootstrap continues waiting instead of
  prematurely committing
- late arrivals after the committed session boundary do not disrupt an already
  formed session

Representative outcomes from the recorded scenarios:

- `0 0 0 0 0 0 0` -> `PASS (7/7 signed)`
- `0 0 0 0 0 0 6` -> `PASS (6/7 signed) - failed: P7`
- `6 6 6 6 0 0 0` -> `PASS (7/7 signed)`
- `0 0 0 6 8 12 17` -> `PASS (4/7 signed) - failed: P5 P6 P7`
- final summary -> `Results: PASS=34  FAIL=0  total=34`
