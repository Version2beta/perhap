# INTEROP-PUBOBS/0.1 — unexecuted conformance vectors

**Standing:** Locally ratified semantic expectations for [INTEROP-PUBOBS/0.1](publication-observation-0.1.md), version `0.1.0`. These are **unexecuted conformance vectors**, not executable tests, passing results or runtime-validated expectations. Symbols are abstract records, not an encoding. Each future adapter must run the same inputs and assert observable transitions, not branch on case IDs. Authorities are representative trusted test authorities, not client-supplied flags. No test may require paid calls, production access or private payload disclosure.

## Common setup and oracle

Registered assertion key K=(context A, producer P, subject X, meaning M), generation G. Another producer with the same subject/meaning is a different key. Projection Q depends on multiple keys plus ordinary events; another projection Q2 shares some inputs. Evidence authority serializes admitted evidence generations g1,g2,... and owns current rule identity. Observer U is allowed Q's public result, not its source payloads or hidden keys. S is Q's durable authorized-view generation. Ordinals denote arbitrary precision order, not a numerical wire type.

For every test assert positive results AND forbidden effects: no overwritten historical record, fabricated acknowledgment, unauthorized payload/metadata, duplicated logical admission, skipped revision, false completeness/freshness or orphaned work. A fixture runner should compare full public transition sequences and final durable records, including explicit value availability. Real crash tests use controlled process cut points and reopen the actual store; mock success is insufficient durability evidence.

