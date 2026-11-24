# Stage 3 — actual Go / TypeScript integration: approval proposal

Design only. No product changes, APIs frozen, kit, model calls, commits or new
interoperability results. spec-loop v1.1.1 approval gate applies before implementation.

## 1. Verified local baselines and authority

Go clean rddd-foundation f13af35e29f669c3247c9980fc13ae1b1e5222ad;
submission implementation 3fa5769aeef9e0add7bda9a84f6d7eeab9c29484 and observation
6b211fa3d41417135d8478d55a3b4e75df40ca84 are ancestors.
Client clean rddd-foundation 81641540cb8e79d002cf9599ff33118b44e12df0;
implementation 5a37ba457c26785e0e429a469c20f9a02e289881 is an ancestor.
Inspected local source, not GitHub. No unpushed work replaced.

Spec clean at 9a80977efaf2ff740a257f6e30314dac42ec384e; frozen draft.6
1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; all19 checksums pass and SHA256SUMS
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f matches.
I/S/O/A/V/B govern; neither existing implementation is normative. Candidate portable
JSON is not silently ratified and Go JSON encoding is not claimed to be JCS.

Read Go submission contracts/implementation/results, observation recovery/proof/release
implementation and GO-GATEWAY-CONTRACT; client submission/session/observation types
and owned session implementation. Previous 11 complete local semantic row judgments
are baseline evidence, not cross-process results. No new tests executed in this pass.

## 2. Smallest actual boundary

One Node test coordinator runs the actual TypeScript session/submissions owners and
launches a Go integration-host executable using the actual SDK packages. The host opens
real private submission and viewjournal SQLite files and registered trusted authorities.
The adapter replaces ONLY Backend.open and submission backend.send/lookup plus the
required trust/equality/bounded host seams. Preserve Stage2A/2B pure core and APIs.

The Go host is test/integration composition, not a fake fact store. It may provide
registered codecs, issuer/verifier and explicit fixture authority because these are
currently injected Go responsibilities. Every reported durable fact comes from
Client.Submit/Resolve and Owner.Evaluate; every observation record comes from the
actual journal and Recovery. No fabricated successful responses to satisfy the client.

Submission admission is authoritative only for gateway-originated events, not legacy
JetStream. Existing publication/delivery stores remain separate. There is no automatic
projector: test control may call Owner.Evaluate and explicitly registered
viewjournal.Accept with a trusted fixture cut/value. Report this as explicit host
publication/projection seeding, never as automatic submission inclusion. Owner.Evaluate
returns no working projection result or admitted-history export; do not invent one.
A real automatic projector would require a separate source/result capability review.

## 3. Concrete missing Go capabilities — separate approval required

### A. Optional renewed resume handshake

Client session.ts requires Frame(kind=resume, certificate) as the FIRST frame when a
cursor was carried. Go Recovery.Resume validates it and returns a session, but Step
emits revisions/barrier directly. Relabeling a later update, replaying an old barrier
as new authorization, or emitting verified:true would violate the client contract.

Propose an additive, explicit resume-handshake delivery operation on a newly resumed
Recovered session (name/signature not frozen). Before Step, it certifies purpose
resume over the ACTUAL validated durable anchor/current Binding and synchronously
releases a RecoveryFrame through the SAME Gate mutex/current Check and recipient.
It must be once-only, bounded, cancellation/cleanup-owned, and unavailable for a
snapshot session or after tail delivery starts. Keep existing Resume/Step framing
unchanged for existing callers; the integration host explicitly requests this extra
operation. New purpose/frame recognition must be reviewed/tested, not a fake barrier
passed through Gate.Release merely to acquire its lock. The old barrier token remains
an opaque retained position; the handshake establishes renewed session binding, not
client application or perpetual freshness.

### B. Read-only semantic-matching resolution

