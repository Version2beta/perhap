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
