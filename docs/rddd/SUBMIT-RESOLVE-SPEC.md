# Binding gateway Submit/Resolve kit

Owner-approved decisions supersede alternatives in SUBMIT-RESOLVE-DESIGN.md:
separate SQLite file; authoritative ONLY for gateway traffic; canonical UUIDv6;
registered versioned codec; lifetime identity retention, capacity refusal, no expiry.
Pins: draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; profile v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e; all19 checksums and manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f verified.
No numbering/new engine/dependency; existing journals/APIs untouched.

New submission owner uses established SQLite single connection, DELETE, synchronous
EXTRA, fullfsync ON, locking EXCLUSIVE, busy timeout0, explicit exclusive transactions.
Private bounded append log + generation metadata; no expiry or reuse. Fresh creation
only for absent file, existing empty/corrupt file refuses recovery. Validate sequence,
immutable normalized event/schema/attribution, fact transitions/versions/generation and
attempt identity on replay; independent Host.Recover rejects rolled-back inventory.

Open returns trusted Owner; Owner.Client(context) exposes only Submit/Resolve.
Owner.Evaluate is host-only and invokes actual core.Evaluate, never client result flags.
One representative int64-state CORE registration suffices; not a generic domain engine.
Client passes Input(perhap.Event, schema, attribution binding); Host.Authorize derives
requester/asserting identity and permissions. Require canonical UUIDv6 event ID and
lossless valid Unicode in stored identity fields. Normalize only Data through immutable
registered Codec; retain canonical bytes and use them BOTH for equality and evaluation.
Codec cannot change event identity/context/type/entity/cause/correlation. Equality
includes all semantic fields, schema and trusted asserting identity; transient binding
credentials/requester credentials are not semantic identity. Data and pointers copy-owned.

Submit first persists durable receipt v1/admission pending/evaluation not-established.
Only Host.Admission may commit admitted/rejected/unresolved. Callback zero leaves
pending; one callback holds authority through commit. No caller admission/evaluation
state. Identical retries return LATEST full facts and original receipt; pending decisions
may be retried explicitly under current authority. Changed identity conflicts without
rewriting original. Terminal admitted/rejected cannot change. Evaluation never implies
admission (must already be admitted), publication, observation or effects.

Evaluate(id,attempt) builds finite admitted history in registered context/entity/schema
from THIS authoritative file and uses registered Codec.CoreEvent + Model with CORE.
It must actually attempt target event before reporting evaluated/failed. Missing required
cause leaves evaluation not-established/blocked. Determinate domain-invalid is completed
evaluation, not rejection. Failed later evaluation preserves admitted fact. Attempt IDs
idempotent; same attempt cannot allocate another fact. Evaluation serialized with local
admission/current registry fence (registered codecs are immutable for schema version).
No JetStream dependency: no bridge, automatic dispatch/retry, reducer side effects or
runtime projection is implemented. Committed events remain pending/readable for evaluation.

Version starts1 and advances once per new full authoritative fact; no wrap, no update
on duplicate. Complete status matches TypeScript receipt/admission/evaluation axes.
Generation and versions independent of publication/reconciliation/checkpoints/cursors.
Resolve absent returns no-retained-knowledge at a cut, never proof of noncommit; lookup
can precede an in-flight submit. No expired outcome implemented: S06 remains unexecuted.

Authorize before existence/status details. Host.Release fences actual synchronous
Recipient callback against revocation, for facts AND lookup dispositions; no unlocked
precheck or deferred release. Deadline/cancellation before callback yields no callback;
error/expiry after callback invokes uncertain-release category. No raw private errors.
Client replies via callback; errors classify blocked/denied/conflict/invalid/capacity,
precommit unavailable, corruption, commitment unknown and uncertain release separately.
A callback failure does not erase committed facts. Cancellation after receipt/admission
cannot undo it. Commit issued but unconfirmed poisons handle until reopen. Before-commit
controlled cut is known rollback. Public outcome never guesses globally no admission.

MaxEvents/MaxEntries/MaxBytes/MaxEventBytes/Timeout all positive. Accepted identity and
facts retained forever within store; duplicates/lookups still work at capacity. Reserve
future transition capacity for receipt/admission pending work before acceptance; if no
room, refuse before recording a receipt. No orphan background tasks. Close cancels and
joins cooperative in-flight work then closes owned connection, idempotently. No cross-file
atomicity with publication/observation or JetStream; legacy ingress untracked/outside scope.

Focused test mapping: S01/S02 receipt loss, attribution, identical/concurrent/conflicting
identity; S03/S04 coherent immutable versioned facts + independent reply merge oracle;
S05 lookup cut before submission; S07 missing authority/denied and forged attribution;
S08 canceled admitted event still resolves, projection progress is not acknowledgment.
P14a/b/c distinguish unknown commit / precommit unavailable / corrupt actual reopen.
Real subprocess cuts before/after receipt and admission commits. Resource ownership,
competing owner, capacity completion/lookup, codec numeric normalization1 vs1.0, semantic
conflicts, actual CORE success/failure/blocked target and immutable attempts.
Count complete shared rows only; client-only/partial rows unexecuted, S06 expired deferred.

Independent reference outside checkout; new tasks fail on stubs, reference passes exact
unchanged v1.1.1 judge including existing24 groups. Clean verified kit commit before
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh. Full Go/race/vet/judge and source review.
No production AUTH/crypto, TypeScript adapter, upstream edits, transport or effect claims.
