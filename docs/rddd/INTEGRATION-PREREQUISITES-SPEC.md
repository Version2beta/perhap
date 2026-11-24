# Binding optional Go integration prerequisites

Approved GO-CLIENT-INTEGRATION-DESIGN.md; pinned draft.6
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87, PUBOBS v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e; all19 checksums verified; manifest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
No transport/client implementation in this prerequisite kit.

Recovery.ResumeCertified is additive. Perform actual existing Resume, then BEFORE
returning session and before any Step, issue purpose resume proof over accepted
anchor/current Binding and deliver kind resume with that record/session through the
existing current-authority Gate fence and synchronous recipient. On failure retire
and observe cleanup, joining primary+cleanup error. Existing Resume/Step unchanged.
No handshake on invalid cursor; existing explicit cannot-resume remains. No automatic
replacement, no silent advance, no new public revision. This handshake proves renewed
scope and established retained position, not client application or ongoing freshness.

Client.ResolveMatching is read-only: authorize lookup with expected Input/binding,
normalize via current registered versioned codec, verify non-Data fields unchanged,
compare against authoritative retained schema/event/asserting identity under owner
mutex. No call to Submit/Admission/Evaluate, no writes/receipt/version changes. Missing
or other unauthorized-context knowledge gives existing no-retained-knowledge policy;
matching returns latest complete facts. Mismatch is Reply.Kind identity-conflict with
zero Facts, delivered THROUGH Host.Release, not a naked unfenced conflict disclosure.
Existing ID-only Resolve unchanged. Unavailable/unknown storage and malformed input
remain distinct errors and cannot become successful absence. Same resource bounds,
copy ownership, cancellation and canonical public errors as existing client methods.

Tests: resumed43 -> certified handshake43 ->44/45/barrier, actual proof verification,
old API emits no handshake, revoke-first emits no handshake or metadata, failure owns
cleanup. Matching1/1.0, mismatch/schema/attribution, current latest versions, absent
lookup produces no receipt, no Admission callback or evaluation, actual SQLite reopen,
concurrent matching/conflict, authority/revocation and unknown store refusal. Existing
all29 judge groups preserved as focused groups plus replacement whole-tree command.
Independent reference outside checkout, failing stubs, exact unchanged judge and clean
kit baseline before MAX_ITER=12 MAX_ATTEMPTS=3 loop. No production trust or wire claims.
