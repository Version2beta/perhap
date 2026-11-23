# Binding Stage 3B completion kit

Approved architecture: OBSERVATION-3B-REVIEW.md, subject to owner's six conditions.
Pins remain draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87 and profile v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e; all19 checksums and manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f verified.

## Compatibility reconciliation before freeze

No new entry kinds/schema/fields. Acquire writes an EXISTING prepare entry with the
latest snapshot (therefore old readers validate it), returning a separate immutable
retained anchor selected in that same transaction. ReadRetained permits older retained
positions while old Read keeps its original lower bound. Append-only retention makes
all accepted positions physically retained; policy expiry remains separate. Plan IDs
in the new private namespace obs/owner/nonce/after encode owned acquisition identity;
no Plan ID is disclosed. after=0 chooses original pinned snapshot on identical retry.
Owned nonce and suffix are validated and immutable; source rejects changed view/reuse.
Existing UUID pins and existing methods/entries remain compatible. Only new methods
are added. Cleanup space is reserved by all new commits while plans are active:
2048 bytes per pin (bounded private IDs <=128 bytes); release may spend its reservation.
Legacy files with insufficient room require explicit larger configured capacity to
complete cleanup, never discarding history. Old binaries can read new entries, but
cannot enforce new resource policy; do not concurrently transfer writer ownership.

## Trusted roles and contract

Registration and rendering remain Journal.Host decisions. Current Gate.Authority
checks independently authenticated disclosure/state/descriptor/epoch before ALL
protected releases. ProofAuthority separately issues/verifies opaque purpose-bound
claims and applies trusted expiry/replay-window policy. Possessing a proof never
authorizes disclosure. Test authority uses real authenticated encryption; no production
AUTH/format/key store supplied. Host must retain trusted keys/policy and anti-rollback
checkpoint authority independently of SQLite. Permission state is never recovered
from a token. P04/P05 registration alone does not permit observing states; P06 probes
must not create a projection/key. Whole projections do not gain assertionStatus.

One Recovery owner slot serializes Open/Resume/Replace, joins prior Session before
new activation, and owns at most one plan/session. Separate slots have independent
bounded owners supplied by host. Owner namespace is trusted service identity, not
observer input; caller bounds number of owners. Source uses existing single SQLite
owner. On NewRecovery, reconcile only that namespace's abandoned plans; caller must
have excluded a previous live owner. Never clean unrelated pins. Cleanup uses service
Host authority, independent of observer revocation. No background goroutine; contexts
and synchronous cooperative callbacks; no reentry or deferred disclosure permitted.

Proof binds Binding, purpose, complete immutable Record and authority validity.
Snapshot handoff and replacement each carry a proof; only a barrier token is resumable.
Check current authority before token parsing/lookup; cap token bytes first. Verifier unavailability/cancellation is a recoverable failure, not proof of an invalid
cursor: return a stable unavailable/context category without emitting cannot-resume.
Issue/Verify must match an immutable expected record, even if a dependency mutates
its input copy. Invalid, expired, incompatible or policy-unretained token yields
cannot-resume, released through
Gate with no hidden details. No automatic replacement. Explicit Replace gives a
purpose-distinct certified replacement snapshot + barrier. Missing/corrupt source or
anchor mismatch gives sequence/unavailable, not a healthy reset. Resume verifies
anchor against durable record and emits successors only; each finite Step validates
entire bounded page before callbacks, then barrier at exactly last released revision.
No ordinary frame clears a gap; no snapshot/barrier is submission acknowledgment.
Available null, evaluation, backend completeness and as-of freshness remain unchanged.

Actual linearization is the same Gate mutex held through authority check and the
actual synchronous recipient callback. Rich envelopes must release INSIDE that
callback; never after it. Cursor issuance may precede revocation but its release
must still be denied if revocation won. Failed/expired callback is ErrUnknown even
if it returned nil; no barrier follows. All failure paths retire and clean session,
join ErrCleanup with primary failure, and retain retryable ownership until cleanup
succeeds. Close is idempotent; reconnect cannot activate over failed cleanup.

Unknown Acquire commit leaves pending ownership in namespace for explicit reopen/reconcile;
Close must report ErrCleanup until resolved, not forget the uncertain pin. Close
cancels an in-flight Open before joining it. No guessing rollback. Controlled subprocess/reopen tests must establish actual cuts.
Max frame/page bytes, token bytes, page count, callback/cleanup deadlines configured
explicitly. Current schema is bounded typed pair-sum/v1; envelope byte sizing is a
private accounting encoding, not a shared wire representation. MaxBytes caps emitted
frames/buffered pages after source read; source decoding/allocation remains additionally
bounded by the separately configured journal MaxBytes, not by a pre-allocation gate.

## Test mapping and acceptance

Retained planner tests: historical anchor, concurrent publication, original API lower
bound, idempotent acquisition, owned enumeration, unknown/crash/reopen, cleanup budget.
Recovery tests: snapshot40/concurrent41/handoff/barrier, 43->44/45 after interruption
and SQLite reopen, substitution of every Binding field/purpose/token bytes, retention
expiry then explicit replacement90, corrupted anchor/scan, stale-slot callbacks,
revoke-before and release-before actual envelope disclosure, capacity/cancel/cleanup.
Independent recipient assertions validate binding, snapshot-first successor continuity
and proof purposes. Source-anchor conflict is rejected before delivery; client-side
repeated-frame application/conflict quarantine is not implemented by this server API. Actual profile row coverage only credited for
complete exercised oracles, not case labels or related baseline tests. All55 rows
enumerated in final evidence. S/client-only rows remain unexecuted as applicable.

## Final-review obligations (same approved contract, no API changes)

Bootstrap Acquire(after=0) returns its pinned snapshot as Anchor, including retries
following newer commits. Its tail cannot start below that anchor. Owned IDs and owner
names must be ASCII and losslessly round-trip storage; invalid UTF-8 must not silently
change an ID and strand its pin. Record-bearing envelopes carry a nonempty opaque
session fence, stable for that session and different after reconnect, never a Plan ID.
CallbackTimeout applies to Issue, Verify and admission checks, not just recipients.
Expired admission results cannot invoke recipient even if Check returned nil. Context
cancellation/deadline errors from authority, proofs and storage must return canonical
safe categories, not private wrapped diagnostics. These are tested by the corrective
B3REVIEW group after reproducing failures in the initial judged implementation.

Independent reference outside checkout; new tasks fail on stubs and full unchanged
judge passes reference before clean kit commit. Preserve Stage3A/3B1 tests and APIs.
Then MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh; full Go/race/vet/judge and source
review. No dependency, transport, TypeScript, upstream or production changes.
