// Package core evaluates finite already-admitted histories without I/O.
// Complete; do not change. See SPEC.md and docs/rddd for the upstream pin.
package core

// Dependency carries trusted context-owned classification. Local dependencies
// order present events; absent local/required-external/unknown dependencies block.
// Trace-only external references never force fetching foreign history.
type Dependency struct {
	ID              string
	Kind            string // local, required-external, trace-external, unknown
	Satisfied       bool   // trusted proof for an external dependency
	RequiresSuccess bool
}

// Event contains immutable semantic fields, not transport credentials. Payload
// is a domain-defined immutable string; callers may encode it as they choose.
// Subjects may name zero, one or multiple aggregates. IDs are opaque byte strings.
type Event struct {
	ID, Context, Type, Payload, AssertedBy string
	Subjects                               []string
	Dependencies                           []Dependency
}

// Frontier is evidence supplied by a trusted host, not authentication performed
// by this library. Certified requires a nonempty identified Boundary.
type Frontier struct {
	Boundary       string
	Certified, Gap bool
}
type Request struct {
	Context, Scope, RuleVersion string
	Events                      []Event
	Frontier                    Frontier
}

// Proposal is a pure candidate, never executable authority.
type Proposal struct{ Key, Action, Arguments string }
type Derivation struct{ Key, Value string }
type Selection struct {
	Proposals   []Proposal
	Derivations []Derivation
}

// Model callbacks must be deterministic, terminating and side-effect-free.
// Clone must isolate all mutable state and must not mutate its argument.
// Validate returns (false,nil) for domain-invalid, versus an error for inability
// to validate. Select sees only successful event inputs and final working state.
// Explicit environmental observations are captured immutably in these callbacks
// and identified by RuleVersion; ambient clocks or random input are forbidden.
type Model[S any] struct {
	Initialize func() (S, error)
	Clone      func(S) (S, error)
	Reduce     func(S, Event) (S, error)
	Validate   func(S) (bool, error)
	Select     func(S, []Event) (Selection, error)
}
type Diagnostic struct {
	Key, EventID, Stage, Condition, RuleVersion string
	Err                                         error
}
type Result[S any] struct {
	Working, LastValid                   S
	WorkingAvailable, LastValidAvailable bool
	LastValidPrefix                      []string
	Evaluation, Completeness             string
	Context, Scope, RuleVersion          string
	Frontier                             Frontier
	History                              []Event
	Attempted, Blocked, Successful       []string
	Diagnostics                          []Diagnostic
	Selection                            Selection
}
