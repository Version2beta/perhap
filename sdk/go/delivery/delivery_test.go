package delivery

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/ledger"
	"github.com/version2beta/perhap/sdk/go/rev"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

var ctx = context.Background()
var opts = Options{MaxRecords: 30, MaxBytes: 65536}
var limits = Limits{MaxCalls: 20, MaxAttempts: 2, MaxPending: 30}

func publication(id, subject string, n uint64, prev string) ledger.Publication {
	return ledger.Publication{Record: rev.Publication{ID: id, Key: rev.Key{Context: "demo", Producer: "P", Subject: subject, Meaning: "m"}, Revision: n, Op: "upsert", Value: id, Supersedes: prev}, Generation: "G", Schema: "v1", Rule: "r1", Attribution: "P", Cut: "cut", ValueAvailable: true}
}
func store(t *testing.T, path string, o Options) *Store {
	t.Helper()
	s, e := Open(ctx, path, "receiver", "P", o)
	if e != nil {
		t.Fatal(e)
	}
	if s == nil {
		t.Fatal("nil durable store")
	}
	t.Cleanup(func() { s.Close() })
	return s
}

type source []ledger.Publication

func (s source) Pending(context.Context) ([]ledger.Publication, error) {
	return append([]ledger.Publication(nil), s...), nil
}

type sinkFunc func(context.Context, ledger.Publication) (Admission, error)

