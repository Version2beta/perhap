# Publication Stage 2 verification before coder loop

Race repair isolated first at 30a3b3733e7e9716791d96816a0aa292c0c80a92.
Reproduced before editing with -race -count=100 -failfast:
TestService_GetOrCreateSupervisorReplaysHistoryForRecreatedSupervisor expected
reducedThrough 01f1e6f6-f867-6054-ac71-020000000000 but got
01f1e6f6-f867-6fe6-ac71-020000000000. The generated first ID was lexically later;
this was a flawed test assumption, not a race detector data-race report.
Reviewed prior correction 0ee509c33009b2f2dc638945a25c18475bac0c7a. Added only
explicit ordered UUIDv6 fixtures to this recreation test; both model counts and
exact reducedThrough assertion preserved. No production UUID/replay change.
Targeted -race -count=100 passed after fix; full Go, full race and vet passed.

Pins: draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
profile 9a80977efaf2ff740a257f6e30314dac42ec384e, locally ratified
INTEROP-PUBOBS/0.1 v0.1.0. All 19 checksums pass, manifest digest
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
No upstream changes. Binding stage scope and mapping: DELIVERY-SPEC.md.
Existing Stage 1 copied profile/vectors are reused, not modified. No dependency
changes: same pinned mattn SQLite driver and CGO environment as Stage 1.

Independent reference written privately outside checkout; temporarily installed
only in sdk/go/delivery/delivery.go, never shipped. Actual checks:
- go test -count=1 -v ./sdk/go/delivery: four substantive tests plus helper pass.
  Six child-process os.Exit(91) cuts: receiver-before_commit/after_commit,
  sender-before_send/after_send, ack-before_commit/after_commit. Real SQLite files
  reopened after every cut. Sender cases use the real Stage 1 source ledger;
  pending identity/content remains unchanged. Receiver duplicates admit one fact.
- scripts/loop.sh --check with reference: all eleven tasks pass, including prior
  CORE/REV/ledger, strict expectations for all six crash subcases, full Go/race/
  vet, complexity and frontend checks. Exit 0. Actual evidence is this local
  SQLite binding, not production authorization or a general transport guarantee.
- Restored zero-return stubs, formatted and repeated exact judge: all four new
  tasks D2STORE/D2RUN/D2CRASH/D2ALL fail with test command exit 1; prior seven pass.
- cmp scripts/loop.sh against installed v1.1.1 script: unchanged.
- git diff --check clean. No Stage 1/CORE/REV implementation/test changes.

Crash tests are process termination at controlled durability boundaries, not power
loss/storage-media fault evidence. Fake sink wraps a real SQLite store to lose an
ack; it is not the sole durability test. Trusted receiver/producer identity and
private handle ownership are explicit test/deployment assumptions; no runtime
AUTH/disclosure service, projection application, NATS or effects are implemented.
Run is one bounded single-worker pass, with context-honoring dependencies; lifecycle
and any scheduling/backoff between passes belong to caller. No unbounded worker.

After verified stub baseline commit and clean tree, authorized loop command:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Pinned skill commands unchanged, new D2 task IDs avoid old attempt counters.
