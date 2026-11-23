package submission

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	perhap "github.com/version2beta/perhap/sdk/go"
	"github.com/version2beta/perhap/sdk/go/core"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

var bg = context.Background()
var limits = Limits{MaxEvents: 100, MaxEntries: 1000, MaxBytes: 2 << 20, MaxEventBytes: 1024, Timeout: time.Second}

type counterCodec struct{ fail bool }

func (c *counterCodec) Normalize(e perhap.Event) (perhap.Event, error) {
	var n json.Number
	if json.Unmarshal(e.Data, &n) != nil {
		return e, ErrInvalid
	}
	v, err := strconv.ParseFloat(string(n), 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || math.Trunc(v) != v || math.Abs(v) > 9007199254740991 || (e.Type != "add" && e.Type != "set") {
		return e, ErrInvalid
	}
	e.Data = []byte(strconv.FormatInt(int64(v), 10))
	return e, nil
}
func (c *counterCodec) CoreEvent(e perhap.Event, actor string) (core.Event, error) {
	out := core.Event{ID: e.ID, Context: e.Context, Type: e.Type, Payload: string(e.Data), AssertedBy: actor, Subjects: []string{e.EntityID}}
	if e.CausationID != nil {
		out.Dependencies = []core.Dependency{{ID: *e.CausationID, Kind: "local"}}
	}
	return out, nil
}
func (c *counterCodec) Model() core.Model[int64] {
	return core.Model[int64]{Initialize: func() (int64, error) { return 0, nil }, Clone: func(v int64) (int64, error) { return v, nil }, Reduce: func(v int64, e core.Event) (int64, error) {
		if c.fail {
			return v, errors.New("private computation failure")
		}
		n, err := strconv.ParseInt(e.Payload, 10, 64)
		if e.Type == "set" {
			return n, err
		}
		return v + n, err
	}, Validate: func(int64) (bool, error) { return true, nil }, Select: func(int64, []core.Event) (core.Selection, error) { return core.Selection{}, nil }}
}

type testHost struct {
	mu        sync.Mutex
	denied    bool
	codec     counterCodec
	admission string
	principal string
	pause     func(context.Context)
	minimum   map[string]uint64
}

func (h *testHost) Authorize(ctx context.Context, op, id, scope, binding string) (Actor, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.denied || scope != "demo" || binding == "forged" {
		return Actor{}, ErrDenied
	}
	if ctx.Value("authority") == "missing" {
		return Actor{}, ErrBlocked
	}
	p := h.principal
	if p == "" {
		p = "alice"
	}
	return Actor{Requester: p, Asserting: p}, nil
}
func (h *testHost) Codec(schema string) (Codec, error) {
	if schema != "counter/v1" {
		return nil, ErrInvalid
	}
	return &h.codec, nil
}
func (h *testHost) Admission(ctx context.Context, a Actor, in Input, fn func(string) error) error {
	if h.pause != nil {
		h.pause(ctx)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	decision := h.admission
	if decision == "" {
		decision = "admitted"
	}
	if decision == "pending" {
		return nil
	}
	return fn(decision)
}
func (h *testHost) Release(ctx context.Context, a Actor, id string, fn func() error) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.denied {
		return ErrDenied
	}
	return fn()
}
func (h *testHost) Recover(ctx context.Context, g string, heads []Head) error {
	for id, v := range h.minimum {
		found := false
		for _, head := range heads {
			if head.EventID == id && head.Version >= v {
				found = true
			}
		}
		if !found {
			return ErrIntegrity
		}
	}
	return nil
}
func (h *testHost) revoke() { h.mu.Lock(); defer h.mu.Unlock(); h.denied = true }
func input(data string) Input {
	return Input{Event: perhap.Event{ID: uuid.Must(uuid.NewV6()).String(), Context: "demo", EntityID: "one", Type: "add", CorrelationID: "correlation", Data: []byte(data)}, Schema: "counter/v1", AttributionBinding: "host-binding"}
}
func open(t *testing.T, path string, h *testHost, l Limits) *Owner {
	t.Helper()
	o, e := Open(bg, path, h, l)
	if e != nil || o == nil {
		t.Fatalf("Open: %v", e)
	}
	t.Cleanup(func() { o.Close() })
	return o
}
func resolve(t *testing.T, o *Owner, id string) Reply {
	t.Helper()
	var r Reply
	e := o.Client("demo").Resolve(bg, id, func(_ context.Context, v Reply) error { r = v; return nil })
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func submit(t *testing.T, o *Owner, in Input) Facts {
	t.Helper()
	var r Reply
	e := o.Client("demo").Submit(bg, in, func(_ context.Context, v Reply) error { r = v; return nil })
	if e != nil || r.Kind != "facts" {
		t.Fatalf("Submit: %v kind=%s", e, r.Kind)
	}
	return r.Facts
}
func TestSubmissionIdentityAndEvaluation(t *testing.T) {
	h := &testHost{}
	path := filepath.Join(t.TempDir(), "s.db")
	o := open(t, path, h, limits)
	in := input("1.0")
	f := submit(t, o, in)
	if !f.Receipt || f.Admission != "admitted" || f.Evaluation != "not-established" || f.Version != 2 || f.Attribution != "alice" {
		t.Fatalf("conflated receipt/admission/evaluation: %+v", f)
	}
	in.Event.Data = []byte("1")
	if again := submit(t, o, in); again != f {
		t.Fatal("equivalent numeric retry changed decision")
	}
	if e := o.Evaluate(bg, in.Event.ID, "attempt1"); e != nil {
		t.Fatal(e)
	}
	f = resolve(t, o, in.Event.ID).Facts
	if f.Version != 3 || f.Evaluation != "evaluated" || f.Admission != "admitted" || f.Model != "counter/v1" || f.Evidence == "" {
		t.Fatalf("actual CORE result missing: %+v", f)
	}
	if e := o.Evaluate(bg, in.Event.ID, "attempt1"); e != nil {
		t.Fatal(e)
	}
	if resolve(t, o, in.Event.ID).Facts != f {
		t.Fatal("duplicate evaluation allocated version")
	}
	h.codec.fail = true
	if e := o.Evaluate(bg, in.Event.ID, "attempt2"); e != nil {
		t.Fatal(e)
	}
	f = resolve(t, o, in.Event.ID).Facts
	if f.Version != 4 || f.Evaluation != "failed" || f.Admission != "admitted" {
		t.Fatal("evaluation erased admission")
	}
	o.Close()
	h.codec.fail = false
	o = open(t, path, h, limits)
	if latest := submit(t, o, in); latest != f {
		t.Fatal("restart retry returned stale receipt only")
	}
	for _, change := range []func(*Input){func(x *Input) { x.Event.Data = []byte("2") }, func(x *Input) { x.Event.Type = "set" }, func(x *Input) { x.Event.CorrelationID = "other" }, func(x *Input) { c := uuid.Must(uuid.NewV6()).String(); x.Event.CausationID = &c }} {
		x := in
		change(&x)
		if e := o.Client("demo").Submit(bg, x, func(context.Context, Reply) error { t.Fatal("conflict released original data"); return nil }); !errors.Is(e, ErrConflict) {
			t.Fatalf("semantic conflict %v", e)
		}
	}
	h.principal = "bob"
	if e := o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { return nil }); !errors.Is(e, ErrConflict) {
		t.Fatal("attribution overwritten")
	}
	h.principal = "alice"
	if resolve(t, o, in.Event.ID).Facts != f {
		t.Fatal("conflict rewrote authoritative facts")
	}
}
func TestSubmissionConcurrentAndLookupCuts(t *testing.T) {
	h := &testHost{}
	o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
	in := input("3")
	if r := resolve(t, o, in.Event.ID); r.Kind != "no-retained-knowledge" {
		t.Fatal("absence inferred rejection")
	}
	start := make(chan struct{})
	done := make(chan error, 12)
	for n := 0; n < 12; n++ {
		go func() {
			<-start
			done <- o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { return nil })
		}()
	}
	close(start)
	for n := 0; n < 12; n++ {
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}
	if r := resolve(t, o, in.Event.ID); r.Facts.Version != 2 {
		t.Fatal("duplicate admission")
	}
	different := in
	different.Event.Data = []byte("4")
	go func() {
		done <- o.Client("demo").Submit(bg, different, func(context.Context, Reply) error { return nil })
	}()
	go func() { done <- o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { return nil }) }()
	a, b := <-done, <-done
	if !(errors.Is(a, ErrConflict) && b == nil || errors.Is(b, ErrConflict) && a == nil) {
		t.Fatal("concurrent conflicting decision overwrote original")
	}
}
func TestSubmissionPendingAuthorityAndCancellation(t *testing.T) {
	h := &testHost{admission: "pending"}
	o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
	in := input("2")
	if f := submit(t, o, in); f.Version != 1 || f.Admission != "pending" || f.Evaluation != "not-established" {
		t.Fatal("receipt implied admission")
	}
	if e := o.Evaluate(bg, in.Event.ID, "no"); !errors.Is(e, ErrBlocked) {
		t.Fatal("unadmitted event evaluated")
	}
	h.admission = "unresolved"
	submit(t, o, in)
	h.admission = "admitted"
	submit(t, o, in)
	if e := o.Evaluate(bg, in.Event.ID, "after-unresolved"); e != nil {
		t.Fatalf("admitted event lost evaluation availability: %v", e)
	}
	missing := context.WithValue(bg, "authority", "missing")
	calls := 0
	if e := o.Client("demo").Resolve(missing, in.Event.ID, func(context.Context, Reply) error { calls++; return nil }); !errors.Is(e, ErrBlocked) || calls != 0 {
		t.Fatal("missing authority fabricated facts")
	}
	forged := in
	forged.AttributionBinding = "forged"
	if e := o.Client("demo").Submit(bg, forged, func(context.Context, Reply) error { calls++; return nil }); !errors.Is(e, ErrDenied) || calls != 0 {
		t.Fatal("forged binding admitted")
	}
	c, cancel := context.WithCancel(bg)
	e := o.Client("demo").Submit(c, in, func(context.Context, Reply) error { cancel(); return nil })
	if !errors.Is(e, ErrRelease) {
		t.Fatalf("possible response exposure treated as rollback: %v", e)
	}
	if resolve(t, o, in.Event.ID).Facts.Admission != "admitted" {
		t.Fatal("cancel undid admission")
	}
	unknown := input("1")
	cause := uuid.Must(uuid.NewV6()).String()
	unknown.Event.CausationID = &cause
	submit(t, o, unknown)
	if e := o.Evaluate(bg, unknown.Event.ID, "blocked-cause"); !errors.Is(e, ErrBlocked) {
		t.Fatal("blocked event reported completed evaluation")
	}
	if resolve(t, o, unknown.Event.ID).Facts.Evaluation != "not-established" {
		t.Fatal("missing history fabricated evaluation")
	}
}
func TestSubmissionFailureCapacityAndRecovery(t *testing.T) {
	for _, phase := range []string{"receipt:before_tx", "receipt:before_commit", "receipt:after_commit", "admission:after_commit"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "s.db")
			h := &testHost{}
			l := limits
			l.Cut = func(at string) error {
				if at == phase {
					return errors.New("private storage fault")
				}
				return nil
			}
			o := open(t, path, h, l)
			in := input("1")
			e := o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { t.Fatal("failed commit returned facts"); return nil })
			after := phase == "receipt:after_commit" || phase == "admission:after_commit"
			want := ErrUnavailable
			if after {
				want = ErrUnknown
			}
			if !errors.Is(e, want) {
				t.Fatalf("failure category %v", e)
			}
			o.Close()
			o = open(t, path, h, limits)
			r := resolve(t, o, in.Event.ID)
			if !after && r.Kind != "no-retained-knowledge" {
				t.Fatal("precommit failure committed")
			}
			if after && r.Kind != "facts" {
				t.Fatal("committed receipt disappeared")
			}
			submit(t, o, in)
			if resolve(t, o, in.Event.ID).Facts.Admission != "admitted" {
				t.Fatal("same identity failed reconciliation")
			}
		})
	}
	h := &testHost{}
	l := limits
	l.MaxEvents = 1
	l.MaxEntries = 4
	o := open(t, filepath.Join(t.TempDir(), "cap.db"), h, l)
	in := input("1")
	f := submit(t, o, in)
	if e := o.Client("demo").Submit(bg, input("2"), func(context.Context, Reply) error { return nil }); !errors.Is(e, ErrCapacity) {
		t.Fatal("capacity not refused")
	}
	if submit(t, o, in) != f {
		t.Fatal("capacity blocked duplicate lookup")
	}
	if e := o.Evaluate(bg, in.Event.ID, "reserved"); e != nil {
		t.Fatalf("completion capacity was not reserved: %v", e)
	}
	path := filepath.Join(t.TempDir(), "corrupt.db")
	if e := os.WriteFile(path, []byte("not a database"), 0600); e != nil {
		t.Fatal(e)
	}
	if bad, e := Open(bg, path, h, limits); e == nil || bad != nil {
		t.Fatal("corruption became fresh authoritative state")
	}
}
func TestSubmissionRealCrashAndOwner(t *testing.T) {
	if cut := os.Getenv("SUBMISSION_CRASH_CUT"); cut != "" {
		h := &testHost{}
		l := limits
		l.Cut = func(at string) error {
			if at == cut {
				os.Exit(77)
			}
			return nil
		}
		o, e := Open(bg, os.Getenv("SUBMISSION_CRASH_DB"), h, l)
		if e != nil {
			t.Fatal(e)
		}
		in := input("1")
		in.Event.ID = os.Getenv("SUBMISSION_CRASH_ID")
		o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { return nil })
		t.Fatal("crash cut missed")
	}
	for _, cut := range []string{"receipt:before_commit", "receipt:after_commit", "admission:before_commit", "admission:after_commit"} {
		t.Run(cut, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "s.db")
			h := &testHost{}
			o := open(t, path, h, limits)
			if other, e := Open(bg, path, h, limits); e == nil || other != nil {
				t.Fatal("competing owner allowed")
			}
			o.Close()
			in := input("1")
			command := exec.Command(os.Args[0], "-test.run=^TestSubmissionRealCrashAndOwner$")
			command.Env = append(os.Environ(), "SUBMISSION_CRASH_CUT="+cut, "SUBMISSION_CRASH_DB="+path, "SUBMISSION_CRASH_ID="+in.Event.ID)
			out, e := command.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(e, &exit) || exit.ExitCode() != 77 {
				t.Fatalf("child cut %v %s", e, out)
			}
			o = open(t, path, h, limits)
			r := resolve(t, o, in.Event.ID)
			if cut == "receipt:before_commit" {
				if r.Kind != "no-retained-knowledge" {
					t.Fatal("rolled back receipt survived")
				}
			} else {
				want := "pending"
				if cut == "admission:after_commit" {
					want = "admitted"
				}
				if r.Kind != "facts" || r.Facts.Admission != want {
					t.Fatalf("wrong durable side %+v", r)
				}
			}
			submit(t, o, in)
			if resolve(t, o, in.Event.ID).Facts.Version != 2 {
				t.Fatal("crash retry duplicated admission")
			}
		})
	}
}
