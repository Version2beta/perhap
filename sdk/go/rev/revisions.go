package rev

import (
	"sort"

	"github.com/version2beta/perhap/sdk/go/core"
)

// Reconstruct selects contiguous verified revisions per key; gaps/conflicts leave
// stale evidence unresolved, never authoritative. Frontier evidence is host-owned.
func Reconstruct(publications []Publication, frontier core.Frontier) View {
	history := make([]Publication, len(publications))
	copy(history, publications)
	records := distinctRecords(publications)
	idCount := map[string]int{}
	groups := map[Key][]Publication{}
	for _, p := range records {
		if p.ID != "" {
			idCount[p.ID]++
		}
		groups[p.Key] = append(groups[p.Key], p)
	}
	view := View{History: history, Assertions: make(map[Key]Assertion, len(groups))}
	var diags []Diagnostic
	for key, recs := range groups {
		a, keyDiags := reconstructKey(key, recs, idCount)
		view.Assertions[key] = a
		diags = append(diags, keyDiags...)
	}
	view.Diagnostics = sortedDistinct(diags)
	// Only a certified, nonempty boundary without a known gap can establish
	// authoritative current keys; otherwise keep the value but mark it unresolved.
	trusted := frontier.Certified && frontier.Boundary != "" && !frontier.Gap
	for key, a := range view.Assertions {
		if !trusted {
			a.Status = "unresolved"
			view.Assertions[key] = a
		}
	}
	switch {
	case frontier.Gap || len(view.Diagnostics) > 0:
		view.Completeness = "incomplete"
	case trusted:
		view.Completeness = "complete_as_of_frontier"
	default:
		view.Completeness = "unavailable"
	}
	return view
}

