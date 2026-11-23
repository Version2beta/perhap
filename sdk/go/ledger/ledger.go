package ledger

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
	"github.com/version2beta/perhap/sdk/go/rev"
)

// Ledger exclusively owns one private local SQLite database until Close.
type Ledger struct {
	mu       sync.Mutex
	db       *sql.DB
	conn     *sql.Conn // the single pinned connection that holds the exclusive lock
	host     Host
	max      int
	cut      func(string) error
	closed   bool
	poisoned bool // a commit outcome is unknown; the handle must be reopened
}

// Open validates and recovers a bounded ledger before returning authority to use it.
func Open(ctx context.Context, path string, host Host, options Options) (*Ledger, error) {
	if host == nil || options.MaxRecords <= 0 || !filepath.IsAbs(path) || filepath.Clean(path) != path || strings.ContainsRune(path, '?') {
		return nil, errInvalidArgument
	}
	created, err := createFile(path)
	if err != nil {
		return nil, err
	}
	l := &Ledger{host: host, max: options.MaxRecords, cut: options.Cut}
	st, err := l.open(ctx, path, created)
	if err != nil {
		_ = l.release()
		return nil, err
	}
	if err := host.Recover(ctx, st.headList()); err != nil {
		_ = l.release()
		return nil, err
	}
	return l, nil
}

// Accept atomically records one host-verified evaluation or returns its original receipt.
func (l *Ledger) Accept(ctx context.Context, r Request) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.usable(); err != nil {
		return Receipt{}, err
	}
	if r.ID == "" || r.Generation == "" || r.EvaluationID == "" || r.Key.Producer == "" {
		return Receipt{}, errInvalidArgument
	}
	if err := l.host.Authorize(ctx, r); err != nil {
		return Receipt{}, err
	}
	st, err := l.load(ctx)
	if err != nil {
		return Receipt{}, err
	}
	if rc, found, err := st.lookup(r); found {
		return rc, err
	}
	if err := st.precheck(r, l.max); err != nil {
		return Receipt{}, err
	}
	return l.evaluate(ctx, r)
}

// Pending returns copies of immutable locally scheduled records; no delivery occurs.
func (l *Ledger) Pending(ctx context.Context) ([]Publication, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.usable(); err != nil {
		return nil, err
	}
	st, err := l.load(ctx)
	if err != nil {
		return nil, err
	}
	return st.pending, nil
}

// Heads returns private owner-only recovery evidence, not a public observation.
func (l *Ledger) Heads(ctx context.Context) ([]Head, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.usable(); err != nil {
		return nil, err
	}
	st, err := l.load(ctx)
	if err != nil {
		return nil, err
	}
	return st.headList(), nil
}

// Close releases the exclusive owner and is idempotent.
func (l *Ledger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	return l.release()
}

var (
	errInvalidArgument = errors.New("ledger: invalid argument")
	errClosed          = fmt.Errorf("%w: handle closed", ErrUnavailable)
	errPoisoned        = fmt.Errorf("%w: handle poisoned by an unconfirmed commit; reopen", ErrUnknown)
	errHostProtocol    = errors.New("ledger: host violated evaluation callback contract")
	errCorrupt         = fmt.Errorf("%w: corrupt or inconsistent ledger record", ErrUnavailable)
)

const (
	storageFormat  = "perhap-publication-ledger"
	storageVersion = "1"
	recordFormat   = 1
)

