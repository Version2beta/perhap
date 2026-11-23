// Package delivery transfers immutable publications to a durable accepting sink.
// Complete; do not change. See docs/rddd/DELIVERY-SPEC.md. No projection is applied.
package delivery

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/ledger"
)

var (
	ErrConflict    = errors.New("delivery: identity or ordinal conflict")
	ErrGap         = errors.New("delivery: missing predecessor")
	ErrUnavailable = errors.New("delivery: unavailable or corrupt store")
	ErrUnknown     = errors.New("delivery: admission outcome unknown")
	ErrCapacity    = errors.New("delivery: capacity exhausted")
	ErrOwned       = errors.New("delivery: competing store owner")
)

type Admission struct {
	Receiver    string
	Publication ledger.Publication
}

// Sink promises durable recipient admission, not a transport receipt or projection
// application. The configured binding/host must authenticate this promise.
type Sink interface {
	Admit(context.Context, ledger.Publication) (Admission, error)
}

// Source must return bounded, immutable complete obligations; ledger.Ledger satisfies it.
type Source interface {
	Pending(context.Context) ([]ledger.Publication, error)
}
type Options struct {
	MaxRecords int
	MaxBytes   int                // maximum private encoded record bytes
	Cut        func(string) error // before_commit, after_commit; controlled failure seam only
}
type Limits struct {
	MaxCalls, MaxAttempts, MaxPending int
	Cut                               func(string) error /* before_send, after_send */
}

// Issue contains only a local publication ID and stable category. Private owner
// metadata, not a public diagnostic envelope; never include payload/raw sink errors.
type Issue struct{ PublicationID, Code string }
type Report struct {
	Calls, Confirmed int
	Issues           []Issue
}
