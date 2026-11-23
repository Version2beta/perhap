package core

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// Enumerate input permutations; this deliberately does not use the evaluator's
// graph representation, ready heap or private helpers.
func eachEventPermutation(es []Event, visit func([]Event)) {
	copyOf := append([]Event(nil), es...)
	var walk func(int)
	walk = func(i int) {
		if i == len(copyOf) {
			visit(copyOf)
			return
		}
		for j := i; j < len(copyOf); j++ {
			copyOf[i], copyOf[j] = copyOf[j], copyOf[i]
			walk(i + 1)
			copyOf[i], copyOf[j] = copyOf[j], copyOf[i]
		}
	}
	walk(0)
}

func TestCOREGeneratedDAGReconstruction(t *testing.T) {
	const seed int64 = 20261123
	rng := rand.New(rand.NewSource(seed))
	for trial := 0; trial < 24; trial++ {
		t.Run(fmt.Sprintf("seed_%d_case_%d", seed, trial), func(t *testing.T) {
			const size = 5
			ids := rng.Perm(size)
			es := make([]Event, size)
			for i := range es {
				es[i] = Event{ID: fmt.Sprintf("%d", ids[i]+1), Type: "digit"}
				for j := 0; j < i; j++ {
					if rng.Intn(2) == 0 {
						es[i].Dependencies = append(es[i].Dependencies, Dependency{ID: es[j].ID, Kind: "local"})
					}
				}
			}
			// The byte-smallest topological sequence can be found independently by
			// enumerating ALL valid topological permutations, not by another Kahn sort.
			want := ""
			validOrders := 0
			eachEventPermutation(es, func(order []Event) {
				seen := map[string]bool{}
				key := ""
				for _, e := range order {
					for _, d := range e.Dependencies {
						if !seen[d.ID] {
							return
						}
					}
					seen[e.ID] = true
					key += e.ID
				}
				validOrders++
				if want == "" || key < want {
					want = key
				}
			})
			if validOrders == 0 {
				t.Fatal("generator produced no topological ordering")
			}
			m := counter()
			m.Reduce = func(s int, e Event) (int, error) { digit, err := strconv.Atoi(e.ID); return s*10 + digit, err }
			expectedState, _ := strconv.Atoi(want)
			wantIDs := strings.Split(want, "")
			eachEventPermutation(es, func(arrival []Event) {
				r := Evaluate(request(arrival...), m)
				if !reflect.DeepEqual(r.Attempted, wantIDs) || r.Working != expectedState || r.LastValid != expectedState || r.Completeness != "complete_as_of_frontier" || r.Evaluation != "valid" || len(r.Blocked) != 0 {
					t.Fatalf("history=%+v arrival=%+v want order %v state %d, got %+v", es, arrival, wantIDs, expectedState, r)
				}
				// Evaluate every progressively enlarged admitted prefix. Intermediate missing
				// parents are permitted; adding a late parent must reconstruct, not append.
				for n := 0; n <= len(arrival); n++ {
					q := request(arrival[:n]...)
					got := Evaluate(q, m)
					if n == len(arrival) {
						if !reflect.DeepEqual(got, r) {
							t.Fatalf("late reconstruction differs: %+v vs %+v", got, r)
						}
					} else {
						present := map[string]bool{}
						for _, e := range arrival[:n] {
							present[e.ID] = true
						}
						missing := false
						for _, e := range arrival[:n] {
							for _, d := range e.Dependencies {
								missing = missing || !present[d.ID]
							}
						}
						if missing && got.Completeness != "incomplete" {
							t.Fatalf("missing parent certified complete: %+v", got)
						}
					}
				}
			})
		})
	}
}

