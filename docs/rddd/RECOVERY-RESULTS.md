# Stage 3B completion — final evidence and 55-row coverage

Implementation state: 6b211fa3d41417135d8478d55a3b4e75df40ca84 on rddd-foundation.
Ratified source: INTEROP-PUBOBS/0.1 v0.1.0 at
9a80977efaf2ff740a257f6e30314dac42ec384e; draft.6
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; all19 checksums verified,
manifest 01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.

## Acceptance and provenance

- Verified kit: 15f1082e843755192c2a7168b4f3ec6b7eb655db.
- Judged planner: 094becf9491c2f67ea66d44d10f3a9441cb46d9e.
- Judged recovery: c025e2cb429c621a97e12dc2339e3fdaa8b50a8d.
- Verified corrective overlay: 3aafcb5bd76db5f2defb72a96977eca633019af2.
- Final judged correction: 6b211fa3d41417135d8478d55a3b4e75df40ca84.

Independent reference outside checkout passed original23 then corrected24 judge
groups; initial stubs failed all6 new groups, prior17 passed. New review regressions
failed actual incomplete judged code; reference passed all24. No private implementation
committed. Protected tests were not weakened. Three coding iterations total, all
attempt1 using pinned claude-haiku-5-5 command; initial MAX_ITER=12 MAX_ATTEMPTS=3,
corrective remaining MAX_ITER=10 MAX_ATTEMPTS=3. Unchanged v1.1.1 judge.

Final commands, all exit0 at implementation state above:

```
go test -count=10 ./sdk/go/observation ./sdk/go/viewjournal
go test -race -count=10 ./sdk/go/observation -run '^(TestRecoveryReleaseOrders|TestRecoveryReconnectAndCancellation|TestRecoveryCloseCancelsOpen|TestRecoveryAuthorityDeadlines|TestRecoveryErrorsAreOpaque)$'
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
scripts/loop.sh --check
git diff --check
```

Focused repetitions: observation 126.827s, journal 23.740s; repeated race selection
21.274s. All24 judge groups passed, including original stages and Go/race/vet/Node/
frontend checks. Source/diffs inspected. Original CORE/REV/ledger/delivery, 3B1 code/
contracts and all earlier tests, dependencies and loop script unchanged. Only earlier
journal.go change is protected reserveCleanup wiring. Review documents preserved
byte-for-byte (their design-review inventory is historical, not current execution).

## Precisely scoped shared-row execution

**6 passed, 0 failed, 49 unexecuted full-row judgments in this Stage 3B report.**
Counts are local semantic oracles using representative trusted authorities, not
production enforcement or complete profile conformance. A passed row does not imply
all role requirements or all failure permutations passed. Existing native P/R/CORE
fixtures and related regression tests were rerun successfully but are NOT automatically
reclassified as new complete 55-row executions; their original evidence reports remain
valid within their own scope. Unexecuted here does not mean no related test ever ran.

The actual row-focused run selected TestRecoverySnapshotAndReopen,
TestRecoveryReconnectAndCancellation, TestRecoverySubstitutionAndReplacement,
TestRecoveryEnvelopeSessionFence, TestRecoveryFailureCapacityAndCleanup,
TestRecoveryReleaseOrders, TestSessionBoundsCancellationAndCleanup,
TestRecoveryCloseCancelsOpen, TestRecoveryAuthorityDeadlines,
TestRecoveryErrorsAreOpaque, TestRecoveryOutputCrash and TestRecoveryOwnedCrash:
`go test -count=1 -v ./sdk/go/observation -run '^(TestRecoverySnapshotAndReopen|TestRecoveryReconnectAndCancellation|TestRecoverySubstitutionAndReplacement|TestRecoveryEnvelopeSessionFence|TestRecoveryFailureCapacityAndCleanup|TestRecoveryReleaseOrders|TestSessionBoundsCancellationAndCleanup|TestRecoveryCloseCancelsOpen|TestRecoveryAuthorityDeadlines|TestRecoveryErrorsAreOpaque|TestRecoveryOutputCrash|TestRecoveryOwnedCrash)$'`. It passed. Full stdout retained in ignored
`.loop/recovery-vector-execution.log`; no test branches on profile row IDs.

