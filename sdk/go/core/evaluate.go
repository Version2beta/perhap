package core

import (
	"container/heap"
	"errors"
	"sort"
	"strconv"
	"strings"
)

// Evaluate reconstructs the finite supplied admitted history under the identified
// rule version. See SPEC.md for ordering, failure, coverage and ownership rules.
func Evaluate[S any](q Request, m Model[S]) Result[S] {
	ev := &evaluator[S]{
		q: q,
		m: withMissingCallbacks(m),
		res: Result[S]{
			Context:         q.Context,
			Scope:           q.Scope,
			RuleVersion:     q.RuleVersion,
			Frontier:        q.Frontier,
			History:         []Event{},
			Attempted:       []string{},
			Blocked:         []string{},
			Successful:      []string{},
			LastValidPrefix: []string{},
			Diagnostics:     []Diagnostic{},
		},
	}
	ev.admit()
	ev.link()
	ev.initialize()
	ev.run()
	ev.settle()
	if q.Frontier.Gap {
		ev.diag("", "coverage", "history_gap", nil)
	}
	ev.selectFinal()
	ev.res.Completeness = ev.completeness()
	ev.res.Evaluation = ev.evaluation()
	ev.collectBlocked()
	return ev.res
}

var errMissingCallback = errors.New("core: model callback is nil")

// withMissingCallbacks turns an absent callback into an explicit callback failure
// so that missing behavior never produces a fabricated valid state.
func withMissingCallbacks[S any](m Model[S]) Model[S] {
	var zero S
	if m.Initialize == nil {
		m.Initialize = func() (S, error) { return zero, errMissingCallback }
	}
	if m.Clone == nil {
		m.Clone = func(S) (S, error) { return zero, errMissingCallback }
	}
	if m.Reduce == nil {
		m.Reduce = func(S, Event) (S, error) { return zero, errMissingCallback }
	}
	if m.Validate == nil {
		m.Validate = func(S) (bool, error) { return false, errMissingCallback }
	}
	if m.Select == nil {
		m.Select = func(S, []Event) (Selection, error) { return Selection{}, errMissingCallback }
	}
	return m
}

type nodeStatus int

const (
	statusPending nodeStatus = iota
	statusSucceeded
	statusFailed
	statusBlocked
)

// edge points from a parent to a child. requiresSuccess is the child's dependency
// class: an occurrence-only child may proceed after its parent's reduction failed.
type edge struct {
	to              int
	requiresSuccess bool
}

// node is one admitted event during a single replay.
type node struct {
	event             Event
	status            nodeStatus
	intrinsic         bool // missing, unknown or unsatisfied dependency
	pending           int  // local parent edges not yet released
	parents           []int
	children          []edge
	blockedByParent   bool // a dependency-blocked parent was released
	blockedBySuccess  bool // a requires-success parent failed
	dependencyBlocked bool // blocked by history coverage, not by initialization
}

// evaluator holds one replay. It is created per Evaluate call and is never shared.
type evaluator[S any] struct {
	q            Request
	m            Model[S]
	res          Result[S]
	nodes        []*node // sorted by ID bytes, so index order is ID order
	ready        readyHeap
	blocking     []int // nodes to resolve as blocked before the next ready node
	successful   []Event
	processable  bool // working state exists and initial validation returned without error
	workingValid bool
	failed       bool // sticky: any callback or clone failure
}

// readyHeap is a min-heap of node indices; index order equals ID order.
type readyHeap []int

func (h readyHeap) Len() int           { return len(h) }
func (h readyHeap) Less(i, j int) bool { return h[i] < h[j] }
func (h readyHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *readyHeap) Push(x any)        { *h = append(*h, x.(int)) }
func (h *readyHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}

