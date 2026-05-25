package perhap

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/nats-io/nats.go"
)

// deepCopy returns a deep copy of v by round-tripping through JSON.
// This ensures that reference types inside M (maps, slices) are not shared
// between the copy and the original, preventing reducer mutations from
// corrupting the stored snapshot.
func deepCopy[M any](v M) M {
	data, err := json.Marshal(v)
	if err != nil {
		return v // fallback: accept shared references if marshal fails
	}
	var out M
	json.Unmarshal(data, &out) //nolint:errcheck
	return out
}

type supervisor[M any] struct {
	entityID    string
	nc          *nats.Conn
	config      ServiceConfig[M]
	onPassivate func()

	// State
	snapshot     M
	snapshotUUID string
	window       []Event // sorted by UUID
	windowUUIDs  map[string]bool
	lastValid    M
	valid        bool
	pending      bool

	inbox      chan Event
	idleTimer  *time.Timer
	closeTimer *time.Timer
}

func newSupervisor[M any](entityID string, nc *nats.Conn, config ServiceConfig[M], onPassivate func()) *supervisor[M] {
	var zero M
	return newSupervisorWithSnapshot(entityID, nc, config, zero, "", onPassivate)
}

// newSupervisorWithSnapshot creates a supervisor pre-seeded with a replayed snapshot so that
// new events arriving after passivation or a server restart are applied on top of existing state.
func newSupervisorWithSnapshot[M any](entityID string, nc *nats.Conn, config ServiceConfig[M], snapshot M, snapshotUUID string, onPassivate func()) *supervisor[M] {
	sv := &supervisor[M]{
		entityID:     entityID,
		nc:           nc,
		config:       config,
		onPassivate:  onPassivate,
		snapshot:     snapshot,
		snapshotUUID: snapshotUUID,
		lastValid:    snapshot,
		windowUUIDs:  make(map[string]bool),
		valid:        true,
		pending:      false,
		inbox:        make(chan Event, 64),
	}
	go sv.run()
	return sv
}

func (sv *supervisor[M]) send(event Event) {
	sv.inbox <- event
}

func (sv *supervisor[M]) run() {
	sv.idleTimer = time.NewTimer(sv.config.WindowIdleClose)
	sv.closeTimer = time.NewTimer(sv.config.PassivateTimeout)
	sv.closeTimer.Stop()

	for {
		select {
		case event := <-sv.inbox:
			sv.processEvent(event)
			// Reset idle timer — window is active.
			sv.idleTimer.Reset(sv.config.WindowIdleClose)
			sv.closeTimer.Stop()
			if !sv.pending {
				sv.setPending(true)
			}

		case <-sv.idleTimer.C:
			// Window idle — close it.
			if sv.pending {
				sv.setPending(false)
				sv.publishModel()
			}
			sv.closeTimer.Reset(sv.config.PassivateTimeout)

		case <-sv.closeTimer.C:
			// Passivate.
			log.Printf("perhap: supervisor %s/%s passivating", sv.config.Context, sv.entityID)
			sv.onPassivate()
			return
		}
	}
}

func (sv *supervisor[M]) processEvent(event Event) {
	// Deduplication: skip if already in window or before snapshot.
	if sv.windowUUIDs[event.ID] {
		log.Printf("perhap: DEDUP %s/%s type=%s id=%s", sv.config.Context, sv.entityID, event.Type, event.ID)
		return
	}
	if sv.snapshotUUID != "" && event.ID <= sv.snapshotUUID {
		log.Printf("perhap: DEDUP (before snapshot) %s/%s type=%s id=%s snapshot=%s", sv.config.Context, sv.entityID, event.Type, event.ID, sv.snapshotUUID)
		return
	}

	log.Printf("perhap: processEvent %s/%s type=%s id=%s", sv.config.Context, sv.entityID, event.Type, event.ID)

	// Insert into window at correct UUID position.
	sv.window = append(sv.window, event)
	sv.windowUUIDs[event.ID] = true
	sort.Slice(sv.window, func(i, j int) bool {
		return sv.window[i].ID < sv.window[j].ID
	})

	// Re-reduce from snapshot through full window.
	// Deep-copy via JSON so that Go map references inside M are not shared with
	// sv.snapshot — reducer mutations (map writes) must not corrupt the stored snapshot.
	model := deepCopy(sv.snapshot)
	var newEvents []Event
	for _, e := range sv.window {
		var evts []Event
		var err error
		model, evts, err = sv.config.Reducer(e.Type, model, e)
		if err != nil {
			log.Printf("perhap: reducer error %s/%s type=%s id=%s: %v — model unchanged",
				sv.config.Context, sv.entityID, e.Type, e.ID, err)
		}
		newEvents = append(newEvents, evts...)
	}

	// Validate.
	errs := sv.config.Validator(model)
	if len(errs) == 0 {
		sv.lastValid = model
		sv.valid = true
	} else {
		log.Printf("perhap: VALIDATION FAILED %s/%s: %v", sv.config.Context, sv.entityID, errs)
		sv.valid = false
		sv.emitException(errs)
	}

	// Publish any new events from the reducer.
	for _, newEvent := range newEvents {
		sv.publishEvent(newEvent)
	}

	// Advance window if over size threshold.
	sv.maybeAdvanceSnapshot()

	// Publish model update.
	log.Printf("perhap: publishModel %s/%s reducedThrough=%s window=%d", sv.config.Context, sv.entityID, sv.reducedThrough(), len(sv.window))
	sv.publishModel()
}

