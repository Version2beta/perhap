# Publication and observation interoperability — INTEROP-PUBOBS/0.1

**Standing:** Locally ratified optional semantic profile under the owner's conditional ratification request. Version `0.1.0`, profile identifier `INTEROP-PUBOBS/0.1`. Ratification is scoped to this profile, not the draft.6 baseline or implementation conformance; it does not assert independent implementer approval. Binding baseline: rDDD draft.6, commit `1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87`; baseline manifest digest `ab81ac9b68bf71d1e19c4f338b8e4b50df097a40218b4e41384288a6a5c9f816`.

MUST/MUST NOT/SHOULD/MAY describe requirements for implementations claiming the applicable role. This document establishes the local shared contract; it does not assert that either implementation team has accepted or implemented it. No wire schema, transport, encoding, database, language API, or dependency is selected.

## 1. Sources, scope and authority

Inputs read locally, all untracked proposals at inspection:

- Go worktree `~/Development/perhap-rddd-foundation`, branch `rddd-foundation`, HEAD `758322024fab69cea3fbae7dfd471fe78adbae7a`: `docs/rddd/PUBLICATION-{REQUIREMENTS,DESIGN,PROTOCOL,STAGE-PLAN}.md`. `PUBLICATION-DECISIONS.md` and `PUBLICATION-SHARED-PROFILE-DRAFT.md` appeared during inspection and were also read before finalizing this proposal. Their Q1/Q5/Q6 recommendations are input, not approved decisions. The main `perhap` checkout is a different worktree, not the proposal source.
- Client worktree `~/Development/perhap-client`, HEAD `c09645c1cd4874e07f370541ab43b5260b560efe`: `docs/rddd/INTEROP-{REQUIREMENTS,DESIGN,FAILURES,STAGE-PLAN}.md`.

Sources remain unmodified and uncommitted. Their descriptions of existing code are research inputs, not runtime evidence or authority for this profile.

Roles: submission service (S), assertion publication ledger (P), projection observation service (O), observing client (C). Roles may be independently implemented. A P implementation need not implement O; a C implementation need not store historical publications or execute domain reducers. Claim exact profile version, baseline, roles, supported operations and untested requirements. No end-to-end claim follows from a role-local test.

### Baseline requirements versus profile decisions

| Kind | Authority / decision |
|---|---|
| Existing rDDD | §3 immutable admission/identity; §4 deterministic evaluation and evidence status; §5 distinct historical publications/current assertion view; §6 authority/disclosure; §7 consequential commitment; §9 distinct acknowledgments, recovery and revisions |
| New optional profile | Per-key unified status/content sequence, generation identity, checkpoint acceptance, initial-key policy, versioned submission facts, observation barriers/resume, as-of-only freshness, full projection replacements |
| Deployment/binding choices | Authentication mechanism, certificate representation, identity encoding, schema language, retention durations, capacity values, storage engine, transport, routing, serialization and internal APIs |

**No change to normative SPEC.md is proposed or required.** This profile specializes permitted behavior. If review finds a conflict, stop the affected role and propose an exact upstream change with a counterexample; do not weaken the baseline by interpretation. Prior original-slot rules and inherited-correction-dependency proposals are not applicable.

## 2. Separate resources and identities [I]

Identity conveys no authority. Each resource MUST bind its authenticated application/context and semantic scope. Equality MUST cover immutable semantic fields, attribution and declared references; transient credentials, routing and delivery attempts MUST NOT change logical equality. Bindings MUST define a non-lossy equality implementation before exchanging encoded values.

| Resource | Identity and order | Meaning |
|---|---|---|
| Submission/event | Globally unambiguous event ID; attempts have separate IDs | One immutable assertion; retries preserve semantic content and attribution |
| Submission facts | Event ID plus durable status generation/version | Receipt, admission and evaluation knowledge, not a projection cursor |
| Reconciliation command | Request ID scoped to accountable producer and target | Idempotent ledger operation; distinct from event ID and evaluation token |
| Assertion stream | Context, producer, subject, meaning, and durable generation | One logical assertion key with ordered publications |
| Assertion publication | Immutable event ID plus stream ordinal | One content or status transition; predecessor identifies prior publication |
| Projection stream | Application/context, named projection, normalized parameters, schema/model, authorized view and durable generation | A scoped observable result, possibly depending on many assertion keys and ordinary events |
| Projection revision | Projection stream plus durable ordinal | Immutable full public observation record; not an assertion ordinal |
| Observation session | Local/session fence plus authorized stream/cursor | Delivery lifecycle; reconnect does not establish new business facts |
| Evidence/checkpoint | Trusted evaluation scope, rule/model generation and evidence cut | What was inspected and accepted; not an event timestamp |