| ID / requirement | Input or schedule | Required observable oracle |
|---|---|---|
| S01 / S,I | E/X durably admitted; reply lost; restart; retry E/X | One admitted fact, original attribution, retained status facts; unknown locally until resolution |
| S02 / S | Reuse E with changed payload, cause, context or asserting identity | Conflicting attempt; no overwrite or new fact; existing admission preserved |
| S03 / S | Receipt v1, admission v2, failed evaluation v3 delivered v3,v1,v2 | Newest full verified facts retained; evaluation failure does not erase admission |
| S04 / S | Same status resource/version with different facts | Protocol conflict, no last-writer-wins |
| S05 / S | Lookup empty at cut while earlier submit still in flight | No rejection/noncommit inference; identical-ID reconciliation only |
| S06 / S | Authorized caller retries after a known receipt expiry; then separately query an identity with no retained outcome knowledge | First lookup reports expired; second reports no-retained-knowledge. Neither proves rejection; neither permits fresh execution/admission under a recycled identity |
| S07 / S,A | First lookup lacks required authority evidence; second has a definitive policy denial | First is blocked/unresolved, second opaque denied under the declared public policy; neither discloses head/existence |
| S08 / S | Abort waits after admission; projection Q/42 arrives | No inferred cancellation, effect outcome or confirmation of E from Q/42 |
| P01 / P1,P2 | g1 current V then g2 identical V | K/1 only; g2 durable no-change receipt/checkpoint; immutable K/1 provenance unchanged |
| P02 / P1 | K/1 current V -> g2 failure -> g3 recovered V | K/2 unresolved stale V, K/3 current V; one sequence with predecessor chain |
| P03 / P3 | Never-published registered K, complete valid absence | Receipt/checkpoint; no publication, ordinal or invented retraction |
| P04 / P3,A | Trusted domain registration establishes K; assertion and disclosure authorities explicitly permit its existence and unresolved/current/retracted states; first evaluation fails, later recovers available null | K/1 unresolved unavailable, K/2 current available null |
| P05 / P3,A | Trusted domain registration establishes K and explicit assertion/disclosure authority permits its existence and unresolved/retracted states; publish K/1 initial unresolved, then complete valid absence | K/2 retracted; no invented prior supported value |
| P06 / P3,A | Failed computation/probe for unregistered forbidden key | No key registration, publication or existence disclosure |
| P07 / P2 | Newer g3 no-op accepted, old g2 valid under new request | g2 rejected stale despite unchanged assertion ordinal; no rollback |
| P08 / P2 | g3 unresolved accepted before g2 valid | Old result cannot clear unresolved or overwrite checkpoint |
| P09 / P2 | A and B share expected head/checkpoint; accept A first, then B without revalidation | B rejected stale; no checkpoint/publication change from B. If a later command is recomputed/revalidated against the new cut and preconditions, it is a separate eligible attempt, never a relabeled B |
| P10 / P2 | Schedule relevant admission before acceptance of old evaluation; separately schedule it after acceptance | Admission first: reject stale unless trusted revalidation explicitly includes it; acceptance first: record the prior cut, and require new evaluation for the later admission. Each subcase pins the authoritative order and revalidation capability |
| P11 / P2 | Retry original request after head advanced; then change its meaning | Original receipt returned for identical retry; changed command conflicts |
| P12 / P2 | Crash at controlled cut before commit becomes durable; separately crash after durable commit, before reply | Before durability: none of receipt/checkpoint/head/publication/pending changes survives. After durability: all survive. An implementation whose cut cannot establish the durability side must classify the caller outcome unknown and verify recovered all-or-none state, not assert rollback |
| P13 / P2,P4 | Durable commit then crash before reply/send | Receipt and one immutable publication recovered; original pending identity retained |
| P14a / P2 | Commit issued; response lost before caller can determine outcome; storage remains healthy | Return unknown; authorized lookup or identical retry resolves original outcome; no guessed rollback or new logical request |
| P14b / P2 | Store unavailable before transaction begins; no commit issued and no earlier in-flight request | Explicit unavailable, no durable-acceptance receipt or checkpoint advancement; do not classify this known pre-commit failure as ambiguous success |
| P14c / P2 | Reopen detects corrupted/truncated committed storage during recovery scan | Refuse authoritative ledger operation with storage-integrity/unavailable failure; no empty-ledger fallback or certified fresh head; earlier unresolved outcomes remain unknown |
| P15 / P2 | Second owner/stale owner attempts acceptance after restart | Durable exclusion/fencing rejects stale owner; ordinals never reused |
| P16 / P4 | Sink admits record; acknowledgment lost; sender and sink restart | Original ID retried; sink admits once; broker receipt never upgraded to recipient application |
| P17 / I,P1 | Same subject/meaning, two accountable producers | Independent keys and heads, no value collision or shared ordinal |
| P18 / I,P1 | Restore stale backup under old generation | Refuse continuity/identity reuse; require approved recovery, not silent counter reset |
| R01 / P1 | Historical b:set(2), c:add(5) caused by b, z:set(7) caused by c | Domain profile gives 7; b,c,z preserved at canonical positions; causes still b and c; separate current assertion=7 |
| R02 / P1 | Same b,c then z retracts assertion | Business state stays 7 for this domain; current assertion absent/retracted; no generic numerical undo |
| R03 / P1 | K/3 arrives before K/2; then conflicting K/2 | Affected assertion incomplete until verified repair; conflict quarantined; no latest-arrival choice |
| O01 / I,O1 | One publication leaves Q unchanged but changes Q2; several publications contribute to one Q change | Zero Q revision for first input; independent Q2 revision; no assertion-to-projection ordinal mapping |
| O02 / O1 | Same projection value/frontier, public evaluation becomes failed then recovers | Ordered full status revisions; equal value does not suppress meaningful public status |
| O03 / O2 | Snapshot S/40 chosen; S/41 commits before tail installed | Certified snapshot+tail covers 41 exactly once logically; barrier through 41; no regression to 40 |
| O04 / O2 | Binding declares out-of-order buffer capacity B; first deliver 41 before snapshot40 within B, then separately exceed B | Within B: buffer then apply40,41 and certify only after barrier. Beyond B: unavailable with explicit resync-required, no continuity certification. A snapshot-first binding instead rejects out-of-order frames as protocol violation before apply; it must declare that contract in advance |
| O05 / O2 | Snapshot scan/decode interrupts before barrier | Remain unverified; no successful empty bootstrap or authoritative stale fallback |
| O06 / O3 | Lose session a after 43; session b replays44,45+barrier; delayed a frame | Verified through45 only after chain/barrier; retired a frame fenced before state/disclosure |
| O07 / O3,O4 | Disconnect, no known missing revision | Delivery continuity unknown; backend completeness not fabricated incomplete; no ongoing live-freshness claim |
| O08 / O3 | Missing44 then receive45; ordinary46 arrives | Gap remains; no cursor advance over gap or automatic repair from newer value |
| O09 / O3 | Expired cursor -> cannot-resume -> certified replacement at90 | Current observation resumes after valid barrier; historical/submission gaps not claimed repaired |
| O10 / I,O3,V | Same ordinal from different scope/generation/schema/observer | Reject before apply; only certified compatible replacement permits generation change |
| O11 / O3 | Identical revision repeated, then same identity with changed status/payload | First repeat idempotent; second conflict suspends stream |
| O12 / O4 | Contiguous delivery with backend missing required cause | Delivery verified-through, backend incomplete, valid-looking payload not authoritative evidence |
| O13 / O4 | Backend complete at F, delivery uncertain, then quiet connection | Backend as-of completeness retained; continuity/freshness not extended by silence |
| O14 / O1 | Available null, unavailable projection, per-key retraction | Distinct values/states; retraction not an implicit projection clear or business undo |
| O15 / O3 | Close twice, callback fails, cleanup fails, old completion arrives | Idempotent ownership cleanup; primary failure preserved; completion observed/fenced; no orphaned work |
| A01 / A | Snapshot/repair/lookup requests for hidden key and nonexistent key | No forbidden existence, ordinal, counts, cursor or internal diagnostic disclosure; deployment-approved indistinguishable public failure |
| A02 / A,O3 | Hidden internal commits between two authorized-view updates | No raw hidden offset gaps/count leaks; certified continuity only for authorized view |
| A03 / A | Revocation ordered before queued release vs release ordered first | First blocks payload+metadata release; second records prior authorized disclosure, no recall claim; later release blocked |
| A04 / A | Revoked while hydrating with buffered snapshot/frame | Fence callbacks and clear owned protected data; no late local disclosure; reauthorize via fresh bootstrap |
| A05 / A | Offline without offline permission; switch principal | Hide cache on disconnect; isolate identities, clear owned data; no silent cache revival |
| A06 / A | Permitted derived outcome with protected source evidence | Disclose result only, not source payloads/identifiers through provenance or errors |
| V01 / V | Unknown required feature/profile/schema/model | Explicit incompatibility before apply; optional feature ignored only if explicitly harmless |
| V02 / V | Descriptor changes incompatibly mid-session | Suspend, renegotiate and certified rebootstrap; never reinterpret old history |
| B01 / P4,O2 | First exhaust acceptance backlog before commit; separately exhaust observation buffer or next ordinal range | Backlog: backpressure with no acceptance. Buffer: unavailable/resync-required per O04. Ordinal exhaustion: refuse advancing commit as unavailable, never wrap. Previously accepted durable work is retained in all subcases |

