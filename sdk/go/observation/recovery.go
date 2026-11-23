package observation

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/version2beta/perhap/sdk/go/viewjournal"
)

// maxOwnerBytes keeps obs/owner/nonce/after within viewjournal's retained ID bound.
const maxOwnerBytes = 64

// Recovery owns one reconnect slot: at most one Recovered session and its pin.
// Its owner namespace is trusted service identity. Only pins under obs/owner/ are
// reconciled or released; the host must exclude any previous live owner.
type Recovery struct {
	source RecoverySource
	gate   *Gate
	proofs ProofAuthority
	owner  string
	limits RecoveryLimits

	slot     sync.Mutex         // serializes Open, Resume, Replace and Close's retirement
	mu       sync.Mutex         // guards the fields below
	active   *Recovered         // the slot's session until its pin is released
	inflight context.CancelFunc // cancels the operation that holds slot
	closed   bool
	// uncertain means an Acquire outcome is unknown. Only slot holders read it.
	// Reopen and NewRecovery reconcile the pin; this instance never guesses.
	uncertain bool
}

// Recovered is one observer session. Its pin is released once, by Close or by the
// first failed Step, and a session with unreleased pin is retried by Close.
type Recovered struct {
	m         *Recovery
	binding   Binding
	recipient RecoveryRecipient
	plan      viewjournal.Plan
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex // serializes Step and cleanup; Close waits on it
	kind      string     // "snapshot" or "replacement" frame still owed by the first Step
	last      viewjournal.Record
	retired   bool
	released  bool
}

// NewRecovery reconciles only its exclusively owned namespace's stale plans.
func NewRecovery(ctx context.Context, source RecoverySource, gate *Gate, proofs ProofAuthority, owner string, l RecoveryLimits) (*Recovery, error) {
	if source == nil || gate == nil || proofs == nil || !validOwner(owner) || !validRecoveryLimits(l) {
		return nil, errArgument
	}
	m := &Recovery{source: source, gate: gate, proofs: proofs, owner: owner, limits: l}
	if err := m.reconcile(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

// Open starts at the latest pinned snapshot; the first Step releases it.
func (m *Recovery) Open(ctx context.Context, b Binding, r RecoveryRecipient) (*Recovered, error) {
	if r == nil {
		return nil, errArgument
	}
	return m.run(ctx, func(opCtx context.Context, cancel context.CancelFunc) (*Recovered, error) {
		return m.snapshotSession(opCtx, cancel, b, r, "snapshot")
	})
}

// Replace starts a purpose-distinct certified replacement at the latest snapshot.
// It never consumes or resumes an earlier token.
func (m *Recovery) Replace(ctx context.Context, b Binding, r RecoveryRecipient) (*Recovered, error) {
	if r == nil {
		return nil, errArgument
	}
	return m.run(ctx, func(opCtx context.Context, cancel context.CancelFunc) (*Recovered, error) {
		return m.snapshotSession(opCtx, cancel, b, r, "replacement")
	})
}

// Resume verifies a barrier token and emits only successors of its anchor. Any token
// that is not a barrier for this binding gets one cannot-resume frame.
func (m *Recovery) Resume(ctx context.Context, b Binding, token string, r RecoveryRecipient) (*Recovered, error) {
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
		ret, err := m.acquire(opCtx, b, claims.Record.Revision)
		if err != nil {
			return nil, err
		}
		s := newRecovered(m, opCtx, cancel, b, r, ret.Plan, "", ret.Anchor)
		if !reflect.DeepEqual(ret.Anchor, claims.Record) {
			return nil, m.discard(s, ErrSequence)
		}
		return m.activate(s)
	})
}

// Close cancels in-flight work and retires the slot's session. It reports ErrCleanup
// while the session's pin or an unknown Acquire outcome remains unresolved.
func (m *Recovery) Close() error {
	m.mu.Lock()
	m.closed = true
	if m.inflight != nil {
		m.inflight()
	}
	m.mu.Unlock()
	m.slot.Lock()
	defer m.slot.Unlock()
	if err := m.retire(); err != nil {
		return err
	}
	if m.uncertain {
		return ErrCleanup
	}
	return nil
}