Ordinals start at 1 and increase by one for committed stream changes; a no-history projection may use a certified revision-0 baseline. Ordinals MUST NOT wrap or reset within a generation. Unrelated streams have no total order. Restoring stale storage or transferring a producer MUST preserve committed history and fencing or use an explicitly authorized new generation with certified rebootstrap; a new generation is never silently compared with an old one. Generation reset MUST NOT permit reuse of event identities or evade admitted-event deduplication.

A projection revision MAY incorporate many publications; a publication MAY cause zero, one or many scoped projection revisions. There is no mapping by ordinal arithmetic. An assertion retraction MUST NOT mean projection absence or historical business undo unless the declared domain projection explicitly produces that result.

## 3. Submission and acknowledgment contract [S]

A service MUST distinguish these facts; no acknowledgment implies a later stage:

| Fact | What it establishes | What it does not establish |
|---|---|---|
| Transport receipt | Binding-defined receipt of bytes | Durable submission or admission |
| Durable submission | Recoverable immutable submitted event, its event identity and durable submission receipt | Authority approval |
| Admitted | Durable authoritative event and trusted admission attribution | Successful evaluation, publication or effects |
| Rejected | Definitive non-admission of this submission under the admission decision | A network timeout or missing lookup |
| Evaluated / evaluation failed | Identified attempt/model/evidence and result | Projection inclusion or effect commitment |
| Publication accepted | Durable ledger record, or durable no-change receipt | Remote delivery/admission |
| Observation applied | Client accepted a certified scoped revision | Admission of any particular submitted event |

An event submission is recovered by its event identity and authenticated semantic content. A reconciliation command instead uses its own scoped request identity to accept an evaluation result into the publication ledger. Evaluation checkpoints and allocated publication event identities are separate resources; no one-to-one relationship to a submitted event is implied. If a binding correlates these resources, it MUST declare the relationship explicitly and MUST NOT substitute an attempt ID, reconciliation request ID or checkpoint for the submitted event ID.

Receipt, admission and evaluation MUST be separately represented. Admission progresses pending/unresolved -> admitted or rejected. Established admission MUST NOT be erased by later evaluation failure or changed permissions. Evaluation attempts and corrections MUST identify their model/evidence and advance status knowledge, preserving previous facts; they are not an irreversible single success flag. Identity conflict is a rejected conflicting attempt against an existing identity, not a rewrite of the existing event's admission.

Status replies MUST carry their resource generation/version and complete authorized facts. Older replies cannot regress newer facts; identical versions with different semantic content are a protocol conflict. Transport unknown is local knowledge, not a persisted rejection. Timeout, abort or cancellation of a wait MUST NOT imply non-admission, rollback or nonexecution.

Authorized lookup and identical retry MUST return retained durable facts/receipt. Reconciliation-command retries return the original receipt even after the head advances; that receipt is not a claim of current head. Changed immutable meaning under the same identity MUST conflict without overwriting prior data. Concurrent same-ID requests MUST resolve atomically to one identity decision.

The deployment MUST declare supported retry and receipt/deduplication retention horizons and recovery rules. Compacting payloads MUST NOT allow an expired admitted identity to be treated as a new fact. After receipt knowledge expires, return no-retained-knowledge/expired only where disclosure permits; neither means rejected. An old request MUST NOT be automatically accepted as fresh work after its dedupe horizon: retain a rejection/tombstone mechanism, retire its namespace, or require explicit recovery that prevents reexecution. A missing lookup at a cut cannot exclude a still-in-flight commit. Unknown callers MUST preserve identity and reconcile rather than invent a replacement ID.

## 4. Assertion publication ledger [P] — Q1/Q5/Q6 resolutions

