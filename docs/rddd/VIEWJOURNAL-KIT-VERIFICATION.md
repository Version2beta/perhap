# Stage 3A kit verification before loop

Go base 3afcc67a47f47eb9b016e78232c18b4bbdd16976, rddd-foundation.
Four existing uncommitted OBSERVATION planning files preserved byte-for-byte in
baseline. CORE/REV/ledger/delivery code, dependencies and tests unchanged.

Source pins: draft.6 1a7269503a9ea0b3dbcd50f6b996caf9d3c96d87;
ratified profile INTEROP-PUBOBS/0.1 v0.1.0
9a80977efaf2ff740a257f6e30314dac42ec384e. All 19 checksums pass;
manifest 01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.
No upstream modification. Existing copied profile/vectors remain unchanged.

Design, scope, equality, registered pair-sum model/nullable schema, trusted cut,
API and mapping documented in VIEWJOURNAL-SPEC.md BEFORE kit baseline freeze.
Private independent reference outside checkout; temporarily installed only as
new viewjournal/journal.go, never shipped. Actual pre-loop evidence:

1. go test -count=1 -v ./sdk/go/viewjournal: seven substantive tests plus helper
   pass. First test seeds S/40, persists its pin, reopens, uses explicit channel
   order to commit S/41 BEFORE tail read, checks exact successor and finite cut.
   MaxEvaluations pressure refuses S/42 without losing S/41. Reopen retains pin
   and usable successor. Pin exhaustion/refusal/release also tested; no sleeps.
2. Real subprocess os.Exit(91) before/after evaluation commit, then reopen actual
   SQLite file. Before commit: no accepted checkpoint/revision; after confirmed
   commit: both recover and identical request resolves original receipt.
3. Real subprocess cuts before/after snapshot plan commit; recovered Plans lists
   committed pin even when planning reply was lost, and Read returns later revision2.
4. Competing subprocess cannot open while owner holds exclusive SQLite lock;
   reopen after crash succeeds. Corrupt and truncated/missing journal refuses;
   trusted independent head floor rejects stale restored inventory.
5. No-op checkpoint/public equality, material failed/recovered status, completeness
   and permitted metadata change, independent authorized view, old checkpoint/source,
   concurrent acceptance, available null/unavailable and interrupted/canceled reads
   tested. No partial Batch/certificate intention on a failed read.
6. scripts/loop.sh --check with reference: exit0; all FOURTEEN tasks pass, including
   previous ten, format_check, strict passing stdout lines, current/earlier task
   tests and kit go vet. Final task includes full Go/race/vet, Node complexity and
   frontend tests. Judge byte-identical to installed v1.1.1 script.
7. Restore zero-return stubs, format, rerun SAME judge: exit1; all FOUR new tasks
   P3ACUT/P3ASTATE/P3ARECOVER/P3AALL fail with behavioral test-command exit1.
   Prior ten tasks pass. Reference removed from checkout before baseline commit.

No in-memory fake substituted for durability. Same existing SQLite driver/toolchain;
one local owner, DELETE journal, synchronous EXTRA, fullfsync ON, exclusive SQLite
lock and zero busy timeout. See SPEC for host/filesystem/namespace assumptions.
Process crashes are not power-cut/fsync/media-loss proofs. Host source/authority
serialization and rendering are representative trusted seams, not production AUTH.
Plans/batches are private retention/certificate-intention data, NOT sessions,
cursors, signed barriers, delivery/application evidence or disclosure permission.
No full O/A/V or complete gateway conformance claim.

Authorized next command after verified stub baseline and clean worktree:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Unchanged pinned model commands; new P3A IDs avoid prior attempt-counter reuse.
No push/merge, history rewrite, other repository or previous implementation changes.
