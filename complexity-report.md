**Reasoning Simplicity**
- Scope: Whole repository assessment of `/Users/rob/Development/perhap`
- Full skill file read: yes
- Overall: Moderately simple. The repository is small, the core event pipeline is explicit, and the Go runtime is locally understandable. The main reasoning cost is contract drift between docs/spec and shipped behavior rather than raw code sprawl.
- Interpretation: Safe local changes are fairly obvious in the Go SDK once you are inside `domain.go`, `service.go`, or `supervisor.go`. Fresh readers pay most of their complexity tax at the contract edges: `_exception` durability, `pending`/outbox semantics, production topology claims, and the frontend's client-side ordering assumptions.
- Validation: `go test ./...` passed; `go vet ./...` passed; `npm run build` passed; `npx tsc --noEmit` passed.

**Evidence Coverage**
| Resource | Used | Evidence Or Reason Skipped |
| --- | --- | --- |
| README and onboarding docs | Yes | `README.md:1-285` and `DEVELOPING.md:1-142` |
| Architecture and design docs | Yes | Targeted sections in `spec/specification.md`, including `:134-253`, `:321`, `:668-707`, `:918-1016`, `:1371-1471` |
| Tests and validation commands | Yes | `go test ./...`, `go vet ./...`, `npm run build`, `npx tsc --noEmit` |
| Schemas and wire contracts | Yes | `README.md:15-24`, `sdk/go/event.go:13-64`, `sdk/go/service.go:89-118`, `demo/web/src/perhap.ts:24-34` |
| Size and symbol scans | Yes | `rg --files`, `rg --files | xargs wc -l | sort -nr`, symbol scans over `sdk/go`, `demo/web/src`, `demo/cart` |
| State, side-effect, and dependency scans | Yes | `rg -n` scans over timers, channels, NATS calls, DOM listeners, `Date.now`, `Math.random`, and process env usage |
| Naming and vocabulary scan | Yes | `rg -n` scans for drift-prone names plus manual comparison of `product_id`, `quantity`, `pending`, `reduced_through`, `_exception` |
| Worktree and current-state context | Yes | `git status --short` showed active edits in `README.md`, `DEVELOPING.md`, `demo/web/src/perhap.ts`, `sdk/go/service.go`, `sdk/go/supervisor.go` |
| Workflows / CI | No | No workflow files appeared in the repository inventory from `rg --files` |

**Dimensions**
| Dimension | Score | Interpretation |
| --- | --- | --- |
| cognitive_load | 76 (computed) | Core runtime reasoning is good because the main event path is explicit and the repo is small. Load rises when switching into the browser SDK or reconciling doc promises with current code. |
| context_load | 68 (computed) | Most changes stay bounded, but safe work still requires cross-checking README, DEVELOPING notes, the draft spec, and the single-file browser SDK. |
| confidence | 71 (computed) | Build, typecheck, vet, and the existing Go tests all pass, but confidence is capped by missing integration coverage and several contract mismatches. |
| orientation | 70 (computed) | A fresh engineer gets a good mental model quickly from the docs and small file count, then slows down at the points where docs and implementation diverge. |
| agentic_readiness | 73 (computed) | An agent can patch this repo safely in bounded areas, but should verify behavior carefully before changing transport contracts, pending semantics, or ordering logic. |

**Facet Families**
| Family | Net Impact | Read |
| --- | --- | --- |
| logic | 37 | strongly reveals intent or correctness |
| context | 9 | meaningfully helps reasoning |
| meaning | 14 | meaningfully helps reasoning |
| correctness | 9 | meaningfully helps reasoning |
| continuity | 18 | meaningfully helps reasoning |

