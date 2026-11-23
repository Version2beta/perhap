package observation

import (
	"context"
	"errors"
	"fmt"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var ctx = context.Background()
var v = viewjournal.Identity{Context: "demo", Projection: "sum", Parameters: "default", Model: viewjournal.Model, Schema: viewjournal.Schema, AuthorizedView: "alice", Generation: "S"}
var binding = Binding{Observer: "alice", View: v, Descriptor: Profile, Epoch: "grant1"}
var limits = Limits{MaxSessions: 2, Page: 2, CallbackTimeout: time.Second, CleanupTimeout: time.Second}

type journalHost struct{ value int64 }

func (h *journalHost) Authorize(context.Context, viewjournal.Identity) error { return nil }
func (h *journalHost) Recover(context.Context, []viewjournal.Head) error     { return nil }
func (h *journalHost) Within(_ context.Context, r viewjournal.Request, f func(viewjournal.Verified) error) error {
	return f(viewjournal.Verified{Public: viewjournal.Public{Value: &viewjournal.Value{Total: h.value}, Available: true, Evaluation: "valid", Completeness: "complete_as_of_frontier", Evidence: "authorized-cut"}, PrivateCut: r.EvaluationID})
}
func journal(t *testing.T, max int) (*viewjournal.Journal, *journalHost) {
	t.Helper()
	h := &journalHost{}
	j, e := viewjournal.Open(ctx, filepath.Join(t.TempDir(), "view.db"), h, viewjournal.Limits{MaxEvaluations: max, MaxPins: 3, MaxBatch: 2, MaxBytes: 2 << 20})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { j.Close() })
	return j, h
}
func accept(t *testing.T, j *viewjournal.Journal, h *journalHost, n int) {
	t.Helper()
	heads, e := j.Heads(ctx)
	if e != nil {
		t.Fatal(e)
	}
	var old viewjournal.Head
	for _, head := range heads {
		if head.View == v {
			old = head
		}
	}
	h.value = int64(n)
	_, e = j.Accept(ctx, viewjournal.Request{ID: fmt.Sprint("r", n), View: v, EvaluationID: fmt.Sprint("g", n), ExpectedCheckpoint: old.Checkpoint, ExpectedRevision: old.Revision})
	if e != nil {
		t.Fatal(e)
	}
}

type authority struct {
	mu    sync.Mutex
	bad   bool
	certs int
}

