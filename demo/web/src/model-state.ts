export interface ModelState {
  entityId: string
  context: string
  reduction: Record<string, unknown> | null
  backendPending: boolean
  valid: boolean
  reducedThrough: string | null
  pending: boolean // backendPending || local optimistic events are still awaiting confirmation
}

export type FrontendReducer = (
  reduction: Record<string, unknown>,
  eventType: string,
  data: unknown
) => Record<string, unknown>

export interface ModelUpdate {
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

export function createModelState(context: string, entityId: string): ModelState {
  return {
    entityId,
    context,
    reduction: null,
    backendPending: false,
    valid: true,
    reducedThrough: null,
    pending: false,
  }
}

export function isNewer(a: string, b: string): boolean {
  if (!a) return false
  if (!b) return true
  return a > b
}

export function pendingAfterConfirm(
  pendingEventIds: string[],
  reducedThrough: string
): string[] {
  return pendingEventIds.filter(id => isNewer(id, reducedThrough))
}

export function applyOptimisticUpdate(
  state: ModelState,
  pendingEventIds: string[],
  eventId: string,
  eventType: string,
  data: unknown,
  reducer?: FrontendReducer
): { state: ModelState; pendingEventIds: string[] } {
  const reduction = reducer && state.reduction != null
    ? reducer(state.reduction, eventType, data)
    : state.reduction
  const nextPendingEventIds = [...pendingEventIds, eventId]
  return {
    pendingEventIds: nextPendingEventIds,
    state: {
      ...state,
      reduction,
      pending: true,
    }
  }
}

export function applyConfirmedUpdate(
  state: ModelState,
  pendingEventIds: string[],
  update: ModelUpdate
): { state: ModelState; pendingEventIds: string[]; stale: boolean } {
  if (state.reducedThrough !== null && update.reduced_through < state.reducedThrough) {
    return { state, pendingEventIds, stale: true }
  }

  // At an equal frontier only accept open → closed. Never regress a closed
  // window or overwrite reduction/optimistic state with an equal-version payload.
  if (state.reducedThrough === update.reduced_through) {
    if (!state.backendPending || update.pending) return { state, pendingEventIds, stale: true }
    return {
      stale: false,
      pendingEventIds,
      state: { ...state, backendPending: false, pending: pendingEventIds.length > 0 },
    }
  }

  const nextPendingEventIds = pendingAfterConfirm(pendingEventIds, update.reduced_through)
  return {
    stale: false,
    pendingEventIds: nextPendingEventIds,
    state: {
      entityId: state.entityId,
      context: state.context,
      reduction: update.reduction,
      backendPending: update.pending,
      valid: update.valid,
      reducedThrough: update.reduced_through,
      pending: update.pending || nextPendingEventIds.length > 0,
    }
  }
}

export function drainBufferedUpdates(
  buffered: ModelUpdate[],
  hydrationReducedThrough: string
): ModelUpdate[] {
  return buffered.filter(update => update.reduced_through >= hydrationReducedThrough)
}

export function sortBufferedUpdates(buffered: ModelUpdate[]): ModelUpdate[] {
  return buffered.slice().sort((a, b) => (a.reduced_through === b.reduced_through ? 0 : isNewer(a.reduced_through, b.reduced_through) ? 1 : -1))
}
