package submission

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSubmissionVersionedFacts(t *testing.T) {
	h := &testHost{admission: "pending"}
	o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
	in := input("1")
	v1 := submit(t, o, in)
	h.admission = "admitted"
	v2 := submit(t, o, in)
	h.codec.fail = true
	if e := o.Evaluate(bg, in.Event.ID, "failed"); e != nil {
		t.Fatal(e)
	}
	v3 := resolve(t, o, in.Event.ID).Facts
	// Independent complete-fact consumer oracle: older facts cannot regress and equal
	// version with different content is conflict, never a rewrite of current facts.
	seen := map[uint64]Facts{}
	var current Facts
	merge := func(f Facts) bool {
		if old, ok := seen[f.Version]; ok && !reflect.DeepEqual(old, f) {
			return false
		}
		seen[f.Version] = f
		if f.Version > current.Version {
			current = f
		}
		return true
	}
	for _, f := range []Facts{v3, v1, v2} {
		if !merge(f) {
			t.Fatal("valid historical fact conflict")
		}
	}
	if current != v3 || current.Admission != "admitted" || current.Evaluation != "failed" || v1.Version != 1 || v2.Version != 2 || v3.Version != 3 {
		t.Fatal("S03 knowledge regressed")
	}
	changed := v3
	changed.Evaluation = "evaluated"
	if merge(changed) || current != v3 {
		t.Fatal("S04 last writer won")
	}
	if resolve(t, o, in.Event.ID).Facts != v3 {
		t.Fatal("consumer conflict mutated authoritative facts")
	}
}
func TestSubmissionCloseAndRevocation(t *testing.T) {
	t.Run("close-during-admission", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "s.db")
		entered := make(chan struct{})
		h := &testHost{pause: func(c context.Context) { close(entered); <-c.Done() }}
		o := open(t, path, h, limits)
		in := input("1")
		done := make(chan error, 1)
		go func() {
			done <- o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { t.Error("canceled release"); return nil })
		}()
		<-entered
		o.Close()
		if e := <-done; !errors.Is(e, context.Canceled) {
			t.Fatal(e)
		}
		h.pause = nil
		o = open(t, path, h, limits)
		if r := resolve(t, o, in.Event.ID); r.Facts.Admission != "pending" || !r.Facts.Receipt {
			t.Fatal("close undid or promoted receipt")
		}
		submit(t, o, in)
	})
	t.Run("revoke-first", func(t *testing.T) {
		entered, finish := make(chan struct{}), make(chan struct{})
		h := &testHost{pause: func(context.Context) { close(entered); <-finish }}
		o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
		done := make(chan error, 1)
		calls := 0
		go func() {
			done <- o.Client("demo").Submit(bg, input("1"), func(context.Context, Reply) error { calls++; return nil })
		}()
		<-entered
		h.revoke()
		close(finish)
		if e := <-done; !errors.Is(e, ErrDenied) || calls != 0 {
			t.Fatal("revoked facts released")
		}
	})
	t.Run("release-first", func(t *testing.T) {
		h := &testHost{}
		o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
		in := input("1")
		entered, finish, revoked := make(chan struct{}), make(chan struct{}), make(chan struct{})
		done := make(chan error, 1)
		calls := 0
		go func() {
			done <- o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { calls++; close(entered); <-finish; return nil })
		}()
		<-entered
		go func() { h.revoke(); close(revoked) }()
		close(finish)
		if e := <-done; e != nil {
			t.Fatal(e)
		}
		<-revoked
		if e := o.Client("demo").Resolve(bg, in.Event.ID, func(context.Context, Reply) error { calls++; return nil }); !errors.Is(e, ErrDenied) || calls != 1 {
			t.Fatal("later release escaped")
		}
	})
}
func TestSubmissionSemanticContextAndBounds(t *testing.T) {
	h := &testHost{}
	in := input("1")
	changed := in
	changed.Event.Context = "other"
	// Same-ID changed context is checked as identity, not as a reconciliation key.
	path := filepath.Join(t.TempDir(), "context.db")
	owner, e := Open(bg, path, &allowContexts{testHost: h}, limits)
	if e != nil {
		t.Fatal(e)
	}
	defer owner.Close()
	f := submit(t, owner, in)
	if e = owner.Client("other").Submit(bg, changed, func(context.Context, Reply) error { return nil }); !errors.Is(e, ErrConflict) {
		t.Fatalf("context conflict %v", e)
	}
	if resolve(t, owner, in.Event.ID).Facts != f {
		t.Fatal("original context overwritten")
	}
	for _, data := range []string{"9007199254740992", "null", "{}", "1 2"} {
		bad := input(data)
		if e = owner.Client("demo").Submit(bg, bad, func(context.Context, Reply) error { t.Fatal("invalid semantic data admitted"); return nil }); !errors.Is(e, ErrInvalid) {
			t.Fatalf("unsupported payload %s: %v", data, e)
		}
	}
}

type allowContexts struct{ *testHost }

func (h *allowContexts) Authorize(c context.Context, op, id, scope, binding string) (Actor, error) {
	if scope == "other" {
		scope = "demo"
	}
	return h.testHost.Authorize(c, op, id, scope, binding)
}
