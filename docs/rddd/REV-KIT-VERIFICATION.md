# Pure REV kit verification before baseline commit

Base c68bf137a819d48f1bb8e11d083555419a88c7a0, branch rddd-foundation.
Upstream committed clean draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
all 17 SHA256SUMS entries pass. Exact manifest copied as REV-SOURCE-SHA256SUMS;
manifest hash ab81ac9b68bf71d1e19c4f338b8e4b50df097a40218b4e41384288a6a5c9f816.

Updated the existing uncommitted three REV documents for corrected semantics;
removed slot reconstruction, anchor integration and obsolete R1 blocking gate.
No existing CORE source, API, tests, fixtures or runtime changes.

Private independent reference outside checkout, not shipped/committed.
Temporarily installed only as sdk/go/rev/revisions.go:
- go test -count=1 -v ./sdk/go/rev: reconstruction/reconciliation and eight
  applicable updated shared fixtures pass; historical set2/add5/set7 yields 7.
- scripts/loop.sh --check: exit 0; T1, T2, R6VIEW, R6PLAN, R6FIX, R6ALL pass.
  Exact stdout expectations include each of the eight fixture subtests.
  Includes format_check, earlier tasks, kit go vet, full Go tree, full race tree,
  seven complexity tests and frontend tests.
- cmp loop against installed spec-loop v1.1.1: identical.
- cmp revised shared fixture copy against upstream: identical.
- git diff --check: clean.

Restored zero-return stubs, formatted, and repeated scripts/loop.sh --check:
T1/T2 pass; R6VIEW/R6PLAN/R6FIX/R6ALL each FAIL with test-command exit 1.
No coder invoked for verification. Earlier tasks are intentionally not new
stub-failure obligations. Reference is removed from checkout before baseline.
No full durable REV conformance claim; five shared runtime cases are deferred.

Next authorized command after local baseline commit and clean-tree assertion:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Skill's pinned model defaults unchanged. No push or upstream mutations.
