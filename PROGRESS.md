# CORE Stage 1 and pure REV Stage 2

REV stage: corrected draft.6; history and current assertions remain distinct.
See docs/rddd/REV-SPEC.md. No original-slot replacement or CORE production changes.

Verified kit baseline; implementation begins from stubs. See SPEC.md.
T1: Implemented pure finite-history Evaluate (sdk/go/core/evaluate.go); TestCOREBehavior, TestCOREOwnership, TestCOREPermutations pass.
Changes: sdk/go/core/evaluate.go only (admission by ID bytes, dependency-ordered processing with smallest-ready selection, callback clones, failure/coverage/diagnostic rules); PROGRESS.md note appended.
Remaining: Interpretations not fixed by SPEC: initializer/initial-validator failure blocks all events without per-event diagnostics; requires-success blocks after reducer failure count as incomplete (fixture-driven); clone_failed/selector_failed/callback_missing are local condition names; Select runs despite sticky failure; the full package incl. T2 fixtures also passed as an observation, not a separate T2/T3 judgment.
T3: Whole-tree run is not reliably green: sdk/go core tests, go vet, complexity node test and demo/web npm test passed in the observed runs; go test ./... passed once and failed in the loop's run and 2 of 15 isolated runs of sdk/go TestService_ReplayFromJetStreamSortsAndSkipsInternalSubjects.
Changes: PROGRESS.md note only; no sdk/go/core file changed because the failing test is outside the allowed scope and the core tests already pass.
Remaining: Intermittent sdk/go failure from string-order assumption on NewEventID (sdk/go/event.go, google/uuid v1.6.0 NewV6 gave 37 inversions in 200k consecutive IDs in a /tmp probe); fixing it needs an authorized change outside sdk/go/core.
R6VIEW: Implemented pure Reconstruct (sdk/go/rev/revisions.go) so TestREVReconstruction passes; Reconcile remains a stub for R6PLAN.
Changes: sdk/go/rev/revisions.go only (distinct history copy, per-key contiguous walk from revision 1 stopping at gap/conflict/bad record, conflicting_id/conflicting_revision/invalid_link/invalid_operation/invalid_status/revision_gap diagnostics, frontier-based completeness and unresolved status); no contracts, tests, fixtures or CORE change.
Remaining: Interpretations not fixed by spec: empty ID reported as invalid_link, revision 0 as revision_gap, non-empty Status on upsert/retract as invalid_status; TestREVReconciliation and TestREVFixtures (rev-failed-recompute, which needs Reconcile) still fail as stubs; race/whole-tree (R6ALL) not run.
R6PLAN: Implemented pure Reconcile (sdk/go/rev/revisions.go) so TestREVReconciliation passes; uncertain, absent and present computations map to unresolved, retract, or upsert/status intentions per REV-SPEC, with prior Revision/PublicationID and ExpectedRevision retained.
Changes: sdk/go/rev/revisions.go only (Reconcile plus unexported reconcileUncertain/reconcileAbsent/reconcilePresent helpers); no contracts, tests, fixtures, CORE or PROGRESS changes beyond this note.
Remaining: Interpretations not fixed by spec: retract is proposed for complete absence even when the prior Assertion is zero-valued (never published); status intentions carry no Value; TestREVReconstruction, TestREVReconciliation and TestREVFixtures pass in the package run, but race, whole-tree (R6ALL) and git-state review were not run.
