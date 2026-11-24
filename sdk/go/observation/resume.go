package observation

import (
	"context"
	"errors"
	"reflect"
)

// ResumeCertified renews verified retained continuity and releases a purpose-bound
// resume handshake before successors; errors retire and observe owned pin cleanup.
// It follows Resume step for step, except that the session is activated only after
// the handshake for its actual accepted anchor has been released.
func (m *Recovery) ResumeCertified(ctx context.Context, b Binding, token string, r RecoveryRecipient) (*Recovered, error) {
	if r == nil {
		return nil, errArgument
	}
	return m.run(ctx, func(opCtx context.Context, cancel context.CancelFunc) (*Recovered, error) {
		if err := m.admit(opCtx, b); err != nil {
			return nil, m.retireWith(err)
		}
		if err := m.retire(); err != nil {
			return nil, err
		}
		claims, err := m.verify(opCtx, b, token)
		if errors.Is(err, ErrCannotResume) {
			if derr := m.deliver(opCtx, b, RecoveryFrame{Kind: "cannot-resume"}, r); derr != nil {
				return nil, derr
			}
			return nil, ErrCannotResume
		}
		if err != nil {
			return nil, err
		}
		session, err := newSessionFence()
		if err != nil {
			return nil, err
		}
		ret, err := m.acquire(opCtx, b, claims.Record.Revision)
		if err != nil {
			return nil, err
		}
		s := newRecovered(m, opCtx, cancel, b, r, ret.Plan, "", session, ret.Anchor)
		if !reflect.DeepEqual(ret.Anchor, claims.Record) {
			return nil, m.discard(s, ErrSequence)
		}
		if err := m.handshake(opCtx, b, s); err != nil {
			return nil, m.discard(s, err)
		}
		return m.activate(s)
	})
}

// handshake issues a resume proof over s's accepted anchor and releases it through
// the Gate fence. s is not yet activated, so no Step can run before this frame.
func (m *Recovery) handshake(ctx context.Context, b Binding, s *Recovered) error {
	token, err := m.issue(ctx, b, "resume", s.last)
	if err != nil {
		return err
	}
	return m.deliver(ctx, b, RecoveryFrame{Kind: "resume", Session: s.session, Record: copyRecord(s.last), Proof: token}, s.recipient)
}
