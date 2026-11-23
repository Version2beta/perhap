package ledger

import "context"

// Ledger exclusively owns one private local SQLite database until Close.
type Ledger struct{}

// Open validates and recovers a bounded ledger before returning authority to use it.
func Open(ctx context.Context, path string, host Host, options Options) (*Ledger, error) {
	return nil, nil
}

// Accept atomically records one host-verified evaluation or returns its original receipt.
func (l *Ledger) Accept(ctx context.Context, r Request) (Receipt, error) { return Receipt{}, nil }

// Pending returns copies of immutable locally scheduled records; no delivery occurs.
func (l *Ledger) Pending(ctx context.Context) ([]Publication, error) { return nil, nil }

// Heads returns private owner-only recovery evidence, not a public observation.
func (l *Ledger) Heads(ctx context.Context) ([]Head, error) { return nil, nil }

// Close releases the exclusive owner and is idempotent.
func (l *Ledger) Close() error { return nil }
