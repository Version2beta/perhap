# Publication Stage 1 kit verification

Base Go HEAD 758322024fab69cea3fbae7dfd471fe78adbae7a, branch rddd-foundation.
Existing six uncommitted PUBLICATION documents preserved unchanged in baseline.
Baseline draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87; locally ratified
profile INTEROP-PUBOBS/0.1 v0.1.0 commit 9a80977efaf2ff740a257f6e30314dac42ec384e.
All 19 current manifest entries verified. SHA256SUMS digest:
01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
Profile/vectors copied unchanged to testdata; manifest retained separately.

Driver selection/API/ownership validation precedes coding loop: LEDGER-DESIGN.md
and binding LEDGER-SPEC.md. Pinned mattn/go-sqlite3 v1.14.32 added directly;
go.mod has one added requirement and go.sum exactly its two new lines.
Go 1.27.2 darwin/arm64, CGO_ENABLED=1, /usr/bin/cc. No other dependency changes.

Independent private reference kept outside checkout, temporarily copied only into
ledger.go. No reference shipped/committed. Actual checks:
- go test -count=1 -v ./sdk/go/ledger: all six substantive tests plus helper pass.
  Real child os.Exit(91) at before_tx, after_write, before_commit and after_commit;
  reopen proves none before commit, all after confirmed commit. Exit(42) from a
  competing subprocess proves owner exclusion. Reopen after owner exit succeeds.
- Lost after_commit response returns ErrUnknown; close/reopen and identical retry
  returns the committed publication identity/checkpoint. No in-memory durability
  substitute. Actual stale-backup file restored then rejected by independent
  host floor. Corrupt/empty DB and missing-parent storage refuse opening.
- scripts/loop.sh --check with reference: exit 0; T1/T2/R6VIEW/R6PLAN/R6FIX and
  LGPSTATE/LGPDURABLE/LGPALL all pass. Includes strict stdout expectations,
  format_check, current/earlier tests and go vet kit check. Final task includes
  full Go tests, full race tests, vet, seven complexity tests, frontend tests.
- Restored zero-return stubs; --check exit 1: all THREE new ledger tasks FAIL
  with behavioral test-command exit 1. Earlier five tasks pass.
- Unchanged judge compared byte-for-byte to installed spec-loop v1.1.1 script;
  diff whitespace check clean. Existing CORE/REV source and tests untouched.

Stage evidence is bounded: child process exits test crash recovery, not power-cut
or broken media/fsync. Source/admission/registration/disclosure/fence correctness
comes from the explicitly injected trusted host, not this test authority or a
caller flag. The ledger is private and local, not a public AUTH service or network
publisher. Capacity covers retained acceptance count, not performance scalability.
No full profile/interoperability/delivery claim.

Authorized next step after verified stub baseline commit and clean worktree:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Pinned skill commands unchanged; use unique LGP task IDs. Coder modifies only new
ledger implementation and progress notes. No push, merge or external repo writes.