TS backend.lookup receives the retained full Event, but Go Client.Resolve receives
only ID. Facts contain EventID/attribution, not the stored semantic envelope/schema.
A bridge cannot truthfully sign the request's semantic digest as the stored meaning
merely because ID-only Resolve succeeded. Submit-as-lookup is NOT acceptable: it can
create a receipt/admission and defeats S05. Private-file scraping/new shadow identity
storage is not an integration seam.

Propose additive Client.ResolveMatching(ctx, Input, Recipient), descriptive name only:
renew lookup authority and trusted attribution, normalize using the registered codec,
compare all immutable semantics against the authoritative retained event under the
owner, return existing conflict on mismatch or complete facts on match. Absence stays
no-retained-knowledge and creates no receipt, decision or evaluation. Fence actual reply
as today. Keep ID-only Resolve unchanged for existing authorized uses. This supplies
the evidence needed for a signed semantic-bound lookup response without exposing the
stored payload to a caller who only has status permission.

No API implementation is authorized by this document itself. If these changes are
not approved, stop resume/full semantic-verified lookup integration rather than weaken
the TS trust contract. Basic bootstrap/ID-only resolution could be tested separately
but would not constitute the requested complete adapter.

## 4. One transport: private Unix-domain stream socket

Use Go net.UnixConn and Node net.Socket; no framework, HTTP, WebSocket, broker or
new production listener. Parent owns a private temporary directory, socket permissions,
child process and SQLite/key paths. Go binds the supplied path; parent waits for a
bounded readiness event, never sleep-based startup. Linux/macOS scope; Windows is
explicitly unsupported by this chosen integration transport.

One connection is pinned to one host-configured test principal/context jurisdiction.
Identity is established by trusted launcher/channel configuration and OS-private IPC,
NOT actor/verified/admitted fields in request JSON. Role-specific endpoints/connections
may exercise Alice/Bob; this is representative test authentication, not protection
against another same-user process or production credentials. Caller-selected scope
still needs independent Go policy checks. Never expose the administrative control
endpoint/capabilities on the client channel.

Framing: uint32 big-endian byte length followed by strict UTF-8 JSON; proposed cap
64KiB per frame, bounded strings/token size/depth and at most16 in-flight operations
per peer. Semantic large integers NEVER use JSON numbers. Reject zero/oversized lengths,
partial EOF, duplicate/unknown fields, invalid Unicode, bad enums and incompatible
binding versions before data interpretation. No repair of malformed frames.

Binding-specific messages (not an upstream standard): hello/descriptor, submit,
resolve-matching, observation open/resume/replace/next/close, cancel and operation
completion. Distinct correlation IDs identify connection attempts/operations only.
IDs are not reused during a connection; reconnection uses a new connection epoch.
No automatic retries. Transport completion is not a durable receipt/admission.

Observation is pull/credit bounded: one next outstanding per iterator, one Go Step
per selected finite page. A Step can emit snapshot + page + barrier, so its receiver
has a declared bounded page queue; no unbounded socket-to-observer fanout. Do not send
iterator EOF after every barrier: keep the iterator open until cancellation/failure;
TS treats actual EOF as disconnect and hides data.

### Actual irreversible release and revocation

The Go Recipient performs bounded serialization and actual socket writes synchronously
INSIDE Recovery's Gate or submission Host.Release fence. No internal enqueue followed
by an unlocked writer goroutine. The declared release is bytes irreversibly accepted
by the OS socket for the authorized connected peer, not TS state application. Kernel
buffers/peer are explicitly within the approved integration jurisdiction. A later
production binding must independently justify its recipient/routing boundary.

A bounded context-aware writer permit prevents interleaved frames. Acquire it within
the release callback, with the operation deadline. Use socket write deadlines and an
owned cancellation interrupt; join/stop cancellation hooks before resetting deadlines
and returning the permit, so an old cancellation cannot poison the next write. Never
spawn an unobserved timeout goroutine. Full/partial write failure is uncertain output.
Revocation waits only within configured cooperative write/check budgets; frames already
released can arrive later and cannot be recalled. Revoke-first releases zero payload,
proof or protected metadata. Release-first is an attributable prior disclosure.

