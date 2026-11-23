# Stage 3B1 verified kit evidence

Initial HEAD 0dc2d511ba03d6c0b62198eea3fba17a3a391b38, clean rddd-foundation;
no later work existed. Preserve CORE/REV and publication stages1/2/3A unchanged.
Explicit bounded split authorized by user: complete finite bootstrap/session/release
slice, not all Stage3B. SESSION-SPEC.md states deferred cursor/resume/reset scope.

Pins: draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
ratified INTEROP-PUBOBS/0.1 v0.1.0 9a80977efaf2ff740a257f6e30314dac42ec384e.
All 19 checksum entries pass, manifest digest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
No external repository changed. Existing ratified fixtures and prior tests preserved.

Requirements map, ownership, callback trust/deadline assumptions, one-grant Gate,
quota/cleanup semantics and minimal API were documented before freezing the kit.
New observation package only; no new dependency/transport/wire encoding.
Private independent reference written outside repository and temporarily copied only
as session.go for verification, removed before stub baseline. Not shipped/committed.

Actual verification:
- go test -count=1 -v ./sdk/go/observation: five substantive tests pass. Actual
  Stage3A SQLite source seeded S/40, session prepares retained cut, S/41 commits
  before Step: exactly snapshot40, revision41, bound authority barrier41. Capacity
  refuses42; next Step cannot invent freshness beyond41. Reopened journal supports
  independently authorized fresh bootstrap, not resume of an old session.
- Gate test uses deterministic channels to establish callback release-first order;
  revocation follows the SAME locked boundary and denies subsequent callback.
  Separate revoke-first session invokes zero recipient callbacks/metadata releases.
  This tests the synchronous injected local boundary, not production policy.
- Negative page tests: gap, conflicting revision and scan error produce no successor
  or barrier after snapshot. Wrong observer/view cannot bootstrap. Incompatible
  descriptor and mid-session version change fail before further disclosure.
- Session/pin capacity, canceled open, repeated close, canceled in-flight callback,
  uncertain recipient failure plus cleanup failure (both retained), cleanup retry,
  and deadline-aware callback completion tested without sleep-based schedule proof.
- scripts/loop.sh --check with reference: exit0, all SEVENTEEN tasks pass including
  prior13, exact passing stdout, format_check and earlier-task/kit checks. Final task
  includes full Go/race/vet, Node complexity and frontend tests.
- Restore zero-return stubs, format and rerun SAME judge: exit1; all FOUR new tasks
  B1BOOT/B1RELEASE/B1FAIL/B1ALL fail with test-command exit1; prior13 pass.
- Judge compared byte-for-byte with installed spec-loop v1.1.1 script; unchanged.
  Diff whitespace check clean. No previous code/test/dependency changes.

Cursor integrity, expiry/restart/generation binding, verified resume and certified
resnapshot are NOT implemented/tested by 3B1. No full O01–O15/A01–A06/V or CLIENT
conformance claim. Existing Stage3A subprocess durability tests remain the storage
crash evidence; new tests use actual reopen for fresh bootstrap only.
Authority certificate seals are representative trusted test evidence, not production
crypto trust or credential verification. Cooperative context-obeying callbacks and
no deferred/uncontrolled asynchronous release are required. A noncooperative Go
callback cannot be forcibly bounded safely; no orphan timeout worker is introduced.

Authorized next command after local verified stub baseline/clean gate:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Use unchanged pinned skill commands, new B1 task IDs. No push/merge/remote changes.

## Corrective verification after initial judged implementation

Initial kit 078da114b7083812fd57281df3eb4868265eb24b; first judged commit
92eff687694c394618000ee492f3deca95ceabc6 passed original17 tasks. Final source
review identified two untested contract edges. Added TestSessionCancellationEdges:
cancel-before-step proved a retained pin/quota leak; expired-success-not-certified
proved a nil return after callback deadline could wrongly certify a barrier.
Both failed against that actual judged implementation, not merely a mock.

New exact B1EDGES judge fails the incomplete implementation, B1ALL fails too; all
other16 focused tasks pass. The independently written original private reference
already handles both cases and passes ALL EIGHTEEN judge tasks with new regressions,
including full Go/race/vet/prior checks. Restored judged implementation unchanged
before committing this test/spec/task overlay; private reference remains outside kit.
No tests weakened, no prior-stage changes. Corrective bounded loop uses new B1EDGES
ID and unchanged configured commands. Initial one coding iteration plus this bounded
repair remains within the requested 12-iteration total budget if completed normally.
