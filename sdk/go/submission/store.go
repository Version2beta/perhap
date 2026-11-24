package submission

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/version2beta/perhap/sdk/go/core"
)

// Owner exclusively owns the private gateway admission store and trusted evaluator
// until Close. mu serializes authoritative decisions and writes; Close cancels and
// joins in-flight operations before releasing the SQLite connection.
type Owner struct {
	life   sync.Mutex // guards closed and entry of new operations
	closed bool
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	db       *sql.DB
	conn     *sql.Conn // single pinned connection holding the exclusive lock
	host     Host
	limits   Limits
	st       *state
	poisoned bool // a commit outcome is unknown; the handle must be reopened
}

// Client exposes only scoped Submit/Resolve, not trusted admission/evaluation writes.
type Client struct {
	owner *Owner
	scope string
}

// Open recovers the private store before returning trusted authority. A missing file
// is created; an existing file is never treated as fresh authoritative state.
func Open(ctx context.Context, path string, h Host, l Limits) (*Owner, error) {
	if h == nil || !validLimits(l) || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, '?') {
		return nil, ErrInvalid
	}
	created, err := createFile(path)
	if err != nil {
		return nil, err
	}
	o := &Owner{host: h, limits: l}
	o.ctx, o.cancel = context.WithCancel(context.Background())
	if err := o.openDB(ctx, path, created); err != nil {
		o.abandon()
		return nil, err
	}
	if err := h.Recover(ctx, o.st.generation, o.st.heads()); err != nil {
		o.abandon()
		return nil, publicErr(err, ErrIntegrity)
	}
	return o, nil
}

// Client returns a facade bound to one context scope; it cannot admit or evaluate.
func (o *Owner) Client(scope string) *Client { return &Client{owner: o, scope: scope} }

// Close cancels in-flight operations, waits for them, and releases the database.
// It is idempotent.
func (o *Owner) Close() error {
	o.life.Lock()
	if o.closed {
		o.life.Unlock()
		return nil
	}
	o.closed = true
	o.life.Unlock()
	o.cancel()
	o.wg.Wait()
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := o.releaseDB(); err != nil {
		return ErrUnavailable
	}
	return nil
}

// Submit records a durable receipt, asks the host for admission only while the decision
// is pending, and releases the latest complete facts through the host fence.
func (c *Client) Submit(ctx context.Context, in Input, r Recipient) error {
	o := c.owner
	done, err := o.enter()
	if err != nil {
		return err
	}
	defer done()
	ctx, stop := o.bound(ctx)
	defer stop()
	if r == nil {
		return ErrInvalid
	}
	if in.Event.Context != c.scope {
		return ErrDenied
	}
	if err := validateEvent(in.Event, in.Schema); err != nil {
		return err
	}
	actor, err := o.host.Authorize(ctx, "submit", in.Event.ID, in.Event.Context, in.AttributionBinding)
	if err != nil {
		return publicErr(err, ErrDenied)
	}
	codec, err := o.host.Codec(in.Schema)
	if err != nil {
		return publicErr(err, ErrInvalid)
	}
	norm, err := codec.Normalize(cloneEvent(in.Event))
	if err != nil || !sameIdentity(in.Event, norm) {
		return ErrInvalid
	}
	ev, err := newStoredEvent(norm, in.Schema, actor.Asserting)
	if err != nil {
		return err
	}
	// MaxEventBytes bounds the normalized envelope, which is what is stored and evaluated.
	if n, err := envelopeBytes(ev); err != nil {
		return err
	} else if n > o.limits.MaxEventBytes {
		return ErrCapacity
	}
	reply, err := o.submit(ctx, actor, ev, Input{Event: norm, Schema: in.Schema, AttributionBinding: in.AttributionBinding})
	if err != nil {
		return err
	}
	return o.release(ctx, actor, ev.ID, reply, r)
}

// Resolve releases the latest retained facts for one event ID in this client's scope.
// Absence means only that no knowledge is retained at this cut.
func (c *Client) Resolve(ctx context.Context, id string, r Recipient) error {
	o := c.owner
	done, err := o.enter()
	if err != nil {
		return err
	}
	defer done()
	ctx, stop := o.bound(ctx)
	defer stop()
	if r == nil {
		return ErrInvalid
	}
	actor, err := o.host.Authorize(ctx, "resolve", id, c.scope, "")
	if err != nil {
		return publicErr(err, ErrDenied)
	}
	reply, err := o.lookup(c.scope, id)
	if err != nil {
		return err
	}
	return o.release(ctx, actor, id, reply, r)
}

