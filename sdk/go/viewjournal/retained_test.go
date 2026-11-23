package viewjournal

import (
	"errors"
	"testing"
)

func TestRetainedCleanupReservation(t *testing.T) {
	st := newJournalState()
	st.bytes = 100
	st.plans["obs/slot/nonce/0"] = Plan{}
	if e := reserveCleanup(st, &entry{Kind: kindAccept}, 500, 2500); !errors.Is(e, ErrCapacity) {
		t.Fatal("ordinary write consumed cleanup reservation")
	}
	if e := reserveCleanup(st, &entry{Kind: kindRelease}, 500, 2500); e != nil {
		t.Fatal("cleanup cannot spend its reservation")
	}
	maxInt := int(^uint(0) >> 1)
	st.bytes = maxInt - 1
	if e := reserveCleanup(st, &entry{Kind: kindAccept}, 4, maxInt); !errors.Is(e, ErrCapacity) {
		t.Fatal("byte accounting wrapped")
	}
	st.bytes = 100
	delete(st.plans, "obs/slot/nonce/0")
	if e := reserveCleanup(st, &entry{Kind: kindAccept}, 500, 2500); e != nil {
		t.Fatal(e)
	}
	if e := reserveCleanup(st, &entry{Kind: kindPrepare}, 500, 2500); !errors.Is(e, ErrCapacity) {
		t.Fatal("plan acquired without cleanup budget")
	}
}
