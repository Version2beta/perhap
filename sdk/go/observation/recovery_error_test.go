package observation

import (
	"context"
	"fmt"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"testing"
)

type contextProofs struct {
	ProofAuthority
	issueError, verifyError bool
}

func (p *contextProofs) Issue(c context.Context, claims Claims) (string, error) {
	if p.issueError {
		return "", fmt.Errorf("protected-internals: %w", context.Canceled)
	}
	return p.ProofAuthority.Issue(c, claims)
}
func (p *contextProofs) Verify(c context.Context, token string) (Claims, error) {
	if p.verifyError {
		return Claims{}, fmt.Errorf("protected-internals: %w", context.Canceled)
	}
	return p.ProofAuthority.Verify(c, token)
}

type contextAuthority struct{ authority }

func (a *contextAuthority) Check(context.Context, Binding) error {
	return fmt.Errorf("protected-internals: %w", context.Canceled)
}

type contextSource struct{ RecoverySource }

func (s *contextSource) ReadRetained(context.Context, string, uint64, int) (viewjournal.Batch, error) {
	return viewjournal.Batch{}, fmt.Errorf("protected-internals: %w", context.Canceled)
}
func TestRecoveryErrorsAreOpaque(t *testing.T) {
	for _, mode := range []string{"issue", "verify", "admission", "read"} {
		t.Run(mode, func(t *testing.T) {
			j, h := journal(t, 10)
			accept(t, j, h, 1)
			p := &contextProofs{ProofAuthority: proofAuthority(t)}
			g := NewGate(&authority{}, binding)
			if mode == "admission" {
				g = NewGate(&contextAuthority{}, binding)
			}
			var source RecoverySource = j
			if mode == "read" {
				source = &contextSource{RecoverySource: j}
			}
			m := recoveryManager(t, source, g, p, "slot")
			calls := 0
			r := func(context.Context, RecoveryFrame) error { calls++; return nil }
			var e error
			if mode == "verify" {
				_, tok := snapshotToken(t, m)
				p.verifyError = true
				_, e = m.Resume(ctx, binding, tok, r)
			} else {
				p.issueError = mode == "issue"
				var s *Recovered
				s, e = m.Open(ctx, binding, r)
				if e == nil {
					e = s.Step()
				}
			}
			if e == nil || e.Error() != context.Canceled.Error() {
				t.Fatalf("protected diagnostics were not normalized in %s", mode)
			}
			want := 0
			if mode == "read" {
				want = 1
			}
			if calls != want {
				t.Fatal("failed operation released unexpected data")
			}
			m.Close()
			noPins(t, j)
		})
	}
}
