# Example: temporal risk emphasis

- Evidence mode: `fresh`
- Scope kind: `targeted`
- Scope roots: `sdk/go`, `demo/web/src`, `README.md`, `DEVELOPING.md`
- HEAD commit: `1e28a3c253df`
- Tree fingerprint: `b7b865b5202a5910`
- Scope files via `rg --files`: 17
- Observation count: 10
- Files touched by evidence: 10

## Read Ledger

- Code anchors: 11
- Test anchors: 3
- Docs anchors: 5
- Validation commands: 1

## Dimensions

| Dimension | Score | Confidence | Top contributors |
| --- | --- | --- | --- |
| `cognitive_load` | 65 | low (0.495) | concurrency (-3), ordering (-2.25), state (-1.5), context_economy (+1), locality (+1) |
| `context_load` | 73 | low (0.313) | radius (-1.5), dependencies (-1.5), locality (+1.5), context_economy (+1.5), architecture_docs (+1.5) |
| `confidence` | 64 | medium (0.594) | validation (-2.25), tests (+2.25), contracts (-2.001), boundaries (+1.5), explainability (-1.5) |
| `orientation` | 86 | low (0.293) | architecture_docs (+2.25), naming (+1.5), shape (+1.5), state_vocabulary (+1.5), orientation (+1.5) |
| `agentic_readiness` | 72 | low (0.485) | validation (-2.25), tests (+2.25), contracts (-2.001), current_state (-1.5), handoff (+1.5) |

## Families

| Family | Score | Confidence | Observed facets |
| --- | --- | --- | --- |
| `logic` | 37 | medium (0.675) | 5/5 |
| `context` | 72 | insufficient (0.2) | 4/5 |
| `meaning` | 81 | insufficient (0.167) | 4/6 |
| `correctness` | 70 | medium (0.604) | 6/6 |
| `continuity` | 82 | low (0.35) | 5/5 |

## Facets

| Facet | Family | Score | Confidence | Obs | Anchors | Kinds |
| --- | --- | --- | --- | --- | --- | --- |
| `state` | `logic` | 36 | high (0.75) | 2 | 4 | 2 |
| `side_effects` | `logic` | 72 | high (0.75) | 2 | 4 | 2 |
| `control_flow` | `logic` | 60 | high (0.875) | 3 | 6 | 2 |
| `ordering` | `logic` | 18 | high (0.75) | 2 | 4 | 3 |
| `concurrency` | `logic` | 0 | low (0.25) | 1 | 2 | 2 |
| `separation` | `context` | 72 | insufficient (0) | 0 | 0 | 0 |
| `radius` | `context` | 36 | low (0.25) | 1 | 2 | 1 |
| `dependencies` | `context` | 36 | low (0.25) | 1 | 2 | 1 |
| `locality` | `context` | 86 | low (0.25) | 1 | 2 | 1 |
| `context_economy` | `context` | 86 | low (0.25) | 1 | 2 | 1 |
| `naming` | `meaning` | 86 | low (0.25) | 1 | 2 | 1 |
| `language` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `shape` | `meaning` | 86 | low (0.25) | 1 | 2 | 2 |
| `state_vocabulary` | `meaning` | 86 | low (0.25) | 1 | 3 | 2 |
| `argument_patterns` | `meaning` | 72 | insufficient (0) | 0 | 0 | 0 |
| `idiomaticity` | `meaning` | 86 | low (0.25) | 1 | 2 | 2 |
| `boundaries` | `correctness` | 86 | low (0.25) | 1 | 2 | 2 |
| `contracts` | `correctness` | 24 | high (0.875) | 3 | 6 | 2 |
| `validation` | `correctness` | 18 | high (0.75) | 2 | 4 | 2 |
| `tests` | `correctness` | 93 | high (0.75) | 2 | 4 | 3 |
| `demonstrations` | `correctness` | 86 | low (0.25) | 1 | 2 | 2 |
| `explainability` | `correctness` | 36 | high (0.75) | 2 | 4 | 2 |
| `orientation` | `continuity` | 86 | low (0.25) | 1 | 2 | 1 |
| `decision_memory` | `continuity` | 86 | low (0.25) | 1 | 2 | 1 |
| `current_state` | `continuity` | 36 | low (0.25) | 1 | 2 | 1 |
| `handoff` | `continuity` | 86 | low (0.25) | 1 | 3 | 2 |
| `architecture_docs` | `continuity` | 93 | high (0.75) | 2 | 5 | 2 |

