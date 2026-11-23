# Observation gateway — proposed deterministic acceptance tests

No tests were implemented/run for these documentation changes. Shared-vector
expectations are ratified profile oracles, not current Go capabilities. Pin and
scope: OBSERVATION-REQUIREMENTS.md. Architecture: OBSERVATION-DESIGN.md.
Use typed semantic records, not case-ID branches or invented transport frames.

## Cheapest experiment that can invalidate the design

After separate implementation authorization, build only a tiny real transactional
projection journal and deterministic authority/recipient harness. No network/UI.
Seed authorized snapshot S/40; pause immediately after the consistent snapshot/tail
plan is recorded. Commit S/41 before any reader wakeup/notification is installed.
Read the durable tail, deliver snapshot40 then41, certify barrier41. Repeat with
retention-capacity pressure and a crash between those cut points. If 41 can vanish
or a partial scan still gets a barrier, the chosen persistence/handoff is invalid.

At the same paused plan, put revocation before release and then invert that order.
The first schedule must disclose ZERO payload/cursor/status metadata; the second
must identify a permitted prior release and reject every later one. A lock that
only guards plan creation fails this experiment. These few schedules should precede
a general gateway, durable cursor service or background fanout infrastructure.
A trusted harness proves local ordering only, not real credential/disclosure policy.

## Shared vector mapping — all observation/security/version cases

| Vector | Controlled schedule / positive and forbidden-effect assertions |
|---|---|
| O01 | Publish K once: Q unchanged, Q2 changes; then several keys yield one Q result. Only changed authorized public records allocate their own revisions. Never map publication ordinal to projection ordinal |
| O02 | Identical value/frontier becomes failed then recovers. Two durable full status revisions, unchanged value; no suppression of material status and no fake new domain events |
| O03 | Pause after snapshot40 cut; commit41 before tail read. Exact logical snapshot40, revision41, barrier41; duplicate retry allowed, no lost41 or regression |
| O04 | Selected snapshot-first contract: reject revision-before-snapshot before apply; page/buffer overflow gives unavailable/resync-required and NO barrier. Future reordered binding must test its declared B/B+1 buffer policy separately |
| O05 | Fail snapshot decode/storage read before certificate. Hydration unverified, no empty success, no unauthorized stale fallback; pin and iterator cleaned up |
| O06 | Old session a at43, new b replays44/45+barrier. Delay a callback until b active; fence a before apply/disclosure, verified-through45 only for b |
| O07 | Disconnect with no demonstrated missing successor. Delivery unknown, preserve permitted backend as-of completeness, no fabricated incomplete history or live-authority claim |
| O08 | Omit44, deliver45 and46. Gap remains, verified cursor stays43; only validated repair or certified replacement clears it |
| O09 | Expire retained cursor, then offer certified snapshot90 and barrier. Explicit cannot-resume; current observation repaired, not historical/submission knowledge |
| O10 | Same numeric revision, wrong principal/view/scope/generation/schema/model. Reject before apply; no cross-view cursor reuse. Valid new-generation replacement requires certificate |
| O11 | Replay identical revision then same identity with different content/status. Retry has no new logical application; conflict suspends affected stream, no last-writer-wins |
| O12 | Contiguous authorized delivery but known required backend gap. Delivery verified-through while backend incomplete; no valid-looking payload upgrade |
| O13 | Backend complete at F; delivery uncertain; quiet connection. No new barrier/freshness from silence; no timer required for as-of-only contract |
| O14 | Available null, unavailable projection and assertion retract as declared content. Distinct results, no universal clear/undo; no fabricated available zero state |
| O15 | Close twice, fail recipient callback, fail cleanup, delay old completion. Idempotent owned shutdown, no orphaned workers; preserve primary error plus protected cleanup context |
| A01 | Hidden and nonexistent scope queried via bootstrap/resume/repair/errors. Deployment-approved equivalent denial without key existence, head, ordinal, count or internal reason leak |
| A02 | Insert hidden source events between equal authorized public records. No visible revision increments or raw offset gaps; cursor/certificate does not expose hidden counts; private checkpoint may advance |
| A03 | Pause at irreversible-release fence; revoke-first versus release-first. First blocks payload AND metadata; second is prior authorized disclosure, no recall claim; subsequent release blocked |
| A04 | Revoke during hydrating snapshot/page already internally queued. Fence old session and clear owned protected buffers; no release of cursor/barrier either; reauthorize through new bootstrap |
| A05 | Disconnect without offline permission, then principal switch. Server refuses later old-principal release/cursor; client harness hides/clears cache and never revives it without fresh authorization. Application references are not remotely erased |
| A06 | Authority allows outcome but not sources. Public renderer releases only declared result/provenance; raw source IDs, core errors and private diagnostics absent from payload, logs, cursor and failure output |
| V01 | Inject unknown required profile/feature/schema/model. Explicit incompatibility before reading/disclosing data; no guessed version compatibility. Unknown optional item ignored only if declared harmless |
| V02 | Incompatible descriptor change after snapshot cut. Suspend old interpretation/session, renegotiate and certified rebootstrap; retained historical records unchanged |
| B01 | Exhaust journal acceptance, session count, pin/page bytes, buffer capacity or ordinal range. Explicit capacity/unavailable/resync; no counter wrap, silent drop or barrier beyond retained coverage |

