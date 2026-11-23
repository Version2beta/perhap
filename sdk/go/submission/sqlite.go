package submission

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/google/uuid"
	sqlite3 "github.com/mattn/go-sqlite3"
)

// This file is the gateway-only SQLite authority. It is separate from JetStream and from
// every other store; its file is never shared with another owner.

const (
	storageFormat  = "perhap-submission-store"
	storageVersion = "1"
)

var schemaStatements = []string{
	`CREATE TABLE submission_meta (name TEXT PRIMARY KEY, value TEXT NOT NULL)`,
	`CREATE TABLE submission_records (seq INTEGER PRIMARY KEY, body TEXT NOT NULL)`,
}

// createFile creates the database file only when it is absent. An existing file is
// never treated as new, so an empty or corrupt store cannot be silently reset.
func createFile(path string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, ErrUnavailable
	}
	if err := f.Close(); err != nil {
		return false, ErrUnavailable
	}
	return true, nil
}

// openDB pins one exclusive connection, verifies durability settings, takes the lock,
// and validates or creates the schema and replays every record before returning.
func (o *Owner) openDB(ctx context.Context, path string, created bool) error {
	// A zero busy timeout makes a competing owner fail at once; the pragma below verifies it.
	db, err := sql.Open("sqlite3", path+"?_busy_timeout=0")
	if err != nil {
		return ErrUnavailable
	}
	o.db = db
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)
	conn, err := db.Conn(ctx)
	if err != nil {
		return openError(err)
	}
	o.conn = conn
	if err := setPragmas(ctx, conn); err != nil {
		return openError(err)
	}
	if _, err := conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return openError(err)
	}
	st, err := initOrLoad(ctx, conn, created)
	if err != nil {
		_, _ = conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
		return openError(err)
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return openError(err)
	}
	o.st = st
	return nil
}

// releaseDB closes the pinned connection and pool; closing the connection releases the lock.
func (o *Owner) releaseDB() error {
	var err error
	if o.conn != nil {
		err = o.conn.Close()
		o.conn = nil
	}
	if o.db != nil {
		if cerr := o.db.Close(); cerr != nil && err == nil {
			err = cerr
		}
		o.db = nil
	}
	return err
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
			return ErrUnavailable
		}
	}
	return nil
}

// initOrLoad creates a new store only for an absent file. Any other file must have
// exactly the expected schema, metadata and integrity.
func initOrLoad(ctx context.Context, conn *sql.Conn, created bool) (*state, error) {
	names, err := tableNames(ctx, conn)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 && created {
		generation := uuid.NewString()
		for _, stmt := range schemaStatements {
			if _, err := conn.ExecContext(ctx, stmt); err != nil {
				return nil, ErrUnavailable
			}
		}
		if _, err := conn.ExecContext(ctx, "INSERT INTO submission_meta (name, value) VALUES ('format', ?), ('version', ?), ('generation', ?)", storageFormat, storageVersion, generation); err != nil {
			return nil, ErrUnavailable
		}
		return loadState(ctx, conn, generation)
	}
	if strings.Join(names, ",") != "submission_meta,submission_records" {
		return nil, ErrIntegrity
	}
	generation, err := readMeta(ctx, conn)
	if err != nil {
		return nil, err
	}
	if err := checkIntegrity(ctx, conn); err != nil {
		return nil, err
	}
	return loadState(ctx, conn, generation)
}

func readMeta(ctx context.Context, conn *sql.Conn) (string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name, value FROM submission_meta")
	if err != nil {
		return "", ErrUnavailable
	}
	defer rows.Close()
	meta := map[string]string{}
	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			return "", ErrUnavailable
		}
		meta[name] = value
	}
	if err := rows.Err(); err != nil {
		return "", ErrUnavailable
	}
	u, err := uuid.Parse(meta["generation"])
	if len(meta) != 3 || meta["format"] != storageFormat || meta["version"] != storageVersion || err != nil || u.String() != meta["generation"] {
		return "", ErrIntegrity
	}
	return meta["generation"], nil
}

