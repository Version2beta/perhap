import test from 'node:test'
import assert from 'node:assert/strict'

import {
  computeDimensionScore,
  computeFacetConfidence,
  computeFacetScore,
  displayScore,
  loadProtocol,
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
