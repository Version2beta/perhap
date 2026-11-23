# Perhap pure CORE Stage 1

Binding local contract, subordinate to rDDD 0.1.0-draft.5 SPEC §§3–4, 7.1,
10–11 and CONFORMANCE's counter profile. Upstream HEAD:
c53e58e99112e77ba6d934edc17e7956f4c38938, with uncommitted draft.5 contents.
Exact provisional snapshot: docs/rddd/SOURCE-SHA256SUMS; upstream manifest
SHA256 078fe67775596c8af6b7a991fb4079a8110b95457833fc16ee52d35b3a13fa14.
Candidate, not ratified. Mapping/migration: docs/rddd/.
Shared fixture bytes under sdk/go/core/testdata/core.json must remain unchanged.

Domain authors use sdk/go/core.Evaluate to reconstruct a finite, already-admitted
set under a pure Model. Request -> normalize/graph -> initialize/validate ->
canonical fold -> final selector -> Result. New package only; no legacy changes.

## Frozen API and local choices

contracts.go is complete/protected. S is private Go state, not mandatory JSON.
Event.Payload and proposal/derivation values are immutable domain-defined strings.
Domain-specific schemas supply equality/encoding before admission; no claim of
INTEROP-JSON or JCS. Equal immutable Event fields represent a retry; conflicts
retain the first already-admitted record, emit conflicting_id, never overwrite.
Do not treat conflicting submissions as an order-independent admitted set.
Preserve History in ID byte order, with copied Subjects/Dependencies slices.
Multiple/no aggregate subjects are allowed; no UUID requirement.
All callbacks mandatory, pure, deterministic, terminating, and error-returning.
Clone must isolate mutable state; do not silently alias on errors. Never give
owned state to Reduce, Validate, Select without cloning it first. Copy event
slice fields before callbacks. Initializer state also needs isolation.
Callbacks may capture immutable explicit observations; RuleVersion identifies
the whole deterministic model/observation contract. No ambient time/randomness.

## Required algorithm

- Coverage: a verified Gap or known blocked prerequisite/cycle is incomplete.
  Otherwise Certified with nonempty Boundary is complete_as_of_frontier.
  Otherwise unavailable. Compute dependency coverage independently of initializer
  failure. Reducer failure alone does not establish a history gap.
- Local dependency edges order all present parents before children, regardless
  of ID. Missing local/unknown/unsatisfied required-external evidence blocks
  event and descendants. A trace-external reference does not block; external
  Satisfied is trusted proof, not validated by this library.
- Detect cycles, block them and their descendants, retain all history. Process
  unrelated eligible nodes. At each step select byte-smallest ready ID.
- Initialize then validate before reduction. Error initializing: working and
  last-valid unavailable, failed, block all ordinary events, no selector.
  Initial validator error: retain initialized working evidence, failed,
  no last-valid/reduction/selector. Initial domain-invalid: retain working,
  no last-valid, continue so events may repair.
- A successful reduction and evaluated validation replace working even if
  domain-invalid. Valid candidates update independently cloned last-valid and
  the attempted evaluated prefix. Invalidity can repair; callback failure is
  sticky failed even after independent valid transitions.
- Reducer or validator errors discard that candidate; retain triggering event
  in Attempted. Occurrence-only children may proceed; RequiresSuccess children
  block, as do their descendants. Blocked is ID sorted; Successful lists only
  successfully processed events. Do not conflate attempted and successful.
- Final Select receives isolated final working state and successful events in
  canonical order. Not union of intermediate hints. Failed selection discards
  outputs and reports failed. Proposal key/action/arguments are descriptions,
  never execution authority. Failed/incomplete absence is not withdrawal.
  Derivations are pure values, not publications.
- Diagnostics carry stable keys based on context/scope/event/stage/condition/
  rule identity (unrelated history cannot change a key), callback errors, and
  conditions initializer_failed, initial_validator_failed, initial_domain_invalid,
  reducer_failed, validator_failed, domain_invalid, dependency_unresolved,
  causal_cycle, conflicting_id, history_gap. Callback/clone/missing-callback
  failures must be explicit, never fabricate valid state.
- Full replay on each call implements late reconstruction; no snapshot cutoff.
- No I/O, transport, event publication, authentication enforcement, effect
  execution, durable recovery or agent infrastructure.

## Evidence boundary

Task T1: independent behavior/ownership/permutation tests.
Task T2: all 29 unchanged shared fixtures via counter adapter.
Task T3: whole existing test tree and checks. Every task must fail on stubs.
This is a bounded CORE implementation slice, not certified complete rDDD.
Full randomized graph coverage, durable runtime and other capabilities remain
outside this kit. The protected tests are evidence, not replacements for
upstream semantics. Implement generally; never special-case fixture names.

## Part 2 — Pure REV Stage 2 (draft.6)

The binding additional stage contract is docs/rddd/REV-SPEC.md, pinned to
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87. This does not replace or change
CORE's contract above. Historical replay never performs original-slot replacement.
See docs/rddd/REV-TRACEABILITY.md for eight applicable and five deferred REV cases.

## Part 3 — Publication Stage 1 ledger

Binding additional contract: docs/rddd/LEDGER-SPEC.md, subordinate to locally
ratified INTEROP-PUBOBS/0.1 v0.1.0 commit 9a80977efaf2ff740a257f6e30314dac42ec384e
and unchanged corrected draft.6 baseline. Only sdk/go/ledger implements this stage;
no CORE/REV changes, network dispatcher or public observation service.

## Part 4 — Publication Stage 2 local durable delivery

Binding additional contract: docs/rddd/DELIVERY-SPEC.md. Preserve Stage 1 ledger
format and semantics. Only sdk/go/delivery implements the additive durable sink,
receiver-bound confirmation journal and bounded delivery passes. No broker binding,
public observation, production AUTH, projection application or effects.
