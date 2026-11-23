# Binding Publication Stage 3B1 — authorized finite bootstrap sessions

## Explicit bounded split and pins

The owner permits independently tested slices. This kit implements 3B1 only:
authorized snapshot-first finite delivery, validated bounded successor pages,
injected as-of barrier certification, a synchronous release/revocation fence,
and owned cancellation/cleanup. It does NOT implement complete Stage 3B.
3B2 is explicitly deferred: opaque integrity-protected cursors, verification across
restart/expiry, verified resume, certified resnapshot, generation/reset migration,
and certificate/key lifecycle. No API pretends private plan IDs are resume cursors.
Do not report the deferred requirements as passing or silently implement them.

Baseline rDDD draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
locally ratified INTEROP-PUBOBS/0.1 v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e. All 19 checksums pass; manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Go initially clean at 0dc2d511ba03d6c0b62198eea3fba17a3a391b38; no later work found.
Four OBSERVATION plans, Stage 3A contract/evidence and approved profile reviewed.
No normative ambiguity found in this slice. No wire format/transport selected.

## Ownership and design before API freeze

New sdk/go/observation only; earlier code, SQLite formats and dependencies unchanged.
Manager owns session quota/dependencies; each Session owns its context, serialized
Step, durable Stage 3A plan pin and retryable cleanup. Caller owns Manager lifetime,
Journal and authority dependencies, closes sessions before closing storage.
One Gate is bound to ONE authenticated observer/view/descriptor/authorization epoch,
not an unbounded revocation registry. Multiple sessions may share that grant within
MaxSessions. Revoke permanently retires that Gate. New authority epoch requires
an independently authorized new Gate; no silent grant revival.

Binding identities are named by caller but validated through injected Authority:
Check proves observer/view/model/schema/epoch and current descriptor compatibility.
Exact Profile string by default; no guessed version compatibility. Each release
checks again. Journal is a trusted source of registered rendered immutable records;
this stage does not authenticate a remote fake Journal or perform public rendering.
Stage 3A Host remains responsible for coherent source cuts, schema/model and public
rendering. Do not combine unrelated source reads or expose private journal evidence.

Authority.Certify returns an explicit typed certificate bound to Binding, Through
revision and permitted public evidence. It must vouch for the finite released prefix
under the configured authority, not copy caller flags. Library checks returned fields
against released state and refuses mismatch/empty seal. Seal representation and
verification/trust distribution remain authority-owned, not a new shared encoding.
No production crypto or cursor authenticity is established by a test seal string.

Prefer synchronous explicit Step over callback fanout/background sessions: one page
per call, no unowned goroutine, durable replay is truth. A stateful infinite watcher
would add concurrency/backoff/transport ownership before it is needed. No mandatory
per-assertion assertionStatus on projections; value availability/evaluation/completeness
stay inside unchanged Public. Session delivery coverage is separate as-of knowledge.

## Bootstrap, page validation and barriers

Open checks authority BEFORE Journal.Prepare or any recipient callback, reserves a
bounded session slot, obtains the durable pin and creates a cancellable Session.
No raw journal/authority error details become public frame metadata. A failed Open
releases its quota; if storage committed a pin but lost reply, private journal owner
must reconcile Plans/release unknown pins under Stage 3A recovery. Physical pin limits
still hold. There is no observer cursor or session recovery in that case.

Step delivers the pinned full snapshot exactly once per session, validates its View
and nonzero Revision, and remembers its last successfully released revision/evidence.
It then reads at most Page successors. Validate ENTIRE returned page before releasing
any revision: PlanID, View, After, Through, SelectedHead, length, strict contiguous
Revision/Previous, and bounded count. No later update can repair a gap/conflict.
Read error/partial scan produces no barrier. Validate page before trusting its cut.

Release revisions in order. Only after those calls succeed may Certify be invoked
for the exact last released revision and approved public evidence. Validate returned
certificate then release the barrier THROUGH THE SAME gate. Revocation between
certification and barrier release still blocks barrier and seal metadata. A barrier
can describe a partial finite batch, never unseen SelectedHead or future revisions.
A later Step may certify an empty successor range at the already reached boundary;
this does not assert new live freshness or client application. Snapshot40 followed by
concurrent commit41 before first tail read must yield snapshot40, revision41, barrier41.