// admit keeps the first record for each ID, reports conflicting retries, and
// orders history by ID bytes with copied slice fields.
func (ev *evaluator[S]) admit() {
	seen := map[string]Event{}
	conflicted := map[string]bool{}
	admitted := []Event{}
	for _, e := range ev.q.Events {
		prior, ok := seen[e.ID]
		if !ok {
			seen[e.ID] = e
			admitted = append(admitted, copyEvent(e))
			continue
		}
		if !sameEvent(prior, e) && !conflicted[e.ID] {
			conflicted[e.ID] = true
			ev.diag(e.ID, "admission", "conflicting_id", nil)
		}
	}
	sort.Slice(admitted, func(i, j int) bool { return admitted[i].ID < admitted[j].ID })
	ev.res.History = admitted
	ev.nodes = make([]*node, len(admitted))
	for i, e := range admitted {
		ev.nodes[i] = &node{event: e}
	}
}

// link builds local dependency edges. Trace-external references are ignored; a
// missing, unknown, or unsatisfied required-external dependency is intrinsic.
func (ev *evaluator[S]) link() {
	index := make(map[string]int, len(ev.nodes))
	for i, n := range ev.nodes {
		index[n.event.ID] = i
	}
	for i, n := range ev.nodes {
		for _, d := range n.event.Dependencies {
			switch d.Kind {
			case "local":
				p, ok := index[d.ID]
				if !ok {
					n.intrinsic = true
				} else {
					n.pending++
					n.parents = append(n.parents, p)
					ev.nodes[p].children = append(ev.nodes[p].children, edge{to: i, requiresSuccess: d.RequiresSuccess})
				}
			case "trace-external":
				// Trace-only references neither order nor block local history.
			case "required-external":
				if !d.Satisfied {
					n.intrinsic = true
				}
			default:
				n.intrinsic = true
			}
		}
	}
}

// initialize builds the initial working state and its last-valid state.
// Initializer or initial-validator failure leaves ordinary events unprocessed.
func (ev *evaluator[S]) initialize() {
	s, err := ev.m.Initialize()
	if err != nil {
		ev.fail("", "initialize", "initializer_failed", err)
		return
	}
	w, ok := ev.cloneState(s, "", "initialize")
	if !ok {
		return
	}
	ev.res.Working, ev.res.WorkingAvailable = w, true
	check, ok := ev.cloneState(w, "", "initialize")
	if !ok {
		return
	}
	valid, err := ev.m.Validate(check)
	if err != nil {
		ev.fail("", "initialize", "initial_validator_failed", err)
		return
	}
	ev.processable = true
	ev.workingValid = valid
	if !valid {
		ev.diag("", "initialize", "initial_domain_invalid", nil)
		return
	}
	ev.commitLastValid(w, "", "initialize")
}

// run processes nodes whose local parents are all resolved, selecting the
// byte-smallest ready ID each time. Blocked nodes are resolved as they become
// determined; nodes left pending are settled after the loop.
func (ev *evaluator[S]) run() {
	for i, n := range ev.nodes {
		if n.pending == 0 {
			ev.enqueue(i)
		}
	}
	for {
		for len(ev.blocking) > 0 {
			i := ev.blocking[len(ev.blocking)-1]
			ev.blocking = ev.blocking[:len(ev.blocking)-1]
			ev.block(i)
		}
		if ev.ready.Len() == 0 {
			return
		}
		ev.process(heap.Pop(&ev.ready).(int))
	}
}

func (ev *evaluator[S]) enqueue(i int) {
	n := ev.nodes[i]
	if ev.processable && !n.intrinsic && !n.blockedByParent && !n.blockedBySuccess {
		heap.Push(&ev.ready, i)
		return
	}
	ev.blocking = append(ev.blocking, i)
}

func (ev *evaluator[S]) block(i int) {
	n := ev.nodes[i]
	n.status = statusBlocked
	n.dependencyBlocked = n.intrinsic || n.blockedByParent || n.blockedBySuccess
	if n.dependencyBlocked {
		ev.diag(n.event.ID, "dependency", "dependency_unresolved", nil)
	}
	ev.release(i)
}