// Evaluate runs the registered model through CORE over the admitted finite history of
// the event's context, entity and schema, then records the outcome as a new fact.
// Repeating an attempt for the same event is idempotent.
func (o *Owner) Evaluate(ctx context.Context, id, attempt string) error {
	done, err := o.enter()
	if err != nil {
		return err
	}
	defer done()
	ctx, stop := o.bound(ctx)
	defer stop()
	if !text(attempt) || len(attempt) > maxAttemptBytes {
		return ErrInvalid
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := o.usable(); err != nil {
		return err
	}
	if owner, used := o.st.attempts[attempt]; used {
		if owner == id {
			return nil
		}
		return ErrConflict
	}
	target, ok := o.st.entries[id]
	if !ok || target.facts.Admission != "admitted" {
		return ErrBlocked
	}
	codec, err := o.host.Codec(target.ev.Schema)
	if err != nil {
		return publicErr(err, ErrUnavailable)
	}
	hist := o.st.history(target.ev.Context, target.ev.EntityID, target.ev.Schema)
	boundary := o.st.generation + ":" + strconv.Itoa(o.st.records)
	res, err := evaluateAdmitted(codec, target, hist, boundary)
	if err != nil {
		return err
	}
	outcome, err := outcomeOf(res, id)
	if err != nil {
		return err
	}
	// The evidence reference is opaque and random per attempt; it reveals no history digest.
	ref, err := uuid.NewRandom()
	if err != nil {
		return ErrUnavailable
	}
	return o.commit(ctx, "evaluation", record{
		Kind: "evaluation", EventID: id, Version: target.facts.Version + 1,
		Attempt: attempt, Evaluation: outcome, Evidence: ref.String(),
	})
}

// enter admits one operation unless Close has begun.
func (o *Owner) enter() (func(), error) {
	o.life.Lock()
	defer o.life.Unlock()
	if o.closed {
		return nil, ErrClosed
	}
	o.wg.Add(1)
	return o.wg.Done, nil
}

// bound applies the per-operation timeout and cancels the operation on Close.
func (o *Owner) bound(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, o.limits.Timeout)
	stop := context.AfterFunc(o.ctx, cancel)
	return ctx, func() {
		stop()
		cancel()
	}
}

// submit decides the authoritative receipt and admission under mu. The reply is built
// here and released by the caller after mu is dropped.
func (o *Owner) submit(ctx context.Context, actor Actor, ev storedEvent, in Input) (Reply, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := o.usable(); err != nil {
		return Reply{}, err
	}
	cur, found := o.st.entries[ev.ID]
	if found {
		if !sameSemantic(cur.ev, ev) {
			return Reply{}, ErrConflict
		}
	} else {
		if len(o.st.entries) >= o.limits.MaxEvents {
			return Reply{}, ErrCapacity
		}
		if err := o.commit(ctx, "receipt", record{Kind: "receipt", EventID: ev.ID, Version: 1, Event: &ev}); err != nil {
			return Reply{}, err
		}
		cur = o.st.entries[ev.ID]
	}
	if pendingAdmission(cur.facts.Admission) {
		if err := o.admit(ctx, actor, cur, in); err != nil {
			return Reply{}, err
		}
	}
	return replyOf(o.st.entries[ev.ID], ev.ID, ev.Context), nil
}

// admit calls the host decision callback at most once. Zero callbacks leave the
// receipt pending; a committed decision stands even if the host later reports an error.
func (o *Owner) admit(ctx context.Context, actor Actor, cur *entry, in Input) error {
	var (
		called, committed bool
		fnErr             error
	)
	err := o.host.Admission(ctx, actor, in, func(decision string) error {
		if called {
			return ErrInvalid
		}
		called = true
		if !validDecision(decision) {
			fnErr = ErrInvalid
			return fnErr
		}
		if decision == cur.facts.Admission {
			return nil
		}
		fnErr = o.commit(ctx, "admission", record{
			Kind: "admission", EventID: cur.facts.EventID, Version: cur.facts.Version + 1, Decision: decision,
		})
		committed = fnErr == nil
		return fnErr
	})
	if fnErr != nil {
		return fnErr
	}
	if err != nil && !committed {
		return publicErr(err, ErrBlocked)
	}
	return nil
}