func TestCOREGeneratedGapsCyclesAndCoverage(t *testing.T) {
	for n := 1; n <= 8; n++ {
		t.Run(fmt.Sprintf("chain_%d", n), func(t *testing.T) {
			es := []Event{{ID: "independent", Type: "one"}}
			blocked := []string{}
			for i := 0; i < n; i++ {
				id := fmt.Sprintf("node-%d", i)
				parent := "missing"
				if i > 0 {
					parent = fmt.Sprintf("node-%d", i-1)
				}
				es = append(es, Event{ID: id, Dependencies: []Dependency{{ID: parent, Kind: "local"}}})
				blocked = append(blocked, id)
			}
			sort.Strings(blocked)
			for _, cycle := range []bool{false, true} {
				history := append([]Event(nil), es...)
				if cycle {
					history[1].Dependencies = []Dependency{{ID: history[len(history)-1].ID, Kind: "local"}}
				}
				for _, f := range []Frontier{{}, {Certified: true, Boundary: "test"}, {Gap: true}, {Certified: true}, {Certified: true, Boundary: "test", Gap: true}} {
					q := request(history...)
					q.Frontier = f
					r := Evaluate(q, counter())
					if r.Completeness != "incomplete" || r.Evaluation != "valid" || r.Working != 1 || !reflect.DeepEqual(r.Blocked, blocked) || !reflect.DeepEqual(r.Attempted, []string{"independent"}) || len(r.History) != len(history) {
						t.Fatalf("cycle=%v frontier=%+v: %+v", cycle, f, r)
					}
				}
			}
		})
	}
	for _, fail := range []bool{false, true} {
		for _, f := range []Frontier{{}, {Certified: true}, {Certified: true, Boundary: "b"}, {Certified: true, Boundary: "b", Gap: true}} {
			ty := "one"
			evaluation := "valid"
			if fail {
				ty = "fail"
				evaluation = "failed"
			}
			q := request(Event{ID: "a", Type: ty})
			q.Frontier = f
			want := "unavailable"
			if f.Certified && f.Boundary != "" {
				want = "complete_as_of_frontier"
			}
			if f.Gap {
				want = "incomplete"
			}
			r := Evaluate(q, counter())
			if r.Completeness != want || r.Evaluation != evaluation || !r.WorkingAvailable {
				t.Fatalf("coverage independent of failure: %+v", r)
			}
		}
	}
}

func TestCOREGeneratedLateOrderSensitiveCorrection(t *testing.T) {
	for value := 3; value <= 18; value++ {
		m := counter()
		m.Reduce = func(s int, e Event) (int, error) {
			switch e.Type {
			case "initial":
				return 1, nil
			case "late-set":
				return value, nil
			case "add":
				return s + 2, nil
			}
			return s, nil
		}
		before := []Event{{ID: "c", Type: "add"}, {ID: "a", Type: "initial"}}
		first := Evaluate(request(before...), m)
		if first.Working != 3 {
			t.Fatalf("before late set: %+v", first)
		}
		enlarged := append(before, Event{ID: "b", Type: "late-set"})
		eachEventPermutation(enlarged, func(arrival []Event) {
			r := Evaluate(request(arrival...), m)
			if r.Working != value+2 || !reflect.DeepEqual(r.Attempted, []string{"a", "b", "c"}) {
				t.Fatalf("late set %d must reconstruct, not append: %+v", value, r)
			}
		})
		if first.Working != 3 {
			t.Fatal("new evaluation mutated earlier result")
		}
	}
}

