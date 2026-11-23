# Durable publication requirements — translation proposal

## Standing and evidence

Design only under spec-loop v1.1.1 step 2a. No API/wire freeze, kit, implementation,
commit or paid loop. Target github.com/version2beta/perhap, rddd-foundation,
758322024fab69cea3fbae7dfd471fe78adbae7a, initially clean.
Upstream candidate draft.6: 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87,
clean; all 17 SHA256SUMS entries verified. Manifest SHA256:
ab81ac9b68bf71d1e19c4f338b8e4b50df097a40218b4e41384288a6a5c9f816.
Candidate status is not ratification. SPEC is normative; conformance vectors
and implementation observations do not independently settle missing contracts.

Labels: U = upstream obligation; P = proposed local/runtime profile, not yet
normative; Q = missing shared contract for upstream/profile review.

## Traceability and proposed evidence

| ID | Source / obligation | Proposed observable acceptance evidence |
|---|---|---|
| U1 | §§3.2,5 stable scoped derivation identity, distinct immutable publication ID | keys survive restart and unrelated history; identical retries reuse ID/content; identity conflicts never overwrite |
| U2 | §5 durable ordered producer/key revisions, supersedes link | restart after each transaction; no duplicate/gapped committed ordinals; concurrent stale proposals cannot both advance same head |
| U3 | §5 unchanged content no new semantic publication; status recovery observable | unchanged current content creates receipt but no publication; current→unresolved→current produces ordered status transitions even with unchanged content |
| U4 | §5 preserve historical events; retract changes current view, not historical business state | set2/add5/set7 remains 7; retract does not delete prior record or undo business transitions |
| U5 | §§4.3,5 failed/invalid/incomplete recomputation is unresolved, not retract | coverage/evaluation matrix; stale value retained explicitly; valid complete disappearance alone retracts |
| U6 | §5 missing/conflicting revisions incomplete; transport freshness/continuity evidence | missing middle/status revision, disconnected consumer, expired cursor: no false authoritative latest claim |
| U7 | §§5,6 assertion attribution and disclosure authority, including metadata | deny content/existence/routing leaks; revocation affects queued retries/replay/snapshots, not just initial subscribe |
| U8 | §§5,7 consequential issuance needs EFFECT commitment | domain-declared binding issuance rejected by this provisional publisher, not exempted by an event name |
| U9 | §9 acknowledgment stages and unknown timeouts | distinguish ledger commit, transport receipt, recipient admission and application; ambiguous outcome resolved by lookup/retry identity |
| U10 | §§9,10 bounded work, explicit failures, stable diagnostics | disk full/backpressure cannot silently drop a record; repeated delivery error has stable operation identity; no protected error leakage |
| U11 | §9 projection revision/status/frontier independent of max UUID | same-event-frontier corrections delivered and applied using appropriate projection revision |
| P1 | Atomic ledger/head/receipt/outbox, not prescribed storage mechanics | crash yields either all transaction records or none; post-commit failure never loses scheduled item |
| P2 | Durable request identity plus compare-and-swap on head and evaluation generation | lost response retry returns exact receipt; changed same-request content conflicts; stale evaluation cannot overwrite newer evidence even after a no-op |
| P3 | Per-key ordered dispatch, durable consumer dedupe | duplicate after lost broker response has one admitted fact; later revision cannot be applied across unresolved gap |
| P4 | Snapshot plus cursor tied to one consistent read boundary | revision committed during snapshot is present in snapshot or resumable tail, never lost between them |

## Current Go evidence and migration implications

sdk/go/rev/contracts.go: Key scopes Context/Producer/Subject/Meaning; Intention
contains only ExpectedRevision, Op/Status/Value. Decision.Assertion deliberately
retains old Revision/PublicationID. Neither is a durable receipt or authorized
publication. Reconcile's status intentions do not carry stale value; the boundary
must load the prior value and availability, not interpret empty Value as an upsert.
Pure status reconstruction cannot establish live delivery continuity or authority.
CORE owns canonical domain replay; publication must not add replacement slots.

sdk/go/supervisor.go:170–211 builds ModelUpdate from lastValid, uses ordinary
NATS Publish for _model, and sets Nats-Msg-Id for JetStream event publishing.
There is no transaction linking those sends to a durable REV head/outbox/receipt.
sdk/go/service.go:120–177 uses plain queue subscriptions and sends an Ack;
this is not proof of durable recipient admission. Its replay path at 270–321
returns on NextMsg error, skips failed decode and sorts by ID. It cannot supply
the proposed certified publication ledger boundary merely by finishing a scan.
Existing behavior is preserved, not adopted as normative publication architecture.
No production defect repair or new conformance claim is made in this assignment.

## Missing shared contracts and counterexamples

These are underspecified protocol/profile decisions, not a discovered contradiction
in draft.6. Do not silently declare them upstream requirements or freeze a wire API.

- Q1 Sequence/status profile (§5): one ordinal stream for content and statuses,
  initial ordinal, supersedes meaning after a status, namespace/epoch identity,
  and restoration/producer transfer rules. Example: value r1, unresolved r2,
  recovered same-value r3. A consumer that counts only upserts sees a fake gap.
  Propose shared status-stream vectors and key namespace rules upstream/profile.
- Q2 Evidence freshness (§5): what establishes authoritative current observation,
  how long a caught-up certificate remains usable, and what happens on lost
  connectivity. Example: disconnected at r1 while unresolved r2 exists; contiguous
  local r1 is not evidence that r2 does not exist. Profile must specify a live
  freshness/continuity contract; never use silence or UUID time as proof.
- Q3 Resume/filter semantics (§§5,6,9): authorized snapshot boundary, opaque cursor,
  retention expiry, policy changes and non-disclosure of filtered keys. Example:
  global offsets 40→43 reveal hidden activity; per-key revision gaps may be
  authorization gaps, not transport loss. Specify no-leak reset/catch-up semantics.
- Q4 Retry/ack scope (§§3.2,9): distinguish durable producer acceptance, broker
  durability, receiver admission, and projection application; stable retry receipt
  after lost response. No exactly-once network guarantee follows from any one ack.
- Q5 Reconciliation admission (§5): preconditions on source evidence/rule generation,
  including no-op results and concurrent evaluations. Example: old valid result
  races a newer failure at unchanged content revision. ExpectedRevision alone
  cannot order a no-op against newer evidence. Propose scoped evaluation tokens
  and tests, leaving storage/fencing mechanism implementation-specific.
- Q6 Bootstrap/no prior key: complete absence should not invent a historical
  retraction, and initial unresolved state must not fabricate stale content.
  Agree explicit initial-state/advertised-key policy; current pure Reconcile
  accepts a zero Assertion but does not define this publisher lifecycle.
- Q7 Revocation's disclosure boundary (§6): broker retention and queued deliveries
  after policy change require a precise enforcement point. Already sent data
  cannot be recalled; deleting an outbox entry is not revocation at a broker.

Upstream additions should standardize these observable guarantees and shared
vectors; numeric codecs, broker subjects, SQL layout, leases, and database choice
belong in implementation/transport profiles. Review gates: Q1/Q5/Q6 before the
smallest ledger stage; Q2/Q3/Q7 before any externally observable transport stage.