// run serializes one owner operation. The prior session is canceled before the Gate
// is requested, so a recipient blocked inside the fence cannot stall admission.
func (m *Recovery) run(ctx context.Context, op func(context.Context, context.CancelFunc) (*Recovered, error)) (*Recovered, error) {
	opCtx, cancel := context.WithCancel(ctx)
	m.slot.Lock()
	defer m.slot.Unlock()
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		cancel()
		return nil, ErrClosed
	}
	if m.uncertain {
		m.mu.Unlock()
		cancel()
		return nil, ErrCleanup
	}
	m.inflight = cancel
	prev := m.active
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.inflight = nil
		m.mu.Unlock()
	}()
	if prev != nil {
		prev.cancel()
	}
	s, err := op(opCtx, cancel)
	if err != nil {
		cancel()
	}
	return s, err
}

// snapshotSession starts at the latest pinned snapshot, which the first Step releases as kind.
func (m *Recovery) snapshotSession(ctx context.Context, cancel context.CancelFunc, b Binding, r RecoveryRecipient, kind string) (*Recovered, error) {
	if err := m.admit(ctx, b); err != nil {
		return nil, m.retireWith(err)
	}
	if err := m.retire(); err != nil {
		return nil, err
	}
	ret, err := m.acquire(ctx, b, 0)
	if err != nil {
		return nil, err
	}
	return m.activate(newRecovered(m, ctx, cancel, b, r, ret.Plan, kind, viewjournal.Record{}))
}

// admit checks current authority under the Gate fence before any state is exposed.
func (m *Recovery) admit(ctx context.Context, b Binding) error {
	m.gate.mu.Lock()
	defer m.gate.mu.Unlock()
	return m.gate.admitLocked(ctx, b)
}

// retire joins the slot's prior session. If its pin cannot be released, the slot keeps
// ownership so a later Close or operation can retry.
func (m *Recovery) retire() error {
	m.mu.Lock()
	s := m.active
	m.mu.Unlock()
	if s == nil {
		return nil
	}
	if err := s.Close(); err != nil {
		return err
	}
	m.mu.Lock()
	m.active = nil
	m.mu.Unlock()
	return nil
}

func (m *Recovery) retireWith(primary error) error {
	if err := m.retire(); err != nil {
		return errors.Join(primary, err)
	}
	return primary
}

// activate installs s as the slot's session. A session created after Close is discarded.
func (m *Recovery) activate(s *Recovered) (*Recovered, error) {
	m.mu.Lock()
	closed := m.closed
	if !closed {
		m.active = s
	}
	m.mu.Unlock()
	if closed {
		return nil, m.discard(s, ErrClosed)
	}
	return s, nil
}

// discard retires a session the caller never received. If its pin cannot be released,
// the slot keeps ownership so Close can retry.
func (m *Recovery) discard(s *Recovered, primary error) error {
	if err := s.Close(); err != nil {
		m.mu.Lock()
		m.active = s
		m.mu.Unlock()
		return errors.Join(primary, err)
	}
	return primary
}

// acquire pins history at after. An unknown commit makes the slot uncertain: the pin
// may exist, and only reopen/reconcile in this owner namespace resolves it.
func (m *Recovery) acquire(ctx context.Context, b Binding, after uint64) (viewjournal.Retained, error) {
	id, err := m.newPinID(after)
	if err != nil {
		return viewjournal.Retained{}, err
	}
	ret, err := m.source.Acquire(ctx, id, b.View, after)
	switch {
	case errors.Is(err, viewjournal.ErrUnknown):
		m.uncertain = true
		return viewjournal.Retained{}, errors.Join(ErrUnknown, ErrCleanup)
	case err != nil:
		return viewjournal.Retained{}, mapJournal(err)
	}
	return ret, nil
}

// newPinID makes a fresh private ID. Released IDs are never reused by viewjournal.
func (m *Recovery) newPinID(after uint64) (string, error) {
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return "", ErrUnavailable
	}
	return fmt.Sprintf("obs/%s/%x/%d", m.owner, nonce[:], after), nil
}

