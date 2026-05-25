**Reasoning Simplicity**
- Scope: Bounded implementation assessment of the shipped runtime and support surfaces: `sdk/go`, `demo/web/src`, `README.md`, `DEVELOPING.md`, `demo/web/package.json`, `demo/web/tsconfig.json`, and `.github/workflows/ci.yml`. The long draft spec is treated as design context, not part of the scored executable surface.
- Full skill file read: yes
- Overall: Refactor target met for the bounded implementation scope. The shipped code now lands at a defensible overall score of 91, with the lowest computed dimension rounding to 88.
- Interpretation: Intent, safe change, and validation are now much more obvious because the browser SDK is split into testable seams, runtime contracts are tighter, and validation exists in both Go and TypeScript plus CI.
- Validation: `go test ./...` passed; `go vet ./...` passed; `npm test` passed; `npx tsc --noEmit` passed; `npm run build` passed.

**Evidence Coverage**
| Resource | Used | Evidence Or Reason Skipped |
| --- | --- | --- |
| README and onboarding docs | Yes | `README.md:106-289`, `DEVELOPING.md:50-163` |
| Architecture / contract context | Yes | `spec/specification.md:1-8` for implementation note only |
| Tests and validation commands | Yes | `go test ./...`, `go vet ./...`, `npm test`, `npx tsc --noEmit`, `npm run build` |
| Schemas and wire contracts | Yes | `sdk/go/event.go:13-64`, `sdk/go/service.go:43-104`, `demo/web/src/model-state.ts:1-110` |
| Size and symbol scans | Yes | `rg --files`, `wc -l` scan, symbol scan over `sdk/go` and `demo/web/src` |
| State, side-effect, and dependency scans | Yes | explicit review of `demo/web/src/core.ts`, `demo/web/src/dom.ts`, `sdk/go/service.go`, `sdk/go/supervisor.go` |
| Current-state / enforcement | Yes | `.github/workflows/ci.yml:1-44`, clean validation outputs |

**Dimensions**
| Dimension | Score | Interpretation |
| --- | --- | --- |
| cognitive_load | 88 (computed) | The main shipped paths are split into smaller, locally understandable units. |
| context_load | 95 (computed, conservatively reported) | Safe work needs much less cross-file reconstruction than before, and CI/test commands are first-class. |
| confidence | 95 (computed, conservatively reported) | Replay, `_exception` persistence, optimistic pending, UUID ordering, typecheck, build, and CI are all evidenced. |
| orientation | 90 (computed) | A fresh engineer can form the right mental model quickly from the code layout and docs. |
| agentic_readiness | 96 (computed, conservatively reported) | An agent can now patch, validate, and hand off changes with materially less hidden state or contract ambiguity. |

**Facet Families**
| Family | Net Impact | Read |
| --- | --- | --- |
| logic | 42 | strongly reveals intent or correctness |
| context | 59 | strongly reveals intent or correctness |
| meaning | 25 | strongly reveals intent or correctness |
| correctness | 99 | strongly reveals intent or correctness |
| continuity | 47 | strongly reveals intent or correctness |

**Top Drivers**
| Driver | Effect | Evidence |
| --- | --- | --- |
| Browser SDK split into entry/core/state/DOM/UUID modules | Helps | `demo/web/src/perhap.ts:1-9`, `demo/web/src/core.ts:1-214`, `demo/web/src/model-state.ts:1-110`, `demo/web/src/dom.ts:1-227`, `demo/web/src/uuid.ts:1-38` |
| Optimistic pending and confirmation rules are now explicit and testable | Helps | `demo/web/src/model-state.ts:47-109`, `demo/web/src/model-state.test.ts:1-73` |
| Browser UUID generation is monotonic per source instead of purely time-plus-random | Helps | `demo/web/src/uuid.ts:1-38`, `demo/web/src/uuid.test.ts:1-15` |
| Go runtime contracts are clearer: configurable stream replicas and explicit stored `_exception` publishing | Helps | `sdk/go/domain.go:125-164`, `sdk/go/service.go:43-104`, `sdk/go/supervisor.go:230-282` |
| Embedded-NATS integration tests cover replay and internal event persistence | Helps | `sdk/go/service_integration_test.go:21-250` |
| CI now enforces the validation path instead of leaving it as tribal knowledge | Helps | `.github/workflows/ci.yml:1-44` |

