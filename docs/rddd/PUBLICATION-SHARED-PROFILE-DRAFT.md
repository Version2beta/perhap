# Minimal durable REV publication semantic profile — review draft

Proposed addition for review in rDDD-spec, NOT an adopted specification, exported
API or frozen wire format. Base: draft.6 commit
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87. Provenance, TypeScript comparison and
counterexamples: PUBLICATION-DECISIONS.md. Nothing is written upstream here.

The MUST terms below describe the candidate profile IF approved. They do not
retroactively change draft.6 or assert another implementation's conformance.
Normative observable behavior is separated from implementation choices.

## Scope and concepts

This profile covers acceptance of pure evaluation results for durable publication
of non-constitutive derived assertions. It is not an EFFECT execution protocol,
a consumer historical replay replacement, or proof of observation authorization.
Existing draft.6 attribution, disclosure and consequential issuance obligations
continue to apply. Readers receive only authorized evidence.

Distinct identities: logical derivation key; immutable publication event ID;
scoped publication stream; accepted evaluation checkpoint; evaluation request;
projection stream/revision; submission disposition version; resume cursor.
A binding MUST distinguish them and MUST NOT infer one from another or from max UUID.
Producer identity covers context/service/subject/meaning. Stream namespace or
continuity generation MUST be unambiguous within its advertised identity.

## Candidate normative semantics

### S1 Publication sequence and identity (Q1)

1. Each scoped derivation key MUST have one contiguous publication sequence,
   beginning at 1. Every published content change, retract, unresolved transition
   and recovery MUST participate in it. No-op evaluation MUST NOT consume it.
2. Each new publication MUST have its own stable immutable event ID; later sequence
   members MUST identify their immediate preceding publication. Identical retry
   MUST reuse identity and immutable content; conflicting same-ID or same-stream/
   ordinal records MUST be diagnosed, never overwrite prior history.
3. Restart/owner transfer MUST preserve sequence and identity. An explicit stream
   generation change MUST be distinguishable from continuation and require verified
   replacement/rebootstrap, not compare/reset ordinals as if they were continuous.
   It MUST NOT reuse a prior publication event ID. Exhaustion cannot silently wrap.
4. Revision records MUST remain historical events at their canonical domain positions.
   Current-assertion changes MUST NOT rewrite earlier payloads or undo transitions.
5. Status recovery with unchanged value MUST still be observable as a status revision;
   this profile selects ordered unresolved publication when publication is authorized,
   strengthening draft.6's SHOULD to a profile requirement for advertised keys.

### S2 Evaluation acceptance and retry (Q5)

1. Every NEW accepted evaluation MUST advance a durable acceptance checkpoint,
   including accepted no-ops and accepted evaluations that report failure or
   incomplete evidence. Replaying the same accepted request MUST return its original
   receipt without advancing the checkpoint or allocating another publication.
2. Acceptance MUST validate expected assertion-head identity/presence AND prior
   accepted evaluation checkpoint. It MUST also establish that the evaluated source/
   rule/admission generation is acceptable at the declared acceptance point under
   trusted host rules. A caller's generation/coverage assertion is not that proof.
3. The host MUST prevent a result known to predate newer accepted evaluation or
   relevant source advancement from overwriting that state. It MUST reject/recompute
   when preconditions are stale or source compatibility cannot be established.
   Retry with refreshed preconditions alone MUST NOT launder stale computation.
4. Receipt, evaluation checkpoint, any publication record/head advancement, and
   durable scheduling obligation MUST have one atomic acceptance outcome. No
   committed publication may lack recoverable scheduling, nor advance a head
   without its immutable record. A no-op still commits receipt/checkpoint together.
5. A lost acceptance response MUST leave outcome unknown until resolved. Exact
   logical retry MUST be safe through supported restart/retention conditions.
   Different immutable request meaning under one scoped retry ID MUST conflict.
   Once retention cannot prove prior outcome, the host MUST NOT silently treat
   that ID as fresh; it must expose unknown/expired semantics under authorized lookup.
6. Checkpoint order MUST NOT be represented as proof of live freshness, complete
   source history, successful evaluation or effect authority. Failed evaluation
   can be freshly accepted evidence of failure. Source ordering may be scoped;
   no total order over independent scopes or incomparable frontiers is implied.

### S3 Bootstrap and availability (Q6)

1. A never-published key MUST be distinguishable from a published retracted key.
   Complete valid initial absence MUST NOT fabricate a retraction/publication;
   it still records accepted evaluation/checkpoint under S2.
