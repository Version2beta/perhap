# Gateway Submit/Resolve — final evidence and adapter boundary

Implementation state: 3fa5769aeef9e0add7bda9a84f6d7eeab9c29484, rddd-foundation.
Approved separate SQLite authority applies ONLY to gateway-originated events.
No new stage number or normative semantics. Legacy JetStream remains outside it.

## Commits / independent process

- Kit 06a18d150a62038c79591756a60b51c871d5929a.
- Initial judged implementation 914dc720692bbb94039995df230a681181ee485c.
- Verified review overlay 1a945fc06b7a0c61eec321b0bae9e4e499c45346.
- Final judged implementation 3fa5769aeef9e0add7bda9a84f6d7eeab9c29484.

Original design preserved. Independent private reference outside checkout passed
all28 initial groups; stubs failed all5 new groups, prior23 focused groups passed.
Review regressions failed actual judged code and passed reference under all29 exact
judge groups before corrective baseline. Private reference removed, not committed.
Two coding iterations total, attempt1 each, pinned claude-haiku-5-5 command unchanged.
Initial MAX_ITER=12 MAX_ATTEMPTS=3; correction MAX_ITER=11 MAX_ATTEMPTS=3 retains
aggregate12 cap. No test weakened or earlier implementation/dependency changed.

Pins: draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; profile v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e; all19 checksums verified; manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
No upstream/client changes. Unchanged script compared to installed spec-loop v1.1.1.

## Final executable evidence

All commands exit0 at implementation state above:

```
go test -count=10 ./sdk/go/submission
go test -race -count=10 ./sdk/go/submission
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
scripts/loop.sh --check
git diff --check
```

Focused10 repetitions14.602s; race10 repetitions15.785s. All29 judge groups pass;
last group includes full Go/race/vet/Node/frontend suites. Actual subprocess exits
before/after receipt and admission commits recover exact pending/admitted sides.
Real committed-file truncation refuses recovery; independent minimum-version host
inventory rejects a stale backup. Competing SQLite owner refused. Counter codec
normalizes1.0 to1 before both identity comparison and actual CORE integer evaluation;
raw1.0 would fail that reducer. Later failed evaluation never undoes admission.
Missing required cause leaves target unattempted/not-established, not fake success.
No JetStream bridge is involved in these evaluations.

Review source corrections protect reserved record/byte completion space, envelope and
reopen bounds, optional correlation/codec-owned foreign links, canonical context errors,
opaque per-attempt evidence instead of private history digest, and release accounting
against late/double/error-swallowing host callbacks. Tests are behavioral boundaries,
not fixture-ID branches. Stage3A/3B source, prior ledger/delivery/Core/REV, all previous
tests and journal formats unchanged; only new submission package and kit/evidence.

Known separate test-suite risk: a preexisting NATS TempDir asynchronous cleanup race
occurred in TestSupervisor_EmitExceptionPublishesToJetStream during reference full-suite
validation despite existing Shutdown/WaitForShutdown. Source/fixture left unchanged.
Subsequent final full Go/race/judge passed. This is not proof the old fixture is free
of intermittent failures. Details/logs in SUBMIT-RESOLVE-KIT-VERIFICATION.md.

## Stable local API

`submission.Open(ctx, absolutePath, Host, Limits)` returns trusted Owner.
`Owner.Client(context)` returns the client-only facade:
`Submit(ctx, Input, Recipient) error`, `Resolve(ctx, EventID, Recipient) error`.
`Owner.Evaluate(ctx, EventID, attemptID) error` is a separate trusted capability;
`Owner.Close()` cancels/joins work and releases its owned file.

Input contains existing perhap.Event, registered Schema and requested AttributionBinding.
Host derives Requester/Asserting identity, Codec, Admission callback and actual Release
fence. Client cannot provide authoritative admission/evaluation flags. Registered codecs
are version-immutable trusted code, normalizing only Data. Gateway UUIDv6 identity is
canonical; no timestamp causal authority. Correlation may be empty; foreign causation
references are interpreted by the codec, not forced into gateway UUID format.

