package submission

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"unicode/utf8"

	"github.com/google/uuid"
	perhap "github.com/version2beta/perhap/sdk/go"
	"github.com/version2beta/perhap/sdk/go/core"
)

const recordFormat = 1

// errRecord marks a record that does not follow the authoritative transition rules.
var errRecord = errors.New("submission: invalid record")

// storedEvent is the normalized immutable semantic identity of one event. Data holds
// the canonical bytes used both for equality and for CORE evaluation.
type storedEvent struct {
	ID            string  `json:"id"`
	Context       string  `json:"context"`
	EntityID      string  `json:"entity_id"`
	Type          string  `json:"type"`
	CausationID   *string `json:"causation_id,omitempty"`
	CorrelationID string  `json:"correlation_id"`
	Schema        string  `json:"schema"`
	Asserting     string  `json:"asserting"`
	Data          []byte  `json:"data"`
}

// record is one private, versioned, append-only log entry. Each record carries exactly
// one fact transition for one event.
type record struct {
	Format     int          `json:"format"`
	Sequence   int          `json:"sequence"`
	Kind       string       `json:"kind"`
	EventID    string       `json:"event_id"`
	Version    uint64       `json:"version"`
	Event      *storedEvent `json:"event,omitempty"`
	Decision   string       `json:"decision,omitempty"`
	Attempt    string       `json:"attempt,omitempty"`
	Evaluation string       `json:"evaluation,omitempty"`
	Evidence   string       `json:"evidence,omitempty"`
}

// entry is the replayed state of one event: its identity and its latest full facts.
type entry struct {
	ev    storedEvent
	facts Facts
}

// state is the validated replay of every durable record.
type state struct {
	generation string
	entries    map[string]*entry
	attempts   map[string]string
	records    int
	bytes      int
}

func newState(generation string) *state {
	return &state{generation: generation, entries: map[string]*entry{}, attempts: map[string]string{}}
}

// check validates a record against the state built so far without mutating it.
func (st *state) check(rec record) error {
	if rec.Format != recordFormat || rec.Sequence != st.records+1 {
		return errRecord
	}
	e, found := st.entries[rec.EventID]
	switch rec.Kind {
	case "receipt":
		if found || rec.Event == nil || rec.Version != 1 || rec.EventID != rec.Event.ID || rec.Decision != "" || rec.Attempt != "" || rec.Evaluation != "" || rec.Evidence != "" {
			return errRecord
		}
		if _, err := newStoredEvent(rec.Event.event(), rec.Event.Schema, rec.Event.Asserting); err != nil {
			return errRecord
		}
	case "admission":
		if !found || rec.Event != nil || rec.Version != e.facts.Version+1 || !pendingAdmission(e.facts.Admission) || !validDecision(rec.Decision) || rec.Decision == e.facts.Admission || rec.Attempt != "" || rec.Evaluation != "" || rec.Evidence != "" {
			return errRecord
		}
	case "evaluation":
		if !found || rec.Event != nil || e.facts.Admission != "admitted" || rec.Version != e.facts.Version+1 || rec.Decision != "" || !text(rec.Attempt) || !validOutcome(rec.Evaluation) || rec.Evidence == "" {
			return errRecord
		}
		if _, used := st.attempts[rec.Attempt]; used {
			return errRecord
		}
	default:
		return errRecord
	}
	return nil
}

// fits enforces the entry and byte bounds for one more record of the given size.
func (st *state) fits(size int, l Limits) error {
	if st.records+1 > l.MaxEntries || st.bytes+size > l.MaxBytes {
		return ErrCapacity
	}
	return nil
}

// fold applies a record that check has already accepted.
func (st *state) fold(rec record, size int) {
	switch rec.Kind {
	case "receipt":
		st.entries[rec.EventID] = &entry{ev: *rec.Event, facts: Facts{
			EventID: rec.EventID, Generation: st.generation, Version: 1, Receipt: true,
			Admission: "pending", Evaluation: "not-established", Attribution: rec.Event.Asserting,
		}}
	case "admission":
		e := st.entries[rec.EventID]
		e.facts.Admission = rec.Decision
		e.facts.Version = rec.Version
	case "evaluation":
		e := st.entries[rec.EventID]
		st.attempts[rec.Attempt] = rec.EventID
		e.facts.Evaluation = rec.Evaluation
		e.facts.Model = e.ev.Schema
		e.facts.Evidence = rec.Evidence
		e.facts.Version = rec.Version
	}
	st.records++
	st.bytes += size
}

// heads returns recovery evidence: one latest version per event, in ID byte order.
func (st *state) heads() []Head {
	out := make([]Head, 0, len(st.entries))
	for id, e := range st.entries {
		out = append(out, Head{EventID: id, Version: e.facts.Version})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EventID < out[j].EventID })
	return out
}

// history returns the admitted events of one context, entity and schema in ID byte order.
func (st *state) history(context, entity, schema string) []*entry {
	var out []*entry
	for _, e := range st.entries {
		if e.facts.Admission == "admitted" && e.ev.Context == context && e.ev.EntityID == entity && e.ev.Schema == schema {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ev.ID < out[j].ev.ID })
	return out
}