2. Initial unknown/invalid/failed/incomplete evaluation MUST NOT fabricate a prior
   value, publication or permission to disclose the key's existence.
3. Initial unresolved publication MAY occur only when trusted domain rules establish
   the key and authorize that observable unresolved status. If published, it MUST
   use first ordinal 1 and explicit value unavailability. No previous value or
   predecessor event may be invented. Absence of authority evidence blocks disclosure.
4. After initial unresolved without a value, a valid present value MUST establish
   content in a new revision. Available null MUST remain distinct from unavailable.
   When complete valid recomputation establishes that a prior advertised conclusion
   is no longer supported under its domain contract, the producer MUST issue a
   retraction; it MUST NOT delete historical records.
5. Repeated unchanged current/retracted/unresolved state MUST NOT produce unbounded
   duplicate semantic publications. A recovered state transition is not such a retry.

### S4 Limited consumer compatibility, not a full observation profile

An authorized projection snapshot MAY establish a baseline at revision 40 or another
verified cut; S1's start-at-1 rule is about producer publication history, not the
client's starting projection cursor. Snapshot baselines MUST NOT certify receipt
of missing historical events. Projection replacements, assertion deltas and
historical event records MUST be semantically distinguishable.

This profile MUST NOT advertise full CLIENT observation support without the
separate snapshot/tail barrier, trusted as-of coverage, authorized resume/reset,
retention, versioning and revocation contracts identified by the TypeScript
proposal. Quiet/disconnected delivery is not a continuing authority guarantee.

## Shared conformance vectors to add upstream after review

These are proposed tests and expected semantic outcomes, not executed tests or
new fixture bytes. Independent implementations should consume the same vectors.

| ID | Given / race | Expected |
|---|---|---|
| Q1-1 | current value at 1, unresolved at 2, recover same content | ordered status at 3, no new content identity for unchanged content meaning; three distinct publication events |
| Q1-2 | accepted revision 1; response lost; retry after restart | same event/ordinal and receipt; one checkpoint advancement for that request |
| Q1-3 | same stream/ordinal, different event or immutable value | conflict/incomplete, no last-writer-wins |
| Q1-4 | restart or backup with process counter reset | durable continuation or explicit unavailable recovery; no silent ordinal reuse |
| Q1-5 | producer stream begins at 1; authorized projection bootstrap at S/40 | valid distinct identities; no demand to replay projection 1–39 or claim historical continuity |
| Q5-1 | E2 accepts newer evidence as a no-op, then E1 arrives with matching publication head | E1 stale checkpoint/source rejection; publication ordinal unchanged by E2 |
| Q5-2 | higher caller generation with stale/unverified source or old rule version | no trusted freshness granted; reject/recompute, not accepted-current |
| Q5-3 | source advances after evaluation, before ledger acceptance | reject/revalidate under source fence; no stale certified result |
| Q5-4 | newer failed evaluation accepted, then old valid result retries | unresolved remains; old computation cannot restore current authority |
| Q5-5 | crash at each write before/after acceptance commit, including no-op | all-or-none receipt/checkpoint/head/history/scheduling; stable retry |
| Q6-1 | never-published key, complete valid absence | no publication/retract; checkpoint advances once |
| Q6-2 | unestablished key, failed evaluation | no invented content or public existence; only separately authorized internal evidence |
| Q6-3 | established observable key, initial unresolved | revision 1, no value/predecessor; valid null upsert later is available |
| Q6-4 | published unresolved without value, later complete valid absence | ordered retraction if domain declares loss of support; not rewritten initial absence |
| Q6-5 | previously retracted key evaluated absent again | checkpoint advances, no additional semantic retract |

## Deliberately non-normative choices / remaining gates

No SQLite requirement, SQL schema, Go interface, storage/transaction library,
UUID version, numeric byte width, transport, subjects, client union or JSON
field names are prescribed. A binding must choose non-lossy representation and
version negotiation; INTEROP-JSON applies only if selected. Acceptance evidence
may be enforced through a local serialized authority or a verified cross-owner
fence; neither caller numbers nor a Go mutex alone prove the distributed case.

Owner/spec review must adopt or amend S1–S3 and agree authoritative source/epoch
and bootstrap policy. The TypeScript documents support the consumer distinctions
but do not approve these producer extensions. Q2/Q3/Q7 and live observation remain
out of scope; do not present this minimal profile as a complete wire specification.
