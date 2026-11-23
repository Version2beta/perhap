// Package ledger owns a private local reconciliation ledger. No delivery is performed.
// Complete; do not change. See docs/rddd/LEDGER-SPEC.md.
package ledger

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/rev"
)

var (
	ErrConflict    = errors.New("ledger: conflicting request identity")
	ErrStale       = errors.New("ledger: stale preconditions or evidence")
	ErrUnavailable = errors.New("ledger: unavailable storage or integrity")
	ErrUnknown     = errors.New("ledger: commit outcome unknown; reopen and retry")
	ErrOwned       = errors.New("ledger: competing owner")
	ErrDenied      = errors.New("ledger: host authorization unavailable or denied")
	ErrCapacity    = errors.New("ledger: capacity exhausted")
)

type Request struct {
	ID                                   string
	Key                                  rev.Key
	Generation                           string
	EvaluationID                         string
	ExpectedCheckpoint, ExpectedRevision uint64
	ExpectedPublicationID                string
}
type Head struct {
	Key        rev.Key
	Generation string
	Checkpoint uint64
	Assertion  rev.Assertion
}

// Evidence is returned only by the trusted host, not by the caller.
type Evidence struct {
	Computation                                 rev.Computation
	Registered, PublishAllowed, InternalAllowed bool
	Schema, Rule, Attribution, Cut              string
}

// Host authenticates this local invocation. WithinEvaluation must hold a source/
// rule/registration/authority fence through fn's commit and prove EvaluationID is
// current, including all relevant admitted evidence. Recover must reject stale
// restored storage/missing heads using authority state NOT restored with this DB.
// A no-op fake does not establish real authority or end-to-end source freshness.
// Methods must not reenter Ledger; fn must be called exactly once or not at all.
type Host interface {
	Authorize(context.Context, Request) error
	WithinEvaluation(context.Context, Request, func(Evidence) error) error
	Recover(context.Context, []Head) error
}
type Publication struct {
	Record                                     rev.Publication
	Generation, Schema, Rule, Attribution, Cut string
	ValueAvailable                             bool
}
type Receipt struct {
	Request     Request
	Checkpoint  uint64
	Publication *Publication // nil explicitly means no publication allocated
}
type Options struct {
	MaxRecords int // positive bounded lifetime receipt count; no compaction/expiry
	// Cut is an optional controlled failure seam: before_tx, after_write,
	// before_commit, after_commit. Not an authorization or production extension.
	Cut func(string) error
}
