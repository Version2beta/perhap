import { connect, StringCodec, NatsConnection, Subscription } from 'nats.ws'

// ── Types ──────────────────────────────────────────────────────────────────

export interface ModelState {
  reduction: Record<string, unknown> | null
  backendPending: boolean
  valid: boolean
  reducedThrough: string | null
  pending: boolean  // backendPending || local outbox has unconfirmed events
}

// A frontend reducer applies an event locally for optimistic updates.
// Receives the current reduction, event type, and parsed data payload.
// Returns the new reduction. Pure — no side effects.
export type FrontendReducer = (
  reduction: Record<string, unknown>,
  eventType: string,
  data: unknown
) => Record<string, unknown>

interface ModelUpdate {
  entity_id: string
  context: string
  domain: string
  snapshot_uuid: string
  window: string[]
  reduced_through: string
  pending: boolean
  valid: boolean
  reduction: Record<string, unknown>
}

interface WatchHandle {
  context: string
  entityId: string
  sub: Subscription
  onUpdate: (state: ModelState) => void
  lastState: ModelState
  buffer: ModelUpdate[]
  hydrated: boolean
  reducer?: FrontendReducer
}

// Registry of active watches, keyed by "context/entityId".
// Used by tell() to apply optimistic updates immediately.
const watchRegistry = new Map<string, WatchHandle>()

// ── Singleton connection ───────────────────────────────────────────────────

let nc: NatsConnection | null = null
const sc = StringCodec()
const watches = new Map<Element, WatchHandle>()

export async function configure(opts?: { server?: string; credentials?: string }) {
  const server = opts?.server
    ?? document.querySelector<HTMLMetaElement>('meta[name="perhap-server"]')?.content
    ?? 'ws://localhost:4223'

  nc = await connect({ servers: server })
  console.log('[perhap] connected to', server)
}

// ── Core API ───────────────────────────────────────────────────────────────

export function watch(
  context: string,
  entityId: string,
  onUpdate: (state: ModelState) => void,
  reducer?: FrontendReducer
): () => void {
  if (!nc) { console.warn('[perhap] not configured'); return () => {} }

  const state: ModelState = {
    reduction: null,
    backendPending: false,
    valid: true,
    reducedThrough: null,
    pending: false,
  }

  const handle: WatchHandle = {
    context,
    entityId,
    sub: null as unknown as Subscription,
    onUpdate,
    lastState: state,
    buffer: [],
    hydrated: false,
    reducer,
  }

  watchRegistry.set(`${context}/${entityId}`, handle)

  // 1. Open subscription first, buffering updates until hydrated.
  const subject = `perhap.${context}.${entityId}._model`
  handle.sub = nc.subscribe(subject, {
    callback: (_err, msg) => {
      const update: ModelUpdate = JSON.parse(sc.decode(msg.data))
      if (!handle.hydrated) {
        handle.buffer.push(update)
        return
      }
      applyUpdate(handle, update)
    }
  })

  // 2. Hydrate: request current model.
  const hydSubject = `perhap.${context}.${entityId}._model.get`
  nc.request(hydSubject, sc.encode(''), { timeout: 5000 })
    .then(msg => {
      const hydration: ModelUpdate = JSON.parse(sc.decode(msg.data))
      handle.hydrated = true
      applyUpdate(handle, hydration)
      // Drain buffer: apply anything newer than hydration.
      for (const buffered of handle.buffer) {
        if (isNewer(buffered.reduced_through, hydration.reduced_through)) {
          applyUpdate(handle, buffered)
        }
      }
      handle.buffer = []
    })
    .catch(err => {
      console.warn('[perhap] hydration failed:', err)
      handle.hydrated = true
      handle.buffer = []
    })

  return () => {
    handle.sub.unsubscribe()
    watchRegistry.delete(`${context}/${entityId}`)
  }
}

export function tell(
  context: string,
  entityId: string,
  eventType: string,
  data: unknown,
  opts?: { causationId?: string }
): void {
  if (!nc) { console.warn('[perhap] not configured'); return }

  const event = {
    id: newUUIDv4(), // MVP: v4 for simplicity; production uses UUIDv6
    entity_id: entityId,
    context,
    type: eventType,
    causation_id: opts?.causationId ?? null,
    correlation_id: newUUIDv4(),
    data,
  }

  // Apply optimistic update immediately if this watch has a frontend reducer.
  const handle = watchRegistry.get(`${context}/${entityId}`)
  if (handle?.reducer && handle.lastState.reduction !== null) {
    const newReduction = handle.reducer(handle.lastState.reduction, eventType, data)
    const optimistic: ModelState = {
      ...handle.lastState,
      reduction: newReduction,
      pending: true,
    }
    handle.lastState = optimistic
    handle.onUpdate(optimistic)
    dispatchModelEvent(entityId, context, newReduction, true, handle.lastState.valid)
  }

  const subject = `perhap.${context}.${entityId}.${eventType}`
  nc.publish(subject, sc.encode(JSON.stringify(event)))
}

