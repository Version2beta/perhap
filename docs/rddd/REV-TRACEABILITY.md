# REV traceability — draft.6 pure subset

Pin: 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; details in REV-DESIGN.md.
This revises the earlier proposal without retaining its obsolete slot/R1 rules.

| Obligation | Executable evidence / owner |
|---|---|
| §5 scoped stable derivation identity | Key fields; TestREVReconstruction isolates producers |
| §§3.2,5 retries/conflicting identities and ordinals | TestREVReconstruction duplicate/conflict cases; immutable History retained |
| §5 ordered contiguous revisions independent of UUID | reverse-ID ordinals and all six permutations, missing ordinal and wrong link |
| §5 current/retracted/unresolved separate from history | TestREVReconstruction status stream, gaps, nullable encoded value |
| §§4.3,5 known gaps dominate unavailable frontier | TestREVReconstruction certificate matrix; host supplies trusted evidence |
| §5 changed/unchanged/retracted derivations | TestREVReconciliation content changes and complete valid disappearance |
| §5 incomplete/invalid/failed is unresolved, not retract | 3x3 evaluation/coverage matrix; stale evidence retained |
| §5 same-value status recovery | TestREVReconciliation, recovered-same-value fixture |
| §5 every publication keeps its historical position | TestREVFixtures through unchanged CORE, set2/add5/set7 = 7 |
| §5 retract does not undo earlier business transition | TestREVFixtures count remains 7, retract event still attempted |
| §5 historical causal references immutable | fixture adapter carries b -> c -> z; no original-slot rewriting |
| §5 durable sequence survives restart | DEFERRED durable publisher; pure intention only has expected prior revision |
| §5 logical-key cycles / downstream convergence | DEFERRED domain-specific dependency/orchestration model and bounded convergence tests |
| §§5–7 publication/disclosure/effect authority | DEFERRED AUTH/EFFECT/runtime; no execution mechanism |
| §9 same-frontier projection notification | DEFERRED publisher/CLIENT |
| §10 structured diagnostics | revision_gap, conflicting_id/revision, invalid_link/operation/status; no error publication |
| §11 capability claims | pure REV subset only, not full durable REV |

## Shared fixtures (unchanged draft.6 JSON copied beside tests)

Eight cases exercised by TestREVFixtures:
- rev-upsert-change
- rev-unchanged-recomputation
- rev-retraction
- rev-missing-number
- rev-later-upsert-not-retroactive-event-replacement
- rev-retraction-does-not-undo-historical-transitions
- rev-failed-recompute-is-unresolved-not-retraction
- rev-recovered-same-value-clears-unresolved

Five explicitly deferred cases, not counted as passing:
rev-equal-event-frontier-updated-projection (CLIENT/runtime);
rev-agent-judgment-not-recomputed (PARTICIPANT integration);
rev-provisional-authorized-publication and rev-provisional-disclosure-denied
(real AUTH/disclosure); rev-constitutive-publication-is-effect (EFFECT).
The pure library has no agent or publication boundary to exercise these honestly.
Fixture names are harness dispatch only; implementation cannot inspect them.
Further evidence: generalized randomized histories, cross-key dependency cycles,
status-stream concurrency and restart-safe publication remain future work.
