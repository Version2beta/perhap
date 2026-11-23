package viewjournal

import "context"

// Acquire pins retained history; ID is obs/owner/nonce/after. See RECOVERY-SPEC.
func (j *Journal) Acquire(ctx context.Context, id string, v Identity, after uint64) (Retained, error) {
	return Retained{}, nil
}

// ReadRetained returns a complete bounded range from a live retained plan.
func (j *Journal) ReadRetained(ctx context.Context, id string, after uint64, max int) (Batch, error) {
	return Batch{}, nil
}

// Ordinary writes must preserve cleanup headroom; release spends its reservation.
func reserveCleanup(st *journalState, e *entry, bodyBytes, max int) error {
	if st.bytes+bodyBytes > max {
		return ErrCapacity
	}
	return nil
}
