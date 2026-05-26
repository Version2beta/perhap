import { spawnSync } from 'node:child_process'
import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '..')
const protocolPath = path.join(repoRoot, '.complexity', 'protocol.json')
const evidenceModes = new Set(['fresh', 'diff_from_baseline', 'reused_exact_tree'])

export function loadProtocol(customPath = protocolPath) {
  return JSON.parse(readFileSync(customPath, 'utf8'))
}

export function flattenFacets(protocol) {
  return Object.values(protocol.families).flat()
}

export function displayScore(latent, mapping) {
  const clamped = Math.max(-1, Math.min(1, latent))
  const neutral = mapping.neutral
  if (clamped >= 0) {
    return Math.round(neutral + clamped * (100 - neutral))
  }
  return Math.round(neutral + clamped * neutral)
}

export function confidenceLabel(confidence) {
  if (confidence < 0.25) return 'insufficient'
  if (confidence < 0.5) return 'low'
  if (confidence < 0.75) return 'medium'
  return 'high'
}

export function computeFacetConfidence(count, anchors, kinds, scopeKind, protocol) {
  const targets = protocol.confidenceTargets[scopeKind]
  const weights = protocol.confidenceWeights

  const countScore = Math.min(count / targets.observations, 1)
  const anchorScore = Math.min(anchors / targets.anchors, 1)
  const kindScore = Math.min(kinds / targets.kinds, 1)

  let confidence =
    weights.observations * countScore +
    weights.anchors * anchorScore +
    weights.kinds * kindScore

  for (const cap of protocol.confidenceCaps) {
    if (cap.metric === 'observations' && count < cap.threshold) {
      confidence = Math.min(confidence, cap.cap)
    }
    if (cap.metric === 'anchors' && anchors < cap.threshold) {
      confidence = Math.min(confidence, cap.cap)
    }
    if (cap.metric === 'kinds' && kinds < cap.threshold) {
      confidence = Math.min(confidence, cap.cap)
    }
  }

  return Number(confidence.toFixed(3))
}

export function computeFacetScore(impacts) {
  if (impacts.length === 0) {
    return {
      raw: 0,
      relevantObservations: 0,
      maxAbs: 0,
      latent: 0,
    }
  }

  const raw = impacts.reduce((sum, value) => sum + value, 0)
  const maxAbs = 2 * impacts.length
  return {
    raw,
    relevantObservations: impacts.length,
    maxAbs,
    latent: Number((raw / maxAbs).toFixed(3)),
  }
}

export function computeDimensionScore(facetResults, weights, mapping) {
  let numerator = 0
  let denominator = 0
  let confidenceNumerator = 0
  let contributionValues = []

  for (const [facet, weight] of Object.entries(weights)) {
    const facetResult = facetResults[facet]
    if (!facetResult) {
      continue
    }
    numerator += weight * facetResult.latent
    denominator += Math.abs(weight)
    confidenceNumerator += Math.abs(weight) * facetResult.confidence
    contributionValues.push({
      facet,
      weight,
      contribution: Number((weight * facetResult.latent).toFixed(3)),
    })
  }

  const latent = denominator === 0 ? 0 : Number((numerator / denominator).toFixed(3))
  const confidence = denominator === 0 ? 0 : Number((confidenceNumerator / denominator).toFixed(3))
  contributionValues = contributionValues
    .sort((a, b) => Math.abs(b.contribution) - Math.abs(a.contribution))
    .slice(0, 5)

  return {
    latent,
    display: displayScore(latent, mapping),
    confidence,
    confidenceLabel: confidenceLabel(confidence),
    topContributors: contributionValues,
  }
}

function parseArgs(argv) {
  const args = { format: 'markdown' }
  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i]
    if (arg === '--input') {
      args.input = argv[i + 1]
      i += 1
    } else if (arg === '--format') {
      args.format = argv[i + 1]
      i += 1
    } else if (arg === '--help' || arg === '-h') {
      args.help = true
    }
  }
  return args
}

