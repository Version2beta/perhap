# Go integration prerequisites — completed phase, integration still pending

This is NOT completion of the approved cross-process integration assignment.
No UDS transport/Go integration host/TypeScript adapter or end-to-end test has been
implemented or executed yet. Those remain authorized by the owner's current request;
no further architectural approval is being requested here.

## Local commits

Verified stub kit: 6d8ff4f76efd1cfe30737a2e7c7442521d379eb9.
Judged optional resume: 9aa55136535032867706619d3efbd693871548b0.
Judged semantic matching: 240dabad0a79e132b4f827989cb4383361e7d476.
Two iterations, first attempt each, pinned claude-haiku-5-5 command, unchanged runner;
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh. Private reference outside checkout,
all31 exact judge groups passed; stubs failed all3 new groups, prior28 focused groups
passed. Existing whole-tree test replaced by equivalent expanded whole-tree test.
All31 passed with actual loop-written prerequisites, including full Go/race/vet and
Node/frontend checks. Additional focused race repetitions5 passed: observation7.949s,
submission2.241s. Source/diffs inspected. No earlier tests weakened.

Actual addition: Recovery.ResumeCertified uses same established Resume validation,
retained anchor equality and current Gate fence. It issues purpose resume at actual
accepted position before exposing a session that can Step; failure cleans/retains pin
ownership through existing discard path. Legacy Resume/Step remains unchanged.

Client.ResolveMatching independently normalizes expected input through registered
codec and compares immutable stored identity/attribution/schema without writing or
calling admission/evaluation. It returns latest facts, no-retained-knowledge or a
protected identity-conflict reply. Conflict goes through Host.Release; no raw original
data is disclosed. Existing ID-only Resolve remains supported. Existing validators,
limits, cancellation and status-disclosure authority still apply.

Baseline draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87 and PUBOBS v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e, all19 checksums verified; manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f. Client unchanged
at 81641540cb8e79d002cf9599ff33118b44e12df0; no upstream modifications.

## Evidence categories

Go-only: prerequisite positive/negative native tests and earlier local semantic
regressions rerun. These add no newly credited complete PUBOBS row by themselves.
TypeScript-only: existing evidence remains historical at supplied baseline; no new
TypeScript test execution in this prerequisite phase.
Actual end-to-end: ZERO rows executed in this phase; no cross-process success claim.
Partial: O06/O10 handshake and S02 matching-related guarantees tested locally; neither
constitutes an end-to-end complete row. Existing11 Go/local row judgments remain in
SUBMIT-RESOLVE-RESULTS.md, not promoted to integration judgments. S06 unsupported.

## Remaining authorized sequence

Use GO-CLIENT-INTEGRATION-DESIGN.md for the approved UDS transport, strict bounded
framing, lossless integers and pinned integration-only signing host. Build actual Go
composition over these verified methods, then independent TS adapter kit. No fake
backend substitution, submit-as-lookup, certificate flags or unguarded output queues.
Keep explicit Evaluate/projection fixture/control operations separate from product
automation. Run real process death/reopen, loss/reconnect/cancel/revoke/scope cases,
Go and TS full regressions and exact55-row end-to-end coverage report. This work has
not started beyond the two independently verified Go prerequisites.