### P1. One ordered content/status stream (Go Q1)

Each key/generation has one durable sequence for `current`, `unresolved`, and `retracted` publications, including status-only changes. Every publication MUST identify its ordinal, predecessor (none at ordinal 1), operation/status, explicit value availability, schema/rule identity, attributable producer and protected provenance references. Historical causes and predecessor/supersedes links are separate relations. `unresolved` may retain a clearly stale last-supported value; unavailable differs from available null. Recovery to current with the same value MUST publish a status transition. Equal status and content with newer inspection evidence alone MUST NOT allocate a semantic assertion revision; checkpoint progress is separate.

All admitted publications remain actual historical events. Consumers maintain the separate latest-contiguous-verified current assertion view, not replacement history. Missing or conflicting publication ordinals make the affected view incomplete; arbitrary latest-arrival selection is prohibited. Revision ordering alone is not business causation. Domain rules define historical transitions and any explicit current-value dependency evaluation; no publication replaces, removes or relocates an earlier event.

### P2. Reconciliation acceptance and checkpoints (Go Q5)

Before acceptance, a trusted host MUST validate producer/target authority, active rule/model, expected key head/presence and publication identity, expected prior accepted checkpoint, ownership fence, evidence scope and evaluation generation. Client-provided success or completeness flags are not proof. The generation/cut MUST be comparable through an authority-owned serialized acceptance order or an equivalent trusted revalidation protocol. No universal ordering on evidence vectors is assumed.

A commit MUST atomically establish the durable request receipt and accepted checkpoint, plus the head, immutable publication and recoverable pending-delivery obligation when a semantic change exists. Atomicity is an observable all-or-none guarantee, not a prescribed transaction technology. Every newly accepted evaluation MUST advance the durable acceptance checkpoint, including accepted failure/unresolved evidence and no-change results. Rejected attempts and identical accepted-request retries MUST NOT advance it. A no-change acceptance MUST persist its checkpoint and receipt without advancing the publication ordinal. Each accepted receipt MUST identify its request and accepted checkpoint, plus the allocated publication identity/ordinal when one exists; a no-change receipt explicitly reports that no publication was allocated. Older evaluation results MUST NOT overwrite a newer accepted checkpoint, including after a newer no-op or unresolved result. A no-change result MUST NOT update the provenance of an existing immutable publication in place. Evidence receipts may advance separately.

Admission/checkpoint races MUST have an authoritative order or cause stale acceptance to fail. All relevant admitted information within the declared ownership scope MUST be considered/revalidated at acceptance. If no sufficient comparison/revalidation exists, refuse acceptance as stale/unresolved. Late evidence requires a new authorized evaluation generation, not reuse of an obsolete token. Single-owner exclusion MUST survive competing processes and restart; lease technology is optional. Storage uncertainty returns unknown pending lookup, never guessed rollback.

### P3. Initial publication (Go Q6)

Before any publication, complete valid absence yields a durable no-change receipt/checkpoint, no ordinal and no fabricated retraction. A complete valid present value first publishes current at ordinal 1. Failed, invalid, incomplete or unavailable required computation for an explicitly registered key first publishes unresolved at ordinal 1 with unavailable value only when trusted domain rules and assertion/disclosure policy explicitly authorize observing that status and key existence—never fabricated stale content. Where that permission is absent or unresolved, withhold publication and retain internal evaluation evidence only where separately authorized. Unregistered keys MUST NOT be created or disclosed by failed probes; return an authorized unresolved disposition without inventing a key.

After a supported value, loss of trustworthy evaluation produces unresolved (stale value only if permitted); complete valid disappearance produces retracted. Repeated identical unresolved/absent results produce checkpoints, not duplicate publications. Recovery from unresolved to present or certified absence publishes current or retracted, respectively, even with unchanged value. Key registration/visibility is an explicit host authority decision, not inferred from arbitrary client input. Domain-declared sufficient unaffected evidence may support a current result despite unrelated failures; isolate and certify that scope explicitly.

### P4. Delivery

