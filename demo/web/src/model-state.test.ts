import test from 'node:test'
import assert from 'node:assert/strict'

import {
  applyConfirmedUpdate,
  applyOptimisticUpdate,
  createModelState,
  drainBufferedUpdates,
  sortBufferedUpdates,
  type ModelUpdate,
} from './model-state.ts'

function update(reducedThrough: string, pending = false, reduction: Record<string, unknown> = {}): ModelUpdate {
  return {
    entity_id: 'cart-1',
    context: 'store',
    domain: 'cart',
    snapshot_uuid: '',
    window: [],
    reduced_through: reducedThrough,
    pending,
    valid: true,
    reduction,
  }
}

test('optimistic updates keep pending true until a confirming update arrives', () => {
  const state = createModelState('store', 'cart-1')
  const optimistic = applyOptimisticUpdate(
    { ...state, reduction: { count: 1 } },
    [],
    '00000000-0000-6000-8000-000000000001',
    'item_added',
    { count: 2 },
    reduction => ({ ...reduction, count: 2 }),
  )
  assert.equal(optimistic.state.pending, true)
  assert.deepEqual(optimistic.pendingEventIds, ['00000000-0000-6000-8000-000000000001'])

  const confirmed = applyConfirmedUpdate(
    optimistic.state,
    optimistic.pendingEventIds,
    update('00000000-0000-6000-8000-000000000001', false, { count: 2 }),
  )
  assert.equal(confirmed.state.pending, false)
  assert.deepEqual(confirmed.pendingEventIds, [])
})

test('confirmed updates that are older than current state are dropped', () => {
  const current = {
    ...createModelState('store', 'cart-1'),
    reducedThrough: '00000000-0000-6000-8000-000000000010',
  }
  const transition = applyConfirmedUpdate(current, [], update('00000000-0000-6000-8000-000000000009'))
  assert.equal(transition.stale, true)
  assert.equal(transition.state.reducedThrough, current.reducedThrough)
})

test('buffer helpers keep only newer updates and sort them oldest-first', () => {
  const buffered = [
    update('00000000-0000-6000-8000-000000000020'),
    update('00000000-0000-6000-8000-000000000011'),
    update('00000000-0000-6000-8000-000000000015'),
  ]
  const drained = drainBufferedUpdates(buffered, '00000000-0000-6000-8000-000000000012')
  assert.deepEqual(
    sortBufferedUpdates(drained).map(item => item.reduced_through),
    [
      '00000000-0000-6000-8000-000000000015',
      '00000000-0000-6000-8000-000000000020',
    ]
  )
})
