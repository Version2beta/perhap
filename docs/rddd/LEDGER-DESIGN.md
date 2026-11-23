# Publication Stage 1 — selected Go storage boundary

## Authority and pin

Binding baseline draft.6: 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87.
Approved optional INTEROP-PUBOBS/0.1 version 0.1.0: locally ratified commit
9a80977efaf2ff740a257f6e30314dac42ec384e, clean checkout. SPEC.md has no diff
from baseline. All 19 current SHA256SUMS entries pass. Manifest digest:
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Ratification covers profile semantics, not core ratification or implementation
conformance. Copied profile/vectors and manifest in ledger testdata pin the input.
All six existing uncommitted PUBLICATION planning documents are preserved as
historical proposals. Approved profile, not those proposals, governs this kit.
No other repository modified. Preserve CORE/REV commit 7583220 and later baseline.

## Selected driver and durability configuration

Pinned github.com/mattn/go-sqlite3 v1.14.32, using database/sql and bundled SQLite.
One added module; no driver transitive modules. Requires CGO and a C toolchain;
observed Go 1.27.2 darwin/arm64, CGO_ENABLED=1 and /usr/bin/cc. Existing SDK already
has dependencies; stdlib has no SQLite implementation. Alternative modernc driver
avoids CGO but brings a larger pure-Go dependency graph. Do not shell out to sqlite3.

One database handle/connection, max open/idle 1, no connection expiry. DELETE
rollback journal, synchronous EXTRA (3), fullfsync ON, locking_mode EXCLUSIVE,
busy timeout zero and exclusive transactions. Acquire real SQLite exclusive lock
before returning Open, retain it until Close/process termination. A second process
cannot open an authoritative ledger while the owner holds that lock. SQLite locks
are released by process exit, not stale PID-file cleanup. Startup checks actual
journal/locking/synchronous settings and integrity before reporting ready.
The independent reference passed real subprocess owner/reopen and transaction
crash tests with these settings before starting the loop.

Supported deployment: local filesystem with reliable SQLite locking/fsync in a
private, caller-owned directory, stable canonical path, no symlink/hardlink aliases,
path replacement or uncooperative raw writers. No network filesystems, multi-host
leases, online backup/restore or malicious storage tamper guarantees. Sensitive
payload protection depends on directory/file access control established by host;
this API is not a public HTTP/auth service. No guarantee against lying storage,
media loss or power loss follows from process-exit tests alone.

## Smallest coherent persistence model

Bounded append-only acceptance ledger: each SQLite transaction appends one record
containing command/receipt, evidence checkpoint, assertion head, immutable publication
when applicable, and explicit pending obligation. Atomic columns/tables are not
required by profile; a single private versioned record is simpler for this bounded
stage. Reopen validates records, contiguous acceptance checkpoints, publication
identities/ordinals/predecessors, transitions and pending/head agreement; rebuild
private heads and retained receipts from that evidence. No receipt expiry, pruning,
compaction or mark-delivered API. MaxRecords is a positive lifetime acceptance bound;
when exhausted, refuse new work but still resolve retained identical retries.
This O(n) bounded replay is deliberate; indexed normalized tables are a later
optimization, not a reason to weaken recovery invariants.

## API/ownership and trusted-host boundary

New sdk/go/ledger package with Open, Accept, Heads, Pending, Close. Context travels
through blocking SQL/host calls; the owner closes its connection; no background
workers or outgoing network. Requests identify key + durable generation, stable
request identity, evaluation reference, expected checkpoint and exact publication
head. Values remain canonical immutable Go strings; persistence JSON is private,
not a shared wire format or INTEROP-JSON claim. Return copies of pending records.

Host.Authorize authenticates the local invocation and producer/target before any
receipt lookup, including old retries. It must authorize private owner queries
by restricting who holds the ledger handle; Heads/Pending are not public observers.
Host.WithinEvaluation proves EvaluationID identifies the actual current computation,
source cut, active rule/schema, accountable producer, registration and permissions.
It MUST serialize relevant source/rule/admission/permission changes with callback
execution through durable commit, or reject stale; cannot mint freshness by copying
request flags. It calls the callback exactly once or not at all and does not
reenter the ledger. A Go mutex inside the ledger alone proves none of this.
Host.Recover verifies generation/ownership and restored inventory against authority
state NOT rolled back with the database; it must reject a stale backup/missing key
before enabling mutation. No single local DB can detect an indistinguishable stale
copy without independent evidence. Tests use an explicit authority floor seam;
there is no production distributed admission authority in this stage.

Host evidence includes separately authorized internal retention and registered-key
status/visibility permission. Unregistered/forbidden probes must not create a key.
Withheld publication may record an internal receipt/checkpoint only where permitted;
no externally current claim is derived from that internal result. Consequential
issuance is excluded by trusted schema registration, not an untrusted event name.

Only the host supplies Computation, not the caller. Boundary calls unchanged pure
rev.Reconcile but handles never-published complete absence before persisting its
zero-Assertion intention. No-op publication state is not updated in place; checkpoint
and evidence receipt progress separately. Status publication loads stale value and
availability from stored previous head, since Intention.Value may be empty.

Compare alternatives: accepting raw Intention trusts unsupported freshness and
omits bootstrap policy; adding SQL/IDs into pure REV breaks its ownership boundary.
A wrapper with an explicit trusted authority callback and atomic append preserves
CORE/REV semantics and requires no transport or shared API changes.

## Failure classes and limits

Sentinels distinguish conflict, stale precondition, competing owner, unavailable/
integrity storage, host denial, capacity, and unknown commit outcome. Before commit,
known storage failure is unavailable; once commit is issued but cannot be confirmed,
return unknown and poison the handle until reopen/lookup via identical retry.
No success before SQLite Commit establishes durability. Controlled after_commit
failure tests simulate a lost response, not physical media uncertainty. Original
receipt survives subsequent head changes. All counters refuse overflow, never wrap.
Random publication IDs are allocated inside acceptance and retained durably; retry
never allocates again. Host generation/namespace authority prevents reset reuse.

No full P role, PUBOBS interoperability, AUTH, public observation, delivery or effect
conformance claim. Shared discovery is a locally pinned/injected descriptor; no
live discovery/authenticity protocol. Stage 2 should send oldest pending per-key
records to an accepting durable sink, retaining identities across lost receipts;
no broker is required to prove that behavior.
