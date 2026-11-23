package observation

import (
	"context"
	"encoding/gob"
	"errors"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRecoveryOutputCrash(t *testing.T) {
	if phase := os.Getenv("PERHAP_OUTPUT_PHASE"); phase != "" {
		key, e := os.ReadFile(os.Getenv("PERHAP_OUTPUT_KEYFILE"))
		if e != nil {
			t.Fatal(e)
		}
		p := &testProofs{key: key, now: 1}
		if phase == "mint" {
			p.hook = func(c Claims) {
				if c.Purpose == "barrier" {
					os.Exit(78)
				}
			}
		}
		j, e := viewjournal.Open(ctx, os.Getenv("PERHAP_OUTPUT_DB"), &journalHost{}, viewjournal.Limits{MaxEvaluations: 100, MaxPins: 3, MaxBatch: 2, MaxBytes: 2 << 20})
		if e != nil {
			t.Fatal(e)
		}
		m, e := NewRecovery(ctx, j, NewGate(&authority{}, binding), p, "output", recoveryLimits)
		if e != nil {
			t.Fatal(e)
		}
		s, e := m.Open(ctx, binding, func(_ context.Context, f RecoveryFrame) error {
			if f.Kind == phase {
				file, e := os.OpenFile(os.Getenv("PERHAP_OUTPUT_FILE"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if e != nil {
					return e
				}
				if e = gob.NewEncoder(file).Encode(f); e != nil {
					return e
				}
				if e = file.Sync(); e != nil {
					return e
				}
				file.Close()
				os.Exit(78)
			}
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
		if e = s.Step(); e != nil {
			t.Fatal(e)
		}
		t.Fatal("output crash did not fire")
	}
	for _, phase := range []string{"snapshot", "mint", "barrier"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "v.db")
			keyfile := filepath.Join(dir, "test-key")
			output := filepath.Join(dir, "recipient.gob")
			h := &journalHost{}
			jl := viewjournal.Limits{MaxEvaluations: 100, MaxPins: 3, MaxBatch: 2, MaxBytes: 2 << 20}
			j, e := viewjournal.Open(ctx, path, h, jl)
			if e != nil {
				t.Fatal(e)
			}
			for n := 1; n <= 43; n++ {
				accept(t, j, h, n)
			}
			p := proofAuthority(t)
			if e = os.WriteFile(keyfile, p.key, 0600); e != nil {
				t.Fatal(e)
			}
			m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "parent")
			_, oldToken := snapshotToken(t, m)
			m.Close()
			j.Close()
			command := exec.Command(os.Args[0], "-test.run=^TestRecoveryOutputCrash$")
			command.Env = append(os.Environ(), "PERHAP_OUTPUT_PHASE="+phase, "PERHAP_OUTPUT_DB="+path, "PERHAP_OUTPUT_KEYFILE="+keyfile, "PERHAP_OUTPUT_FILE="+output)
			out, e := command.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(e, &exit) || exit.ExitCode() != 78 {
				t.Fatalf("child output cut: %v %s", e, out)
			}
			key, e := os.ReadFile(keyfile)
			if e != nil {
				t.Fatal(e)
			}
			root := &testProofs{key: key, now: 1}
			var released RecoveryFrame
			if phase == "mint" {
				if _, e = os.Stat(output); !errors.Is(e, os.ErrNotExist) {
					t.Fatal("issuer cut invented recipient release")
				}
			} else {
				file, e := os.Open(output)
				if e != nil {
					t.Fatal(e)
				}
				e = gob.NewDecoder(file).Decode(&released)
				file.Close()
				if e != nil {
					t.Fatal(e)
				}
				c, e := root.Verify(ctx, released.Proof)
				if e != nil || c.Purpose != phase || c.Record.Revision != 43 {
					t.Fatal("released proof not authentic")
				}
			}
			j, e = viewjournal.Open(ctx, path, h, jl)
			if e != nil {
				t.Fatal(e)
			}
			defer j.Close()
			fresh := recoveryManager(t, j, NewGate(&authority{}, binding), root, "output")
			noPins(t, j)
			accept(t, j, h, 44)
			accept(t, j, h, 45)
			if phase == "snapshot" {
				s, e := fresh.Resume(ctx, binding, released.Proof, func(context.Context, RecoveryFrame) error { return nil })
				if !errors.Is(e, ErrCannotResume) || s != nil {
					t.Fatal("snapshot proof manufactured completed hydration")
				}
			}
			token := oldToken
			if phase == "barrier" {
				token = released.Proof
			}
			var frames []RecoveryFrame
			s, e := fresh.Resume(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { frames = append(frames, f); return nil })
			if e != nil || s == nil {
				t.Fatal(e)
			}
			if e = s.Step(); e != nil {
				t.Fatal(e)
			}
			if len(frames) != 3 || frames[0].Record.Revision != 44 || frames[1].Record.Revision != 45 || frames[2].Kind != "barrier" || frames[2].Record.Revision != 45 {
				t.Fatal("crash recovery manufactured/skipped delivery")
			}
			fresh.Close()
			noPins(t, j)
		})
	}
}
