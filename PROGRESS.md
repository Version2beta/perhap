# CORE Stage 1

Verified kit baseline; implementation begins from stubs. See SPEC.md.
T1: Implemented pure finite-history Evaluate (sdk/go/core/evaluate.go); TestCOREBehavior, TestCOREOwnership, TestCOREPermutations pass.
Changes: sdk/go/core/evaluate.go only (admission by ID bytes, dependency-ordered processing with smallest-ready selection, callback clones, failure/coverage/diagnostic rules); PROGRESS.md note appended.
Remaining: Interpretations not fixed by SPEC: initializer/initial-validator failure blocks all events without per-event diagnostics; requires-success blocks after reducer failure count as incomplete (fixture-driven); clone_failed/selector_failed/callback_missing are local condition names; Select runs despite sticky failure; the full package incl. T2 fixtures also passed as an observation, not a separate T2/T3 judgment.
