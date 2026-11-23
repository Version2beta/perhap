# CORE requirement-to-test proposal

All tests are proposed, unimplemented and unverified. Owners below are local
package responsibilities, not claims of upstream approval. All CORE rows target
Stage 1. C1 is now resolved by draft.5 §4.3 and six new fixture vectors. Source pin: SOURCE-SHA256SUMS.

## Normative obligations

| Source | Observable assertion / independent test | Local owner |
|---|---|---|
| §3.1, §4.1 | Evaluate needs no broker, filesystem, wall clock, participant or publisher; identical explicit observations give identical outputs | pure API, review and deterministic tests |
| §3.2 | Identical admitted identity retry collapses; conflicting content/attribution/links diagnosed and never overwrites retained record | identity normalization; admission-phase fixture harness |
| §3.2 | Trusted attribution is consumed, not authenticated by a string field; preserve immutable records and subjects without single-aggregate assumption | request adapter; real admission deferred AUTH |
| §3.3, §4.2.2–5 | Present parent outranks ID, multiple parents respected, byte-smallest ready ID first, no timestamp inference | graph/linearizer |
| §4.2.3 | Required local/external and unknown missing predecessors block dependent chains; trace-only external does not; supplied admissible proof unblocks required foreign evidence | dependency resolver input/graph |
| §4.2.4 | Cycles and descendants blocked, unrelated nodes evaluated, all records retained | graph |
| §4.1 | Initial valid/invalid/null/failed states distinguish availability; invalid initialization repairs; initial validator error halts without proposals or fabricated last-valid | initialization |
| §4.2 | Invalid successful candidate replaces working and feeds later repair; last-valid/prefix stays separate | fold |
| §4.2 | Reducer/validator errors discard mutated candidate and hints; independent events may proceed; occurrence child proceeds, success child blocked | fold |
| §4.3 | Separate evaluation, completeness, working availability, last-valid availability/prefix and rule/frontier provenance | result; TestCOREBehavior and TestCOREFixtures cover clarified precedence |
| §4.3 | Known gaps never certify complete; missing frontier never certifies complete; known gap takes precedence; otherwise unavailable without certified continuity | result |
| §4.4 | Insert late event, replay enlarged history, compare to one-pass final set; do not reject by snapshot ID | Evaluate |
| §7.1 | Selector uses final state/effective evidence, not union of intermediate hints; unsupported action disappears; args/key retained, never executable authority | selector |
| §10 | Stable diagnostic key from scope/event/stage/condition/rule; unrelated history does not change key; repeated evaluation does not publish duplicate events | diagnostics |
| §11 | Report exact provisional revision and only implemented capabilities; no unqualified conformance claim | package docs/evidence |

Pure derivation outputs belong to result design; stable publication identities,
revision/retraction reconciliation and durable diagnostic publication are deferred
REV/runtime concerns, not implicit CORE publication.
Proposal withdrawal versus unresolved reconciliation across evaluations requires
the separate host lifecycle; CORE must provide enough status/evidence to prevent
absence on failed/incomplete recomputation from implying withdrawal.
Effect evidence isolation/commit authority is deferred EFFECT, not a global
commit_eligible=true on valid CORE state.

## Every shared CORE case

Each row maps to a proposed TestCOREFixtures/<exact-case-id> in the test-only
counter/v1 adapter, comparing every expected field without case-ID branching.
Admission cases use their explicitly separate admission phase; replay case checks
both before and enlarged histories. Shared fixture bytes must stay unchanged.

| Fixture | Additional targeted guarantee |
|---|---|
| core-empty | Valid initialized empty state |
| core-independent-id-tiebreak | set before add despite input order |
| core-cause-overrides-id | z parent precedes a child |
| core-missing-local-independent | Missing local blocks only affected chain |
| core-causal-cycle | Cycle retained, independent c evaluated |
| core-failed-parent-occurrence-dependency | Occurrence child runs after failed parent |
| core-failed-parent-success-dependency | Success child blocked after failed parent |
| core-validation-repaired | Negative candidate retained, then repaired |
| core-trace-only-external-cause | Foreign trace does not block |
| core-unknown-cause-classification | Unknown classification blocks |
| core-invalid-state-pure-proposal | Invalid diagnostic proposal has no execution authority |
| core-late-event-replay | Full reconstruction changes 3 to 12 |
| core-admission-identical-retry | One admitted fact |
| core-admission-conflicting-id | First admitted content retained, conflict recorded |
| core-invalid-candidate-retained | Current -2 differs from last-valid 0 |
| core-invalid-initial-repair | Initial -2 becomes 3 |
| core-invalid-initial-no-repair | Available invalid state, no last-valid |
| core-initializer-failed | No reduction, unavailable working |
| core-initial-validator-failed | Initial state retained only as failed evidence |
| core-proposal-final-selection | Earlier positive hint loses support |
| core-proposal-final-selection-supported | Supported final candidate remains non-executable |
| core-proposal-explicit-cancel | Final cancellation removes key |

## Independent tests, beyond shared vectors

- Exhaust all permutations of several small valid admitted histories and compare
  full semantic results, including diagnostics, prefix and proposal ordering.
  Do not permute conflicting submissions as though both were admitted facts.
- Random finite DAGs with multiple dependencies; verify every parent precedes child,
  and independent ready-node comparison. Add missing-parent chains and cycle descendants.
- Reducer mutates map/slice then errors; validator fails after mutation;
  no prior state, last-valid value, input event or caller-owned data changes.
- Nil as valid state distinct from unavailable; initializer error versus invalid
  initializer versus validator error; clone errors cannot silently alias.
- Repaired invalid state followed by callback failure remains failed; success and
  occurrence dependencies remain distinct; test transitive blocking.
- Stable diagnostic keys under unrelated history growth and repaired conditions.
- Unsupported/unknown event remains historical; multiple aggregate subjects allowed.
- Same final history via late insertion and arbitrary batching equals full replay.
- Final selector sees final state and applicable evidence; failed invocations cannot
  contribute accepted reducer hints; all proposals remain non-executable.
- Missing/uncertified frontier is unavailable, known gaps take precedence;
  TestCOREBehavior and the six draft.5 fixture vectors cover this resolution.

## Draft.4/.5 additions

TestCOREFixtures also covers core-blocked-proposal-late-parent (blocked inputs
cannot propose until replay with their parent), core-no-frontier-empty-not-complete,
core-no-frontier-with-apparent-valid-model,
core-explicit-uncertified-frontier-not-a-known-gap,
core-known-required-parent-missing-without-frontier,
core-verified-gap-without-blocked-event, and
core-specific-gap-dominates-nominal-frontier.

The detailed list above remains the broader test proposal, not a claim that every
independent property is implemented. Current executable scope: TestCOREBehavior,
TestCOREOwnership, TestCOREPermutations and all 29 shared cases. Random DAGs,
exhaustive multi-parent graphs and generalized environment encoding remain
additional validation work; do not imply full CORE certification from these tests.

## Explicit deferrals

AUTH: trusted admission enforcement and observation filtering, real principals.
REV: revisions.json and publication/retraction/effective logical-slot machinery.
EFFECT: effects.json, commitments, fencing, deduplication and outcome reconciliation.
PARTICIPANT/CLIENT/INTEROP-JSON/STATIC: all associated runtime/profile obligations.
Durable resource limits, retention/restart, NATS replay and disclosure are not
provided by the pure library. No shared suite outside CORE is claimed.
