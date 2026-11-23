# Durable Submit/Resolve — bounded design for approval

Proposal only: no frozen Go signatures, kit, implementation, model calls or commits.
No new stage number assigned. This extends the gateway contract, not its authority.

## Verified state and sources

Clean rddd-foundation HEAD 5243f8fe387f67829d4aef41c279bad90ba4b333.
Specification tree clean at 9a80977efaf2ff740a257f6e30314dac42ec384e;
draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87 is an ancestor. All19 checksums
pass; SHA256SUMS digest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Installed spec-loop v1.1.1 script matches repository script byte-for-byte.

Inspected GO-GATEWAY-CONTRACT, RECOVERY-RESULTS, publication stage plan; Event,
Service/Domain/supervisor ingress/replay; CORE/ledger/delivery contracts and ledger
schema/recovery; ratified profile I/S/A/V and vectors, candidate portable-json rules.
Historical plans are design inputs, not current authority. Prior 24-group/Go/race/vet
results are recorded baseline evidence, not tests newly run in this design pass.

Read-only client comparison: clean perhap-client rddd-foundation at
81641540cb8e79d002cf9599ff33118b44e12df0, src/submission-types.ts, submissions.ts
and docs/rddd/STAGE-2B.md. Its semantic string is explicitly NOT an encoding/JCS;
trusted attributionBinding is a binding reference, not a payload principal. No client
files modified and no client acceptance is attributed to this design.

## 1. Existing reusable interfaces and important mismatch

| Existing component | Reuse | Cannot honestly establish |
|---|---|---|
| perhap.Event | ID/context/entity/type/data/causation/correlation interpretation; existing UUIDv6 producer convention | Authenticated asserting attribution, durable receipt or admission facts |
| ledger.Ledger | Single SQLite owner, append/transaction/recovery methods, checked retry/conflict pattern, durability settings | Event-ID admission: Accept is keyed reconciliation request/evaluation, not submitted event |
| delivery.Store | Reference durable identity/conflict and admission/ack separation | Arbitrary event receipt/status store: it accepts ledger publications only |
| core.Evaluate | Existing pure finite-history/model/evidence evaluation | Transport admission, authenticated authority, effects, or runtime NATS storage |
| viewjournal / observation.Recovery | Existing projection acceptance and independently scoped certified observation | Submission acknowledgment or event inclusion inferred from a barrier |
| Service / supervisor | Existing JetStream file-backed history and typed domain handlers | Permanent semantic dedupe/conflict or versioned submission knowledge |

Critical finding: the legacy runtime history is in JetStream, not the SQLite
publication ledger. Service.Run uses a plain NATS queue subscription; supervisor
publishing sets Nats-Msg-Id. Window dedupe / broker duplicate windows and a PubAck are
NOT the proposed permanent event identity/admission contract. Raw legacy ingress has
no trusted asserting-attribution field or Submit/Resolve fact journal.

Consequently a SQLite receipt alone must NOT claim an event was admitted into that
legacy runtime. The authoritative admission jurisdiction must be approved explicitly.

## 2. Recommended bounded admission/storage choice — approval required

Recommend an opt-in integrated SQLite owner: add an immutable admitted-event portion
and submission fact records alongside the existing publication ledger, reusing its
SQLite driver, connection ownership and transaction/durability policy. This is a new
record family, NOT a new storage engine or substitution of publication identities.
For gateway traffic, this event portion is the declared authoritative admission log.
Expose its admitted finite cuts to a trusted CORE evaluation host. It is not a second
uncoordinated mirror falsely claiming to be JetStream's admission state.

```
trusted caller + registered codec -> Submit -> receipt v1 / pending
  -> trusted admission decision -> authoritative event + admission v2 + pending work
  -> bounded trusted CORE evaluation -> identified evaluation facts v3...
Resolve -> latest complete authorized facts (same EventID + status generation)
publication / projection / observation remain independently acknowledged resources
```

Receipt and later admission/evaluation are separate commits/facts. Admission commit
atomically establishes the authoritative event, attribution, terminal admission fact
and any required pending evaluation obligation in that SAME owner transaction.
No atomic transaction across SQLite and JetStream is claimed.

Existing schemas reject unknown tables and record bodies. Do NOT hide event records
inside fake reconciliation keys or relax integrity scanning to ignore unknown data.
A same-file extension therefore needs an explicit versioned, opt-in schema migration
and combined recovery entry point, retaining old ledger APIs on unextended files and
publication operations through the integrated owner. No automatic migration. Old
binaries will reject the extended file; this is a compatibility approval question.

Alternative if old-binary compatibility is mandatory: a narrowly scoped separate
SQLite submission file using the SAME established durability policy, with no atomic
claim over publication/projection stores. This avoids altering old files but adds a
separate owner. It is an alternative for review, not an assumed new subsystem.

Neither option automatically integrates legacy JetStream execution. If actual
JetStream admission is required now, the design must instead provide durable
same-ID semantic lookup/admission in that authority, reconcile response loss, and
handle ingress bypass/permanent dedupe beyond the broker window. That is more than
publishing then marking a local row admitted. Do not silently add that bridge.

