# Publication Stage 3 — observation requirements and coverage

Planning only. No implementation, dependency, kit, frozen API/encoding, model call
or commit. Existing CORE/REV/publication implementation and tests remain unchanged.

## Authority and inspected state

Go: Version2beta/perhap, /Users/rob/Development/perhap-rddd-foundation,
rddd-foundation, clean HEAD 3afcc67a47f47eb9b016e78232c18b4bbdd16976.
Baseline: rDDD draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87.
Ratified optional profile: INTEROP-PUBOBS/0.1 v0.1.0,
9a80977efaf2ff740a257f6e30314dac42ec384e, clean upstream checkout.
All 19 SHA256SUMS entries pass; manifest digest:
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Profile approval is not baseline ratification or implementation conformance.

Sources inspected: profile I/O/A/V/B semantics and shared vectors; CORE/REV and
ledger/delivery contracts, ledger transaction/checkpoint and delivery confirmation
paths; DELIVERY-SPEC and six PUBLICATION-* planning documents. Earlier publication
proposals are preserved historical design inputs, not authority over ratified rules.

TypeScript checkout: /Users/rob/Development/perhap-client, verified origin
https://github.com/Version2beta/perhap-client.git, rddd-foundation, clean HEAD
8d28ee051d600881bdd6afde426727f8aa14bba4. INTEROP-REQUIREMENTS/DESIGN/FAILURES/
STAGE-PLAN were reviewed as non-normative proposals. Their historical provenance
mentions older HEAD c09645c; this review uses the actual checkout above. No approval
is attributed to another team. During this read-only review another actor advanced
client HEAD to 15ec0a90e41fcbf303e5dd022965c4196672111e; all four compared
INTEROP document hashes remained identical. No client implementation result is
inferred from that new commit. Relevant unchanged document hashes:
DESIGN 8e551b8f54ca4701df385dd9bf9cd2ac2fcd9504fa996ad4faf3c51e273aef72;
REQUIREMENTS f96214769c2b8f6c7a1e3109dd90aef00ac62289a60ee02303d05fe353148058;
FAILURES 2abd4b2ee2c68893521acee7da59d15cfd2120a56f856e0da5fefeeba5046269;
STAGE-PLAN 6703a356d87291e69289816a515055b7f379d2f7ca5fe950b03613d44e028615.

## Normative obligations -> proposed owners and tests

| Authority | Required observation guarantee | Proposed owner / test-plan rows |
|---|---|---|
| I, O1; SPEC §§5,9 | Projection identity/revisions independent of publication ordinal, max UUID and submissions | Projection journal/evaluator; O01/O02 |
| O1 | Full public replacements with value availability, evaluation, completeness, schema/model and permitted evidence reference | Projection acceptance + public renderer; O02/O12/O14 |
| O2 | Consistent authorized snapshot + complete retained successor tail at a certified cut | Journal bootstrap planner; O03–O05 |
| O2/O4 | Barrier proves delivery coverage through a named authorized-view boundary, not future freshness | Session/certificate authority; O03/O05/O12/O13 |
| O3 | Verified resume or explicit cannot-resume + certified replacement; no ordinary update repairs a gap | Session + retention/cursor authority; O06–O11 |
| O4; SPEC §4.3 | Backend completeness, evaluation validity, availability, delivery continuity and as-of freshness stay independent | Evaluator evidence + session; O07/O12–O14 |
| A; SPEC §6 | Policy covers existence, payload, metadata, diagnostics, replay, retained reads and cursors | Policy renderer/disclosure authority; A01/A02/A06 |
| A/O3 | Race-safe release ordered against revocation; old sessions fenced; no recall claim | Disclosure authority + session owner; A03–A05/O15 |
| V | Trusted compatible descriptor before use; incompatible change suspends interpretation | Configured descriptor authority; V01/V02 |
| B01/O2/O15 | Bounded buffers/work/retention/sessions, explicit overflow, owned cancellation and recovery | Journal/session owners; limits matrix |

All tests in OBSERVATION-TEST-PLAN.md are proposed, not executed evidence.

## Existing capability coverage — not an observation stream

- core.Result separates Working availability, Evaluation, Completeness, RuleVersion,
  Frontier and diagnostics. Its Frontier flags are supplied by a trusted host;
  they are not authenticated certificates. CORE does no persistence/disclosure.
- rev.Reconstruct and Reconcile compute current assertions and publication
  intentions. Assertion status belongs to producer/key, NOT an arbitrary projection.
- ledger.Accept atomically persists receipt/checkpoint/head/publication/pending;
  its ordinal belongs to an assertion key/generation. Heads is a private owner API.
- delivery.Store persists immutable recipient admissions and Run persists matched
  confirmations. Neither evaluates a domain projection nor allocates its revisions.
  Records/Reports contain protected source metadata; do not expose them to observers.
- Delivery confirmation is recipient admission, not projection processing, snapshot
  readiness, cursor continuity, observation application, authorization or freshness.

There is currently NO durable projection journal, authorized-view stream, snapshot/
tail certificate issuer, observer cursor authority, replay-retention protocol or
revocation-safe public release path. Reuse private typed values and SQLite experience,
not publication identity as observation identity or source logs as public feeds.

## TypeScript comparison and review gates

The proposals align on full snapshots, explicit status axes, arbitrary certified
initial snapshot revision, barriers, gap/reset rules and session fencing. One sketch
has mandatory assertionStatus on every observation; profile O1 explicitly says whole-
projection evaluation is not single-key current/retracted status. Do not copy that
field generically. Assertion statuses appear only in declared projection content.
The ratified profile governs; coordinate this correction before freezing adapter types.

No normative contradiction found. Deployment/design decisions still needed:
projection schemas/semantic equality/material public evidence; scope and consistency
jurisdiction of evaluator cuts; principal/view identity and permitted provenance;
retention/capacity/certificate key lifecycle; exact irreversible disclosure boundary;
unknown-release audit rules; policy revision handling and session generation recovery.
Cross-owner atomic projection cuts require additional contracts: minimal stage uses
one authoritative finite input/evaluation cut, not an accidental merge of live reads.
No wire encoding, transport or production credential mechanism selected.