// ── Helpers ────────────────────────────────────────────────────────────────

function dispatchModelEvent(
  entityId: string,
  context: string,
  reduction: Record<string, unknown>,
  pending: boolean,
  valid: boolean
) {
  window.dispatchEvent(new CustomEvent('perhap:model', {
    detail: { entity_id: entityId, context, reduction, pending, valid }
  }))
}

function applyUpdate(handle: WatchHandle, update: ModelUpdate) {
  if (handle.lastState.reducedThrough !== null &&
      !isNewer(update.reduced_through, handle.lastState.reducedThrough)) {
    return // stale
  }
  handle.lastState = {
    reduction: update.reduction,
    backendPending: update.pending,
    valid: update.valid,
    reducedThrough: update.reduced_through,
    pending: update.pending,
  }
  handle.onUpdate(handle.lastState)
  dispatchModelEvent(update.entity_id, update.context, update.reduction, update.pending, update.valid)
}

function isNewer(a: string, b: string): boolean {
  if (!a) return false
  if (!b) return true
  return a > b
}

// Simplified UUID for MVP — production would use UUIDv6.
function newUUIDv4(): string {
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
    const r = Math.random() * 16 | 0
    return (c === 'x' ? r : (r & 0x3 | 0x8)).toString(16)
  })
}

// ── Declarative attribute layer ────────────────────────────────────────────

function parseWatch(value: string): { context: string; entityId: string } | null {
  const parts = value.split('/')
  if (parts.length !== 2) return null
  return { context: parts[0], entityId: parts[1] }
}

function nearestWatch(el: Element): { context: string; entityId: string } | null {
  let node = el.parentElement
  while (node) {
    const val = node.getAttribute('data-watch')
    if (val) return parseWatch(val)
    node = node.parentElement
  }
  return null
}

function applyBindings(el: Element, state: ModelState) {
  const reduction = state.reduction ?? {}

  // Write state attributes on the watch element itself.
  el.setAttribute('data-pending', String(state.pending))
  el.setAttribute('data-valid', String(state.valid))

  // Apply data-bind children.
  el.querySelectorAll('[data-bind]').forEach(child => {
    if (nearestWatchElement(child) !== el) return // belongs to inner scope
    const prop = child.getAttribute('data-bind')!
    if (prop.startsWith('@')) {
      child.textContent = String(metaProp(state, prop))
    } else {
      child.textContent = String(resolvePath(reduction, prop) ?? '')
    }
  })

  // Apply data-bind-attr children.
  el.querySelectorAll('[data-bind-attr]').forEach(child => {
    if (nearestWatchElement(child) !== el) return
    const spec = child.getAttribute('data-bind-attr')!
    const colon = spec.indexOf(':')
    if (colon === -1) return
    const attrName = spec.slice(0, colon)
    const propPath = spec.slice(colon + 1)
    const value = resolvePath(reduction, propPath)
    if (typeof value === 'boolean') {
      if (value) child.setAttribute(attrName, '')
      else child.removeAttribute(attrName)
    } else {
      child.setAttribute(attrName, String(value ?? ''))
    }
  })
}

function metaProp(state: ModelState, prop: string): unknown {
  switch (prop) {
    case '@pending':      return state.pending
    case '@valid':        return state.valid
    case '@reduced-through': return state.reducedThrough ?? ''
    default:              return ''
  }
}

function resolvePath(obj: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((cur, key) => {
    if (cur && typeof cur === 'object') return (cur as Record<string, unknown>)[key]
    return undefined
  }, obj)
}

function nearestWatchElement(el: Element): Element | null {
  let node = el.parentElement
  while (node) {
    if (node.hasAttribute('data-watch')) return node
    node = node.parentElement
  }
  return null
}