**Top Drivers**
| Driver | Effect | Evidence |
| --- | --- | --- |
| Typed domain builder with sealed configuration makes handler intent obvious | Helps | `sdk/go/domain.go:11-30`, `sdk/go/domain.go:60-88`, `sdk/go/domain.go:135-181`, `sdk/go/domain_test.go:19-133` |
| Per-entity supervisor and replay model keep state and ordering locally tractable | Helps | `sdk/go/service.go:176-304`, `sdk/go/supervisor.go:28-183`, `DEVELOPING.md:68-86` |
| Small repository size and concentrated source surface keep exploration bounded | Helps | `rg --files` inventory, `wc -l` scan showing 4667 total lines, largest implementation files at `demo/web/src/perhap.ts:521`, `sdk/go/service.go:305`, `sdk/go/supervisor.go:288` |
| Documentation is strong, but current implementation does not fully match the written contract | Hurts | `spec/specification.md:192`, `:228-241`, `:321`, `:668-707`, `demo/web/src/perhap.ts:5-12`, `:241-249`, `:263-290`, `sdk/go/service.go:94-100`, `sdk/go/supervisor.go:249-287` |
| Browser SDK centralizes transport, optimistic state, DOM binding, and mutation wiring in one file | Hurts | `demo/web/src/perhap.ts:1-521` |
| Validation is present but narrow outside the domain builder | Hurts | `sdk/go/domain_test.go:19-133`, passing command output for `go test ./...`, and lack of additional test files in `sdk/go`, `demo/web`, and `demo/cart` |

**Findings**
| Finding | Observation IDs | Summary |
| --- | --- | --- |
| F1. Core runtime is locally understandable | O05, O07, O08, O12, O14, O15, O17, O18 | The main Go runtime favors explicit control flow, per-entity isolation, and small APIs, so correctness is usually inspectable in bounded code reads. |
| F2. Contract drift is the main reasoning tax | O03, O04, O13, O20, O22, O23, O25 | Complexity comes less from algorithms and more from reconciling what the docs/spec promise with what the implementation actually does today. |
| F3. Confidence is limited more by coverage gaps and hidden failure handling than by size | O09, O10, O11, O16, O19 | Existing validation is real, but it concentrates on the domain builder and misses replay, exception, and browser ordering/pending edge cases. |
| F4. The browser SDK is the highest-context area | O21, O22, O23, O24, O25 | Hydration behavior is thoughtfully implemented, but the frontend still carries the largest share of state vocabulary drift, ordering assumptions, and concern mixing. |

**Recommendations**
| Recommendation | Addresses | Expected Impact | Validation |
| --- | --- | --- | --- |
| R1. Align written contracts with shipped behavior for `_exception`, `pending`/outbox, and production topology, or implement the missing behavior so the code matches the docs. | O04, O13, O20, O22, O23, O25 | `+4 confidence`, `+4 orientation`, `+3 agentic_readiness` | Update docs and add scenario checks proving `_exception` durability, `pending` union behavior, and explicit production guidance. |
| R2. Add focused integration tests for replay, reducer-error handling, hydration buffering, stale update dropping, and optimistic pending. | O11, O16, O19, O21, O22, O23 | `+5 confidence`, `+3 agentic_readiness`, `+2 context_load` | `go test ./...`, browser-side test runner for SDK ordering/pending cases, `npm run build`, `npx tsc --noEmit` |
| R3. Make client event IDs monotonic per source, or stop depending on client-side UUID ordering for same-source sequencing. | O21, O23 | `+3 confidence`, `+2 agentic_readiness` | Deterministic tests that emit many frontend events in the same millisecond and verify sort/order behavior. |
| R4. Split `demo/web/src/perhap.ts` into transport, state-model, and DOM-wiring modules without changing the public API. | O24 | `+4 cognitive_load`, `+4 context_load`, `+3 orientation` | `npm run build`, `npx tsc --noEmit`, demo smoke test with live NATS. |
| R5. Remove or contract-test the duplicated demo reducer logic so browser demo behavior cannot silently drift from the Go reducer. | O11, O25 | `+2 confidence`, `+2 orientation` | Shared fixture tests for cart event sequences on both Go and browser implementations. |

**Gaps**
- No live NATS end-to-end run was performed in this pass. Runtime behavior was assessed from source, docs, and local build/test/typecheck results.
- `spec/specification.md` is much larger than the implementation. This report sampled and cross-checked the sections that define current code contracts, not every design-only section line by line.
- The worktree is dirty in core docs and runtime files, so current-state evidence may move while the project is actively being edited.
- No CI or workflow configuration was present in the repo inventory, so ongoing enforcement is unclear.