Untracked legacy EventIDs cannot safely be treated as new globally unique work:
require an approved exclusive ingress namespace/cut or trusted legacy identity
inventory/import before activating the new authority. Never reset IDs by changing
status generation. Existing legacy APIs can remain available, but their traffic is
outside gateway guarantees unless explicitly integrated.

## 3. Proposed API/outcomes — descriptive, not frozen signatures

Public facade:
- Submit(ctx, EventEnvelope, synchronous ReplyRecipient): establish/reconcile the
  immutable identity, process a bounded admission step when possible, deliver latest
  complete authorized status. No client admission/evaluation/success flags.
- Resolve(ctx, EventID, synchronous ReplyRecipient): read authoritative retained
  knowledge at a cut; never interpret a miss as rejected or safe new work.
- Close: cancel/join owned work and close the shared owner only according to explicit
  ownership. Closing the facade cannot unexpectedly dispose a borrowed ledger.

EventEnvelope references perhap.Event plus registered schema/interpretation identity
and a requested attribution binding. Trusted host resolves requester versus immutable
asserting principal/delegation. Credential rotation is not new event meaning.
Transient request/attempt IDs are optional tracing only, never dedupe identities.

Facts: EventID, durable status Generation/Version, original durable Receipt,
Admission(pending/unresolved/admitted/rejected), Evaluation knowledge and identified
attempt/model/evidence/attribution. Preserve earlier attempt facts privately. Public
complete facts have one declared schema; prefer full-public-status permission or
opaque denial over silently omitting fields under the same version.

Reply kinds: facts, no-retained-knowledge, blocked, denied, identity-conflict,
unavailable. Expired is reserved only if an actual approved expiry policy exists;
initial recommendation is no receipt expiry/pruning, bounded lifetime capacity.
Errors distinguish known precommit unavailability, commit-unknown, cancellation and
integrity failure. None of these is a persisted rejection. Failed response release
is separate from an unknown storage commit: both may require same-ID reconciliation.

Host-only capabilities (not present on the client facade): admission decision and
bounded pending evaluation/verified evaluation acceptance. Wire callers cannot supply
authoritative result objects. A concrete trusted evaluator uses existing core.Evaluate
on an admitted finite cut; no automatic reducer emissions or effects. Bind results to
actual attempt/model/source cut/event participation, not just presence in History or
an unrelated successful projection. Domain-invalid completed evaluation differs from
inability to evaluate. Blocked/not-attempted events do not gain fabricated success.

## 4. Identity, equality, versioning and authority

- One EventID identity decision across the owner's supported contexts; not scoped by
  request ID, producer command, checkpoint or publication ordinal. Canonical UUIDv6
  is proposed for the existing Go runtime boundary; CORE opaque IDs stay unchanged.
- Equality covers context/entity/type, schema, content, causation/correlation and
  trusted asserting attribution. Decide reference semantics in the registered codec;
  never infer business causality from UUID time. Different caller credentials alone
  do not conflict if trusted immutable asserting identity/delegation is unchanged.
- Normalize/validate through a versioned registered codec before identity comparison;
  preserve raw first input privately only if needed, never use it as semantic truth.
  Reject lossy Unicode/ambiguous duplicate keys and unsupported numeric values.
  Execute from the SAME normalized representation used for equality: legacy typed
  json.Unmarshal treats 1 and 1.0 differently for integer fields, so merely comparing
  them equal while evaluating original bytes would make outcome depend on first retry.
- Candidate portable-json is not automatically ratified. Propose bounded semantic
  JSON equality for the selected public schema (1 equals 1.0; missing differs from
  null; no unsafe integer conversion), with no JCS byte/digest claim or new wire
  format. The client must use the same approved interpretation before track().
- Identical Submit returns latest facts plus original receipt, not an old v1 snapshot
  that could regress v3 knowledge. Conflict rejects the changed attempt and preserves
  the original immutable event/admission. No semantic fact change on identical retry.
- Each changed complete fact advances that event's version exactly once; no wrap.
  Receipt-only v1, later admission v2 and evaluation v3 remain distinguishable.
  Admission terminal facts do not regress; evaluation may later fail without erasing
  admission. Host attempt IDs are idempotent and conflicting reuse is refused.
- Trusted identity/lookup authority precedes any public existence/status diagnostics.
  Distinguish unresolved authority evidence (blocked) from definitive opaque denied.
  Authorization change does not rewrite stored admission facts.
- Use a trusted synchronous disclosure fence for ReplyRecipient, ordered with status
  revocation, like observation's proven boundary. Do not disguise status as a fake
  projection or return protected facts after an unlocked precheck. Callback/error
  metadata is policy-controlled, errors canonical and private causes not exposed.
- Preserve ledger lock order: owner mutex -> host acceptance fence -> transaction,
  for BOTH admission and existing publication acceptance. Never acquire these in
  reverse order or reenter the ledger from a host callback. Validate source admission
  cut/model freshness at evaluation acceptance; stale results require explicit new
  work, not relabeled counters or automatic retry.

