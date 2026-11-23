# Stage 3B completion — design approval requested

Proposal only. No API frozen, product code changed, implementation kit generated,
coder invoked or commit made. This is spec-loop v1.1.1 step 2a's separate review gate.
The assignment authorizes the objective, not approval of this newly proposed design.

## Verified starting point

Clean rddd-foundation HEAD 0d70ab726edf7d06bd836f6ebdabae0563c14391.
Both supplied Stage 3A commits are ancestors. Current checkout additionally contains
3B1 kits/implementation and the cancellation correction; preserve all four commits.
The historical OBSERVATION plans describe an older state; their statements that no
journal/gateway exists are obsolete, not requirements to recreate completed work.
SESSION-SPEC.md documents 3B1's deliberate cursor/recovery deferral.

Specification checkout clean at 9a80977efaf2ff740a257f6e30314dac42ec384e;
draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87 is an ancestor.
All 19 checksums pass. Manifest SHA256:
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Ratified profile, all 55 vector rows, four OBSERVATION plans, current observation
contracts/implementation/tests and journal planning/replay implementation inspected.

Current validation: go test -count=1 ./..., go test -race -count=1 ./...,
go vet ./..., scripts/loop.sh --check all pass (18 current judge tasks).
Loop script matches installed v1.1.1 byte-for-byte. These are current baseline
regressions, NOT execution of a new complete 55-row conformance runner.

## Reconciliation and remaining gaps

- P04/P05 concern assertion registration AND explicit permission for existence and
  particular states. Registration alone is neither publication-state authority nor
  observation permission. Preserve existing ledger semantics, independently test
  the host's registration/state decision and downstream observation denial.
- P06: forbidden probes must create neither key nor observation stream and disclose
  no existence. Authorize before resolving a cursor, planner lookup or diagnostics.
- O2 requires a certified snapshot handoff, not only 3B1's end-of-page barrier.
- O3 requires integrity-verified bound cursors, retained complete successor chains,
  explicit cannot-resume and certified replacement. Existing Prepare always chooses
  latest, Read rejects positions older than that plan: it cannot implement resume.
- Current Gate's mutex covers Check AND the actual synchronous recipient callback;
  preserve it. Grant revocation and retiring a single reconnecting session are
  different actions: reconnect must not revoke other sessions sharing that grant.
- Stage 3A append-only capacity is valid retention, not a license to fabricate
  pruning/crash recovery. Its private Plan ID is never an observer cursor.

No normative contradiction identified. The choices below are local design and
compatibility decisions requiring approval, not proposed upstream changes.

## Proposed design (names descriptive, not frozen signatures)

1. Add a recoverable session API alongside unchanged 3B1 API. Its typed envelope
   carries snapshot/update/barrier/replacement purpose, authenticated scope, local
   session fence and opaque cursor/proof. Snapshot receives its own handoff proof;
   barrier certifies only actually released contiguous coverage. Replacement proof
   is distinguishable from ordinary updates. No transport or shared encoding.
2. Inject cursor issue/verify authority separately from registration/rendering and
   current disclosure authority. Verified claims bind observer, complete view identity,
   model/schema, descriptor/authorization epoch, boundary record identity/equality,
   certificate purpose and authority-controlled validity. No caller Certified flag.
   Unverified tokens have a strict byte limit before parsing. Test authority must
   reject modifications/substitution; use real test-only authenticated encryption,
   not an always-success verifier. Production token format/keys remain host-owned.
3. Extend journal with an atomic retained-position planner and privately owned,
   idempotently identifiable plan acquisition. Validate the requested boundary against
   actual durable records/checkpoint and retained successor availability under the
   same owner lock/transaction, then pin it. New private entry kind(s) preserve old
   entry validation; no rewriting existing records/databases. Old readers fail closed
   on new kinds; rolling downgrade requires review. Existing methods remain valid.
4. Keep physical append-only storage and capacity refusal. Inject a trusted replay
   window/expiry policy to refuse expired positions even if historical bytes remain;
   logical retention expiry is distinct from corruption and physical pruning. No
   pruning added. Missing/conflicting durable state is integrity/unavailable, never
   automatic cannot-resume plus a falsely healthy reset. Clock/expiry policy must be
   explicit and tested; as-of freshness does not become a live authorization lease.
5. Renew authorization before token resolution/existence checks. Resume validates
   certificate and durable anchor, pins the retained start, releases bounded complete
   successor pages and issues a new finite barrier/cursor. Verification does not
   establish client application. Invalid/expired/incompatible token returns an opaque
   cannot-resume disposition only where authorized; explicit resnapshot separately
   authorizes and produces a certified replacement. No silent fallback or gap clearing.
6. A recoverable owner slot has one active session. Reconnect/identity switch cancels
   and joins the prior session before activating its replacement; independent slots
   remain bounded. No silent revival of a retired session. Reuse the SAME Gate fence
   for every protected envelope, including cursor and reset diagnostics. Internal
   factoring of Gate may be needed, without changing its public signatures.
7. Journal ownership, disclosure policy and token keys have distinct jurisdictions.
   Private cleanup authority is not revoked merely because an observer loses access.
   Track only this recovery owner's plans; reconcile uncertain acquisitions by their
   durable identity after reopen, never release unrelated Stage 3A user pins. Startup
   retires old volatile sessions and checks independent ownership/policy authority;
   never reconstruct permission from rolled-back SQLite state. Valid tokens survive
   reopen only with independently preserved trusted issuer keys/policy and history.
