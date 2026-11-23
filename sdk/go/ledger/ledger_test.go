package ledger

import (
	"context"
	"errors"
	"fmt"
	"github.com/version2beta/perhap/sdk/go/rev"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

var ctx = context.Background()
var k = rev.Key{Context: "demo", Producer: "verified-p", Subject: "x", Meaning: "m"}

type authority struct {
	mu                  sync.Mutex
	current             string
	registered, allowed bool
	floor               uint64
	floorKey            rev.Key
}

func (a *authority) Authorize(_ context.Context, r Request) error {
	if r.Key.Producer != "verified-p" && r.Key.Producer != "verified-p2" {
		return ErrDenied
	}
	return nil
}
func (a *authority) Recover(_ context.Context, hs []Head) error {
	if a.floor > 0 {
		found := false
		for _, h := range hs {
			if h.Key == a.floorKey && h.Checkpoint >= a.floor {
				found = true
			}
		}
		if !found {
			return ErrStale
		}
	}
	return nil
}
func (a *authority) WithinEvaluation(_ context.Context, r Request, fn func(Evidence) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.current != "" && r.EvaluationID != a.current {
		return ErrStale
	}
	e := Evidence{Registered: a.registered, PublishAllowed: a.allowed, InternalAllowed: true, Schema: "counter/v1", Rule: "r1", Attribution: r.Key.Producer, Cut: r.EvaluationID, Computation: rev.Computation{Key: r.Key, Present: true, Value: "V", Evaluation: "valid", Completeness: "complete_as_of_frontier"}}
	switch r.EvaluationID {
	case "absent":
		e.Computation.Present = false
	case "failed":
		e.Computation.Evaluation = "failed"
	case "null":
		e.Computation.Value = "null"
	case "changed":
		e.Computation.Value = "new"
	}
	return fn(e)
}
func auth() *authority { return &authority{registered: true, allowed: true} }
func open(t *testing.T, path string, a Host, o Options) *Ledger {
	t.Helper()
	l, e := Open(ctx, path, a, o)
	if e != nil {
		t.Fatal(e)
	}
	if l == nil {
		t.Fatal("Open returned no owner")
	}
	t.Cleanup(func() { l.Close() })
	return l
}
func req(id, eval string, h Head) Request {
	return Request{ID: id, Key: k, Generation: "G", EvaluationID: eval, ExpectedCheckpoint: h.Checkpoint, ExpectedRevision: h.Assertion.Revision, ExpectedPublicationID: h.Assertion.PublicationID}
}
func head(t *testing.T, l *Ledger) Head {
	t.Helper()
	hs, e := l.Heads(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, h := range hs {
		if h.Key == k {
			return h
		}
	}
	return Head{Key: k, Generation: "G"}
}
func accept(t *testing.T, l *Ledger, id, eval string) Receipt {
	t.Helper()
	r, e := l.Accept(ctx, req(id, eval, head(t, l)))
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestLedgerAcceptance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ledger.db")
	a := auth()
	l := open(t, path, a, Options{MaxRecords: 50})
	r1 := accept(t, l, "R1", "valid")
	if r1.Checkpoint != 1 || r1.Publication == nil || r1.Publication.Record.Revision != 1 {
		t.Fatalf("P01: %+v", r1)
	}
	r2 := accept(t, l, "R2", "valid")
	if r2.Checkpoint != 2 || r2.Publication != nil {
		t.Fatalf("P01 no-op: %+v", r2)
	}
	r3 := accept(t, l, "R3", "failed")
	r4 := accept(t, l, "R4", "valid")
	if r3.Publication == nil || r3.Publication.Record.Status != "unresolved" || r3.Publication.Record.Value != "V" || r4.Publication == nil || r4.Publication.Record.Status != "current" || r4.Publication.Record.Revision != 3 || r4.Publication.Record.Supersedes != r3.Publication.Record.ID {
		t.Fatalf("P02: %+v %+v", r3, r4)
	}
	l.Close()
	l = open(t, path, a, Options{MaxRecords: 50})
	retry, e := l.Accept(ctx, r1.Request)
	if e != nil || !reflect.DeepEqual(retry, r1) {
		t.Fatalf("P11 retry: %+v %v", retry, e)
	}
	bad := r1.Request
	bad.EvaluationID = "changed"
	if _, e = l.Accept(ctx, bad); !errors.Is(e, ErrConflict) {
		t.Fatalf("P11 conflict: %v", e)
	}
	stale := req("old", "valid", Head{})
	if _, e = l.Accept(ctx, stale); !errors.Is(e, ErrStale) {
		t.Fatalf("P07 old checkpoint: %v", e)
	}
	a.current = "changed"
	fresh := req("source-stale", "valid", head(t, l))
	if _, e = l.Accept(ctx, fresh); !errors.Is(e, ErrStale) {
		t.Fatalf("P10 source: %v", e)
	}
	a.current = ""
	h := head(t, l)
	shared := req("concurrent-a", "valid", h)
	other := shared
	other.ID = "concurrent-b"
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, r := range []Request{shared, other} {
		wg.Add(1)
		go func(r Request) { defer wg.Done(); _, e := l.Accept(ctx, r); results <- e }(r)
	}
	wg.Wait()
	close(results)
	ok, st := 0, 0
	for e := range results {
		if e == nil {
			ok++
		} else if errors.Is(e, ErrStale) {
			st++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || st != 1 {
		t.Fatalf("P09: success=%d stale=%d", ok, st)
	}
	pending, e := l.Pending(ctx)
	if e != nil || len(pending) != 3 {
		t.Fatalf("pending %v %v", pending, e)
	}
	pending[0].Record.Value = "mutated"
	again, _ := l.Pending(ctx)
	if again[0].Record.Value != "V" {
		t.Fatal("caller mutated immutable record")
	}
}
func TestLedgerInitialAndAuthority(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			l := open(t, filepath.Join(t.TempDir(), "l.db"), &authority{registered: true, allowed: allowed}, Options{MaxRecords: 10})
			r := accept(t, l, "absent", "absent")
			if r.Publication != nil || r.Checkpoint != 1 {
				t.Fatalf("P03: %+v", r)
			}
			r = accept(t, l, "fail", "failed")
			if allowed {
				if r.Publication == nil || r.Publication.Record.Revision != 1 || r.Publication.ValueAvailable || r.Publication.Record.Value != "" {
					t.Fatalf("P04: %+v", r)
				}
			} else if r.Publication != nil {
				t.Fatal("unauthorized initial status")
			}
			r = accept(t, l, "recovery", "null")
			if allowed && (r.Publication == nil || !r.Publication.ValueAvailable || r.Publication.Record.Value != "null") {
				t.Fatalf("P04 null: %+v", r)
			}
			r = accept(t, l, "retract", "absent")
			if allowed && (r.Publication == nil || r.Publication.Record.Op != "retract") {
				t.Fatalf("P05: %+v", r)
			}
		})
	}
	l := open(t, filepath.Join(t.TempDir(), "forbidden.db"), &authority{}, Options{MaxRecords: 3})
	if _, e := l.Accept(ctx, req("probe", "failed", Head{})); !errors.Is(e, ErrDenied) {
		t.Fatalf("P06: %v", e)
	}
	hs, _ := l.Heads(ctx)
	if len(hs) != 0 {
		t.Fatal("probe registered key")
	}
}
func TestLedgerIndependentKeysAndStaleFailure(t *testing.T) {
	l := open(t, filepath.Join(t.TempDir(), "keys.db"), auth(), Options{MaxRecords: 20})
	first := accept(t, l, "one", "valid")
	secondRequest := req("one", "valid", Head{})
	secondRequest.Key.Producer = "verified-p2"
	second, err := l.Accept(ctx, secondRequest)
	if err != nil || second.Publication == nil || second.Publication.Record.Revision != 1 || second.Publication.Record.ID == first.Publication.Record.ID {
		t.Fatalf("P17 independent keys: %+v %v", second, err)
	}
	old := req("old", "valid", head(t, l))
	accept(t, l, "failed", "failed")
	if _, err := l.Accept(ctx, old); !errors.Is(err, ErrStale) {
		t.Fatalf("P08 old evaluation cleared failure: %v", err)
	}
	if head(t, l).Assertion.Status != "unresolved" {
		t.Fatal("P08 unresolved overwritten")
	}
}

func TestLedgerUnknownCapacityAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "l.db")
	a := auth()
	l := open(t, path, a, Options{MaxRecords: 1, Cut: func(s string) error {
		if s == "after_commit" {
			return errors.New("lost response")
		}
		return nil
	}})
	r := req("lost", "valid", Head{})
	if _, e := l.Accept(ctx, r); !errors.Is(e, ErrUnknown) {
		t.Fatalf("P14a: %v", e)
	}
	l.Close()
	l = open(t, path, a, Options{MaxRecords: 1})
	got, e := l.Accept(ctx, r)
	if e != nil || got.Publication == nil || got.Checkpoint != 1 {
		t.Fatalf("P13: %+v %v", got, e)
	}
	if _, e = l.Accept(ctx, req("overflow", "valid", head(t, l))); !errors.Is(e, ErrCapacity) {
		t.Fatalf("B01: %v", e)
	}
	l.Close()
	a.floor = 2
	a.floorKey = k
	if owner, e := Open(ctx, path, a, Options{MaxRecords: 10}); !errors.Is(e, ErrStale) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("P18 stale restore: %v", e)
	}
	corrupt := filepath.Join(t.TempDir(), "bad.db")
	if e = os.WriteFile(corrupt, []byte("not a database"), 0600); e != nil {
		t.Fatal(e)
	}
	if owner, e := Open(ctx, corrupt, auth(), Options{MaxRecords: 10}); !errors.Is(e, ErrUnavailable) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("P14c: %v", e)
	}
	if owner, e := Open(ctx, filepath.Join(t.TempDir(), "missing", "l.db"), auth(), Options{MaxRecords: 10}); !errors.Is(e, ErrUnavailable) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("P14b: %v", e)
	}
}
func TestLedgerSharedSchedules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scheduled.db")
	a := auth()
	l := open(t, path, a, Options{MaxRecords: 20})
	failed := accept(t, l, "initial-failure", "failed")
	retracted := accept(t, l, "initial-disappeared", "absent")
	if failed.Publication == nil || failed.Publication.ValueAvailable || retracted.Publication == nil || retracted.Publication.Record.Op != "retract" || retracted.Publication.Record.Revision != 2 {
		t.Fatalf("P05: %+v %+v", failed, retracted)
	}
	l.Close()
	backup, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	l = open(t, path, a, Options{MaxRecords: 20})
	accept(t, l, "after-backup", "null")
	l.Close()
	if err = os.WriteFile(path, backup, 0600); err != nil {
		t.Fatal(err)
	}
	a.floor, a.floorKey = 3, k
	if owner, err := Open(ctx, path, a, Options{MaxRecords: 20}); !errors.Is(err, ErrStale) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("P18 actual restored backup accepted: %v", err)
	}
	truncated := filepath.Join(t.TempDir(), "empty.db")
	if err := os.WriteFile(truncated, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if owner, err := Open(ctx, truncated, auth(), Options{MaxRecords: 20}); !errors.Is(err, ErrUnavailable) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("P14c empty committed storage fallback: %v", err)
	}
}

