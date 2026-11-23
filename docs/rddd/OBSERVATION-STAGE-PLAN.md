# Publication Stage 3 — review gates and smallest implementation slices

Planning only. No APIs, wire formats, dependencies, kit, paid loop or commits.
See OBSERVATION-REQUIREMENTS/DESIGN/TEST-PLAN.md for normative source pin, proposed
architecture, all-vector mapping and evidence limits. Existing work remains intact.

## Recommendation: first build the durable observation journal and cut planner

Stage 3A is independently usable by a local trusted host: accept a verified rendered
projection evaluation, durably recover its public view/head and private evaluation
checkpoint, and prepare a consistent snapshot/successor plan. No external observers,
network transport, production authentication or live revocation service yet. This
is a foundation for an authorized gateway, NOT a complete gateway/conformance claim.

Use one declared projection consistency owner and representative registered public
schema; support multiple assertion inputs plus ordinary events in tests to prevent
an accidental assertion-ordinal API. An injected evaluator/evidence authority owns
consistent input cuts, rule versions and public rendering. Do not copy current
assertion values into earlier history or infer backend completeness from delivery.
The host can supply verified results initially; domain projection evaluation belongs
to that host/model, not the observation store. Existing CORE may perform it without
changing CORE/REV semantics. No universal cross-domain projection engine is needed.

Smallest coherent 3A task boundaries after approval:
1. Define typed private projection/view identity, public-record equality and trusted
   evidence/descriptor contract; no shared encoding. Pure sequence/status oracle.
2. New single-owner durable journal: atomic idempotent evaluation acceptance,
   checkpoint/no-op handling, immutable full public revision/head and restart checks.
3. Consistent snapshot+tail plan with bounded retention protection, finite batch cuts
   and certificate-intention data. Snapshot-first framing; no external release.
4. Real subprocess crash/reopen, hidden-change no-op, stale-result and concurrent
   commit-at-handoff tests; full previous suites/race/vet/judge acceptance.

Do not combine mutable policy state and arbitrary source journals by assuming a
transaction across their independent files. Use a trusted serialized authority cut
or refuse unverifiable input. Storage layout is a Go implementation decision; a new
private journal may use existing SQLite experience after review, without migrating
Stage 1/2 databases. No storage choice or dependency is added in this assignment.

## Stage 3B — local authorized observation sessions over trusted seams

Complete a bounded backend-neutral in-process gateway: authorize bootstrap/resume,
release full snapshot and successor batches through a trusted disclosure callback,
issue finite barriers, validate bound opaque cursors, handle explicit cannot-resume/
certified replacement, and own cancellation/cleanup. Pins, sessions, pages and work
budgets have concrete approved limits. No persistent client cache, patches, background
unbounded fanout or transport choice. No production AUTH claim.

Source consistency, observation policy, descriptor, cursor/certificate authority and
actual release fence are injected trusted responsibilities. Race tests use real local
persistence plus deterministic adversarial schedules, not boolean permission alone.
A03 requires the injected authority to serialize release and revocation at the actual
recipient callback; merely testing that Authorize was called is insufficient.
The local recipient harness must enforce session fencing/no late callbacks. This
can be built/tested independently of TypeScript and exercised with synthetic observers.

Stage 3A does not yet satisfy A03/O15/session semantics; Stage 3B must pass O01–O15,
applicable A/V/B schedules with precise trusted-seam limitations before describing
itself as a local observation gateway. Pure mocks cannot replace 3A durability tests.

## Stage 3C — actual authority and one separately approved binding

Supply real principal authentication, policy/epoch administration, public schemas,
certificate authenticity and trusted release jurisdiction. Demonstrate queued/retry/
retained-data revocation safety and opaque forbidden/nonexistent responses under
actual enforcement. Pick transport/framing/encoding ONLY in a separate review;
this plan selects none. Test descriptor negotiation, key rotation, retention/cursor
expiry and bounded downstream output under real failures. Add TypeScript adapter
integration using its Stage 2A protocol tests, not a different inferred contract.
A second independently implemented backend is needed for tested cross-backend claims.

Before integration, align TypeScript's non-normative mandatory assertionStatus sketch
with profile O1: whole-projection evaluation is not one assertion's status. Review
scope/parameter normalization, public schemas, cursor/barrier semantics and snapshot-
first ordering together; no symmetry of Go and TypeScript APIs is required.
No TypeScript file is changed or approval assumed here.

## Concrete review decisions required before any kit

- Approve stage 3A scope/API proposal and one projection model/schema with explicit
  value/evidence equality and public/nonpublic fields. Who certifies its finite cut?
- Identify projection consistency and active policy/descriptor owners, source
  checkpoints and stale-evaluation rejection. Cross-owner atomic cuts are deferred
  unless an explicit shared authority/fencing contract is added.
- Approve authorized-view identity/generation and policy-change invalidation policy;
  avoid shared-observer caches initially. Agree initial unavailable/revision-0 rules.
- Select journal durability/recovery assumptions, receipt and replay retention/capacity,
  session/pin quotas and overflow behavior. No silent ordinal wrap or record eviction.
- Define cursor/certificate trust roots, restart/key-rotation/expiry behavior; they are
  not self-authenticating because a Go object has Certified=true.
- Before 3B, name the irreversible disclosure boundary and bounded callback contract,
  revocation effective order, possible-release auditing and permitted error metadata.
- Before 3C, identify actual production authentication/authority and binding semantics.
- Obtain separate implementation/kit/commit/model-call authorization. Planning-only
  instructions do not authorize starting a coding loop or committing these documents.

## Missing decisions versus normative blockers

Ratified O/A/V rules already resolve snapshot/barrier, as-of freshness, cursor binding,
revocation ordering and full replacement semantics. No contradiction was identified
in the bounded one-owner design. Certificate encoding, exact budgets/retention,
public equality/normalization, key lifecycle and release API are implementation/
deployment decisions that still require approval, not invitations to guess authority.
Stronger live-freshness leases, arbitrary filtered historical feeds, automatic producer
migration and cross-scope atomic projection cuts need additional contracts if later
requested; the minimum profile does not promise them. Stop affected work if such
requirements become necessary or a genuine semantic conflict appears.

## First disproof experiment and stop condition

Run the two tiny journal/authority schedules in OBSERVATION-TEST-PLAN.md after
implementation authorization: commit41 between snapshot40 and tail activation,
then revoke before versus after irreversible release. Add retention pressure and
reopen at those cuts. Fail immediately on missing41, false barrier, wrong public
revision or revoke-first payload/metadata escape. No client, broker or transport is
needed to invalidate the architecture cheaply.

For this assignment the stop condition is the four uncommitted documents and owner
review. No test execution/durability/production authorization claim is made. The
repository and profile identities/checksums were checked; only documentation added.