## 5. Durability, ownership and failure contract

Reuse existing exclusive owner/SQLite connection, DELETE rollback journal,
synchronous EXTRA/fullfsync, busy timeout0 and exclusive transactions. One owner
serializes concurrent same-ID calls; unique constraints/replay enforce one immutable
identity even across process reopen. Shared recovery validates both record families;
independent host inventory/fence rejects stale backup restoration. No distributed
lease/failover or crash-proof external authority service is added.

Before first transaction / controlled pre-durability failure: no new receipt for this
attempt; report unavailable/canceled as justified. This never excludes another earlier
in-flight request. Once commit is issued and outcome cannot be confirmed: unknown,
poison affected owner until reopen, then authorized Resolve/identical Submit reconciles.
After durable commit before reply: receipt/admission/obligation survives; no duplicate.
Corrupt/truncated evidence refuses authoritative service; no empty fallback, negative
lookup certification or automatic generation reset. Cancellation after receipt/admission
cannot undo it. Response release uncertainty never erases stored facts.

Configure MaxEvents, pending work, fact-history/transition/byte limits and operation/
cleanup deadlines. Reserve completion space where receipt/admission creates future
obligations, or refuse before acceptance. No silent eviction, lost work or capacity-
blocked duplicate lookup. All accepted identities retained for store lifetime; on
capacity refuse new work. No old identity recycled after restart/namespace transfer.
No background retry/watch loops or client cache. Context cancellation and explicit
Close own all work; noncooperative callbacks invalidate the declared time bound.

## 6. Requirements -> planned independent evidence

| Source | Proposed observable test / owner |
|---|---|
| I/S01 | Lost receipt/admission reply, restart, original attribution and same-ID retry; real owner transaction/reopen |
| S02 | Payload/type/cause/context/schema/attribution conflicts; concurrent identical/conflicting inputs; immutable original preserved |
| S03/S04 | v1/v2/v3 full facts, failed evaluation preserves admission; immutable versions; delayed/conflicting replies through independent merge oracle (server checks alone do not prove client behavior) |
| S05 | Pause earlier Submit before commit while Resolve sees a valid empty cut; no rejection/noncommit inference |
| S06 | Declare lifetime retention/no reuse; full expired-receipt branch unexecuted unless expiry is separately implemented |
| S07/A | Missing authority vs denied; forbidden/nonexistent equivalent failures; caller flags/attribution cannot forge state; actual status release fence |
| S08 | Cancel after durable admission, Resolve still admitted; independent projection42 never confirms another EventID or effect |
| P14a | Real after-commit response-loss cut -> unknown -> same-ID lookup/retry resolves |
| P14b | Known pretransaction failure with no earlier pending request -> unavailable, no receipt; not ambiguous success |
| P14c | Actual corrupt/truncated reopen refuses authority, earlier unknown remains unknown |
| B01/local | Capacity before commit, retained duplicate lookup at capacity, reserved outcome completion; no version wrap/drop |
| Existing I/P/O/A | Original suites, observation revocation/recovery, publication ordering and old storage opening regressions |

Cheapest disproof first: event E receipt/admission commit, kill before reply, reopen,
concurrent identical/conflicting Submit and Resolve must yield one event and monotonic
full facts. Then stage a no-knowledge lookup racing a paused submission, and test
canonical-number retries against actual typed evaluation. Use real process cuts,
deterministic channels and actual SQLite bytes, not simulated fake persistence.

After approval: precise contracts/mapping, independent reference outside checkout,
all new tasks fail on stubs and all pass under unchanged judge, clean verified kit,
bounded loop, full regression/race/vet/judge and exact 55-row counts. Client-only or
partially exercised rows remain unexecuted; do not add counts based on related tests.

## 7. Exclusions and approval questions

No HTTP/WebSocket/new transport, automatic NATS dispatch/retries, production AUTH,
effects, client adapter/cache or upstream changes. No observation/submission identity
conflation, source history rewrite or generic storage-engine refactor.

Approval needed before kit:
1. **Admission jurisdiction:** approve authoritative SQLite gateway event admission
   plus actual CORE evaluation/host integration, keeping legacy JetStream ingress
   outside that guarantee? Or require actual JetStream runtime admission now, which
   needs a separately designed permanent dedupe/lookup/ingress bridge?
2. **Storage compatibility:** prefer opt-in same-ledger-file extension/combined owner,
   accepting explicit migration and old-reader rejection of extended files, or require
   old-binary readability and use an isolated file under the existing SQLite policy?
3. **Interpretation:** approve canonical UUIDv6 at this Go boundary and one registered
   semantic JSON/event schema codec, normalized before both comparison and evaluation?
   No assumption that the client's opaque semantic string is already canonical/trusted.
4. **Retention/status scope:** approve lifetime retained identity/facts with bounded
   refusal, no expiry/automatic retries, host-only bounded evaluation and all-or-nothing
   declared public status disclosure? S06 expiry remains explicitly unexecuted.

No upstream normative ambiguity identified. The admission jurisdiction and compatibility
questions are deployment/design decisions, not permission to silently choose new truth.
