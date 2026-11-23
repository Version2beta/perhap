package submission

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSubmissionCommittedRecoveryIntegrity(t *testing.T) {
	t.Run("truncated-committed-file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "s.db")
		h := &testHost{}
		o := open(t, path, h, limits)
		submit(t, o, input("1"))
		o.Close()
		if e := os.Truncate(path, 4096); e != nil {
			t.Fatal(e)
		}
		bad, e := Open(bg, path, h, limits)
		if bad != nil || e == nil || (!errors.Is(e, ErrIntegrity) && !errors.Is(e, ErrUnavailable)) {
			t.Fatal("truncated committed file became fresh authoritative state")
		}
	})
	t.Run("stale-backup-inventory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "s.db")
		h := &testHost{}
		o := open(t, path, h, limits)
		o.Close()
		backup, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		o = open(t, path, h, limits)
		in := input("1")
		submit(t, o, in)
		o.Close()
		h.minimum = map[string]uint64{in.Event.ID: 2}
		if e = os.WriteFile(path, backup, 0600); e != nil {
			t.Fatal(e)
		}
		bad, e := Open(bg, path, h, limits)
		if bad != nil || !errors.Is(e, ErrIntegrity) {
			t.Fatal("independent inventory did not reject rollback")
		}
	})
}