Reader/control handling must remain responsive while a writer blocks: bounded worker
pool, separate cancellation/control admission budget, no holding global reader mutex
through a write or SQLite operation. Resource exhaustion refuses operations; it does
not advance cursors or fabricate empty successful hydration.

## 5. Exact mapping to the existing TypeScript APIs

### Descriptor, interpretation and trust

Configure a binding identifier/version and trusted descriptor containing exact pins,
application/authority, approved manifest/model/schema/equality, required features,
retention and resource limits. Map profile/version/baseline directly. Descriptor.identity
is a reviewed deterministic encoding of complete Go scope including observer/view,
generation and authorization epoch, not just projection name. maxRecords is a bounded
safe JS integer. Unknown required feature/model/version fails before apply. A changed
incompatible descriptor retires interpretation and requires new bootstrap.

Register one integration-only versioned counter codec in both languages with documented
immutable event schema. It accepts canonical integer-string content throughout int64,
with checked arithmetic; it is not the previous test-only scalar-number codec silently
changed under counter/v1. Build TS semantic deterministically BEFORE track() from the
full normalized event (context/entity/type/content/cause/correlation/schema). Go's codec
independently validates/normalizes the same meaning before compare/evaluate. Missing,
null and empty optional values are not conflated. Canonical lowercase UUIDv6 is gateway
identity; validate it, never infer causality from timestamp. Do not trust semantic as a
hash or producer claim. attributionBinding resolves through host-controlled delegation.

Use an integration-host ProofAuthority with real Ed25519 signing/verification and a
strict binding-specific public claims encoding. Private issuer key is Go-only runtime
state; public verification key/issuer identity is pinned by trusted launcher config,
not accepted from each frame. Persist the fixture key outside the reopened databases
when restart-valid cursors are tested. Separate renewal policy/revocation/window state
from key authenticity; signature alone never grants current permission.

Sign exact bounded body bytes and verify them before parsing/applying claims; avoid
cross-language reserialization ambiguity. Claims bind full descriptor/scope, purpose,
immutable record and validity jurisdiction. Body uses decimal strings for large scalars;
no private offsets/plan IDs/source diagnostics. No production key custody/rotation or
security claim. Raw Go proof strings are arbitrary bytes: transport uses canonical
base64url, decoded losslessly before Go Verify. Consumers treat cursor as opaque.

### Observation mapping

| Go / binding output | TS input / required validation |
|---|---|
| Record.View + current binding | Record.identity equals trusted Descriptor.identity, never arbitrary caller identity |
| Revision / Previous uint64 | canonical decimal strings -> checked bigint; Previous0 becomes null only for ordinal1, retain all other predecessor values |
| Public.Available=false | value {kind:unavailable}; nonnil value is invalid |
| Available=true, Value=nil | value {kind:available,value:null} |
| Available=true, Value.Total int64 | value {kind:available,value:{total:bigint}}; equality compares declared values, not JSON.stringify(bigint) |
| Evaluation / Completeness / Evidence | exact named TS axes/ref; no assertionStatus or completeness upgrade |
| snapshot + handoff proof | snapshot(record, certificate); trust.snapshot verifies exact binding/purpose/record |
| new optional resume handshake | resume(certificate) FIRST, trust.resume verifies carried record/cursor and renewed scope; no fabricated update |
| revision | update(record), preserving identity/predecessor; connection/local operation fence checked before dispatch |
| barrier + purpose-bound proof | barrier(through,cursor,certificate); trust.barrier verifies exact reached record/boundary; cursor lossless/opaque |
| cannot-resume | cannot-resume; explicit subsequent Go Replace required, never from corruption/unavailable |
| purpose replacement snapshot | snapshot with replacement certificate, accepted only in declared rebootstrap lifecycle |
| process EOF/disconnect | iterator completion/error -> existing unknown continuity, no backend-completeness downgrade |
| revoke / close | abort/fence pending delivery and clear TS owned state via existing lifecycle, observe iterator.return completion |

