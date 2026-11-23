# Stage 1 plan — no implementation kit yet

Status: C1 resolved by draft.5; proceeding with the authorized Stage 1 kit.
Root SPEC.md and tasks.json record the current executable slice.
The plan and earlier evidence below are retained as historical context.
Signatures and tests below are proposals, not protected contracts.
Authorization for a verified kit and bounded local loop is already present;
the unresolved semantics, not lack of execution permission, prevents proceeding.

## Smallest coherent slice

A finite-history pure generic evaluator in sdk/go/core, unchanged fixture copy
and a test-only counter adapter. Proposed ordered tasks:

1. Initialize/validate, availability, current versus last-valid state and prefixes.
2. Normalize identity and build classified causal/dependency graph; deterministic
   ready-node sort, cycles and blocked descendants.
3. Reduce with isolated candidate ownership, distinguish validation invalidity
   from callback failure, retain invalid state and permit repair.
4. Final pure selection, stable diagnostic keys, provenance and late reconstruction.
5. Shared fixtures plus independent permutations, negative tests and full-tree gate.

Do not pad task count; split only if task ownership and behavioral tests warrant.
Counter fixture adapter is not the production model.

After upstream resolution: freeze subordinate SPEC.md, exported types,
unchanged shared fixtures, behavioral tests and exact tasks.json expectations.
Protect those plus PROMPT.md, README.md, loop and judge configuration.
Leave implementation bodies as zero-return stubs. Keep the independently written
reference outside the checkout/kit. Copy installed v1.1.1 scripts/loop.sh unchanged
after reading its header and references/kit-format.md in full.

For every task use go test -count=1 -v with an exact anchored passing-test line
for each named guarantee; exit 0 alone and package ok lines are insufficient.
Final task includes go test -count=1 ./..., go vet ./..., frontend tests and
node --test scripts/complexity-score.test.mjs. Resolve frontend dependency
availability using its authoritative lockfile before claiming a whole-tree pass.

Verify every new task fails on stubs using scripts/loop.sh --check and every
task passes on the private reference using the SAME exact judge, including
earlier tasks, protected-file restoration, format_check and top-level check.
Inspect representative fixture results manually through Go tests.
Restore stubs, inspect only kit-owned changes, commit the verified baseline
locally, assert clean status, then run:
MAX_ITER=12 MAX_ATTEMPTS=3 scripts/loop.sh
Use unchanged configured model commands, no substitutes. Inspect judged commits,
task passes and remaining gaps; never push.

## Evidence at base b274799f1a246c9d19d08eac8af25a8a2ffb1a3a

- go test ./...: exit 0, sdk/go passes including embedded-NATS tests;
  demo/cart has no tests.
- go vet ./...: exit 0.
- node --test scripts/complexity-score.test.mjs: 7 passed.
- cd demo/web && npm test: exit 1, 5 passed, 2 test files fail to load:
  missing nats.ws and esbuild. No dependencies installed in this review.
- Upstream manifest: 14/14 verified. No upstream smoke-run claimed.
- No new CORE tests, stubs, reference, tasks or loop verification yet.
- No baseline/loop commits. No new capability conformance claim.
- No specification deviations implemented.

## Later capabilities

REV publication/effective-revision reconstruction, AUTH admission/observation,
EFFECT commitment/execution, PARTICIPANT invocation, CLIENT protocol,
INTEROP-JSON and STATIC remain separate, unimplemented stage obligations.
Runtime migration, durable recovery and optimization require separate evidence.
