# Complexity Run Rules

These rules exist to stop stale evidence reuse from quietly turning into fake certainty.

## Required run declaration

Every packet must declare one evidence mode:

- `fresh`
- `diff_from_baseline`
- `reused_exact_tree`

Meaning:

- `fresh`: observations were rebuilt from current code, docs, tests, and validation.
- `diff_from_baseline`: observations were updated from an existing baseline packet for a changed tree.
- `reused_exact_tree`: an existing packet is being reused without rescoring because the scored tree is identical.

## Tree fingerprint

Every scoring run records a scope fingerprint built from:

- current `HEAD`
- `git status --short`
- scope file list hash from `rg --files`
- scope content hash from the bytes of the scope files
- protocol version

This is emitted in the result `runManifest`.

## Reuse rules

### `fresh`

- allowed on any tree
- no reuse metadata required

### `diff_from_baseline`

- must include `reuse.baselinePath`
- may use a prior packet as a starting point
- does not claim that previous observations are still valid by default

### `reused_exact_tree`

- must include `reuse.expectedFingerprint`
- scorer compares that fingerprint to the current scope fingerprint
- mismatch is a hard failure

If the fingerprint does not match, the packet cannot be presented as an exact reuse.

## Read ledger

Every result includes a read ledger derived from packet evidence:

- code anchors
- test anchors
- docs anchors
- validation commands

This does not prove what the scorer process literally opened in the terminal. It does make the evidence surface explicit and auditable.

## Practical use

Recommended workflow:

1. Start from the canonical baseline packet.
2. If the tree changed, create a `diff_from_baseline` packet or a fresh packet.
3. Only use `reused_exact_tree` when you already have a matching fingerprint.
4. Treat mismatched exact reuse as invalid, not low-confidence.