Backend.open synchronously returns an owned iterator; it can perform IPC open on first
next(). return() sends close/cancel and awaits bounded completion with an independent
cleanup budget. Session.reconnect joins old iterator before new activation. Go Session
fence and local Node connection/operation generations both matter; a server ID alone
cannot authorize a stale callback. Trust callbacks are synchronous: verify locally with
the pinned key, never launch hidden async verification RPCs.

After cannot-resume, the iterator may perform ONE explicitly declared replace request
under approved resnapshot policy and only then yield the certified replacement. This
is an explicit adapter policy decision, not Go silently clearing a gap. Invalid source
integrity/unavailable or descriptor incompatibility must not become healthy replacement.

### Submission mapping

| Existing TS seam | Actual Go/binding mapping |
|---|---|
| backend.send(Event, attempt, signal) | parse approved semantic envelope -> Input; Client.Submit; attempt is request correlation only |
| backend.lookup(Event, attempt, signal) | proposed read-only ResolveMatching, not Submit and not an echoed semantic assertion |
| receipt | Receipt true -> durable; never infer it from transport completion |
| admission/evaluation/model/evidence/attribution | direct complete Facts mapping, validated enums and scoped authenticated attribution |
| generation / version | generation string / canonical uint64 decimal -> bigint; never Number or projection ordinal |
| verified(event,reply) | locally verify signed reply/descriptor/context/event binding and trusted attribution before returning true |
| equalFacts | complete field equality; same version with changed facts conflicts |
| ErrConflict | identity-conflict, never replacement admission |
| ErrBlocked / ErrDenied | blocked / denied, do not collapse missing authority into definitive policy denial |
| capacity / unavailable / integrity failure | unavailable with stable approved error category; integrity suspends connection; never no-retained-knowledge |
| ErrUnknown / ErrRelease / cancellation / response loss | reject/abort owned wait -> unknown; retain identity/facts and explicitly resolve same ID |
| invalid input / incompatible schema | binding/protocol failure before facts; never fabricate rejected admission |
| no-retained-knowledge | direct outcome only at authorized lookup cut; not safe fresh identity or proof of noncommit |
| expired | unsupported, never synthesized; S06 remains unexecuted |

A signed success reply binds the canonical semantic expectation established by Submit
or successful ResolveMatching. Do not sign an echoed client field as authoritative.
If callback already delivered facts but later operation completion is unknown, preserve
uncertainty rather than replacing it with a second fabricated result. The adapter owns
exactly one completion per request and suppresses late responses after cancellation.

## 6. Executable verification design (all proposed, none executed here)

Real Node process + compiled Go child + real SQLite files for every end-to-end case.
Retain command/source SHAs, binary hash, Node/Go versions, configuration, full transcripts
with redaction and actual process exit/cleanup results. Baseline backend-fake tests stay
as regressions, not evidence for this stage. Existing 11 row judgments are not carried
forward automatically as new cross-process passes.

