package perhap

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Event is an immutable record of something that happened.
// The ID is UUIDv6, assigned at source. No timestamp field —
// the timestamp is extracted from the UUID.
type Event struct {
	ID            string          `json:"id"`
	EntityID      string          `json:"entity_id"`
	Context       string          `json:"context"`
	Type          string          `json:"type"`
	CausationID   *string         `json:"causation_id,omitempty"`
	CorrelationID string          `json:"correlation_id"`
	Data          json.RawMessage `json:"data"`
}

// Time extracts the timestamp encoded in the UUIDv6 ID.
func (e Event) Time() (time.Time, error) {
	id, err := uuid.Parse(e.ID)
	if err != nil {
		return time.Time{}, err
	}
	sec, nsec := id.Time().UnixTime()
	return time.Unix(sec, nsec), nil
}

// NewEventID returns a new UUIDv6 for use as an event ID.
// UUIDv6 provides 100ns timestamp resolution and lexicographic sort order.
func NewEventID() string {
	return uuid.Must(uuid.NewV6()).String()
}

// ValidationError describes a single domain invariant violation.
type ValidationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

// ModelUpdate is published to perhap.{context}.{entity_id}._model
// after every reduction.
type ModelUpdate struct {
	EntityID      string          `json:"entity_id"`
	Context       string          `json:"context"`
	Domain        string          `json:"domain"`
	SnapshotUUID  string          `json:"snapshot_uuid"`
	Window        []string        `json:"window"`
	ReducedThrough string         `json:"reduced_through"`
	Pending       bool            `json:"pending"`
	Valid         bool            `json:"valid"`
	Reduction     json.RawMessage `json:"reduction"`
}