func (f sinkFunc) Admit(c context.Context, p ledger.Publication) (Admission, error) { return f(c, p) }
func records(t *testing.T, s *Store) []ledger.Publication {
	t.Helper()
	ps, e := s.Records(ctx)
	if e != nil {
		t.Fatal(e)
	}
	return ps
}
func TestDeliveryDurableSink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sink.db")
	s := store(t, path, opts)
	p1 := publication("1", "x", 1, "")
	p2 := publication("2", "x", 2, "1")
	if _, e := s.Admit(ctx, p2); !errors.Is(e, ErrGap) {
		t.Fatalf("missing predecessor: %v", e)
	}
	a, e := s.Admit(ctx, p1)
	if e != nil || a.Publication != p1 || a.Receiver != "receiver" {
		t.Fatalf("durable receipt %+v %v", a, e)
	}
	s.Close()
	s = store(t, path, opts)
	b, e := s.Admit(ctx, p1)
	if e != nil || b != a || len(records(t, s)) != 1 {
		t.Fatalf("P16 duplicate after restart %+v %v", b, e)
	}
	bad := p1
	bad.Record.Value = "changed"
	if _, e = s.Admit(ctx, bad); !errors.Is(e, ErrConflict) {
		t.Fatalf("ID collision: %v", e)
	}
	bad = p1
	bad.Record.ID = "different"
	if _, e = s.Admit(ctx, bad); !errors.Is(e, ErrConflict) {
		t.Fatalf("ordinal collision: %v", e)
	}
	bad = p2
	bad.Record.Supersedes = "wrong"
	if _, e = s.Admit(ctx, bad); !errors.Is(e, ErrGap) {
		t.Fatalf("wrong predecessor: %v", e)
	}
	if _, e = s.Admit(ctx, p2); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s = store(t, path, opts)
	if !reflect.DeepEqual(records(t, s), []ledger.Publication{p1, p2}) {
		t.Fatal("restart lost content")
	}
	if other, e := Open(ctx, path, "receiver", "P", opts); e == nil {
		if other != nil {
			other.Close()
		}
		t.Fatal("second store owner accepted")
	}
}
func TestDeliveryOrderingAndUnknown(t *testing.T) {
	dir := t.TempDir()
	sink := store(t, filepath.Join(dir, "sink.db"), opts)
	acks := store(t, filepath.Join(dir, "acks.db"), opts)
	p1 := publication("x1", "x", 1, "")
	p2 := publication("x2", "x", 2, "x1")
	other := publication("y1", "y", 1, "")
	calls := []string{}
	lost := false
	wrapped := sinkFunc(func(c context.Context, p ledger.Publication) (Admission, error) {
		calls = append(calls, p.Record.ID)
		a, e := sink.Admit(c, p)
		if e == nil && p.Record.ID == "x1" && !lost {
			lost = true
			return Admission{}, ErrUnknown
		}
		return a, e
	})
	r, e := Run(ctx, source{p2, other, p1}, wrapped, acks, limits)
	if e != nil || r.Confirmed != 3 || len(records(t, sink)) != 3 || len(records(t, acks)) != 3 {
		t.Fatalf("P16: %+v %v calls=%v", r, e, calls)
	}
	x2, first, second := -1, -1, -1
	for i, id := range calls {
		if id == "x1" {
			if first < 0 {
				first = i
			} else {
				second = i
			}
		}
		if id == "x2" {
			x2 = i
		}
	}
	if second < first || x2 < second {
		t.Fatalf("bypassed unknown predecessor: %v", calls)
	}
	r, e = Run(ctx, source{p2, other, p1}, sink, acks, limits)
	if e != nil || r.Calls != 0 {
		t.Fatalf("confirmed work resent: %+v %v", r, e)
	}
}
func TestDeliveryBoundsAndIndependentKeys(t *testing.T) {
	dir := t.TempDir()
	s := store(t, filepath.Join(dir, "sink.db"), opts)
	a := store(t, filepath.Join(dir, "acks.db"), opts)
	p1 := publication("x1", "x", 1, "")
	p2 := publication("x2", "x", 2, "x1")
	y := publication("y1", "y", 1, "")
	calls := map[string]int{}
	f := sinkFunc(func(c context.Context, p ledger.Publication) (Admission, error) {
		calls[p.Record.ID]++
		if p.Record.Key.Subject == "x" {
			return Admission{}, ErrUnavailable
		}
		return s.Admit(c, p)
	})
	r, e := Run(ctx, source{p2, y, p1}, f, a, limits)
	if e != nil || r.Confirmed != 1 || calls["x1"] != 2 || calls["x2"] != 0 || calls["y1"] != 1 {
		t.Fatalf("independent progress/retry bounds: %+v %v %v", r, e, calls)
	}
	c, cancel := context.WithCancel(ctx)
	f = sinkFunc(func(context.Context, ledger.Publication) (Admission, error) {
		cancel()
		return Admission{}, context.Canceled
	})
	r, e = Run(c, source{p1}, f, a, limits)
	if !errors.Is(e, context.Canceled) || r.Calls != 1 {
		t.Fatalf("cancellation %+v %v", r, e)
	}
	bound := limits
	bound.MaxPending = 1
	if _, e = Run(ctx, source{p1, p2}, s, a, bound); !errors.Is(e, ErrCapacity) {
		t.Fatalf("backpressure %v", e)
	}
	small := store(t, filepath.Join(dir, "small.db"), Options{MaxRecords: 1, MaxBytes: 65536})
	if _, e = small.Admit(ctx, p1); e != nil {
		t.Fatal(e)
	}
	if _, e = small.Admit(ctx, p2); !errors.Is(e, ErrCapacity) {
		t.Fatalf("sink capacity: %v", e)
	}
	if _, e = small.Admit(ctx, p1); e != nil {
		t.Fatal("capacity prevented duplicate resolution")
	}
	// A forged/nonmatching durable receipt cannot retire any obligation.
	fake := sinkFunc(func(_ context.Context, p ledger.Publication) (Admission, error) {
		return Admission{Receiver: "different", Publication: p}, nil
	})
	_, e = Run(ctx, source{p1}, fake, a, limits)
	if e != nil {
		t.Fatal(e)
	}
	if len(records(t, a)) != 1 {
		t.Fatal("mismatched receipt advanced confirmations")
	}
}

type host struct{}

