package core

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sort"
	"testing"
)

type fixtureEvent struct {
	ID, Context, Scope, Type string
	Data                     json.RawMessage
	AssertedBy               string `json:"asserted_by"`
	Cause                    *struct{ ID, Scope string }
	RequiresExternal         bool `json:"requires_external"`
	RequiresSuccess          bool `json:"requires_success"`
}
type fixtureInput struct {
	Events   []fixtureEvent
	Frontier struct {
		Continuous, Gap bool `json:"-"`
	}
	Expected map[string]json.RawMessage
}
type fixtureCase struct {
	ID, Phase           string
	Events, Submissions []fixtureEvent
	Frontier            json.RawMessage
	Initialization      struct {
		State             *struct{ N int }
		Failure           string
		ValidationFailure string `json:"validation_failure"`
	}
	Expected      map[string]json.RawMessage
	Before, After struct {
		Events   []fixtureEvent
		Frontier json.RawMessage
		Expected map[string]json.RawMessage
	}
	Late fixtureEvent
}

func fixtureEvents(es []fixtureEvent) []Event {
	out := []Event{}
	for _, e := range es {
		v := Event{ID: e.ID, Context: e.Context, Type: e.Type, Payload: string(e.Data), AssertedBy: e.AssertedBy, Subjects: []string{e.Scope}}
		if e.Cause != nil {
			kind := "unknown"
			switch e.Cause.Scope {
			case "local":
				kind = "local"
			case "external":
				kind = "trace-external"
				if e.RequiresExternal {
					kind = "required-external"
				}
			}
			v.Dependencies = []Dependency{{ID: e.Cause.ID, Kind: kind, RequiresSuccess: e.RequiresSuccess}}
		}
		out = append(out, v)
	}
	return out
}
func fixtureResult(t *testing.T, c fixtureCase, es []fixtureEvent, frontier json.RawMessage) map[string]any {
	t.Helper()
	q := request(fixtureEvents(es)...)
	q.Frontier = Frontier{}
	var f struct {
		Continuous bool
		Gap        bool `json:"gap_detected"`
	}
	if len(frontier) > 0 {
		if err := json.Unmarshal(frontier, &f); err != nil {
			t.Fatal(err)
		}
	}
	q.Frontier = Frontier{Certified: f.Continuous, Gap: f.Gap}
	if f.Continuous {
		q.Frontier.Boundary = "trusted-fixture-closed-set"
	}
	m := counter()
	m.Initialize = func() (int, error) {
		if c.Initialization.Failure != "" {
			return 0, errors.New("init")
		}
		if c.Initialization.State != nil {
			return c.Initialization.State.N, nil
		}
		return 0, nil
	}
	m.Validate = func(s int) (bool, error) {
		if c.Initialization.ValidationFailure != "" {
			return false, errors.New("validator")
		}
		return s >= 0, nil
	}
	m.Reduce = func(s int, e Event) (int, error) {
		var d struct{ N int }
		if err := json.Unmarshal([]byte(e.Payload), &d); err != nil {
			return s, err
		}
		switch e.Type {
		case "set":
			return d.N, nil
		case "add":
			return s + d.N, nil
		case "fail":
			return 0, errors.New("reducer")
		}
		return s, nil
	}
	m.Select = func(s int, es []Event) (Selection, error) {
		p := map[string]Proposal{}
		for _, e := range es {
			var d struct {
				Key string `json:"action_key"`
			}
			if err := json.Unmarshal([]byte(e.Payload), &d); err != nil {
				return Selection{}, err
			}
			switch e.Type {
			case "propose":
				p[d.Key] = Proposal{Key: d.Key, Action: "fixture"}
			case "propose_if_positive":
				if s > 0 {
					p[d.Key] = Proposal{Key: d.Key, Action: "fixture"}
				}
			case "cancel_proposal":
				delete(p, d.Key)
			}
		}
		keys := []string{}
		for k := range p {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := Selection{}
		for _, k := range keys {
			out.Proposals = append(out.Proposals, p[k])
		}
		return out, nil
	}
	r := Evaluate(q, m)
	codes := []string{}
	seen := map[string]bool{}
	for _, d := range r.Diagnostics {
		if !seen[d.Condition] {
			codes = append(codes, d.Condition)
			seen[d.Condition] = true
		}
	}
	proposals := []map[string]any{}
	for _, p := range r.Selection.Proposals {
		proposals = append(proposals, map[string]any{"action_key": p.Key, "commit_eligible": false})
	}
	return map[string]any{"attempted": r.Attempted, "blocked": r.Blocked, "working": map[string]int{"n": r.Working}, "working_available": r.WorkingAvailable, "last_valid": map[string]int{"n": r.LastValid}, "last_valid_available": r.LastValidAvailable, "evaluation": r.Evaluation, "completeness": r.Completeness, "diagnostics": codes, "proposals": proposals}
}
func subset(t *testing.T, actual map[string]any, expected map[string]json.RawMessage) {
	t.Helper()
	for k, want := range expected {
		v, ok := actual[k]
		if !ok {
			t.Fatalf("missing %s", k)
		}
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		var a, b any
		if err = json.Unmarshal(raw, &a); err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(want, &b); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(a, b) {
			t.Errorf("%s: got %s want %s", k, raw, want)
		}
	}
}
func TestCOREFixtures(t *testing.T) {
	raw, err := os.ReadFile("testdata/core.json")
	if err != nil {
		t.Fatal(err)
	}
	var suite struct{ Cases []fixtureCase }
	if err = json.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}
	if len(suite.Cases) != 29 {
		t.Fatalf("expected 29 fixtures, got %d", len(suite.Cases))
	}
	for _, c := range suite.Cases {
		t.Run(c.ID, func(t *testing.T) {
			switch c.Phase {
			case "admission":
				q := request(fixtureEvents(c.Submissions)...)
				r := Evaluate(q, counter())
				ids := []string{}
				for _, e := range r.History {
					ids = append(ids, e.ID)
				}
				codes := []string{}
				for _, d := range r.Diagnostics {
					if d.Condition == "conflicting_id" {
						codes = append(codes, d.Condition)
					}
				}
				result := map[string]any{"admitted_ids": ids, "diagnostics": codes}
				if len(r.History) > 0 {
					var d any
					if err := json.Unmarshal([]byte(r.History[0].Payload), &d); err != nil {
						t.Fatal(err)
					}
					result["admitted_content"] = d
				}
				subset(t, result, c.Expected)
			case "replay":
				subset(t, fixtureResult(t, c, c.Before.Events, c.Before.Frontier), c.Before.Expected)
				es := append(append([]fixtureEvent{}, c.Before.Events...), c.Late)
				subset(t, fixtureResult(t, c, es, c.After.Frontier), c.After.Expected)
			default:
				subset(t, fixtureResult(t, c, c.Events, c.Frontier), c.Expected)
			}
		})
	}
}
