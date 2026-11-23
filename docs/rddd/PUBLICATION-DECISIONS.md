# Q1/Q5/Q6 reconciliation — recommendations for owner review

## Status and provenance

Spec-loop v1.1.1 step 2a only. These are recommended resolutions, NOT approved
shared semantics, published upstream amendments, or frozen Go/wire contracts.
No code, dependency, kit, commit or model loop in this assignment.

Go: /Users/rob/Development/perhap-rddd-foundation, origin Version2beta/perhap,
branch rddd-foundation, HEAD 758322024fab69cea3fbae7dfd471fe78adbae7a.
The four existing PUBLICATION documents were uncommitted and are preserved;
only the stage plan receives an additive reconciliation update.
Spec: committed clean candidate draft.6
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; all 17 checksums verified.
Candidate is not ratification. See existing requirements for manifest hash.

TypeScript checkout actually inspected: /Users/rob/Development/perhap-client,
origin https://github.com/Version2beta/perhap-client.git, rddd-foundation,
HEAD c09645c1cd4874e07f370541ab43b5260b560efe. The interoperability documents
are UNCOMMITTED proposals; HEAD alone does not identify their contents.
SHA256 of compared local snapshots:
- INTEROP-REQUIREMENTS.md: f96214769c2b8f6c7a1e3109dd90aef00ac62289a60ee02303d05fe353148058
- INTEROP-DESIGN.md: 8e551b8f54ca4701df385dd9bf9cd2ac2fcd9504fa996ad4faf3c51e273aef72
- INTEROP-FAILURES.md: 2abd4b2ee2c68893521acee7da59d15cfd2120a56f856e0da5fefeeba5046269
No approval or agreement is asserted on behalf of its author. No external files
were changed. PUBLICATION-SHARED-PROFILE-DRAFT.md contains the reviewable proposal.

## Comparison

| Question | Draft.6 basis | TypeScript evidence | Recommendation / missing guarantee |
|---|---|---|---|
| Q1 one sequence including statuses | §5 durable ordered producer/key revisions and status continuity; §9 independent scoped projection revisions | requirements R1/R5/R7/R9; design successor/generation and status-only rules; failures duplicate/restart/gap rows | Compatible intent. Starting at 1 and a single content/status stream are added profile constraints, not already mandated field encodings |
| Q5 every accepted evaluation checkpoint | §§4.3,5 complete valid versus unresolved, unchanged content no new publication | R6/R8/R9 need trusted backend evidence; design explicitly says contiguous delivery is not evidence of complete backend history | Compatible but client proposal does NOT define producer acceptance/checkpoint ordering. Add backend proof/precondition and retry contracts; do not treat caller numbers as evidence |
| Q6 initially absent, not retract | §5 retraction when a prior conclusion ceases to be supported; availability separate from null | R9/R10 and explicit retract target; design local union lists current/unresolved/retracted but not never-published absence | No stated disagreement; initial absence/authorized observable initial unresolved are missing shared cases. Recommend explicit lookup presence, not forcing all projections into an assertion-status union |

Important non-conflicts:
- A complete producer/key publication stream starts at 1. TypeScript's certified
  projection snapshot S/40 may legitimately start observation at 40; it is not a
  claim to have received producer history 1–39. Publication, projection, submission
  disposition version, evaluation checkpoint and opaque resume cursor are distinct.
- TypeScript recommends full public projection replacements, not REV historical
  event transport. Do not deliver assertion deltas to that API as if they were
  projection replacements. Historical events remain immutable; current assertion
  changes never automatically undo business transitions.
- Unknown/disconnected continuity is not automatically incomplete backend evidence.
  Expose as-of barriers, not an unlimited live-freshness promise. A no-op evaluation
  need not notify a client unless its public projection/status changes.

## Q1 — accept with explicit identity scope

Recommend one contiguous sequence for each immutable scoped derivation identity
(context, accountable producer/service, subject, meaning). Content upsert,
retraction, unresolved status and same-content recovery all allocate from that
sequence. First PUBLICATION is 1; never-published internal head is no publication,
not a revision-zero event. No-op evaluations allocate no publication revision.

Each committed publication has a stable globally unambiguous event identity and
immutable meaning. Persist it once, then retry the same identity/content through
restart and lost acknowledgment. A durable request identity is distinct from that
publication identity; identical request retry returns the original durable receipt
and cannot allocate a second ordinal or checkpoint. Changed request meaning conflicts.
Link each later publication to the immediately prior publication in the stream,
including status revisions; optionally carry a separately named prior-content link.
Never overload historical causation with either link.

Restart resumes identity/sequence, not an epoch reset. Ownership transfer preserves
continuity. A declared namespace/generation change creates a DISTINCT identity and
requires explicit certified rebootstrap/migration; it cannot silently reuse old
(publication stream, revision) pairs or reuse an event ID. Restoring a stale backup
cannot establish safe continuation without authoritative recovery evidence.

Counterexample: value r1, unresolved r2, recovered same value r3. Ignoring statuses
as revisions makes recovery either invisible or an apparent missing revision.
Tests Q1-1..4 in the profile cover this and retry/reset conflicts.

## Q5 — separate acceptance ordering from source freshness

Every newly accepted evaluation durably advances an acceptance checkpoint, even
if it changes no public content/status. Rejected/stale/denied evaluations and
idempotent replay of the same request do NOT advance it. Checkpoints order accepted
evaluations; they are not proof that the source evidence itself is up-to-date.