| Case | Controlled executable scenario / required assertion | PUBOBS full-row target / qualification |
|---|---|---|
| I1 | Normalize/build canonical E, send -> durable/admitted/not-established, explicit Owner.Evaluate -> new facts; resolve via semantic-match | S01/S02/S03 combined where full schedules below exercised; I identity separation |
| I2 | Drop connection after actual admission COMMIT before client reply; SIGKILL child; reopen same files/key; identical retry/lookup, one event/no duplicate version | S01 complete target; P14a distinction exercised, not a P-ledger full-row claim |
| I3 | Existing ID with changed payload/cause/context/attribution; lookup mismatch too; no original overwrite | S02 complete target with approved lookup dependency |
| I4 | Actual v1 pending, explicit admission v2 and failed evaluation v3; authorized Node dispatcher delivers3/1/2 into real TS owner | S03 complete target; no fake backend fact generation |
| I5 | Tamper one signed status fact/version and replay; TS rejects unverified/conflicting facts without mutation; real Go invariant refuses corrupted recovery | S04 partial adversarial evidence unless the full same-resource/version conflict oracle is actually exercised; no forced pass |
| I6 | Two TS owners of E; hold first real request before store transaction, second lookup returns no knowledge, then release first -> admission; no new ID/rejection | S05 complete target, declared request-start/durability cut |
| I7 | Missing trusted authority then definitive deny; forged identity/binding fields do not change host actor; hidden/nonexistent scopes same public failure | S07/A01 complete targets only for full endpoint/negative metadata assertions |
| I8 | Cancel wait after durable admission; deliver late response to retired request; Resolve same E recovers; separately advance Q42 while another E stays unknown | S08 complete target; projection update is explicit host fixture, not automatic pipeline |
| I9 | Seed actual S40, commit41 between prepare and Step; real TS snapshot40/update41/barrier41 with verified proof | O03 complete target; O04 snapshot-first branch separately assert before-apply rejection under controlled dispatch reorder |
| I10 | Disconnect at43; new real connection/request uses resume handshake, receives44/45/barrier; late old frame fenced; repeat after child termination/reopen | O06 complete target plus durable restart; O07 continuity unknown/as-of axes assertions |
| I11 | Advance observation proof replay floor / expire trusted observation certificate, cannot-resume, explicit Replace at90; certified current continuity only | O09 complete target; NOT receipt expiry S06; physical pruning remains unsupported |
| I12 | Observer/view/schema/model/generation/cursor substitution, tampered proof, incompatible descriptor | O10/V01/V02 complete targets if every required branch actually covered; no permissive reinterpretation |
| I13 | Suppress actual received44 at controlled authorized Node dispatch layer, deliver45 then46; no cursor advance; duplicate actual44 is idempotent | O08 complete target; O11 identical replay partial, conflicting-record branch remains separate/unexecuted unless fully exercised |
| I14 | Real Go incomplete/failed/unavailable/null public records seeded explicitly; verify separate TS axes, no live freshness from quiet socket | O02/O05/O12/O13 targets where full oracle covered; O14 partial unless actual declared retraction semantics included |
| I15 | Revoke after plan but before release vs first bytes released before revoke; block writer via latches, then resume; no later writes/callbacks; clear client cache, identity switch | A03/A04/A05/O15 complete targets; distinguish kernel-buffer prior disclosure from post-revoke new release |
| I16 | Framing split/truncation/oversize/duplicate fields, unsafe numeric JSON, signed/unsigned extrema; full writer/queue/session/process exhaustion; cancellation/disposal joins | V01/O04/O15/B01 applicable partials; full B01 requires all backlog/buffer/ordinal subcases, not just frame cap |
| I17 | Fail before transaction vs unknown after commit vs actual corrupted SQLite reopen; no fake empty knowledge/recovery | P14a/b/c distinctions for S; do not credit complete P rows solely from submission analogues |

Process/control fault hooks are integration-test-only and use deterministic barriers,
not sleeps. Control may pause before release, drop a response after commit or terminate
the Go child. Do not acknowledge a queue as actual client disclosure or alter authoritative
records merely to produce desired client states. Controlled peer-side reorder/drop acts
AFTER the authorized IPC release and remains bounded/fenced; distinguish it from Go
incorrectly claiming it released missing bytes. Assert both client state and recovered
Go event/version/pin state. Check child exit and sockets/files/iterators cleaned up.

### All55 allocation before execution

S01–S05/S07/S08 are full end-to-end targets where listed oracles can be completed;
S06 remains unsupported/unexecuted. P01–P18 (including P14a/b/c) and R01–R03 remain
prior-role regressions/unexecuted as new full integration rows: this stage adds no
publication bridge or historical event consumer. O01 remains unexecuted without a
real publication-to-projection mapping; explicit seeding does not satisfy it.
O02–O10/O12/O13/O15 are targets with stated branch gates; O11/O14 remain partial until
conflicting-record/retraction oracles really run. A01/A03–A05 and V01/V02 are conditional
complete targets; A02/A06 remain partial until hidden-source/provenance/error coverage
is complete. B01 remains partial unless all row subcases execute. Final runner must
list all55 IDs as passed/failed/unexecuted with exact test/subcase anchors; no present
pass is asserted. Stronger production AUTH/cross-backend claims never follow.