Recipient gets Reply(kind facts/no-retained-knowledge). Complete Facts identify EventID,
store status Generation/Version, durable receipt, admission, evaluation, model, opaque
per-attempt evidence and immutable asserting attribution. Identical retry returns latest
facts, not stale receipt-only status. Changing semantic meaning conflicts; original stays.
Admission/rejection terminal; evaluation changes cannot erase admission. Attempts are
owner-global idempotent identifiers, distinct from event identity and client attempts.

Errors distinguish invalid input, blocked authority, denied disclosure, identity conflict,
capacity, unavailable, integrity failure, commit-unknown, closed, context cancellation and
uncertain reply release. Never turn cancellation/unknown/no knowledge into rejection or
safe new identity. Release callback is synchronous/current-authority fenced, never an
uncontrolled output queue. Callback error/expiry may mean facts already escaped.

## Durable guarantees and limits

Separate file uses existing sqlite3 driver/exclusive single connection, DELETE rollback
journal, synchronous EXTRA/fullfsync, busy timeout0, explicit exclusive transactions.
Receipt and later admission/evaluation are separate durable transitions. Admission atomically
establishes the normalized event decision and not-established evaluation obligation.
Events/facts/attempt identity persist for store lifetime. Bounded capacity refuses before
consuming promised pending completion space; no pruning, expiry or automatic retries.
Reopening under smaller limits explicitly refuses excessive recovered state.

Unconfirmed COMMIT poisons owner until reopen. Pretransaction/precommit failure does not
fabricate a receipt; earlier in-flight work remains a separate uncertainty. Corruption
never becomes an empty fresh store. Host.Recover must independently reject stale backups;
this trusted anti-rollback authority is not implemented by SQLite or a caller boolean.
No power-cut/filesystem deployment, distributed failover, durable external reply audit,
production AUTH or unified JetStream admission guarantee follows from process-crash tests.

Codec/Core model is currently int64-state, with actual generic CORE machinery underneath;
this is a representative registration boundary, not an arbitrary-model runtime service.
Schema-specific pure/terminating callbacks and semantic validation remain host duties.
No production general JSON/JCS codec is shipped. The test codec registers a bounded
scalar integer counter; other domains must register their own reviewed semantic codec.
No automatic event-to-publication/projector or legacy supervisor/JetStream dispatch.
A committed event remains available for explicit host evaluation or awaits it honestly.

## Complete shared-row accounting

**New complete submission rows: S01, S02, S03, S04, S07 (5 passed, 0 failed).**
The six prior local observation rows were rerun; combined report: **11 passed,
0 failed, 44 unexecuted complete-row judgments out of55**. These are local semantic
oracles with representative authorities/independent merge oracle, not an actual
Go-TypeScript adapter or production AUTH. Earlier native P/R fixtures rerun but not
reclassified as new full-row execution. Partial tests are not full-row passes.
S05's earlier in-flight lookup schedule, S08's full projection/cancellation oracle,
and S06's expired-receipt scenario remain unexecuted here; S06 expiry is unsupported
by the approved lifetime-retention policy, never reported passed.

Exact row-focused command (exit0; stdout .loop/submission-vector-execution.log):

```
go test -count=1 -v ./sdk/go/submission ./sdk/go/observation -run '^(TestSubmissionIdentityAndEvaluation|TestSubmissionSemanticContextAndBounds|TestSubmissionVersionedFacts|TestSubmissionPendingAuthorityAndCancellation|TestSubmissionRealCrashAndOwner|TestSubmissionFailureCapacityAndRecovery|TestRecoverySnapshotAndReopen|TestRecoveryReconnectAndCancellation|TestRecoveryEnvelopeSessionFence|TestRecoverySubstitutionAndReplacement|TestRecoveryFailureCapacityAndCleanup|TestRecoveryReleaseOrders|TestSessionBoundsCancellationAndCleanup|TestRecoveryCloseCancelsOpen)$'
```

