import test from 'node:test'
import assert from 'node:assert/strict'

import { createUUIDv6Generator } from './uuid.ts'

test('uuid generator stays lexicographically monotonic within one source', () => {
  const generator = createUUIDv6Generator(
    () => 1_700_000_000_000,
    () => 0,
  )

  const ids = [generator(), generator(), generator()]
  assert.deepEqual([...ids].sort(), ids)
  assert.equal(new Set(ids).size, ids.length)
})
