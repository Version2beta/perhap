# Publication boundary design — recommended, not approved API

Authority, source pin, U/P/Q labels and open contracts: PUBLICATION-REQUIREMENTS.md.
No implementation or wire format is frozen. No semantic contradiction found;
profile decisions remain open and block claiming an implementation-ready protocol.

## Recommend one transactional ledger with an outbox

Pure CORE/REV result + admitted evaluation evidence
  -> trusted reconciliation admission
  -> ONE durable transaction: receipt + evaluation checkpoint + key head
     + immutable publication record + pending outbox reference
  -> ordered retrying dispatcher
  -> authorized disclosure gateway / optional transport
  -> receiver durable admission and dedupe
  -> historical replay AND separate current-assertion view

The ledger, not a broker, owns publication identities and revisions. Outbox rows
reference immutable ledger records; retries never regenerate an event ID or
content. A local commit is one logical publication, but the network may deliver
it many times. Receiver uniqueness on publication ID and producer/key/ordinal
(with immutable-content equality checks) prevents duplicate admission. It does
not make arbitrary downstream effects exactly-once.

Recommended first persistence candidate: a single-owner embedded transactional
store such as SQLite with a documented durable configuration, rather than a
hand-built append log. Requires explicit later dependency/storage approval; no
package is added now. One database keeps the head, history, receipts and outbox
atomic without distributed transactions. Durability claims require real reopen/
process-crash tests and storage assumptions; an in-memory fake cannot prove them.
No claims against media loss, lying fsync or restored stale backups.

Alternatives: direct broker publish then update head loses dedupe on lost response;
head then direct publish loses work on crash; two stores without a transaction
have the same dual-write hole. Broker-as-only-ledger can work only with an explicit
CAS/idempotency/recovery contract beyond finite dedupe windows. A custom fsynced
append log avoids a dependency but adds framing, torn-write recovery, indexes,
compaction and locking; not the smallest demonstrably reliable implementation.
A general distributed workflow engine is unnecessary for the bounded first stage.

## Transaction and concurrency rules (proposal P1/P2)

Start with one owner/one key per command; unrelated keys have no global semantic
order. Serialize ownership through storage-enforced fencing, not just a Go mutex.
Second owner is rejected in the minimal deployment; a later multi-owner design
needs store-checked generations and takeover tests. Scope/evidence admissions
must be serialized with checkpoint acceptance or supply a trustworthy comparable
CAS token. Frontiers may be vectors/opaque certificates, not max UUIDs.

1. Authenticate/authorize the producer and target through a trusted host. Reject
   consequential publication unless a separate EFFECT contract has authorized it;
   the minimal publisher accepts only declared provisional assertion schemas.
2. Load durable request receipt FIRST. Same request and identical immutable command
   returns original outcome even if head advanced; changed command conflicts.
   Bind request identity to authenticated producer, scope and canonical command.
   Keep receipts for the supported retry horizon; after expiry do not silently
   reinterpret an old ID as new work. New epochs require explicit recovery policy.
3. Verify head ordinal/publication ID, owner fence, active rule version, and current
   evaluation/admission generation. Recompute/revalidate a stale intention; never
   merely patch its ExpectedRevision. Caller cancellation before commit cannot
   certify non-commit if a storage response was lost.
4. Load stored previous assertion; check intended transition agrees with verified
   computation. Reconcile's empty status Value does not erase retained content.
   Handle initial never-published keys via agreed Q6 policy, not fabricated events.
5. For a real change allocate next per-key ordinal and globally unambiguous opaque
   event ID, link prior publication, persist immutable record and update head plus
   pending outbox in the same transaction as receipt/checkpoint. Allocation that
   rolls back is never visible. Status-only unresolved/recovery uses the same stream.
6. For a no-op, persist receipt and evaluation checkpoint WITHOUT a new publication
   ordinal. This prevents an older evaluation becoming current just because content
   stayed unchanged. Late older inputs trigger new authorized reconstruction, not
   reuse of an obsolete evaluation token.
7. Reply only after durable commit. An uncertain commit result is unknown; lookup
   by request ID determines outcome. On recovery rebuild heads from ledger/verify
   invariants; do not allocate using a remembered process-local counter.

Retraction is a new record. History never shrinks because current value changes.
No queued corrections/status transitions are coalesced away in the first stage:
consumers need continuity and historical interpretation. A publisher's accepted
head may be newer than its transport-accepted head, which may be newer than a
consumer's applied head. Expose these distinctions, subject to authorization.

## Delivery and recovery

Send the oldest pending item per key only after its prior item has a durable
transport-acceptance receipt (or explicitly authorized terminal handling).
Lost acknowledgment leaves the item pending/unknown; retry the SAME record.
Broker acceptance is not recipient admission/application. Record attempts as
mutable scheduling metadata, never by changing publication contents. Old delayed
network duplicates may still arrive after newer revisions; recipients verify
contiguity/dedupe rather than relying on transport FIFO alone.

Bound queues, retries, fetch sizes and backoff; infrastructure timers are allowed
outside pure evaluation. On exhausted capacity reject/backpressure acceptance
before commit, or retain durable pending work and report delivery unavailable.
Never drop or silently skip a permanently denied/failed item to advance the stream.
Independent keys may proceed. Resource/retention choices need an explicit policy.
Liveness assumes store availability, authorized reachable recipients and eventual
successful delivery; this design promises no unconditional delivery deadline.

Observation policy is checked at actual disclosure as well as producer acceptance.
Store immutable author attribution/proof references but do not freeze recipient
permissions forever. A revoked queued send becomes held, not deleted or rewritten.
Replay, current reads, history reads, notifications, cursors, diagnostics and key
existence all need the same policy boundary. Avoid granting direct broker access
that bypasses these checks. Publication to a third-party broker is itself a
disclosure requiring approval; purging later does not undo prior disclosure.
For policy changes mid-send, the transport profile must define/fence the disclosure
linearization point. A pre-send check alone cannot prove revocation safety.

## Idiomatic Go proposal and migration

Use concrete value records and a small storage transaction owner plus narrow
ledger/transport/authorization interfaces. Descriptive operations: accept command,
lookup receipt, read consistent snapshot, read authorized tail, dispatch pending,
record transport receipt. These are not frozen exported signatures.
Pass context.Context through storage/network waits; the acquiring service owns
Close, transaction rollback and worker shutdown. Keep primary errors when cleanup
fails. Distinguish conflict, stale precondition, denied, unresolved policy,
backpressure and unknown commit/transport outcome; no raw internal error disclosure.
Bound workers by active keys, serialize within a key, observe all worker completion.

Do not expand rev.Intention into a wire envelope or treat Decision.Assertion as
already admitted. Add boundary-owned evidence/identity/receipt records around the
existing pure API after review. CORE/REV need no semantic changes. Existing NATS
runtime stays untouched until an optional binding passes the protocol tests.
