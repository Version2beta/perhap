# Binding Publication Stage 3A kit — durable projection journal

## Pin, scope and translation

Baseline draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
ratified INTEROP-PUBOBS/0.1 v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e. All 19 checksums pass;
manifest 01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
The four prior OBSERVATION documents are preserved unchanged as planning history;
this bounded kit realizes only 3A. No normative contradiction found in this scope.

New package sdk/go/viewjournal only. Preserve CORE, REV, publication/delivery code,
APIs, storage formats and tests. No observer session, cursor, signed certificate,
network, disclosure, revocation, production AUTH/effects or cross-owner transactions.
No new dependency. Completed driver mattn/go-sqlite3 v1.14.32 requires CGO/compiler.

## Ownership and selected local design (established before kit freeze)

One SQLite owner, one persistent connection: DELETE rollback journal, synchronous
EXTRA, fullfsync ON, locking_mode EXCLUSIVE, zero busy timeout, exclusive transactions.
Acquire real exclusive lock before Open succeeds; process crash releases it. Check
actual pragmas/integrity. Existing empty/unknown/corrupt store must never be reset.
Same local filesystem/fsync/private-directory/stable-path assumptions as Stage 1;
no hostile path aliasing, stale backup restore without independent authority, media
loss or power-cut proof. Process-exit tests are not those guarantees.

Prefer an additive journal over mutating the assertion ledger: projection equality,
checkpoints/revisions and authorized views have distinct ownership and lifecycle.
Prefer typed batch calls over implicit goroutine/session state. Use a bounded
transactional state containing immutable accepted evaluations/receipts plus durable
private plans; an equivalent append-log/schema is permitted if all invariants hold.
No SQL schema is a shared protocol. Private JSON persistence is not a wire format.

One registered deterministic model is pair-sum/v1 with nullable-sum/v1 public schema:
trusted host evaluates two assertion inputs under ONE locked finite cut and renders
an optional typed Total int64. Multiple input changes can yield one result or no
material result; the journal never reads independent databases to invent that cut.
Host owns input range/domain validity and null/unavailable meaning. No generic
cross-domain projection engine added. Parameters are already normalized by host.

Identity binds Context, Projection scope, normalized Parameters, Model, Schema,
AuthorizedView and durable Generation. Every field matters; wrong registered
Model/Schema rejected. No sharing of revisions between authorized views. No IDs
are inferred from publication ordinal, UUID or the host's private cut.

Public equality compares the complete typed record, not pointer addresses: nullable
Value, Available, Evaluation, Completeness and permitted Evidence. Available true
with nil Value is available null. Available false requires nil Value. Evaluation
valid/invalid/failed and completeness complete_as_of_frontier/incomplete/unavailable
are independent. Evidence is an approved public reference, NOT PrivateCut. Changing
private source generation alone cannot manufacture public revisions or modify older
public provenance. Changing any material public field must allocate a revision.

## Trusted Host contract and lifecycle

Host.Authorize validates registered identity/normalization and private handle access,
including receipt retries, snapshot plans, plan enumeration and reads. Heads is a
private recovery API owned by the trusted host, not a public view enumeration.
Host.Within validates EvaluationID against current inputs/rule/model/schema/policy,
returns immutable Verified public rendering plus PrivateCut, and serializes relevant
source/authority updates through callback commit. It calls fn exactly once or not
at all, propagates its error, and never reenters Journal. Request fields/flags are
not authority. Host.Recover checks complete persisted head inventory and generations
against independent authority state not rolled back with this DB; reject stale or
missing inventory. Test authorities prove only the specified local cut scheduling,
not real authentication or distributed freshness. These assumptions are essential.

Caller owns Journal.Close, contexts and private plan handles. No async work or
worker lifecycle. Return copies/owned typed values; callers cannot mutate durable
records via returned pointers. Closed/poisoned handles refuse authoritative work.
Use error sentinels for conflict, stale, unavailable/corrupt, unknown commit,
capacity, competing owner and denied/unresolved authority. No public error protocol.

## Acceptance and recovery

Scope request identity by Identity plus Request.ID. Exact retry returns its original
Receipt, even after head advances/reopen; changed immutable Request conflicts.
Authorize before lookup; retry does not require old evidence to still be current.
New request checks expected private checkpoint AND public revision plus trusted
Within evidence. Every accepted evaluation advances checkpoint, including failures
and no-ops. Material public changes allocate first revision1 then contiguous successor;
receipt says Changed false for no-op. No public record before first accepted result;
Prepare on an unevaluated view returns unavailable, not fabricated zero or baseline.

