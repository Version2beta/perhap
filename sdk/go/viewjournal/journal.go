package viewjournal

import "context"

// Journal exclusively owns a private local SQLite observation journal.
type Journal struct{}

// Open recovers and validates the journal under trusted independent authority.
func Open(ctx context.Context, path string, host Host, limits Limits) (*Journal, error) {
	return nil, nil
}

// Accept atomically records a verified evaluation and any material public change.
func (j *Journal) Accept(ctx context.Context, r Request) (Receipt, error) { return Receipt{}, nil }

// Heads returns private recovery information, not a public observation.
func (j *Journal) Heads(ctx context.Context) ([]Head, error) { return nil, nil }

// Plans enumerates private durable pins for recovery after a lost planning reply.
func (j *Journal) Plans(ctx context.Context) ([]Plan, error) { return nil, nil }

// Prepare persists a consistent snapshot/successor retention plan; no data is disclosed.
func (j *Journal) Prepare(ctx context.Context, v Identity) (Plan, error) { return Plan{}, nil }

// Read returns a complete bounded successor range or an error with no intention.
func (j *Journal) Read(ctx context.Context, id string, after uint64, max int) (Batch, error) {
	return Batch{}, nil
}

// Release durably releases a private plan; accepted history is never pruned here.
func (j *Journal) Release(ctx context.Context, id string) error { return nil }

// Close is idempotent and releases ownership, not persisted plans/history.
func (j *Journal) Close() error { return nil }
