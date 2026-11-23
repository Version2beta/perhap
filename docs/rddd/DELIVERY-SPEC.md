# Binding Publication Stage 2 — durable local admission and ordered delivery

## Authority and translation

Baseline draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
locally ratified INTEROP-PUBOBS/0.1 v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e. All 19 checksums verified;
manifest 01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Read publication designs and approved shared vectors. Normative focus: profile
I, P1/P4, P16, R03 and B01 delivery limits. No normative contradiction found.
This local binding proves neither production AUTH nor general interoperability.

## Architecture and idiomatic API choice

New sdk/go/delivery package. Existing CORE/REV and Stage 1 storage/schema unchanged.
Source.Pending provides immutable obligations (Stage 1 Ledger implements it).
Sink.Admit is replaceable; it promises authenticated durable RECIPIENT admission,
not a transport receipt or projection application. Admission binds receiver identity
and entire immutable Publication. No projection/effect API is provided.

Store is a minimal independent SQLite recipient ledger. A separate Store file
is the sender's durable confirmation journal bound to the SAME receiver identity.
Stage 1 Pending remains archival scheduling evidence; outstanding delivery is that
set MINUS matching durable receiver admissions in the confirmation journal. Never
delete immutable source records. Losing confirmations may cause safe duplicate
retries, not loss of source work. No distributed transaction is required: sink
commit first, then local confirmation commit. Unknown between commits resends SAME ID.
Do not modify Stage 1 format or its host obligations to add delivery flags.

Prefer explicit bounded Run passes over a background goroutine pool: one worker,
one outstanding call, no timers/sleeps or unowned async work. A service owner may
invoke subsequent passes after errors/restart; no forever loop is hidden here.
Alternative modifying Stage 1 transaction schema risks its recovery guarantees;
a separate journal keeps this stage additive. A transport-first design cannot
establish durable recipient admission. No broker/dispatcher framework needed.

## Store contract (local, not shared wire)

contracts.go is complete/protected; implement only delivery.go/new helpers.
Open uses an absolute stable private path, positive MaxRecords/MaxBytes, configured
nonempty receiver and accountable producer. Pin identities in store metadata and
reject incompatible reopen. Host owns directories/permissions/capabilities and
forbids alias replacement, stale restores, malicious writers and NFS. No backup
restore/failover protocol is implemented here. The owner opens/closes store, calls
Run serially, and closes dependencies after it returns. Close idempotent.

Use existing mattn/go-sqlite3 v1.14.32 dependency, one connection held for store
lifetime, DELETE rollback journal, synchronous EXTRA, fullfsync ON, SQLite exclusive
ownership and zero busy timeout. No second writer; process exit releases locks.
Verify SQLite integrity and all persisted admission records/continuity before
returning an authoritative store. Existing empty/unknown/corrupt schema is not a
new empty ledger. Settings and reopen are validated with actual file-backed tests;
process termination is not power-cut/media-loss evidence.

Admit atomically persists one whole Publication under a globally unique event ID
and scoped key/generation/ordinal. Preserve generation, producer attribution, schema,
rule/cut, predecessor, content/op/status and explicit value availability exactly.
An identical replay returns original Admission even at capacity or after later heads.
Changed same event ID OR changed record for same key/generation/ordinal conflicts.
Initial ordinal must be 1 with no predecessor; thereafter require next ordinal and
exact previous publication ID. Missing or wrong predecessor returns ErrGap with
NO head/admission advance. A local store admits only its configured producer with
matching attributable key identity; this is a trusted local capability arrangement,
not authentication of untrusted remote claims. It is not a business reducer.

Record limits bound lifetime accepted records and private serialized record size;
no compaction/retention eviction. At capacity reject new records but still resolve
identical duplicates. Reopening preserves order/identity/metadata and detects
conflicting/broken/corrupt records. Records returns independent copies for the
private owner, never an observation gateway. No public errors expose raw payloads.
ErrConflict, ErrGap, ErrUnavailable, ErrUnknown, ErrCapacity, ErrOwned distinguish
outcomes. A commit not confirmed is unknown, never guessed rollback; poison store
until close/reopen. No durable Admission before SQLite Commit is established.
Options.Cut before_commit runs after transactional writes but before Commit;
after_commit runs only after Commit. Errors at cuts are unavailable before commit
or unknown/poisoned after; subprocess exit exercises real rollback/reopen.

