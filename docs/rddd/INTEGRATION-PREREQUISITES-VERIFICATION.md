# Optional Go prerequisite kit evidence

Baseline f13af35e29f669c3247c9980fc13ae1b1e5222ad clean except approved untracked
GO-CLIENT-INTEGRATION-DESIGN.md, preserved unchanged. Client clean at supplied
81641540cb8e79d002cf9599ff33118b44e12df0. Frozen spec pins and all19 checksums
verified; manifest 01ef78d4344b52dee458856abeac2c676f14b74fcf5437a091cc07c4f102b68f.

Owner separately approved both prerequisites and integration. This kit verifies Go
prerequisites FIRST. Two new implementation files: observation/resume.go and
submission/matching.go. Existing recovery validator is wired for purpose resume;
otherwise existing source protected. Existing APIs and default framing preserved.

Independent private reference outside checkout passed all31 unchanged-judge groups,
including previous28 focused groups and three new GIPRESUME/GIPMATCH/GIPALL. Focused
native tests pass. Restored formatted stubs: all3 new groups fail through actual test
commands, old28 focused groups pass. Reference removed before clean stub baseline.
Last group includes full Go/race/vet/Node/frontend. No TypeScript runtime/code changes
in this kit. Passing these tests is Go-only evidence, not cross-process row execution.

Tests cover accepted43 -> purpose-bound handshake43 ->44/45/barrier45, valid scope/
observer/proof, revoked-before-handshake zero callbacks, callback uncertainty/pin
cleanup, bad cursor explicit cannot-resume. Existing Resume tests protect no-handshake
legacy framing. Matching tests use actual SQLite reopen and normalization1.0/1,
latest evaluated facts, mismatch/attribution with zero original-data disclosure,
missing lookup without receipt, no Admission callback and revocation-fenced conflict.
Mismatch is a protected identity-conflict Reply rather than an unfenced return value.

Authorized next command after clean baseline: MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh.
Exact v1.1.1 script and pinned models unchanged. No production AUTH or transport yet.
