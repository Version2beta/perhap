package observation

import (
	"context"
	"errors"
	"testing"
)

type faultProofs struct {
	ProofAuthority
	mutate, unavailable bool
}

func (p *faultProofs) Issue(c context.Context, claims Claims) (string, error) {
	if p.mutate && claims.Record.Public.Value != nil {
		claims.Record.Public.Value.Total++
	}
	return p.ProofAuthority.Issue(c, claims)
}
func (p *faultProofs) Verify(c context.Context, token string) (Claims, error) {
	if p.unavailable {
		return Claims{}, errors.New("private authority diagnostic")
	}
	return p.ProofAuthority.Verify(c, token)
}
func TestRecoveryProofFailures(t *testing.T) {
	t.Run("signed-mutated-record", func(t *testing.T) {
		j, h := journal(t, 10)
		accept(t, j, h, 1)
		p := &faultProofs{ProofAuthority: proofAuthority(t), mutate: true}
		m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
		calls := 0
		s, e := m.Open(ctx, binding, func(context.Context, RecoveryFrame) error { calls++; return nil })
		if e != nil || s == nil {
			t.Fatal(e)
		}
		if e = s.Step(); !errors.Is(e, ErrSequence) || calls != 0 {
			t.Fatalf("signed response laundered changed record: %v callbacks=%d", e, calls)
		}
		noPins(t, j)
	})
	t.Run("verification-unavailable-not-invalid", func(t *testing.T) {
		j, h := journal(t, 10)
		accept(t, j, h, 1)
		p := &faultProofs{ProofAuthority: proofAuthority(t)}
		m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
		_, token := snapshotToken(t, m)
		p.unavailable = true
		calls := 0
		s, e := m.Resume(ctx, binding, token, func(context.Context, RecoveryFrame) error { calls++; return nil })
		if s != nil || e != ErrUnavailable || calls != 0 {
			t.Fatalf("transient verifier failure invalidated/revealed cursor: %v callbacks=%d", e, calls)
		}
		noPins(t, j)
		p.unavailable = false
		s, e = m.Resume(ctx, binding, token, func(context.Context, RecoveryFrame) error { return nil })
		if e != nil || s == nil {
			t.Fatalf("original cursor no longer usable: %v", e)
		}
		if e = s.Step(); e != nil {
			t.Fatal(e)
		}
		s.Close()
		noPins(t, j)
	})
}
