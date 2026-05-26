import test from 'node:test'
import assert from 'node:assert/strict'

import {
  computeTreeFingerprint,
  computeDimensionScore,
  computeFacetConfidence,
  computeFacetScore,
  displayScore,
  loadProtocol,
  scorePacket,
} from './complexity-score.mjs'

const protocol = loadProtocol()

test('facet score normalizes summed impacts to a bounded latent score', () => {
  const score = computeFacetScore([2, 1, -1, 2])
  assert.equal(score.raw, 4)
  assert.equal(score.maxAbs, 8)
  assert.equal(score.latent, 0.5)
})

test('display score uses 72 as the neutral calibration point', () => {
  assert.equal(displayScore(0, protocol.displayMapping), 72)
  assert.equal(displayScore(1, protocol.displayMapping), 100)
  assert.equal(displayScore(-1, protocol.displayMapping), 0)
})

test('facet confidence is capped when evidence comes from one anchor and one kind', () => {
  const confidence = computeFacetConfidence(6, 1, 1, 'whole_repo', protocol)
  assert.equal(confidence, 0.5)
})

test('dimension scores are weighted averages over facet latent values', () => {
  const facets = {
    control_flow: { latent: -0.5, confidence: 0.8 },
    state: { latent: -0.25, confidence: 0.8 },
    ordering: { latent: -0.5, confidence: 0.6 },
    concurrency: { latent: -1, confidence: 0.7 },
    context_economy: { latent: 0.25, confidence: 0.9 },
    locality: { latent: 0.5, confidence: 0.8 },
    shape: { latent: 0.25, confidence: 0.8 },
    state_vocabulary: { latent: 0.5, confidence: 0.8 },
    side_effects: { latent: 0.25, confidence: 0.7 },
    naming: { latent: 0.5, confidence: 0.7 },
    idiomaticity: { latent: 0.5, confidence: 0.8 },
  }

  const score = computeDimensionScore(
    facets,
    protocol.dimensionWeights.cognitive_load,
    protocol.displayMapping,
  )

  assert.equal(score.latent, -0.094)
  assert.equal(score.display, 65)
  assert.equal(score.confidenceLabel, 'high')
})

test('tree fingerprint is stable across repeated reads of the same scope', () => {
  const first = computeTreeFingerprint(['sdk/go', 'demo/web/src'], protocol.version)
  const second = computeTreeFingerprint(['sdk/go', 'demo/web/src'], protocol.version)

  assert.equal(first.treeFingerprint, second.treeFingerprint)
  assert.equal(first.scopeContentHash, second.scopeContentHash)
  assert.equal(first.scopeFileListHash, second.scopeFileListHash)
})

test('reused_exact_tree packets fail on fingerprint mismatch', () => {
  const packet = {
    title: 'mismatch reuse packet',
    evidenceMode: 'reused_exact_tree',
    reuse: {
      expectedFingerprint: 'definitely-not-the-current-tree',
    },
    scope: {
      kind: 'targeted',
      roots: ['sdk/go'],
    },
    observations: [
      {
        id: 'O01',
        fact: 'The domain builder is well named.',
        evidence: [
          { anchor: 'sdk/go/domain.go:11-30', kind: 'code' },
        ],
        impacts: {
          naming: 1,
        },
      },
    ],
  }

  assert.throws(
    () => scorePacket(packet, protocol),
    /reused_exact_tree packet fingerprint mismatch/
  )
})

test('score packets emit a run manifest with evidence mode and read ledger', () => {
  const packet = {
    title: 'fresh packet manifest',
    evidenceMode: 'fresh',
    scope: {
      kind: 'targeted',
      roots: ['sdk/go'],
    },
    observations: [
      {
        id: 'O01',
        fact: 'The domain builder is well named and tested.',
        evidence: [
          { anchor: 'sdk/go/domain.go:11-30', kind: 'code' },
          { anchor: 'sdk/go/domain_test.go:19-133', kind: 'tests' },
          { anchor: 'go test ./...', kind: 'validation/runtime' },
        ],
        impacts: {
          naming: 1,
          tests: 1,
        },
      },
    ],
  }

  const result = scorePacket(packet, protocol)
  assert.equal(result.packet.evidenceMode, 'fresh')
  assert.ok(result.runManifest.treeFingerprint.length > 10)
  assert.deepEqual(result.runManifest.readLedger.codeAnchors, ['sdk/go/domain.go:11-30'])
  assert.deepEqual(result.runManifest.readLedger.testAnchors, ['sdk/go/domain_test.go:19-133'])
  assert.deepEqual(result.runManifest.readLedger.validationCommands, ['go test ./...'])
})
