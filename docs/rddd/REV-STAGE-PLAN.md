# Pure REV stage plan — corrected draft.6

The owner authorized this kit and bounded loop. Earlier planning-only/R1 gates
are superseded by corrected draft.6; no original-slot implementation remains.
See REV-DESIGN.md for API comparison and immutable upstream pin.

Smallest coherent deliverable: a pure current-assertion reconstructor plus
per-key reconciliation planner. Historical replay remains unchanged CORE.
No durable publication, storage, transport, AUTH/EFFECT or participant execution.

Tasks:
- R6VIEW: contiguous scoped revisions, gaps/conflicts/status/coverage.
- R6PLAN: pure correction/retraction/unresolved/recovery intentions.
- R6FIX: eight applicable unchanged draft.6 fixture cases, including CORE replay.
- R6ALL: every test, Go race/vet and existing Node/frontend checks.
Preserve prior CORE tasks and all CORE source/tests as protected baseline.

Protected contracts.go, tests, fixtures, specification, docs and judge configuration;
zero-return implementation stubs. Private reference outside checkout. Unchanged
v1.1.1 loop exact --check judge must fail every NEW task on stubs and pass every
task with reference, including earlier tasks/format/check. Commit stub baseline
only after verification, assert clean, then:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Use skill's pinned commands, no substitutions. Unique R6 task IDs avoid prior
CORE attempt-counter reuse. Logs stay ignored under .loop/.

Acceptance: focused tests, full Go tree, full race tree, vet, unchanged fixture
comparison, diff review and unchanged loop comparison. No conformance claim for
five deferred shared runtime cases or full durable REV.
Future stages require separate authorization: durable ordered publisher/outbox,
restart/retry/fencing, trusted admission/disclosure and effect commitment,
projection notification, explicit domain-key dependency/nonconvergence machinery.