Pending records MUST survive crash and be retried under their original identity/content. Acceptance does not require a reachable observer. Delivery progress and transport receipts are separate mutable scheduling facts. No silent skipping/coalescing of historical publications is permitted in this profile's historical feed. Duplicate recipient admission MUST not create another historical fact. Durable recipient admission/cursor advancement MUST be atomic or equivalently recoverable before acknowledging that stage; application is separate. Consequential issuance remains subject to SPEC §7 and is outside a provisional-only ledger stage.

## 5. Scoped projection observation [O,C]

### O1. Observable record and revision policy

The initial profile uses full public projection replacements, not patches or digest-only updates. Each record MUST bind stream identity, revision/predecessor, explicit value availability, evaluation, backend completeness, protected evidence-cut reference and model/schema. Whole-projection evaluation status is not a single assertion key's current/retracted state. Assertion statuses may appear only as explicitly declared projection content. A status/evidence change material to the declared public observation MUST advance the projection revision even if its value is unchanged. If the declared public observation record is unchanged, no new projection revision is allocated. Internal assertion/checkpoint activity MUST NOT manufacture observable revisions that reveal forbidden activity. Internal progress may instead be certified at a delivery barrier without changing the public record.

### O2. Atomic bootstrap and handoff

Authenticate and authorize scope before returning existence, snapshot, cursor or status. Choose a consistent authorized snapshot cut and a replayable tail strictly after that cut, atomically or by equivalent certification. Snapshot carries its revision (not necessarily 1), authorized-view generation, backend evidence status and handoff certificate. The service MUST guarantee that every subsequent public revision is in the snapshot or the certified tail, never lost between fetch and subscribe. A barrier certifies delivery coverage through a particular tail boundary for that authorized view. A barrier is not a domain event or submission acknowledgment.

The client MUST validate snapshot identity and binding, then predecessor continuity through the barrier before reporting verified-through observation. Out-of-order frames may be boundedly buffered; an older snapshot cannot replace a newer applied revision. Without a valid barrier, hydration remains unverified. Resource exhaustion/partial decode/interrupted scan MUST NOT produce successful empty hydration. Preserve only permitted stale data and report unavailable/gap as appropriate.

### O3. Resume and resynchronization

Cursors MUST be opaque to consumers, integrity protected and bound to authenticated observer/view, stream generation, schema/model and retained boundary. Clients MUST not infer hidden offsets or compare cursors from different scopes/generations. Identical revision delivery is idempotent; differing content/status at one identity is conflict and suspends the affected stream pending repair. A missing predecessor demonstrates a delivery gap; a disconnect alone makes continuity unknown.

Resume requires renewed authorization and certification of the complete authorized successor chain from the supplied verified cursor through a new barrier. A normal later frame cannot clear a gap. Expired/unrecognized/incompatible cursor or changed authorization view requires explicit cannot-resume and certified resnapshot, subject to non-disclosure rules. Replacement may start at an arbitrary revision/new generation, but only the replacement certificate resets delivery continuity. It does not restore missing backend history, historical publication storage, or submission/effect knowledge. Current snapshots MUST NOT be represented as a complete historical event replay.

Session owners MUST fence delayed prior-session frames before applying or disclosing them; reconnect, identity change, revocation and close retire the old session. Close/abort stops local waits and delivery, not backend admissions. All async completion/cleanup has an owner; failures must be observed, bounded and reported without exposing protected details.

### O4. Three independent axes and minimal freshness contract

1. **Backend evidence completeness:** SPEC §4.3's complete-as-of-frontier / incomplete / unavailable, based on trusted backend evidence, independently of evaluation validity and value availability.
2. **Client delivery continuity:** unverified/hydrating, verified-through certified cursor, unknown on disconnect, or demonstrated gap/conflict. A healthy feed cannot repair missing backend evidence.
3. **Observation freshness:** verified **as of** the named snapshot/barrier and evidence cut; it never promises perpetual present truth or future finality.

The minimum profile grants no live-authority lease and requires no clock synchronization. Silence/connection health never extends certification. If an application requires age-bounded or continuously current authorization, it MUST use a separately approved lease/clock/expiry contract; this minimum profile cannot satisfy that requirement. As-of records may be displayed only while observation policy permits; they never independently authorize effects.

## 6. Authority, disclosure and caches [A]

