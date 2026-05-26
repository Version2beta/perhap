import { connect, StringCodec, type NatsConnection, type Subscription } from 'nats.ws'

import {
  applyConfirmedUpdate,
  applyOptimisticUpdate,
  createModelState,
  drainBufferedUpdates,
  sortBufferedUpdates,
  type FrontendReducer,
  type ModelState,
  type ModelUpdate,
} from './model-state'
import { newUUIDv6 } from './uuid'

interface WatchHandle {
  context: string
  entityId: string
  sub: Subscription
  onUpdate: (state: ModelState) => void
  lastState: ModelState
  buffer: ModelUpdate[]
  hydrated: boolean
  pendingEventIds: string[]
  reducer?: FrontendReducer
}

const watchRegistry = new Map<string, WatchHandle>()
const sc = StringCodec()

let nc: NatsConnection | null = null
let readyResolve: (() => void) | null = null
let readySettled = false

export const ready: Promise<void> = new Promise(resolve => {
  readyResolve = () => {
    if (readySettled) return
    readySettled = true
    resolve()
  }
})

export async function configure(opts?: { server?: string; credentials?: string }) {
  const server = opts?.server
    ?? document.querySelector<HTMLMetaElement>('meta[name="perhap-server"]')?.content
    ?? 'ws://localhost:4223'

  if (nc) return

  console.log('[perhap] connecting to', server)
  try {
    nc = await connect({ servers: server })
    console.log('[perhap] connected to', server)
    readyResolve?.()
  } catch (err) {
    console.error('[perhap] connection failed:', err)
  }
}

export function watch(
  context: string,
  entityId: string,
  onUpdate: (state: ModelState) => void,
  reducer?: FrontendReducer
): () => void {
  if (!nc) {
    console.warn('[perhap] watch() called before nc ready — call after ready resolves')
    return () => {}
  }

  console.log(`[perhap] watch ${context}/${entityId}`)
  const handle = createWatchHandle(context, entityId, onUpdate, reducer)
  registerWatch(handle)
  subscribeToModelUpdates(nc, handle)
  requestHydration(nc, handle)

  return () => {
    unregisterWatch(handle)
  }
}

export function tell(
  context: string,
  entityId: string,
  eventType: string,
  data: unknown,
  opts?: { causationId?: string }
): void {
  if (!nc) {
    console.warn('[perhap] tell() called before nc ready')
    return
  }

  const eventId = newUUIDv6()
  const event = {
    id: eventId,
    entity_id: entityId,
    context,
    type: eventType,
    causation_id: opts?.causationId ?? null,
    correlation_id: newUUIDv6(),
    data,
  }

  console.log(`[perhap] tell: ${context}/${entityId} ${eventType} id=${eventId}`, data)
  const handle = watchRegistry.get(watchKey(context, entityId))
  if (handle) {
    applyOptimisticUpdateAndNotify(handle, eventId, eventType, data)
  } else {
    console.log('[perhap] no local watch to update optimistically')
  }

  const subject = `perhap.${context}.${entityId}.${eventType}`
  console.log(`[perhap] publish → ${subject}`)
  nc.publish(subject, sc.encode(JSON.stringify(event)))
}

function applyConfirmedUpdateAndNotify(handle: WatchHandle, update: ModelUpdate) {
  const transition = applyConfirmedUpdate(handle.lastState, handle.pendingEventIds, update)
  if (transition.stale) {
    console.warn(`[perhap] DROPPED stale update: update.rt=${update.reduced_through || '(empty)'} <= prev.rt=${handle.lastState.reducedThrough}`)
    return
  }

  console.log(`[perhap] applyUpdate: rt=${update.reduced_through || '(empty)'} pending=${update.pending}`, update.reduction)
  handle.pendingEventIds = transition.pendingEventIds
  handle.lastState = transition.state
  handle.onUpdate(handle.lastState)
  dispatchModelEvent(update.entity_id, update.context, update.reduction, handle.lastState.pending, update.valid)
}

function createWatchHandle(
  context: string,
  entityId: string,
  onUpdate: (state: ModelState) => void,
  reducer?: FrontendReducer
): WatchHandle {
  return {
    context,
    entityId,
    sub: null as unknown as Subscription,
    onUpdate,
    lastState: createModelState(context, entityId),
    buffer: [],
    hydrated: false,
    pendingEventIds: [],
    reducer,
  }
}

