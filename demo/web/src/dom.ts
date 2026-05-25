import type { ModelState } from './model-state'

interface DomApi {
  watch(
    context: string,
    entityId: string,
    onUpdate: (state: ModelState) => void
  ): () => void
  tell(
    context: string,
    entityId: string,
    eventType: string,
    data: unknown
  ): void
}

const wiredTells = new WeakSet<Element>()
const unsubscribeFns = new Map<Element, () => void>()

export function initDOM(api: DomApi) {
  console.log('[perhap] initDOM')
  const watchEls = document.querySelectorAll('[data-watch]')
  const tellEls = document.querySelectorAll('[data-tell]')
  console.log(`[perhap] found ${watchEls.length} data-watch, ${tellEls.length} data-tell elements`)

  watchEls.forEach(el => mountWatch(el, api))
  tellEls.forEach(el => wireTell(el, api))

  const observer = new MutationObserver(mutations => {
    for (const mutation of mutations) {
      mutation.addedNodes.forEach(node => {
        if (node.nodeType !== Node.ELEMENT_NODE) return
        const el = node as Element
        if (el.hasAttribute('data-watch')) mountWatch(el, api)
        el.querySelectorAll('[data-watch]').forEach(child => mountWatch(child, api))
        wireTellsIn(el, api)
      })

      mutation.removedNodes.forEach(node => {
        if (node.nodeType !== Node.ELEMENT_NODE) return
        const el = node as Element
        if (el.hasAttribute('data-watch')) unmountWatch(el)
        el.querySelectorAll('[data-watch]').forEach(child => unmountWatch(child))
      })

      if (mutation.type === 'attributes' && mutation.attributeName === 'data-watch') {
        const el = mutation.target as Element
        unmountWatch(el)
        mountWatch(el, api)
      }
      if (mutation.type === 'attributes' && mutation.attributeName === 'data-tell-target') {
        wireTell(mutation.target as Element, api)
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

function mountWatch(el: Element, api: DomApi) {
  const value = el.getAttribute('data-watch')
  if (!value) return
  const scope = parseWatch(value)
  if (!scope) return

  el.querySelectorAll('[data-tell]').forEach(child => {
    if (nearestWatchElement(child) === el) {
      attachTell(child, scope, api)
    }
  })

  const unsubscribe = api.watch(scope.context, scope.entityId, state => {
    applyBindings(el, state)
  })
  unsubscribeFns.set(el, unsubscribe)
}

function unmountWatch(el: Element) {
  const unsubscribe = unsubscribeFns.get(el)
  if (!unsubscribe) return
  unsubscribe()
  unsubscribeFns.delete(el)
}

function wireTellsIn(root: Element, api: DomApi) {
  if (root.hasAttribute('data-tell')) wireTell(root, api)
  root.querySelectorAll('[data-tell]').forEach(child => wireTell(child, api))
}

function wireTell(el: Element, api: DomApi) {
  const explicitTarget = el.getAttribute('data-tell-target')
  if (explicitTarget) {
    const scope = parseWatch(explicitTarget)
    if (scope) attachTell(el, scope, api)
    return
  }

  const watchEl = nearestWatchElement(el)
  if (!watchEl) {
    console.log(`[perhap] wireTell: ${el.tagName}[data-tell=${el.getAttribute('data-tell')}] has no target and no watch ancestor — skipped`)
    return
  }

  const scope = parseWatch(watchEl.getAttribute('data-watch')!)
  if (scope) attachTell(el, scope, api)
}

function attachTell(el: Element, scope: { context: string; entityId: string }, api: DomApi) {
  if (wiredTells.has(el)) return
  wiredTells.add(el)

  const eventType = el.getAttribute('data-tell')!
  const target = el.getAttribute('data-tell-target')
  const destination = target ? (parseWatch(target) ?? scope) : scope
  console.log(`[perhap] attachTell: ${el.tagName}[data-tell=${eventType}] → ${destination.context}/${destination.entityId}`)

  if (el.tagName === 'FORM') {
    el.addEventListener('submit', event => {
      event.preventDefault()
      api.tell(destination.context, destination.entityId, eventType, serializeFormData(el as HTMLFormElement))
    })
    return
  }

  if (el.tagName === 'INPUT' || el.tagName === 'SELECT' || el.tagName === 'TEXTAREA') {
    el.addEventListener('change', () => {
      const input = el as HTMLInputElement
      api.tell(destination.context, destination.entityId, eventType, { name: input.name, value: input.value })
    })
    return
  }

  el.addEventListener('click', () => {
    const staticData = el.getAttribute('data-tell-data')
    const data = staticData ? JSON.parse(staticData) : {}
    api.tell(destination.context, destination.entityId, eventType, data)
  })
}

function applyBindings(el: Element, state: ModelState) {
  const reduction = state.reduction ?? {}

  el.setAttribute('data-entity-id', state.entityId)
  el.setAttribute('data-context', state.context)
  el.setAttribute('data-pending', String(state.pending))
  el.setAttribute('data-valid', String(state.valid))

  el.querySelectorAll('[data-bind]').forEach(child => {
    if (nearestWatchElement(child) !== el) return
    const prop = child.getAttribute('data-bind')!
    child.textContent = prop.startsWith('@')
      ? String(metaProp(state, prop))
      : String(resolvePath(reduction, prop) ?? '')
  })

  el.querySelectorAll('[data-bind-attr]').forEach(child => {
    if (nearestWatchElement(child) !== el) return
    const spec = child.getAttribute('data-bind-attr')!
    const separator = spec.indexOf(':')
    if (separator === -1) return

    const attrName = spec.slice(0, separator)
    const propPath = spec.slice(separator + 1)
    const value = resolvePath(reduction, propPath)
    if (typeof value === 'boolean') {
      if (value) child.setAttribute(attrName, '')
      else child.removeAttribute(attrName)
      return
    }
    child.setAttribute(attrName, String(value ?? ''))
  })
}

function parseWatch(value: string): { context: string; entityId: string } | null {
  const parts = value.split('/')
  if (parts.length !== 2) return null
  return { context: parts[0], entityId: parts[1] }
}

function nearestWatchElement(el: Element): Element | null {
  let node = el.parentElement
  while (node) {
    if (node.hasAttribute('data-watch')) return node
    node = node.parentElement
  }
  return null
}

function metaProp(state: ModelState, prop: string): unknown {
  switch (prop) {
    case '@entity-id': return state.entityId
    case '@context': return state.context
    case '@pending': return state.pending
    case '@valid': return state.valid
    case '@reduced-through': return state.reducedThrough ?? ''
    default: return ''
  }
}

function resolvePath(obj: Record<string, unknown>, path: string): unknown {
  return path.split('.').reduce<unknown>((current, key) => {
    if (current && typeof current === 'object') {
      return (current as Record<string, unknown>)[key]
    }
    return undefined
  }, obj)
}

function serializeFormData(form: HTMLFormElement): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  const formData = new FormData(form)
  formData.forEach((value, key) => {
    if (value instanceof File) return
    if (!(key in result)) {
      result[key] = value
      return
    }
    const existing = result[key]
    result[key] = Array.isArray(existing) ? [...existing, value] : [existing, value]
  })
  return result
}
