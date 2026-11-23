# Publication stages and failure/recovery test matrix

Design only. No implementation run, exported API/wire freeze, dependency changes,
commits or paid calls authorized here. See the other three PUBLICATION documents
for source pin, requirements U1–U11, proposed P1–P4, and unresolved contracts Q1–Q7.
Tests below are proposals, not executed evidence. Completed CORE/REV are preserved.

## Q1/Q5/Q6 reconciliation update — still awaiting review

PUBLICATION-DECISIONS.md compares the proposed resolutions against draft.6 and
actual uncommitted TypeScript proposals. PUBLICATION-SHARED-PROFILE-DRAFT.md is
an upstream/profile review draft, not an adopted requirement or wire schema.
No direct disagreement found; producer checkpoint/bootstrap details were missing
from the client proposal, so compatibility is not evidence of cross-team approval.
The original matrix and follow-on stages below remain proposals and are preserved.

First ledger stage now explicitly needs:
- One status/content/retract sequence from 1; no revision for a no-op; immutable
  retries and explicit non-reused namespace/continuity identities.
- A durable checkpoint for each NEW accepted evaluation, including no-ops;
  identical request replay does not advance it. Check head AND checkpoint AND
  trusted current source/rule generation at acceptance, not a client counter.
- Never-published presence separate from retracted; initial absence emits nothing;
  initial unresolved needs domain existence/observability grounds and no fake value.
- Atomic receipt/checkpoint/head/publication/outbox, exclusive ownership and real
  SQLite reopen/process-crash tests. No outgoing transport or client changes.

Approve the profile/vectors, trusted host/fencing and bootstrap policies, SQLite
driver/version/durable configuration/dependency, resource and backup assumptions,
Go design, and separate future kit/loop authority before implementation. Preserve
all uncommitted documentation before any clean-start loop; no commit authorized now.
Q2/Q3/Q7 remain external disclosure/observation gates, not solved here.

Smallest proposed Stage 1 spec-loop breakdown after approval:
1. L1 — owned SQLite lifecycle, exclusivity, transaction/reopen durability experiment.
2. L2 — atomic acceptance and idempotent receipts/checkpoints/head/history/outbox.
3. L3 — stale source/head/checkpoint rejection, no-op and bootstrap/status transitions.
4. L4 — subprocess crash, duplicate/conflict/restart and bounded fault recovery.
5. L5 — all previous CORE/REV, full Go/race/vet and diff/artifact acceptance.
These are not executable tasks or frozen APIs. No kit is constructed.

## Review gates

No contradiction found between the proposed separation and draft.6. Missing
contracts are not resolved by treating existing NATS code as normative. Before
preparing a kit, approve Q1 (stream/status and key scope), Q5 (evaluation freshness
and no-op checkpoint), Q6 (bootstrap), single-owner/recovery contract, storage
engine/dependency and actual durability target. Before network publication approve
Q2/Q3/Q7 (reader freshness, resume/filtering and revocation/disclosure boundary).
Send shared observable rules/vectors upstream for review, without modifying
rDDD-spec here. Freeze neither a message schema nor an exported Go API meanwhile.

## Smallest next implementation stage: single-owner transactional ledger

A local host can durably accept a verified reconciliation result, read its receipt,
restart, and enumerate pending immutable publications. NO network dispatcher,
broker, reader subscription or real AUTH/EFFECT enforcement. Inputs are trusted
host evidence through explicit test seams; this does not confer a runtime AUTH
claim. The ledger's files are sensitive and private to its authorized owner.

Deliver one coherent store boundary: transactionally persist the request receipt,
validated evaluation checkpoint, per-key head, publication and outbox reference;
handle accepted-no-change without allocating a revision; support lookup and
reopen/recovery; reject conflicting IDs/stale owner or evidence generations;
bound backlog/disk failures. Do not introduce leases or distributed ownership
when enforced single ownership suffices. Do not use an in-memory store to claim
durability. SQLite is recommended subject to dependency/configuration approval;
choose the cheapest real process-crash experiment before infrastructure work.

Proposed implementation tasks after approval: store schema/open-close and exclusive
ownership; atomic acceptance/idempotent receipts; transition/checkpoint validation
and revision allocation; pending ledger queries/recovery; failure-injection and
whole-tree acceptance. Exact signatures and task count follow design review.
Use existing pure REV functions as calculations, never as persistence success.
Stop at a usable reopenable ledger; no outgoing disclosure in this first stage.

## Follow-on stages

2. Ordered dispatcher to a representative accepting sink: persistent pending state,
   same-ID retries, unknown outcome lookup, per-key flow/backpressure. Process-kill
   tests with a real receiving ledger before claiming end-to-end admission.
3. Authorized read/catch-up gateway: snapshot/cursor consistency, retention repair,
   live freshness contract, policy change/revocation and metadata non-disclosure.
   No externally accessible consumer feed until actual enforcement is established.
4. Optional JetStream binding: declared durability/retention/consumer settings,
   disconnect/restart tests, expired dedupe windows, broker receipt translation,
   disclosure gateway and ACL integration. Do not change pure CORE/REV semantics.

Stages 2–4 are not independent permission to publish or access live systems.
Multi-owner failover, cross-domain cascades, CLIENT changes and consequential
issuance need separate explicit scope and validation. No full REV/AUTH/EFFECT
claim follows from stage 1's local ledger.

## Failure/recovery matrix

