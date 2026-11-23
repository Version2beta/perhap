package viewjournal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
)

var ctx = context.Background()
var view = Identity{Context: "demo", Projection: "sum", Parameters: "default", Model: Model, Schema: Schema, AuthorizedView: "alice", Generation: "S"}
var limits = Limits{MaxEvaluations: 80, MaxPins: 2, MaxBatch: 10, MaxBytes: 2 << 20}

// authority's two assertions and private evidence generation form ONE locked
// finite cut. This is not a certificate invented from separate database reads.
type authority struct {
	mu                      sync.Mutex
	left, right             int64
	generation              string
	evidence                string
	evaluation, coverage    string
	available, null, denied bool
	floor                   uint64
}

func host() *authority {
	return &authority{left: 1, right: 2, generation: "g1", evaluation: "valid", coverage: "complete_as_of_frontier", available: true}
}
func (a *authority) Authorize(_ context.Context, v Identity) error {
	if a.denied || v.Model != Model || v.Schema != Schema || v.Context != "demo" || v.Projection != "sum" || v.Parameters != "default" || v.AuthorizedView == "" || v.Generation == "" {
		return ErrDenied
	}
	return nil
}
func (a *authority) Within(_ context.Context, r Request, fn func(Verified) error) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if r.EvaluationID != a.generation {
		return ErrStale
	}
	evidence := a.evidence
	if evidence == "" {
		evidence = "public-sum-scope"
	}
	p := Public{Available: a.available, Evaluation: a.evaluation, Completeness: a.coverage, Evidence: evidence}
	if a.available && !a.null {
		p.Value = &Value{a.left + a.right}
	}
	return fn(Verified{Public: p, PrivateCut: a.generation})
}
func (a *authority) Recover(_ context.Context, hs []Head) error {
	if a.floor > 0 {
		for _, h := range hs {
			if h.View == view && h.Checkpoint >= a.floor {
				return nil
			}
		}
		return ErrStale
	}
	return nil
}
func open(t *testing.T, path string, h Host, l Limits) *Journal {
	t.Helper()
	j, e := Open(ctx, path, h, l)
	if e != nil {
		t.Fatal(e)
	}
	if j == nil {
		t.Fatal("Open returned nil journal")
	}
	t.Cleanup(func() { j.Close() })
	return j
}
func head(t *testing.T, j *Journal, v Identity) Head {
	t.Helper()
	hs, e := j.Heads(ctx)
	if e != nil {
		t.Fatal(e)
	}
	for _, h := range hs {
		if h.View == v {
			return h
		}
	}
	return Head{View: v}
}
func request(t *testing.T, j *Journal, a *authority, id string, v Identity) Request {
	t.Helper()
	h := head(t, j, v)
	return Request{ID: id, View: v, EvaluationID: a.generation, ExpectedCheckpoint: h.Checkpoint, ExpectedRevision: h.Revision}
}
func accept(t *testing.T, j *Journal, a *authority, id string) Receipt {
	t.Helper()
	r, e := j.Accept(ctx, request(t, j, a, id, view))
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestViewSnapshot40Tail41(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.db")
	a := host()
	l := limits
	l.MaxEvaluations = 41
	l.MaxPins = 1
	j := open(t, path, a, l)
	for n := 1; n <= 40; n++ {
		a.left = int64(n)
		a.generation = fmt.Sprint("g", n)
		r := accept(t, j, a, fmt.Sprint("r", n))
		if r.Revision != uint64(n) {
			t.Fatal(r)
		}
	}
	p, e := j.Prepare(ctx, view)
	if e != nil || p.Snapshot.Revision != 40 {
		t.Fatalf("snapshot40: %+v %v", p, e)
	}
	if _, e = j.Prepare(ctx, view); !errors.Is(e, ErrCapacity) {
		t.Fatalf("pin capacity: %v", e)
	}
	// The cut is durably recorded, but no tail activation has occurred. Restart
	// before the concurrent successor; the plan must still pin its replay start.
	j.Close()
	j = open(t, path, a, l)
	start, done := make(chan struct{}), make(chan error, 1)
	a.left = 41
	a.generation = "g41"
	r := request(t, j, a, "r41", view)
	go func() { <-start; _, err := j.Accept(ctx, r); done <- err }()
	close(start)
	if e = <-done; e != nil {
		t.Fatal(e)
	}
	b, e := j.Read(ctx, p.ID, 40, 1)
	if e != nil || len(b.Records) != 1 || b.Records[0].Revision != 41 || b.Records[0].Previous != 40 || b.Through != 41 || b.After != 40 || b.SelectedHead != 41 || b.Records[0].Public.Value.Total != 43 {
		t.Fatalf("lost41/false cut: %+v %v", b, e)
	}
	a.generation = "g42"
	a.left = 42
	if _, e = j.Accept(ctx, request(t, j, a, "r42", view)); !errors.Is(e, ErrCapacity) {
		t.Fatalf("retention pressure: %v", e)
	}
	j.Close()
	j = open(t, path, a, l)
	b, e = j.Read(ctx, p.ID, 40, 1)
	if e != nil || b.Through != 41 || len(b.Records) != 1 {
		t.Fatalf("pressure/restart lost tail: %+v %v", b, e)
	}
	if e = j.Release(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = j.Read(ctx, p.ID, 40, 1); e == nil {
		t.Fatal("released pin still usable")
	}
	p2, e := j.Prepare(ctx, view)
	if e != nil || p2.Snapshot.Revision != 41 {
		t.Fatalf("new snapshot: %+v %v", p2, e)
	}
}
func TestViewEqualityFreshnessAndIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.db")
	a := host()
	j := open(t, path, a, limits)
	r1 := accept(t, j, a, "r1")
	if r1.Revision != 1 || !r1.Changed {
		t.Fatal(r1)
	}
	// One hidden/private source change, no public change; both inputs change but
	// their sum is equal. Private checkpoint advances, no public revision/offset leak.
	a.left = 2
	a.right = 1
	a.generation = "g2"
	r2 := accept(t, j, a, "r2")
	if r2.Checkpoint != 2 || r2.Revision != 1 || r2.Changed {
		t.Fatalf("hidden no-op: %+v", r2)
	}
	old := r1.Request
	old.ID = "old-evaluation"
	if _, e := j.Accept(ctx, old); !errors.Is(e, ErrStale) {
		t.Fatalf("old checkpoint %v", e)
	}
	old = request(t, j, a, "forged", view)
	old.EvaluationID = "g1"
	if _, e := j.Accept(ctx, old); !errors.Is(e, ErrStale) {
		t.Fatalf("caller freshness %v", e)
	}
	a.generation = "g3"
	a.evaluation = "failed"
	r3 := accept(t, j, a, "r3")
	if !r3.Changed || r3.Revision != 2 {
		t.Fatalf("status-only: %+v", r3)
	}
	a.evaluation = "valid"
	a.generation = "g4"
	r4 := accept(t, j, a, "r4")
	if r4.Revision != 3 {
		t.Fatal(r4)
	}
	j.Close()
	j = open(t, path, a, limits)
	retry, e := j.Accept(ctx, r1.Request)
	if e != nil || retry != r1 {
		t.Fatalf("old receipt changed %+v %v", retry, e)
	}
	bad := r1.Request
	bad.EvaluationID = "different"
	if _, e = j.Accept(ctx, bad); !errors.Is(e, ErrConflict) {
		t.Fatalf("request conflict %v", e)
	}
	// Same rendering belongs to independent authorized view/model identity, not
	// an assertion publication ordinal. A different authorized view starts at1.
	other := view
	other.AuthorizedView = "bob"
	r, e := j.Accept(ctx, request(t, j, a, "r1", other))
	if e != nil || r.Revision != 1 {
		t.Fatalf("view collision %+v %v", r, e)
	}
	a.denied = true
	if _, e = j.Prepare(ctx, view); !errors.Is(e, ErrDenied) {
		t.Fatalf("authority assumed %v", e)
	}
	a.denied = false
	shared := request(t, j, a, "concurrent-a", view)
	second := shared
	second.ID = "concurrent-b"
	out := make(chan error, 2)
	var wg sync.WaitGroup
	for _, r := range []Request{shared, second} {
		wg.Add(1)
		go func(r Request) { defer wg.Done(); _, err := j.Accept(ctx, r); out <- err }(r)
	}
	wg.Wait()
	close(out)
	success, stale := 0, 0
	for e := range out {
		if e == nil {
			success++
		} else if errors.Is(e, ErrStale) {
			stale++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatalf("concurrent acceptance: %d %d", success, stale)
	}
}
func TestViewNullBatchesAndInterruptedScan(t *testing.T) {
	a := host()
	path := filepath.Join(t.TempDir(), "view.db")
	j := open(t, path, a, limits)
	if _, e := j.Prepare(ctx, view); e == nil {
		t.Fatal("fabricated initial projection")
	}
	accept(t, j, a, "r1")
	p, e := j.Prepare(ctx, view)
	if e != nil {
		t.Fatal(e)
	}
	a.null = true
	a.generation = "null"
	accept(t, j, a, "r2")
	a.available = false
	a.generation = "unavailable"
	accept(t, j, a, "r3")
	b, e := j.Read(ctx, p.ID, 1, 1)
	if e != nil || b.Through != 2 || b.SelectedHead != 3 || len(b.Records) != 1 || !b.Records[0].Public.Available || b.Records[0].Public.Value != nil {
		t.Fatalf("available null batch %+v %v", b, e)
	}
	b, e = j.Read(ctx, p.ID, 2, 1)
	if e != nil || b.Through != 3 || b.Records[0].Public.Available {
		t.Fatalf("unavailable batch %+v %v", b, e)
	}
	if _, e = j.Read(ctx, p.ID, 0, 1); !errors.Is(e, ErrStale) {
		t.Fatalf("tail before snapshot %v", e)
	}
	if _, e = j.Read(ctx, p.ID, 3, limits.MaxBatch+1); !errors.Is(e, ErrCapacity) {
		t.Fatalf("unbounded batch %v", e)
	}
	j.Close()
	l := limits
	l.Cut = func(c string) error {
		if c == "during_read" {
			return errors.New("interrupted")
		}
		return nil
	}
	j = open(t, path, a, l)
	b, e = j.Read(ctx, p.ID, 1, 2)
	if e == nil || b.PlanID != "" || len(b.Records) != 0 {
		t.Fatalf("partial scan certified %+v %v", b, e)
	}
	c, cancel := context.WithCancel(ctx)
	cancel()
	if b, e = j.Read(c, p.ID, 1, 1); e == nil || b.PlanID != "" {
		t.Fatalf("cancelled scan certified %+v %v", b, e)
	}
}
func TestViewPublicMetadataEquality(t *testing.T) {
	a := host()
	j := open(t, filepath.Join(t.TempDir(), "equality.db"), a, limits)
	accept(t, j, a, "first")
	p, err := j.Prepare(ctx, view)
	if err != nil {
		t.Fatal(err)
	}
	a.generation = "g2"
	a.coverage = "incomplete"
	if r := accept(t, j, a, "coverage"); r.Revision != 2 || !r.Changed {
		t.Fatalf("public coverage lost: %+v", r)
	}
	a.generation = "g3"
	a.evidence = "authorized-public-change"
	if r := accept(t, j, a, "metadata"); r.Revision != 3 || !r.Changed {
		t.Fatalf("public metadata lost: %+v", r)
	}
	a.generation = "private-g4"
	if r := accept(t, j, a, "private-only"); r.Revision != 3 || r.Changed || r.Checkpoint != 4 {
		t.Fatalf("private evidence leaked: %+v", r)
	}
	if p.Snapshot.Public.Completeness != "complete_as_of_frontier" || p.Snapshot.Public.Evidence != "public-sum-scope" {
		t.Fatal("older record rewritten")
	}
}

func TestViewUnknownCorruptAndOwnership(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.db")
	a := host()
	l := limits
	l.Cut = func(s string) error {
		if s == "after_commit" {
			return errors.New("lost reply")
		}
		return nil
	}
	j := open(t, path, a, l)
	r := request(t, j, a, "r1", view)
	if _, e := j.Accept(ctx, r); !errors.Is(e, ErrUnknown) {
		t.Fatalf("lost reply: %v", e)
	}
	j.Close()
	j = open(t, path, a, limits)
	rc, e := j.Accept(ctx, r)
	if e != nil || rc.Revision != 1 || rc.Checkpoint != 1 {
		t.Fatalf("restart retry %+v %v", rc, e)
	}
	if owner, e := Open(ctx, path, a, limits); !errors.Is(e, ErrOwned) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("owner lock %v", e)
	}
	j.Close()
	a.floor = 2
	if owner, e := Open(ctx, path, a, limits); !errors.Is(e, ErrStale) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("stale recovery %v", e)
	}
	// Missing all previously committed journal bytes is not a new empty store.
	if err := os.Truncate(path, 0); err != nil {
		t.Fatal(err)
	}
	if owner, err := Open(ctx, path, host(), limits); !errors.Is(err, ErrUnavailable) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("missing journal reset: %v", err)
	}
	corrupt := filepath.Join(t.TempDir(), "bad.db")
	if e = os.WriteFile(corrupt, []byte("corrupt database"), 0600); e != nil {
		t.Fatal(e)
	}
	if owner, e := Open(ctx, corrupt, host(), limits); !errors.Is(e, ErrUnavailable) {
		if owner != nil {
			owner.Close()
		}
		t.Fatalf("corrupt reset %v", e)
	}
}
func TestViewProcessHelper(t *testing.T) {
	path := os.Getenv("VIEW_TEST_PATH")
	if path == "" {
		return
	}
	mode := os.Getenv("VIEW_TEST_MODE")
	l := limits
	l.Cut = func(point string) error {
		if point == mode || "plan-"+point == mode {
			os.Exit(91)
		}
		return nil
	}
	j, e := Open(ctx, path, host(), l)
	if e != nil {
		os.Exit(42)
	}
	if mode == "contender" {
		j.Close()
		os.Exit(43)
	}
	if mode == "plan-before_commit" || mode == "plan-after_commit" {
		_, e = j.Prepare(ctx, view)
	} else {
		_, e = j.Accept(ctx, Request{ID: "crash", View: view, EvaluationID: "g1"})
	}
	if e != nil {
		os.Exit(44)
	}
	os.Exit(45)
}
func child(t *testing.T, path, mode string) int {
	t.Helper()
	c := exec.Command(os.Args[0], "-test.run=^TestViewProcessHelper$")
	c.Env = append(os.Environ(), "VIEW_TEST_PATH="+path, "VIEW_TEST_MODE="+mode)
	out, e := c.CombinedOutput()
	var ex *exec.ExitError
	if errors.As(e, &ex) {
		return ex.ExitCode()
	}
	t.Fatalf("no controlled exit: %s %v", out, e)
	return 0
}
func TestViewPlanCrashRecovery(t *testing.T) {
	for _, cut := range []string{"plan-before_commit", "plan-after_commit"} {
		t.Run(cut, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "plans.db")
			a := host()
			j := open(t, path, a, limits)
			accept(t, j, a, "seed")
			j.Close()
			if code := child(t, path, cut); code != 91 {
				t.Fatalf("cut failed %d", code)
			}
			j = open(t, path, a, limits)
			plans, err := j.Plans(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if cut == "plan-before_commit" {
				if len(plans) != 0 {
					t.Fatal("partial pin survived")
				}
			} else {
				if len(plans) != 1 || plans[0].Snapshot.Revision != 1 {
					t.Fatalf("lost durable pin: %+v", plans)
				}
				a.left = 5
				a.generation = "g2"
				accept(t, j, a, "next")
				batch, err := j.Read(ctx, plans[0].ID, 1, 1)
				if err != nil || batch.Through != 2 || len(batch.Records) != 1 {
					t.Fatalf("recovered pin unusable: %+v %v", batch, err)
				}
			}
		})
	}
}

func TestViewRealCrashRecovery(t *testing.T) {
	for _, cut := range []string{"before_commit", "after_commit"} {
		t.Run(cut, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "view.db")
			if code := child(t, path, cut); code != 91 {
				t.Fatalf("cut failed %d", code)
			}
			j := open(t, path, host(), limits)
			hs, e := j.Heads(ctx)
			if e != nil {
				t.Fatal(e)
			}
			if cut == "before_commit" {
				if len(hs) != 0 {
					t.Fatal("partial transaction survived")
				}
			} else {
				if len(hs) != 1 || hs[0].Checkpoint != 1 || hs[0].Revision != 1 {
					t.Fatalf("lost commit: %+v", hs)
				}
				r, e := j.Accept(ctx, Request{ID: "crash", View: view, EvaluationID: "g1"})
				if e != nil || r.Revision != 1 {
					t.Fatalf("duplicate after crash: %+v %v", r, e)
				}
			}
			if code := child(t, path, "contender"); code != 42 {
				t.Fatalf("competing process %d", code)
			}
		})
	}
}
