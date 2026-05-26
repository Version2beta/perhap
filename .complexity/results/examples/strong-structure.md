# Example: strong structural signals

- Evidence mode: `fresh`
- Scope kind: `whole_repo`
- Scope roots: `sdk/go`, `demo/web/src`, `demo/cart`, `README.md`, `DEVELOPING.md`, `spec/specification.md`
- HEAD commit: `c86c6f1f1c2d`
- Tree fingerprint: `3d0f77b9330ca7d8`
- Scope files via `rg --files`: 20
- Observation count: 10
- Files touched by evidence: 14

## Read Ledger

- Code anchors: 11
- Test anchors: 5
- Docs anchors: 8
- Validation commands: 2

## Dimensions

| Dimension | Score | Confidence | Top contributors |
| --- | --- | --- | --- |
| `cognitive_load` | 88 | low (0.381) | control_flow (+3), side_effects (+2), state (+1.5), ordering (+1.5), context_economy (+1) |
| `context_load` | 81 | low (0.435) | architecture_docs (+2), separation (+1.5), dependencies (-1.5), locality (+1.5), context_economy (+1.5) |
| `confidence` | 84 | medium (0.583) | tests (+2.25), boundaries (+1.5), explainability (+1.5), demonstrations (+1), idiomaticity (+1) |
| `orientation` | 87 | low (0.461) | architecture_docs (+3), naming (+1.5), shape (+1.5), state_vocabulary (+1.5), orientation (+1.5) |
| `agentic_readiness` | 85 | medium (0.546) | tests (+2.25), idiomaticity (+2), current_state (+1.5), handoff (+1.5), boundaries (+1) |

## Families

| Family | Score | Confidence | Observed facets |
| --- | --- | --- | --- |
| `logic` | 89 | low (0.31) | 4/5 |
| `context` | 78 | low (0.38) | 4/5 |
| `meaning` | 84 | low (0.317) | 4/6 |
| `correctness` | 84 | medium (0.633) | 6/6 |
| `continuity` | 89 | low (0.39) | 5/5 |

## Facets

| Facet | Family | Score | Confidence | Obs | Anchors | Kinds |
| --- | --- | --- | --- | --- | --- | --- |
| `state` | `logic` | 86 | low (0.25) | 1 | 3 | 2 |
| `side_effects` | `logic` | 100 | low (0.25) | 1 | 2 | 2 |
| `control_flow` | `logic` | 100 | high (0.8) | 3 | 7 | 2 |
| `ordering` | `logic` | 86 | low (0.25) | 1 | 2 | 2 |
| `concurrency` | `logic` | 72 | insufficient (0) | 0 | 0 | 0 |
| `separation` | `context` | 86 | medium (0.7) | 2 | 4 | 2 |
| `radius` | `context` | 72 | insufficient (0) | 0 | 0 | 0 |
| `dependencies` | `context` | 36 | low (0.25) | 1 | 3 | 1 |
| `locality` | `context` | 86 | medium (0.7) | 2 | 5 | 2 |
| `context_economy` | `context` | 86 | low (0.25) | 1 | 2 | 1 |
| `naming` | `meaning` | 86 | medium (0.7) | 2 | 6 | 2 |
| `language` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `shape` | `meaning` | 86 | low (0.25) | 1 | 2 | 2 |
| `state_vocabulary` | `meaning` | 86 | medium (0.7) | 2 | 6 | 3 |
| `argument_patterns` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `idiomaticity` | `meaning` | 100 | low (0.25) | 1 | 3 | 2 |
| `boundaries` | `correctness` | 86 | low (0.25) | 1 | 4 | 2 |
| `contracts` | `correctness` | 77 | high (0.8) | 3 | 8 | 3 |
| `validation` | `correctness` | 77 | high (0.8) | 3 | 8 | 3 |
| `tests` | `correctness` | 93 | high (0.9) | 4 | 11 | 3 |
| `demonstrations` | `correctness` | 86 | low (0.25) | 1 | 4 | 2 |
| `explainability` | `correctness` | 86 | high (0.8) | 3 | 7 | 3 |
| `orientation` | `continuity` | 86 | low (0.25) | 1 | 3 | 1 |
| `decision_memory` | `continuity` | 86 | low (0.25) | 1 | 3 | 1 |
| `current_state` | `continuity` | 86 | medium (0.6) | 2 | 5 | 1 |
| `handoff` | `continuity` | 86 | low (0.25) | 1 | 3 | 1 |
| `architecture_docs` | `continuity` | 100 | medium (0.6) | 2 | 6 | 1 |