// distinctRecords drops identical retries, keeping the first supplied occurrence.
// Publication is comparable, so identical values are exact duplicates.
func distinctRecords(publications []Publication) []Publication {
	seen := map[Publication]bool{}
	var out []Publication
	for _, p := range publications {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// reconstructKey reports every conflict and malformed record for one key, then
// applies only the contiguous prefix that ends before the first problem.
func reconstructKey(key Key, recs []Publication, idCount map[string]int) (Assertion, []Diagnostic) {
	byRev := map[uint64][]Publication{}
	var revs []uint64
	for _, p := range recs {
		if _, ok := byRev[p.Revision]; !ok {
			revs = append(revs, p.Revision)
		}
		byRev[p.Revision] = append(byRev[p.Revision], p)
	}
	sort.Slice(revs, func(i, j int) bool { return revs[i] < revs[j] })
	var diags []Diagnostic
	add := func(rev uint64, condition string) {
		diags = append(diags, Diagnostic{Key: key, Revision: rev, Condition: condition})
	}
	for _, rev := range revs {
		ps := byRev[rev]
		if len(ps) > 1 {
			add(rev, "conflicting_revision")
		}
		for _, p := range ps {
			if p.ID != "" && idCount[p.ID] > 1 {
				add(rev, "conflicting_id")
			}
			if c := recordProblem(p); c != "" {
				add(rev, c)
			}
		}
	}

	a := Assertion{Status: "unresolved"}
	prevID := ""
	expected := uint64(1)
	for _, rev := range revs {
		if rev != expected {
			if rev > expected {
				add(expected, "revision_gap")
			}
			break
		}
		ps := byRev[rev]
		if len(ps) != 1 || (ps[0].ID != "" && idCount[ps[0].ID] > 1) || recordProblem(ps[0]) != "" {
			break
		}
		p := ps[0]
		if p.Supersedes != "" && p.Supersedes != prevID {
			add(rev, "invalid_link")
			break
		}
		next, condition := applyPublication(a, p)
		if condition != "" {
			add(rev, condition)
			break
		}
		a = next
		prevID = p.ID
		expected++
	}
	if len(diags) > 0 {
		a.Status = "unresolved"
	}
	return a, diags
}

// recordProblem names the condition that prevents a single record from being
// interpreted, or returns "" when its own fields are well formed.
func recordProblem(p Publication) string {
	if p.Revision == 0 {
		return "revision_gap"
	}
	if p.ID == "" {
		return "invalid_link"
	}
	switch p.Op {
	case "upsert", "retract":
		if p.Status != "" {
			return "invalid_status"
		}
	case "status":
		if p.Status != "current" && p.Status != "unresolved" {
			return "invalid_status"
		}
	default:
		return "invalid_operation"
	}
	return ""
}

// applyPublication returns the assertion after one well-formed, contiguous record.
// A status change to current needs retained content, so it fails on a retracted
// or never-valued prior state.
func applyPublication(a Assertion, p Publication) (Assertion, string) {
	switch p.Op {
	case "upsert":
		return Assertion{Revision: p.Revision, PublicationID: p.ID, Status: "current", Value: p.Value, ValueAvailable: true}, ""
	case "retract":
		return Assertion{Revision: p.Revision, PublicationID: p.ID, Status: "retracted"}, ""
	}
	if p.Status == "current" && !a.ValueAvailable {
		return a, "invalid_status"
	}
	a.Revision, a.PublicationID, a.Status = p.Revision, p.ID, p.Status
	return a, ""
}

// sortedDistinct orders diagnostics by key, revision and condition, and drops
// exact repeats so one fault is reported once.
func sortedDistinct(diags []Diagnostic) []Diagnostic {
	sort.SliceStable(diags, func(i, j int) bool { return diagnosticLess(diags[i], diags[j]) })
	var out []Diagnostic
	for i, d := range diags {
		if i == 0 || d != diags[i-1] {
			out = append(out, d)
		}
	}
	return out
}

func diagnosticLess(a, b Diagnostic) bool {
	if a.Key != b.Key {
		if a.Key.Context != b.Key.Context {
			return a.Key.Context < b.Key.Context
		}
		if a.Key.Producer != b.Key.Producer {
			return a.Key.Producer < b.Key.Producer
		}
		if a.Key.Subject != b.Key.Subject {
			return a.Key.Subject < b.Key.Subject
		}
		if a.Key.Meaning != b.Key.Meaning {
			return a.Key.Meaning < b.Key.Meaning
		}
	}
	if a.Revision != b.Revision {
		return a.Revision < b.Revision
	}
	return a.Condition < b.Condition
}

// Reconcile computes one pure next-state/intention without allocating IDs or
// revision numbers. Unknown/failed evidence never justifies disappearance.
func Reconcile(previous Assertion, computation Computation) Decision {
	if computation.Evaluation != "valid" || computation.Completeness != "complete_as_of_frontier" {
		return reconcileUncertain(previous, computation.Key)
	}
	if !computation.Present {
		return reconcileAbsent(previous, computation.Key)
	}
	return reconcilePresent(previous, computation)
}

// reconcileUncertain keeps the prior value and availability: failed, invalid or
// incomplete evidence never retracts and never supplies a new value.
func reconcileUncertain(previous Assertion, key Key) Decision {
	next := Assertion{Revision: previous.Revision, PublicationID: previous.PublicationID, Status: "unresolved", Value: previous.Value, ValueAvailable: previous.ValueAvailable}
	if previous.Status == "unresolved" {
		return Decision{Assertion: next}
	}
	return Decision{Assertion: next, Intention: &Intention{Key: key, ExpectedRevision: previous.Revision, Op: "status", Status: "unresolved"}}
}

// reconcileAbsent is complete valid absence: the assertion is retracted and its
// value is unavailable.
func reconcileAbsent(previous Assertion, key Key) Decision {
	next := Assertion{Revision: previous.Revision, PublicationID: previous.PublicationID, Status: "retracted"}
	if previous.Status == "retracted" {
		return Decision{Assertion: next}
	}
	return Decision{Assertion: next, Intention: &Intention{Key: key, ExpectedRevision: previous.Revision, Op: "retract"}}
}

// reconcilePresent is complete valid presence: the computed value becomes current.
// Only unchanged current content needs no intention; recovering unresolved content
// with the same value needs a status change, not a new value.
func reconcilePresent(previous Assertion, computation Computation) Decision {
	next := Assertion{Revision: previous.Revision, PublicationID: previous.PublicationID, Status: "current", Value: computation.Value, ValueAvailable: true}
	intention := &Intention{Key: computation.Key, ExpectedRevision: previous.Revision}
	switch {
	case !previous.ValueAvailable || previous.Status == "retracted" || previous.Value != computation.Value:
		intention.Op, intention.Value = "upsert", computation.Value
	case previous.Status == "current":
		return Decision{Assertion: next}
	default:
		intention.Op, intention.Status = "status", "current"
	}
	return Decision{Assertion: next, Intention: intention}
}
