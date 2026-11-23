package rev

import (
	"encoding/json"
	"github.com/version2beta/perhap/sdk/go/core"
	"os"
	"reflect"
	"testing"
)

var certificate = core.Frontier{Boundary: "closed-test-set", Certified: true}
var testKey = Key{Context: "demo", Producer: "p", Subject: "s", Meaning: "value"}

func pub(id string, n uint64, op, value string) Publication {
	return Publication{ID: id, Key: testKey, Revision: n, Op: op, Value: value}
}
func TestREVReconstruction(t *testing.T) {
	ps := []Publication{pub("z", 1, "upsert", "2"), pub("a", 2, "upsert", "7"), pub("b", 3, "retract", "")}
	var perm func(int)
	perm = func(i int) {
		if i == len(ps) {
			v := Reconstruct(ps, certificate)
			a, ok := v.Assertions[testKey]
			if !ok || a.Status != "retracted" || a.ValueAvailable || a.Revision != 3 || v.Completeness != "complete_as_of_frontier" || !reflect.DeepEqual(v.History, ps) {
				t.Fatalf("bad reconstruction: %+v", v)
			}
			return
		}
		for j := i; j < len(ps); j++ {
			ps[i], ps[j] = ps[j], ps[i]
			perm(i + 1)
			ps[i], ps[j] = ps[j], ps[i]
		}
	}
	perm(0)
	first := pub("one", 1, "upsert", "null")
	for _, f := range []core.Frontier{certificate, {}, {Certified: true, Boundary: "b", Gap: true}} {
		v := Reconstruct([]Publication{first, pub("three", 3, "upsert", "new")}, f)
		a := v.Assertions[testKey]
		if v.Completeness != "incomplete" || a.Status != "unresolved" || a.Value != "null" || !a.ValueAvailable || len(v.Diagnostics) == 0 {
			t.Fatalf("gap lost stale evidence: %+v", v)
		}
	}
	v := Reconstruct([]Publication{first, first}, certificate)
	if v.Assertions[testKey].Revision != 1 || len(v.Diagnostics) != 0 || len(v.History) != 2 {
		t.Fatalf("retry: %+v", v)
	}
	other := first
	other.Value = "conflict"
	for _, es := range [][]Publication{{first, other}, {other, first}, {first, pub("another-id", 1, "upsert", "different")}} {
		v = Reconstruct(es, certificate)
		if v.Completeness != "incomplete" || v.Assertions[testKey].Status != "unresolved" || len(v.Diagnostics) == 0 {
			t.Fatalf("conflict silently accepted: %+v", v)
		}
	}
	for _, f := range []core.Frontier{{}, {Certified: true}, {Gap: true}} {
		v = Reconstruct([]Publication{first}, f)
		want := "unavailable"
		if f.Gap {
			want = "incomplete"
		}
		if v.Completeness != want || v.Assertions[testKey].Status != "unresolved" {
			t.Fatalf("uncertified authority: %+v", v)
		}
	}
	second := pub("two", 2, "upsert", "7")
	second.Supersedes = "wrong"
	v = Reconstruct([]Publication{first, second}, certificate)
	if v.Assertions[testKey].Status != "unresolved" || v.Completeness != "incomplete" {
		t.Fatalf("wrong link: %+v", v)
	}
	second = pub("foreign", 1, "upsert", "separate")
	second.Key.Producer = "different"
	v = Reconstruct([]Publication{first, second}, certificate)
	if len(v.Assertions) != 2 {
		t.Fatalf("scoped key collision: %+v", v)
	}
	ps = []Publication{first, pub("two", 2, "status", "")}
	ps[1].Status = "unresolved"
	v = Reconstruct(ps, certificate)
	if v.Assertions[testKey].Value != "null" || v.Assertions[testKey].Status != "unresolved" {
		t.Fatalf("status lost value: %+v", v)
	}
	third := pub("three", 3, "status", "")
	third.Status = "current"
	v = Reconstruct(append(ps, third), certificate)
	if v.Assertions[testKey].Status != "current" || v.Assertions[testKey].Value != "null" {
		t.Fatalf("status recovery: %+v", v)
	}
}
func TestREVReconciliation(t *testing.T) {
	a := Assertion{Revision: 8, PublicationID: "old", Status: "current", Value: "null", ValueAvailable: true}
	for _, evaluation := range []string{"valid", "invalid", "failed"} {
		for _, coverage := range []string{"complete_as_of_frontier", "incomplete", "unavailable"} {
			c := Computation{Key: testKey, Evaluation: evaluation, Completeness: coverage}
			d := Reconcile(a, c)
			if evaluation == "valid" && coverage == "complete_as_of_frontier" {
				if d.Assertion.Status != "retracted" || d.Intention == nil || d.Intention.Op != "retract" {
					t.Fatalf("complete disappearance: %+v", d)
				}
			} else if d.Assertion.Status != "unresolved" || d.Assertion.Value != a.Value || d.Intention == nil || d.Intention.Op != "status" {
				t.Fatalf("uncertainty retracts: %+v", d)
			}
			if d.Intention.ExpectedRevision != 8 || d.Assertion.Revision != 8 {
				t.Fatalf("allocated durable revision: %+v", d)
			}
		}
	}
	c := Computation{Key: testKey, Present: true, Value: "null", Evaluation: "valid", Completeness: "complete_as_of_frontier"}
	if d := Reconcile(a, c); d.Intention != nil {
		t.Fatalf("unchanged publication: %+v", d)
	}
	a.Status = "unresolved"
	d := Reconcile(a, c)
	if d.Intention == nil || d.Intention.Op != "status" || d.Intention.Status != "current" {
		t.Fatalf("same value recovery: %+v", d)
	}
	c.Value = "new"
	d = Reconcile(a, c)
	if d.Intention == nil || d.Intention.Op != "upsert" || d.Assertion.Value != "new" {
		t.Fatalf("correction: %+v", d)
	}
}

