package core

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func counter() Model[int] {
	return Model[int]{Initialize: func() (int, error) { return 0, nil }, Clone: func(s int) (int, error) { return s, nil }, Validate: func(s int) (bool, error) { return s >= 0, nil }, Reduce: func(s int, e Event) (int, error) {
		switch e.Type {
		case "fail":
			return -999, errors.New("failed")
		case "negative":
			return -2, nil
		case "repair":
			return s + 5, nil
		case "one":
			return 1, nil
		}
		return s, nil
	}, Select: func(s int, es []Event) (Selection, error) { return Selection{}, nil }}
}
func request(es ...Event) Request {
	return Request{Context: "demo", Scope: "counter", RuleVersion: "v1", Events: es, Frontier: Frontier{Boundary: "closed-test-set", Certified: true}}
}
func TestCOREBehavior(t *testing.T) {
	m := counter()
	r := Evaluate(request(Event{ID: "a", Type: "negative"}), m)
	if r.Working != -2 || r.LastValid != 0 || !r.LastValidAvailable || r.Evaluation != "invalid" {
		t.Fatalf("invalid retained: %+v", r)
	}
	r = Evaluate(request(Event{ID: "b", Type: "repair"}, Event{ID: "a", Type: "negative"}), m)
	if r.Working != 3 || r.LastValid != 3 || r.Evaluation != "valid" || !reflect.DeepEqual(r.Attempted, []string{"a", "b"}) {
		t.Fatalf("repair/order: %+v", r)
	}
	r = Evaluate(request(Event{ID: "a", Type: "fail"}, Event{ID: "b", Type: "one", Dependencies: []Dependency{{ID: "a", Kind: "local"}}}, Event{ID: "c", Dependencies: []Dependency{{ID: "a", Kind: "local", RequiresSuccess: true}}}, Event{ID: "d", Dependencies: []Dependency{{ID: "c", Kind: "local"}}}), m)
	if r.Working != 1 || r.Evaluation != "failed" || !reflect.DeepEqual(r.Attempted, []string{"a", "b"}) || !reflect.DeepEqual(r.Blocked, []string{"c", "d"}) {
		t.Fatalf("dependency classes: %+v", r)
	}
	m.Initialize = func() (int, error) { return -2, nil }
	r = Evaluate(request(Event{ID: "a", Type: "repair"}), m)
	if r.Working != 3 || r.Evaluation != "valid" {
		t.Fatalf("initial repair: %+v", r)
	}
	m.Initialize = func() (int, error) { return 0, errors.New("init") }
	r = Evaluate(request(Event{ID: "a"}), m)
	if r.WorkingAvailable || r.LastValidAvailable || len(r.Attempted) != 0 || r.Evaluation != "failed" {
		t.Fatalf("failed init: %+v", r)
	}
	m = counter()
	m.Validate = func(int) (bool, error) { return false, errors.New("validation") }
	r = Evaluate(request(Event{ID: "a"}), m)
	if !r.WorkingAvailable || r.LastValidAvailable || len(r.Attempted) != 0 || r.Evaluation != "failed" {
		t.Fatalf("failed initial validation: %+v", r)
	}
	for _, tc := range []struct {
		f    Frontier
		deps []Dependency
		want string
	}{{Frontier{}, nil, "unavailable"}, {Frontier{Certified: true}, nil, "unavailable"}, {Frontier{Certified: true, Boundary: "b"}, nil, "complete_as_of_frontier"}, {Frontier{Certified: true, Boundary: "b", Gap: true}, nil, "incomplete"}, {Frontier{}, []Dependency{{ID: "absent", Kind: "local"}}, "incomplete"}} {
		q := request(Event{ID: "a", Dependencies: tc.deps})
		q.Frontier = tc.f
		r = Evaluate(q, counter())
		if r.Completeness != tc.want || !r.WorkingAvailable || r.Evaluation != "valid" {
			t.Fatalf("coverage %s: %+v", tc.want, r)
		}
	}
}
func TestCOREOwnership(t *testing.T) {
	original := []int{0}
	m := Model[[]int]{Initialize: func() ([]int, error) { return original, nil }, Clone: func(s []int) ([]int, error) { return append([]int(nil), s...), nil }, Validate: func(s []int) (bool, error) { return s[0] >= 0, nil }, Reduce: func(s []int, e Event) ([]int, error) { s[0] = -5; return s, errors.New("mutation before failure") }, Select: func(s []int, e []Event) (Selection, error) { s[0] = 900; return Selection{}, nil }}
	r := Evaluate(request(Event{ID: "a"}), m)
	if r.Working[0] != 0 || r.LastValid[0] != 0 || original[0] != 0 || r.Evaluation != "failed" {
		t.Fatalf("alias after failed callback: %+v original %v", r, original)
	}
	m.Reduce = func(s []int, e Event) ([]int, error) { s[0] = 7; return s, nil }
	m.Validate = func(s []int) (bool, error) {
		if s[0] == 7 {
			s[0] = 99
			return false, errors.New("validator")
		}
		s[0] = 33
		return true, nil
	}
	r = Evaluate(request(Event{ID: "a"}), m)
	if r.Working[0] != 0 || r.LastValid[0] != 0 || r.Evaluation != "failed" {
		t.Fatalf("validator mutation: %+v", r)
	}
	m.Clone = func([]int) ([]int, error) { return nil, errors.New("cannot copy") }
	r = Evaluate(request(), m)
	if r.Evaluation != "failed" || r.WorkingAvailable {
		t.Fatalf("clone failure: %+v", r)
	}
	nilModel := Model[*int]{Initialize: func() (*int, error) { return nil, nil }, Clone: func(s *int) (*int, error) { return s, nil }, Validate: func(*int) (bool, error) { return true, nil }, Reduce: func(s *int, e Event) (*int, error) { return s, nil }, Select: func(*int, []Event) (Selection, error) { return Selection{}, nil }}
	nr := Evaluate(request(), nilModel)
	if !nr.WorkingAvailable || !nr.LastValidAvailable || nr.Working != nil {
		t.Fatalf("valid null: %+v", nr)
	}
}
func TestCOREPermutations(t *testing.T) {
	events := []Event{{ID: "a", Type: "repair", Dependencies: []Dependency{{ID: "z", Kind: "local"}}}, {ID: "z", Type: "negative"}, {ID: "m", Type: "one"}}
	var baseline Result[int]
	n := 0
	var perm func(int)
	perm = func(i int) {
		if i == len(events) {
			r := Evaluate(request(events...), counter())
			if n == 0 {
				baseline = r
			}
			if !reflect.DeepEqual(r, baseline) {
				t.Fatalf("arrival-dependent: %+v vs %+v", r, baseline)
			}
			n++
			return
		}
		for j := i; j < len(events); j++ {
			events[i], events[j] = events[j], events[i]
			perm(i + 1)
			events[i], events[j] = events[j], events[i]
		}
	}
	perm(0)
	if n != 6 || !reflect.DeepEqual(baseline.Attempted, []string{"m", "z", "a"}) || baseline.Working != 3 {
		t.Fatalf("causal permutation: %+v", baseline)
	}
	q := request(Event{ID: "a", Dependencies: []Dependency{{ID: "b", Kind: "local"}}}, Event{ID: "b", Dependencies: []Dependency{{ID: "a", Kind: "local"}}}, Event{ID: "c", Dependencies: []Dependency{{ID: "a", Kind: "local"}}}, Event{ID: "d", Type: "one"})
	r := Evaluate(q, counter())
	if !reflect.DeepEqual(r.Blocked, []string{"a", "b", "c"}) || r.Working != 1 || r.Completeness != "incomplete" {
		t.Fatalf("cycle descendants: %+v", r)
	}
	q = request(Event{ID: "a", Dependencies: []Dependency{{ID: "missing", Kind: "local"}}})
	before := Evaluate(q, counter())
	q.Events = append(q.Events, Event{ID: "z", Type: "one"})
	after := Evaluate(q, counter())
	if len(before.Diagnostics) == 0 || len(after.Diagnostics) == 0 || before.Diagnostics[0].Key != after.Diagnostics[0].Key || !strings.Contains(before.Diagnostics[0].Key, "v1") {
		t.Fatalf("unstable diagnostic keys: %+v %+v", before, after)
	}
}
