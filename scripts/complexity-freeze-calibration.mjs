import { mkdirSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { renderMarkdown, scorePacket } from './complexity-score.mjs'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '..')
const complexityRoot = path.join(repoRoot, '.complexity')
const outputRoot = path.join(complexityRoot, 'results')

function collectPackets(dir) {
  const root = path.join(complexityRoot, dir)
  return readdirSync(root)
    .filter(name => name.endsWith('.json'))
    .sort()
    .map(name => ({
      inputPath: path.join(root, name),
      outputStem: path.join(outputRoot, dir, name.replace(/\.json$/, '')),
    }))
}

function ensureDir(dir) {
  mkdirSync(dir, { recursive: true })
}

function freezePacket({ inputPath, outputStem }) {
  const packet = JSON.parse(readFileSync(inputPath, 'utf8'))
  const result = scorePacket(packet)
  ensureDir(path.dirname(outputStem))
  writeFileSync(`${outputStem}.json`, `${JSON.stringify(result, null, 2)}\n`)
  writeFileSync(`${outputStem}.md`, `${renderMarkdown(result)}\n`)
}

function main() {
  const packets = [
    ...collectPackets('baselines'),
    ...collectPackets('examples'),
  ]

  for (const packet of packets) {
    freezePacket(packet)
  }
}

main()
