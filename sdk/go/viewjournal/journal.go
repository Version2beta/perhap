package viewjournal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// Journal exclusively owns a private local SQLite observation journal.
type Journal struct {
	mu       sync.Mutex
	db       *sql.DB
	conn     *sql.Conn // the single pinned connection that holds the exclusive lock
	host     Host
	limits   Limits
	closed   bool
	poisoned bool // a commit outcome is unknown; the handle must be reopened
}

// Open recovers and validates the journal under trusted independent authority.
func Open(ctx context.Context, path string, host Host, limits Limits) (*Journal, error) {
	if host == nil || limits.MaxEvaluations <= 0 || limits.MaxPins <= 0 || limits.MaxBatch <= 0 || limits.MaxBytes <= 0 || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, '?') {
		return nil, errInvalidArgument
	}
	created, err := createFile(path)
	if err != nil {
		return nil, err
	}
	j := &Journal{host: host, limits: limits}
	st, err := j.open(ctx, path, created)
	if err != nil {
		_ = j.release()
		return nil, err
	}
	if err := st.withinLimits(limits); err != nil {
		_ = j.release()
		return nil, err
	}
	if err := host.Recover(ctx, st.headList()); err != nil {
		_ = j.release()
		return nil, err
	}
	return j, nil
}

// Accept atomically records a verified evaluation and any material public change.
func (j *Journal) Accept(ctx context.Context, r Request) (Receipt, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return Receipt{}, err
	}
	if r.ID == "" || r.EvaluationID == "" {
		return Receipt{}, errInvalidArgument
	}
	// Authorize and look up the retry before consulting Within: an exact retry
	// returns its original receipt even if the evaluation is no longer current.
	if err := j.host.Authorize(ctx, r.View); err != nil {
		return Receipt{}, err
	}
	st, err := j.load(ctx)
	if err != nil {
		return Receipt{}, err
	}
	if rc, found, err := st.lookup(r); found {
		return rc, err
	}
	if err := st.precheck(r, j.limits); err != nil {
		return Receipt{}, err
	}
	return j.evaluate(ctx, r)
}

// Heads returns private recovery information, not a public observation.
func (j *Journal) Heads(ctx context.Context) ([]Head, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return nil, err
	}
	st, err := j.load(ctx)
	if err != nil {
		return nil, err
	}
	return st.headList(), nil
}

// Plans enumerates private durable pins for recovery after a lost planning reply.
func (j *Journal) Plans(ctx context.Context) ([]Plan, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return nil, err
	}
	st, err := j.load(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Plan, 0, len(st.plans))
	for _, p := range st.plans {
		out = append(out, p)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].ID < out[b].ID })
	for _, p := range out {
		if err := j.host.Authorize(ctx, p.View); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Prepare persists a consistent snapshot/successor retention plan; no data is disclosed.
func (j *Journal) Prepare(ctx context.Context, v Identity) (Plan, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return Plan{}, err
	}
	if err := j.host.Authorize(ctx, v); err != nil {
		return Plan{}, err
	}
	var plan Plan
	_, err := j.transact(ctx, func(st *journalState) (*entry, error) {
		snap, ok := st.latest(v)
		if !ok {
			return nil, fmt.Errorf("%w: no evaluated public snapshot", ErrUnavailable)
		}
		if len(st.plans) >= j.limits.MaxPins {
			return nil, ErrCapacity
		}
		plan = Plan{ID: uuid.NewString(), View: v, Snapshot: snap}
		return &entry{Format: entryFormat, Sequence: st.entries + 1, Kind: kindPrepare, Plan: &plan}, nil
	})
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

// Read returns a complete bounded successor range or an error with no intention.
func (j *Journal) Read(ctx context.Context, id string, after uint64, max int) (Batch, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return Batch{}, err
	}
	if max < 1 {
		return Batch{}, errInvalidArgument
	}
	if max > j.limits.MaxBatch {
		return Batch{}, ErrCapacity
	}
	if err := ctx.Err(); err != nil {
		return Batch{}, err
	}
	st, err := j.load(ctx)
	if err != nil {
		return Batch{}, err
	}
	p, ok := st.plans[id]
	if !ok {
		return Batch{}, errUnknownPlan
	}
	if err := j.host.Authorize(ctx, p.View); err != nil {
		return Batch{}, err
	}
	h := st.head(p.View)
	if after < p.Snapshot.Revision || after > h.Revision {
		return Batch{}, fmt.Errorf("%w: position outside plan tail", ErrStale)
	}
	// The selected head is the committed head at this read cut; the batch covers
	// only (after, through] and may be a partial catch-up when through < head.
	through := h.Revision
	if h.Revision-after > uint64(max) {
		through = after + uint64(max)
	}
	recs := make([]Record, through-after)
	copy(recs, st.records[p.View][after:through])
	b := Batch{PlanID: id, View: p.View, After: after, Through: through, SelectedHead: h.Revision, Records: recs}
	if err := j.hook("during_read"); err != nil {
		return Batch{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return Batch{}, err
	}
	return b, nil
}

// Release durably releases a private plan; accepted history is never pruned here.
func (j *Journal) Release(ctx context.Context, id string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.usable(); err != nil {
		return err
	}
	st, err := j.load(ctx)
	if err != nil {
		return err
	}
	p, ok := st.plans[id]
	if !ok {
		// Releasing an unknown or already released plan is an idempotent no-op.
		return nil
	}
	if err := j.host.Authorize(ctx, p.View); err != nil {
		return err
	}
	_, err = j.transact(ctx, func(st *journalState) (*entry, error) {
		if _, ok := st.plans[id]; !ok {
			return nil, nil
		}
		return &entry{Format: entryFormat, Sequence: st.entries + 1, Kind: kindRelease, Release: id}, nil
	})
	return err
}

// Close is idempotent and releases ownership, not persisted plans/history.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return nil
	}
	j.closed = true
	return j.release()
}

