# CORE translation design — C1 resolved in draft.5

## Authority and scope

Target: github.com/version2beta/perhap, worktree
/Users/rob/Development/perhap-rddd-foundation, branch rddd-foundation,
base b274799f1a246c9d19d08eac8af25a8a2ffb1a3a (initially clean).
Read the complete /Users/rob/Downloads/rddd-spec-loop-handoffs/go.md.
The owner's execution authorization supersedes its planning-only restriction,
but not the requirement to stop for observable semantic ambiguity.

Upstream: /Users/rob/Development/rDDD-spec, version 0.1.0-draft.5,
HEAD c53e58e99112e77ba6d934edc17e7956f4c38938.
This is a **provisional pinned local snapshot**, not an immutable approved revision.
All 16 entries in SHA256SUMS passed shasum -a 256 -c SHA256SUMS.
The manifest fingerprint is
078fe67775596c8af6b7a991fb4079a8110b95457833fc16ee52d35b3a13fa14.
See SOURCE-SHA256SUMS for exact per-file hashes including the manifest.
Modified upstream files: CONFORMANCE.md, DECISIONS.md, PAPER-ALIGNMENT.md,
README.md, SHA256SUMS, SPEC.md, conformance/{authority,core,effects,revisions}.json.
Untracked upstream files: CHANGES-draft.3.md, CHANGES-draft.4.md, CHANGES-draft.5.md,
conformance/reference_core.py, conformance/reference_revisions.py.
No upstream files were changed by this work.

CORE only: a library for domain authors to replay finite admitted histories,
inspect provisional state/status, and derive pure proposals. No runtime,
publication, enforcement, execution, agent, recovery, or transport claims.
INTEROP-JSON and STATIC profiles were inspected, not selected capabilities.
Shared fixture JSON is harness input, not a required Go wire envelope.

## Historical upstream semantic question C1 — now resolved

Draft.5 §4.3 and its six new fixtures resolve this question. Known required
incompleteness wins; otherwise certified continuity through an identified
boundary is complete_as_of_frontier; otherwise unavailable. Evaluation and
working availability are independent. core-no-frontier-empty-not-complete
matches the counterexample below exactly. No remaining contradiction found
in this clarification. The historical investigation follows for provenance.

SPEC §4.3 requires three completeness values but does not define the transition
to unavailable. CONFORMANCE's CORE interpretation defines incomplete for
missing causes/cycles/gaps and otherwise refers to trusted frontier state.
Its fixtures contain no unavailable case. The illustrative Python helper
defaults an omitted frontier to continuous=true; that cannot establish the
identified verified boundary required by §4.3.

Counterexample: counter/v1 initializes successfully to n=0, supplied history
is empty, there is no frontier evidence and no known missing dependency.
The state is available and valid. Completeness cannot be certified complete.
Should it be incomplete or unavailable? Both prevent completeness claims,
but they are different observable required status values.

Requested upstream resolution: define the distinction and add vectors for
absent frontier evidence, known retention gaps, and unavailable history.
A possible resolution (NOT adopted here) is unavailable for absent/unobtainable
boundary evidence and incomplete for a known gap through an identified boundary.
Also specify precedence when a known dependency gap coexists with absent
frontier evidence. Do not copy the Python default into Go.
This blocked the earlier draft.3 kit; it no longer blocks draft.5 development.
The current kit and executable contracts are described in root SPEC.md.

## Proposed Go API (not frozen)

Recommend a new package sdk/go/core with Evaluate[S,E] over an immutable
Request and Model callbacks. Request carries context/scope, admitted events,
trusted dependency classifications, frontier evidence, explicit observations,
and rule version. Model supplies Initialize, Clone, Reduce, Validate, and a
final Select function. Result contains independent working/last-valid
availability, values, last-valid prefix, evaluation/completeness, attempted
and blocked IDs, retained evidence, structured diagnostics, derivations and
non-executable proposals. Domain-invalid validation is distinct from an error
evaluating validation. Proposal identity includes stable key and action/arguments.

Alternative: a stateful Evaluator object with Add/Replay methods could reuse
incremental history and caching, but introduces mutable lifecycle, invalidation
and synchronization obligations before equivalence is established. Prefer the
single finite-history function and full replay first.

Go-local choices: generics rather than mandatory JSON state; explicit typed
dependency edges (occurrence versus processing success); stable byte comparator
for the portable harness; availability booleans rather than nil sentinels.
No context parameter, goroutines or resource acquisition for bounded pure work.
Callbacks must be deterministic and side-effect free. Clone must return errors
rather than silently alias on failure; clone before giving mutable state to a
callback and preserve last-valid independently. Test pointer/map/slice aliasing
and mutation-before-error. Return callback identity as diagnostic evidence,
without treating internal errors as public event publications.

## Migration

Existing Event requires EntityID, lacks trusted attribution and dependency
classification; timestamp extraction and UUID generation are not CORE helpers.
Existing Reducer returns events for publication, not final supported proposals.
Validator cannot represent failure separately from invalidity; initialization
uses Go zero state. supervisor.currentModel serves lastValid, unlike CORE's
explicit current working candidate. shouldSkipEvent excludes history before
the snapshot; ID sort does not honor causal edges. Service replay uses timeouts
and cannot supply certified frontier evidence. deepCopy silently aliases on
marshal failure. None of these establish the new contract.

Reuse typed domain-handler ideas and error-preservation regression scenarios,
not runtime implementation. Keep all existing APIs and tests unchanged.
A later adapter must supply admission evidence, subjects, callback failure
semantics and certified replay; cannot relabel current ModelUpdate as CORE.
Do not wire the new evaluator into NATS in Stage 1.