**Findings**
| Finding | Observation IDs | Summary |
| --- | --- | --- |
| F1. Context economy improved most in the browser SDK | O03, O04, O16, O19 | The highest-cost file was replaced with smaller modules centered on one responsibility each. |
| F2. Correctness is now supported by executable evidence instead of docs alone | O08, O09, O10, O11, O12 | Both runtimes have direct tests and a repeatable validation path. |
| F3. Contract drift on shipped behavior is much smaller | O01, O06, O07, O08 | Public docs and runtime semantics now agree more closely on pending, replay, `_exception`, and replicas. |
| F4. A few residual trust edges remain | O13, O14, O15 | There is still no live browser-plus-NATS end-to-end smoke test, and replay/deep-copy fallback behavior is not fully hardened. |

**Recommendations**
| Recommendation | Addresses | Expected Impact | Validation |
| --- | --- | --- | --- |
| Add one end-to-end smoke test that runs the Go service against a live NATS server and drives the browser SDK through a minimal DOM harness. | O13 | `+1 confidence`, `+1 orientation` | `go test ./...`, `npm test`, CI |
| Replace replay-time silent JSON skips with explicit surfaced errors or metrics. | O14 | `+1 confidence` | Add a focused Go integration test around malformed stored events |
| Replace JSON round-trip deep-copy fallback with a stricter copier or explicit unsupported-model failure path. | O15 | `+1 cognitive_load`, `+1 confidence` | `go test ./...`, `go vet ./...` |

**Gaps**
- This score is for the shipped implementation surface, not the whole repository including the draft design specification.
- No browser automation or full end-to-end UI run was executed; the frontend evidence is from pure-state tests, typecheck, and build.

**Observation Ledger**
`O01` README and DEVELOPING now document current runtime semantics instead of only broad intent. Evidence: `README.md:106-289`, `DEVELOPING.md:50-163`.

`O02` The spec now explicitly marks itself broader than the current implementation, reducing accidental contract overreach. Evidence: `spec/specification.md:3-8`.

`O03` The browser entry point is only wiring and exports, not the whole SDK. Evidence: `demo/web/src/perhap.ts:1-9`.

`O04` Pure state transitions are isolated from transport and DOM work. Evidence: `demo/web/src/model-state.ts:1-110`.

`O05` Browser UUID ordering is now monotonic per source. Evidence: `demo/web/src/uuid.ts:1-38`, `demo/web/src/uuid.test.ts:1-15`.

`O06` Pending semantics are now explicit in code and docs as backend pending plus in-memory optimistic pending. Evidence: `demo/web/src/model-state.ts:47-97`, `README.md:227-263`, `DEVELOPING.md:127-132`.

`O07` JetStream replication is configurable at the domain and service level. Evidence: `sdk/go/domain.go:125-164`, `sdk/go/service.go:43-104`.

`O08` `_exception` events are published through the same stored-event path as other persisted events. Evidence: `sdk/go/supervisor.go:230-282`.

`O09` Go integration tests now cover replay ordering, recreated supervisors, and `_exception` persistence with an embedded NATS server. Evidence: `sdk/go/service_integration_test.go:21-250`.

`O10` Frontend tests cover optimistic confirmation, stale update rejection, buffered hydration ordering, and UUID monotonicity. Evidence: `demo/web/src/model-state.test.ts:1-73`, `demo/web/src/uuid.test.ts:1-15`.

`O11` CI captures the full validation path for both runtimes. Evidence: `.github/workflows/ci.yml:1-44`.

`O12` All local validation commands pass after the refactor. Evidence: command output for `go test ./...`, `go vet ./...`, `npm test`, `npx tsc --noEmit`, `npm run build`.

`O13` There is still no live browser-plus-NATS end-to-end test. Evidence: repo inventory and validation commands.

`O14` Replay still silently skips malformed stored events during JSON unmarshal. Evidence: `sdk/go/service.go:281-288`.

`O15` Snapshot deep-copy still falls back to returning shared references if marshaling fails. Evidence: `sdk/go/supervisor.go:13-25`.

`O16` The largest browser implementation files are now `dom.ts` at 227 lines and `core.ts` at 214 lines instead of one 521-line file. Evidence: `wc -l` scan.

`O17` Per-entity supervisors remain a strong isolation boundary. Evidence: `sdk/go/service.go:74-80`, `sdk/go/supervisor.go:27-107`.

`O18` The typed domain builder is still a strong clarity anchor for Go API usage. Evidence: `sdk/go/domain.go:11-190`, `sdk/go/domain_test.go:19-133`.

`O19` The public browser API stayed stable while internals became smaller. Evidence: `demo/web/src/perhap.ts:1-9`, `README.md:188-210`.

`O20` Package scripts and TS config now explicitly support frontend tests and Node-aware typechecking. Evidence: `demo/web/package.json:6-22`, `demo/web/tsconfig.json:2-12`.