**Observation Ledger**
`O01`
Summary: The README gives a crisp system model and subject contract up front.
Evidence: `README.md:1-24`
Scale: `scope: 5`, `intensity: 3`, `confidence: high`
Facet impacts: `{ language:+3, contracts:+4, explainability:+4, orientation:+6, architecture_docs:+5 }`
Qualitative notes: First-pass orientation is efficient because the event loop and subject names are explicit.
Recommendation refs: `R1`

`O02`
Summary: `DEVELOPING.md` turns the architecture into a runnable and debuggable workflow.
Evidence: `DEVELOPING.md:3-142`
Scale: `scope: 4`, `intensity: 3`, `confidence: high`
Facet impacts: `{ handoff:+5, current_state:+4, demonstrations:+4, orientation:+6, architecture_docs:+3 }`
Qualitative notes: The lifecycle and troubleshooting sections materially reduce handoff cost.
Recommendation refs: `R1`, `R2`

`O03`
Summary: The draft specification preserves design memory and broader intent well, but it is much larger than the current codebase.
Evidence: `spec/specification.md:134-253`, `:321`, `:668-707`, `:918-1016`, `:1371-1471`
Scale: `scope: 5`, `intensity: 3`, `confidence: medium`
Facet impacts: `{ decision_memory:+8, architecture_docs:+8, orientation:+3, radius:-3, context_economy:-4, current_state:-3 }`
Qualitative notes: Strong decision memory helps, but the size delta between design and implementation increases context retrieval cost.
Recommendation refs: `R1`

`O04`
Summary: Several spec claims do not match current implementation details, creating contract drift.
Evidence: `spec/specification.md:192`, `:228-241`, `:321`, `:668-707`, `sdk/go/service.go:94-100`, `sdk/go/supervisor.go:249-287`, `demo/web/src/perhap.ts:5-12`, `:241-249`
Scale: `scope: 5`, `intensity: 5`, `confidence: high`
Facet impacts: `{ contracts:-7, current_state:-6, architecture_docs:-4, explainability:-4, dependencies:-2 }`
Qualitative notes: This is the single biggest reasoning cost in the repo.
Recommendation refs: `R1`

`O05`
Summary: The repository is small and the implementation is concentrated in a few files.
Evidence: `rg --files` inventory; line-count scan showing 4667 total lines, largest implementation files `demo/web/src/perhap.ts:521`, `sdk/go/service.go:305`, `sdk/go/supervisor.go:288`
Scale: `scope: 5`, `intensity: 2`, `confidence: high`
Facet impacts: `{ radius:+6, locality:+5, context_economy:+4, orientation:+3 }`
Qualitative notes: Bounded size is a real simplicity advantage here.
Recommendation refs: `R4`

`O06`
Summary: The worktree is active in docs and runtime files, which weakens current-state certainty.
Evidence: `git status --short` showed modifications in `README.md`, `DEVELOPING.md`, `demo/web/src/perhap.ts`, `sdk/go/service.go`, `sdk/go/supervisor.go`
Scale: `scope: 4`, `intensity: 2`, `confidence: high`
Facet impacts: `{ current_state:-5, handoff:-3 }`
Qualitative notes: This does not make the code harder, but it makes the current contract less stable during assessment.
Recommendation refs: `R1`

`O07`
Summary: The Go domain builder uses typed handlers and documents its non-obvious generic API choice.
Evidence: `sdk/go/domain.go:11-30`, `:45-88`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ separation:+5, language:+4, argument_patterns:+4, idiomaticity:+5, explainability:+5 }`
Qualitative notes: The package-level `On` function would be surprising without the comment block; with it, the choice is easy to justify.
Recommendation refs: `R2`

`O08`
Summary: Sealing the domain after `ServiceConfig()` and snapshotting handlers reduces mutable configuration ambiguity.
Evidence: `sdk/go/domain.go:35-42`, `:73-88`, `:135-181`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ state:+4, control_flow:+3, boundaries:+4, contracts:+5, explainability:+3 }`
Qualitative notes: This is a good contract-preserving boundary in a generic API.
Recommendation refs: `R2`

