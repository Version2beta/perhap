import test from 'node:test'
import assert from 'node:assert/strict'
import { build } from 'esbuild'
import { fileURLToPath } from 'node:url'
import type { ModelState, ModelUpdate } from './model-state.ts'

test('callback and global event preserve the same optimistic state on equal-frontier idle close', async () => {
  // Bundle the public API with an in-memory transport: no browser or NATS server required.
  const bundled = await build({
    stdin: {
      contents: `export * from './core.ts'; export { transport } from 'nats.ws'`,
      resolveDir: fileURLToPath(new URL('.', import.meta.url)),
    },
    bundle: true,
    write: false,
    format: 'esm',
    platform: 'node',
    plugins: [{
      name: 'in-memory-nats',
      setup(build) {
        build.onResolve({ filter: /^nats\.ws$/ }, () => ({ path: 'nats.ws', namespace: 'test' }))
        build.onLoad({ filter: /.*/, namespace: 'test' }, () => ({ contents: `
          export const transport = {
            subscribe(_subject, options) {
              this.callback = options.callback;
              return { unsubscribe() {} };
            },
            request() { return new Promise(resolve => { this.hydrate = resolve; }); },
            publish() {},
          };
          export async function connect() { return transport; }
          export function StringCodec() {
            return { encode: s => new TextEncoder().encode(s), decode: b => new TextDecoder().decode(b) };
          }
          export function credsAuthenticator() { throw new Error('unexpected credentials'); }
        ` }))
      },
    }],
  })
  const api = await import(`data:text/javascript;base64,${Buffer.from(bundled.outputFiles[0].text).toString('base64')}`)
  const previousWindow = Object.getOwnPropertyDescriptor(globalThis, 'window')
  const events: Array<{ reduction: unknown; pending: boolean; valid: boolean }> = []
  const target = new EventTarget()
  target.addEventListener('perhap:model', event => events.push((event as CustomEvent).detail))
  Object.defineProperty(globalThis, 'window', { configurable: true, value: target })
  let unwatch: (() => void) | undefined
  try {
    await api.configure({ server: 'ws://test', credentials: '' })
    const states: ModelState[] = []
    unwatch = api.watch('store', 'cart-1', (state: ModelState) => states.push(state),
      () => ({ count: 2 }))
    const update: ModelUpdate = {
      entity_id: 'cart-1', context: 'store', domain: 'cart', snapshot_uuid: '',
      window: [], reduced_through: '001', pending: true, valid: true,
      reduction: { count: 1 },
    }
    const message = (value: ModelUpdate) => ({ data: new TextEncoder().encode(JSON.stringify(value)) })
    api.transport.hydrate(message(update))
    await new Promise(resolve => setImmediate(resolve))
    assert.deepEqual(states[states.length - 1]?.reduction, { count: 1 })

    api.tell('store', 'cart-1', 'increment', {})
    assert.deepEqual(states[states.length - 1]?.reduction, { count: 2 })
    // An equal-frontier payload must not overwrite reduction or validity.
    api.transport.callback(null, message({ ...update, pending: false, valid: false }))
    const closed = states[states.length - 1]!
    assert.equal(closed.backendPending, false)
    assert.equal(closed.pending, true)
    assert.equal(closed.valid, true)
    assert.deepEqual(closed.reduction, { count: 2 })
    assert.equal(states.length, 3)
    assert.equal(events.length, states.length)
    states.forEach((state, i) => {
      assert.deepEqual(events[i].reduction, state.reduction)
      assert.equal(events[i].pending, state.pending)
      assert.equal(events[i].valid, state.valid)
    })
  } finally {
    unwatch?.()
    if (previousWindow) Object.defineProperty(globalThis, 'window', previousWindow)
    else Reflect.deleteProperty(globalThis, 'window')
  }
})