function usage() {
  return [
    'Usage: node scripts/complexity-score.mjs --input <packet.json> [--format markdown|json]',
    '',
    'Scores an observation packet against the repo-local complexity protocol.',
    'Packets must declare an evidence mode: fresh, diff_from_baseline, or reused_exact_tree.',
  ].join('\n')
}

function listScopeFiles(roots) {
  const args = ['--files', ...roots]
  const result = spawnSync('rg', args, {
    cwd: repoRoot,
    encoding: 'utf8',
  })
  if (result.status !== 0) {
    throw new Error(result.stderr.trim() || 'failed to enumerate scope files with rg')
  }
  return result.stdout
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean)
}

function runGit(args) {
  const result = spawnSync('git', args, {
    cwd: repoRoot,
    encoding: 'utf8',
  })
  if (result.status !== 0) {
    throw new Error(result.stderr.trim() || `git ${args.join(' ')} failed`)
  }
  return result.stdout.trim()
}

function normalizeMultiline(value) {
  if (!value) {
    return ''
  }
  return value
    .split('\n')
    .map(line => line.trimEnd())
    .filter(Boolean)
    .sort()
    .join('\n')
}

function hashStrings(values) {
  const hash = createHash('sha256')
  for (const value of values) {
    hash.update(value)
    hash.update('\n')
  }
  return hash.digest('hex')
}

function hashScopeContents(scopeFiles) {
  const hash = createHash('sha256')
  for (const relativePath of scopeFiles) {
    hash.update(relativePath)
    hash.update('\0')
    hash.update(readFileSync(path.join(repoRoot, relativePath)))
    hash.update('\0')
  }
  return hash.digest('hex')
}

export function computeTreeFingerprint(roots, protocolVersion = null) {
  const scopeFiles = listScopeFiles(roots)
  const headCommit = runGit(['rev-parse', 'HEAD'])
  const globalStatusShort = normalizeMultiline(runGit(['status', '--short']))
  const scopeStatusShort = normalizeMultiline(runGit(['status', '--short', '--', ...roots]))
  const scopeFileListHash = hashStrings(scopeFiles)
  const scopeContentHash = hashScopeContents(scopeFiles)
  const fingerprintInputs = [
    headCommit,
    globalStatusShort,
    scopeStatusShort,
    scopeFileListHash,
    scopeContentHash,
  ]

  if (protocolVersion !== null) {
    fingerprintInputs.push(String(protocolVersion))
  }

  return {
    headCommit,
    globalStatusShort,
    scopeStatusShort,
    scopeFileCount: scopeFiles.length,
    scopeFileListHash,
    scopeContentHash,
    treeFingerprint: hashStrings(fingerprintInputs),
    scopeFiles,
  }
}

function anchorPath(anchor) {
  const colonIndex = anchor.indexOf(':')
  if (colonIndex === -1) {
    return anchor
  }
  return anchor.slice(0, colonIndex)
}

function buildReadLedger(observations) {
  const ledger = {
    codeAnchors: new Set(),
    testAnchors: new Set(),
    docsAnchors: new Set(),
    validationCommands: new Set(),
  }

  for (const observation of observations) {
    for (const evidence of observation.evidence) {
      if (evidence.kind === 'code') {
        ledger.codeAnchors.add(evidence.anchor)
      } else if (evidence.kind === 'tests') {
        ledger.testAnchors.add(evidence.anchor)
      } else if (evidence.kind === 'docs/contracts') {
        ledger.docsAnchors.add(evidence.anchor)
      } else if (evidence.kind === 'validation/runtime') {
        ledger.validationCommands.add(evidence.anchor)
      }
    }
  }

  return {
    codeAnchors: Array.from(ledger.codeAnchors).sort(),
    testAnchors: Array.from(ledger.testAnchors).sort(),
    docsAnchors: Array.from(ledger.docsAnchors).sort(),
    validationCommands: Array.from(ledger.validationCommands).sort(),
  }
}