// evaluateAdmitted runs the registered model through CORE over the finite admitted
// history. The target must be part of hist. The codec is trusted host code.
func evaluateAdmitted(codec Codec, target *entry, hist []*entry, boundary string) (core.Result[int64], error) {
	events := make([]core.Event, 0, len(hist))
	for _, h := range hist {
		ce, err := codec.CoreEvent(h.ev.event(), h.ev.Asserting)
		if err != nil || ce.ID != h.ev.ID {
			return core.Result[int64]{}, ErrUnavailable
		}
		events = append(events, ce)
	}
	req := core.Request{
		Context:     target.ev.Context,
		Scope:       target.ev.EntityID,
		RuleVersion: target.ev.Schema,
		Events:      events,
		Frontier:    core.Frontier{Boundary: boundary, Certified: true},
	}
	return core.Evaluate(req, codec.Model()), nil
}

type evidenceDoc struct {
	Events       []storedEvent `json:"events"`
	Evaluation   string        `json:"evaluation"`
	Completeness string        `json:"completeness"`
	Attempted    []string      `json:"attempted"`
	Blocked      []string      `json:"blocked"`
	Successful   []string      `json:"successful"`
	Diagnostics  []string      `json:"diagnostics"`
}

// evidenceOf digests the exact admitted history and the CORE outcome labels. It does
// not digest private working state, which the store cannot encode generically.
func evidenceOf(hist []*entry, res core.Result[int64]) (string, error) {
	doc := evidenceDoc{
		Events:       make([]storedEvent, 0, len(hist)),
		Evaluation:   res.Evaluation,
		Completeness: res.Completeness,
		Attempted:    res.Attempted,
		Blocked:      res.Blocked,
		Successful:   res.Successful,
		Diagnostics:  make([]string, 0, len(res.Diagnostics)),
	}
	for _, h := range hist {
		doc.Events = append(doc.Events, h.ev)
	}
	for _, d := range res.Diagnostics {
		doc.Diagnostics = append(doc.Diagnostics, d.Key+"|"+d.Condition)
	}
	b, err := json.Marshal(doc)
	if err != nil {
		return "", ErrUnavailable
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// validateEvent checks the identity fields a caller supplies before any host call.
func validateEvent(e perhap.Event, schema string) error {
	if !canonicalV6(e.ID) || !text(e.Context) || !text(e.EntityID) || !text(e.Type) || !text(e.CorrelationID) || !text(schema) {
		return ErrInvalid
	}
	if e.CausationID != nil && !canonicalV6(*e.CausationID) {
		return ErrInvalid
	}
	return nil
}

// newStoredEvent copies the normalized event and its trusted schema and asserting identity.
func newStoredEvent(e perhap.Event, schema, asserting string) (storedEvent, error) {
	if err := validateEvent(e, schema); err != nil {
		return storedEvent{}, err
	}
	if !text(asserting) {
		return storedEvent{}, ErrDenied
	}
	return storedEvent{
		ID: e.ID, Context: e.Context, EntityID: e.EntityID, Type: e.Type,
		CausationID: copyPtr(e.CausationID), CorrelationID: e.CorrelationID,
		Schema: schema, Asserting: asserting, Data: append([]byte(nil), e.Data...),
	}, nil
}

// event returns a fresh copy so that callbacks cannot mutate stored state.
func (s storedEvent) event() perhap.Event {
	return perhap.Event{
		ID: s.ID, EntityID: s.EntityID, Context: s.Context, Type: s.Type,
		CausationID: copyPtr(s.CausationID), CorrelationID: s.CorrelationID,
		Data: append(json.RawMessage(nil), s.Data...),
	}
}

func cloneEvent(e perhap.Event) perhap.Event {
	e.CausationID = copyPtr(e.CausationID)
	e.Data = append(json.RawMessage(nil), e.Data...)
	return e
}

// sameIdentity reports whether normalization left every non-Data field unchanged.
func sameIdentity(a, b perhap.Event) bool {
	return a.ID == b.ID && a.Context == b.Context && a.EntityID == b.EntityID && a.Type == b.Type &&
		a.CorrelationID == b.CorrelationID && sameCause(a.CausationID, b.CausationID)
}

// sameSemantic compares every semantic field, including schema and trusted asserting
// identity. Transient binding and requester credentials are never stored.
func sameSemantic(a, b storedEvent) bool {
	return a.ID == b.ID && a.Context == b.Context && a.EntityID == b.EntityID && a.Type == b.Type &&
		a.CorrelationID == b.CorrelationID && sameCause(a.CausationID, b.CausationID) &&
		a.Schema == b.Schema && a.Asserting == b.Asserting && bytes.Equal(a.Data, b.Data)
}

func sameCause(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func copyPtr(p *string) *string {
	if p == nil {
		return nil
	}
	c := *p
	return &c
}

// text requires a nonempty string that survives lossless storage as valid Unicode.
func text(s string) bool {
	return s != "" && utf8.ValidString(s)
}

// canonicalV6 requires the lowercase hyphenated UUIDv6 form, so equal IDs compare bytewise.
func canonicalV6(s string) bool {
	u, err := uuid.Parse(s)
	return err == nil && u.Version() == 6 && u.String() == s
}

func validDecision(d string) bool {
	return d == "admitted" || d == "rejected" || d == "unresolved"
}

func pendingAdmission(s string) bool {
	return s == "pending" || s == "unresolved"
}

func validOutcome(s string) bool {
	return s == "evaluated" || s == "failed"
}
