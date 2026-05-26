# Complexity Calibration Set

This directory contains the repo-local calibration material for the complexity scorer.

Freshness and evidence reuse rules are in [RUN_RULES.md](/Users/rob/Development/perhap/.complexity/RUN_RULES.md:1).

## Canonical baseline

- [baselines/perhap-whole-repo-v1.json](/Users/rob/Development/perhap/.complexity/baselines/perhap-whole-repo-v1.json:1)

Use this packet as the default comparison point for future whole-repo refactor passes.

## Reference examples

- [examples/strong-structure.json](/Users/rob/Development/perhap/.complexity/examples/strong-structure.json:1)
  Positive example emphasizing good decomposition, docs, and validation.
- [examples/temporal-risk.json](/Users/rob/Development/perhap/.complexity/examples/temporal-risk.json:1)
  Risk-heavy example showing how concurrency, ordering, and contract drift pull scores down.
- [examples/insufficient-coverage.json](/Users/rob/Development/perhap/.complexity/examples/insufficient-coverage.json:1)
  Thin packet showing low-confidence output when evidence density is poor.

## Frozen results

Frozen result snapshots live under `.complexity/results/` and can be regenerated with:

```bash
node scripts/complexity-freeze-calibration.mjs
```

This writes both JSON and Markdown result views for every baseline and example packet.
