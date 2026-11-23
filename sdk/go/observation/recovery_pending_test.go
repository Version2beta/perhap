package observation

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"path/filepath"
	"testing"
)

func TestRecoveryUnknownAcquisition(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.db")
	h := &journalHost{}
	armed := false
	jl := viewjournal.Limits{MaxEvaluations: 10, MaxPins: 3, MaxBatch: 2, MaxBytes: 1 << 20, Cut: func(at string) error {
		if armed && at == "after_commit" {
			return errors.New("lost commit reply")
		}
		return nil
	}}
	j, e := viewjournal.Open(ctx, path, h, jl)
	if e != nil {
		t.Fatal(e)
	}
	accept(t, j, h, 1)
	p := proofAuthority(t)
	m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
	armed = true
	calls := 0
	s, e := m.Open(ctx, binding, func(context.Context, RecoveryFrame) error { calls++; return nil })
	if s != nil || !errors.Is(e, ErrUnknown) || !errors.Is(e, ErrCleanup) || calls != 0 {
		t.Fatalf("unconfirmed acquisition misclassified: %v callbacks=%d", e, calls)
	}
	if e = m.Close(); !errors.Is(e, ErrCleanup) {
		t.Fatal("pending unknown pin ownership disappeared")
	}
	j.Close()
	armed = false
	j, e = viewjournal.Open(ctx, path, h, jl)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	ps, e := j.Plans(ctx)
	if e != nil || len(ps) != 1 {
		t.Fatal("expected committed unknown plan")
	}
	fresh := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
	noPins(t, j)
	fresh.Close()
}

type waitingSource struct {
	RecoverySource
	entered chan struct{}
}

func (w *waitingSource) Acquire(c context.Context, _ string, _ viewjournal.Identity, _ uint64) (viewjournal.Retained, error) {
	close(w.entered)
	<-c.Done()
	return viewjournal.Retained{}, c.Err()
}
func TestRecoveryCloseCancelsOpen(t *testing.T) {
	j, h := journal(t, 10)
	accept(t, j, h, 1)
	w := &waitingSource{RecoverySource: j, entered: make(chan struct{})}
	m := recoveryManager(t, w, NewGate(&authority{}, binding), proofAuthority(t), "slot")
	done := make(chan error, 1)
	go func() {
		_, e := m.Open(ctx, binding, func(context.Context, RecoveryFrame) error { t.Error("late callback"); return nil })
		done <- e
	}()
	<-w.entered
	if e := m.Close(); e != nil {
		t.Fatal(e)
	}
	if e := <-done; !errors.Is(e, context.Canceled) {
		t.Fatalf("in-flight open unowned: %v", e)
	}
	noPins(t, j)
}
