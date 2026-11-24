package submission

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestReadOnlySemanticResolution(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.db")
	h := &testHost{}
	o := open(t, path, h, limits)
	in := input("1.0")
	facts := submit(t, o, in)
	if e := o.Evaluate(bg, in.Event.ID, "evaluate"); e != nil {
		t.Fatal(e)
	}
	facts = resolve(t, o, in.Event.ID).Facts
	o.Close()
	h.admission = "pending"
	o = open(t, path, h, limits)
	get := func(v Input) Reply {
		var r Reply
		e := o.Client("demo").ResolveMatching(bg, v, func(_ context.Context, out Reply) error { r = out; return nil })
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	in.Event.Data = []byte("1")
	if r := get(in); r.Kind != "facts" || r.Facts != facts {
		t.Fatal("normalized match lost current facts")
	}
	bad := in
	bad.Event.Data = []byte("2")
	if r := get(bad); r.Kind != "identity-conflict" || r.Facts.EventID != "" {
		t.Fatal("mismatch trusted echoed semantic claim")
	}
	h.principal = "bob"
	if r := get(in); r.Kind != "identity-conflict" {
		t.Fatal("attribution mismatch ignored")
	}
	h.principal = "alice"
	missing := input("1")
	if r := get(missing); r.Kind != "no-retained-knowledge" {
		t.Fatal("missing knowledge manufactured")
	}
	if r := resolve(t, o, missing.Event.ID); r.Kind != "no-retained-knowledge" {
		t.Fatal("lookup created receipt")
	}
	if r := resolve(t, o, in.Event.ID); r.Facts != facts {
		t.Fatal("matching resolution mutated durable facts")
	}
	h.denied = true
	calls := 0
	if e := o.Client("demo").ResolveMatching(bg, in, func(context.Context, Reply) error { calls++; return nil }); !errors.Is(e, ErrDenied) || calls != 0 {
		t.Fatal("denied matching metadata escaped")
	}
}
func TestMatchingNoAdmissionAndFencedMismatch(t *testing.T) {
	h := &testHost{admission: "pending"}
	o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
	in := input("1")
	before := submit(t, o, in)
	h.pause = func(context.Context) { t.Error("read-only lookup called admission") }
	var out Reply
	if e := o.Client("demo").ResolveMatching(bg, in, func(_ context.Context, r Reply) error { out = r; return nil }); e != nil || out.Facts != before {
		t.Fatal("pending match changed state")
	}
	h.pause = nil
	// Release host revokes after lookup authorization, before conflict release.
	wrapped := &matchingRevoke{testHost: h}
	o.Close()
	path := filepath.Join(t.TempDir(), "other.db")
	owner, e := Open(bg, path, wrapped, limits)
	if e != nil {
		t.Fatal(e)
	}
	defer owner.Close()
	submit(t, owner, in)
	wrapped.armed = true
	in.Event.Data = []byte("2")
	calls := 0
	e = owner.Client("demo").ResolveMatching(bg, in, func(context.Context, Reply) error { calls++; return nil })
	if !errors.Is(e, ErrDenied) || calls != 0 {
		t.Fatal("unfenced conflict information escaped")
	}
}

type matchingRevoke struct {
	*testHost
	armed bool
}

func (h *matchingRevoke) Release(c context.Context, a Actor, id string, fn func() error) error {
	if h.armed {
		h.testHost.revoke()
	}
	return h.testHost.Release(c, a, id, fn)
}