func (sv *supervisor[M]) maybeAdvanceSnapshot() {
	if len(sv.window) <= sv.config.WindowSize {
		return
	}
	n := (sv.config.WindowSize + 1) / 2 // ceil(WindowSize/2)
	toFold := sv.window[:n]
	model := deepCopy(sv.snapshot)
	for _, e := range toFold {
		model, _, _ = sv.config.Reducer(e.Type, model, e)
		delete(sv.windowUUIDs, e.ID)
	}
	sv.snapshot = model
	sv.snapshotUUID = toFold[len(toFold)-1].ID
	sv.window = sv.window[n:]
}

func (sv *supervisor[M]) setPending(p bool) {
	sv.pending = p
}

func (sv *supervisor[M]) reducedThrough() string {
	if len(sv.window) == 0 {
		return sv.snapshotUUID
	}
	return sv.window[len(sv.window)-1].ID
}

func (sv *supervisor[M]) windowUUIDList() []string {
	uuids := make([]string, len(sv.window))
	for i, e := range sv.window {
		uuids[i] = e.ID
	}
	return uuids
}

func (sv *supervisor[M]) currentModel() M {
	return sv.lastValid
}

func (sv *supervisor[M]) currentModelUpdate() []byte {
	data, _ := json.Marshal(sv.currentModel())
	update := ModelUpdate{
		EntityID:       sv.entityID,
		Context:        sv.config.Context,
		Domain:         sv.config.Domain,
		SnapshotUUID:   sv.snapshotUUID,
		Window:         sv.windowUUIDList(),
		ReducedThrough: sv.reducedThrough(),
		Pending:        sv.pending,
		Valid:          sv.valid,
		Reduction:      data,
	}
	b, _ := json.Marshal(update)
	return b
}

func (sv *supervisor[M]) publishModel() {
	subject := fmt.Sprintf("perhap.%s.%s._model", sv.config.Context, sv.entityID)
	sv.nc.Publish(subject, sv.currentModelUpdate())
}

func (sv *supervisor[M]) publishEvent(event Event) {
	sv.publishStoredEvent(event)
}

func (sv *supervisor[M]) publishStoredEvent(event Event) {
	subject := fmt.Sprintf("perhap.%s.%s.%s", sv.config.Context, event.EntityID, event.Type)
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("perhap: marshal event: %v", err)
		return
	}
	msg := nats.NewMsg(subject)
	msg.Data = data
	msg.Header.Set("Nats-Msg-Id", event.ID)
	if _, err := sv.nc.JetStream(); err == nil {
		js, _ := sv.nc.JetStream()
		if _, err := js.PublishMsg(msg); err != nil {
			log.Printf("perhap: publish event %s: %v", event.Type, err)
		}
	}
}

func (sv *supervisor[M]) emitException(errs []ValidationError) {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}

	uuids := make([]string, len(sv.window))
	for i, e := range sv.window {
		uuids[i] = e.ID
	}

	type exceptionData struct {
		WindowUUIDs      []string `json:"window_uuids"`
		ValidationErrors []string `json:"validation_errors"`
		LastValidUUID    string   `json:"last_valid_uuid"`
	}
	data, _ := json.Marshal(exceptionData{
		WindowUUIDs:      uuids,
		ValidationErrors: msgs,
		LastValidUUID:    sv.reducedThrough(),
	})

	exID := NewEventID()
	event := Event{
		ID:       exID,
		EntityID: sv.entityID,
		Context:  sv.config.Context,
		Type:     "_exception",
		Data:     data,
	}
	sv.publishStoredEvent(event)
}