// verify caps token bytes before parsing. It returns claims only for a barrier of b.
// Verifier failure is unavailability, not evidence that the cursor is invalid.
func (m *Recovery) verify(ctx context.Context, b Binding, token string) (Claims, error) {
	if token == "" || len(token) > m.limits.MaxTokenBytes {
		return Claims{}, ErrCannotResume
	}
	c, err := m.proofs.Verify(ctx, token)
	if errors.Is(err, ErrCannotResume) {
		return Claims{}, ErrCannotResume
	}
	if err != nil {
		return Claims{}, mapProof(err)
	}
	if c.Binding != b || c.Purpose != "barrier" || c.Record.View != b.View || c.Record.Revision == 0 {
		return Claims{}, ErrCannotResume
	}
	return c, nil
}

// issue returns a proof only after verifying that it certifies the exact expected claims.
// Issue receives its own copy, so a mutated copy cannot change the expected record.
func (m *Recovery) issue(ctx context.Context, b Binding, purpose string, rec viewjournal.Record) (string, error) {
	token, err := m.proofs.Issue(ctx, claimsFor(b, purpose, rec))
	if err != nil {
		return "", mapProof(err)
	}
	if token == "" {
		return "", ErrUnavailable
	}
	if len(token) > m.limits.MaxTokenBytes {
		return "", ErrCapacity
	}
	got, err := m.proofs.Verify(ctx, token)
	if err != nil {
		return "", mapProof(err)
	}
	want := claimsFor(b, purpose, rec)
	if got.Binding != want.Binding || got.Purpose != want.Purpose || !reflect.DeepEqual(got.Record, want.Record) {
		return "", ErrSequence
	}
	return token, nil
}

// deliver releases one frame under the Gate fence: the same mutex and admission as
// Gate.Release, with the recipient invoked before the fence is dropped.
func (m *Recovery) deliver(ctx context.Context, b Binding, f RecoveryFrame, r RecoveryRecipient) error {
	if err := validRecoveryFrame(b, f); err != nil {
		return err
	}
	n, err := frameSize(f)
	if err != nil {
		return ErrUnavailable
	}
	if n > m.limits.MaxBytes {
		return ErrCapacity
	}
	cctx, cancel := context.WithTimeout(ctx, m.limits.CallbackTimeout)
	defer cancel()
	g := m.gate
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.admitLocked(cctx, b); err != nil {
		return err
	}
	// The recipient may have disclosed before returning an error or expiry.
	if err := r(cctx, f); err != nil || cctx.Err() != nil {
		return ErrUnknown
	}
	return nil
}

// pageFits bounds a buffered page before any of its revisions is released.
func (m *Recovery) pageFits(recs []viewjournal.Record) error {
	total := 0
	for _, r := range recs {
		n, err := frameSize(RecoveryFrame{Kind: "revision", Record: r})
		if err != nil {
			return ErrUnavailable
		}
		total += n
		if total > m.limits.MaxBytes {
			return ErrCapacity
		}
	}
	return nil
}

// reconcile releases this owner's abandoned plans and never touches other pins.
func (m *Recovery) reconcile(ctx context.Context) error {
	plans, err := m.source.Plans(ctx)
	if err != nil {
		return mapJournal(err)
	}
	prefix := "obs/" + m.owner + "/"
	for _, p := range plans {
		if !strings.HasPrefix(p.ID, prefix) {
			continue
		}
		if err := m.releasePin(p.ID); err != nil {
			return err
		}
	}
	return nil
}

// releasePin uses service cleanup authority, independent of observer revocation.
func (m *Recovery) releasePin(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.limits.CleanupTimeout)
	defer cancel()
	if err := m.source.Release(ctx, id); err != nil {
		return ErrCleanup
	}
	return nil
}

func newRecovered(m *Recovery, ctx context.Context, cancel context.CancelFunc, b Binding, r RecoveryRecipient, p viewjournal.Plan, kind string, last viewjournal.Record) *Recovered {
	return &Recovered{m: m, binding: b, recipient: r, plan: p, ctx: ctx, cancel: cancel, kind: kind, last: last}
}