var schemaStatements = []string{
	`CREATE TABLE ledger_meta (name TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	`CREATE TABLE ledger_records (seq INTEGER PRIMARY KEY, body TEXT NOT NULL)`,
	`INSERT INTO ledger_meta (name, value) VALUES ('format', '` + storageFormat + `'), ('version', '` + storageVersion + `')`,
}

// storedRecord is one private, versioned, append-only log entry. It is the only
// durable unit: receipt, evidence, post-state head and pending obligation together.
type storedRecord struct {
	Format   int      `json:"format"`
	Sequence int      `json:"sequence"`
	Receipt  Receipt  `json:"receipt"`
	Evidence Evidence `json:"evidence"`
	Head     Head     `json:"head"`
	Pending  bool     `json:"pending"`
}

type headKey struct {
	key        rev.Key
	generation string
}

type identityKey struct {
	key        rev.Key
	generation string
	id         string
}

// ledgerState is the validated replay of every durable record.
type ledgerState struct {
	count          int
	heads          map[headKey]Head
	receipts       map[identityKey]Receipt
	publicationIDs map[string]bool
	pending        []Publication
}

func newState() *ledgerState {
	return &ledgerState{
		heads:          map[headKey]Head{},
		receipts:       map[identityKey]Receipt{},
		publicationIDs: map[string]bool{},
	}
}

// head returns the stored head, or the empty head for an unregistered key.
func (st *ledgerState) head(k rev.Key, generation string) Head {
	if h, ok := st.heads[headKey{k, generation}]; ok {
		return h
	}
	return Head{Key: k, Generation: generation}
}

func (st *ledgerState) headList() []Head {
	out := make([]Head, 0, len(st.heads))
	for _, h := range st.heads {
		out = append(out, h)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Key != b.Key {
			return keyLess(a.Key, b.Key)
		}
		return a.Generation < b.Generation
	})
	return out
}

func keyLess(a, b rev.Key) bool {
	if a.Context != b.Context {
		return a.Context < b.Context
	}
	if a.Producer != b.Producer {
		return a.Producer < b.Producer
	}
	if a.Subject != b.Subject {
		return a.Subject < b.Subject
	}
	return a.Meaning < b.Meaning
}

// lookup returns the original receipt for an identical retry, or ErrConflict when
// the same identity carries different immutable request fields.
func (st *ledgerState) lookup(r Request) (Receipt, bool, error) {
	rc, ok := st.receipts[identityKey{r.Key, r.Generation, r.ID}]
	if !ok {
		return Receipt{}, false, nil
	}
	if rc.Request != r {
		return Receipt{}, true, ErrConflict
	}
	return rc, true, nil
}

// precheck verifies the expected head and capacity for a new request.
func (st *ledgerState) precheck(r Request, max int) error {
	h := st.head(r.Key, r.Generation)
	if r.ExpectedCheckpoint != h.Checkpoint || r.ExpectedRevision != h.Assertion.Revision || r.ExpectedPublicationID != h.Assertion.PublicationID {
		return ErrStale
	}
	if st.count >= max {
		return ErrCapacity
	}
	return nil
}

// apply validates one record against the state built so far and then folds it in.
// Any inconsistency is corruption, never a reset or a repaired state.
func (st *ledgerState) apply(rec storedRecord) error {
	r := rec.Receipt.Request
	if rec.Format != recordFormat || rec.Sequence != st.count+1 {
		return errCorrupt
	}
	if r.Key != rec.Head.Key || r.Generation != rec.Head.Generation || rec.Receipt.Checkpoint != rec.Head.Checkpoint {
		return errCorrupt
	}
	if err := validEvidence(rec.Evidence, r); err != nil {
		return errCorrupt
	}
	ident := identityKey{r.Key, r.Generation, r.ID}
	if _, dup := st.receipts[ident]; dup {
		return errCorrupt
	}
	prev := st.head(r.Key, r.Generation)
	if r.ExpectedCheckpoint != prev.Checkpoint || r.ExpectedRevision != prev.Assertion.Revision || r.ExpectedPublicationID != prev.Assertion.PublicationID {
		return errCorrupt
	}
	if prev.Checkpoint == math.MaxUint64 || rec.Head.Checkpoint != prev.Checkpoint+1 {
		return errCorrupt
	}
	if rec.Receipt.Publication == nil {
		if rec.Pending || rec.Head.Assertion != prev.Assertion {
			return errCorrupt
		}
	} else {
		p := *rec.Receipt.Publication
		if !rec.Pending || p.Generation != r.Generation || p.Record.Key != r.Key || p.Record.ID == "" || st.publicationIDs[p.Record.ID] {
			return errCorrupt
		}
		if p.Schema != rec.Evidence.Schema || p.Rule != rec.Evidence.Rule || p.Attribution != rec.Evidence.Attribution || p.Cut != rec.Evidence.Cut {
			return errCorrupt
		}
		if prev.Assertion.Revision == math.MaxUint64 || p.Record.Revision != prev.Assertion.Revision+1 || p.Record.Supersedes != prev.Assertion.PublicationID {
			return errCorrupt
		}
		want, err := transition(prev.Assertion, p.Record)
		if err != nil || p.ValueAvailable != want.ValueAvailable || rec.Head.Assertion != want {
			return errCorrupt
		}
		st.publicationIDs[p.Record.ID] = true
		st.pending = append(st.pending, p)
	}
	st.heads[headKey{r.Key, r.Generation}] = rec.Head
	st.receipts[ident] = rec.Receipt
	st.count++
	return nil
}

// transition is the only allowed publication effect on an assertion. Status
// operations retain the prior value and availability; retraction clears them.
func transition(prev rev.Assertion, p rev.Publication) (rev.Assertion, error) {
	switch p.Op {
	case "upsert":
		if p.Status != "" {
			return rev.Assertion{}, errCorrupt
		}
		return rev.Assertion{Revision: p.Revision, PublicationID: p.ID, Status: "current", Value: p.Value, ValueAvailable: true}, nil
	case "retract":
		if p.Status != "" || p.Value != "" {
			return rev.Assertion{}, errCorrupt
		}
		return rev.Assertion{Revision: p.Revision, PublicationID: p.ID, Status: "retracted"}, nil
	case "status":
		if p.Value != prev.Value {
			return rev.Assertion{}, errCorrupt
		}
		if p.Status == "current" && !prev.ValueAvailable {
			return rev.Assertion{}, errCorrupt
		}
		if p.Status != "current" && p.Status != "unresolved" {
			return rev.Assertion{}, errCorrupt
		}
		return rev.Assertion{Revision: p.Revision, PublicationID: p.ID, Status: p.Status, Value: prev.Value, ValueAvailable: prev.ValueAvailable}, nil
	}
	return rev.Assertion{}, errCorrupt
}

// validEvidence accepts only host evidence that names this key and its accountable producer.
func validEvidence(ev Evidence, r Request) error {
	switch {
	case !ev.Registered:
		return fmt.Errorf("%w: key not registered", ErrDenied)
	case !ev.InternalAllowed:
		return fmt.Errorf("%w: internal evaluation not authorized", ErrDenied)
	case ev.Computation.Key != r.Key:
		return fmt.Errorf("%w: computation key mismatch", ErrDenied)
	case ev.Attribution != r.Key.Producer:
		return fmt.Errorf("%w: attribution mismatch", ErrDenied)
	case ev.Schema == "" || ev.Rule == "" || ev.Cut == "":
		return fmt.Errorf("%w: incomplete evidence provenance", ErrDenied)
	}
	return nil
}

// newRecord computes the next durable record from the validated state and host evidence.
// A publication is allocated only for a semantic change the host permits to publish.
func newRecord(st *ledgerState, r Request, ev Evidence) (storedRecord, error) {
	prev := st.head(r.Key, r.Generation)
	if prev.Checkpoint == math.MaxUint64 {
		return storedRecord{}, ErrCapacity
	}
	next := Head{Key: r.Key, Generation: r.Generation, Checkpoint: prev.Checkpoint + 1, Assertion: prev.Assertion}
	var pub *Publication
	decision := rev.Reconcile(prev.Assertion, ev.Computation)
	// Complete absence of a never-published key is not a publication.
	neverPublishedAbsence := prev.Assertion.PublicationID == "" && decision.Intention != nil && decision.Intention.Op == "retract"
	if decision.Intention != nil && ev.PublishAllowed && !neverPublishedAbsence {
		p, asr, err := newPublication(prev.Assertion, r, ev, *decision.Intention)
		if err != nil {
			return storedRecord{}, err
		}
		pub = &p
		next.Assertion = asr
	}
	return storedRecord{
		Format:   recordFormat,
		Sequence: st.count + 1,
		Receipt:  Receipt{Request: r, Checkpoint: next.Checkpoint, Publication: pub},
		Evidence: ev,
		Head:     next,
		Pending:  pub != nil,
	}, nil
}

func newPublication(prev rev.Assertion, r Request, ev Evidence, in rev.Intention) (Publication, rev.Assertion, error) {
	if prev.Revision == math.MaxUint64 {
		return Publication{}, rev.Assertion{}, ErrCapacity
	}
	p := rev.Publication{ID: uuid.NewString(), Key: r.Key, Revision: prev.Revision + 1, Op: in.Op, Supersedes: prev.PublicationID}
	switch in.Op {
	case "upsert":
		p.Value = in.Value
	case "status":
		p.Status = in.Status
		p.Value = prev.Value
	case "retract":
	default:
		return Publication{}, rev.Assertion{}, errCorrupt
	}
	asr, err := transition(prev, p)
	if err != nil {
		return Publication{}, rev.Assertion{}, err
	}
	return Publication{
		Record:         p,
		Generation:     r.Generation,
		Schema:         ev.Schema,
		Rule:           ev.Rule,
		Attribution:    ev.Attribution,
		Cut:            ev.Cut,
		ValueAvailable: asr.ValueAvailable,
	}, asr, nil
}

// evaluate runs the host fence around one commit. fn is called at most once, and
// a commit attempt that cannot be confirmed is reported as ErrUnknown.
func (l *Ledger) evaluate(ctx context.Context, r Request) (Receipt, error) {
	var (
		rc        Receipt
		called    bool
		attempted bool
	)
	err := l.host.WithinEvaluation(ctx, r, func(ev Evidence) error {
		if called {
			return errHostProtocol
		}
		called = true
		var e error
		rc, attempted, e = l.commit(ctx, r, ev)
		return e
	})
	if attempted {
		if err != nil {
			l.poisoned = true
			return Receipt{}, fmt.Errorf("%w: %w", ErrUnknown, err)
		}
		return rc, nil
	}
	if err != nil {
		return Receipt{}, err
	}
	if !called {
		return Receipt{}, ErrDenied
	}
	return rc, nil
}

// commit runs one exclusive transaction. attempted reports whether COMMIT was issued.
func (l *Ledger) commit(ctx context.Context, r Request, ev Evidence) (Receipt, bool, error) {
	if err := validEvidence(ev, r); err != nil {
		return Receipt{}, false, err
	}
	if err := l.hook("before_tx"); err != nil {
		return Receipt{}, false, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if _, err := l.conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return Receipt{}, false, storageError(err)
	}
	rec, rc, err := l.stage(ctx, r, ev)
	if err != nil {
		return Receipt{}, false, l.rollback(ctx, err)
	}
	if rec == nil {
		// An identical retry was resolved inside the transaction; nothing to commit.
		return rc, false, l.rollback(ctx, nil)
	}
	if _, err := l.conn.ExecContext(ctx, "COMMIT"); err != nil {
		l.poisoned = true
		return Receipt{}, true, fmt.Errorf("%w: commit: %w", ErrUnknown, err)
	}
	if err := l.hook("after_commit"); err != nil {
		l.poisoned = true
		return Receipt{}, true, fmt.Errorf("%w: %w", ErrUnknown, err)
	}
	return rec.Receipt, true, nil
}

// stage performs the checks and the single insert inside an open transaction.
func (l *Ledger) stage(ctx context.Context, r Request, ev Evidence) (*storedRecord, Receipt, error) {
	st, err := l.load(ctx)
	if err != nil {
		return nil, Receipt{}, err
	}
	if rc, found, err := st.lookup(r); found {
		return nil, rc, err
	}
	if err := st.precheck(r, l.max); err != nil {
		return nil, Receipt{}, err
	}
	rec, err := newRecord(st, r, ev)
	if err != nil {
		return nil, Receipt{}, err
	}
	if err := st.apply(rec); err != nil {
		return nil, Receipt{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return nil, Receipt{}, storageError(err)
	}
	if _, err := l.conn.ExecContext(ctx, "INSERT INTO ledger_records (seq, body) VALUES (?, ?)", rec.Sequence, string(body)); err != nil {
		return nil, Receipt{}, storageError(err)
	}
	if err := l.hook("after_write"); err != nil {
		return nil, Receipt{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	if err := l.hook("before_commit"); err != nil {
		return nil, Receipt{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return &rec, rec.Receipt, nil
}

// rollback ends an uncommitted transaction. A failed rollback poisons the handle and
// is added to the primary error rather than replacing it.
func (l *Ledger) rollback(ctx context.Context, primary error) error {
	if _, rbErr := l.conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); rbErr != nil {
		l.poisoned = true
		if primary == nil {
			return fmt.Errorf("%w: rollback: %w", ErrUnknown, rbErr)
		}
		return fmt.Errorf("%w; rollback failed: %w", primary, rbErr)
	}
	return primary
}

func (l *Ledger) hook(point string) error {
	if l.cut == nil {
		return nil
	}
	return l.cut(point)
}

func (l *Ledger) usable() error {
	if l.closed {
		return errClosed
	}
	if l.poisoned {
		return errPoisoned
	}
	return nil
}

func (l *Ledger) load(ctx context.Context) (*ledgerState, error) {
	return loadState(ctx, l.conn)
}

func loadState(ctx context.Context, conn *sql.Conn) (*ledgerState, error) {
	rows, err := conn.QueryContext(ctx, "SELECT seq, body FROM ledger_records ORDER BY seq")
	if err != nil {
		return nil, storageError(err)
	}
	defer rows.Close()
	st := newState()
	for rows.Next() {
		var seq int64
		var body string
		if err := rows.Scan(&seq, &body); err != nil {
			return nil, storageError(err)
		}
		if seq != int64(st.count+1) {
			return nil, errCorrupt
		}
		rec, err := decodeRecord(body)
		if err != nil {
			return nil, errCorrupt
		}
		if err := st.apply(rec); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, storageError(err)
	}
	return st, nil
}

func decodeRecord(body string) (storedRecord, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	var rec storedRecord
	if err := dec.Decode(&rec); err != nil {
		return storedRecord{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return storedRecord{}, errCorrupt
	}
	return rec, nil
}

// open opens the pinned connection, verifies durability settings, takes the exclusive
// lock, and validates or creates the schema before returning the replayed state.
func (l *Ledger) open(ctx context.Context, path string, created bool) (*ledgerState, error) {
	// The driver runs its own PRAGMAs during connection open, which take a shared lock.
	// A zero busy timeout in the DSN makes a competing owner fail at once instead of
	// waiting out the driver default of 5 seconds; the explicit pragma below verifies it.
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=0")
	if err != nil {
		return nil, storageError(err)
	}
	l.db = db
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, openError(err)
	}
	l.conn = conn
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
func (l *Ledger) release() error {
	var err error
	if l.conn != nil {
		err = l.conn.Close()
	}
	if l.db != nil {
		if cerr := l.db.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// createFile creates the database file only if absent. An existing file is never
// treated as new, so an empty or truncated ledger cannot be silently reset.
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
	if strings.Join(names, ",") != "ledger_meta,ledger_records" {
		return fmt.Errorf("%w: not a ledger database", ErrUnavailable)
	}
	meta := map[string]string{}
	rows, err := conn.QueryContext(ctx, "SELECT name, value FROM ledger_meta")
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
		return fmt.Errorf("%w: unsupported ledger format", ErrUnavailable)
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