function findFamilyByFacet(protocol, facet) {
  for (const [family, facets] of Object.entries(protocol.families)) {
    if (facets.includes(facet)) {
      return family
    }
  }
  return null
}

function validatePacket(packet, protocol) {
  const facets = new Set(flattenFacets(protocol))
  const evidenceKinds = new Set(protocol.evidenceKinds)
  const ids = new Set()

  if (!packet.title || typeof packet.title !== 'string') {
    throw new Error('packet.title is required')
  }
  if (!evidenceModes.has(packet.evidenceMode)) {
    throw new Error(`packet.evidenceMode must be one of: ${Array.from(evidenceModes).join(', ')}`)
  }
  if (!packet.scope || typeof packet.scope !== 'object') {
    throw new Error('packet.scope is required')
  }
  if (!protocol.confidenceTargets[packet.scope.kind]) {
    throw new Error(`unsupported scope kind: ${packet.scope.kind}`)
  }
  if (!Array.isArray(packet.scope.roots) || packet.scope.roots.length === 0) {
    throw new Error('packet.scope.roots must be a non-empty array')
  }
  if (!Array.isArray(packet.observations) || packet.observations.length === 0) {
    throw new Error('packet.observations must be a non-empty array')
  }
  if (packet.evidenceMode === 'diff_from_baseline' && !packet.reuse?.baselinePath) {
    throw new Error('diff_from_baseline packets must provide reuse.baselinePath')
  }
  if (packet.evidenceMode === 'reused_exact_tree' && !packet.reuse?.expectedFingerprint) {
    throw new Error('reused_exact_tree packets must provide reuse.expectedFingerprint')
  }

  for (const observation of packet.observations) {
    if (ids.has(observation.id)) {
      throw new Error(`duplicate observation id: ${observation.id}`)
    }
    ids.add(observation.id)
    if (!Array.isArray(observation.evidence) || observation.evidence.length === 0) {
      throw new Error(`observation ${observation.id} must include evidence`)
    }
    if (!observation.impacts || typeof observation.impacts !== 'object') {
      throw new Error(`observation ${observation.id} must include impacts`)
    }

    for (const [facet, impact] of Object.entries(observation.impacts)) {
      if (!facets.has(facet)) {
        throw new Error(`observation ${observation.id} references unknown facet: ${facet}`)
      }
      if (!Number.isInteger(impact) || impact < -2 || impact > 2 || impact === 0) {
        throw new Error(`observation ${observation.id} has invalid impact for ${facet}: ${impact}`)
      }
    }

    for (const evidence of observation.evidence) {
      if (!evidenceKinds.has(evidence.kind)) {
        throw new Error(`observation ${observation.id} has invalid evidence kind: ${evidence.kind}`)
      }
    }
  }
}

function enforceEvidenceMode(packet, runManifest) {
  if (packet.evidenceMode !== 'reused_exact_tree') {
    return
  }
  if (packet.reuse.expectedFingerprint !== runManifest.treeFingerprint) {
    throw new Error(
      [
        'reused_exact_tree packet fingerprint mismatch:',
        `expected ${packet.reuse.expectedFingerprint}`,
        `current  ${runManifest.treeFingerprint}`,
      ].join('\n')
    )
  }
}