`O09`
Summary: The existing Go tests cover the domain builder's typed dispatch and immutability semantics.
Evidence: `sdk/go/domain_test.go:19-133`
Scale: `scope: 3`, `intensity: 3`, `confidence: high`
Facet impacts: `{ tests:+5, validation:+3, contracts:+3, explainability:+2 }`
Qualitative notes: The tests are meaningful for the code they touch.
Recommendation refs: `R2`

`O10`
Summary: Native validation commands all pass.
Evidence: `go test ./...` passed, `go vet ./...` passed, `npm run build` passed, `npx tsc --noEmit` passed
Scale: `scope: 5`, `intensity: 3`, `confidence: high`
Facet impacts: `{ validation:+6, tests:+3, demonstrations:+3, current_state:+2 }`
Qualitative notes: Passing validation supports moderate confidence, but it does not erase missing scenario coverage.
Recommendation refs: `R2`

`O11`
Summary: Automated coverage stops at the domain builder; the service, supervisor, browser SDK, and demo paths are largely untested.
Evidence: `sdk/go/domain_test.go` is the only test file in the repo inventory; no test files were present under `demo/web` or `demo/cart`
Scale: `scope: 5`, `intensity: 4`, `confidence: high`
Facet impacts: `{ tests:-9, validation:-4, demonstrations:-3 }`
Qualitative notes: This keeps confidence lower than the file count would otherwise suggest.
Recommendation refs: `R2`, `R5`

`O12`
Summary: Service defaults and stream creation are centralized in one place.
Evidence: `sdk/go/service.go:43-113`
Scale: `scope: 4`, `intensity: 3`, `confidence: high`
Facet impacts: `{ dependencies:+4, boundaries:+3, separation:+3, orientation:+3 }`
Qualitative notes: This keeps setup logic discoverable and avoids hidden initialization paths.
Recommendation refs: `R1`

`O13`
Summary: The runtime hardcodes JetStream `Replicas: 1`, which conflicts with the production topology described in the spec.
Evidence: `sdk/go/service.go:94-100`, `spec/specification.md:321`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ dependencies:-3, contracts:-6, current_state:-4 }`
Qualitative notes: Even if intentional for now, the mismatch should be explicit.
Recommendation refs: `R1`

`O14`
Summary: Queue processing and hydration are separated by subject prefix, which makes the transport boundary easy to follow.
Evidence: `README.md:15-24`, `sdk/go/service.go:115-166`, `demo/web/src/perhap.ts:112-168`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ ordering:+5, boundaries:+5, control_flow:+4, explainability:+4 }`
Qualitative notes: The `_perhap` boundary is one of the clearest parts of the wire protocol.
Recommendation refs: `R2`

`O15`
Summary: Replay sorts events by UUID and restores state before new processing, which makes passivation behavior legible.
Evidence: `sdk/go/service.go:176-304`, `README.md:130-145`, `DEVELOPING.md:68-86`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ ordering:+6, state:+3, boundaries:+2, explainability:+2 }`
Qualitative notes: This is a strong simplifier for restart and hydration reasoning.
Recommendation refs: `R2`

`O16`
Summary: Replay error handling is weak: timeout-based completion and silent decode skips hide failure modes.
Evidence: `sdk/go/service.go:256-304`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ validation:-5, contracts:-4, explainability:-3, ordering:-2 }`
Qualitative notes: The code is readable, but the trust boundary is softer than the surrounding docs imply.
Recommendation refs: `R2`

`O17`
Summary: Per-entity supervisors isolate mutable state and concurrency to one goroutine plus a small registry.
Evidence: `sdk/go/service.go:72-77`, `:176-203`, `sdk/go/supervisor.go:28-107`
Scale: `scope: 5`, `intensity: 4`, `confidence: high`
Facet impacts: `{ concurrency:+6, state:+5, separation:+5, locality:+3 }`
Qualitative notes: This is the core structural reason the Go runtime is easier to reason about than its event-sourcing domain might suggest.
Recommendation refs: `R2`

