// Package rev computes a current-assertion view, separate from historical replay.
// Complete; do not change. See docs/rddd/REV-DESIGN.md and SPEC.md Part 2.
package rev

// Key identifies a reproducible conclusion, never a publication event or history hash.
type Key struct{ Context, Producer, Subject, Meaning string }

// Publication is already admitted, attributed and verified by the host. Value is
// a domain-canonical immutable encoding (not necessarily JSON). A status revision
// retains existing content; current/unresolved are the supported status values.
type Publication struct {
	ID         string
	Key        Key
	Revision   uint64
	Op         string // upsert, retract, status
	Value      string
	Supersedes string
	Status     string // for status operations: current or unresolved
}
type Assertion struct {
	Revision       uint64
	PublicationID  string
	Status         string // current, retracted, unresolved
	Value          string
	ValueAvailable bool // a valid encoded null is available
}
type Diagnostic struct {
	Key       Key
	Revision  uint64
	Condition string
}
type View struct {
	// History retains every supplied immutable publication in supplied order.
	// The caller owns canonical event replay; this is NOT a replacement history.
	History      []Publication
	Assertions   map[Key]Assertion
	Completeness string
	Diagnostics  []Diagnostic
}

// Computation describes one key's required-evidence closure, not a global
// commitment flag. Present false is not retraction unless valid and complete.
type Computation struct {
	Key                      Key
	Present                  bool
	Value                    string
	Evaluation, Completeness string
}

// Intention is pure, never a publication or an allocation of a durable revision.
// ExpectedRevision is a precondition for a future separately authorized publisher.
type Intention struct {
	Key               Key
	ExpectedRevision  uint64
	Op, Status, Value string
}
type Decision struct {
	Assertion Assertion
	Intention *Intention
}
