# Stage 1 kit verification

Target branch rddd-foundation; prior HEAD
b274799f1a246c9d19d08eac8af25a8a2ffb1a3a.
Upstream provisional draft.5 HEAD c53e58e99112e77ba6d934edc17e7956f4c38938,
manifest SHA256 078fe67775596c8af6b7a991fb4079a8110b95457833fc16ee52d35b3a13fa14.
All 16 upstream manifest entries verified. SOURCE-SHA256SUMS pins working files.
Candidate approval for this work is owner authorization, not upstream ratification.

The existing three planning documents were preserved with targeted clarification
and provenance updates. Their earlier stop/evidence descriptions are historical.
C1 is resolved; no contradiction found in the clarified completeness semantics.

## Actual checks before baseline commit

- Private independently written Go reference kept outside this repository.
- Copied reference into only sdk/go/core/evaluate.go temporarily.
- scripts/loop.sh --check: exit 0; pass T1, T2, T3. Includes format_check,
  current and earlier task tests, go vet ./..., all Go packages, frontend tests,
  and seven complexity-script tests. Node dependencies installed using the
  repository's existing demo/web/package-lock.json with npm ci --ignore-scripts
  --no-audit --no-fund. No dependency/lockfile changes.
- go test -count=1 -v ./sdk/go/core: independent behavior, ownership and all six
  permutations pass; 29/29 unchanged shared fixtures pass, including both phases
  of late replay and all six completeness clarification cases.
- Restored zero-return Evaluate stub.
- scripts/loop.sh --check: exit 1; FAIL T1, T2, T3, each with test command exit 1.
  This is behavioral failure, not compile failure or missing expected test names.
- cmp scripts/loop.sh against installed spec-loop v1.1.1 script: identical.
- cmp shared core.json against pinned upstream: identical.
- git diff --check: exit 0.

The verified kit is an additive overlay on the real existing repository, not a
sandbox replacement. Existing code/tests remain protected from loop edits.
The private reference is not shipped or committed. The baseline contains stubs,
not a completed implementation. No live coder/model success is asserted here.

## Claims and limitations

Only pure finite-history behavior is implemented by the kit's task scope.
No NATS runtime, REV publication, AUTH enforcement, effect execution, recovery,
agent infrastructure, JCS/INTEROP-JSON or STATIC conformance is claimed.
Generic state uses an explicit domain clone callback; its correctness and purity
remain domain-author obligations. The library does not authenticate frontier
certificates or callback implementations. No new external dependencies.
The bounded tests are not a full proof of every arbitrary domain model or graph.
Additional randomized DAG/multiple-parent and boundary-schema validation remain
future evidence work. No intentional upstream semantic deviation is adopted.