`O18`
Summary: Snapshot deep-copying prevents reducer mutations from aliasing into stored state.
Evidence: `sdk/go/supervisor.go:14-25`, `:130-143`, `:170-183`
Scale: `scope: 4`, `intensity: 3`, `confidence: high`
Facet impacts: `{ state:+6, side_effects:+4, explainability:+3 }`
Qualitative notes: This is an explicit defense against a common Go map/slice pitfall.
Recommendation refs: `R2`

`O19`
Summary: The deep-copy fallback and reducer-error handling still hide important failure states.
Evidence: `sdk/go/supervisor.go:18-25`, `:135-155`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ state:-4, contracts:-6, idiomaticity:-2, validation:-2, explainability:-3 }`
Qualitative notes: If a reducer or marshal path violates assumptions, the system can proceed with less visibility than the surrounding API contracts imply.
Recommendation refs: `R2`

`O20`
Summary: `_exception` handling does not currently match the specification's durability claims.
Evidence: `spec/specification.md:192`, `:668-707`, `sdk/go/supervisor.go:271-285`
Scale: `scope: 5`, `intensity: 5`, `confidence: high`
Facet impacts: `{ contracts:-7, current_state:-5, side_effects:-3, architecture_docs:-4 }`
Qualitative notes: This is a direct code-vs-spec mismatch on a correctness-relevant boundary.
Recommendation refs: `R1`

`O21`
Summary: The browser SDK's hydration sequence is explicit and buffered, including a timeout fallback that avoids blank UI.
Evidence: `demo/web/src/perhap.ts:112-168`, `DEVELOPING.md:100-111`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ ordering:+6, control_flow:+4, demonstrations:+3, explainability:+4, boundaries:+4 }`
Qualitative notes: This is the clearest and best-defended part of the frontend runtime.
Recommendation refs: `R2`, `R3`

`O22`
Summary: The implemented frontend state model does not provide the documented outbox-backed `pending` union.
Evidence: `demo/web/src/perhap.ts:5-12`, `:198-210`, `:234-249`, `spec/specification.md:228-241`, `:927-953`, `README.md:241-249`
Scale: `scope: 5`, `intensity: 5`, `confidence: high`
Facet impacts: `{ state_vocabulary:-6, contracts:-6, state:-4, current_state:-4 }`
Qualitative notes: The code comments describe a richer model than the implementation currently carries.
Recommendation refs: `R1`, `R2`

`O23`
Summary: Client-side UUID generation uses `Date.now()` and `Math.random()` without a monotonic source-local sequence.
Evidence: `demo/web/src/perhap.ts:263-290`, `sdk/go/event.go:33-36`, `DEVELOPING.md:107-111`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ ordering:-6, contracts:-4, idiomaticity:-2 }`
Qualitative notes: This weakens the strength of the ordering story for rapid same-source frontend events.
Recommendation refs: `R1`, `R3`

`O24`
Summary: The browser SDK keeps DOM scoping rules local and dynamic wiring workable, but it concentrates too many concerns in one file.
Evidence: `demo/web/src/perhap.ts:1-521`, especially `:311-517`
Scale: `scope: 5`, `intensity: 4`, `confidence: high`
Facet impacts: `{ radius:-6, separation:-4, context_economy:-5, locality:+3, demonstrations:+2, idiomaticity:+2 }`
Qualitative notes: The code is not sprawling, but concern mixing raises patch-planning cost.
Recommendation refs: `R4`

`O25`
Summary: The demo shares field names with the Go reducer, but it duplicates cart reduction logic inline, so behavior can still drift.
Evidence: `demo/cart/cart.go:48-133`, `demo/web/index.html:208-237`, `:256-301`
Scale: `scope: 4`, `intensity: 4`, `confidence: high`
Facet impacts: `{ contracts:-4, state_vocabulary:-2, locality:-2, tests:-2, naming:+4, shape:+4, boundaries:+2 }`
Qualitative notes: Shared vocabulary helps, but duplicated business logic still raises safe-change cost.
Recommendation refs: `R5`