func (a *authority) Check(_ context.Context, b Binding) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.bad {
		return ErrVersion
	}
	if b.Observer != "alice" || b.View != v || b.Epoch != "grant1" {
		return ErrDenied
	}
	return nil
}
func (a *authority) Certify(_ context.Context, b Binding, n uint64, e string) (Certificate, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.certs++
	return Certificate{Binding: b, Through: n, Evidence: e, Seal: fmt.Sprint("test-authority:", n)}, nil
}
func manager(t *testing.T, j Journal, g *Gate, l Limits) *Manager {
	t.Helper()
	m, e := NewManager(j, g, l)
	if e != nil {
		t.Fatal(e)
	}
	if m == nil {
		t.Fatal("nil manager")
	}
	return m
}
func TestBootstrap40RaceAndBarrier(t *testing.T) {
	j, h := journal(t, 41)
	for n := 1; n <= 40; n++ {
		accept(t, j, h, n)
	}
	a := &authority{}
	m := manager(t, j, NewGate(a, binding), limits)
	frames := []Frame{}
	s, e := m.Open(ctx, binding, func(_ context.Context, f Frame) error { frames = append(frames, f); return nil })
	if e != nil || s == nil {
		t.Fatalf("open %v", e)
	}
	defer s.Close()
	accept(t, j, h, 41)
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(frames) != 3 || frames[0].Kind != "snapshot" || frames[0].Record.Revision != 40 || frames[1].Kind != "revision" || frames[1].Record.Revision != 41 || frames[2].Kind != "barrier" || frames[2].Certificate.Through != 41 || frames[2].Certificate.Binding != binding || frames[2].Certificate.Evidence != "authorized-cut" || frames[2].Certificate.Seal == "" {
		t.Fatalf("snapshot handoff/barrier: %+v", frames)
	}
	acceptErr := func() error {
		_, e := j.Accept(ctx, viewjournal.Request{ID: "over", View: v, EvaluationID: "over", ExpectedCheckpoint: 41, ExpectedRevision: 41})
		return e
	}()
	if !errors.Is(acceptErr, viewjournal.ErrCapacity) {
		t.Fatal("capacity did not refuse")
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(frames) != 4 || frames[3].Certificate.Through != 41 {
		t.Fatal("false future freshness")
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	plans, e := j.Plans(ctx)
	if e != nil || len(plans) != 0 {
		t.Fatalf("pin leak %v %v", plans, e)
	}
}
func TestFreshBootstrapAfterReopenAndLimits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "retained.db")
	h := &journalHost{}
	jl := viewjournal.Limits{MaxEvaluations: 50, MaxPins: 1, MaxBatch: 2, MaxBytes: 2 << 20}
	j, err := viewjournal.Open(ctx, path, h, jl)
	if err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= 40; n++ {
		accept(t, j, h, n)
	}
	j.Close()
	j, err = viewjournal.Open(ctx, path, h, jl)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	a := &authority{}
	m := manager(t, j, NewGate(a, binding), limits)
	frames := []Frame{}
	s, err := m.Open(ctx, binding, func(_ context.Context, f Frame) error { frames = append(frames, f); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if other, err := m.Open(ctx, binding, func(context.Context, Frame) error { return nil }); err == nil || other != nil {
		t.Fatal("pin capacity bypassed")
	}
	accept(t, j, h, 41)
	if err = s.Step(); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 3 || frames[0].Record.Revision != 40 || frames[1].Record.Revision != 41 || frames[2].Certificate.Through != 41 {
		t.Fatalf("fresh authorized bootstrap after reopen: %+v", frames)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	bad := binding
	bad.Descriptor = "unsupported"
	badManager := manager(t, j, NewGate(a, bad), limits)
	if s, err := badManager.Open(ctx, bad, func(context.Context, Frame) error { t.Fatal("incompatible disclosure"); return nil }); !errors.Is(err, ErrVersion) || s != nil {
		t.Fatalf("V01 compatibility: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if s, err := m.Open(cancelled, binding, func(context.Context, Frame) error { t.Fatal("canceled disclosure"); return nil }); !errors.Is(err, context.Canceled) || s != nil {
		t.Fatalf("canceled bootstrap: %v", err)
	}
}

func TestReleaseRevocationOrders(t *testing.T) {
	a := &authority{}
	g := NewGate(a, binding)
	if g == nil {
		t.Fatal("nil gate")
	}
	f := Frame{Kind: "snapshot", Record: viewjournal.Record{View: v, Revision: 40}}
	calls := 0
	g.Revoke()
	e := g.Release(ctx, binding, f, func(context.Context, Frame) error { calls++; return nil })
	if !errors.Is(e, ErrDenied) || calls != 0 {
		t.Fatalf("revoke-first leaked %d %v", calls, e)
	}
	g = NewGate(a, binding)
	entered, finish := make(chan struct{}), make(chan struct{})
	releaseDone, revokeDone := make(chan error, 1), make(chan struct{})
	trace := []string{}
	var mu sync.Mutex
	go func() {
		releaseDone <- g.Release(ctx, binding, f, func(context.Context, Frame) error {
			mu.Lock()
			trace = append(trace, "released")
			mu.Unlock()
			close(entered)
			<-finish
			return nil
		})
	}()
	<-entered
	started := make(chan struct{})
	go func() {
		close(started)
		g.Revoke()
		mu.Lock()
		trace = append(trace, "revoked")
		mu.Unlock()
		close(revokeDone)
	}()
	<-started
	close(finish)
	if e = <-releaseDone; e != nil {
		t.Fatal(e)
	}
	<-revokeDone
	if len(trace) != 2 || trace[0] != "released" || trace[1] != "revoked" {
		t.Fatalf("release-first order %v", trace)
	}
	if e = g.Release(ctx, binding, f, func(context.Context, Frame) error { calls++; return nil }); !errors.Is(e, ErrDenied) || calls != 0 {
		t.Fatalf("later release leaked %v", e)
	}
	// Revocation before Step forbids payload, barrier and all protected metadata.
	j, h := journal(t, 10)
	accept(t, j, h, 1)
	g = NewGate(a, binding)
	m := manager(t, j, g, limits)
	s, e := m.Open(ctx, binding, func(context.Context, Frame) error { calls++; return nil })
	if e != nil {
		t.Fatal(e)
	}
	g.Revoke()
	if e = s.Step(); !errors.Is(e, ErrDenied) || calls != 0 {
		t.Fatalf("session revoke leaked %d %v", calls, e)
	}
	s.Close()
}

type wrappedJournal struct {
	Journal
	read            func(context.Context, string, uint64, int) (viewjournal.Batch, error)
	cleanupFailures int
}

func (w *wrappedJournal) Read(c context.Context, id string, n uint64, max int) (viewjournal.Batch, error) {
	if w.read != nil {
		return w.read(c, id, n, max)
	}
	return w.Journal.Read(c, id, n, max)
}
func (w *wrappedJournal) Release(c context.Context, id string) error {
	if w.cleanupFailures > 0 {
		w.cleanupFailures--
		return errors.New("private storage reason")
	}
	return w.Journal.Release(c, id)
}
func TestInvalidScanAndDescriptor(t *testing.T) {
	for _, mode := range []string{"gap", "conflict", "interrupted", "version", "observer", "view"} {
		t.Run(mode, func(t *testing.T) {
			j, h := journal(t, 10)
			accept(t, j, h, 1)
			w := &wrappedJournal{Journal: j}
			a := &authority{}
			m := manager(t, w, NewGate(a, binding), limits)
			frames := []Frame{}
			b := binding
			if mode == "observer" {
				b.Observer = "mallory"
			}
			if mode == "view" {
				b.View.Generation = "other"
			}
			s, e := m.Open(ctx, b, func(_ context.Context, f Frame) error { frames = append(frames, f); return nil })
			if mode == "observer" || mode == "view" {
				if !errors.Is(e, ErrDenied) || s != nil || len(frames) != 0 {
					t.Fatalf("unauthorized metadata: %v", e)
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			accept(t, j, h, 2)
			accept(t, j, h, 3)
			if mode == "version" {
				a.bad = true
			} else {
				w.read = func(c context.Context, id string, n uint64, max int) (viewjournal.Batch, error) {
					b, e := j.Read(c, id, n, max)
					if mode == "interrupted" {
						return b, errors.New("private scan failure")
					}
					if len(b.Records) != 2 {
						t.Fatal("test did not get page")
					}
					if mode == "gap" {
						b.Records[0].Previous = 99
					} else {
						b.Records[1].Revision = b.Records[0].Revision
					}
					return b, e
				}
			}
			e = s.Step()
			if e == nil {
				t.Fatal("invalid stream accepted")
			}
			for _, f := range frames {
				if f.Kind != "snapshot" {
					t.Fatalf("partial page certified/released: %+v", frames)
				}
			}
			if a.certs != 0 {
				t.Fatal("false barrier")
			}
		})
	}
}
func TestSessionCancellationEdges(t *testing.T) {
	t.Run("cancel-before-step", func(t *testing.T) {
		j, h := journal(t, 10)
		accept(t, j, h, 1)
		a := &authority{}
		l := limits
		l.MaxSessions = 1
		m := manager(t, j, NewGate(a, binding), l)
		c, cancel := context.WithCancel(ctx)
		calls := 0
		s, err := m.Open(c, binding, func(context.Context, Frame) error { calls++; return nil })
		if err != nil {
			t.Fatal(err)
		}
		cancel()
		if err = s.Step(); err == nil {
			t.Fatal("canceled step succeeded")
		}
		plans, err := j.Plans(ctx)
		if err != nil || len(plans) != 0 || calls != 0 {
			t.Fatalf("canceled Step leaked pin/quota or disclosed: pins=%d calls=%d err=%v", len(plans), calls, err)
		}
		next, err := m.Open(ctx, binding, func(context.Context, Frame) error { return nil })
		if err != nil {
			t.Fatalf("canceled Step retained quota: %v", err)
		}
		next.Close()
		s.Close()
	})
	t.Run("expired-success-not-certified", func(t *testing.T) {
		j, h := journal(t, 10)
		accept(t, j, h, 1)
		a := &authority{}
		l := limits
		l.CallbackTimeout = time.Millisecond
		m := manager(t, j, NewGate(a, binding), l)
		s, err := m.Open(ctx, binding, func(c context.Context, _ Frame) error { <-c.Done(); return nil })
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		if err = s.Step(); !errors.Is(err, ErrUnknown) || a.certs != 0 {
			t.Fatalf("expired callback produced success/barrier: %v certificates=%d", err, a.certs)
		}
	})
}

func TestSessionBoundsCancellationAndCleanup(t *testing.T) {
	j, h := journal(t, 20)
	accept(t, j, h, 1)
	a := &authority{}
	l := limits
	l.MaxSessions = 1
	m := manager(t, j, NewGate(a, binding), l)
	s, e := m.Open(ctx, binding, func(context.Context, Frame) error { return nil })
	if e != nil {
		t.Fatal(e)
	}
	if other, e := m.Open(ctx, binding, func(context.Context, Frame) error { return nil }); !errors.Is(e, ErrCapacity) || other != nil {
		t.Fatalf("session quota %v", e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if e = s.Step(); !errors.Is(e, ErrClosed) {
		t.Fatalf("retired work %v", e)
	}
	entered := make(chan struct{})
	m = manager(t, j, NewGate(a, binding), l)
	s, e = m.Open(ctx, binding, func(c context.Context, _ Frame) error { close(entered); <-c.Done(); return c.Err() })
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- s.Step() }()
	<-entered
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, ErrUnknown) {
		t.Fatalf("uncertain callback release %v", e)
	}
	if e = s.Step(); !errors.Is(e, ErrClosed) {
		t.Fatal(e)
	}
	w := &wrappedJournal{Journal: j, cleanupFailures: 1}
	m = manager(t, w, NewGate(a, binding), l)
	s, e = m.Open(ctx, binding, func(context.Context, Frame) error { return errors.New("recipient may have disclosed") })
	if e != nil {
		t.Fatal(e)
	}
	e = s.Step()
	if !errors.Is(e, ErrUnknown) || !errors.Is(e, ErrCleanup) {
		t.Fatalf("primary/cleanup lost %v", e)
	}
	if e = s.Close(); e != nil {
		t.Fatalf("cleanup not retryable %v", e)
	}
	plans, e := j.Plans(ctx)
	if e != nil || len(plans) != 0 {
		t.Fatalf("pin cleanup %v %v", plans, e)
	}
	// Deadline is a contract on cooperative recipients; no goroutine is abandoned.
	l.CallbackTimeout = time.Millisecond
	m = manager(t, j, NewGate(a, binding), l)
	s, e = m.Open(ctx, binding, func(c context.Context, _ Frame) error {
		if _, ok := c.Deadline(); !ok {
			t.Fatal("callback not bounded")
		}
		<-c.Done()
		return c.Err()
	})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Step(); !errors.Is(e, ErrUnknown) {
		t.Fatalf("deadline release outcome %v", e)
	}
	s.Close()
}