The deployment MUST identify trusted admission, evaluation-evidence, ledger, observation-certificate and disclosure authorities and their jurisdiction. Trust is established by the approved authenticated boundary, not a boolean in data. Assertion authority, observation permission and consequential action authority remain distinct. Permission to observe a projection does not imply access to its sources or publication history.

Authorization MUST cover payload, existence, routing identifiers, revision/status metadata, diagnostics, lookup, snapshots, repair, retained delivery and cached observations. Filtered delivery MUST use an authorized-view continuity certificate; raw hidden offsets/omitted per-key publications MUST NOT be disclosed as unexplained gaps. A historical feed lacking permission for required revisions cannot claim complete history; it may offer an explicitly certified current baseline instead. Policy changes invalidate or explicitly rebind the old view/cursor; do not disclose hidden interval sizes.

A disclosure boundary MUST serialize effective revocation with new release of protected information, including queued delivery and retries. A pre-send check with no race-safe boundary is insufficient. Information irreversibly released before revocation may arrive later and cannot be recalled; ordering/certification must identify the release boundary rather than pretend physical receipt is recallable. Revocation effective first blocks release; release effective first remains an attributable prior disclosure. Third-party delivery infrastructure is itself a disclosure recipient requiring authority.

For the minimal client role, no persistent/shared cache is specified. While an authorized session is active, owned cached values are usable only under that session's permission; on disconnect they MUST be hidden unless an explicit offline retention permission was granted. On revocation/identity change clients MUST fence callbacks and clear protected owned payload/metadata/optimism references. Offline retention permission, if enabled later, MUST specify scope, expiry and clock assumptions and cannot promise immediate remote erasure. Server revocation always blocks later releases. Previously exported application references cannot be recalled. Reauthorization requires fresh bootstrap, not silent revival of old data.

## 7. Discovery and compatibility [V]

Before data use, peers MUST establish a trusted descriptor binding application/authority, exact baseline/profile versions, role capabilities, required features, schema/model identities, public manifest version, semantic equality contract, supported retry/resume horizons and resource limits. Only explicitly public schemas/annotations are exported; no private rules or executable code. Discovery may be configured/injected; no network discovery mechanism or automatic remote schema execution/fetch is required.

Version compatibility MUST be explicit: exact profile-version match by default. A separately reviewed compatibility declaration may name tested compatible versions/features; numerical similarity is not evidence. Unknown required feature, schema, model or incompatible version MUST reject before applying data. Optional ignored features MUST be explicitly declared semantically ignorable. A changed incompatible descriptor suspends interpretation and requires negotiation plus certified rebootstrap; never reinterpret retained records under a new schema. A version transition MUST NOT rewrite historical records. Bindings must separately prove encoding validation, equality and certificate authenticity; no encoding is implied by this profile.

## 8. Representative exchanges (semantic notation, not messages)

- **Submission:** submit E/X -> durable submission -> admitted E at status v2 -> evaluation failed v3. Admission remains established. Reply lost -> unknown locally -> authorized resolve E or identical E/X retry returns retained facts. Projection P/42 alone cannot resolve E.
- **Publication:** registered K, evaluation g8 current V -> request R8 commits K/1 and pending delivery. Crash before reply -> R8 retry returns same receipt. New g9 same V -> no publication, durable checkpoint. Old g8 under a new request -> stale. g10 failed -> K/2 unresolved/stale V; g11 recovered same V -> K/3 current.
- **Initial absence:** trusted domain-registered K, with explicit assertion and disclosure authority for its existence, unresolved and retracted states, absent at complete valid g1 -> no ordinal; failed g2 -> K/1 unresolved/unavailable; certified absent g3 -> K/2 retracted. No old business transition is removed.
- **Observation:** observe Q -> snapshot S/40 at evidence F + tail -> S/41(previous 40) -> barrier through 41. Client reports verified-as-of 41. A concurrent assertion publication need not cause any Q revision; another may change several projections.
- **Reconnect:** lose session a at S/41 -> unknown continuity; session b resumes -> S/42, S/43 + barrier -> verified-through 43. Old a frame ignored. Same F with changed public status still permits S/43.
- **Recovery:** expired cursor -> cannot-resume -> authorized replacement snapshot S/90/new barrier. No assertion about recovered historical events or missing submission receipts follows. Conflicting S/90 data quarantines the stream.
- **Revocation:** queued protected frame, revoke ordered before release -> no release and fenced local callback. Frame released before revocation may arrive late; do not claim recall. Reauthorization creates a fresh permitted view.