func TestLedgerProcessHelper(t *testing.T) {
	path := os.Getenv("LEDGER_TEST_PATH")
	if path == "" {
		return
	}
	mode := os.Getenv("LEDGER_TEST_MODE")
	if mode == "contender" {
		owner, e := Open(ctx, path, auth(), Options{MaxRecords: 10})
		if errors.Is(e, ErrOwned) {
			os.Exit(42)
		}
		if owner != nil {
			owner.Close()
		}
		os.Exit(43)
	}
	cut := func(s string) error {
		if s == mode {
			os.Exit(91)
		}
		return nil
	}
	l, e := Open(ctx, path, auth(), Options{MaxRecords: 10, Cut: cut})
	if e != nil {
		os.Exit(44)
	}
	_, e = l.Accept(ctx, req("crashed", "valid", Head{}))
	if e != nil {
		os.Exit(45)
	}
	os.Exit(46)
}
func runProcess(t *testing.T, path, mode string) int {
	t.Helper()
	c := exec.Command(os.Args[0], "-test.run=^TestLedgerProcessHelper$")
	c.Env = append(os.Environ(), "LEDGER_TEST_PATH="+path, "LEDGER_TEST_MODE="+mode)
	out, e := c.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(e, &exit) {
		return exit.ExitCode()
	}
	t.Fatalf("child did not exit at cut: %s %v", out, e)
	return 0
}
func TestLedgerRealCrashAndOwners(t *testing.T) {
	for _, cut := range []string{"before_tx", "after_write", "before_commit", "after_commit"} {
		t.Run(cut, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "l.db")
			if code := runProcess(t, path, cut); code != 91 {
				t.Fatalf("crash cut exit %d", code)
			}
			l := open(t, path, auth(), Options{MaxRecords: 10})
			hs, _ := l.Heads(ctx)
			p, _ := l.Pending(ctx)
			if cut == "after_commit" {
				if len(hs) != 1 || hs[0].Checkpoint != 1 || len(p) != 1 {
					t.Fatalf("P12 post durability: %v %v", hs, p)
				}
				r, e := l.Accept(ctx, req("crashed", "valid", Head{}))
				if e != nil || r.Publication == nil || r.Publication.Record.ID != p[0].Record.ID {
					t.Fatalf("P13 identity: %+v %v", r, e)
				}
			} else if len(hs) != 0 || len(p) != 0 {
				t.Fatalf("P12 partial state: %v %v", hs, p)
			}
			if code := runProcess(t, path, "contender"); code != 42 {
				t.Fatalf("P15 owner exclusion: %d", code)
			}
			l.Close()
			l = open(t, path, auth(), Options{MaxRecords: 10})
			_ = l
		})
	}
}