A04/A05/O06/O15 client fencing assertions need either an independent recipient harness
or later tests with perhap-client Stage 2A. Server tests alone cannot prove the real
client erased references or fenced application callbacks. Vectors A01/A03/A06 require
actual deployment enforcement before production security claims; a fake authority is
not that evidence. S submission, P ledger/delivery and EFFECT remain separate roles.

## Deterministic storage/crash matrix

Use explicit channels/latches and cut hooks, not sleeps. Before-durability cuts must
be demonstrably before commit; otherwise report unknown and inspect all-or-none
recovered state rather than assert rollback. Subprocess termination with real store
reopen is needed for durability; an in-memory fake supplies only model evidence.

| Cut/failure | Required recovery result |
|---|---|
| Before/inside projection acceptance commit | receipt/checkpoint/head/public revision/replay obligation all absent or all present; no half head |
| After durable commit before acceptance reply | identical evaluation request returns original receipt/revision; no duplicate public record |
| No-op after hidden/internal input change, then restart | private checkpoint advanced, immutable public record and revision unchanged |
| Old evaluation races newer no-op/failed evaluation | trusted input/rule/checkpoint preconditions reject stale overwrite; acceptance order explicit |
| Crash after snapshot cut before any frame release | no observation certified/applied; restart retires session; retained history supports authorized resume or resnapshot |
| Crash after snapshot release before tail/barrier | client remains unverified; restart handles repeated immutable frames, never assumes delivered barrier |
| Commit41 during bootstrap / retention floor advance | protected tail includes41 or operation explicitly fails; cannot report successful empty tail |
| Crash before/after cursor certificate issuance | valid durable authority key/generation permits only valid retained resume; unknown token cannot bypass authorization or retention |
| Crash during possible irreversible output | protected attempt order preserved; classify release uncertain if necessary, never claim rollback/recall of possible disclosure |
| Revocation committed before service restart | old policy/session/cursor cannot revive permission; authoritative policy rechecked before any record release |
| Journal corruption, missing revision, failed scan | refuse authoritative continuity; no empty-new-generation fallback |
| Restored stale backup with reused generation | independent recovery authority rejects identity/ordinal reuse; new generation needs authorized certified rebootstrap |

Use complete logical transition sequences, immutable public records and negative
observations, not only final state or successful return. Include enumeration of
all permitted small schedules around snapshot/read/commit, revoke/release and
close/reconnect; derive expectations from the profile, not from the implementation.
Inspect raw outputs for protected metadata in negative cases. Timing/traffic-analysis
privacy beyond the declared profile needs separate deployment requirements.

## Acceptance levels and tooling for later stages

- Pure oracle tests: public equality/rendering, identity/successor, axis distinctions.
- Local real storage tests: journal reopen, projection checkpoint, snapshot/tail cut
  and retention protection. Preserve completed CORE/REV/ledger/delivery tests.
- Trusted injected authority tests: source-cut validation, permission-view isolation,
  release/revoke total order inside one declared jurisdiction, descriptor binding.
- Production authority/binding integration: real credential mapping, policy update,
  queued output/replay/cache release boundary, certificate authenticity/key rotation,
  deployed filesystem and restart behavior. No earlier layer proves this one.
- Optional TypeScript integration: feed the same abstract sequences through its
  adapter with approved framing/schema mapping; no new wire format is chosen here.

Future acceptance commands after authorization: focused deterministic schedules,
repeated real subprocess tests, full Go suite, full race suite, vet, prior judge
checks, diff/artifact/source-pin inspection. Any later spec-loop kit needs failing
stubs and independent reference under the unchanged exact judge, not just mocks.
No such kit or tests are constructed in this planning task.
