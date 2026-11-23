package observation

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/version2beta/perhap/sdk/go/viewjournal"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// Real test-only authenticated encryption. No production format or trust root.
type sealedClaims struct {
	Claims  Claims
	Expires int64
}
type testProofs struct {
	mu    sync.Mutex
	key   []byte
	now   int64
	floor uint64
	hook  func(Claims)
}

func (p *testProofs) Issue(_ context.Context, c Claims) (string, error) {
	p.mu.Lock()
	now := p.now
	hook := p.hook
	p.mu.Unlock()
	if hook != nil {
		hook(c)
	}
	block, e := aes.NewCipher(p.key)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return "", e
	}
	data, e := json.Marshal(sealedClaims{Claims: c, Expires: now + 10})
	if e != nil {
		return "", e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return string(g.Seal(nonce, nonce, data, nil)), nil
}
func (p *testProofs) Verify(_ context.Context, token string) (Claims, error) {
	block, e := aes.NewCipher(p.key)
	if e != nil {
		return Claims{}, e
	}
	g, e := cipher.NewGCM(block)
	if e != nil {
		return Claims{}, e
	}
	data := []byte(token)
	if len(data) < g.NonceSize() {
		return Claims{}, ErrCannotResume
	}
	plain, e := g.Open(nil, data[:g.NonceSize()], data[g.NonceSize():], nil)
	if e != nil {
		return Claims{}, ErrCannotResume
	}
	var c sealedClaims
	if json.Unmarshal(plain, &c) != nil {
		return Claims{}, ErrCannotResume
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if c.Expires <= p.now || c.Claims.Record.Revision < p.floor {
		return Claims{}, ErrCannotResume
	}
	return c.Claims, nil
}
func proofAuthority(t *testing.T) *testProofs {
	t.Helper()
	key := make([]byte, 32)
	if _, e := rand.Read(key); e != nil {
		t.Fatal(e)
	}
	return &testProofs{key: key, now: 1}
}

var recoveryLimits = RecoveryLimits{Page: 2, MaxBytes: 64 << 10, MaxTokenBytes: 8 << 10, CallbackTimeout: time.Second, CleanupTimeout: time.Second}

func recoveryManager(t *testing.T, j RecoverySource, g *Gate, p ProofAuthority, owner string) *Recovery {
	t.Helper()
	m, e := NewRecovery(ctx, j, g, p, owner, recoveryLimits)
	if e != nil || m == nil {
		t.Fatalf("NewRecovery: %v", e)
	}
	t.Cleanup(func() { m.Close() })
	return m
}
func snapshotToken(t *testing.T, m *Recovery) (*Recovered, string) {
	t.Helper()
	var token string
	s, e := m.Open(ctx, binding, func(_ context.Context, f RecoveryFrame) error {
		if f.Kind == "barrier" {
			token = f.Proof
		}
		return nil
	})
	if e != nil || s == nil {
		t.Fatalf("open %v", e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if token == "" {
		t.Fatal("no barrier cursor")
	}
	return s, token
}
func noPins(t *testing.T, j *viewjournal.Journal) {
	t.Helper()
	ps, e := j.Plans(ctx)
	if e != nil || len(ps) != 0 {
		t.Fatalf("pins remain %d %v", len(ps), e)
	}
}
func TestRetainedPlannerAndBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.db")
	h := &journalHost{}
	jl := viewjournal.Limits{MaxEvaluations: 100, MaxPins: 3, MaxBatch: 2, MaxBytes: 16000}
	j, e := viewjournal.Open(ctx, path, h, jl)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	for n := 1; n <= 3; n++ {
		accept(t, j, h, n)
	}
	id := "obs/slot/retained/1"
	p, e := j.Acquire(ctx, id, v, 1)
	if e != nil || p.Plan.ID != id || p.Anchor.Revision != 1 || p.Plan.Snapshot.Revision != 3 {
		t.Fatalf("retained plan %v %+v", e, p)
	}
	if _, e = j.Read(ctx, id, 1, 2); !errors.Is(e, viewjournal.ErrStale) {
		t.Fatal("original API lower bound changed")
	}
	b, e := j.ReadRetained(ctx, id, 1, 2)
	if e != nil || b.Through != 3 || len(b.Records) != 2 || b.Records[0].Revision != 2 || b.Records[1].Revision != 3 {
		t.Fatalf("retained successors %v %+v", e, b)
	}
	accept(t, j, h, 4)
	again, e := j.Acquire(ctx, id, v, 1)
	if e != nil || !reflect.DeepEqual(p, again) {
		t.Fatal("acquisition retry changed")
	}
	j.Close()
	j, e = viewjournal.Open(ctx, path, h, jl)
	if e != nil {
		t.Fatalf("existing prepare format rejected on reopen: %v", e)
	}
	defer j.Close()
	refused := false
	for n := 5; n <= 100; n++ {
		heads, e := j.Heads(ctx)
		if e != nil {
			t.Fatal(e)
		}
		old := heads[0]
		h.value = int64(n)
		_, e = j.Accept(ctx, viewjournal.Request{ID: fmt.Sprint("budget", n), View: v, EvaluationID: fmt.Sprint("g", n), ExpectedCheckpoint: old.Checkpoint, ExpectedRevision: old.Revision})
		if errors.Is(e, viewjournal.ErrCapacity) {
			refused = true
			break
		}
		if e != nil {
			t.Fatal(e)
		}
	}
	if !refused {
		t.Fatal("capacity was unbounded")
	}
	if e = j.Release(ctx, id); e != nil {
		t.Fatalf("cleanup reservation consumed: %v", e)
	}
	noPins(t, j)
}
func TestRecoverySnapshotAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.db")
	h := &journalHost{}
	jl := viewjournal.Limits{MaxEvaluations: 100, MaxPins: 3, MaxBatch: 2, MaxBytes: 2 << 20}
	j, e := viewjournal.Open(ctx, path, h, jl)
	if e != nil {
		t.Fatal(e)
	}
	for n := 1; n <= 40; n++ {
		accept(t, j, h, n)
	}
	p := proofAuthority(t)
	m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
	var frames []RecoveryFrame
	s, e := m.Open(ctx, binding, func(_ context.Context, f RecoveryFrame) error { frames = append(frames, f); return nil })
	if e != nil || s == nil {
		t.Fatalf("open %v", e)
	}
	accept(t, j, h, 41)
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(frames) != 3 || frames[0].Kind != "snapshot" || frames[0].Record.Revision != 40 || frames[1].Kind != "revision" || frames[1].Record.Revision != 41 || frames[2].Kind != "barrier" || frames[2].Record.Revision != 41 {
		t.Fatalf("O03: %+v", frames)
	}
	for _, i := range []int{0, 2} {
		c, e := p.Verify(ctx, frames[i].Proof)
		if e != nil || c.Purpose != frames[i].Kind || c.Binding != binding || !reflect.DeepEqual(c.Record, frames[i].Record) {
			t.Fatalf("unbound proof %v", e)
		}
	}
	token := frames[2].Proof
	m.Close()
	accept(t, j, h, 42)
	accept(t, j, h, 43)
	j.Close()
	j, e = viewjournal.Open(ctx, path, h, jl)
	if e != nil {
		t.Fatal(e)
	}
	defer j.Close()
	p2 := &testProofs{key: append([]byte(nil), p.key...), now: 1}
	m2 := recoveryManager(t, j, NewGate(&authority{}, binding), p2, "slot")
	frames = nil
	s, e = m2.Resume(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { frames = append(frames, f); return nil })
	if e != nil || s == nil {
		t.Fatalf("resume after SQLite reopen: %v", e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(frames) != 3 || frames[0].Kind != "revision" || frames[0].Record.Revision != 42 || frames[1].Record.Revision != 43 || frames[2].Kind != "barrier" || frames[2].Record.Revision != 43 {
		t.Fatalf("resume skipped/replayed: %+v", frames)
	}
	s.Close()
	noPins(t, j)
}
func TestRecoverySubstitutionAndReplacement(t *testing.T) {
	j, h := journal(t, 100)
	for n := 1; n <= 40; n++ {
		accept(t, j, h, n)
	}
	p := proofAuthority(t)
	m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
	_, token := snapshotToken(t, m)
	claim, e := p.Verify(ctx, token)
	if e != nil {
		t.Fatal(e)
	}
	changes := []func(*Claims){func(c *Claims) { c.Binding.Observer = "bob" }, func(c *Claims) { c.Binding.View.Context = "other" }, func(c *Claims) { c.Binding.View.Projection = "other" }, func(c *Claims) { c.Binding.View.Parameters = "other" }, func(c *Claims) { c.Binding.View.Generation = "other" }, func(c *Claims) { c.Binding.View.AuthorizedView = "other" }, func(c *Claims) { c.Binding.View.Schema = "other" }, func(c *Claims) { c.Binding.View.Model = "other" }, func(c *Claims) { c.Binding.Epoch = "other" }, func(c *Claims) { c.Binding.Descriptor = "other" }, func(c *Claims) { c.Purpose = "snapshot" }}
	tokens := []string{"", token + "tampered"}
	for _, change := range changes {
		c := claim
		change(&c)
		tok, e := p.Issue(ctx, c)
		if e != nil {
			t.Fatal(e)
		}
		tokens = append(tokens, tok)
	}
	for _, tok := range tokens {
		var fs []RecoveryFrame
		s, e := m.Resume(ctx, binding, tok, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil })
		if !errors.Is(e, ErrCannotResume) || s != nil || len(fs) != 1 || fs[0].Kind != "cannot-resume" || fs[0].Proof != "" || fs[0].Record.Revision != 0 {
			t.Fatalf("O10 substitution leaked: %v %+v", e, fs)
		}
		noPins(t, j)
	}
	p.now = 100
	var fs []RecoveryFrame
	if _, e = m.Resume(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil }); !errors.Is(e, ErrCannotResume) {
		t.Fatal(e)
	}
	p.now = 1
	p.floor = 41
	if _, e = m.Resume(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil }); !errors.Is(e, ErrCannotResume) {
		t.Fatal(e)
	}
	for n := 41; n <= 90; n++ {
		accept(t, j, h, n)
	}
	fs = nil
	s, e := m.Replace(ctx, binding, func(_ context.Context, f RecoveryFrame) error { fs = append(fs, f); return nil })
	if e != nil || s == nil {
		t.Fatal(e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	if len(fs) != 2 || fs[0].Kind != "replacement" || fs[0].Record.Revision != 90 || fs[1].Kind != "barrier" {
		t.Fatalf("O09: %+v", fs)
	}
	c, e := p.Verify(ctx, fs[0].Proof)
	if e != nil || c.Purpose != "replacement" {
		t.Fatal("replacement not certified")
	}
	s.Close()
	noPins(t, j)
}
func TestRecoveryReleaseOrders(t *testing.T) {
	for _, order := range []string{"revoke-first", "release-first"} {
		t.Run(order, func(t *testing.T) {
			j, h := journal(t, 10)
			accept(t, j, h, 1)
			p := proofAuthority(t)
			g := NewGate(&authority{}, binding)
			m := recoveryManager(t, j, g, p, "slot")
			entered, finish := make(chan struct{}), make(chan struct{})
			calls := 0
			revoked := make(chan struct{})
			if order == "revoke-first" {
				p.hook = func(c Claims) {
					if c.Purpose == "snapshot" {
						close(entered)
						<-finish
					}
				}
			} else {
				p.hook = func(c Claims) {
					if c.Purpose == "barrier" {
						<-revoked
					}
				}
			}
			s, e := m.Open(ctx, binding, func(c context.Context, f RecoveryFrame) error {
				calls++
				if order == "release-first" {
					close(entered)
					<-finish
				}
				return nil
			})
			if e != nil || s == nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() { done <- s.Step() }()
			<-entered
			if order == "revoke-first" {
				g.Revoke()
				close(finish)
			} else {
				go func() { g.Revoke(); close(revoked) }()
				close(finish)
				<-revoked
			}
			e = <-done
			if !errors.Is(e, ErrDenied) {
				t.Fatalf("order %v", e)
			}
			want := 0
			if order == "release-first" {
				want = 1
			}
			if calls != want {
				t.Fatalf("A03 cursor/payload escape: %d", calls)
			}
			s.Close()
			noPins(t, j)
		})
	}
}
func TestRecoveryReconnectAndCancellation(t *testing.T) {
	j, h := journal(t, 100)
	for n := 1; n <= 43; n++ {
		accept(t, j, h, n)
	}
	p := proofAuthority(t)
	m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
	entered := make(chan struct{})
	barriers := 0
	var token string
	s, e := m.Open(ctx, binding, func(c context.Context, f RecoveryFrame) error {
		if f.Kind == "barrier" {
			barriers++
			if barriers == 1 {
				token = f.Proof
			} else {
				close(entered)
				<-c.Done()
				return c.Err()
			}
		}
		return nil
	})
	if e != nil || s == nil {
		t.Fatal(e)
	}
	if e = s.Step(); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- s.Step() }()
	<-entered
	accept(t, j, h, 44)
	accept(t, j, h, 45)
	var frames []RecoveryFrame
	next, e := m.Resume(ctx, binding, token, func(_ context.Context, f RecoveryFrame) error { frames = append(frames, f); return nil })
	if e != nil || next == nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, ErrUnknown) {
		t.Fatalf("old completion %v", e)
	}
	if e = s.Step(); !errors.Is(e, ErrClosed) {
		t.Fatal("retired session revived")
	}
	if e = next.Step(); e != nil {
		t.Fatal(e)
	}
	if len(frames) != 3 || frames[0].Record.Revision != 44 || frames[1].Record.Revision != 45 || frames[2].Kind != "barrier" || frames[2].Record.Revision != 45 {
		t.Fatalf("O06 reconnect %+v", frames)
	}
	next.Close()
	noPins(t, j)
	c, cancel := context.WithCancel(ctx)
	next, e = m.Open(c, binding, func(context.Context, RecoveryFrame) error { t.Fatal("canceled disclosure"); return nil })
	if e != nil {
		t.Fatal(e)
	}
	cancel()
	if e = next.Step(); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	noPins(t, j)
}

