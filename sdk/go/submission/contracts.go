// Package submission implements the gateway-only durable submission boundary.
// Complete; do not change. See SUBMIT-RESOLVE-SPEC.md. No JetStream admission.
package submission

import (
	"context"
	"errors"
	perhap "github.com/version2beta/perhap/sdk/go"
	"github.com/version2beta/perhap/sdk/go/core"
	"time"
)

var (
	ErrDenied      = errors.New("submission: denied")
	ErrBlocked     = errors.New("submission: authority unresolved")
	ErrConflict    = errors.New("submission: identity conflict")
	ErrUnavailable = errors.New("submission: unavailable")
	ErrIntegrity   = errors.New("submission: corrupt recovery evidence")
	ErrUnknown     = errors.New("submission: commitment unknown; reopen")
	ErrCapacity    = errors.New("submission: capacity")
	ErrClosed      = errors.New("submission: closed")
	ErrInvalid     = errors.New("submission: invalid envelope")
	ErrRelease     = errors.New("submission: reply may have escaped")
)

type Input struct {
	Event                      perhap.Event
	Schema, AttributionBinding string
}
type Actor struct{ Requester, Asserting string }
type Facts struct {
	EventID, Generation                                 string
	Version                                             uint64
	Receipt                                             bool
	Admission, Evaluation, Model, Evidence, Attribution string
}
type Reply struct {
	Kind  string
	Facts Facts
}
type Recipient func(context.Context, Reply) error
type Head struct {
	EventID string
	Version uint64
}

// Codec is registered, version-immutable trusted interpretation, never client code.
// Normalize changes only Data, preserving semantic links/identity; evaluation uses
// those SAME normalized bytes. CoreEvent binds authenticated attribution and explicit
// dependency classification; a cause does not imply RequiresSuccess.
type Codec interface {
	Normalize(perhap.Event) (perhap.Event, error)
	CoreEvent(perhap.Event, string) (core.Event, error)
	Model() core.Model[int64]
}

// All host callbacks are trusted, bounded/context-cooperative and non-reentrant.
// Admission calls fn zero or once and holds authority/current policy through commit.
// Release orders actual synchronous recipient release against revocation. No deferred
// output. Recover uses authority not rolled back with this separate SQLite file.
type Host interface {
	Authorize(context.Context, string, string, string, string) (Actor, error) // operation, eventID, context, requested attribution binding
	Codec(string) (Codec, error)
	Admission(context.Context, Actor, Input, func(string) error) error
	Release(context.Context, Actor, string, func() error) error
	Recover(context.Context, string, []Head) error
}
type Limits struct {
	MaxEvents, MaxEntries, MaxBytes, MaxEventBytes int
	Timeout                                        time.Duration
	Cut                                            func(string) error
}
