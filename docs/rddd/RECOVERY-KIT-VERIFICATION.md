# Stage 3B completion — verified kit

Starting HEAD 0d70ab726edf7d06bd836f6ebdabae0563c14391, rddd-foundation. Only
preexisting untracked files were OBSERVATION-3B-REVIEW.md and
OBSERVATION-3B-VECTOR-STATUS.md; both preserved unchanged and included in baseline.
Owner separately approved architecture and kit/local commits/bounded loop, subject
to six constraints. RECOVERY-SPEC.md freezes the reconciled obligations and tests.
No normative contradiction found. Compatibility improvement: use existing prepare
entries/latest snapshot plus separate retained anchor; no new entry/schema/version.

Pins checked again: draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
INTEROP-PUBOBS/0.1 v0.1.0 9a80977efaf2ff740a257f6e30314dac42ec384e;
all19 checksum entries pass, manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Specification tree unchanged/clean. Judge byte-identical to installed v1.1.1.

New complete/protected contracts: retained_contracts.go, recovery_contracts.go.
New implementation stubs: viewjournal/retained.go and observation/recovery.go.
Existing journal.go gets only approved protected reserveCleanup wiring, retaining
old capacity behavior on stubs. Existing observation/session.go and all prior code,
contracts/tests/dependencies remain protected. No source/storage/wire migration.

Private reference written outside checkout before judging, installed temporarily
only for exact judge verification, then removed/restored stubs. Not shipped or
committed. Behavioral assertions and real test-only AES-GCM verifier exercise proof
binding, not always-success flags. Current permission still checked separately.
Reference verification uses actual loop-written prior stages, not reconstructed copies.

Actual checks before baseline:
- Focused observation/viewjournal tests pass with reference.
- Focused race tests pass with reference.
- scripts/loop.sh --check: ALL23 tasks pass with final reference, including original
  17 focused groups and six new groups B3PLAN/B3RESUME/B3ORDER/B3OWN/B3FAIL/B3ALL.
  Every expected passing stdout line required; final whole-tree includes Go, race,
  vet, Node complexity and frontend tests. Script/format/kit checks unchanged.
- SAME judge on formatted stubs: exit1, ALL SIX new groups fail through actual test
  commands; original17 groups pass. Overlay compiles against actual prior build.
- Source reservation property tests protect cleanup headroom and arithmetic overflow;
  real SQLite capacity/reopen test preserves original Read lower-bound behavior.
- Real subprocess cuts: retained acquisition before/after commit; process exits during
  actual recipient snapshot output, before barrier issuance completes, and during
  barrier output. Recipient artifacts and test keys are private temporary files;
  independent issuer key file survives store-owner process death. New authority
  object verifies proofs after real reopen, never inferring client application.
- Unknown acquired-plan commit reply: ErrUnknown plus owned ErrCleanup, poisoned
  handle cannot pretend cleanup success, namespace recovery after reopen confirms it.
- Extra fault tests: signed mutated record is rejected before callback; verifier
  outage is unavailable rather than false token invalidation, same cursor retries.
- Registration, state acceptance and disclosure are distinct trusted decisions.
  Revoked observation permission does not prevent private service cleanup.

Source/release scope: one trusted exclusively owned namespace/slot, append-only
history plus separately injected replay-window expiry. No actual pruning or automatic
ownership/production credential/key administration. Caller bounds slot count and
excludes previous live namespace owner. Cooperative synchronous callbacks required;
no forced termination or abandoned timeout goroutines. No TypeScript/transport/effect
work, cross-domain transaction or complete CLIENT/security/interoperability claim.

Authorized bounded run after verified clean stub baseline:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Pinned default model commands unchanged; no substitution or push/merge/upstream edits.
Final 55-row execution coverage will be recorded separately from historical
review inventory; partial related baseline evidence is not automatically a full pass.