type fixturePublication struct {
	ID              string `json:"event_id"`
	Producer, Key   string
	Revision        uint64
	Op              string
	Value           json.RawMessage
	Supersedes      string
	Type            string `json:"event_type"`
	Cause           string `json:"cause_event_id"`
	RequiresSuccess bool   `json:"requires_success"`
}

func convert(p fixturePublication) Publication {
	return Publication{ID: p.ID, Key: Key{Context: "demo", Producer: p.Producer, Subject: "fixture", Meaning: p.Key}, Revision: p.Revision, Op: p.Op, Value: string(p.Value), Supersedes: p.Supersedes}
}
func TestREVFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/revisions.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Cases []struct {
			ID           string
			Publications []fixturePublication
			Previous     fixturePublication
			Recomputed   struct {
				Value                    json.RawMessage
				Evaluation, Completeness string
			}
			StatusRevision struct{ Status string }
			Ordinary       []struct {
				ID    string `json:"event_id"`
				Type  string
				Value int
				Cause string `json:"cause_publication"`
			} `json:"ordinary_events"`
			Expected map[string]json.RawMessage
		}
	}
	if err = json.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}
	tested := 0
	for _, c := range suite.Cases {
		if len(c.Publications) == 0 && c.Previous.ID == "" {
			continue
		}
		tested++
		t.Run(c.ID, func(t *testing.T) {
			if len(c.Publications) == 0 {
				a := Assertion{Revision: c.Previous.Revision, PublicationID: c.Previous.ID, Status: "current", Value: string(c.Previous.Value), ValueAvailable: true}
				if c.StatusRevision.Status != "" {
					a.Status = c.StatusRevision.Status
				}
				evaluation, coverage := c.Recomputed.Evaluation, c.Recomputed.Completeness
				if evaluation == "" {
					evaluation = "valid"
				}
				if coverage == "" {
					coverage = "complete_as_of_frontier"
				}
				d := Reconcile(a, Computation{Key: convert(c.Previous).Key, Present: len(c.Recomputed.Value) > 0, Value: string(c.Recomputed.Value), Evaluation: evaluation, Completeness: coverage})
				if a.Status == "unresolved" {
					if d.Intention == nil || d.Intention.Status != "current" || d.Assertion.Value != a.Value {
						t.Fatalf("recovery fixture: %+v", d)
					}
				} else if evaluation == "failed" {
					if d.Assertion.Status != "unresolved" || d.Assertion.Value != a.Value || d.Intention == nil || d.Intention.Op == "retract" {
						t.Fatalf("failure fixture: %+v", d)
					}
				} else if d.Intention != nil {
					t.Fatalf("unchanged fixture: %+v", d)
				}
				return
			}
			ps := []Publication{}
			for _, p := range c.Publications {
				ps = append(ps, convert(p))
			}
			v := Reconstruct(ps, certificate)
			if len(c.Ordinary) == 0 {
				if len(ps) == 1 && ps[0].Revision == 2 {
					if v.Completeness != "incomplete" || v.Assertions[ps[0].Key].Status != "unresolved" {
						t.Fatalf("gap: %+v", v)
					}
				} else {
					a := v.Assertions[ps[0].Key]
					last := ps[len(ps)-1]
					if last.Op == "retract" {
						if a.Status != "retracted" || a.ValueAvailable {
							t.Fatalf("retract: %+v", a)
						}
					} else if a.Value != last.Value || a.Status != "current" {
						t.Fatalf("correction: %+v", a)
					}
					if len(v.History) != len(ps) {
						t.Fatal("lost history")
					}
				}
				return
			}
			// Domain-defined counter history goes through UNCHANGED CORE, not the
			// current-assertion view. Every publication remains at its own event position.
			es := []core.Event{}
			for _, p := range c.Publications {
				e := core.Event{ID: p.ID, Type: p.Type, Payload: string(p.Value)}
				if p.Cause != "" {
					e.Dependencies = []core.Dependency{{ID: p.Cause, Kind: "local", RequiresSuccess: p.RequiresSuccess}}
				}
				es = append(es, e)
			}
			for _, e := range c.Ordinary {
				payload, _ := json.Marshal(e.Value)
				es = append(es, core.Event{ID: e.ID, Type: e.Type, Payload: string(payload), Dependencies: []core.Dependency{{ID: e.Cause, Kind: "local"}}})
			}
			m := core.Model[int]{Initialize: func() (int, error) { return 0, nil }, Clone: func(s int) (int, error) { return s, nil }, Validate: func(int) (bool, error) { return true, nil }, Select: func(int, []core.Event) (core.Selection, error) { return core.Selection{}, nil }, Reduce: func(s int, e core.Event) (int, error) {
				if e.Type == "" {
					return s, nil
				}
				var n int
				if err := json.Unmarshal([]byte(e.Payload), &n); err != nil {
					return s, err
				}
				if e.Type == "set" {
					return n, nil
				}
				return s + n, nil
			}}
			r := core.Evaluate(core.Request{Context: "demo", Scope: "counter", RuleVersion: "fixture/v1", Events: es, Frontier: certificate}, m)
			var expectedIDs []string
			if err := json.Unmarshal(c.Expected["interpreted_event_ids"], &expectedIDs); err != nil {
				t.Fatal(err)
			}
			if r.Working != 7 || !reflect.DeepEqual(r.Attempted, expectedIDs) || r.Completeness != "complete_as_of_frontier" {
				t.Fatalf("historical replay: %+v", r)
			}
			a := v.Assertions[ps[0].Key]
			if ps[len(ps)-1].Op == "retract" {
				if a.Status != "retracted" {
					t.Fatal(a)
				}
			} else if a.Value != "7" || a.Status != "current" {
				t.Fatal(a)
			}
		})
	}
	if tested != 8 {
		t.Fatalf("expected 8 pure fixture cases, got %d", tested)
	}
}
