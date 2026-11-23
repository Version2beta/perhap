# Pure REV design — corrected draft.6

## Pin and decision

Authority: rDDD candidate 0.1.0-draft.6, locally committed at
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87. Checkout clean; all 17 checksum entries
verified. SHA256SUMS hash:
ab81ac9b68bf71d1e19c4f338b8e4b50df097a40218b4e41384288a6a5c9f816.
Read SPEC §5, CHANGES-draft.6, CONFORMANCE and all updated REV fixtures.
Candidate status is not ratification. This supersedes this document's draft.5
proposal; its original-slot architecture and R1 problem are obsolete, not pending.
The corrected R1 history b:set(2), c:add(5), z:set(7) yields 7. Causation
b -> c -> z is historical and does not manufacture a retrospective cycle.
No normative contradiction found in the selected pure subset.

## Two separate interpretations

Historical admitted events -> unchanged CORE canonical replay -> domain state.
All publications remain immutable events at their own positions. A retract is
not universal undo; domain-defined adjustments are new events.
Separately, admitted verified publications -> Reconstruct -> current assertions.
Current assertions are queryable derived evidence, never replacement history.
Do not add anchor vertices, fake historical noops or modify CORE's public API.
The test-only counter reducer declares retract to leave its business count
unchanged; it still attempts the retract event. No generic business semantics
are imposed by the REV package.

## Idiomatic API and ownership

New sdk/go/rev package: batch Reconstruct and per-key Reconcile with plain
typed records, comparable scoped Key, maps for lookup, slices for evidence,
explicit value availability, statuses and diagnostics. Values are immutable
domain-canonical strings: callers establish semantic encoding/equality; no JCS
or INTEROP-JSON claim. Key identifies context, producer, subject and meaning,
not history hash, publication UUID or payload. Error conditions are structured
and do not publish domain events. No context, goroutine, storage or I/O lifecycle
is necessary for bounded pure functions.

Alternative: stateful subscriber/store object could cache incremental changes,
but introduces concurrency, continuity and restart ownership before equivalence
is proven. Prefer pure batch reconstruction and caller-owned evidence.
History is copied and retained in supplied order; the caller supplies canonical
historical order through CORE where needed. Current view revision selection
is independent of publication arrival order and unrelated event IDs.
Input publications are already admitted/verified by the host; the library
checks revision sequence, links and conflicts, not signatures/authority.

Per key, contiguous ordinals start at 1 for this local profile; optional
supersedes must identify the preceding verified publication. upsert, retract,
and status operations are a local internal representation, not a prescribed
rDDD wire envelope. Gaps/conflicts retain the last contiguous value as stale,
mark unresolved/incomplete, never accept a later unsupported value. A trusted
closed finite certificate may establish coverage; absent certificate is
unavailable unless a specific gap is known. No unverified value is authoritative.

Reconcile consumes the previous assertion plus the key's evaluation/coverage.
A complete valid disappearance proposes retract; failure, invalidity or unknown
coverage proposes unresolved status while retaining stale content. Unchanged
current content emits nothing; same-content recovery requires current status.
Intentions carry ExpectedRevision, not allocated publication IDs/next ordinals.
No publisher can treat them as authorized execution instructions.

## Compatibility and scope

CORE source/API/tests and existing runtime remain unchanged. Domain-model
dependencies on current keys, genuine cycles and bounded cross-domain cascading
are not a new generic graph engine here: models must explicitly consume verified
view status and report unresolved/incomplete required values under CORE.
The pure subset does not supply durable ordering, restart-safe sequence allocation,
publication/reliable delivery, AUTH disclosure/admission or EFFECT commitment.
Those remain later separately authorized runtime work.