## Stage allocation and honest acceptance

Go ledger Stage 1: P01–P15, P17–P18 and B01's local persistence part; use R01–R02 as regression constraints, not instructions to add a domain runtime. Retain real reopen/crash evidence. P16 and any network disclosure wait for later stages. Real authority claims are not established by injected trusted seams.

TypeScript Stage 2A: O01–O15, A04–A05 and V01–V02 with an injected descriptor and deterministic backend; A01–A03/A06 assert the client's handling of promised evidence, not actual server enforcement. No new submission runtime is required; S08's projection-nonacknowledgment invariant can be checked without implementing submission. S01–S07 and full S08 belong to later submission Stage 2B.

Shared integration: P16, actual admission/evidence races, O03's real atomic cut, retention/repair, A01–A03/A06 at the real disclosure boundary and version binding. A fake cannot prove these. Exhaustive schedule permutations around snapshot/handoff, acceptance/admission and revocation/release are required before general concurrency claims. One real backend proves that binding only; a second independently implemented backend is needed for tested cross-backend interoperability.

These vectors have locally ratified semantic oracles but remain unexecuted, are not added to the baseline fixture suites, and are not represented as passing tests. An implementation test must pin the injected ordering, durability cut, declared binding policy and permissions for each subcase before judging it. Additional negative/property tests must avoid fixture-ID special cases and verify absence of unauthorized effects.
