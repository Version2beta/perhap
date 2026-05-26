# Perhap baseline whole repo v1

- Scope kind: `whole_repo`
- Scope roots: `sdk/go`, `demo/web/src`, `demo/cart`, `README.md`, `DEVELOPING.md`, `spec/specification.md`
- Scope files via `rg --files`: 20
- Observation count: 16
- Files touched by evidence: 15

## Dimensions

| Dimension | Score | Confidence | Top contributors |
| --- | --- | --- | --- |
| `cognitive_load` | 80 | medium (0.66) | control_flow (+1.875), concurrency (-1.5), context_economy (+1), locality (+1), shape (+1) |
| `context_load` | 78 | medium (0.527) | separation (+1.5), radius (-1.5), dependencies (-1.5), locality (+1.5), context_economy (+1.5) |
| `confidence` | 82 | high (0.843) | tests (+3), boundaries (+1.5), demonstrations (+1), idiomaticity (+0.75), contracts (+0.642) |
| `orientation` | 86 | medium (0.678) | architecture_docs (+2.25), naming (+1.5), shape (+1.5), state_vocabulary (+1.5), orientation (+1.5) |
| `agentic_readiness` | 83 | high (0.756) | tests (+3), handoff (+1.5), idiomaticity (+1.5), boundaries (+1), context_economy (+1) |

## Families

| Family | Score | Confidence | Observed facets |
| --- | --- | --- | --- |
| `logic` | 75 | medium (0.67) | 5/5 |
| `context` | 75 | low (0.43) | 5/5 |
| `meaning` | 83 | medium (0.517) | 4/6 |
| `correctness` | 83 | high (0.883) | 6/6 |
| `continuity` | 86 | medium (0.65) | 5/5 |

## Facets

| Facet | Family | Score | Confidence | Obs | Anchors | Kinds |
| --- | --- | --- | --- | --- | --- | --- |
| `state` | `logic` | 72 | medium (0.7) | 2 | 4 | 2 |
| `side_effects` | `logic` | 79 | medium (0.7) | 2 | 5 | 3 |
| `control_flow` | `logic` | 90 | high (0.9) | 4 | 9 | 3 |
| `ordering` | `logic` | 77 | high (0.8) | 3 | 6 | 3 |
| `concurrency` | `logic` | 36 | low (0.25) | 1 | 2 | 2 |
| `separation` | `context` | 86 | medium (0.7) | 2 | 5 | 3 |
| `radius` | `context` | 36 | low (0.25) | 1 | 2 | 1 |
| `dependencies` | `context` | 36 | low (0.25) | 1 | 2 | 1 |
| `locality` | `context` | 86 | medium (0.7) | 2 | 6 | 2 |
| `context_economy` | `context` | 86 | low (0.25) | 1 | 3 | 2 |
| `naming` | `meaning` | 86 | high (0.9) | 4 | 9 | 3 |
| `language` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `shape` | `meaning` | 86 | medium (0.7) | 2 | 5 | 4 |
| `state_vocabulary` | `meaning` | 86 | high (0.8) | 3 | 7 | 2 |
| `argument_patterns` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `idiomaticity` | `meaning` | 93 | medium (0.7) | 2 | 6 | 3 |
| `boundaries` | `correctness` | 86 | high (0.8) | 3 | 9 | 4 |
| `contracts` | `correctness` | 78 | high (1) | 7 | 14 | 4 |
| `validation` | `correctness` | 72 | high (0.9) | 4 | 9 | 3 |
| `tests` | `correctness` | 100 | high (0.8) | 3 | 6 | 3 |
| `demonstrations` | `correctness` | 86 | high (0.8) | 3 | 7 | 4 |
| `explainability` | `correctness` | 77 | high (1) | 6 | 14 | 4 |
| `orientation` | `continuity` | 86 | medium (0.7) | 2 | 7 | 2 |
| `decision_memory` | `continuity` | 86 | low (0.25) | 1 | 3 | 2 |
| `current_state` | `continuity` | 77 | high (0.8) | 3 | 7 | 2 |
| `handoff` | `continuity` | 86 | high (0.8) | 3 | 10 | 2 |
| `architecture_docs` | `continuity` | 93 | medium (0.7) | 2 | 6 | 2 |