type recoveryWrapped struct {
	RecoverySource
	failRelease bool
	badAnchor   bool
	badRead     string
}

func (w *recoveryWrapped) Acquire(c context.Context, id string, v viewjournal.Identity, n uint64) (viewjournal.Retained, error) {
	p, e := w.RecoverySource.Acquire(c, id, v, n)
	if w.badAnchor {
		p.Anchor.Public.Evidence = "corrupt"
	}
	return p, e
}
func (w *recoveryWrapped) Release(c context.Context, id string) error {
	if w.failRelease {
		return errors.New("private diagnostic")
	}
	return w.RecoverySource.Release(c, id)
}
func (w *recoveryWrapped) ReadRetained(c context.Context, id string, n uint64, max int) (viewjournal.Batch, error) {
	b, e := w.RecoverySource.ReadRetained(c, id, n, max)
	if w.badRead == "interrupt" {
		return b, errors.New("private scan")
	}
	if w.badRead == "gap" && len(b.Records) > 0 {
		b.Records[0].Previous = 999
	}
	if w.badRead == "capacity" && len(b.Records) > 0 {
		b.Records[0].Public.Evidence = strings.Repeat("allowed-public-evidence", 10000)
	}
	return b, e
}
func TestRecoveryFailureCapacityAndCleanup(t *testing.T) {
	for _, mode := range []string{"interrupt", "gap", "capacity", "cleanup", "anchor", "denied"} {
		t.Run(mode, func(t *testing.T) {
			j, h := journal(t, 10)
			accept(t, j, h, 1)
			p := proofAuthority(t)
			g := NewGate(&authority{}, binding)
			w := &recoveryWrapped{RecoverySource: j}
			m := recoveryManager(t, w, g, p, "slot")
			_, token := snapshotToken(t, m)
			accept(t, j, h, 2)
			var fs []RecoveryFrame
			if mode == "anchor" {
				w.badAnchor = true
			}
			if mode == "denied" {
				g.Revoke()
			}
			s, e := m.Resume(ctx, binding, token, func(context.Context, RecoveryFrame) error { return nil })
			if mode == "anchor" || mode == "denied" {
				if e == nil || s != nil {
					t.Fatal("invalid resumed")
				}
				m.Close()
				noPins(t, j)
				return
			}
			if e != nil || s == nil {
				t.Fatal(e)
			}
			s.Close()
			s, e = m.Open(ctx, binding, func(_ context.Context, f RecoveryFrame) error {
				fs = append(fs, f)
				if mode == "cleanup" {
					return errors.New("recipient possibly released")
				}
				return nil
			})
			if e != nil || s == nil {
				t.Fatal(e)
			}
			accept(t, j, h, 3)
			w.badRead = mode
			if mode == "cleanup" {
				w.failRelease = true
			}
			e = s.Step()
			if e == nil {
				t.Fatal("failure certified")
			}
			for _, f := range fs {
				if f.Kind == "barrier" || f.Kind == "revision" {
					t.Fatalf("partial/failed page released %+v", fs)
				}
			}
			if mode == "cleanup" {
				if !errors.Is(e, ErrUnknown) || !errors.Is(e, ErrCleanup) {
					t.Fatalf("lost primary/cleanup: %v", e)
				}
				if e = m.Close(); !errors.Is(e, ErrCleanup) {
					t.Fatal("cleanup silently forgotten")
				}
				w.failRelease = false
			}
			if e = m.Close(); e != nil {
				t.Fatal(e)
			}
			noPins(t, j)
		})
	}
}
func TestRecoveryOwnedCrash(t *testing.T) {
	if cut := os.Getenv("PERHAP_RECOVERY_CUT"); cut != "" {
		path := os.Getenv("PERHAP_RECOVERY_DB")
		h := &journalHost{}
		jl := viewjournal.Limits{MaxEvaluations: 100, MaxPins: 3, MaxBatch: 2, MaxBytes: 2 << 20, Cut: func(at string) error {
			if at == cut {
				os.Exit(77)
			}
			return nil
		}}
		j, e := viewjournal.Open(ctx, path, h, jl)
		if e != nil {
			t.Fatal(e)
		}
		_, e = j.Acquire(ctx, "obs/slot/crash/40", v, 40)
		t.Fatalf("crash hook did not fire: %v", e)
	}
	for _, cut := range []string{"before_commit", "after_commit"} {
		t.Run(cut, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "v.db")
			h := &journalHost{}
			jl := viewjournal.Limits{MaxEvaluations: 100, MaxPins: 3, MaxBatch: 2, MaxBytes: 2 << 20}
			j, e := viewjournal.Open(ctx, path, h, jl)
			if e != nil {
				t.Fatal(e)
			}
			for n := 1; n <= 40; n++ {
				accept(t, j, h, n)
			}
			p := proofAuthority(t)
			m := recoveryManager(t, j, NewGate(&authority{}, binding), p, "slot")
			_, token := snapshotToken(t, m)
			m.Close()
			foreign, e := j.Prepare(ctx, v)
			if e != nil {
				t.Fatal(e)
			}
			j.Close()
			command := exec.Command(os.Args[0], "-test.run=^TestRecoveryOwnedCrash$")
			command.Env = append(os.Environ(), "PERHAP_RECOVERY_CUT="+cut, "PERHAP_RECOVERY_DB="+path)
			out, e := command.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(e, &exit) || exit.ExitCode() != 77 {
				t.Fatalf("real child cut: %v %s", e, out)
			}
			j, e = viewjournal.Open(ctx, path, h, jl)
			if e != nil {
				t.Fatal(e)
			}
			defer j.Close()
			ps, e := j.Plans(ctx)
			want := 1
			if cut == "after_commit" {
				want = 2
			}
			if e != nil || len(ps) != want {
				t.Fatalf("wrong durability side %d %v", len(ps), e)
			}
			m2 := recoveryManager(t, j, NewGate(&authority{}, binding), &testProofs{key: p.key, now: 1}, "slot")
			ps, e = j.Plans(ctx)
			if e != nil || len(ps) != 1 || ps[0].ID != foreign.ID {
				t.Fatal("unowned pin removed or owned pin leaked")
			}
			accept(t, j, h, 41)
			s, e := m2.Resume(ctx, binding, token, func(context.Context, RecoveryFrame) error { return nil })
			if e != nil || s == nil {
				t.Fatal(e)
			}
			if e = s.Step(); e != nil {
				t.Fatal(e)
			}
			m2.Close()
			j.Release(ctx, foreign.ID)
			noPins(t, j)
		})
	}
}