export function scorePacket(packet, protocol = loadProtocol()) {
  validatePacket(packet, protocol)

  const runManifest = computeTreeFingerprint(packet.scope.roots, protocol.version ?? null)
  enforceEvidenceMode(packet, runManifest)

  const scopeFileSet = new Set(runManifest.scopeFiles)
  const allFacets = flattenFacets(protocol)
  const facetResults = {}
  const missingEvidencePaths = new Set()

  for (const facet of allFacets) {
    const relevant = packet.observations.filter(observation => observation.impacts[facet] !== undefined)
    const impacts = relevant.map(observation => observation.impacts[facet])
    const score = computeFacetScore(impacts)
    const anchors = new Set()
    const kinds = new Set()
    const observationIds = []

    for (const observation of relevant) {
      observationIds.push(observation.id)
      for (const evidence of observation.evidence) {
        anchors.add(evidence.anchor)
        kinds.add(evidence.kind)
        if (evidence.kind !== 'validation/runtime') {
          const evidencePath = anchorPath(evidence.anchor)
          if (!scopeFileSet.has(evidencePath)) {
            missingEvidencePaths.add(evidencePath)
          }
        }
      }
    }

    const confidence = computeFacetConfidence(
      relevant.length,
      anchors.size,
      kinds.size,
      packet.scope.kind,
      protocol
    )

    facetResults[facet] = {
      family: findFamilyByFacet(protocol, facet),
      ...score,
      display: displayScore(score.latent, protocol.displayMapping),
      confidence,
      confidenceLabel: confidenceLabel(confidence),
      anchors: anchors.size,
      evidenceKinds: kinds.size,
      observationIds,
    }
  }

  const familyResults = {}
  for (const [family, facets] of Object.entries(protocol.families)) {
    const latent = facets.reduce((sum, facet) => sum + facetResults[facet].latent, 0) / facets.length
    const confidence = facets.reduce((sum, facet) => sum + facetResults[facet].confidence, 0) / facets.length
    const observedFacets = facets.filter(facet => facetResults[facet].relevantObservations > 0).length

    familyResults[family] = {
      latent: Number(latent.toFixed(3)),
      display: displayScore(latent, protocol.displayMapping),
      confidence: Number(confidence.toFixed(3)),
      confidenceLabel: confidenceLabel(confidence),
      observedFacets,
      totalFacets: facets.length,
    }
  }

  const dimensionResults = {}
  for (const [dimension, weights] of Object.entries(protocol.dimensionWeights)) {
    dimensionResults[dimension] = computeDimensionScore(
      facetResults,
      weights,
      protocol.displayMapping
    )
  }

  const observedFiles = new Set()
  for (const observation of packet.observations) {
    for (const evidence of observation.evidence) {
      const evidencePath = anchorPath(evidence.anchor)
      if (scopeFileSet.has(evidencePath)) {
        observedFiles.add(evidencePath)
      }
    }
  }

  return {
    packet: {
      title: packet.title,
      evidenceMode: packet.evidenceMode,
      reuse: packet.reuse ?? null,
      scope: packet.scope,
    },
    runManifest: {
      protocolVersion: protocol.version ?? null,
      headCommit: runManifest.headCommit,
      globalStatusShort: runManifest.globalStatusShort,
      scopeStatusShort: runManifest.scopeStatusShort,
      scopeFileListHash: runManifest.scopeFileListHash,
      scopeContentHash: runManifest.scopeContentHash,
      treeFingerprint: runManifest.treeFingerprint,
      readLedger: buildReadLedger(packet.observations),
    },
    coverage: {
      scopeFileCount: runManifest.scopeFiles.length,
      observedFileCount: observedFiles.size,
      observationCount: packet.observations.length,
      facetCount: allFacets.length,
      missingEvidencePaths: Array.from(missingEvidencePaths).sort(),
    },
    dimensions: dimensionResults,
    families: familyResults,
    facets: facetResults,
    observations: packet.observations,
  }
}

