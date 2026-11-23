package observation

import (
	"context"
	"errors"
	"sync"

	"github.com/version2beta/perhap/sdk/go/viewjournal"
)

// Gate serializes the actual synchronous release with trusted local revocation.
type Gate struct {
	mu        sync.Mutex // the single fence held through Check and the recipient call
	authority Authority
	binding   Binding
	revoked   bool
}

// errArgument reports a nil or invalid configuration/argument supplied by the caller.
var errArgument = errors.New("observation: invalid argument or configuration")

// NewGate constructs a local release boundary over injected authority.
func NewGate(authority Authority, binding Binding) *Gate {
	if authority == nil {
		return nil
	}
	return &Gate{authority: authority, binding: binding}
}

// Revoke retires one exact authorization epoch; later release is forbidden.
func (g *Gate) Revoke() {
	if g == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.revoked = true
}

// Release holds the revocation fence through the recipient callback.
func (g *Gate) Release(ctx context.Context, binding Binding, frame Frame, recipient Recipient) error {
	if g == nil || recipient == nil {
		return ErrDenied
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.admitLocked(ctx, binding); err != nil {
		return err
	}
	if err := validFrame(binding, frame); err != nil {
		return err
	}
	// The recipient may have disclosed before returning an error or expiry.
	if err := recipient(ctx, frame); err != nil {
		return ErrUnknown
	}
	return nil
}

// admitLocked denies before any callback; the caller must hold g.mu.
func (g *Gate) admitLocked(ctx context.Context, binding Binding) error {
	if g.revoked || binding != g.binding {
		return ErrDenied
	}
	if binding.Descriptor != Profile {
		return ErrVersion
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := g.authority.Check(ctx, binding); err != nil {
		return mapAuthority(err)
	}
	return nil
}

// Manager owns quotas and dependencies; each returned Session owns its durable pin.
type Manager struct {
	mu       sync.Mutex // guards reserved only
	journal  Journal
	gate     *Gate
	limits   Limits
	reserved int
}

// NewManager creates bounded bootstrap/session orchestration, not a transport.
func NewManager(journal Journal, gate *Gate, limits Limits) (*Manager, error) {
	if journal == nil || gate == nil || limits.MaxSessions <= 0 || limits.Page <= 0 ||
		limits.CallbackTimeout <= 0 || limits.CleanupTimeout <= 0 {
		return nil, errArgument
	}
	return &Manager{journal: journal, gate: gate, limits: limits}, nil
}

// Open authorizes before preparing any snapshot or exposing its metadata.
func (m *Manager) Open(ctx context.Context, binding Binding, recipient Recipient) (*Session, error) {
	if m == nil || recipient == nil {
		return nil, errArgument
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	m.gate.mu.Lock()
	err := m.gate.admitLocked(ctx, binding)
	m.gate.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if err := m.reserve(); err != nil {
		return nil, err
	}
	plan, err := m.journal.Prepare(ctx, binding.View)
	if err != nil {
		m.unreserve()
		return nil, mapJournal(err)
	}
	sctx, cancel := context.WithCancel(ctx)
	return &Session{
		m:         m,
		binding:   binding,
		recipient: recipient,
		plan:      plan,
		ctx:       sctx,
		cancel:    cancel,
	}, nil
}

func (m *Manager) reserve() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reserved >= m.limits.MaxSessions {
		return ErrCapacity
	}
	m.reserved++
	return nil
}

func (m *Manager) unreserve() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reserved--
}

// Session owns one cancellation fence, serialized step and cleanup obligation.
type Session struct {
	m            *Manager
	binding      Binding
	recipient    Recipient
	plan         viewjournal.Plan
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex         // serializes Step and cleanup; Close waits on it
	last         viewjournal.Record // last successfully released snapshot or revision
	snapshotSent bool
	retired      bool
	released     bool // the durable pin is released and quota returned
}

// Step releases snapshot once, at most one page of successors, then a finite barrier.
func (s *Session) Step() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retired || s.ctx.Err() != nil {
		return ErrClosed
	}
	err := s.step()
	if err == nil {
		return nil
	}
	// No work resumes after a failed step: retire, cancel and attempt cleanup.
	s.retired = true
	s.cancel()
	if cerr := s.cleanup(); cerr != nil {
		return errors.Join(err, cerr)
	}
	return err
}