Acceptance checks BOTH expected assertion head (ordinal + identity/presence) and
expected prior accepted evaluation checkpoint, plus trusted current source/rule/
admission generation for the relevant evidence closure. A monotone caller integer
or UUID timestamp cannot establish any of these. The host must bind evaluation
inputs, rule version, source boundary and owner to evidence it can validate.
An old rule version with a larger client number is still stale.

For a single-owner stage, propose a coherent host-owned source-generation token
checked at the ledger transaction's acceptance point, serialized with relevant
source advancement; acceptance increments its own checkpoint. If the source lives
elsewhere, require an explicit fence/proof spanning that boundary, or stop/recompute
rather than promise atomic source freshness. Do not compare arbitrary frontier
strings or infer total order over unrelated scopes/vector clocks. Incomparable or
unverifiable source evidence is not accepted as certified fresh.

A new attempt with updated preconditions must actually recompute/revalidate; do not
relabel an old result with a newer ExpectedRevision/checkpoint. Each acceptance
atomically records the receipt, checkpoint, and any head/publication/outbox change.
A complete valid no-op still persists the checkpoint and receipt. A newer failed or
incomplete evaluation may be accepted as unresolved evidence; accepting it is NOT
certifying coverage. Older evaluations cannot override that unresolved state.

Counterexample: E1 evaluates value V; E2 accepts newer evidence with same V and no
publication. E1's publication head still matches. Without the checkpoint/source
check, E1 can later retract based on obsolete evidence. A second counterexample is
source admission advancing after evaluation but before acceptance: CAS on the
publication head alone detects neither race.

## Q6 — explicit presence and evidence, not a fabricated zero assertion

A never-published key is absent. Complete valid absence records the accepted
evaluation/receipt/checkpoint, but no initial retract and no publication revision.
Initial invalid/incomplete/failed evidence does not invent a key's existence or
previous value merely because a callback returned an identifier.

Initial unresolved publication is permitted only when trusted domain rules and
assertion authority establish the key's existence AND a policy permits observing
that unresolved status. It uses revision 1, unresolved, value unavailable, with
no fabricated prior publication or stale content. Permission to publish metadata
is not permission to publish a value or protected existence. If those grounds are
missing, withhold the public assertion; record internal evaluation evidence only
where separately authorized. Unknown authority is not fabricated denial/acceptance.

Later complete present content after initial unresolved becomes a new upsert,
not a same-value recovery from a nonexistent value. Complete absence after an
already published unresolved/current key can retract at the next ordinal.
Once published, retracted remains an explicit tombstone, not never-published
absence; storage expiration must never silently erase that distinction.
Available encoded null is a value, not absence or a retraction.

Current rev.Reconcile takes a previous Assertion value but does not encode
never-published existence or initial observability proof. The durable boundary
must supply that lifecycle distinction BEFORE invoking/persisting its result.
Do not persist a zero-Assertion-derived retract intention blindly. No production
semantic change or new exported enum is authorized by this recommendation.

## What belongs upstream / profile versus implementation

Recommend shared semantic clarification/vectors for status-inclusive continuity,
absence versus retract/unresolved, stable retries, reset identity and trusted
acceptance freshness. Profile choices include first ordinal 1, one combined stream,
exact predecessor relation and retry/checkpoint contract. These strengthen a
selected interoperability profile without claiming every draft.6 implementation
already follows these local choices. SQLite, schema, numeric width/codec, Go
interfaces, transaction mechanism and transport remain outside shared semantics.

## Gates and stop point

No observed conflict with the inspected TypeScript proposal; Q5 and bootstrap
semantics were not fully specified there. Recommendations are NOT cross-repository
acceptance. Request owner/upstream review and matching shared vectors before kit/API
freeze. Any different TypeScript decision on identity/reset/status/absence blocks
the affected implementation rather than being patched with a local exception.
Q2/Q3/Q7 (reader freshness, authorized resume/filtering, revocation) remain gates
for external publication/observation, not solved by Q1/Q5/Q6.

To begin the single-owner SQLite ledger stage, explicitly approve:
1. These Q1/Q5/Q6 semantics and source-of-truth profile version/vectors.
2. The trusted host source/checkpoint/owner model and initial-key observability rule.
3. SQLite driver/dependency/version, durable configuration, filesystem assumptions,
   exclusive-owner/crash recovery/backup refusal contract and bounded backlog.
4. Stage scope: local ledger only; no transport, live AUTH/EFFECT or client changes.
5. Go API design and kit preparation/verification authority; any later model calls,
   spending, baseline commits and attempt discards need explicit authorization.
6. A plan to preserve all currently uncommitted documents before a clean-start loop.

Smallest task breakdown after those approvals:
L1 owned store open/close, transaction/reopen and exclusive ownership;
L2 durable idempotent receipts and atomic checkpoint/head/publication/outbox;
L3 stale-evidence/head/checkpoint rejection plus no-op/bootstrap/status transitions;
L4 subprocess crash/fault/retry recovery and bounded resource failures;
L5 full earlier CORE/REV and Go/race/vet acceptance.
Tasks are proposed, not a tasks.json or authorization to run a loop. L1's first
experiment should prove committed bytes survive process restart before abstractions.