func TestCOREGeneratedRetriesAndConflicts(t *testing.T) {
	base := Event{ID: "a", Context: "demo", Type: "repair", Payload: "original", AssertedBy: "verified:test", Subjects: []string{"s"}, Dependencies: []Dependency{{ID: "foreign", Kind: "trace-external"}}}
	for copies := 1; copies <= 12; copies++ {
		es := make([]Event, copies)
		for i := range es {
			es[i] = base
		}
		r := Evaluate(request(es...), counter())
		if len(r.History) != 1 || r.Working != 5 || len(r.Attempted) != 1 || len(r.Diagnostics) != 0 {
			t.Fatalf("retry %d: %+v", copies, r)
		}
	}
	changes := []func(*Event){func(e *Event) { e.Type = "one" }, func(e *Event) { e.Payload = "changed" }, func(e *Event) { e.AssertedBy = "other" }, func(e *Event) { e.Context = "other" }, func(e *Event) { e.Subjects = []string{"other"} }, func(e *Event) { e.Dependencies = []Dependency{{ID: "other", Kind: "trace-external"}} }}
	for i, change := range changes {
		other := base
		change(&other)
		for _, pair := range [][]Event{{base, other}, {other, base}} {
			r := Evaluate(request(pair...), counter())
			if len(r.History) != 1 || !reflect.DeepEqual(r.History[0], pair[0]) {
				t.Fatalf("conflict %d overwrote first admitted record: %+v", i, r)
			}
			found := false
			for _, d := range r.Diagnostics {
				found = found || d.Condition == "conflicting_id"
			}
			if !found {
				t.Fatalf("conflict %d not diagnosed", i)
			}
		}
	}
	// Conflicting submissions are not equivalent normalized admitted inputs: the
	// record already admitted first must remain, whichever content it contains.
}

func TestCOREGeneratedInvalidRepairAndFinalSelection(t *testing.T) {
	callbackFailure := errors.New("failed proposal handler")
	for debt := 1; debt <= 12; debt++ {
		t.Run(fmt.Sprintf("debt_%d", debt), func(t *testing.T) {
			m := counter()
			m.Reduce = func(s int, e Event) (int, error) {
				switch e.Type {
				case "debit":
					return -debt, nil
				case "credit":
					return s + debt + 1, nil
				case "fail-proposal":
					return s, callbackFailure
				}
				return s, nil
			}
			m.Select = func(s int, es []Event) (Selection, error) {
				out := Selection{}
				if s <= 0 {
					return out, nil
				}
				for _, e := range es {
					if e.Type == "proposal" || e.Type == "fail-proposal" {
						out.Proposals = append(out.Proposals, Proposal{Key: e.ID, Action: "notify"})
					}
				}
				return out, nil
			}
			es := []Event{{ID: "a", Type: "debit"}, {ID: "b", Type: "credit"}, {ID: "c", Type: "proposal"}, {ID: "d", Type: "fail-proposal"}, {ID: "e", Type: "proposal", Dependencies: []Dependency{{ID: "absent", Kind: "local"}}}}
			invalid := Evaluate(request(es[0]), m)
			if invalid.Working != -debt || invalid.Evaluation != "invalid" || invalid.LastValid != 0 || !invalid.LastValidAvailable {
				t.Fatalf("lost invalid candidate: %+v", invalid)
			}
			eachEventPermutation(es, func(arrival []Event) {
				r := Evaluate(request(arrival...), m)
				if r.Working != 1 || r.LastValid != 1 || r.Evaluation != "failed" || r.Completeness != "incomplete" || !reflect.DeepEqual(r.Selection.Proposals, []Proposal{{Key: "c", Action: "notify"}}) || !reflect.DeepEqual(r.Blocked, []string{"e"}) {
					t.Fatalf("failed/blocked proposal or failed repair: %+v", r)
				}
			})
			// Newly admitted late evidence unblocks only e; d remains ineffective.
			es = append(es, Event{ID: "absent"})
			r := Evaluate(request(es...), m)
			if len(r.Selection.Proposals) != 2 || r.Selection.Proposals[1].Key != "e" || r.Completeness != "complete_as_of_frontier" || r.Evaluation != "failed" {
				t.Fatalf("late unblock: %+v", r)
			}
			// An intermediate positive state does not keep a final unsupported proposal.
			m.Initialize = func() (int, error) { return 1, nil }
			r = Evaluate(request(Event{ID: "a", Type: "proposal"}, Event{ID: "b", Type: "debit"}), m)
			if len(r.Selection.Proposals) != 0 || r.Working != -debt {
				t.Fatalf("stale intermediate proposal: %+v", r)
			}
		})
	}
}
