import { configure, ready, tell, watch } from './core'
import { initDOM } from './dom'

export type { FrontendReducer, ModelState } from './model-state'
export { configure, ready, tell, watch }

if (typeof document !== 'undefined' && typeof window !== 'undefined') {
  configure().then(() => initDOM({ watch, tell }))
}
