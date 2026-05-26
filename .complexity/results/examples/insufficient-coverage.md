# Example: insufficient coverage

- Scope kind: `whole_repo`
- Scope roots: `sdk/go`, `demo/web/src`, `demo/cart`, `README.md`, `DEVELOPING.md`, `spec/specification.md`
- Scope files via `rg --files`: 20
- Observation count: 3
- Files touched by evidence: 3

## Dimensions

| Dimension | Score | Confidence | Top contributors |
| --- | --- | --- | --- |
| `cognitive_load` | 70 | insufficient (0.115) | control_flow (+3), concurrency (-3), state (-1.5), context_economy (+1), ordering (+0) |
| `context_load` | 76 | insufficient (0.052) | architecture_docs (+2), context_economy (+1.5), separation (+0), radius (+0), dependencies (+0) |
| `confidence` | 68 | insufficient (0.013) | concurrency (-1), boundaries (+0), contracts (+0), validation (+0), tests (+0) |
| `orientation` | 77 | insufficient (0.065) | architecture_docs (+3), orientation (+1.5), naming (+0), shape (+0), state_vocabulary (+0) |
| `agentic_readiness` | 73 | insufficient (0.02) | context_economy (+1), contracts (+0), validation (+0), tests (+0), current_state (+0) |

## Families

| Family | Score | Confidence | Observed facets |
| --- | --- | --- | --- |
| `logic` | 65 | insufficient (0.15) | 3/5 |
| `context` | 75 | insufficient (0.05) | 1/5 |
| `meaning` | 72 | insufficient (0) | 0/6 |
| `correctness` | 72 | insufficient (0) | 0/6 |
| `continuity` | 80 | insufficient (0.1) | 2/5 |

## Facets

| Facet | Family | Score | Confidence | Obs | Anchors | Kinds |
| --- | --- | --- | --- | --- | --- | --- |
| `state` | `logic` | 36 | low (0.25) | 1 | 1 | 1 |
| `side_effects` | `logic` | 72 | insufficient (0) | 0 | 0 | 0 |
| `control_flow` | `logic` | 100 | low (0.25) | 1 | 1 | 1 |
| `ordering` | `logic` | 72 | insufficient (0) | 0 | 0 | 0 |
| `concurrency` | `logic` | 0 | low (0.25) | 1 | 1 | 1 |
| `separation` | `context` | 72 | insufficient (0) | 0 | 0 | 0 |
| `radius` | `context` | 72 | insufficient (0) | 0 | 0 | 0 |
| `dependencies` | `context` | 72 | insufficient (0) | 0 | 0 | 0 |
| `locality` | `context` | 72 | insufficient (0) | 0 | 0 | 0 |
| `context_economy` | `context` | 86 | low (0.25) | 1 | 1 | 1 |
| `naming` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `language` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `shape` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `state_vocabulary` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `argument_patterns` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `idiomaticity` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `boundaries` | `correctness` | 72 | insufficient (0) | 0 | 0 | 0 |
| `contracts` | `correctness` | 72 | insufficient (0) | 0 | 0 | 0 |
| `validation` | `correctness` | 72 | insufficient (0) | 0 | 0 | 0 |
| `tests` | `correctness` | 72 | insufficient (0) | 0 | 0 | 0 |
| `demonstrations` | `correctness` | 72 | insufficient (0) | 0 | 0 | 0 |
| `explainability` | `correctness` | 72 | insufficient (0) | 0 | 0 | 0 |
| `orientation` | `continuity` | 86 | low (0.25) | 1 | 1 | 1 |
| `decision_memory` | `continuity` | 72 | insufficient (0) | 0 | 0 | 0 |
| `current_state` | `continuity` | 72 | insufficient (0) | 0 | 0 | 0 |
| `handoff` | `continuity` | 72 | insufficient (0) | 0 | 0 | 0 |
| `architecture_docs` | `continuity` | 100 | low (0.25) | 1 | 1 | 1 |