| Row | Judgment in this report | Actual witness / scope |
|---|---|---|
| S01 | Unexecuted | Submission role not implemented in this stage |
| S02 | Unexecuted | Submission role not implemented in this stage |
| S03 | Unexecuted | Submission role not implemented in this stage |
| S04 | Unexecuted | Submission role not implemented in this stage |
| S05 | Unexecuted | Submission role not implemented in this stage |
| S06 | Unexecuted | Submission role not implemented in this stage |
| S07 | Unexecuted | Submission role not implemented in this stage |
| S08 | Unexecuted | Submission role not implemented in this stage |
| P01 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P02 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P03 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P04 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P05 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P06 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P07 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P08 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P09 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P10 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P11 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P12 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P13 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P14a | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P14b | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P14c | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P15 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P16 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P17 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| P18 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| R01 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| R02 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| R03 | Unexecuted | Earlier role regressions preserved/rerun; no new full-row judgment |
| O01 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O02 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O03 | Passed | TestRecoverySnapshotAndReopen: snapshot40, concurrent41, authenticated handoff and barrier41; repeated as real reopen |
| O04 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O05 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O06 | Passed | TestRecoveryReconnectAndCancellation + TestRecoveryEnvelopeSessionFence: 43 -> 44,45/barrier; old callback joined/fenced, distinct session envelopes |
| O07 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O08 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O09 | Passed | TestRecoverySubstitutionAndReplacement: expired proof / logical retention floor -> cannot-resume -> purpose-certified replacement90/barrier; no historical/submission repair claim |
| O10 | Passed | TestRecoverySubstitutionAndReplacement: authenticated claims with wrong observer/context/projection/parameters/view/generation/schema/model/epoch/descriptor/purpose rejected before data |
| O11 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O12 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O13 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O14 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| O15 | Passed | TestSessionBoundsCancellationAndCleanup + TestRecoveryFailureCapacityAndCleanup + TestRecoveryCloseCancelsOpen + TestRecoveryReconnectAndCancellation: repeated close, callback uncertainty, primary+cleanup errors/retry, old completion joined |
| A01 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| A02 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| A03 | Passed | TestRecoveryReleaseOrders + TestReleaseRevocationOrders: actual local callback/revoke fence, both orders, zero revoked payload/proof/metadata callbacks |
| A04 | Unexecuted | Real client cache/application lifecycle not implemented; local server fencing is partial evidence |
| A05 | Unexecuted | Real client cache/application lifecycle not implemented; local server fencing is partial evidence |
| A06 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| V01 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| V02 | Unexecuted | Related local tests may pass; complete shared-row oracle not executed |
| B01 | Unexecuted | Real capacity/cleanup/overflow tests pass; complete multi-role backlog/buffer/ordinal oracle not executed |

## Actual durability/recovery evidence and limits

Actual SQLite transactions/subprocess exits establish owned-plan before/after-commit
cuts. Another real child exits during synchronous snapshot output, before barrier
issuance completes, and during barrier output. Parent reopens actual store and uses
an independently retained test issuer key file/new issuer object: snapshot-purpose
proof cannot resume; prior valid barrier or possibly released barrier resumes44/45.
Unknown commit reply preserves ErrUnknown+ErrCleanup; poisoned handles require reopen,
namespace recovery removes owned pins and preserves unrelated UUID plans. These tests
do not establish atomic database-plus-output delivery, exactly-once application or a
production durable release-attempt audit. Possible output remains uncertain.

No physical pruning: expiry is authority-controlled certificate/replay-window expiry;
all accepted history is append-only. Real missing/corrupt storage refuses continuity,
never an empty database/reset fallback. Existing Stage3A corruption/crash tests remain
passing. External Host.Recover anti-rollback jurisdiction, surviving issuer keys/policy
and exclusion of earlier live namespace owners remain trusted injected obligations,
not implemented production services. No remote AUTH/ownership/key-rotation guarantee.

New admission/proof/recipient deadlines require cooperative dependencies; no arbitrary
Go callback can be forcibly terminated safely. The actual synchronization fence is
Gate.mu held through current Check and actual synchronous recipient release, not an
unlocked precheck or deferred queue. Revocation-first blocks all frames/proofs;
release-first is prior authorized disclosure, not recallable. Expired admission never
calls recipient even if Check returned nil. Context diagnostics are normalized.

## Compatibility / API changes

Additive viewjournal.Acquire and ReadRetained, Retained; existing Prepare/Read/Plans/
Release semantics and private entry format remain. Bootstrap zero means pinned
snapshot, never fake revision0; retries keep original cut. Owned IDs are bounded ASCII
and cannot be confused with public cursors. New writes reserve2048 bytes per active
pin for cleanup, refusing capacity earlier. Legacy undersized files may need explicit
larger configured capacity to finish cleanup; no accepted history discarded. Original
reader/entry validation paths are unchanged. Additional actual backward-reader
experiment: git archive 0d70ab726edf7d06bd836f6ebdabae0563c14391 of go.mod/go.sum/
sdk/go/viewjournal into a new temporary module; compile its old reader with
GOPROXY=off GOSUMDB=off GOTOOLCHAIN=local go run -mod=readonly. Current writer at
6b211fa3 created records1/2/3 and an existing-format owned plan with snapshot3,
retained anchor1 and successor2/3. Old reader reopened that actual file, validated
head/checkpoint3 and value3, enumerated the new plan, rejected old Read(after1) as
stale and accepted Read(after3). Both commands exit0. Validation helpers/database
retained temporarily under /tmp/perhap-backward-compat.Oavurs; no dependencies fetched
or repository source changed. This is old-reader compatibility, NOT mixed-writer or
rolling-downgrade approval: old writers cannot enforce cleanup reservation; exclusive
ownership/approved writer transfer remains required.

Additive observation.Recovery/Recovered with Open/Resume/Replace/Step/Close,
RecoverySource, ProofAuthority, Claims, RecoveryFrame/Recipient/Limits.
One active session per trusted owner slot. Current Check before verified resume and
every release; registration, state and disclosure permission separate. Opaque purpose-
bound proofs cover full scope/observer/record; only barrier tokens resume. Session
fences are nonempty, stable per session and different after reconnect; private Plan
IDs never emitted. No production issuer implementation or shared codec selected.
Old 3B1 API stays available; it does not gain the new recovery guarantees implicitly.

Minimum observation/submission adapter boundary and remaining integration blockers:
see GO-GATEWAY-CONTRACT.md. No TypeScript, upstream, transport or effect work done.
