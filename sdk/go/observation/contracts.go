// Package observation provides bounded in-process bootstrap sessions (Stage 3B1).
// Complete; do not change. Opaque cursors/resume/resnapshot are deferred to 3B2.
package observation

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"time"
)

var (
	ErrDenied      = errors.New("observation: denied or unresolved authority")
	ErrVersion     = errors.New("observation: incompatible descriptor")
	ErrClosed      = errors.New("observation: retired session")
	ErrCapacity    = errors.New("observation: resource limit")
	ErrSequence    = errors.New("observation: invalid successor chain")
	ErrUnknown     = errors.New("observation: release may have occurred")
	ErrCleanup     = errors.New("observation: cleanup incomplete")
	ErrUnavailable = errors.New("observation: unavailable")
)

const Profile = "INTEROP-PUBOBS/0.1@0.1.0"

type Binding struct {
	Observer          string
	View              viewjournal.Identity
	Descriptor, Epoch string
}
type Certificate struct {
	Binding        Binding
	Through        uint64
	Evidence, Seal string
}
type Frame struct {
	Kind        string /* snapshot, revision, barrier */
	Record      viewjournal.Record
	Certificate Certificate
}

// Recipient must synchronously perform the actual irreversible in-process release,
// obey context deadlines, and never queue later uncontrolled disclosure or reenter
// Session/Gate. Failure after invocation is uncertain disclosure, never rollback.
type Recipient func(context.Context, Frame) error

// Authority checks the authenticated observer/current epoch/view/descriptor, not
// caller flags. Certify binds verified as-of coverage; it is not a cursor issuer.
// Methods must honor contexts and cannot reenter Gate. No production trust claimed.
type Authority interface {
	Check(context.Context, Binding) error
	Certify(context.Context, Binding, uint64, string) (Certificate, error)
}
type Journal interface {
	Prepare(context.Context, viewjournal.Identity) (viewjournal.Plan, error)
	Read(context.Context, string, uint64, int) (viewjournal.Batch, error)
	Release(context.Context, string) error
}
type Limits struct {
	MaxSessions, Page               int
	CallbackTimeout, CleanupTimeout time.Duration
}
