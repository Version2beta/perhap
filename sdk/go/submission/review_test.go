package submission

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubmissionReservedCompletionAndReopenBounds(t *testing.T) {
	h := &testHost{}
	l := limits
	l.MaxEntries = 3
	path := filepath.Join(t.TempDir(), "s.db")
	o := open(t, path, h, l)
	one, two := input("1"), input("2")
	submit(t, o, one)
	if e := o.Client("demo").Submit(bg, two, func(context.Context, Reply) error { return nil }); !errors.Is(e, ErrCapacity) {
		t.Fatal("new receipt consumed completion budget")
	}
	if resolve(t, o, two.Event.ID).Kind != "no-retained-knowledge" {
		t.Fatal("refused receipt left admitted work without budget")
	}
	if e := o.Evaluate(bg, one.Event.ID, "reserved"); e != nil {
		t.Fatalf("accepted work lost reserved completion: %v", e)
	}
	o.Close()
	path = filepath.Join(t.TempDir(), "reopen.db")
	o = open(t, path, h, limits)
	submit(t, o, one)
	submit(t, o, two)
	o.Close()
	small := limits
	small.MaxEvents = 1
	if bad, e := Open(bg, path, h, small); e == nil || bad != nil {
		t.Fatal("recovery silently exceeded configured resource bounds")
	}
}
func TestSubmissionEnvelopeLimitsAndReferences(t *testing.T) {
	h := &testHost{}
	l := limits
	l.MaxEventBytes = 1
	o := open(t, filepath.Join(t.TempDir(), "small.db"), h, l)
	in := input("1")
	if e := o.Client("demo").Submit(bg, in, func(context.Context, Reply) error { return nil }); !errors.Is(e, ErrCapacity) {
		t.Fatal("event limit counted only raw data, not normalized envelope")
	}
	if resolve(t, o, in.Event.ID).Kind != "no-retained-knowledge" {
		t.Fatal("oversized envelope persisted receipt")
	}
	o.Close()
	o = open(t, filepath.Join(t.TempDir(), "normal.db"), h, limits)
	in = input("1")
	in.Event.CorrelationID = ""
	cause := "foreign-reference"
	in.Event.CausationID = &cause
	if f := submit(t, o, in); f.Admission != "admitted" {
		t.Fatal("optional correlation/codec-owned reference interpretation changed")
	}
}

type releaseFault struct {
	*testHost
	mode   string
	cancel context.CancelFunc
}

func (h *releaseFault) Release(c context.Context, a Actor, id string, fn func() error) error {
	switch h.mode {
	case "late":
		h.cancel()
		return fn()
	case "swallow":
		_ = fn()
		return nil
	case "twice":
		_ = fn()
		_ = fn()
		return nil
	}
	return h.testHost.Release(c, a, id, fn)
}
func TestSubmissionReplyAccountingAndOpaqueErrors(t *testing.T) {
	for _, mode := range []string{"late", "swallow", "twice"} {
		t.Run(mode, func(t *testing.T) {
			caller, cancel := context.WithCancel(bg)
			defer cancel()
			h := &releaseFault{testHost: &testHost{}, mode: mode, cancel: cancel}
			l := limits
			o, e := Open(bg, filepath.Join(t.TempDir(), "s.db"), h, l)
			if e != nil {
				t.Fatal(e)
			}
			defer o.Close()
			calls := 0
			e = o.Client("demo").Submit(caller, input("1"), func(context.Context, Reply) error {
				calls++
				if mode == "swallow" {
					return errors.New("recipient may have disclosed")
				}
				return nil
			})
			if mode == "late" {
				if calls != 0 || !errors.Is(e, context.Canceled) {
					t.Fatal("expired fence invoked recipient")
				}
			} else if calls != 1 || !errors.Is(e, ErrRelease) {
				t.Fatal("host response hid callback uncertainty or repeated disclosure")
			}
		})
	}
	h := &contextFault{testHost: &testHost{}}
	o, e := Open(bg, filepath.Join(t.TempDir(), "opaque.db"), h, limits)
	if e != nil {
		t.Fatal(e)
	}
	defer o.Close()
	e = o.Client("demo").Resolve(bg, input("1").Event.ID, func(context.Context, Reply) error { t.Fatal("unauthorized reply"); return nil })
	if e == nil || e.Error() != context.Canceled.Error() {
		t.Fatal("private wrapped context diagnostics escaped")
	}
}

type contextFault struct{ *testHost }

func (h *contextFault) Authorize(context.Context, string, string, string, string) (Actor, error) {
	return Actor{}, fmt.Errorf("protected-internals: %w", context.Canceled)
}
func TestSubmissionOpaqueEvaluationEvidence(t *testing.T) {
	h := &testHost{}
	o := open(t, filepath.Join(t.TempDir(), "s.db"), h, limits)
	in := input("1")
	submit(t, o, in)
	if e := o.Evaluate(bg, in.Event.ID, "first"); e != nil {
		t.Fatal(e)
	}
	first := resolve(t, o, in.Event.ID).Facts.Evidence
	if strings.HasPrefix(first, "sha256:") || first == "" {
		t.Fatal("private history digest used as public redaction/evidence reference")
	}
	if e := o.Evaluate(bg, in.Event.ID, "second"); e != nil {
		t.Fatal(e)
	}
	second := resolve(t, o, in.Event.ID).Facts.Evidence
	if second == first {
		t.Fatal("new identified attempt lacked separate opaque evidence reference")
	}
}
