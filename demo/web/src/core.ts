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
  const handle: WatchHandle = {
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

  watchRegistry.set(watchKey(context, entityId), handle)

  const subject = `perhap.${context}.${entityId}._model`
  console.log(`[perhap] subscribing to ${subject}`)
  handle.sub = nc.subscribe(subject, {
    callback: (_err, msg) => {
      const update: ModelUpdate = JSON.parse(sc.decode(msg.data))
      console.log(`[perhap] _model msg: rt=${update.reduced_through || '(empty)'} pending=${update.pending} hydrated=${handle.hydrated}`)
      if (!handle.hydrated) {
        console.log('[perhap] buffering (not yet hydrated)')
        handle.buffer.push(update)
        return
      }
      applyConfirmedUpdateAndNotify(handle, update)
    }
  })

  const hydSubject = `_perhap.${context}.${entityId}.model`
  console.log(`[perhap] hydration request → ${hydSubject}`)
  nc.request(hydSubject, sc.encode(''), { timeout: 5000 })
    .then(msg => {
      const hydration: ModelUpdate = JSON.parse(sc.decode(msg.data))
      console.log(`[perhap] hydration response: rt=${hydration.reduced_through || '(empty)'} items=${JSON.stringify(hydration.reduction)}`)
      handle.hydrated = true
      applyConfirmedUpdateAndNotify(handle, hydration)

      const drained = drainBufferedUpdates(handle.buffer, hydration.reduced_through)
      for (const update of drained) {
        applyConfirmedUpdateAndNotify(handle, update)
      }
      if (handle.buffer.length > 0) {
        console.log(`[perhap] buffer drained: ${drained.length}/${handle.buffer.length} applied`)
      }
      handle.buffer = []
    })
    .catch(err => {
      console.warn('[perhap] hydration failed:', err)
      handle.hydrated = true

      const buffered = sortBufferedUpdates(handle.buffer)
      handle.buffer = []
      if (buffered.length > 0) {
        console.log(`[perhap] hydration timeout — applying ${buffered.length} buffered update(s) instead`)
        for (const update of buffered) {
          applyConfirmedUpdateAndNotify(handle, update)
        }
      }
    })

  return () => {
    handle.sub.unsubscribe()
    watchRegistry.delete(watchKey(context, entityId))
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
    dispatchModelEvent(entityId, context, transition.state.reduction, transition.state.pending, transition.state.valid)
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