function registerWatch(handle: WatchHandle) {
  watchRegistry.set(watchKey(handle.context, handle.entityId), handle)
}

function unregisterWatch(handle: WatchHandle) {
  handle.sub.unsubscribe()
  watchRegistry.delete(watchKey(handle.context, handle.entityId))
}

function subscribeToModelUpdates(connection: NatsConnection, handle: WatchHandle) {
  const subject = `perhap.${handle.context}.${handle.entityId}._model`
  console.log(`[perhap] subscribing to ${subject}`)
  handle.sub = connection.subscribe(subject, {
    callback: (_err, msg) => {
      handleModelUpdate(handle, JSON.parse(sc.decode(msg.data)) as ModelUpdate)
    }
  })
}

function handleModelUpdate(handle: WatchHandle, update: ModelUpdate) {
  console.log(`[perhap] _model msg: rt=${update.reduced_through || '(empty)'} pending=${update.pending} hydrated=${handle.hydrated}`)
  if (!handle.hydrated) {
    console.log('[perhap] buffering (not yet hydrated)')
    handle.buffer.push(update)
    return
  }
  applyConfirmedUpdateAndNotify(handle, update)
}

function requestHydration(connection: NatsConnection, handle: WatchHandle) {
  const hydSubject = `_perhap.${handle.context}.${handle.entityId}.model`
  console.log(`[perhap] hydration request → ${hydSubject}`)
  connection.request(hydSubject, sc.encode(''), { timeout: 5000 })
    .then(msg => finishHydration(handle, JSON.parse(sc.decode(msg.data)) as ModelUpdate))
    .catch(err => recoverBufferedState(handle, err))
}

function finishHydration(handle: WatchHandle, hydration: ModelUpdate) {
  console.log(`[perhap] hydration response: rt=${hydration.reduced_through || '(empty)'} items=${JSON.stringify(hydration.reduction)}`)
  handle.hydrated = true
  applyConfirmedUpdateAndNotify(handle, hydration)
  applyBufferedUpdates(handle, drainBufferedUpdates(handle.buffer, hydration.reduced_through), 'buffer drained')
}

function recoverBufferedState(handle: WatchHandle, err: unknown) {
  console.warn('[perhap] hydration failed:', err)
  handle.hydrated = true
  applyBufferedUpdates(handle, sortBufferedUpdates(handle.buffer), 'hydration timeout')
}

function applyBufferedUpdates(handle: WatchHandle, updates: ModelUpdate[], reason: 'buffer drained' | 'hydration timeout') {
  const bufferedCount = handle.buffer.length
  handle.buffer = []
  if (updates.length === 0) return

  if (reason === 'buffer drained') {
    console.log(`[perhap] buffer drained: ${updates.length}/${bufferedCount} applied`)
  } else {
    console.log(`[perhap] hydration timeout — applying ${updates.length} buffered update(s) instead`)
  }

  for (const update of updates) {
    applyConfirmedUpdateAndNotify(handle, update)
  }
}

function applyOptimisticUpdateAndNotify(
  handle: WatchHandle,
  eventId: string,
  eventType: string,
  data: unknown,
) {
  const transition = applyOptimisticUpdate(
    handle.lastState,
    handle.pendingEventIds,
    eventId,
    eventType,
    data,
    handle.reducer,
  )
  handle.pendingEventIds = transition.pendingEventIds
  handle.lastState = transition.state
  console.log(`[perhap] optimistic pending tracked: ${handle.pendingEventIds.length} event(s) in flight`)
  handle.onUpdate(transition.state)
  dispatchModelEvent(handle.entityId, handle.context, transition.state.reduction, transition.state.pending, transition.state.valid)
}

function dispatchModelEvent(
  entityId: string,
  context: string,
  reduction: Record<string, unknown> | null,
  pending: boolean,
  valid: boolean
) {
  window.dispatchEvent(new CustomEvent('perhap:model', {
    detail: { entity_id: entityId, context, reduction, pending, valid }
  }))
}

function watchKey(context: string, entityId: string): string {
  return `${context}/${entityId}`
}
