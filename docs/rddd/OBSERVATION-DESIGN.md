# Publication Stage 3 — observation gateway design proposal

Review only; names are descriptive responsibilities, NOT frozen Go interfaces or
message formats. Authority/pins and existing coverage: OBSERVATION-REQUIREMENTS.md.

## Recommended separation

Immutable admitted history + explicit current-assertion inputs
 -> projection evaluator over a trusted consistent finite evidence cut
 -> public-view renderer under a registered observation policy
 -> independent transactional projection/view journal
 -> authorized snapshot/tail plan with retention protection
 -> bounded observation session
 -> race-safe disclosure authority -> approved recipient boundary

Do not turn delivery.Store into an observation gateway or modify its history.
The source continues to admit publications independently; an asynchronous projector
may lag. A barrier is through committed PROJECTION records at a stated source cut,
not a claim that all source events accepted elsewhere have already been projected.

## Ownership and identity

| Resource | Owner | Persistence / distinction |
|---|---|---|
| Domain projection definition and evaluation | Context-owned model/evaluation authority | Identifies ordinary historical events, any explicit current-key inputs, rule/schema, cut/coverage; not a ledger inference |
| Consistent input cut | Evidence authority | Immutable bundle/verified boundary or trusted serialized revalidation; concurrent independent Records reads do not create a multi-store transaction |
| Projection result acceptance | Single projection-journal owner | Durable evaluation checkpoint and idempotent result receipt; reject stale/relabeled evaluations |
| Public authorized-view identity | Policy/descriptor authority | Principal or policy-equivalence view, normalized projection parameters, schema/model, durable generation; no accidental sharing across observers |
| Public revisions/head | Journal owner | Contiguous per-view semantic-change ordinals + predecessor and immutable full public records; no arithmetic mapping to assertion ordinals |
| Resume capability/certificate | Observation authority | Opaque integrity-protected binding to observer/view/generation/schema and retained cut; authority key identity survives or explicitly invalidates restart |
| Live sessions and retention pins | Gateway/session owner | Bounded owned resources; server restart retires sessions. Durable log/cursors enable later resume, not revival of an old session |
| Disclosure/revocation | Authority owning the actual release point | Serialized permission epoch/order; cannot delegate correctness to a pre-send check |
| Client applied cursor/cache | Client session owner | Gateway does not infer application from a transport return; client validates chain/barrier and fences retired sessions |

Minimal view identity is per authorized observer rather than a clever shared ACL
cache. Sharing across policy-equivalent observers is an optimization requiring proof
of equal visible records and separate observer-bound cursors. Same parameters under
different model/policy generations are not interchangeable. Compatible policy change
may rebind only with explicit authority; default is new view generation + rebootstrap.

## Projection journal and revision policy

Recommend a NEW private transactional journal, separate from existing publication
and recipient stores, potentially using the already selected SQLite driver later.
No dependency or schema selected/changed now. Persist accepted evaluation receipt,
source/evidence checkpoint, active descriptor/view generation, visible head, immutable
full public revision when changed, and replay availability atomically. Accepted no-op
advances PRIVATE inspection checkpoint without inventing public revision or changing
old record/provenance. Rule changes require approved compatibility or new generation.

Equality is over the declared public observation: value/availability, evaluation,
backend completeness, permitted evidence reference and other schema-declared public
status. Same value/frontier with changed public evaluation produces a revision.
Unchanged public state does not. A private hidden source change must not manufacture
public sequence increments, raw offsets, source IDs or diagnostic counts. Keep the
private inspected frontier distinct from its permitted public attestation. A new
as-of barrier may certify inspection progress without modifying prior public records;
its opaque authorized-scope evidence must not expose hidden offsets/activity counts.
The renderer/policy authority defines what evidence metadata is materially public;
random token generation alone is not a non-disclosure proof.

Projection history is full replacement records, not deltas. Per-key retract may
change a projection, leave it unchanged, or affect several projections under domain
rules. It never universally clears the projection or undoes historical transitions.
Available null and unavailable projection stay distinct. Initially no evaluated
projection must not be fabricated as an available zero value; use an authorized
unavailable baseline or remain unverified. A certified revision-0 baseline is allowed
only under the profile's no-history case, not as a reset of a used generation.

## Atomic bootstrap without a fetch/subscribe hole

Prefer durable pull from a retained journal to an ephemeral push subscription:
1. Authenticate/select compatible injected descriptor and authorize existence/scope.
2. In one consistent journal operation, select head S/h, snapshot public record and
   the tail start h+1, and establish a retention pin covering the handoff. Bind the
   plan to policy/view generation. No need to install a network subscription atomically
   with this transaction: the durable tail itself is the subscription source.
3. Frame snapshot first. Read successor pages strictly after h under that same view.
   For each batch select a finite committed end b, protect its retained records, then
   deliver every successor through b. Only then offer barrier through b with the
   corresponding authorized evidence cut. Updates after b belong to a later batch.