## Run contract and failure semantics

Run(context, Source, Sink, confirmationStore, Limits) executes a finite pass.
Positive MaxCalls limits TOTAL sink calls, MaxAttempts limits per-record calls
PER PASS, MaxPending bounds source enumeration (source itself must be bounded).
Reject oversized snapshot as ErrCapacity without sending/dropping anything.
The caller owns policy/backoff between passes; no unbounded retry or worker spawn.
Cancellation propagates to dependency calls; dependencies must honor contexts.
No ongoing worker remains after Run returns. Immediate retry has no sleep contract.

Reconcile source records with durable confirmations. Never retire a source record
on a mismatched receiver/immutable admission. Group by scoped key/generation and
sort by ordinal; only send next unconfirmed record whose predecessor is confirmed
(or ordinal1). Round-robin groups so a failing key does not monopolize a sufficiently
budgeted pass. No global causal order across unrelated keys. A key with exhausted
retries, conflict, gap, permanent failure or unconfirmed predecessor blocks later
records on THAT key; independent keys may proceed. Never skip/coalesce/drop it.

Lost sink acknowledgment or ErrUnknown means retry identical Publication. The sink
may already have committed; dedupe must return that original admission on recovery.
Only after a matching durable Admission may the sender append to confirmation store.
Confirmation failure/unknown retains obligation; reopen resolves it through duplicate
admission. Confirmations say RECIPIENT ADMITTED, never projection applied. Physical
calls may repeat; logical admission is idempotent, not exactly-once network delivery.

Report counts calls and newly persisted confirmations; Issues carry only local
publication ID and stable code (e.g. failed, unknown, predecessor, receipt-conflict,
checkpoint). No raw sink errors/content in report. IDs remain protected private
owner metadata; this is not a public response taxonomy. Input/store/cancellation
errors return error. Per-key retry exhaustion/permanent failure may return a report
with issues and no global error; caller must inspect it and retry/repair later.
Limits exhausted may leave work even if no per-key error occurred; Run completion
is NEVER a claim that every source obligation was delivered.
Limits.Cut before_send precedes side effect; after_send follows matching durable
sink admission but precedes confirmation persistence. No environment/test-name
special cases are permitted in production code.

## Requirement-to-test mapping

| Obligation | Executable test / evidence boundary |
|---|---|
| P4/P16 durable identity/content, duplicate after restart | TestDeliveryDurableSink; real separate SQLite file |
| I/P1 immutable ID and ordinal conflicts, R03 predecessor gap | TestDeliveryDurableSink negative cases; no admission advance |
| P4 ordered keys/unknown outcome, no skip | TestDeliveryOrderingAndUnknown; reverse input, lost acknowledgment, identical retry |
| P4/B01 independent progress, bounded retries/cancellation/backpressure | TestDeliveryBoundsAndIndependentKeys; synchronous deterministic callbacks, no sleeps |
| P16 recipient crash before/after commit | TestDeliveryRealCrashRecovery receiver cuts; actual os.Exit child/reopen |
| P13/P16 publisher before-send and after-send crash | same test uses REAL Stage 1 source ledger; same pending ID restored |
| P16 sender confirmation before/after commit crash | same test ack cuts; absent/committed confirmation recovered correctly |
| ACK does not imply application | Admission/Report has no projection-application field/API; no projection execution |

Child helper alone is not a passing guarantee. Controlled crash cuts are known
pre-durability/confirmed post-durability; they do not simulate every device fault.
Tests retain source obligations and count unique sink records to rule out duplicate
logical admission. Additional disk faults/arbitrary interleavings, full S/O/A/V
services, production authentication/disclosure and second-backend interoperability
are not established. Prior ledger P01–P15/P17–18 remain regressions, not retested
production host authority. No full publication/profile conformance claim.

Next stage: authorized consistent snapshot + replay/observation gateway, coordinating
with perhap-client Stage 2A. Requires actual authorization/revocation fences and
snapshot-tail certificates; durable admission alone supplies neither.
