package submission

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"sort"
	"strings"
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
	// reservedRecords and reservedBytes total reserveOf over all entries, kept by fold.
	reservedRecords int
	reservedBytes   int
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
		if !found || rec.Event != nil || e.facts.Admission != "admitted" || rec.Version != e.facts.Version+1 || rec.Decision != "" || !text(rec.Attempt) || len(rec.Attempt) > maxAttemptBytes || !validOutcome(rec.Evaluation) || rec.Evidence == "" {
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

// maxAttemptBytes bounds the only caller-supplied text in an evaluation record, so the
// worst-case size of a reserved evaluation is known when the receipt is accepted.
const maxAttemptBytes = 128

const zeroUUID = "00000000-0000-0000-0000-000000000000"

// Worst-case encoded sizes of the two transitions an entry can still owe. Every field is
// at its largest: maximal sequence and version numbers, a canonical UUID, and an attempt
// of maxAttemptBytes '<' characters, which JSON escapes to six bytes each.
var (
	admissionReserveBytes = encodedBytes(record{Format: recordFormat, Sequence: math.MaxInt, Kind: "admission",
		EventID: zeroUUID, Version: math.MaxUint64, Decision: "unresolved"})
	evaluationReserveBytes = encodedBytes(record{Format: recordFormat, Sequence: math.MaxInt, Kind: "evaluation",
		EventID: zeroUUID, Version: math.MaxUint64, Attempt: strings.Repeat("<", maxAttemptBytes),
		Evaluation: "evaluated", Evidence: zeroUUID})
)

func encodedBytes(rec record) int {
	b, err := json.Marshal(rec)
	if err != nil {
		panic(err) // a fixed template of plain strings and numbers cannot fail to encode
	}
	return len(b)
}

// reserveOf returns the record slots and encoded bytes an entry still owes: its admission
// decision while pending, and its first evaluation until a terminal admission rejects it.
func reserveOf(f Facts) (records, bytes int) {
	if pendingAdmission(f.Admission) {
		records++
		bytes += admissionReserveBytes
	}
	if f.Admission != "rejected" && f.Evaluation == "not-established" {
		records++
		bytes += evaluationReserveBytes
	}
	return records, bytes
}

// fits reports ErrCapacity unless appending rec keeps every owed transition within the
// record and byte bounds. Ordinary writes therefore cannot consume the room promised to
// pending admissions or first evaluations.
func (st *state) fits(rec record, size int, l Limits) error {
	owedRecords, owedBytes := st.owedAfter(rec)
	if st.records+1+owedRecords > l.MaxEntries || st.bytes+size+owedBytes > l.MaxBytes {
		return ErrCapacity
	}
	return nil
}

// owedAfter returns the reservation totals that remain owed once rec is appended.
// Only the entry rec touches changes, so its old reservation is replaced by its new one.
func (st *state) owedAfter(rec record) (int, int) {
	var cur Facts
	var ev storedEvent
	oldRecords, oldBytes := 0, 0
	if e, ok := st.entries[rec.EventID]; ok {
		cur, ev = e.facts, e.ev
		oldRecords, oldBytes = reserveOf(cur)
	}
	if rec.Kind == "receipt" {
		ev = *rec.Event
	}
	newRecords, newBytes := reserveOf(nextFacts(st.generation, ev, cur, rec))
	return st.reservedRecords - oldRecords + newRecords, st.reservedBytes - oldBytes + newBytes
}

// within checks replayed state against the configured bounds. Reopening with smaller
// bounds fails rather than keeping state that later writes could not fit.
func (st *state) within(l Limits) error {
	if len(st.entries) > l.MaxEvents || st.records+st.reservedRecords > l.MaxEntries || st.bytes+st.reservedBytes > l.MaxBytes {
		return ErrCapacity
	}
	for _, e := range st.entries {
		n, err := envelopeBytes(e.ev)
		if err != nil {
			return err
		}
		if n > l.MaxEventBytes {
			return ErrCapacity
		}
	}
	return nil
}

// envelopeBytes is the encoded size of a normalized envelope, which MaxEventBytes bounds.
func envelopeBytes(ev storedEvent) (int, error) {
	b, err := json.Marshal(ev)
	if err != nil {
		return 0, ErrUnavailable
	}
	return len(b), nil
}

// nextFacts returns the full facts after rec, which check has already accepted.
func nextFacts(generation string, ev storedEvent, cur Facts, rec record) Facts {
	switch rec.Kind {
	case "receipt":
		return Facts{
			EventID: rec.EventID, Generation: generation, Version: 1, Receipt: true,
			Admission: "pending", Evaluation: "not-established", Attribution: ev.Asserting,
		}
	case "admission":
		cur.Admission = rec.Decision
		cur.Version = rec.Version
	case "evaluation":
		cur.Evaluation = rec.Evaluation
		cur.Model = ev.Schema
		cur.Evidence = rec.Evidence
		cur.Version = rec.Version
	}
	return cur
}

// fold applies a record that check has already accepted.
func (st *state) fold(rec record, size int) {
	if rec.Kind == "receipt" {
		st.entries[rec.EventID] = &entry{ev: *rec.Event}
	}
	e := st.entries[rec.EventID]
	oldRecords, oldBytes := reserveOf(e.facts)
	e.facts = nextFacts(st.generation, e.ev, e.facts, rec)
	newRecords, newBytes := reserveOf(e.facts)
	st.reservedRecords += newRecords - oldRecords
	st.reservedBytes += newBytes - oldBytes
	if rec.Kind == "evaluation" {
		st.attempts[rec.Attempt] = rec.EventID
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

// validateEvent checks the identity fields a caller supplies before any host call.
func validateEvent(e perhap.Event, schema string) error {
	if !canonicalV6(e.ID) || !text(e.Context) || !text(e.EntityID) || !text(e.Type) || !lossless(e.CorrelationID) || !text(schema) {
		return ErrInvalid
	}
	// Causation links are codec-owned opaque references, so only their text is checked.
	if e.CausationID != nil && !text(*e.CausationID) {
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

// lossless requires a string that survives storage as valid Unicode; it may be empty.
func lossless(s string) bool {
	return utf8.ValidString(s)
}

// text requires a nonempty string that survives lossless storage as valid Unicode.
func text(s string) bool {
	return s != "" && lossless(s)
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