4. If revision h+1 commits between selecting h and tail reading, it remains in the
   durable tail. Never use an unpinned in-memory notification queue as sole evidence.
5. Release each snapshot/page/barrier only through CURRENT disclosure fencing.
   Authorization at plan creation cannot authorize future replay forever. Policy
   change cancels/rebinds the plan, not a partial-success certificate for a new view.

This proposal selects snapshot-first framing (an allowed O04 alternative), avoiding
out-of-order buffering machinery initially. A future binding that reorders frames
must explicitly implement the approved bounded buffering variant. No barrier on
partial scan/decode, missing successor, overflow, unauthorized frame or failed release.
A client may see duplicates after failure/restart; stable record identity and equality
make those safe. Never claim exactly-once physical observation delivery.

Retention pins have bounded count/bytes/lifetime controlled by the owner. If capacity
cannot preserve the handoff, refuse/terminate with explicit unavailable/resync-required
before dropping required data. No silent coalescing or cursor advance. After process
restart old sessions are invalid; resume must reauthorize and check retained continuity.
Retention pruning is initially deferred; a bounded append-only journal can refuse
new acceptance at capacity while retaining all accepted work. Later pruning must
atomically update the replay floor and honor pins. Ordinals never wrap.

## Resume and status axes

Validate opaque cursor integrity, principal/view/scope/model/generation and retention
before use; reauthorize before reporting existence or head. If complete successor
chain remains, select a new protected batch and barrier. Otherwise explicit cannot-
resume followed by authorized certified snapshot/new barrier. A normal newer update
cannot clear a gap. Replacement repairs current observation only, not missing admitted
history, consumer publication storage or submission knowledge. Server does not treat
a cursor as evidence of permission or a claim that the client applied a revision.

Backend completeness remains trusted as-of source evidence; evaluation remains
valid/invalid/failed; value availability remains independent. Delivery continuity
is session knowledge (unverified, verified-through, unknown, gap/conflict). Freshness
is ONLY as-of a named verified snapshot/barrier; no live lease/clock inferred.
Quiet connection does not advance it. Disconnect alone does not fabricate a missing
backend event. Unknown/failed hydration cannot silently serve stale data as current.

## Actual disclosure boundary and revocation

A narrow trusted authority operation must check principal/view and serialize effective
revocation with the IRREVERSIBLE release of each permitted frame, including barriers,
cursors, errors and retained reads. A token copied under a mutex and later sent after
unlock is insufficient. For the first local stage, an injected authority and synchronous
in-process recipient callback can own this exact boundary. The callback may not queue
future unguarded releases after returning; all async work must remain owned/fenced.
An internal enqueue is NOT proof of observer disclosure safety unless that queue itself
is the approved authorized recipient/release boundary. No transport selected here.

Define one order: revoke-first blocks release; release-first is an attributable prior
disclosure even if physical receipt is delayed. Cancellation or an I/O error after
possible bytes escaped cannot prove non-disclosure. Persist a protected release-attempt
record/permit if audit recovery is required, then classify uncertain completion as
possibly released; do not claim a database transaction atomically covers arbitrary
external output. Revoke can wait only within declared bounded callback/deadline rules;
a binding unable to stop post-return release cannot claim this contract.

Revocation retires session epoch, fences queued work and removes owned protected caches;
reauthorization requires a fresh permitted bootstrap. No remote-erasure guarantee for
application-owned references or previously released data. Unknown policy evidence blocks
release without fabricated permission or forbidden denial detail. Third-party delivery
infrastructure itself requires disclosure authorization. Production credentials, policy
updates and recipient boundaries must actually share this ordering authority; injected
flags or SQLite alone do not establish production AUTH/revocation safety.

## Idiomatic Go shape and alternatives

Prefer one journal owner plus explicit session objects with context cancellation and
bounded synchronous Next-page/Release operations. Descriptive seams: accept verified
projection, prepare snapshot/resume plan, read retained successors, certify finite cut,
release under authority, close session. These are NOT exported signatures to freeze.
Model typed values internally; no JSON/wire choice, transport, automatic discovery or
remote schema execution. Immutable/copy-owned records; retain useful internal error
identity while public policy controls diagnostic payload. Never serialize raw core.Err.

Avoid an all-in-one Client/Service global singleton or channel fanout with hidden state.
An ephemeral channel-only handoff loses concurrent commits; a transaction held for an
entire slow observation blocks evaluation and revocation; materializing every history
per watcher is unbounded. Durable replay + bounded pins and pages makes ownership local.
A later signal/channel may wake a reader but cannot replace journal/certificate truth.
Each acquired session/iterator/transaction has an owner; repeated Close is idempotent,
cleanup completion observed, primary failure preserved. No goroutine may outlive its
session unnoticed. Slow observers cause bounded backpressure/resync, not silent gaps.