| ID | Full-row judgment | Witness / explicit gap |
|---|---|---|
| S01 | Passed | TestSubmissionFailureCapacityAndRecovery + TestSubmissionRealCrashAndOwner: durable reply lost; real reopen/identical retry recovers one admission and attribution |
| S02 | Passed | TestSubmissionIdentityAndEvaluation + TestSubmissionSemanticContextAndBounds: payload/cause/context/asserting-identity conflicts preserve original decision |
| S03 | Passed | TestSubmissionVersionedFacts: actual receipt1/admission2/failed evaluation3 delivered3/1/2 to independent full-fact merge oracle; admission retained |
| S04 | Passed | TestSubmissionVersionedFacts: differing same-version facts conflict, no overwrite; authoritative facts remain unchanged |
| S05 | Unexecuted | Related lookup/concurrency tests pass; full earlier-in-flight schedule not executed |
| S06 | Unexecuted | Expired receipt unsupported; lifetime identity retention |
| S07 | Passed | TestSubmissionPendingAuthorityAndCancellation: unresolved authority vs opaque denied; forged binding cannot establish state or release facts |
| S08 | Unexecuted | Canceled admitted facts recover; full projection-nonacknowledgment oracle not executed |
| P01 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P02 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P03 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P04 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P05 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P06 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P07 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P08 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P09 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P10 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P11 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P12 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P13 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P14a | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P14b | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P14c | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P15 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P16 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P17 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| P18 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| R01 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| R02 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| R03 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O01 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O02 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O03 | Passed | Existing TestRecoverySnapshotAndReopen rerun |
| O04 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O05 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O06 | Passed | Existing TestRecoveryReconnectAndCancellation + TestRecoveryEnvelopeSessionFence rerun |
| O07 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O08 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O09 | Passed | Existing TestRecoverySubstitutionAndReplacement rerun |
| O10 | Passed | Existing TestRecoverySubstitutionAndReplacement rerun |
| O11 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O12 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O13 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O14 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| O15 | Passed | Existing recovery failure/cleanup, reconnect and SessionBoundsCancellationAndCleanup rerun |
| A01 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| A02 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| A03 | Passed | Existing TestRecoveryReleaseOrders rerun; new TestSubmissionCloseAndRevocation separately exercises status release fence |
| A04 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| A05 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| A06 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| V01 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| V02 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |
| B01 | Unexecuted | Prior-role/partial tests may pass; no complete shared-row judgment in this report |

## One Go–TypeScript adapter can now target this boundary

Map Stage2B backend.send to scoped Submit and lookup to Resolve, capturing replies
only through the synchronous approved boundary and local operation fence. Map facts
receipt bool to durable/not-established, uint64 Version losslessly to bigint, and keep
generation/admission/evaluation/model/evidence/attribution unchanged. Map blocked/denied/
identity-conflict/unavailable distinctly; commit/release/cancellation uncertainty stays
unknown and reconciles SAME EventID. Capacity is unavailable/backpressure, never rejection.
Malformed/unsupported input is protocol validation failure, not fabricated admission.

Agree trusted descriptor, scalar/schema encoding and the exact normalized semantic
representation BEFORE client.track(); its opaque semantic string/attributionBinding
is not itself proof. Transport/client attempt IDs are not event IDs or durable idempotency
keys. Host authority verifies binding and delegation. Retention is declared lifetime/no
expiry. Do not forward Owner or admission/evaluation capabilities to clients.

This is sufficient for a single separately reviewed adapter against this gateway
namespace, not legacy JetStream ingress. Actual adapter/transport, credentials, descriptor
negotiation and cross-language encoding/equality tests remain to be done. Observation
uses existing Recovery independently; no projection revision resolves a submitted event.