// Step releases the owed snapshot once, then at most one validated page of successors,
// then a barrier at the last released revision. Any failure retires and cleans up.
func (s *Recovered) Step() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.retired {
		return ErrClosed
	}
	err := s.step()
	if err == nil {
		return nil
	}
	s.retired = true
	s.cancel()
	if cerr := s.cleanup(); cerr != nil {
		return errors.Join(err, cerr)
	}
	return err
}

// Close cancels in-flight synchronous work, then releases the pin. It is idempotent
// on success; failed cleanup stays owned and retryable.
func (s *Recovered) Close() error {
	s.cancel()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.retired = true
	return s.cleanup()
}

// step must be called with s.mu held.
func (s *Recovered) step() error {
	if s.kind != "" {
		snap := s.plan.Snapshot
		if snap.View != s.binding.View || snap.Revision == 0 {
			return ErrSequence
		}
		token, err := s.m.issue(s.ctx, s.binding, s.kind, snap)
		if err != nil {
			return err
		}
		if err := s.m.deliver(s.ctx, s.binding, RecoveryFrame{Kind: s.kind, Record: copyRecord(snap), Proof: token}, s.recipient); err != nil {
			return err
		}
		s.kind = ""
		s.last = snap
	}
	b, err := s.m.source.ReadRetained(s.ctx, s.plan.ID, s.last.Revision, s.m.limits.Page)
	if err != nil {
		return mapJournal(err)
	}
	recs, err := validPage(b, s.plan, s.last.Revision, s.m.limits.Page)
	if err != nil {
		return err
	}
	if err := s.m.pageFits(recs); err != nil {
		return err
	}
	for _, r := range recs {
		if err := s.m.deliver(s.ctx, s.binding, RecoveryFrame{Kind: "revision", Record: copyRecord(r)}, s.recipient); err != nil {
			return err
		}
		s.last = r
	}
	token, err := s.m.issue(s.ctx, s.binding, "barrier", s.last)
	if err != nil {
		return err
	}
	return s.m.deliver(s.ctx, s.binding, RecoveryFrame{Kind: "barrier", Record: copyRecord(s.last), Proof: token}, s.recipient)
}

// cleanup releases the pin once. The caller must hold s.mu.
func (s *Recovered) cleanup() error {
	if s.released {
		return nil
	}
	if err := s.m.releasePin(s.plan.ID); err != nil {
		return err
	}
	s.released = true
	return nil
}

func claimsFor(b Binding, purpose string, rec viewjournal.Record) Claims {
	return Claims{Binding: b, Purpose: purpose, Record: copyRecord(rec)}
}

// validRecoveryFrame keeps record-bearing and proof-bearing frames bound to b.
func validRecoveryFrame(b Binding, f RecoveryFrame) error {
	switch f.Kind {
	case "snapshot", "replacement", "barrier":
		if f.Record.View != b.View || f.Record.Revision == 0 || f.Proof == "" {
			return ErrDenied
		}
	case "revision":
		if f.Record.View != b.View || f.Record.Revision == 0 || f.Proof != "" {
			return ErrDenied
		}
	case "cannot-resume":
		if f.Record.Revision != 0 || f.Proof != "" {
			return ErrDenied
		}
	default:
		return ErrDenied
	}
	return nil
}

// frameSize is the private accounting size of one emitted frame, not a wire format.
func frameSize(f RecoveryFrame) (int, error) {
	data, err := json.Marshal(f)
	return len(data), err
}

// mapProof keeps context categories and reports every other proof failure as unavailable.
func mapProof(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return ErrUnavailable
}

func validOwner(owner string) bool {
	return owner != "" && len(owner) <= maxOwnerBytes && !strings.Contains(owner, "/")
}

func validRecoveryLimits(l RecoveryLimits) bool {
	return l.Page > 0 && l.MaxBytes > 0 && l.MaxTokenBytes > 0 && l.CallbackTimeout > 0 && l.CleanupTimeout > 0
}