func (host) Authorize(context.Context, ledger.Request) error { return nil }
func (host) Recover(context.Context, []ledger.Head) error    { return nil }
func (host) WithinEvaluation(_ context.Context, r ledger.Request, fn func(ledger.Evidence) error) error {
	return fn(ledger.Evidence{Registered: true, PublishAllowed: true, InternalAllowed: true, Schema: "v1", Rule: "r1", Attribution: "P", Cut: r.EvaluationID, Computation: rev.Computation{Key: r.Key, Present: true, Value: r.EvaluationID, Evaluation: "valid", Completeness: "complete_as_of_frontier"}})
}
func publisher(t *testing.T, dir string) *ledger.Ledger {
	t.Helper()
	l, e := ledger.Open(ctx, filepath.Join(dir, "publisher.db"), host{}, ledger.Options{MaxRecords: 20})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { l.Close() })
	return l
}
func TestDeliveryProcessHelper(t *testing.T) {
	dir := os.Getenv("DELIVERY_TEST_DIR")
	if dir == "" {
		return
	}
	mode := os.Getenv("DELIVERY_TEST_MODE")
	cut := func(s string) error {
		if mode == "receiver-"+s || mode == "sender-"+s {
			os.Exit(91)
		}
		return nil
	}
	o := opts
	o.Cut = cut
	s, e := Open(ctx, filepath.Join(dir, "sink.db"), "receiver", "P", o)
	if e != nil {
		os.Exit(42)
	}
	if mode == "receiver-before_commit" || mode == "receiver-after_commit" {
		_, e = s.Admit(ctx, publication("p1", "x", 1, ""))
		if e != nil {
			os.Exit(43)
		}
		os.Exit(44)
	}
	l, e := ledger.Open(ctx, filepath.Join(dir, "publisher.db"), host{}, ledger.Options{MaxRecords: 20})
	if e != nil {
		os.Exit(45)
	}
	ackOptions := opts
	ackOptions.Cut = func(point string) error {
		if mode == "ack-"+point {
			os.Exit(91)
		}
		return nil
	}
	a, e := Open(ctx, filepath.Join(dir, "acks.db"), "receiver", "P", ackOptions)
	if e != nil {
		os.Exit(46)
	}
	bound := limits
	bound.Cut = cut
	_, e = Run(ctx, l, s, a, bound)
	if e != nil {
		os.Exit(47)
	}
	os.Exit(48)
}
func TestDeliveryRealCrashRecovery(t *testing.T) {
	for _, mode := range []string{"receiver-before_commit", "receiver-after_commit", "sender-before_send", "sender-after_send", "ack-before_commit", "ack-after_commit"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			sender := mode != "receiver-before_commit" && mode != "receiver-after_commit"
			var original ledger.Publication
			if sender {
				l := publisher(t, dir)
				k := rev.Key{Context: "demo", Producer: "P", Subject: "x", Meaning: "m"}
				rc, e := l.Accept(ctx, ledger.Request{ID: "R1", Key: k, Generation: "G", EvaluationID: "eval1"})
				if e != nil || rc.Publication == nil {
					t.Fatalf("seed: %+v %v", rc, e)
				}
				original = *rc.Publication
				l.Close()
			}
			c := exec.Command(os.Args[0], "-test.run=^TestDeliveryProcessHelper$")
			c.Env = append(os.Environ(), "DELIVERY_TEST_DIR="+dir, "DELIVERY_TEST_MODE="+mode)
			out, e := c.CombinedOutput()
			var ex *exec.ExitError
			if !errors.As(e, &ex) || ex.ExitCode() != 91 {
				t.Fatalf("cut %s: %s %v", mode, out, e)
			}
			s := store(t, filepath.Join(dir, "sink.db"), opts)
			if !sender {
				want := 0
				if mode == "receiver-after_commit" {
					want = 1
				}
				if len(records(t, s)) != want {
					t.Fatalf("wrong receiver durability side: %v", records(t, s))
				}
				p := publication("p1", "x", 1, "")
				if _, e = s.Admit(ctx, p); e != nil {
					t.Fatal(e)
				}
				if len(records(t, s)) != 1 {
					t.Fatal("duplicate logical admission")
				}
				return
			}
			l := publisher(t, dir)
			pending, e := l.Pending(ctx)
			if e != nil || len(pending) != 1 || pending[0] != original {
				t.Fatalf("publisher recovery %v %v", pending, e)
			}
			a := store(t, filepath.Join(dir, "acks.db"), opts)
			expectedConfirmations := 1
			if mode == "ack-after_commit" {
				expectedConfirmations = 0
				if len(records(t, a)) != 1 {
					t.Fatal("durable sender acknowledgment lost")
				}
			} else if len(records(t, a)) != 0 {
				t.Fatal("unconfirmed acceptance persisted")
			}
			r, e := Run(ctx, l, s, a, limits)
			if e != nil || r.Confirmed != expectedConfirmations || len(records(t, s)) != 1 || records(t, s)[0] != original || len(records(t, a)) != 1 {
				t.Fatalf("P16 restart: %+v %v", r, e)
			}
			s.Close()
			a.Close()
			s = store(t, filepath.Join(dir, "sink.db"), opts)
			a = store(t, filepath.Join(dir, "acks.db"), opts)
			r, e = Run(ctx, l, s, a, limits)
			if e != nil || r.Calls != 0 {
				t.Fatalf("post-checkpoint recovery: %+v %v", r, e)
			}
		})
	}
}