var (
	errInvalidArgument = errors.New("viewjournal: invalid argument")
	errClosed          = fmt.Errorf("%w: handle closed", ErrUnavailable)
	errPoisoned        = fmt.Errorf("%w: handle poisoned by an unconfirmed commit; reopen", ErrUnknown)
	errHostProtocol    = errors.New("viewjournal: host violated verification contract")
	errCorrupt         = fmt.Errorf("%w: corrupt or inconsistent journal record", ErrUnavailable)
	errUnknownPlan     = fmt.Errorf("%w: unknown or released plan", ErrStale)
)

const (
	storageFormat  = "perhap-view-journal"
	storageVersion = "1"
	entryFormat    = 1
	kindAccept     = "accept"
	kindPrepare    = "prepare"
	kindRelease    = "release"
)

var schemaStatements = []string{
	`CREATE TABLE journal_meta (name TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	`CREATE TABLE journal_entries (seq INTEGER PRIMARY KEY, body TEXT NOT NULL)`,
	`INSERT INTO journal_meta (name, value) VALUES ('format', '` + storageFormat + `'), ('version', '` + storageVersion + `')`,
}

// entry is one private, versioned, append-only journal record. An accept entry
// carries the receipt, the verified rendering and the private cut together; plan
// entries carry only pin state. Replay validates every entry before use.
type entry struct {
	Format   int     `json:"format"`
	Sequence int     `json:"sequence"`
	Kind     string  `json:"kind"`
	Receipt  Receipt `json:"receipt"`
	Public   Public  `json:"public"`
	Cut      string  `json:"cut"`
	Plan     *Plan   `json:"plan"`
	Release  string  `json:"release"`
}

type receiptKey struct {
	view Identity
	id   string
}

// journalState is the validated replay of every durable entry.
type journalState struct {
	entries     int
	evaluations int
	bytes       int
	heads       map[Identity]Head
	receipts    map[receiptKey]Receipt
	records     map[Identity][]Record // records[v][i] has Revision i+1
	plans       map[string]Plan
	planIDs     map[string]bool
}

func newJournalState() *journalState {
	return &journalState{
		heads:    map[Identity]Head{},
		receipts: map[receiptKey]Receipt{},
		records:  map[Identity][]Record{},
		plans:    map[string]Plan{},
		planIDs:  map[string]bool{},
	}
}

// head returns the stored head, or the empty head for an unevaluated view.
func (st *journalState) head(v Identity) Head {
	if h, ok := st.heads[v]; ok {
		return h
	}
	return Head{View: v}
}

// latest returns the newest public record of a view, if any evaluation was accepted.
func (st *journalState) latest(v Identity) (Record, bool) {
	rs := st.records[v]
	if len(rs) == 0 {
		return Record{}, false
	}
	return rs[len(rs)-1], true
}

func (st *journalState) headList() []Head {
	out := make([]Head, 0, len(st.heads))
	for _, h := range st.heads {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool { return identityLess(out[i].View, out[j].View) })
	return out
}

// lookup returns the original receipt for an identical retry, or ErrConflict when
// the same request ID in the same view carries different immutable request fields.
func (st *journalState) lookup(r Request) (Receipt, bool, error) {
	rc, ok := st.receipts[receiptKey{r.View, r.ID}]
	if !ok {
		return Receipt{}, false, nil
	}
	if rc.Request != r {
		return Receipt{}, true, ErrConflict
	}
	return rc, true, nil
}

// precheck verifies the expected private checkpoint and public revision, then capacity.
func (st *journalState) precheck(r Request, lim Limits) error {
	h := st.head(r.View)
	if r.ExpectedCheckpoint != h.Checkpoint || r.ExpectedRevision != h.Revision {
		return ErrStale
	}
	if st.evaluations >= lim.MaxEvaluations {
		return ErrCapacity
	}
	return nil
}

// withinLimits refuses to serve a stored state that exceeds the configured bounds.
func (st *journalState) withinLimits(lim Limits) error {
	if st.evaluations > lim.MaxEvaluations || st.bytes > lim.MaxBytes || len(st.plans) > lim.MaxPins {
		return fmt.Errorf("%w: stored state exceeds configured limits", ErrUnavailable)
	}
	return nil
}

// apply validates one entry against the state built so far and then folds it in.
// Any inconsistency is corruption, never a reset or a repaired state.
func (st *journalState) apply(e entry, size int) error {
	if e.Format != entryFormat || e.Sequence != st.entries+1 {
		return errCorrupt
	}
	var err error
	switch e.Kind {
	case kindAccept:
		err = st.applyAccept(e)
	case kindPrepare:
		err = st.applyPrepare(e)
	case kindRelease:
		err = st.applyRelease(e)
	default:
		err = errCorrupt
	}
	if err != nil {
		return err
	}
	st.entries++
	st.bytes += size
	return nil
}

func (st *journalState) applyAccept(e entry) error {
	r, rc := e.Receipt.Request, e.Receipt
	v := r.View
	if e.Plan != nil || e.Release != "" || r.ID == "" || e.Cut == "" || !validPublic(e.Public) {
		return errCorrupt
	}
	if _, dup := st.receipts[receiptKey{v, r.ID}]; dup {
		return errCorrupt
	}
	prev := st.head(v)
	if r.ExpectedCheckpoint != prev.Checkpoint || r.ExpectedRevision != prev.Revision {
		return errCorrupt
	}
	if prev.Checkpoint == math.MaxUint64 || rc.Checkpoint != prev.Checkpoint+1 {
		return errCorrupt
	}
	if uint64(len(st.records[v])) != prev.Revision {
		return errCorrupt
	}
	changed := true
	if last, ok := st.latest(v); ok {
		changed = !samePublic(last.Public, e.Public)
	}
	if rc.Changed != changed {
		return errCorrupt
	}
	revision := prev.Revision
	if changed {
		if prev.Revision == math.MaxUint64 || rc.Revision != prev.Revision+1 {
			return errCorrupt
		}
		revision = rc.Revision
		st.records[v] = append(st.records[v], Record{View: v, Revision: revision, Previous: prev.Revision, Public: e.Public})
	} else if rc.Revision != prev.Revision {
		return errCorrupt
	}
	st.heads[v] = Head{View: v, Checkpoint: rc.Checkpoint, Revision: revision}
	st.receipts[receiptKey{v, r.ID}] = rc
	st.evaluations++
	return nil
}

func (st *journalState) applyPrepare(e entry) error {
	if e.Plan == nil || e.Release != "" || e.Cut != "" || e.Receipt != (Receipt{}) {
		return errCorrupt
	}
	p := *e.Plan
	if p.ID == "" || st.planIDs[p.ID] {
		return errCorrupt
	}
	last, ok := st.latest(p.View)
	if !ok || !sameRecord(p.Snapshot, last) {
		return errCorrupt
	}
	st.planIDs[p.ID] = true
	st.plans[p.ID] = p
	return nil
}

func (st *journalState) applyRelease(e entry) error {
	if e.Plan != nil || e.Release == "" || e.Cut != "" || e.Receipt != (Receipt{}) {
		return errCorrupt
	}
	if _, ok := st.plans[e.Release]; !ok {
		return errCorrupt
	}
	delete(st.plans, e.Release)
	return nil
}

// validPublic checks the public rendering's enumerations and the Available/Value rule.
// Available true with a nil Value is a valid null; Available false requires nil Value.
func validPublic(p Public) bool {
	if !p.Available && p.Value != nil {
		return false
	}
	switch p.Evaluation {
	case "valid", "invalid", "failed":
	default:
		return false
	}
	switch p.Completeness {
	case "complete_as_of_frontier", "incomplete", "unavailable":
	default:
		return false
	}
	return true
}

// samePublic compares the complete typed public rendering, never pointer addresses.
func samePublic(a, b Public) bool {
	if (a.Value == nil) != (b.Value == nil) {
		return false
	}
	if a.Value != nil && a.Value.Total != b.Value.Total {
		return false
	}
	return a.Available == b.Available && a.Evaluation == b.Evaluation &&
		a.Completeness == b.Completeness && a.Evidence == b.Evidence
}

func sameRecord(a, b Record) bool {
	return a.View == b.View && a.Revision == b.Revision && a.Previous == b.Previous && samePublic(a.Public, b.Public)
}

func identityLess(a, b Identity) bool {
	fa := [...]string{a.Context, a.Projection, a.Parameters, a.Model, a.Schema, a.AuthorizedView, a.Generation}
	fb := [...]string{b.Context, b.Projection, b.Parameters, b.Model, b.Schema, b.AuthorizedView, b.Generation}
	for i := range fa {
		if fa[i] != fb[i] {
			return fa[i] < fb[i]
		}
	}
	return false
}

// evaluate runs the host fence around one commit. fn is called at most once, and
// a commit attempt that cannot be confirmed is reported as ErrUnknown.
func (j *Journal) evaluate(ctx context.Context, r Request) (Receipt, error) {
	var (
		rc        Receipt
		called    bool
		attempted bool
		commitErr error
	)
	err := j.host.Within(ctx, r, func(v Verified) error {
		if called {
			return errHostProtocol
		}
		called = true
		if v.PrivateCut == "" || !validPublic(v.Public) {
			return errHostProtocol
		}
		build := func(st *journalState) (*entry, error) {
			if prior, found, err := st.lookup(r); found {
				rc = prior
				return nil, err
			}
			if err := st.precheck(r, j.limits); err != nil {
				return nil, err
			}
			h := st.head(r.View)
			if h.Checkpoint == math.MaxUint64 {
				return nil, ErrCapacity
			}
			changed := true
			if last, ok := st.latest(r.View); ok {
				changed = !samePublic(last.Public, v.Public)
			}
			revision := h.Revision
			if changed {
				if h.Revision == math.MaxUint64 {
					return nil, ErrCapacity
				}
				revision++
			}
			rc = Receipt{Request: r, Checkpoint: h.Checkpoint + 1, Revision: revision, Changed: changed}
			return &entry{Format: entryFormat, Sequence: st.entries + 1, Kind: kindAccept, Receipt: rc, Public: v.Public, Cut: v.PrivateCut}, nil
		}
		attempted, commitErr = j.transact(ctx, build)
		return commitErr
	})
	if attempted {
		if err == nil && commitErr == nil {
			return rc, nil
		}
		j.poisoned = true
		if err == nil {
			err = commitErr
		}
		return Receipt{}, fmt.Errorf("%w: %w", ErrUnknown, err)
	}
	if err != nil {
		return Receipt{}, err
	}
	if !called {
		return Receipt{}, ErrDenied
	}
	return rc, nil
}

// transact runs one exclusive transaction. build returns a nil entry for a no-op.
// attempted reports whether COMMIT was issued; once it is, the outcome is unknown on error.
func (j *Journal) transact(ctx context.Context, build func(*journalState) (*entry, error)) (bool, error) {
	if _, err := j.conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return false, storageError(err)
	}
	wrote, err := j.stage(ctx, build)
	if err != nil {
		return false, j.rollback(ctx, err)
	}
	if !wrote {
		return false, j.rollback(ctx, nil)
	}
	if _, err := j.conn.ExecContext(ctx, "COMMIT"); err != nil {
		j.poisoned = true
		return true, fmt.Errorf("%w: commit: %w", ErrUnknown, err)
	}
	if err := j.hook("after_commit"); err != nil {
		j.poisoned = true
		return true, fmt.Errorf("%w: %w", ErrUnknown, err)
	}
	return true, nil
}

// stage performs the checks and the single insert inside an open transaction.
func (j *Journal) stage(ctx context.Context, build func(*journalState) (*entry, error)) (bool, error) {
	st, err := loadState(ctx, j.conn)
	if err != nil {
		return false, err
	}
	e, err := build(st)
	if err != nil {
		return false, err
	}
	if e == nil {
		return false, nil
	}
	body, err := json.Marshal(*e)
	if err != nil {
		return false, storageError(err)
	}
	if err := reserveCleanup(st, e, len(body), j.limits.MaxBytes); err != nil {
		return false, err
	}
	if err := st.apply(*e, len(body)); err != nil {
		return false, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if _, err := j.conn.ExecContext(ctx, "INSERT INTO journal_entries (seq, body) VALUES (?, ?)", e.Sequence, string(body)); err != nil {
		return false, storageError(err)
	}
	if err := j.hook("before_commit"); err != nil {
		return false, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return true, nil
}

// rollback ends an uncommitted transaction. A failed rollback poisons the handle and
// is added to the primary error rather than replacing it.
func (j *Journal) rollback(ctx context.Context, primary error) error {
	if _, rbErr := j.conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); rbErr != nil {
		j.poisoned = true
		if primary == nil {
			return fmt.Errorf("%w: rollback: %w", ErrUnknown, rbErr)
		}
		return fmt.Errorf("%w; rollback failed: %w", primary, rbErr)
	}
	return primary
}

func (j *Journal) hook(point string) error {
	if j.limits.Cut == nil {
		return nil
	}
	return j.limits.Cut(point)
}

func (j *Journal) usable() error {
	if j.closed {
		return errClosed
	}
	if j.poisoned {
		return errPoisoned
	}
	return nil
}

func (j *Journal) load(ctx context.Context) (*journalState, error) {
	return loadState(ctx, j.conn)
}

func loadState(ctx context.Context, conn *sql.Conn) (*journalState, error) {
	rows, err := conn.QueryContext(ctx, "SELECT seq, body FROM journal_entries ORDER BY seq")
	if err != nil {
		return nil, storageError(err)
	}
	defer rows.Close()
	st := newJournalState()
	for rows.Next() {
		var seq int64
		var body string
		if err := rows.Scan(&seq, &body); err != nil {
			return nil, storageError(err)
		}
		if seq != int64(st.entries+1) {
			return nil, errCorrupt
		}
		e, err := decodeEntry(body)
		if err != nil {
			return nil, errCorrupt
		}
		if err := st.apply(e, len(body)); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, storageError(err)
	}
	return st, nil
}

func decodeEntry(body string) (entry, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	var e entry
	if err := dec.Decode(&e); err != nil {
		return entry{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return entry{}, errCorrupt
	}
	return e, nil
}

// open opens the pinned connection, verifies durability settings, takes the exclusive
// lock, and validates or creates the schema before returning the replayed state.
func (j *Journal) open(ctx context.Context, path string, created bool) (*journalState, error) {
	// The driver runs its own PRAGMAs during connection open, which take a shared lock.
	// A zero busy timeout in the DSN makes a competing owner fail at once instead of
	// waiting out the driver default of 5 seconds; the explicit pragma below verifies it.
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=0")
	if err != nil {
		return nil, storageError(err)
	}
	j.db = db
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, openError(err)
	}
	j.conn = conn
	if err := setPragmas(ctx, conn); err != nil {
		return nil, openError(err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return nil, openError(err)
	}
	if err := initOrValidate(ctx, conn, created); err != nil {
		return nil, openError(err)
	}
	st, err := loadState(ctx, conn)
	if err != nil {
		return nil, openError(err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return nil, openError(err)
	}
	return st, nil
}

// release closes the pinned connection and the pool; closing the driver connection
// releases the SQLite exclusive lock.
func (j *Journal) release() error {
	var err error
	if j.conn != nil {
		err = j.conn.Close()
	}
	if j.db != nil {
		if cerr := j.db.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// createFile creates the database file only if absent. An existing file is never
// treated as new, so an empty or truncated journal cannot be silently reset.
func createFile(path string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, storageError(err)
	}
	if err := f.Close(); err != nil {
		return false, storageError(err)
	}
	return true, nil
}

func setPragmas(ctx context.Context, conn *sql.Conn) error {
	for _, stmt := range []string{
		"PRAGMA busy_timeout=0",
		"PRAGMA journal_mode=DELETE",
		"PRAGMA synchronous=EXTRA",
		"PRAGMA fullfsync=ON",
		"PRAGMA locking_mode=EXCLUSIVE",
	} {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	checks := []struct{ query, want string }{
		{"PRAGMA busy_timeout", "0"},
		{"PRAGMA journal_mode", "delete"},
		{"PRAGMA synchronous", "3"},
		{"PRAGMA fullfsync", "1"},
		{"PRAGMA locking_mode", "exclusive"},
	}
	for _, c := range checks {
		var got string
		if err := conn.QueryRowContext(ctx, c.query).Scan(&got); err != nil {
			return err
		}
		if got != c.want {
			return fmt.Errorf("%w: %s is %q, want %q", ErrUnavailable, c.query, got, c.want)
		}
	}
	return nil
}

func initOrValidate(ctx context.Context, conn *sql.Conn, created bool) error {
	names, err := tableNames(ctx, conn)
	if err != nil {
		return err
	}
	if len(names) == 0 && created {
		for _, stmt := range schemaStatements {
			if _, err := conn.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}
		return nil
	}
	if strings.Join(names, ",") != "journal_entries,journal_meta" {
		return fmt.Errorf("%w: not a view journal database", ErrUnavailable)
	}
	meta := map[string]string{}
	rows, err := conn.QueryContext(ctx, "SELECT name, value FROM journal_meta")
	if err != nil {
		return err
	}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			rows.Close()
			return err
		}
		meta[name] = value
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if meta["format"] != storageFormat || meta["version"] != storageVersion || len(meta) != 2 {
		return fmt.Errorf("%w: unsupported view journal format", ErrUnavailable)
	}
	integrity, err := conn.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return err
	}
	defer integrity.Close()
	var results []string
	for integrity.Next() {
		var s string
		if err := integrity.Scan(&s); err != nil {
			return err
		}
		results = append(results, s)
	}
	if err := integrity.Err(); err != nil {
		return err
	}
	if len(results) != 1 || results[0] != "ok" {
		return fmt.Errorf("%w: integrity check failed", ErrUnavailable)
	}
	return nil
}

func tableNames(ctx context.Context, conn *sql.Conn) ([]string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// storageError marks a storage failure as unavailable while keeping the driver error.
func storageError(err error) error {
	return fmt.Errorf("%w: %w", ErrUnavailable, err)
}

// openError maps SQLite lock contention to a competing owner and every other
// failure to unavailable storage.
func openError(err error) error {
	var se sqlite3.Error
	if errors.As(err, &se) && (se.Code == sqlite3.ErrBusy || se.Code == sqlite3.ErrLocked) {
		return fmt.Errorf("%w: %w", ErrOwned, err)
	}
	if errors.Is(err, ErrUnavailable) {
		return err
	}
	return storageError(err)
}