// lookup reads the latest facts for an event in the caller's scope. Another scope's
// event is reported exactly like an absent one.
func (o *Owner) lookup(scope, id string) (Reply, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := o.usable(); err != nil {
		return Reply{}, err
	}
	return replyOf(o.st.entries[id], id, scope), nil
}

// replyOf copies the latest full facts, or reports that no knowledge is retained.
func replyOf(e *entry, id, scope string) Reply {
	if e == nil || e.ev.ID != id || e.ev.Context != scope {
		return Reply{Kind: "no-retained-knowledge"}
	}
	return Reply{Kind: "facts", Facts: e.facts}
}

// release delivers reply through the host fence. The recipient runs only inside the
// host's synchronous Release call, while the context is live, and at most once. Once it
// has been called, any error, expiry or host misuse is an uncertain release, not a rollback.
func (o *Owner) release(ctx context.Context, actor Actor, id string, reply Reply, r Recipient) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	g := &replyGate{}
	hostErr := o.host.Release(ctx, actor, id, func() error { return g.deliver(ctx, reply, r) })
	return g.finish(ctx, hostErr)
}

// replyGate owns the accounting for one reply, whatever the host does with the fence.
type replyGate struct {
	mu           sync.Mutex
	sealed       bool  // Release has returned; a late host call cannot reach the recipient
	called       bool  // the recipient has been invoked
	misused      bool  // the host invoked the fence again, or after Release returned
	recipientErr error // the recipient's own result
}

// deliver is the fence the host calls. It refuses repeats and late calls and checks the
// context before any recipient call.
func (g *replyGate) deliver(ctx context.Context, reply Reply, r Recipient) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.sealed || g.called {
		g.misused = true
		return ErrRelease
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	g.called = true
	g.recipientErr = r(ctx, reply)
	return g.recipientErr
}

// finish seals the gate and classifies the outcome. A host error or context expiry after
// the recipient call is uncertain even if the host swallowed it.
func (g *replyGate) finish(ctx context.Context, hostErr error) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.sealed = true
	if !g.called {
		if hostErr != nil {
			return publicErr(hostErr, ErrDenied)
		}
		return ErrDenied
	}
	if g.misused || g.recipientErr != nil || hostErr != nil || ctx.Err() != nil {
		return ErrRelease
	}
	return nil
}

// publicErr maps host or internal failures onto the public sentinels. Context causes
// become the canonical context sentinels; wrapped private text never escapes.
func publicErr(err, fallback error) error {
	if err == nil {
		return nil
	}
	for _, p := range publicErrors {
		if errors.Is(err, p) {
			return p
		}
	}
	if errors.Is(err, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return fallback
}

var publicErrors = []error{ErrDenied, ErrBlocked, ErrConflict, ErrUnavailable, ErrIntegrity, ErrUnknown, ErrCapacity, ErrClosed, ErrInvalid, ErrRelease}

// abandon releases an Owner that failed during Open.
func (o *Owner) abandon() {
	o.cancel()
	_ = o.releaseDB()
}

// outcomeOf maps the CORE result for one target onto the public evaluation axis.
// A blocked target is not-established (ErrBlocked). A model-level CORE failure or a
// target that did not succeed is failed. Determinate domain-invalid is still evaluated.
func outcomeOf(res core.Result[int64], id string) (string, error) {
	if contains(res.Blocked, id) {
		return "", ErrBlocked
	}
	if !contains(res.Attempted, id) {
		return "", ErrUnavailable
	}
	if res.Evaluation == "failed" || !contains(res.Successful, id) {
		return "failed", nil
	}
	return "evaluated", nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func validLimits(l Limits) bool {
	return l.MaxEvents > 0 && l.MaxEntries > 0 && l.MaxBytes > 0 && l.MaxEventBytes > 0 && l.Timeout > 0
}