Recipient gets independent typed frames; mutation must not change pinned state.
Callbacks are actual in-process disclosure, not arbitrary queued future release.
No transport/admission/submission/projection-application fact follows from a barrier.

## Release/revocation boundary and failure meaning

Gate.Release holds one local synchronization fence through BOTH current authority
Check and the actual synchronous Recipient call. Gate.Revoke uses that SAME fence;
pre-check then unlocked callback is forbidden. Revoke-first invokes no callback and
releases no payload, certificate or protected metadata. Release-first is an attributable
prior authorized disclosure; all later calls are denied. No recall claim is made.
Frame must match bound view/certificate binding and a recognized kind.

Callback must obey context deadlines, release synchronously and never schedule
uncontrolled post-return disclosure, nor reenter Gate/Session/Manager. Injected
Authority methods also honor contexts/no reentry. No library can forcibly terminate
an arbitrary Go callback safely; limits are a trusted cooperative contract. We do
not spawn and abandon a goroutine to simulate a timeout. A noncooperative callback
invalidates the stated bound and is not a production-safe integration.

CallbackTimeout applies to every release/certification. An error or expired context
AFTER invoking recipient means ErrUnknown (may have disclosed), never proven rollback.
Do not return raw recipient error payloads. Before callback, denied/retired/canceled
checks establish no callback. Certification errors do not produce a barrier.
Current descriptor/epoch changes are checked again for every frame.

## Lifecycle and bounds

MaxSessions >0, Page >0, callback and cleanup timeouts >0. Durable pin quota and
retention are enforced by unchanged Stage 3A. Each Step buffers at most one bounded
page plus snapshot/barrier. It returns rather than chase an ever-growing stream.
No background polling, cache or session goroutine. Manager quota is finite; Gate
retains just one grant/revocation bit. Callers bound number/lifetime of Managers/Gates.

Step is serialized; Close first cancels context, waits for in-flight synchronous work,
then releases pin under an independent bounded cleanup context. No callback after
Close completes. Calls on retired sessions return ErrClosed. No work is resumed after
failed Step: it cancels/retires and attempts cleanup. Successful Close idempotent.
Failed cleanup remains retryable/owned and keeps quota reserved; a later Close retries.
Join stable ErrCleanup with the primary failure without exposing private errors.
No silent pin leak or release of quota before cleanup succeeds. Cleanup after unknown
storage outcome may require independent owner recovery; no exactly-once cleanup claim.

## Executable coverage and explicit deferrals

TestBootstrap40RaceAndBarrier: O03 local in-process snapshot handoff, O04 snapshot-first,
O13 as-of-only empty batch, B01 retained capacity. Uses actual Stage 3A SQLite journal.
TestReleaseRevocationOrders: A03 both orders at the actual callback fence, plus revoked
session releases no metadata. Deterministic channels, no sleeps.
TestInvalidScanAndDescriptor: O05/O08/O10/O11 local chain/error validation, V02 descriptor
change, A01 wrong observer/view is denied before callback; no partial page/barrier.
TestSessionBoundsCancellationAndCleanup: O15 local closure/join, uncertainty, primary+
cleanup errors, quota, cooperative deadline. Previous Stage 3A tests cover O01/O02,
O14 typed axes, hidden-change/no-op and storage reopen/crash.

NOT claimed: full O06–O11 resume/reset/cursor contracts (3B2), A04/A05 real client
cache/session fencing, A06 public renderer/filter security, all A01/A02 metadata
noninterference, V network negotiation, actual credential trust, production revocation,
or end-to-end CLIENT conformance. Injected fake authorities prove schedules, not AUTH.
No new process-crash protocol is added by 3B1: persistent history/pins recover under
Stage 3A; process restart retires volatile sessions and starts an independently
authorized fresh bootstrap. Cursor/key trust across restart is not implemented.

Future 3B2 should first add a reviewed retained-position plan/resume seam rather than
reuse Prepare's current-head snapshot as a cursor verifier. Approve opaque token
identity/expiry/key rotation, full successor availability, explicit cannot-resume and
certified replacement with adversarial restart/view/schema/generation tests. Only then
recommend a separately approved real Go/TypeScript observation binding integration.