Atomically persist receipt, private evidence checkpoint, any public record/head and
recoverable replay obligation. The obligation is retention of every immutable public
record in this append-only stage, not a network outbox. No pruning/coalescing; new
acceptance at MaxEvaluations or MaxBytes capacity refuses rather than discard history.
Checkpoint/ordinal overflow refuses before mutation. MaxBytes bounds stored state;
recovery validates size/count/identity/checkpoint/ordinal/equality and plan consistency.
Missing/invalid records cannot produce authoritative empty hydration.

Cut before_commit runs after writes inside transaction, after_commit only after
successful Commit. A known precommit hook error rolls back/unavailable; after commit
error is unknown and poisons handle. Uncertain Commit cannot produce success or
assert rollback. Close/reopen/identical retry resolves retained outcome. Hooks are
controlled test seams, not arbitrary external side effects or environment branches.

## Snapshot/tail planner, pins, finite cuts

Prepare atomically persists a private Plan with unique ID, exact View and full
latest public Snapshot. Its successor position is Snapshot.Revision+1. Active plans
count against MaxPins, survive restart and are enumerable via Plans after a lost
planning reply. These are NOT observer sessions, capabilities or signed cursors.
Plans/Heads never imply observation application or public permission/freshness.
Release removes a plan durably, idempotently; no accepted history is deleted.
No expiry/time/retention-pruning mechanism in 3A; append-only history plus explicit
capacity refusal protects every plan's tail, including across restart/pressure.

Read(planID, after, max) requires a live persisted plan, current host authorization,
Snapshot.Revision <= after <= committed public head, 1 <= max <= MaxBatch. The local
owner supplies after; no client-applied proof is inferred. Select a finite public
head at the read cut and return at most max contiguous successors through it. Batch
After/Through/SelectedHead and records describe ONLY (After,Through] coverage. If
Through < SelectedHead the batch is partial catch-up, not a barrier through head.
An empty batch may describe (after,after] at a verified existing head; no head can
be invented. A future 3B session must prove the client predecessor/chain and authorize
release before turning this data into an externally certified delivery barrier.

No network subscription activation is needed to retain revision41 committed after
snapshot40: it stays in durable history. The snapshot plan never relocates after
later acceptance. Interrupted/canceled/corrupt read returns error and zero Batch,
not partially populated certificate intentions. during_read hook tests this boundary.
Model/schema/authorized-view/generation cannot be crossed by a plan. Unknown/released
plan refuses. No cursor issuance or revocation ordering is implemented.

## Requirement-to-test mapping (profile subsets only)

| Requirement/vector | Focused test |
|---|---|
| O03 local atomic snapshot/tail, B01 retention pressure | TestViewSnapshot40Tail41: seed40, durable plan, reopen, channel-ordered commit41 before tail read; max capacity refuses42 but retains41 |
| O01 independent projection identity, hidden/no-op O1/A02 local rendering | TestViewEqualityFreshnessAndIdentity: two assertion values sum, no-op private checkpoint, independent authorized views |
| O02 status-only revision and stale acceptance P2 | same test: failed/recovery with unchanged sum; old checkpoint/source rejected; concurrent one-winner |
| O14 null versus unavailable; O04/O05 bounded/interrupted scan | TestViewNullBatchesAndInterruptedScan: finite pages, no partial intention on error/cancel |
| P12–P15 adapted to projection acceptance; O05 recovery | TestViewUnknownCorruptAndOwnership, TestViewRealCrashRecovery: actual os.Exit before/after commit, reopen receipts and state, competing child owner, truncation/corruption refusal |
| O03 local durable handoff pin recovery | TestViewPlanCrashRecovery: before/after pin commit subprocess, recover/enumerate pin and consume later successor |

Not claimed: complete O01–O15 observation service, signed O2/O3 certificates,
A01–A06 production disclosure safety, V negotiation, client continuity/application,
retention-expired cursors or cross-backend interoperability. Prior shared test suites
remain regression checks. Cheapest experiment must run first against real SQLite.
Future stage 3B: bounded authorized in-process sessions, release fencing, barrier/
cursor authority and resume/reset atop this private journal under separate approval.
