# Backend-neutral publication interaction proposal

Not a frozen API, schema, encoding or wire format. Field groups below explain
information requirements, not JSON/Protobuf definitions. See REQUIREMENTS for
normative U requirements versus proposed P choices and unresolved Q decisions.

## Identities and minimal conceptual records

- Logical assertion key: context, accountable producer/service, subject, meaning
  (existing rev.Key is the starting Go representation). Not a credential.
- Reconciliation command: stable scoped request identity; authenticated submitting
  identity; key; expected head ordinal AND publication ID; current owner/evaluation
  preconditions; model/rule version; evidence closure/frontier/status; proposed
  pure Decision/Intention or sufficient deterministic recomputation inputs.
  Receipt equality excludes ephemeral delivery tokens; domain semantic encoding
  must be explicit. A client-supplied success/frontier flag is not trusted evidence.
- Immutable publication: unique event ID; key; durable ordinal; predecessor link;
  content operation (upsert/retract) or status transition; content/value availability
  or reference to retained prior content; schema/rule identity; trustworthy producer
  attribution; relevant evidence provenance. Separate historical cause links from
  supersedes links. Do not expose private source evidence merely because a consumer
  may read the conclusion. Public projection of provenance is policy-controlled.
- Durable receipt: request identity, accepted-no-change versus accepted-publication,
  publication reference if any, checkpoint/precondition result. Stable lookup must
  survive a lost response. Denied/conflict details have a separately authorized
  observation policy; not every caller may learn head existence.
- Delivery metadata: publication reference, attempt identity, transport receipt,
  scheduling/unknown status. Never alter immutable publication payload on retries.
- Current assertion response: key, latest verified ordinal/publication identity,
  current/retracted/unresolved, value AND availability, evaluation/coverage and
  certified boundary/freshness information. Preserve tombstones and unresolved
  states; absent key is not shorthand for either of them.

Values may be Go-private internally. A shared encoding, canonical equality,
integer range, and protocol version handling must be approved before exchange.
If selecting INTEROP-JSON, obey its profile rather than assuming json.Marshal
supplies JCS. No exact field spelling, UUID version or counter width is approved.
Ordinals cannot silently wrap; exhaustion is explicit backpressure/unavailability.

## Producer interactions and dispositions

AcceptReconciliation (descriptive name): command -> durable receipt or explicit
stale/conflict/denied/unresolved/backpressure; timeout -> outcome unknown.
LookupRequest: same authenticated scope/request -> persisted receipt, known absent
at an identified storage boundary, or unavailable/unknown. A known-absent lookup
alone is not proof an outstanding concurrent accept cannot still commit; retry
uses the same request identity. Never issue a new request/ID merely after timeout.
Reusing a request ID with different immutable meaning is a conflict.

Pending publication, transport accepted, recipient admitted and projection applied
are separate facts. Only a dedicated recipient acknowledgment can establish the
last two; a broker acknowledgment is insufficient. Request replay returns the
original receipt, not an assertion that its earlier publication is still current.
No remote consumer or broker availability is required to atomically accept work
into the local ledger, assuming capacity and authorization allow acceptance.

## Reader/catch-up protocol (Q2/Q3 proposal)

1. Authenticate observer; authorize scope and schema. Return no key existence,
   diagnostic payload, event counts or cursor details beyond permitted disclosure.
2. Obtain a consistent authorized snapshot AND resume position at one ledger cut,
   including current/retracted/unresolved assertions and coverage. Or read full
   authorized history if historical replay is required. A current-view snapshot
   is NOT historical event reconstruction and cannot replace missing business events.
3. Resume authorized changes strictly after that cut. Page responses certify a
   bounded inspected boundary and their authorized coverage, not future finality.
   Persist admitted records and cursor/checkpoint together before acknowledging
   durable consumer acceptance. Apply projection idempotently or track it as a
   separate durable stage; do not claim application merely on receipt.
4. Each consumer verifies publication identity, key ordinal continuity and status
   semantics. r3 before r2 means buffer within bounds, mark affected key incomplete,
   request repair. Different content for one identity/ordinal is conflict, never
   last-writer-wins. Identical repeats cannot execute domain processing twice as
   new admitted facts. Receiving all known ordinals does not establish live freshness.
5. Resume cursor is opaque, scope/observer-policy/epoch bound, integrity protected.
   A filtered feed must not leak hidden global offsets or imply a missing authorized
   event when denied keys were omitted. Policy changes invalidate/rebind the cursor
   and require authorized rebootstrap; do not reveal the denied interval's size.
6. Retention expiry or unknown cursor returns explicit resync-required/unavailable,
   not an empty successful tail. Freshness cannot be asserted after disconnect by
   assuming an unresolved status was delivered. Reauthorize snapshots and historical
   repair reads. If prior revisions are no longer permitted/retained, a verified
   current baseline can support current queries under an agreed profile but cannot
   certify full historical replay; distinguish these capabilities explicitly.

No projection-specific revision is inferred from highest event UUID, broker
sequence, request ID or a different key's revision. A multi-key projection needs
its own durable revision/frontier contract. No global order across contexts is
required. Publication revision order is not automatically a business causal edge:
receiving domains must declare canonical ordering/dependencies when that order
matters to historical transitions. Nor does a collection of per-key atomic commits
certify an atomic multi-key projection; that needs its own scoped consistency cut.
Freshness duration, policy binding and certified-baseline semantics
remain Q2/Q3 approval gates; this proposal does not invent an authority proof.

## Optional NATS/JetStream binding — later, not normative rDDD

Backend-neutral: identity, per-key revision/continuity, admission/disclosure,
atomic local persistence, unknown outcomes, snapshot/tail consistency and repair.
Binding-specific: subjects/routing and metadata privacy, credentials/ACL mapping,
stream durability/replicas/retention, durable consumer configuration, explicit ack
policy, redelivery/flow limits, publish receipt translation and cursor mapping.

Possible implementation: dispatcher sends immutable ledger bytes with publication
ID as Nats-Msg-Id; wait for PubAck before recording transport acceptance. Lost
PubAck -> retry same record. JetStream dedupe windows are an optimization, never
the lifetime identity ledger. Consumer persists admission/checkpoint before an
explicit ack; redelivery after crash remains safe. No broker defaults are assumed
or selected here; the binding must prove actual settings under the chosen version.
A broker stream sequence is not a producer/key revision or rDDD frontier proof.

Prefer a trusted gateway for dynamic recipient authorization. Shared subjects,
queue groups or wildcard grants are not evidence of observation authority; subject
names and headers can leak identifiers. The binding must either enforce revocation
at every new delivery (including retained replay) or deny that access path.
Offline consumers and cached prior disclosures cannot be made to forget data.
Plain NATS _model notifications are insufficient as the durable protocol; do not
retrofit its current code path by renaming fields or treating msg.Ack as persistence.
