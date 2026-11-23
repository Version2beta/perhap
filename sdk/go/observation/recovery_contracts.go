package observation

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"time"
)

// Complete; do not change. Proof authority is not disclosure/state authority.
var ErrCannotResume = errors.New("observation: cannot resume")

type Claims struct {
	Binding Binding
	Purpose string
	Record  viewjournal.Record
}

// Verify returns ErrCannotResume only for invalid/expired/policy-unretained proofs.
// Unavailable authority and cancellation are not evidence that a cursor is invalid.
// Issue receives a copy; its result must certify the exact requested immutable data.
type ProofAuthority interface {
	Issue(context.Context, Claims) (string, error)
	Verify(context.Context, string) (Claims, error)
}

// RecoverySource retains immutable anchors; private ownership IDs must never escape.
type RecoverySource interface {
	Acquire(context.Context, string, viewjournal.Identity, uint64) (viewjournal.Retained, error)
	ReadRetained(context.Context, string, uint64, int) (viewjournal.Batch, error)
	Plans(context.Context) ([]viewjournal.Plan, error)
	Release(context.Context, string) error
}
type RecoveryFrame struct {
	Kind    string
	Session string
	Record  viewjournal.Record
	Proof   string
}
type RecoveryRecipient func(context.Context, RecoveryFrame) error
type RecoveryLimits struct {
	Page, MaxBytes, MaxTokenBytes   int
	CallbackTimeout, CleanupTimeout time.Duration
}

// Host must exclusively own Owner namespace across startup/recovery, independently
// of observer permission and database rollback. It bounds the number of owners.
// Source cleanup uses service authority; Gate always enforces current disclosure.