export function renderMarkdown(result) {
  const lines = []
  lines.push(`# ${result.packet.title}`)
  lines.push('')
  lines.push(`- Evidence mode: \`${result.packet.evidenceMode}\``)
  lines.push(`- Scope kind: \`${result.packet.scope.kind}\``)
  lines.push(`- Scope roots: ${result.packet.scope.roots.map(root => `\`${root}\``).join(', ')}`)
  lines.push(`- HEAD commit: \`${result.runManifest.headCommit.slice(0, 12)}\``)
  lines.push(`- Tree fingerprint: \`${result.runManifest.treeFingerprint.slice(0, 16)}\``)
  lines.push(`- Scope files via \`rg --files\`: ${result.coverage.scopeFileCount}`)
  lines.push(`- Observation count: ${result.coverage.observationCount}`)
  lines.push(`- Files touched by evidence: ${result.coverage.observedFileCount}`)
  lines.push('')

  if (result.packet.reuse?.baselinePath) {
    lines.push(`- Baseline reference: \`${result.packet.reuse.baselinePath}\``)
    lines.push('')
  }

  if (result.coverage.missingEvidencePaths.length > 0) {
    lines.push('## Scope Warnings')
    lines.push('')
    for (const missingPath of result.coverage.missingEvidencePaths) {
      lines.push(`- Evidence path not found in scope: \`${missingPath}\``)
    }
    lines.push('')
  }

  if (result.runManifest.scopeStatusShort) {
    lines.push('## Worktree')
    lines.push('')
    lines.push('```text')
    lines.push(result.runManifest.scopeStatusShort)
    lines.push('```')
    lines.push('')
  }

  lines.push('## Read Ledger')
  lines.push('')
  lines.push(`- Code anchors: ${result.runManifest.readLedger.codeAnchors.length}`)
  lines.push(`- Test anchors: ${result.runManifest.readLedger.testAnchors.length}`)
  lines.push(`- Docs anchors: ${result.runManifest.readLedger.docsAnchors.length}`)
  lines.push(`- Validation commands: ${result.runManifest.readLedger.validationCommands.length}`)
  lines.push('')

  lines.push('## Dimensions')
  lines.push('')
  lines.push('| Dimension | Score | Confidence | Top contributors |')
  lines.push('| --- | --- | --- | --- |')
  for (const [dimension, data] of Object.entries(result.dimensions)) {
    const contributors = data.topContributors
      .map(item => `${item.facet} (${item.contribution >= 0 ? '+' : ''}${item.contribution})`)
      .join(', ')
    lines.push(`| \`${dimension}\` | ${data.display} | ${data.confidenceLabel} (${data.confidence}) | ${contributors} |`)
  }
  lines.push('')

  lines.push('## Families')
  lines.push('')
  lines.push('| Family | Score | Confidence | Observed facets |')
  lines.push('| --- | --- | --- | --- |')
  for (const [family, data] of Object.entries(result.families)) {
    lines.push(`| \`${family}\` | ${data.display} | ${data.confidenceLabel} (${data.confidence}) | ${data.observedFacets}/${data.totalFacets} |`)
  }
  lines.push('')

  lines.push('## Facets')
  lines.push('')
  lines.push('| Facet | Family | Score | Confidence | Obs | Anchors | Kinds |')
  lines.push('| --- | --- | --- | --- | --- | --- | --- |')
  for (const [facet, data] of Object.entries(result.facets)) {
    lines.push(`| \`${facet}\` | \`${data.family}\` | ${data.display} | ${data.confidenceLabel} (${data.confidence}) | ${data.relevantObservations} | ${data.anchors} | ${data.evidenceKinds} |`)
  }
  lines.push('')

  return lines.join('\n')
}

function main() {
  const args = parseArgs(process.argv.slice(2))
  if (args.help || !args.input) {
    console.log(usage())
    process.exit(args.help ? 0 : 1)
  }

  const packetPath = path.resolve(repoRoot, args.input)
  const packet = JSON.parse(readFileSync(packetPath, 'utf8'))
  const result = scorePacket(packet)

  if (args.format === 'json') {
    console.log(JSON.stringify(result, null, 2))
    return
  }

  console.log(renderMarkdown(result))
}

if (process.argv[1] === __filename) {
  main()
}