// Close cancels and joins in-flight synchronous work, then releases the retention pin.
// It is idempotent on success; failed cleanup remains owned/retryable.
func (s *Session) Close() error {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retired = true
	return s.cleanup()
}

// step must be called with s.mu held.
func (s *Session) step() error {
	if !s.snapshotSent {
		snap := s.plan.Snapshot
		if snap.View != s.binding.View || snap.Revision == 0 {
			return ErrSequence
		}
		if err := s.release(Frame{Kind: "snapshot", Record: copyRecord(snap)}); err != nil {
			return err
		}
		s.snapshotSent = true
		s.last = snap
	}
	b, err := s.m.journal.Read(s.ctx, s.plan.ID, s.last.Revision, s.m.limits.Page)
	if err != nil {
		return mapJournal(err)
	}
	recs, err := validPage(b, s.plan, s.last.Revision, s.m.limits.Page)
	if err != nil {
		return err
	}
	// The whole page is validated above; only then is any revision released.
	for _, r := range recs {
		if err := s.release(Frame{Kind: "revision", Record: copyRecord(r)}); err != nil {
			return err
		}
		s.last = r
	}
	cert, err := s.certify()
	if err != nil {
		return err
	}
	return s.release(Frame{Kind: "barrier", Certificate: cert})
}

// release delivers one frame through the Gate fence with a bounded callback context.
func (s *Session) release(f Frame) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.m.limits.CallbackTimeout)
	defer cancel()
	return s.m.gate.Release(ctx, s.binding, f, s.recipient)
}

// certify asks the authority for a seal over the exact last released revision.
func (s *Session) certify() (Certificate, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.m.limits.CallbackTimeout)
	defer cancel()
	evidence := s.last.Public.Evidence
	cert, err := s.m.gate.authority.Certify(ctx, s.binding, s.last.Revision, evidence)
	if err != nil {
		return Certificate{}, mapAuthority(err)
	}
	if cert.Binding != s.binding || cert.Through != s.last.Revision || cert.Evidence != evidence || cert.Seal == "" {
		return Certificate{}, ErrDenied
	}
	return cert, nil
}

// cleanup releases the durable pin and quota once; the caller must hold s.mu.
func (s *Session) cleanup() error {
	if s.released {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.m.limits.CleanupTimeout)
	defer cancel()
	if err := s.m.journal.Release(ctx, s.plan.ID); err != nil {
		return ErrCleanup
	}
	s.released = true
	s.m.unreserve()
	return nil
}

// validPage checks the entire batch before any of its revisions is released.
func validPage(b viewjournal.Batch, p viewjournal.Plan, after uint64, page int) ([]viewjournal.Record, error) {
	if b.PlanID != p.ID || b.View != p.View || b.After != after || b.Through < b.After || b.SelectedHead < b.Through {
		return nil, ErrSequence
	}
	n := b.Through - b.After
	if n > uint64(page) || uint64(len(b.Records)) != n {
		return nil, ErrSequence
	}
	prev := after
	for i, r := range b.Records {
		want := after + uint64(i) + 1
		if r.View != p.View || r.Revision != want || r.Previous != prev {
			return nil, ErrSequence
		}
		prev = r.Revision
	}
	return b.Records, nil
}

// validFrame refuses any frame that does not match the bound view or certificate.
func validFrame(b Binding, f Frame) error {
	switch f.Kind {
	case "snapshot", "revision":
		if f.Record.View != b.View || f.Record.Revision == 0 {
			return ErrDenied
		}
	case "barrier":
		if f.Certificate.Binding != b || f.Certificate.Through == 0 || f.Certificate.Seal == "" {
			return ErrDenied
		}
	default:
		return ErrDenied
	}
	return nil
}

// copyRecord isolates a recipient frame from the pinned journal value.
func copyRecord(r viewjournal.Record) viewjournal.Record {
	if r.Public.Value != nil {
		v := *r.Public.Value
		r.Public.Value = &v
	}
	return r
}

// mapJournal converts storage errors to public categories without private detail.
func mapJournal(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, viewjournal.ErrCapacity):
		return ErrCapacity
	case errors.Is(err, viewjournal.ErrDenied):
		return ErrDenied
	case errors.Is(err, viewjournal.ErrUnknown):
		return ErrUnknown
	default:
		return ErrUnavailable
	}
}

// mapAuthority keeps only the version category; other authority failures are denials.
func mapAuthority(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return err
	case errors.Is(err, ErrVersion):
		return ErrVersion
	default:
		return ErrDenied
	}
}
