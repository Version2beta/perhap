# Complexity Scoring Protocol

This repo uses a three-layer scoring model:

1. Observations are evidence-backed facts.
2. Facets are the scored layer.
3. Dimensions are computed from facet scores with a fixed weight matrix.

## File Universe

Scope files are whatever `rg --files` returns for the chosen roots. That means:

- ignore handling comes from the repo's existing `.gitignore` and related rules
- new and moved files are picked up automatically
- there is no separate scoring exclusion list by default

## Observation Packet

Packets live as JSON and follow [.complexity/packet.schema.json](/Users/rob/Development/perhap/.complexity/packet.schema.json:1).

The repo-local calibration set lives in [.complexity/CALIBRATION.md](/Users/rob/Development/perhap/.complexity/CALIBRATION.md:1), including the canonical baseline packet and a few scored reference examples.
Run freshness and reuse policy live in [.complexity/RUN_RULES.md](/Users/rob/Development/perhap/.complexity/RUN_RULES.md:1).

Each observation contains:

- `id`
- `fact`
- `evidence[]`
- `impacts`

`impacts` is keyed by fixed facet name. Each impact is an integer in `[-2, 2]`.

Each packet also declares:

- `evidenceMode`
- optional `reuse`

Supported evidence modes:

- `fresh`
- `diff_from_baseline`
- `reused_exact_tree`

Exact-tree reuse is validated against the current scope fingerprint.

Meaning of impact values:

- `-2`: strong reasoning cost
- `-1`: noticeable reasoning cost
- `0`: omitted rather than stored
- `+1`: clearly helps reasoning
- `+2`: unusually strong help

## Facet Scoring

For each facet:

```text
raw_f = sum(observation impacts for facet f)
max_f = 2 * count(relevant observations for facet f)
latent_f = raw_f / max_f
```

This yields one normalized facet score in `[-1, +1]`.

Equivalent form:

```text
latent_f = average(observation impacts for facet f) / 2
```

Important consequence: more observations increase confidence, not score range.

## Facet Confidence

Confidence is computed from:

- relevant observation count
- distinct evidence anchors
- distinct evidence kinds

For a given scope kind:

```text
count_score  = min(observations / target_observations, 1)
anchor_score = min(anchors / target_anchors, 1)
kind_score   = min(kinds / target_kinds, 1)

confidence =
  0.5 * count_score +
  0.3 * anchor_score +
  0.2 * kind_score
```

Hard caps prevent one noisy evidence source from pretending to be high confidence:

- fewer than 2 relevant observations: cap at `0.25`
- fewer than 2 distinct anchors: cap at `0.50`
- fewer than 2 evidence kinds: cap at `0.75`

## Dimensions

Dimensions are weighted averages over facet latent scores:

```text
dimension_latent(d) =
  sum(weight(d,f) * latent_f) / sum(abs(weight(d,f)))
```

The matrix is fixed in [.complexity/protocol.json](/Users/rob/Development/perhap/.complexity/protocol.json:1).

## Display Scores

Latent scores are mapped to `0-100` with a calibrated neutral point at `72`:

```text
if latent >= 0:
  display = 72 + latent * 28
else:
  display = 72 + latent * 72
```

So:

- `-1.0 -> 0`
- `0.0 -> 72`
- `+1.0 -> 100`

This keeps ordinary competent code in the low 70s instead of presenting it as a failing score.

## Running The Scorer

```bash
node scripts/complexity-score.mjs --input .complexity/baselines/perhap-whole-repo-v1.json
```

Useful flags:

- `--format markdown` (default)
- `--format json`

The scorer validates packet structure, enumerates the `rg` scope files, checks evidence anchors against that scope, computes a scope fingerprint, enforces evidence-mode reuse rules, and prints dimensions, families, facets, confidence, and a run manifest.

To refresh the frozen baseline and example outputs:

```bash
node scripts/complexity-freeze-calibration.mjs
```
