# Binding Stage 1 ledger kit contract

Subordinate to draft.6 commit 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87 and
locally ratified INTEROP-PUBOBS/0.1 v0.1.0 commit
9a80977efaf2ff740a257f6e30314dac42ec384e. See LEDGER-DESIGN.md for exact
manifest pin, driver, platform/durability assumptions, API alternatives and
trusted-host responsibilities. Copied profile/vector bytes must stay unchanged.
This is only the local ledger subset of role P, NOT full profile conformance.

## API and lifecycle

contracts.go is complete/protected. Implement only ledger.go/new implementation
helpers under sdk/go/ledger. Preserve all existing CORE/REV/runtime semantics.
Open(context, absolute local file path, Host, Options) returns an exclusively owned
private SQLite ledger or explicit error. Positive MaxRecords limits lifetime
acceptances; no retention pruning, generation migration, dispatcher or network.
Require nonnil Host; refuse invalid options/relative paths. An existing empty/
truncated or corrupt ledger must not be silently reset to a new authoritative DB.
Caller establishes a private owned directory and excludes hostile alias/path
replacement; no public authority is established by returning Heads/Pending.
Close releases resources, is idempotent; closed/poisoned handles refuse operations.

Use github.com/mattn/go-sqlite3 v1.14.32 with one persistent connection, DELETE
journal, synchronous EXTRA, fullfsync ON, locking_mode EXCLUSIVE, busy_timeout 0,
and exclusive transactions. Acquire the real DB exclusive lock before returning
Open, retain until Close/process death; second process returns ErrOwned.
Verify selected durability/locking pragmas at startup. Process locks must not
be a mutex/PID-file-only claim. Integrity/recovery validation precedes use.

## Persistence and reconstruction

A bounded append-only SQLite acceptance log is sufficient; each durable record
contains immutable request/receipt, accepted head/checkpoint, trusted evaluation
evidence, optional immutable publication and pending reference. Any equivalent
SQLite schema meeting atomicity/recovery requirements is permitted. On reopen,
validate contiguous acceptance checkpoints, head/predecessor/ordinal consistency,
unique immutable publication IDs and request identities, pending agreement and
valid transitions. Reject corrupt data, unknown/truncated records or mismatches.
Host.Recover must validate the entire head inventory against independent authority
state, including missing keys/stale backup/generation reuse. No fabricated empty
success or reset on decode failure. Pending returns copies of all unsent immutable
records; no delivered flag API yet. Heads returns private copied recovery heads.
Keep every accepted receipt for the lifetime; capacity refusal must not prevent
idempotent retry of retained receipts. No live backup/restore support is claimed.

## Acceptance

Serialize operations within the owner. Authorize BEFORE receipt lookup using
Host.Authorize (covers replay under current invocation authority). Request identity
is scoped by Key + Generation + ID. Equality includes ALL immutable Request fields,
not pointer equality. Exact retry returns the ORIGINAL Receipt even after the
head/evidence changed; no new checkpoint or publication. Different meaning under
same identity is ErrConflict. No collision across accountable producers/targets.

For a new request, verify exact expected head ordinal/publication ID and accepted
checkpoint; mismatches return ErrStale with no write. Host.WithinEvaluation owns
source/rule/admission/permission fencing through callback commit, validates actual
EvaluationID freshness and producer/target, and supplies Evidence. Reject key
mismatch, unregistered/forbidden probes, missing required internal authority or
invalid identity/provenance. Caller data cannot establish registration, permission,
completeness, evaluation success or source-generation freshness. Trust is in the
host implementation and configured invocation boundary, not boolean syntax.

Every accepted new evaluation advances its key/generation checkpoint exactly once,
including no-op/failed evaluations. No-op does NOT advance publication ordinal or
update prior immutable provenance. On semantic change allocate a fresh globally
unambiguous publication ID and next contiguous ordinal, predecessor points to prior
publication, generation/schema/rule/attribution/cut accompany immutable record.
No generation reset or ordinal overflow. Use existing UUID library or equivalent
cryptographically generated ID, not lexical time ordering. Retain original ID on
retry. Status transitions share the sequence with content/retractions.

Call pure rev.Reconcile on trusted Computation/stored head, but initial complete
valid absence produces no publication. Initial permitted unresolved has ordinal1,
no predecessor/value. Permission to publish unresolved/key existence is explicit
trusted registration/policy, NOT inferred from an arbitrary failed probe. When
publication is not permitted, retain only separately authorized internal evaluation
checkpoint/receipt and leave the publication head unchanged; do not externally
claim that prior value remains current. Accepted internal evidence and publication
state are separate. A supported value followed by failure yields stale unresolved;
same-value recovery yields ordered current status; disappearance yields retract.
Repeated identical current/unresolved/retracted states are no-ops. Status intentions
retain value/availability from prior assertion; do not overwrite with empty string.
Encoded null is available. Consequential issuance excluded by host schema authority.

Persist receipt, accepted checkpoint/head, immutable publication and pending
obligation atomically; they survive together or not at all. Publication accepted
never means delivered. Request cancellation/timeouts do not prove rollback.

## Failures and controlled test seam

ErrConflict identity reuse; ErrStale preconditions/evidence/restore rejection;
ErrOwned competing owner; ErrDenied host authorization/registration unresolved or
denied (private local category, not a public denial taxonomy); ErrCapacity bounded
backlog; ErrUnavailable known unavailable/integrity/precommit failure; ErrUnknown
unconfirmed commit outcome requiring close/reopen and authorized identical retry.
Never return a durable success receipt before Commit is confirmed. Commit errors
are conservatively unknown; poison that handle. Lost reply after confirmed commit
also returns unknown, retaining the committed record. No guessed rollback.

Options.Cut is ONLY a controlled testing/failure-injection seam. Invoke before_tx
before transaction begin; after_write inside transaction after persistence writes;
before_commit before durability; after_commit only after successful Commit.
Hooks returning errors before commit trigger rollback/unavailable; after_commit
error returns unknown and poisons handle. Subprocess os.Exit at cuts exercises
actual SQLite rollback/reopen and released ownership, not an in-memory fake.
Do not add test-name/environment special cases to implementation.

## Vector/test mapping and evidence limits

TestLedgerAcceptance: P01/P02/P07/P09/P10-before/P11/P13 retained identity.
TestLedgerInitialAndAuthority: P03/P04 and retraction lifecycle; P06 denied probes;
permission withholding without publication, independently available null.
TestLedgerIndependentKeysAndStaleFailure: P17/P08.
TestLedgerUnknownCapacityAndRecovery: P14a/P14b/P14c, B01 count capacity, P18
independent authority-floor refusal (not production backup authority).
TestLedgerRealCrashAndOwners: P12/P13/P15 real child processes, every named cut,
reopen/read/retry; precommit none vs confirmed postcommit all. Test helper is not
an independent task guarantee and does not count as durability by itself.
R01/R02 remain prior REV/CORE integration regressions; no replacement history.
Additional explicit P05 initial-unresolved-to-retract, P10 ordering, backup-restore
and storage/schema/resource cases may be added to strengthen this boundary.
These mappings are bounded evidence, not exhaustive schedule proof. P16 delivery,
all O/S public services and real A enforcement are deferred. Shared vectors are
unexecuted upstream; only concrete mapped Go schedules establish local evidence.

Prior tasks remain passing, new tasks fail on zero-return stubs, independent
private reference must pass through unchanged scripts/loop.sh --check. Final gate:
focused ledger tests, full Go tree, race, vet, previous CORE/REV, Node/frontend,
fixture/loop comparison and diff inspection. No paid loop before verified baseline.
