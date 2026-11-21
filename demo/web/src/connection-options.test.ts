import test from 'node:test'
import assert from 'node:assert/strict'
import { connectionOptions } from './connection-options.ts'

test('anonymous connection omits authentication; malformed credentials are not ignored', () => {
  assert.deepEqual(connectionOptions('ws://localhost:4223'), { servers: 'ws://localhost:4223' })
  const authenticator = connectionOptions('ws://localhost:4223', 'not a creds file').authenticator
  assert.equal(typeof authenticator, 'function')
  assert.throws(() => (authenticator as (nonce?: string) => unknown)('nonce'))
})