// release tells each child that this node has resolved, and enqueues children
// whose local parents are all resolved.
func (ev *evaluator[S]) release(i int) {
	n := ev.nodes[i]
	for _, e := range n.children {
		c := ev.nodes[e.to]
		c.pending--
		c.blockedByParent = c.blockedByParent || (n.status == statusBlocked && n.dependencyBlocked)
		c.blockedBySuccess = c.blockedBySuccess || (n.status == statusFailed && e.requiresSuccess)
		if c.pending == 0 {
			ev.enqueue(e.to)
		}
	}
}

func (ev *evaluator[S]) process(i int) {
	n := ev.nodes[i]
	ev.res.Attempted = append(ev.res.Attempted, n.event.ID)
	if ev.apply(n.event) {
		n.status = statusSucceeded
		ev.res.Successful = append(ev.res.Successful, n.event.ID)
		ev.successful = append(ev.successful, copyEvent(n.event))
	} else {
		n.status = statusFailed
	}
	ev.release(i)
}

// apply reduces and validates one candidate. A failed candidate is discarded and
// the working state is unchanged. A domain-invalid candidate replaces working.
func (ev *evaluator[S]) apply(e Event) bool {
	w, ok := ev.cloneState(ev.res.Working, e.ID, "reduce")
	if !ok {
		return false
	}
	candidate, err := ev.m.Reduce(w, copyEvent(e))
	if err != nil {
		ev.fail(e.ID, "reduce", "reducer_failed", err)
		return false
	}
	check, ok := ev.cloneState(candidate, e.ID, "validate")
	if !ok {
		return false
	}
	valid, err := ev.m.Validate(check)
	if err != nil {
		ev.fail(e.ID, "validate", "validator_failed", err)
		return false
	}
	if valid {
		if !ev.commitLastValid(candidate, e.ID, "last_valid") {
			return false
		}
	} else {
		ev.diag(e.ID, "validate", "domain_invalid", nil)
	}
	ev.res.Working = candidate
	ev.workingValid = valid
	return true
}

// commitLastValid stores an independent copy of s as last-valid and records the
// attempted prefix that produced it.
func (ev *evaluator[S]) commitLastValid(s S, eventID, stage string) bool {
	lv, ok := ev.cloneState(s, eventID, stage)
	if !ok {
		return false
	}
	ev.res.LastValid, ev.res.LastValidAvailable = lv, true
	ev.res.LastValidPrefix = copyStrings(ev.res.Attempted)
	return true
}

// settle blocks nodes that never became ready. Every such node is upstream of,
// or inside, a causal cycle; a node on a cycle gets causal_cycle.
func (ev *evaluator[S]) settle() {
	pending := []int{}
	for i, n := range ev.nodes {
		if n.status == statusPending {
			pending = append(pending, i)
		}
	}
	onCycle := make([]bool, len(pending))
	for k, i := range pending {
		onCycle[k] = ev.reachesItself(i)
	}
	for k, i := range pending {
		n := ev.nodes[i]
		n.status = statusBlocked
		n.dependencyBlocked = true
		condition := "dependency_unresolved"
		if onCycle[k] {
			condition = "causal_cycle"
		}
		ev.diag(n.event.ID, "dependency", condition, nil)
	}
}

// reachesItself reports whether start is reachable from its own pending parents.
func (ev *evaluator[S]) reachesItself(start int) bool {
	seen := map[int]bool{}
	stack := append([]int(nil), ev.nodes[start].parents...)
	for len(stack) > 0 {
		p := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if p == start {
			return true
		}
		if seen[p] || ev.nodes[p].status != statusPending {
			continue
		}
		seen[p] = true
		stack = append(stack, ev.nodes[p].parents...)
	}
	return false
}