8. Bound sessions, page count/bytes, token size, pins and synchronous callback duration.
   Reserve enough storage budget for owned plan release when admitting plans/future
   entries; otherwise full append-only capacity can permanently block cleanup.
   This justified journal capacity change requires old-state compatibility checks:
   insufficient existing cleanup headroom must produce explicit recovery/capacity
   refusal, not discard accepted history. Failed cleanup stays owned/retryable;
   unknown commit requires reopen/reconciliation, not guessed rollback. No orphan
   timeout goroutines or unaudited background work. Limits are explicit configuration,
   not guessed production defaults.

Rejected shortcuts: in-memory-only cursor registry, exposing Plan IDs, treating a
current-head snapshot as replay, using ordinary updates to clear gaps, swallowing
corruption as expiry, raw hidden checkpoints in readable tokens, and a permission
check followed by an unlocked callback.

## Requirement -> independently judged evidence plan

| Source | Observable test | Owner |
|---|---|---|
| P04/P05/P06, A01/A06 | independent registration/state/disclosure decisions; forbidden/nonexistent probes indistinguishable; no creation/source diagnostics | trusted host + gateway |
| O01/O02/O14 | prior independent revision/public status/null regressions, plus complete envelope transitions | journal + gateway |
| O03/O04/O05 | SQLite snapshot40 -> concurrent41 -> handoff/update/barrier; interrupted scan, bounded page and no partial certificate | planner + session |
| O06/O07/O08/O11 | resume43 -> 44/45/barrier, retired callbacks fenced; disconnect unknown, gap/conflict cannot advance | recovery slot + independent recipient harness |
| O09/O10 | expire token/window -> cannot-resume -> certified replacement90; wrong observer/view/model/schema/generation/purpose and modified tokens denied | verifier + planner |
| O12/O13 | delivery certificate leaves backend incompleteness and as-of freshness unchanged | certificate/recipient oracle |
| O15/A03/A04/A05 | both callback/revoke orders with channels; reconnect/close/cancel fencing, clear owned buffers, cleanup retry | release gate + slot/harness |
| A02 | hidden accepted checkpoint changes do not expose raw positions or alter public revisions/cursor continuity | renderer + token authority |
| V01/V02 | explicit trusted descriptor compatibility, change requires certified replacement; no reinterpretation | descriptor authority |
| B01 | session/pin/page/token/cleanup budget exhaustion; no wrap/drop/leak or false barrier | journal + owner |

Execute full logical transitions, not just final values. Test snapshot and resume with
actual SQLite reopen; add controlled subprocess cuts before/after plan persistence,
after possible disclosure, and across token issuance with independently retained
issuer keys/policy. Distinguish recoverable interruption, unknown commitment and
corrupt storage. No claim of atomic SQLite commit plus arbitrary external output.

Vector accounting will enumerate all 55 IDs with exact test/subcase anchors and
executed-pass/executed-fail/unexecuted status. Client-only cache/application and S
submission rows remain unexecuted unless their full oracle is actually exercised.
Partial related regressions are not full-row passes. Current-review status is in
OBSERVATION-3B-VECTOR-STATUS.md; no new vector execution is claimed yet.

## Build/acceptance plan after approval

- Freeze only the reviewed API/contracts and test fixtures. Extend protected paths,
  preserve existing tests and unchanged judge. First test actual replay-after-reopen
  and revoke-before-token-release schedules to disprove unsafe designs cheaply.
- Independent private reference outside checkout; all new tasks fail on stubs and
  all tasks pass under exact scripts/loop.sh --check with reference. Restore stubs,
  commit verified clean kit, then bounded loop (proposed MAX_ITER=12 MAX_ATTEMPTS=3).
- Progressive deterministic/failure/property tests, process cuts, full Go/race/vet,
  all judge tasks, diff/ownership checks and explicit 55-row coverage report.
- Stop on genuine normative ambiguity. Do not report completion of Stage 3B until
  authorized resume, replacement and owned cleanup meet the tests above.

## Minimum future Go submission contract (not implemented in observation stage)

For TypeScript Stage 2B: trusted descriptor/capabilities, Submit(event ID, immutable
semantic event + authenticated attribution), Resolve(same ID), and optionally owned
status watch/wait. Return generation/versioned full facts separating durable receipt,
admission pending/admitted/rejected, model/evidence-specific evaluation, conflict,
expired/no-retained-knowledge and transport-unknown. Define equality, retry/dedupe
horizons and authority-controlled recovery so unknown/expiry never authorizes a new
identity or reexecution. Projection revisions/barriers are NOT submission acknowledgments.
Cancellation stops local waiting, not admission/effects. No HTTP or shared wire chosen.
This is a minimum semantic proposal, not a claim about the current TypeScript adapter.

## Approval requested

Approve this additive recovery API, purpose-bound opaque proof authority, retained
position/owned-plan journal extension, logical retention policy, cleanup reservation
and old-reader fail-closed compatibility boundary. Approval permits freezing the kit
and proceeding with independently verified local commits and bounded coder loop under
the established process. No production AUTH, upstream mutation, transport, effects or
TypeScript modification is included.