## 9. Compatibility and approval matrix

| Shared contract / source | Go ledger Stage 1 | TypeScript observation Stage 2A | Later integration |
|---|---|---|---|
| I, P1 unified sequence (Q1; U1–4) | Required | Assertion stream not implemented | Historical publication consumers |
| P2 checkpoint/no-op/owner (Q5; P1/P2) | Required with trusted host seam | Backend promise, not client storage | Real admission/evidence concurrency |
| P3 initial lifecycle (Q6) | Required; explicit registered-key fixture | Do not impose assertion state on whole projection | Public key registration policy |
| S submission status/retries (Q4; R1–4) | Reconciliation receipts only | Deferred | Client Stage 2B and admission service |
| O1 independent projection identity (U11; R5,R9,R10) | Deferred | Required | Backend projection revision owner |
| O2 bootstrap/barrier (P4; R6) | Deferred | Required with deterministic fake | Real snapshot/tail atomicity |
| O3 resume/reset (Q3; R7,R13) | Deferred | Required | Retention and repair gateway |
| O4 as-of freshness (Q2; R8) | Checkpoints only | Required; no live lease | Any stronger freshness contract |
| A disclosure/revocation (Q7; U7; R11) | Private local store; no outgoing delivery | Required lifecycle/permission seam | Real race-safe disclosure boundary |
| V discovery (R12,R14) | Pin injected descriptor | Pin injected descriptor | Encoding/auth/discovery binding |
| Historical replay prohibition (U4; R10) | Required | No historical rewrite/inferred undo | All domains retain declared semantics |

Smallest independently implementable stages **after separate implementation authorization**:

1. Go: one exclusively owned durable reconciliation ledger with receipts, no-op checkpoint, initial/status lifecycle, recoverable pending records, reopen/process-crash tests. No dispatcher, public observation, live authority or effects. Storage/dependency/durability target needs separate owner approval.
2. TypeScript: one owned observation session over an injected approved descriptor and deterministic fake; full snapshots/updates, status axes, barriers, duplicate conflicts, reconnect/reset, revocation/session fencing and bounded cleanup. No production transport, submission orchestration, persistent cache, assertion ledger or effects. Explicit migration review for Stage 1 types is required.

Neither stage depends on the other's runtime being implemented. Fakes prove response to promised contracts, not real backend evidence/atomicity. A later backend observation service and binding must pass the same vectors with real persistence and disclosure enforcement before claiming integration.

## 10. Review gates and remaining decisions

Q1/Q5/Q6, as-of-only freshness, no offline cache by default, and the disclosure-ordering requirements are locally ratified profile rules. Deployment/implementation gates remain: identify the trusted checkpoint/ownership jurisdiction and initial-key authorization authority; demonstrate the revocation release boundary and authorized-view certificates; review stage-specific API migrations. This status is not approval of an implementation design, dependency, kit or runtime rollout. Each deployment must supply retention/capacity values, schema/equality definitions and trust roots. These parameters cannot be guessed from a transport.

Potential upstream clarification, not a proposed SPEC edit: formalize role-specific conformance dependency mapping if profile experience demonstrates ambiguity. General current-value dependency evaluation remains domain-defined under draft.6; this profile does not invent an algorithm. Strong live freshness, arbitrary partial-publication filtering, producer migration after lost storage and cross-scope atomic projection cuts need additional contracts if requested. No production binding is ready to freeze.

Shared unexecuted conformance vectors and required failure/crash observations: [publication-observation-0.1-vectors.md](publication-observation-0.1-vectors.md). Their expected behaviors are approved for this profile; the scenarios are not an executed conformance suite. Stop before kits, frozen APIs or runtime implementation unless separately authorized.

## 11. Profile changelog

- `0.1.0`: local ratification against corrected draft.6; distinguish event submission from reconciliation requests; establish initial-key disclosure/state authority; split ambiguous commit, unavailable storage and corruption oracles and condition other permitted alternatives. No normative SPEC.md semantic changes. Shared vectors remain unexecuted.