// selectFinal runs the pure final selection on isolated working state and
// successful events. It is skipped when no working state was established.
func (ev *evaluator[S]) selectFinal() {
	if !ev.processable {
		return
	}
	state, ok := ev.cloneState(ev.res.Working, "", "select")
	if !ok {
		return
	}
	events := make([]Event, len(ev.successful))
	for i, e := range ev.successful {
		events[i] = copyEvent(e)
	}
	sel, err := ev.m.Select(state, events)
	if err != nil {
		ev.fail("", "select", "selector_failed", err)
		return
	}
	ev.res.Selection = sel
}

// completeness: incompleteness wins; otherwise certified continuity through an
// identified boundary is complete; otherwise coverage is unavailable.
func (ev *evaluator[S]) completeness() string {
	dependencyGap := false
	for _, n := range ev.nodes {
		dependencyGap = dependencyGap || n.dependencyBlocked
	}
	switch {
	case ev.q.Frontier.Gap || dependencyGap:
		return "incomplete"
	case ev.q.Frontier.Certified && ev.q.Frontier.Boundary != "":
		return "complete_as_of_frontier"
	default:
		return "unavailable"
	}
}

func (ev *evaluator[S]) evaluation() string {
	switch {
	case ev.failed:
		return "failed"
	case !ev.workingValid:
		return "invalid"
	default:
		return "valid"
	}
}

func (ev *evaluator[S]) collectBlocked() {
	for _, n := range ev.nodes {
		if n.status == statusBlocked {
			ev.res.Blocked = append(ev.res.Blocked, n.event.ID)
		}
	}
}

// cloneState isolates mutable state before it is handed to a callback or stored.
func (ev *evaluator[S]) cloneState(s S, eventID, stage string) (S, bool) {
	c, err := ev.m.Clone(s)
	if err != nil {
		ev.fail(eventID, stage, "clone_failed", err)
		var zero S
		return zero, false
	}
	return c, true
}

func (ev *evaluator[S]) fail(eventID, stage, condition string, err error) {
	ev.failed = true
	ev.diag(eventID, stage, condition, err)
}

func (ev *evaluator[S]) diag(eventID, stage, condition string, err error) {
	ev.res.Diagnostics = append(ev.res.Diagnostics, Diagnostic{
		Key:         diagnosticKey(ev.q.Context, ev.q.Scope, ev.q.RuleVersion, eventID, stage, condition),
		EventID:     eventID,
		Stage:       stage,
		Condition:   condition,
		RuleVersion: ev.q.RuleVersion,
		Err:         err,
	})
}

// diagnosticKey is stable across unrelated history: it depends only on the
// context, scope, rule version, event, stage and condition.
func diagnosticKey(parts ...string) string {
	quoted := make([]string, len(parts))
	for i, p := range parts {
		quoted[i] = strconv.Quote(p)
	}
	return "core:" + strings.Join(quoted, ":")
}

func copyStrings(s []string) []string {
	if s == nil {
		return nil
	}
	return append(make([]string, 0, len(s)), s...)
}

func copyDependencies(d []Dependency) []Dependency {
	if d == nil {
		return nil
	}
	return append(make([]Dependency, 0, len(d)), d...)
}

func copyEvent(e Event) Event {
	e.Subjects = copyStrings(e.Subjects)
	e.Dependencies = copyDependencies(e.Dependencies)
	return e
}

// sameEvent reports whether two records with the same ID are an identical retry.
func sameEvent(a, b Event) bool {
	if a.ID != b.ID || a.Context != b.Context || a.Type != b.Type || a.Payload != b.Payload || a.AssertedBy != b.AssertedBy {
		return false
	}
	if len(a.Subjects) != len(b.Subjects) {
		return false
	}
	for i := range a.Subjects {
		if a.Subjects[i] != b.Subjects[i] {
			return false
		}
	}
	if len(a.Dependencies) != len(b.Dependencies) {
		return false
	}
	for i := range a.Dependencies {
		if a.Dependencies[i] != b.Dependencies[i] {
			return false
		}
	}
	return true
}
