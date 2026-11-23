package rev

import "github.com/version2beta/perhap/sdk/go/core"

// Reconstruct selects contiguous verified revisions per key; gaps/conflicts leave
// stale evidence unresolved, never authoritative. Frontier evidence is host-owned.
func Reconstruct(publications []Publication, frontier core.Frontier) View { return View{} }

// Reconcile computes one pure next-state/intention without allocating IDs or
// revision numbers. Unknown/failed evidence never justifies disappearance.
func Reconcile(previous Assertion, computation Computation) Decision { return Decision{} }
