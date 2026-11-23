package observation

import (
	"context"
	"errors"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"strings"
	"testing"
	"time"
)

func TestRetainedBootstrapAnchor(t *testing.T) {
	j, h := journal(t, 10)
	for n := 1; n <= 3; n++ {
		accept(t, j, h, n)
	}
	id := "obs/slot/bootstrap/0"
	p, e := j.Acquire(ctx, id, v, 0)
	if e != nil || p.Anchor.Revision != 3 || p.Anchor.View != v {
		t.Fatalf("bootstrap anchor is not pinned snapshot: %+v %v", p, e)
	}
	accept(t, j, h, 4)
	again, e := j.Acquire(ctx, id, v, 0)
	if e != nil || again.Anchor.Revision != 3 || again.Plan.Snapshot.Revision != 3 {
		t.Fatal("bootstrap retry moved its cut")
	}
	if _, e = j.ReadRetained(ctx, id, 2, 2); !errors.Is(e, viewjournal.ErrStale) {
		t.Fatal("bootstrap pin exposed pre-anchor history")
	}
	if e = j.Release(ctx, id); e != nil {
		t.Fatal(e)
	}
}
func TestRetainedOwnerEncoding(t *testing.T) {
	j, h := journal(t, 10)
	accept(t, j, h, 1)
	for _, owner := range []string{"\xff", "é"} {
		if _, e := j.Acquire(ctx, "obs/"+owner+"/nonce/1", v, 1); e == nil {
			t.Fatal("non-ASCII owned ID committed lossy/unsupported encoding")
		}
		m, e := NewRecovery(ctx, j, NewGate(&authority{}, binding), proofAuthority(t), owner, recoveryLimits)
		if e == nil || m != nil {
			t.Fatal("non-ASCII owner accepted")
		}
	}
	noPins(t, j)
}

func TestRecoveryEnvelopeSessionFence(t *testing.T) {
	j, h := journal(t, 10)
	accept(t, j, h, 1)
	m := recoveryManager(t, j, NewGate(&authority{}, binding), proofAuthority(t), "slot")
	var fs []RecoveryFrame
	s, e := m.Open(ctx, binding, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil })
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(fs) != 2 || fs[0].Session == "" || fs[0].Session != fs[1].Session || strings.Contains(fs[0].Session, "/") {
		t.Fatalf("missing/private session fence on %d frames", len(fs))
	}
	first := fs[0].Session
	token := fs[1].Proof
	fs = nil
	next, e := m.Resume(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil })
	if e != nil {
		t.Fatal(e)
	}
	if e = next.Step(); e != nil {
		t.Fatal(e)
	}
	if len(fs) != 1 || fs[0].Session == "" || fs[0].Session == first {
		t.Fatal("reconnect reused old envelope fence")
	}
	if e = s.Step(); !errors.Is(e, ErrClosed) {
		t.Fatal("old session revived")
	}
	next.Close()
	noPins(t, j)
}

type deadlineProofs struct {
	ProofAuthority
	issueWait, verifyWait, seen bool
}

func (p *deadlineProofs) Issue(c context.Context, claim Claims) (string, error) {
	if p.issueWait {
		_, p.seen = c.Deadline()
		if !p.seen {
			return "", ErrUnavailable
		}
		<-c.Done()
		return "", c.Err()
	}
	return p.ProofAuthority.Issue(c, claim)
}
func (p *deadlineProofs) Verify(c context.Context, tok string) (Claims, error) {
	if p.verifyWait {
		_, p.seen = c.Deadline()
		if !p.seen {
			return Claims{}, ErrUnavailable
		}
		<-c.Done()
		return Claims{}, c.Err()
	}
	return p.ProofAuthority.Verify(c, tok)
}

type deadlineAuthority struct {
	authority
	wait, seen bool
}

func (a *deadlineAuthority) Check(c context.Context, b Binding) error {
	if a.wait {
		_, a.seen = c.Deadline()
		if !a.seen {
			return ErrUnavailable
		}
		<-c.Done()
		return nil
	}
	return a.authority.Check(c, b)
}
func TestRecoveryAuthorityDeadlines(t *testing.T) {
	for _, mode := range []string{"issue", "verify", "admission", "late-admission"} {
		t.Run(mode, func(t *testing.T) {
			j, h := journal(t, 10)
			accept(t, j, h, 1)
			root := proofAuthority(t)
			p := &deadlineProofs{ProofAuthority: root}
			a := &deadlineAuthority{}
			l := recoveryLimits
			l.CallbackTimeout = 50 * time.Millisecond
			m, e := NewRecovery(ctx, j, NewGate(a, binding), p, "slot", l)
			if e != nil {
				t.Fatal(e)
			}
			defer m.Close()
			calls := 0
			r := func(context.Context, RecoveryFrame) error { calls++; return nil }
			if mode == "admission" {
				a.wait = true
				s, e := m.Open(ctx, binding, r)
				if s != nil || e == nil || !a.seen || calls != 0 {
					t.Fatalf("unbounded/late successful admission: %v deadline=%v", e, a.seen)
				}
				noPins(t, j)
				return
			}
			if mode == "verify" {
				_, token := snapshotToken(t, m)
				p.verifyWait = true
				s, e := m.Resume(ctx, binding, token, r)
				if s != nil || e == nil || !p.seen || calls != 0 {
					t.Fatalf("unbounded verification: %v deadline=%v", e, p.seen)
				}
				noPins(t, j)
				return
			}
			if mode == "issue" {
				p.issueWait = true
			} else {
				root.hook = func(c Claims) { a.wait = true }
			}
			s, e := m.Open(ctx, binding, r)
			if e != nil {
				t.Fatal(e)
			}
			e = s.Step()
			seen := p.seen
			if mode == "late-admission" {
				seen = a.seen
			}
			if e == nil || !seen || calls != 0 {
				t.Fatalf("late proof/policy permitted release: %v deadline=%v callbacks=%d", e, seen, calls)
			}
			noPins(t, j)
		})
	}
}
