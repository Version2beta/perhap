// Package viewjournal persists private projection/view observation state.
// Complete; do not change. Stage 3A has no sessions, cursors or external disclosure.
package viewjournal

import (
	"context"
	"errors"
)

const Model = "pair-sum/v1"
const Schema = "nullable-sum/v1"

var (
	ErrConflict    = errors.New("viewjournal: request identity conflict")
	ErrStale       = errors.New("viewjournal: stale evaluation")
	ErrUnavailable = errors.New("viewjournal: unavailable or corrupt storage")
	ErrUnknown     = errors.New("viewjournal: commit outcome unknown; reopen")
	ErrCapacity    = errors.New("viewjournal: capacity exhausted")
	ErrOwned       = errors.New("viewjournal: competing owner")
	ErrDenied      = errors.New("viewjournal: trusted authority denied or unresolved")
)

// Identity is not an assertion key or a credential. Parameters must already be
// normalized by the trusted registration authority; no normalization is guessed.
type Identity struct{ Context, Projection, Parameters, Model, Schema, AuthorizedView, Generation string }
type Value struct{ Total int64 }

// Public equality covers ALL these fields. Available true with nil Value is a
// valid null; Available false requires nil Value. Evidence is permitted public
// metadata, never an automatic copy of the private source cut.
type Public struct {
	Value                              *Value
	Available                          bool
	Evaluation, Completeness, Evidence string
}
type Request struct {
	ID                                   string
	View                                 Identity
	EvaluationID                         string
	ExpectedCheckpoint, ExpectedRevision uint64
}
type Verified struct {
	Public     Public
	PrivateCut string
}
type Record struct {
	View               Identity
	Revision, Previous uint64
	Public             Public
}
type Receipt struct {
	Request              Request
	Checkpoint, Revision uint64
	Changed              bool
}
type Head struct {
	View                 Identity
	Checkpoint, Revision uint64
}

// Host owns registration, normalization, model/rule/schema, source-cut authority,
// public rendering/equality policy and private handle access. Within must validate
// EvaluationID and hold source/authority serialization through fn's commit.
// Recover rejects stale/missing inventory using authority NOT restored with DB.
// Authorize covers private plan/read/retry operations. No supplied flag is proof.
// Within calls fn exactly once or not at all and must not reenter this journal.
type Host interface {
	Authorize(context.Context, Identity) error
	Within(context.Context, Request, func(Verified) error) error
	Recover(context.Context, []Head) error
}

// Plan is private durable retention state, NOT an observer cursor or certificate.
type Plan struct {
	ID       string
	View     Identity
	Snapshot Record
}

// Batch covers only (After,Through] of a finite selected public head. It does not
// prove any recipient has applied After or received the batch. No signed barrier.
type Batch struct {
	PlanID                       string
	View                         Identity
	After, Through, SelectedHead uint64
	Records                      []Record
}
type Limits struct {
	MaxEvaluations, MaxPins, MaxBatch, MaxBytes int
	Cut                                         func(string) error /* before_commit, after_commit, during_read */
}
