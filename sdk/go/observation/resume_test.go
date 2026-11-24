package observation

import (
	"context"
	"errors"
	"testing"
)

func TestCertifiedResumeHandshake(t *testing.T) {
	j, h := journal(t, 100)
	for n := 1; n <= 43; n++ {
		accept(t, j, h, n)
	}
	p := proofAuthority(t)
	g := NewGate(&authority{}, binding)
	m := recoveryManager(t, j, g, p, "slot")
	_, token := snapshotToken(t, m)
	accept(t, j, h, 44)
	accept(t, j, h, 45)
	var fs []RecoveryFrame
	s, e := m.ResumeCertified(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil })
	if e != nil || s == nil {
		t.Fatalf("resume %v", e)
	}
	if len(fs) != 1 || fs[0].Kind != "resume" || fs[0].Record.Revision != 43 || fs[0].Session == "" {
		t.Fatal("no ordered accepted anchor handshake")
	}
	claims, e := p.Verify(ctx, fs[0].Proof)
	if e != nil || claims.Purpose != "resume" || claims.Binding != binding || claims.Record.Revision != 43 {
		t.Fatal("unverified/misbound anchor")
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(fs) != 4 || fs[1].Record.Revision != 44 || fs[2].Record.Revision != 45 || fs[3].Kind != "barrier" || fs[3].Record.Revision != 45 {
		t.Fatal("resume chain changed")
	}
	s.Close()
	noPins(t, j)
}
func TestCertifiedResumeFencesAndFailure(t *testing.T) {
	for _, mode := range []string{"revoke", "callback", "invalid"} {
		t.Run(mode, func(t *testing.T) {
			j, h := journal(t, 10)
			accept(t, j, h, 1)
			p := proofAuthority(t)
			g := NewGate(&authority{}, binding)
			m := recoveryManager(t, j, g, p, "slot")
			_, token := snapshotToken(t, m)
			if mode == "invalid" {
				token += "bad"
			}
			if mode == "revoke" {
				p.hook = func(c Claims) {
					if c.Purpose == "resume" {
						g.Revoke()
					}
				}
			}
			calls := 0
			s, e := m.ResumeCertified(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error {
				calls++
				if mode == "invalid" {
					if f.Kind != "cannot-resume" || f.Proof != "" {
						t.Error("invalid cursor certified")
					}
					return nil
				}
				return errors.New("possible exposure")
			})
			if s != nil || e == nil {
				t.Fatal("failed resume returned session")
			}
			if mode == "revoke" && (!errors.Is(e, ErrDenied) || calls != 0) {
				t.Fatal("revoked handshake escaped")
			}
			if mode == "callback" && !errors.Is(e, ErrUnknown) {
				t.Fatal("uncertain callback misclassified")
			}
			m.Close()
			noPins(t, j)
		})
	}
}
