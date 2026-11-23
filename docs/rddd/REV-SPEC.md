# Binding pure REV Stage 2 contract

Subordinate to draft.6 SPEC §5 at committed
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; not a ratified standard.
Provenance, mapping, scope and API tradeoffs: REV-DESIGN/TRACEABILITY/STAGE-PLAN.md.
contracts.go is complete/protected. Implement in sdk/go/rev/revisions.go only.
No modification of CORE or runtime. Standard library plus existing CORE only.

Reconstruct consumes already admitted, attributed, verified publications and a
trusted frontier for their required evidence scope. Values are domain-canonical
immutable strings, not raw encodings whose semantic equality is unspecified.
Key fields scope producer/subject/meaning/context; no history-derived key.

History is an independent copy of ALL supplied records in supplied order, including
identical retries as supplied evidence; no payload replacement, removal or relocation.
Actual event canonicalization/reduction belongs to CORE and the domain, not this
view function. Assertion selection must be independent of input arrival ordering.

For this local internal profile, ordinals start at 1 per key. Deduplicate identical
records for view processing, not History. Sort by revision (never event ID);
each ordinal must be contiguous. Optional supersedes, when supplied, must refer
to immediately preceding verified publication ID. Empty ID, invalid operation or
invalid link cannot establish current authority. Conflicting same-ID content or
different records at the same key/ordinal leave affected keys unresolved/incomplete.
Do not choose a winner by ID or arrival order. Retain the last unambiguous contiguous
prefix value where available; do not accept a later unsupported candidate. At a
conflict no conflicting ordinal is authoritative. Independent keys remain separate.

upsert sets current/value available (including encoded null). retract marks retracted
and value unavailable, not undo of historical transitions. status changes status to
current/unresolved without changing value; current requires available prior content.
Successful contiguous publications set Revision and PublicationID. Known gaps,
conflicts, bad links/operations/statuses mark affected assertions unresolved and
view completeness incomplete with structured Key/Revision/Condition diagnostics.
Condition vocabulary: revision_gap, conflicting_id, conflicting_revision, invalid_link,
invalid_operation, invalid_status. Diagnostic order is deterministic by key/revision
where possible; duplicate reports are not publications.

If Frontier.Gap or any demonstrable required chain gap exists, incomplete wins.
Otherwise certified nonempty Boundary yields complete_as_of_frontier; otherwise
unavailable. Uncertified or globally gapped scope cannot establish authoritative
current keys: mark unresolved while retaining last known value evidence.
This scope-level frontier rule does not prevent a caller from evaluating isolated
required evidence closures separately.

Reconcile receives previous known Assertion plus one Computation for that key's
required evidence closure. It does not allocate durable revisions or event IDs.
The returned Assertion retains prior Revision/PublicationID until publication.
ExpectedRevision is the prior ordinal, a future publisher precondition.
- Non-valid or non-complete computation: unresolved, retain prior value and
  availability; propose status unresolved unless already unresolved.
- Valid complete absence: retracted/value unavailable; propose retract unless
  already retracted.
- Valid complete present: current with computed value. Propose upsert if no prior
  value, different value, or previously retracted; no intention for unchanged
  current content; status current for unchanged content recovering unresolved.
Every intention carries Key, ExpectedRevision, Op and value/status as applicable.
Neither a current Assertion nor an Intention grants effect or disclosure authority.

Eight applicable shared fixtures must pass unchanged. Historical counter adapter
must attempt b,c,z in causal order and produce 7 for both corrected-upsert and
retraction fixtures, while current assertion is respectively 7 or retracted.
No synthetic slot/anchor graph, universal undo or retrospective cycle.
Five runtime/CLIENT/AUTH/EFFECT/PARTICIPANT fixtures are explicitly deferred,
not skipped-and-counted as passed. No full durable REV conformance claim.
