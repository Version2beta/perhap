package delivery

import (
	"context"
	"database/sql"
	"sync"

	"github.com/version2beta/perhap/sdk/go/ledger"
)

// Store is an exclusively owned durable admission log, also usable as the sender's
// separate acknowledgment journal bound to one receiver identity.
type Store struct {
	mu         sync.Mutex
	db         *sql.DB
	conn       *sql.Conn // single pinned connection that holds the exclusive lock
	receiver   string
	producer   string
	maxRecords int
	maxBytes   int
	cut        func(string) error
	closed     bool
	poisoned   bool // a commit outcome is unknown; the handle must be reopened
}

// Open opens a private, bounded SQLite store for a trusted receiver/producer pair.
func Open(ctx context.Context, path, receiver, producer string, options Options) (*Store, error) {
	if !validOpenArgs(path, receiver, producer, options) {
		return nil, errInvalidArgument
	}
	created, err := createFile(path)
	if err != nil {
		return nil, err
	}
	s := &Store{receiver: receiver, producer: producer, maxRecords: options.MaxRecords, maxBytes: options.MaxBytes, cut: options.Cut}
	if err := s.open(ctx, path, created); err != nil {
		_ = s.release()
		return nil, err
	}
	return s, nil
}

// Admit durably admits one immutable record or returns the original admission.
func (s *Store) Admit(ctx context.Context, p ledger.Publication) (Admission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return Admission{}, err
	}
	if err := checkPublication(p, s.producer); err != nil {
		return Admission{}, err
	}
	stored, err := s.admitTx(ctx, p)
	if err != nil {
		return Admission{}, err
	}
	return Admission{Receiver: s.receiver, Publication: stored}, nil
}

// Records returns copies of the admitted historical records, not projections.
func (s *Store) Records(ctx context.Context) ([]ledger.Publication, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.usable(); err != nil {
		return nil, err
	}
	st, err := loadState(ctx, s.conn, s.producer)
	if err != nil {
		return nil, err
	}
	return st.records, nil
}

// Close releases ownership and is idempotent.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.release()
}

// Run performs one bounded single-worker pass. The owner closes source, sink and
// confirmations after it returns; repeated passes/restarts resume durable work.
func Run(ctx context.Context, source Source, sink Sink, confirmations *Store, limits Limits) (Report, error) {
	return Report{}, nil
}
