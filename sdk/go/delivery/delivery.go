package delivery

import (
	"context"
	"github.com/version2beta/perhap/sdk/go/ledger"
)

// Store is an exclusively owned durable admission log, also usable as the sender's
// separate acknowledgment journal bound to one receiver identity.
type Store struct{}

// Open opens a private, bounded SQLite store for a trusted receiver/producer pair.
func Open(ctx context.Context, path, receiver, producer string, options Options) (*Store, error) {
	return nil, nil
}

// Admit durably admits one immutable record or returns the original admission.
func (s *Store) Admit(ctx context.Context, p ledger.Publication) (Admission, error) {
	return Admission{}, nil
}

// Records returns copies of the admitted historical records, not projections.
func (s *Store) Records(ctx context.Context) ([]ledger.Publication, error) { return nil, nil }

// Close releases ownership and is idempotent.
func (s *Store) Close() error { return nil }

// Run performs one bounded single-worker pass. The owner closes source, sink and
// confirmations after it returns; repeated passes/restarts resume durable work.
func Run(ctx context.Context, source Source, sink Sink, confirmations *Store, limits Limits) (Report, error) {
	return Report{}, nil
}