Cut points are controlled hooks or real subprocess termination at specified
boundaries, not sleeps. Verify records by reopening storage independently.
For any committed request, assert one immutable publication identity and at most
one head transition; multiple physical deliveries remain permitted.

| Scenario / injected failure | Required observation after recovery | Stage |
|---|---|---|
| Before persistence begins | no receipt/head/publication/outbox change; same-ID retry may commit | 1 |
| After head write, before transaction commit | crash leaves all-or-none head/receipt/history/outbox/checkpoint; never partial | 1 |
| Commit durable, response lost | lookup/same request returns identical receipt and publication; no second ordinal | 1 |
| Ambiguous commit error | report unknown, reconcile by durable receipt; never guess rollback | 1 |
| Disk full/fsync/transaction failure | explicit error/unknown as appropriate; no false durable acceptance or forgotten accepted item | 1 |
| Corrupt/truncated store or failed recovery scan | refuse authoritative operation; no empty fresh ledger fallback | 1 |
| Duplicate request, changed command/key/evidence | immutable-meaning conflict; no overwrite even if head changed | 1 |
| Concurrent commands same expected head | at most one advancing commit; other is stale or idempotent prior receipt | 1 |
| Old valid evaluation after newer no-op/failure | evidence-generation check rejects stale result despite unchanged content revision | 1 |
| Lost/expired owner or attempted second owner | storage-enforced exclusion/fence; stale worker cannot advance head | 1 |
| Reopen after each status/content transition | ordinals and supersedes survive; unresolved then same-value current both observable | 1 |
| New never-published key: absent or failed | agreed Q6 behavior, no fabricated prior value/retraction; zero versus encoded null distinct | 1 |
| Valid complete disappearance / invalid or missing evidence | respectively retract / unresolved; historical records retained in both cases | 1 |
| Process killed after persistence, before send | pending item recovered, original ID/ordinal/content reused | 2 |
| Send failed before sink acceptance | pending, bounded retry; no accepted receipt fabricated | 2 |
| Sink accepted, response lost | same immutable record resent; sink admits one fact | 2 |
| Broker accepted, local delivery receipt not committed | retry original record; no new logical publication or lost pending work | 2/4 |
| Consumer crashes after admission, before ack | redelivery deduped; atomic admission/checkpoint protects resumption | 2/3 |
| Delayed duplicate arrives after a later revision | dedupe historical fact; do not roll back current view or interpret it twice as new | 2/3 |
| Revision r3 arrives before r2 / conflicting r2 | affected key incomplete; bounded repair/buffer; no last-writer-wins | 3 |
| Pending item permanently denied / backlog exhausted | hold and expose authorized delivery-unavailable/backpressure; no skip to later ordinal | 2/3 |
| Snapshot overlaps a new commit | snapshot plus tail has neither lost change nor duplicate admitted effect; stable cut | 3 |
| Cursor expired, scan interrupted, consumer disconnected | explicit repair/unavailable/incomplete per evidence; no false up-to-date claim | 3 |
| Filtered keys and changed observer policy | no hidden key/count/offset leak; invalidated cursor rebootstrap is authorized | 3 |
| Revocation before acceptance / between persistence and send / during replay | protected operation denied or held at specified disclosure fence; no new unauthorized payload/metadata | 3/4 |
| Policy changes after irrevocable send | do not claim recall; demonstrate exactly which disclosure preceded revocation and fence later sends | 3/4 |
| Retry beyond JetStream duplicate window / broker restart | stable ID ledger still dedupes; configured broker receipt semantics verified | 4 |
| Same max input UUID, newer projection revision | consumer accepts correction based on scoped projection protocol, not UUID | 3/CLIENT |
| Ledger backup rollback or producer migration | refuse to reuse identities/ordinals under same continuity claim; approved epoch/repair procedure required | later |

Extend shared REV vectors for status sequencing/recovery, duplicate durable
acceptance, gap repair, disconnected freshness, snapshots, and denied metadata.
Use local engine-specific crash tests for atomicity/fsync rather than prescribing
SQL to rDDD. Negative cases should check absence of unauthorized persistence or
disclosure, not merely an error return.

## Future spec-loop verification and acceptance

After separate approval: subordinate stage SPEC with source pin and mapping,
protected contracts/tests/fixtures, failing stubs, private independent reference,
and unchanged v1.1.1 judge. Every new task fails on stubs and passes with reference
through scripts/loop.sh --check with strict complete-line passing-test expectations;
preserve all earlier tasks. Mocked transactions do not satisfy durability tasks.
Use temporary owned databases and bounded subprocess crash tests, not live systems.

Proposed checks: focused failure matrix, repeated restart/concurrency cases,
go test -count=1 ./..., go test -race -count=1 ./..., go vet ./..., earlier
CORE/REV fixtures, existing frontend/Node checks, artifact and diff inspection.
Require evidence of committed bytes after reopening and of no outgoing network
calls in stage 1. Keep command/source/engine/configuration/crash-cut receipts.
A later transport stage requires real local transport integration in addition.

## Evidence for this design assignment

Repository/upstream identity and clean initial state checked; all 17 upstream
checksums verified; source inspected: CORE/REV contracts and evaluator design,
REV reconciliation/reconstruction implementation and docs, NATS supervisor send
and replay paths, SPEC §§5–10 and conformance scope. No software tests run for
these documentation-only additions. No runtime/durability/security verification
or independent approval claimed. No production, TypeScript, Elixir, upstream or
wire-format changes; no commit, push or loop.
