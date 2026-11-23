package delivery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/version2beta/perhap/sdk/go/ledger"
	"github.com/version2beta/perhap/sdk/go/rev"
)

var (
	errInvalidArgument = errors.New("delivery: invalid argument")
	errClosed          = fmt.Errorf("%w: store closed", ErrUnavailable)
	errPoisoned        = fmt.Errorf("%w: store poisoned by an unconfirmed commit; reopen", ErrUnknown)
	errCorrupt         = fmt.Errorf("%w: corrupt or inconsistent delivery record", ErrUnavailable)
)

const (
	storageFormat  = "perhap-delivery-store"
	storageVersion = "1"
	recordFormat   = 1
)

var schemaStatements = []string{
	`CREATE TABLE delivery_meta (name TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	`CREATE TABLE delivery_records (seq INTEGER PRIMARY KEY, body TEXT NOT NULL)`,
}

// storedRecord is one private, versioned, append-only admission record. It holds
// the whole immutable Publication; no field is interpreted or repaired on read.
type storedRecord struct {
	Format      int                `json:"format"`
	Sequence    int                `json:"sequence"`
	Publication ledger.Publication `json:"publication"`
}

// scope is the ordering domain for ordinals: one key within one generation.
type scope struct {
	key        rev.Key
	generation string
}

// state is the validated replay of every durable admission record.
type state struct {
	records []ledger.Publication
	byID    map[string]ledger.Publication
	heads   map[scope]ledger.Publication // latest admitted publication per scope
}

func newState() *state {
	return &state{
		records: []ledger.Publication{},
		byID:    map[string]ledger.Publication{},
		heads:   map[scope]ledger.Publication{},
	}
}

func scopeOf(p ledger.Publication) scope {
	return scope{key: p.Record.Key, generation: p.Generation}
}

// admissible checks the scope's next ordinal and predecessor. An ordinal at or below
// the head is occupied by a different publication ID (identity is checked first), so
// it conflicts. A missing or wrong predecessor is a gap and changes nothing.
func (st *state) admissible(p ledger.Publication) error {
	head := st.heads[scopeOf(p)]
	if p.Record.Revision <= head.Record.Revision {
		return ErrConflict
	}
	if p.Record.Revision != head.Record.Revision+1 || p.Record.Supersedes != head.Record.ID {
		return ErrGap
	}
	return nil
}

func (st *state) add(p ledger.Publication) {
	st.records = append(st.records, p)
	st.byID[p.Record.ID] = p
	st.heads[scopeOf(p)] = p
}

// checkPublication rejects publications this store cannot admit without interpreting
// them: a missing identity, generation or ordinal, or attribution to another producer.
func checkPublication(p ledger.Publication, producer string) error {
	if p.Record.ID == "" || p.Generation == "" || p.Record.Revision == 0 {
		return errInvalidArgument
	}
	if p.Record.Key.Producer != producer || p.Attribution != producer {
		return errInvalidArgument
	}
	return nil
}

func validOpenArgs(path, receiver, producer string, options Options) bool {
	return receiver != "" && producer != "" && options.MaxRecords > 0 && options.MaxBytes > 0 &&
		filepath.IsAbs(path) && filepath.Clean(path) == path && !strings.ContainsRune(path, '?')
}

// open opens the pinned connection, verifies durability settings, takes the exclusive
// lock, and validates or creates the schema and every record before returning.
func (s *Store) open(ctx context.Context, path string, created bool) error {
	// The driver runs its own PRAGMAs during connection open, which take a shared lock.
	// A zero busy timeout in the DSN makes a competing owner fail at once instead of
	// waiting out the driver default; the explicit pragma below verifies it.
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=0")
	if err != nil {
		return storageError(err)
	}
	s.db = db
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	conn, err := db.Conn(ctx)
	if err != nil {
		return openError(err)
	}
	s.conn = conn
	if err := setPragmas(ctx, conn); err != nil {
		return openError(err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return openError(err)
	}
	if err := initOrValidate(ctx, conn, created, s.receiver, s.producer); err != nil {
		return openError(err)
	}
	if _, err := loadState(ctx, conn, s.producer); err != nil {
		return openError(err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return openError(err)
	}
	return nil
}

// admitTx runs one exclusive transaction. Identical replays resolve to the original
// record even at capacity; any new record is inserted only after every check passes.
func (s *Store) admitTx(ctx context.Context, p ledger.Publication) (ledger.Publication, error) {
	if _, err := s.conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return ledger.Publication{}, storageError(err)
	}
	st, err := loadState(ctx, s.conn, s.producer)
	if err != nil {
		return ledger.Publication{}, s.rollback(ctx, err)
	}
	if stored, found := st.byID[p.Record.ID]; found {
		if stored != p {
			return ledger.Publication{}, s.rollback(ctx, ErrConflict)
		}
		return stored, s.rollback(ctx, nil)
	}
	if err := st.admissible(p); err != nil {
		return ledger.Publication{}, s.rollback(ctx, err)
	}
	if len(st.records) >= s.maxRecords {
		return ledger.Publication{}, s.rollback(ctx, ErrCapacity)
	}
	seq := len(st.records) + 1
	body, err := encodeRecord(seq, p)
	if err != nil {
		return ledger.Publication{}, s.rollback(ctx, err)
	}
	if len(body) > s.maxBytes {
		return ledger.Publication{}, s.rollback(ctx, ErrCapacity)
	}
	if _, err := s.conn.ExecContext(ctx, "INSERT INTO delivery_records (seq, body) VALUES (?, ?)", seq, body); err != nil {
		return ledger.Publication{}, s.rollback(ctx, storageError(err))
	}
	if err := s.hook("before_commit"); err != nil {
		return ledger.Publication{}, s.rollback(ctx, fmt.Errorf("%w: %w", ErrUnavailable, err))
	}
	if _, err := s.conn.ExecContext(ctx, "COMMIT"); err != nil {
		s.poisoned = true
		return ledger.Publication{}, fmt.Errorf("%w: commit: %w", ErrUnknown, err)
	}
	if err := s.hook("after_commit"); err != nil {
		s.poisoned = true
		return ledger.Publication{}, fmt.Errorf("%w: %w", ErrUnknown, err)
	}
	return p, nil
}

// rollback ends an uncommitted transaction. A failed rollback poisons the handle and
// is added to the primary error rather than replacing it.
func (s *Store) rollback(ctx context.Context, primary error) error {
	if _, err := s.conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); err != nil {
		s.poisoned = true
		if primary == nil {
			return fmt.Errorf("%w: rollback: %w", ErrUnknown, err)
		}
		return fmt.Errorf("%w; rollback failed: %w", primary, err)
	}
	return primary
}

func (s *Store) hook(point string) error {
	if s.cut == nil {
		return nil
	}
	return s.cut(point)
}

func (s *Store) usable() error {
	if s.closed {
		return errClosed
	}
	if s.poisoned {
		return errPoisoned
	}
	return nil
}

// release closes the pinned connection and the pool; closing the driver connection
// releases the SQLite exclusive lock.
func (s *Store) release() error {
	var err error
	if s.conn != nil {
		err = s.conn.Close()
	}
	if s.db != nil {
		if cerr := s.db.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

func encodeRecord(seq int, p ledger.Publication) (string, error) {
	body, err := json.Marshal(storedRecord{Format: recordFormat, Sequence: seq, Publication: p})
	if err != nil {
		return "", storageError(err)
	}
	return string(body), nil
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

// loadState replays every record in sequence order and validates each one against the
// state built so far. Any inconsistency is corruption, never a reset or a repair.
func loadState(ctx context.Context, conn *sql.Conn, producer string) (*state, error) {
	rows, err := conn.QueryContext(ctx, "SELECT seq, body FROM delivery_records ORDER BY seq")
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
		rec, err := decodeRecord(body)
		if err != nil {
			return nil, errCorrupt
		}
		p := rec.Publication
		if seq != int64(len(st.records)+1) || rec.Sequence != len(st.records)+1 || rec.Format != recordFormat {
			return nil, errCorrupt
		}
		if checkPublication(p, producer) != nil || st.admissible(p) != nil {
			return nil, errCorrupt
		}
		if _, dup := st.byID[p.Record.ID]; dup {
			return nil, errCorrupt
		}
		st.add(p)
	}
	if err := rows.Err(); err != nil {
		return nil, storageError(err)
	}
	return st, nil
}

// createFile creates the database file only if absent. An existing file is never
// treated as new, so an empty or truncated store cannot be silently reset.
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

// initOrValidate creates an absent schema in a newly created file, or verifies that an
// existing store is this format, pinned to this receiver and producer, and intact.
func initOrValidate(ctx context.Context, conn *sql.Conn, created bool, receiver, producer string) error {
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
		_, err := conn.ExecContext(ctx,
			"INSERT INTO delivery_meta (name, value) VALUES ('format', ?), ('version', ?), ('receiver', ?), ('producer', ?)",
			storageFormat, storageVersion, receiver, producer)
		return err
	}
	if strings.Join(names, ",") != "delivery_meta,delivery_records" {
		return fmt.Errorf("%w: not a delivery store", ErrUnavailable)
	}
	meta, err := readMeta(ctx, conn)
	if err != nil {
		return err
	}
	if len(meta) != 4 || meta["format"] != storageFormat || meta["version"] != storageVersion {
		return fmt.Errorf("%w: unsupported delivery store format", ErrUnavailable)
	}
	if meta["receiver"] != receiver || meta["producer"] != producer {
		return fmt.Errorf("%w: store is pinned to a different receiver or producer", ErrUnavailable)
	}
	return integrityCheck(ctx, conn)
}

func readMeta(ctx context.Context, conn *sql.Conn) (map[string]string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name, value FROM delivery_meta")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	meta := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return nil, err
		}
		meta[name] = value
	}
	return meta, rows.Err()
}

func integrityCheck(ctx context.Context, conn *sql.Conn) error {
	rows, err := conn.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	var results []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return err
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
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
