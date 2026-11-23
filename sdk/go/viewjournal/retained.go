package viewjournal

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// retainedPrefix namespaces private retained IDs: obs/owner/nonce/after.
const retainedPrefix = "obs"

// retainedIDMax bounds private retained IDs so their storage cost is known.
const retainedIDMax = 128

// cleanupReserveBytes is the headroom kept for each active pin so its release
// can always be written, even when ordinary history has filled the journal.
const cleanupReserveBytes = 2048

// Acquire pins retained history; ID is obs/owner/nonce/after. See RECOVERY-SPEC.
// The pin reuses an existing prepare entry with the latest snapshot. The anchor is
// the immutable record at revision after (the pinned snapshot when after is 0),
// selected in the same transaction. An identical retry returns the original pin,
// even after newer revisions exist.
func (j *Journal) Acquire(ctx context.Context, id string, v Identity, after uint64) (Retained, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return Retained{}, err
	}
	if err := ctx.Err(); err != nil {
		return Retained{}, err
	}
	suffix, err := parseRetainedID(id)
	if err != nil || suffix != after {
		return Retained{}, errInvalidArgument
	}
	if err := j.host.Authorize(ctx, v); err != nil {
		return Retained{}, err
	}
	var out Retained
	_, err = j.transact(ctx, func(st *journalState) (*entry, error) {
		if p, ok := st.plans[id]; ok {
			if p.View != v {
				return nil, ErrConflict
			}
			anchor, err := st.anchorOf(p, after)
			if err != nil {
				return nil, err
			}
			out = Retained{Plan: p, Anchor: anchor}
			return nil, nil
		}
		// A released ID stays in planIDs, so it can never be reused for another pin.
		if st.planIDs[id] {
			return nil, ErrConflict
		}
		snap, ok := st.latest(v)
		if !ok {
			return nil, fmt.Errorf("%w: no evaluated public snapshot", ErrUnavailable)
		}
		plan := Plan{ID: id, View: v, Snapshot: snap}
		anchor, err := st.anchorOf(plan, after)
		if err != nil {
			return nil, err
		}
		if len(st.plans) >= j.limits.MaxPins {
			return nil, ErrCapacity
		}
		out = Retained{Plan: plan, Anchor: anchor}
		return &entry{Format: entryFormat, Sequence: st.entries + 1, Kind: kindPrepare, Plan: &plan}, nil
	})
	if err != nil {
		return Retained{}, err
	}
	return out, nil
}

// ReadRetained returns a complete bounded range from a live retained plan.
// Its lower bound is the anchor named by the plan ID, not the snapshot.
func (j *Journal) ReadRetained(ctx context.Context, id string, after uint64, max int) (Batch, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return Batch{}, err
	}
	if max < 1 {
		return Batch{}, errInvalidArgument
	}
	if max > j.limits.MaxBatch {
		return Batch{}, ErrCapacity
	}
	if err := ctx.Err(); err != nil {
		return Batch{}, err
	}
	st, err := j.load(ctx)
	if err != nil {
		return Batch{}, err
	}
	p, ok := st.plans[id]
	if !ok {
		return Batch{}, errUnknownPlan
	}
	pinned, err := parseRetainedID(id)
	if err != nil {
		// Old UUID pins are not retained plans and are never served as cursors.
		return Batch{}, errUnknownPlan
	}
	if err := j.host.Authorize(ctx, p.View); err != nil {
		return Batch{}, err
	}
	anchor, err := st.anchorOf(p, pinned)
	if err != nil {
		return Batch{}, err
	}
	h := st.head(p.View)
	if after < anchor.Revision || after > h.Revision {
		return Batch{}, fmt.Errorf("%w: position outside retained range", ErrStale)
	}
	through := h.Revision
	if h.Revision-after > uint64(max) {
		through = after + uint64(max)
	}
	recs := make([]Record, through-after)
	copy(recs, st.records[p.View][after:through])
	b := Batch{PlanID: id, View: p.View, After: after, Through: through, SelectedHead: h.Revision, Records: recs}
	if err := j.hook("during_read"); err != nil {
		return Batch{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return Batch{}, err
	}
	return b, nil
}

// Ordinary writes must preserve cleanup headroom; release spends its reservation.
// Every active pin, including the one a prepare creates, keeps cleanupReserveBytes
// free after the entry is written. A release may use that reserve.
func reserveCleanup(st *journalState, e *entry, bodyBytes, max int) error {
	free := max - st.bytes
	if bodyBytes > free {
		return ErrCapacity
	}
	if e.Kind == kindRelease {
		return nil
	}
	free -= bodyBytes
	pins := len(st.plans)
	if e.Kind == kindPrepare {
		pins++
	}
	if pins > free/cleanupReserveBytes {
		return ErrCapacity
	}
	return nil
}

// parseRetainedID checks obs/owner/nonce/after and returns the after suffix.
// The suffix must be canonical decimal so that one ID names one position. The whole
// ID must be ASCII, so storage round-trips it without lossy re-encoding.
func parseRetainedID(id string) (uint64, error) {
	if len(id) > retainedIDMax {
		return 0, errInvalidArgument
	}
	for i := 0; i < len(id); i++ {
		if id[i] >= 0x80 {
			return 0, errInvalidArgument
		}
	}
	parts := strings.Split(id, "/")
	if len(parts) != 4 || parts[0] != retainedPrefix || parts[1] == "" || parts[2] == "" {
		return 0, errInvalidArgument
	}
	n, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil || strconv.FormatUint(n, 10) != parts[3] {
		return 0, errInvalidArgument
	}
	return n, nil
}

// anchorOf returns the immutable record a plan's ID names: the plan's pinned snapshot
// for position 0, otherwise the record at revision after.
func (st *journalState) anchorOf(p Plan, after uint64) (Record, error) {
	if after == 0 {
		return p.Snapshot, nil
	}
	rs := st.records[p.View]
	if after > uint64(len(rs)) {
		return Record{}, fmt.Errorf("%w: anchor is beyond the evaluated history", ErrStale)
	}
	return rs[after-1], nil
}
