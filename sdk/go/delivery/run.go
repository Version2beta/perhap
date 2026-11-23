package delivery

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/version2beta/perhap/sdk/go/ledger"
)

// Issue codes. Each names why one source obligation was not delivered in this pass.
const (
	issueFailed          = "failed"
	issueUnknown         = "unknown"
	issuePredecessor     = "predecessor"
	issueReceiptConflict = "receipt-conflict"
)

// lane is one scoped key/generation. Its obligations are sent strictly in ordinal order;
// a blocked or exhausted lane stops for this pass and never stops another lane.
type lane struct {
	scope    scope
	queue    []ledger.Publication // outstanding obligations, ascending by ordinal
	head     ledger.Publication   // newest durable confirmation in this scope; zero when none
	attempts int                  // sink attempts for queue[0] in this pass
	done     bool                 // blocked or exhausted for the rest of this pass
}

// runPass reconciles source obligations with the confirmation journal, then gives each
// open lane one sink attempt per round until no lane can progress or a limit is reached.
// Every loop iteration either sends, confirms, counts a failed attempt, or closes a lane,
// so the pass terminates.
func runPass(ctx context.Context, source Source, sink Sink, confirmations *Store, limits Limits) (Report, error) {
	var r Report
	if source == nil || sink == nil || confirmations == nil ||
		limits.MaxCalls <= 0 || limits.MaxAttempts <= 0 || limits.MaxPending <= 0 {
		return r, errInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return r, err
	}
	pending, err := source.Pending(ctx)
	if err != nil {
		return r, fmt.Errorf("delivery: source pending: %w", err)
	}
	if len(pending) > limits.MaxPending {
		return r, ErrCapacity
	}
	confirmed, err := confirmations.Records(ctx)
	if err != nil {
		return r, err
	}
	lanes, err := planLanes(pending, confirmed, confirmations, &r)
	if err != nil {
		return r, err
	}
	for {
		open := false
		for _, ln := range lanes {
			if ln.done || len(ln.queue) == 0 {
				continue
			}
			if r.Calls >= limits.MaxCalls {
				return r, nil
			}
			if err := ctx.Err(); err != nil {
				return r, err
			}
			open = true
			if err := ln.step(ctx, sink, confirmations, limits, &r); err != nil {
				return r, err
			}
		}
		if !open {
			return r, nil
		}
	}
}

// planLanes groups pending obligations by scope, drops those already confirmed with the
// same content, and orders each lane by ordinal. A source record whose ID is confirmed or
// repeated with different content contradicts the journal, so its lane is blocked.
func planLanes(pending, confirmed []ledger.Publication, confirmations *Store, r *Report) ([]*lane, error) {
	confirmedByID := make(map[string]ledger.Publication, len(confirmed))
	heads := make(map[scope]ledger.Publication)
	for _, c := range confirmed {
		confirmedByID[c.Record.ID] = c
		heads[scopeOf(c)] = c
	}
	seen := make(map[string]ledger.Publication)
	var lanes []*lane
	index := make(map[scope]*lane)
	for _, p := range pending {
		if err := checkPublication(p, confirmations.producer); err != nil {
			return nil, err
		}
		s := scopeOf(p)
		ln, ok := index[s]
		if !ok {
			ln = &lane{scope: s, head: heads[s]}
			index[s] = ln
			lanes = append(lanes, ln)
		}
		if c, found := confirmedByID[p.Record.ID]; found {
			if c != p {
				ln.block(r, p, issueReceiptConflict)
			}
			continue
		}
		if prev, found := seen[p.Record.ID]; found {
			if prev != p {
				ln.block(r, p, issueReceiptConflict)
			}
			continue
		}
		seen[p.Record.ID] = p
		ln.queue = append(ln.queue, p)
	}
	for _, ln := range lanes {
		sort.SliceStable(ln.queue, func(i, j int) bool {
			return ln.queue[i].Record.Revision < ln.queue[j].Record.Revision
		})
	}
	return lanes, nil
}

// step makes at most one sink attempt for the lane's next obligation. It persists the
// confirmation only after the sink reports a matching durable admission.
func (ln *lane) step(ctx context.Context, sink Sink, confirmations *Store, limits Limits, r *Report) error {
	p := ln.queue[0]
	if code := ln.blockReason(p); code != "" {
		ln.block(r, p, code)
		return nil
	}
	if limits.Cut != nil {
		if err := limits.Cut("before_send"); err != nil {
			return err
		}
	}
	r.Calls++
	adm, err := sink.Admit(ctx, p)
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		ln.sinkFailed(r, p, err, limits.MaxAttempts)
		return nil
	}
	if adm.Receiver != confirmations.receiver || adm.Publication != p {
		ln.block(r, p, issueReceiptConflict)
		return nil
	}
	if limits.Cut != nil {
		if err := limits.Cut("after_send"); err != nil {
			return err
		}
	}
	if _, err := confirmations.Admit(ctx, p); err != nil {
		return err
	}
	r.Confirmed++
	ln.head = p
	ln.queue = ln.queue[1:]
	ln.attempts = 0
	return nil
}

// blockReason reports why p cannot be sent now. The next obligation must directly follow
// the newest durable confirmation in its scope, so an unconfirmed predecessor blocks it.
func (ln *lane) blockReason(p ledger.Publication) string {
	if p.Record.Revision <= ln.head.Record.Revision {
		return issueReceiptConflict
	}
	if p.Record.Revision != ln.head.Record.Revision+1 || p.Record.Supersedes != ln.head.Record.ID {
		return issuePredecessor
	}
	return ""
}

// sinkFailed classifies a sink error without treating it as success. Gap and conflict are
// permanent for this key; other errors are retried until MaxAttempts is reached.
func (ln *lane) sinkFailed(r *Report, p ledger.Publication, err error, maxAttempts int) {
	switch {
	case errors.Is(err, ErrGap):
		ln.block(r, p, issuePredecessor)
	case errors.Is(err, ErrConflict):
		ln.block(r, p, issueReceiptConflict)
	default:
		ln.attempts++
		if ln.attempts >= maxAttempts {
			code := issueFailed
			if errors.Is(err, ErrUnknown) {
				code = issueUnknown
			}
			ln.block(r, p, code)
		}
	}
}

// block stops the lane for the rest of this pass and records the first reason, naming
// the obligation that stopped it. Later obligations on the lane are never skipped.
func (ln *lane) block(r *Report, p ledger.Publication, code string) {
	if ln.done {
		return
	}
	ln.done = true
	r.Issues = append(r.Issues, Issue{PublicationID: p.Record.ID, Code: code})
}