func checkIntegrity(ctx context.Context, conn *sql.Conn) error {
	rows, err := conn.QueryContext(ctx, "PRAGMA integrity_check")
	if err != nil {
		return ErrUnavailable
	}
	defer rows.Close()
	var results []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return ErrUnavailable
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return ErrUnavailable
	}
	if len(results) != 1 || results[0] != "ok" {
		return ErrIntegrity
	}
	return nil
}

func tableNames(ctx context.Context, conn *sql.Conn) ([]string, error) {
	rows, err := conn.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name")
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, ErrUnavailable
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}
	return names, nil
}

// loadState replays every record in sequence order. Any inconsistency is corruption;
// the store is never reset or repaired.
func loadState(ctx context.Context, conn *sql.Conn, generation string) (*state, error) {
	rows, err := conn.QueryContext(ctx, "SELECT seq, body FROM submission_records ORDER BY seq")
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	st := newState(generation)
	for rows.Next() {
		var seq int64
		var body string
		if err := rows.Scan(&seq, &body); err != nil {
			return nil, ErrUnavailable
		}
		rec, err := decodeRecord(body)
		if err != nil || int64(rec.Sequence) != seq {
			return nil, ErrIntegrity
		}
		if err := st.check(rec); err != nil {
			return nil, ErrIntegrity
		}
		st.fold(rec, len(body))
	}
	if err := rows.Err(); err != nil {
		return nil, ErrUnavailable
	}
	return st, nil
}

func decodeRecord(body string) (record, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.DisallowUnknownFields()
	var rec record
	if err := dec.Decode(&rec); err != nil {
		return record{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return record{}, errRecord
	}
	return rec, nil
}

// openError maps every open-time failure onto a public category without exposing
// driver text. A file that is not a SQLite database is integrity failure.
func openError(err error) error {
	if errors.Is(err, ErrIntegrity) {
		return ErrIntegrity
	}
	if errors.Is(err, ErrUnavailable) {
		return ErrUnavailable
	}
	var se sqlite3.Error
	if errors.As(err, &se) && (se.Code == sqlite3.ErrNotADB || se.Code == sqlite3.ErrCorrupt) {
		return ErrIntegrity
	}
	return ErrUnavailable
}

// commit appends one record that check has accepted. Failure before COMMIT is a known
// rollback. Failure after COMMIT was issued is an unknown outcome that poisons the handle.
func (o *Owner) commit(ctx context.Context, stage string, rec record) error {
	if err := o.usable(); err != nil {
		return err
	}
	rec.Format = recordFormat
	rec.Sequence = o.st.records + 1
	if err := o.st.check(rec); err != nil {
		return ErrUnavailable
	}
	body, err := json.Marshal(rec)
	if err != nil {
		return ErrUnavailable
	}
	if err := o.st.fits(len(body), o.limits); err != nil {
		return err
	}
	if err := o.hook(stage + ":before_tx"); err != nil {
		return ErrUnavailable
	}
	if _, err := o.conn.ExecContext(ctx, "BEGIN EXCLUSIVE"); err != nil {
		return ErrUnavailable
	}
	if _, err := o.conn.ExecContext(ctx, "INSERT INTO submission_records (seq, body) VALUES (?, ?)", rec.Sequence, string(body)); err != nil {
		return o.rollback(ctx, ErrUnavailable)
	}
	if err := o.hook(stage + ":before_commit"); err != nil {
		return o.rollback(ctx, ErrUnavailable)
	}
	if _, err := o.conn.ExecContext(context.WithoutCancel(ctx), "COMMIT"); err != nil {
		o.poisoned = true
		return ErrUnknown
	}
	o.st.fold(rec, len(body))
	if err := o.hook(stage + ":after_commit"); err != nil {
		o.poisoned = true
		return ErrUnknown
	}
	return nil
}

// rollback ends an uncommitted transaction. If rollback itself fails, the state is unknown.
func (o *Owner) rollback(ctx context.Context, primary error) error {
	if _, err := o.conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK"); err != nil {
		o.poisoned = true
		return ErrUnknown
	}
	return primary
}

func (o *Owner) hook(point string) error {
	if o.limits.Cut == nil {
		return nil
	}
	return o.limits.Cut(point)
}

func (o *Owner) usable() error {
	if o.poisoned {
		return ErrUnknown
	}
	return nil
}