function serializeFormData(form: HTMLFormElement): Record<string, unknown> {
  const fd = new FormData(form)
  const result: Record<string, unknown> = {}
  fd.forEach((value, key) => {
    if (value instanceof File) return // exclude file inputs
    if (key in result) {
      const existing = result[key]
      result[key] = Array.isArray(existing) ? [...existing, value] : [existing, value]
    } else {
      result[key] = value
    }
  })
  return result
}

const wiredTells = new WeakSet<Element>()

function attachTell(el: Element, scope: { context: string; entityId: string }) {
  if (wiredTells.has(el)) return  // already wired
  wiredTells.add(el)
  const eventType = el.getAttribute('data-tell')!
  const target = el.getAttribute('data-tell-target')
  const { context, entityId } = target ? (parseWatch(target) ?? scope) : scope

  if (el.tagName === 'FORM') {
    el.addEventListener('submit', (e) => {
      e.preventDefault()
      tell(context, entityId, eventType, serializeFormData(el as HTMLFormElement))
    })
  } else if (el.tagName === 'INPUT' || el.tagName === 'SELECT' || el.tagName === 'TEXTAREA') {
    el.addEventListener('change', () => {
      const input = el as HTMLInputElement
      tell(context, entityId, eventType, { name: input.name, value: input.value })
    })
  } else {
    el.addEventListener('click', () => {
      const staticData = el.getAttribute('data-tell-data')
      const data = staticData ? JSON.parse(staticData) : {}
      tell(context, entityId, eventType, data)
    })
  }
}

const unsubscribeFns = new Map<Element, () => void>()

function mountWatch(el: Element) {
  const value = el.getAttribute('data-watch')
  if (!value) return
  const scope = parseWatch(value)
  if (!scope) return

  // Attach tell handlers to children in this scope.
  el.querySelectorAll('[data-tell]').forEach(child => {
    if (nearestWatchElement(child) === el) {
      attachTell(child, scope)
    }
  })

  const unsub = watch(scope.context, scope.entityId, (state) => {
    applyBindings(el, state)
  })
  unsubscribeFns.set(el, unsub)
}

function unmountWatch(el: Element) {
  const unsub = unsubscribeFns.get(el)
  if (unsub) {
    unsub()
    unsubscribeFns.delete(el)
  }
}

// Resolve and wire a single data-tell element. Works whether the element
// is inside a data-watch scope, outside with an explicit data-tell-target,
// or added dynamically after the initial scan.
function wireTell(el: Element) {
  const explicitTarget = el.getAttribute('data-tell-target')
  if (explicitTarget) {
    const scope = parseWatch(explicitTarget)
    if (scope) attachTell(el, scope)
    return
  }
  const watchEl = nearestWatchElement(el)
  if (!watchEl) return
  const scope = parseWatch(watchEl.getAttribute('data-watch')!)
  if (scope) attachTell(el, scope)
}

function wireTellsIn(root: Element) {
  if (root.hasAttribute('data-tell')) wireTell(root)
  root.querySelectorAll('[data-tell]').forEach(wireTell)
}

function initDOM() {
  document.querySelectorAll('[data-watch]').forEach(mountWatch)
  // Wire any data-tell elements already in the DOM (e.g. catalog buttons
  // with explicit data-tell-target set before initDOM runs).
  document.querySelectorAll('[data-tell]').forEach(wireTell)

  const observer = new MutationObserver(mutations => {
    for (const mutation of mutations) {
      mutation.addedNodes.forEach(node => {
        if (node.nodeType !== Node.ELEMENT_NODE) return
        const el = node as Element
        if (el.hasAttribute('data-watch')) mountWatch(el)
        el.querySelectorAll('[data-watch]').forEach(mountWatch)
        // Wire any data-tell elements in the added subtree.
        wireTellsIn(el)
      })
      mutation.removedNodes.forEach(node => {
        if (node.nodeType !== Node.ELEMENT_NODE) return
        const el = node as Element
        if (el.hasAttribute('data-watch')) unmountWatch(el)
        el.querySelectorAll('[data-watch]').forEach(unmountWatch)
      })
      if (mutation.type === 'attributes' && mutation.attributeName === 'data-watch') {
        const el = mutation.target as Element
        unmountWatch(el)
        mountWatch(el)
      }
      if (mutation.type === 'attributes' && mutation.attributeName === 'data-tell-target') {
        wireTell(mutation.target as Element)
      }
    }
  })

  observer.observe(document.body, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ['data-watch', 'data-tell-target'],
  })
}

// ── Auto-init ──────────────────────────────────────────────────────────────

configure().then(initDOM)
