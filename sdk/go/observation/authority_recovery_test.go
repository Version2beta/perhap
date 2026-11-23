package observation

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"path/filepath"
	"testing"
)

type distinctHost struct {
	registered, stateAllowed bool
	public                   viewjournal.Public
}

func (h *distinctHost) Authorize(context.Context, viewjournal.Identity) error {
	if !h.registered {
		return viewjournal.ErrDenied
	}
	return nil
}
func (h *distinctHost) Recover(context.Context, []viewjournal.Head) error { return nil }
func (h *distinctHost) Within(_ context.Context, _ viewjournal.Request, fn func(viewjournal.Verified) error) error {
	if !h.registered || !h.stateAllowed {
		return viewjournal.ErrDenied
	}
	return fn(viewjournal.Verified{Public: h.public, PrivateCut: "protected-source-identifiers"})
}

type disclosurePolicy struct{ allow bool }

func (p *disclosurePolicy) Check(_ context.Context, b Binding) error {
	if !p.allow || b != binding {
		return ErrDenied
	}
	return nil
}
func (p *disclosurePolicy) Certify(_ context.Context, b Binding, n uint64, e string) (Certificate, error) {
	return Certificate{Binding: b, Through: n, Evidence: e, Seal: "trusted-test-only"}, nil
}
func TestRecoveryAuthoritySeparation(t *testing.T) {
	h := &distinctHost{public: viewjournal.Public{Available: false, Evaluation: "failed", Completeness: "unavailable", Evidence: "public-outcome-only"}}
	j, e := viewjournal.Open(ctx, filepath.Join(t.TempDir(), "v.db"), h, viewjournal.Limits{MaxEvaluations: 10, MaxPins: 2, MaxBatch: 2, MaxBytes: 1 << 20})
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	req := viewjournal.Request{ID: "first", EvaluationID: "g1", View: v}
	if _, e = j.Accept(ctx, req); !errors.Is(e, viewjournal.ErrDenied) {
		t.Fatal("unregistered probe created state")
	}
	heads, e := j.Heads(ctx)
	if e != nil || len(heads) != 0 {
		t.Fatal("forbidden probe registered view")
	}
	h.registered = true
	if _, e = j.Accept(ctx, req); !errors.Is(e, viewjournal.ErrDenied) {
		t.Fatal("registration substituted for state authority")
	}
	h.stateAllowed = true
	if _, e = j.Accept(ctx, req); e != nil {
		t.Fatal(e)
	}
	policy := &disclosurePolicy{}
	m := recoveryManager(t, j, NewGate(policy, binding), proofAuthority(t), "slot")
	calls := 0
	for _, b := range []Binding{binding, {Observer: "alice", View: viewjournal.Identity{Projection: "nonexistent"}, Descriptor: Profile, Epoch: "grant1"}} {
		s, e := m.Open(ctx, b, func(context.Context, RecoveryFrame) error { calls++; return nil })
		if !errors.Is(e, ErrDenied) || s != nil || calls != 0 {
			t.Fatal("state/registration substituted for disclosure or existence leaked")
		}
	}
	policy.allow = true
	var fs []RecoveryFrame
	s, e := m.Open(ctx, binding, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil })
	if e != nil || s == nil {
		t.Fatal(e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(fs) != 2 || fs[0].Record.Public.Available || fs[0].Record.Public.Value != nil || fs[0].Record.Public.Evaluation != "failed" {
		t.Fatal("authorized unavailable state lost")
	}
	h.public = viewjournal.Public{Available: true, Value: nil, Evaluation: "valid", Completeness: "complete_as_of_frontier", Evidence: "public-outcome-only"}
	req.ID = "second"
	req.EvaluationID = "g2"
	req.ExpectedCheckpoint = 1
	req.ExpectedRevision = 1
	if _, e = j.Accept(ctx, req); e != nil {
		t.Fatal(e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(fs) != 4 || fs[2].Kind != "revision" || !fs[2].Record.Public.Available || fs[2].Record.Public.Value != nil {
		t.Fatal("available null conflated with unavailable")
	}
	policy.allow = false
	if e = s.Step(); !errors.Is(e, ErrDenied) {
		t.Fatal("stale proof granted permission")
	}
	if e = s.Close(); e != nil {
		t.Fatal("observer revocation blocked private cleanup")
	}
	noPins(t, j)
}