Integer extrema require two distinct evidence levels: actual large int64 payloads/
projection values can traverse the real gateway, while near-uint64-max ordinals and
status versions are separate Go-encoder/Node-decoder conformance fixtures. Do not edit
journal counters to pretend an enormous committed history exists or credit ordinal
exhaustion from an echo fixture. Reject numeric JSON where this binding requires a
decimal string, out-of-range values, leading zeros and negative zero.

Normative traceability: profile I/§2 governs identity and separate orders; S/§3 governs
receipt/admission/evaluation, immutable retries and uncertain lookup; O1–O4/§5 govern
full records, snapshot/tail, verified resume/reset and as-of-only axes; A/§6 governs
actual release jurisdiction, identity isolation and cache retirement; V/§7 governs
trusted descriptors/compatibility. B01 governs resource refusal. Binding framing,
Unix socket, issuer encoding and test control are local choices, not upstream semantics.

## 7. Bounded implementation sequence after approval

1. Separate Go prerequisite approval/kit: optional resume-handshake delivery and
   read-only semantic matching. Independent reference, failing stubs, unchanged Go
   judge, all existing API/storage regressions. No changed framing for legacy callers.
2. Freeze one integration-only manifest/codec/frame schema, with positive/negative
   golden vectors decoded independently in Go and TS. Integration host supplies real
   authorities/ProofAuthority and private UDS composition, no backend fact fakes.
3. Go host kit: synchronous bounded write fence, process/connection ownership, explicit
   trusted control operations and real restart cuts. Build from approved local source;
   record Go source SHA/artifact hash, no GitHub substitute or committed binary.
4. TS adapter kit in perhap-client: existing session/submissions interfaces unchanged;
   bounded iterator/request owners, trusted local signature verification and strict
   mapping. Independent reference must pass new tests against actual Go child before
   paid loop; stubs fail. Runner remains unchanged in each repository.
5. Run joint scenarios plus each repo's full regression/check/judge, Go race/vet,
   repeated deterministic lifecycle/fault schedules, exact55-row report. Preserve
   unrelated changes; no concurrent writers; separate local commits/source manifests
   for the two repositories. Proposed paid cap12 iterations/3 attempts per approved
   kit; no model spending/commits under this design-only request.

Owner of processes is the Node coordinator: startup deadline, readiness validation,
close graceful protocol, bounded wait, SIGTERM then SIGKILL only for its own child,
observe exit/stdout/stderr and outstanding promises, remove only owned temp artifacts
once processes/connections stopped. Go owns DBs and namespace/session cleanup. Crash
recovery uses surviving fixture keys/policy/inventory and caller exclusion of old live
owners, not reconstruction of permission from a rolled-back DB.

## 8. Approval decisions

- Approve private UDS transport, local Linux/macOS integration scope and explicit
  test-only identity/Ed25519 issuer configuration, not production networking/AUTH?
- Separately approve the two additive Go prerequisites, with existing APIs unchanged:
  optional gated resume handshake and read-only semantic-matching resolution?
- Approve one integration-only integer-string schema/manifest, lossless framing,
  trusted-key distribution and explicit cannot-resume -> Replace adapter policy?
- Approve explicit administrative Evaluate/projection seed/revoke/fault control on a
  separate parent-only channel, with no automatic projector claim, and implementation
  split across independently verified Go and client kits after this review?

No normative spec conflict identified. These are actual capability gaps and local
binding decisions; do not bypass them with fake certificates, echoed semantic claims,
Submit-as-lookup, number rounding or asynchronous unguarded output. Stop here for approval.
