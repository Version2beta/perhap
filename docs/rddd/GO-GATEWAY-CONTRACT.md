# Minimum stable Go boundary for the TypeScript integration adapter

This records the required semantic boundary, NOT a new transport/wire API or an
implemented submission service. No TypeScript source changed. Observation uses the
new completed recovery API; old 3B1 Manager alone is not sufficient for integration.

## 1. Trusted configuration and identity before use

Provide an injected, trusted descriptor covering application/authority jurisdiction,
exact draft.6/profile versions, role capabilities, required/optional features, public
schema/model and manifest identity, semantic equality, retry/resume horizons, and
resource limits. Exact version match unless compatibility was separately reviewed.
Current Go Binding/Authority.Check must resolve this configured contract; the library's
Profile string alone is NOT a complete interoperable discovery descriptor.

The gateway resolves authenticated caller identity and authorized view from trusted
context; a client-supplied observer/view/epoch is a request, never proof of permission.
Keep registration, assertion/state authority, observation disclosure and submission
admission separate. Source/provenance access does not follow from projection access.
Deny hidden/nonexistent resources under a reviewed equivalent public failure policy.

## 2. Observation contract now available locally

Host-owned Recovery slot provides Open, Resume(opaque barrier token), explicit Replace,
bounded Recovered.Step and observed Close. Bind callback ownership to a local adapter
session generation before accepting data. Reconnect joins/cancels prior callbacks;
identity/policy changes require retiring old recovery/gate and an independently
approved new binding/view, never silent cache revival.

Full typed frames:
- snapshot / replacement: full Record, local Session fence, purpose-bound handoff proof;
- revision: immutable scoped Record with exact Revision/Previous;
- barrier: certified coverage through its Record and the new opaque resume proof;
- cannot-resume: no hidden offsets/head/existence detail; explicit replacement remains
  a separate newly authorized operation, not an automatic healthy recovery.

Proof binds authenticated observer, complete scope, generation/model/schema, descriptor/
authorization epoch, purpose and immutable record. Verify under the agreed trust root;
no Certified boolean. Only barrier-purpose tokens resume. Client validates snapshot
binding, predecessor chain and barrier before verified-through; identical deliveries
are idempotent, conflicting identities quarantine, ordinary later data never repairs a
gap. Replacement certificate resets current observation continuity only.

Keep value availability (including available null), evaluation validity, backend
completeness, delivery continuity and as-of freshness separate. No automatic
assertionStatus on arbitrary projections. No current projection or barrier is an
acknowledgment of a submitted event. No future/live lease follows from silence.

The recipient callback is the actual synchronous irreversible disclosure boundary,
not admission to an uncontrolled asynchronous queue. It must honor deadlines and
never reenter or defer release after return. A later transport binding must prove
its own queue/output revocation fence; wrapping a socket enqueue is not that proof.
Callback error/expiry means possible release, never rollback or remote erasure.

Opaque proofs are Go strings which MAY contain arbitrary bytes (the test issuer does).
A future bridge must encode them losslessly: do NOT blindly JSON-encode raw Go token
strings or infer offsets. No shared codec is selected here. Also prove lossless scalar
encoding: int64 Total / uint64 revisions are not generally JavaScript safe integers.
Representation/equality must be reviewed; no lossy Number conversion.

Current representative public model/schema is pair-sum/v1 / nullable-sum/v1. Domain
projection evaluation/rendering and normalized parameters belong to the registered
host; there is no automatic projector from existing runtime events or remote schema
execution. New domain models require explicit registration/equality review.

## 3. Submission contract still required — not implemented by this stage

The minimal adapter seam needs these operations (names descriptive):

- Submit: globally unambiguous stable EventID plus immutable semantic envelope and
  trusted asserting/admission attribution. Authenticate and validate scope/schema.
- Resolve: authorized lookup of that SAME identity; identical retry reconciles it.
- Optional Watch/Wait: owned cancelable status updates; not required if bounded
  Submit/Resolve are enough. Cancel stops waiting, not durable admission/effects.

Return complete authorized snapshots with status-resource Generation and monotonic
Version, separating durable submission receipt, admission pending/admitted/rejected,
and model/evidence-specific evaluated/evaluation-failed attempts. Admission remains
established even when later evaluation fails. Older status cannot regress newer;
same generation/version with different facts is conflict, not last-writer-wins.

Identity conflict rejects the changed attempt, not the original admission. Immutable
equality includes payload, context, causes, attribution and declared references;
transient credentials, retry route and attempt ID do not change logical identity.
Do not substitute reconciliation request ID, evaluation checkpoint or projection
revision for EventID. Any correlation must be explicit and must not imply inclusion.

Declare durable receipt/dedupe horizons and safe expiry rules. Distinguish definitive
rejection, transport-unknown, known precommit unavailable, expired receipt knowledge,
and no-retained-knowledge. Empty lookup cannot exclude in-flight admission. Neither
expiry nor unknown permits generating a replacement ID or reexecuting old work;
retain tombstones, retire namespace or require explicit authorized recovery.

Current reconciliation ledger receipts and delivery confirmation journals do NOT
supply this submission service. This is the minimum remaining Go capability needed
to unblock the submission portion of a TypeScript Stage 2B adapter.

## 4. Integration gate

Observation can now support a separately reviewed in-process adapter harness. Full
TypeScript submission integration remains blocked on actual Submit/Resolve/status
semantics above. Before a network adapter: settle descriptor/equality/encoding,
issuer/verifier trust and key/expiry policy, caller-to-owner-slot isolation, actual
output/revocation boundary and bounded transport disposal. Run shared S/O/A/V cases
through actual Go and TypeScript implementations; fake authorities or one backend
cannot establish production AUTH, transport security or cross-backend interoperability.
