package perhap

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Reducer is a pure function: given an event type, current model, and event,
// return the new model and any new events to publish.
type Reducer[M any] func(eventType string, model M, event Event) (M, []Event)

// Validator checks domain invariants on a reduced model.
// Returns an empty slice when valid.
type Validator[M any] func(model M) []ValidationError

// LateEventPolicy declares how a domain handles events that arrive
// after their window has passed.
type LateEventPolicy int

const (
	LateEventReject          LateEventPolicy = iota // store + emit exception
	LateEventRebuild                                // rebuild from earlier snapshot
	LateEventAcceptWithFlag                         // apply at head, flag in metadata
)

// ServiceConfig wires together a domain service.
type ServiceConfig[M any] struct {
	Context          string
	Domain           string
	Reducer          Reducer[M]
	Validator        Validator[M]
	WindowSize       int           // max window events before snapshot advancement; default 100
	WindowIdleClose  time.Duration // idle before pending→false; default 500ms
	PassivateTimeout time.Duration // idle after window close before supervisor terminates; default 5m
	LateEventPolicy  LateEventPolicy
}

func (c *ServiceConfig[M]) withDefaults() {
	if c.WindowSize == 0 {
		c.WindowSize = 100
	}
	if c.WindowIdleClose == 0 {
		c.WindowIdleClose = 500 * time.Millisecond
	}
	if c.PassivateTimeout == 0 {
		c.PassivateTimeout = 5 * time.Minute
	}
}

// Service manages entity supervisors and handles NATS subscriptions
// for a single domain within a context.
type Service[M any] struct {
	nc          *nats.Conn
	js          nats.JetStreamContext
	config      ServiceConfig[M]
	supervisors map[string]*supervisor[M]
	mu          sync.Mutex
}

// NewService creates a service ready to run.
func NewService[M any](nc *nats.Conn, config ServiceConfig[M]) (*Service[M], error) {
	config.withDefaults()

	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("jetstream: %w", err)
	}

	// Ensure the stream exists for this context.
	streamName := "PERHAP_" + strings.ToUpper(config.Context)
	subjectFilter := "perhap." + config.Context + ".>"
	_, err = js.StreamInfo(streamName)
	if err != nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:       streamName,
			Subjects:   []string{subjectFilter},
			Storage:    nats.FileStorage,
			Replicas:   1,
			NoAck:      false,
		})
		if err != nil {
			return nil, fmt.Errorf("create stream: %w", err)
		}
		log.Printf("perhap: created stream %s", streamName)
	}

	return &Service[M]{
		nc:          nc,
		js:          js,
		config:      config,
		supervisors: make(map[string]*supervisor[M]),
	}, nil
}

// Run subscribes to domain events and blocks until ctx is cancelled.
func (s *Service[M]) Run(ctx context.Context) error {
	queueGroup := "perhap." + s.config.Context + "." + s.config.Domain
	subject := "perhap." + s.config.Context + ".*.>"

	sub, err := s.nc.QueueSubscribe(subject, queueGroup, func(msg *nats.Msg) {
		// Skip internal subjects (_model, _model.get)
		parts := strings.Split(msg.Subject, ".")
		if len(parts) < 4 {
			return
		}
		eventType := parts[len(parts)-1]
		if strings.HasPrefix(eventType, "_") {
			// Handle _model.get hydration requests
			if msg.Subject == "perhap."+s.config.Context+"."+parts[2]+"._model.get" {
				s.handleHydrationRequest(msg, parts[2])
			}
			return
		}

		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("perhap: unmarshal event: %v", err)
			msg.Nak()
			return
		}

		s.getOrCreateSupervisor(event.EntityID).send(event)
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	// Also subscribe to hydration requests.
	hydrationSubject := "perhap." + s.config.Context + ".*._model.get"
	hydSub, err := s.nc.Subscribe(hydrationSubject, func(msg *nats.Msg) {
		parts := strings.Split(msg.Subject, ".")
		if len(parts) >= 3 {
			s.handleHydrationRequest(msg, parts[2])
		}
	})
	if err != nil {
		sub.Unsubscribe()
		return fmt.Errorf("subscribe hydration: %w", err)
	}

	log.Printf("perhap: service %s/%s running", s.config.Context, s.config.Domain)
	<-ctx.Done()
	sub.Unsubscribe()
	hydSub.Unsubscribe()
	log.Printf("perhap: service %s/%s stopped", s.config.Context, s.config.Domain)
	return nil
}

func (s *Service[M]) getOrCreateSupervisor(entityID string) *supervisor[M] {
	s.mu.Lock()
	defer s.mu.Unlock()
	sv, ok := s.supervisors[entityID]
	if !ok {
		sv = newSupervisor(entityID, s.nc, s.config, func() {
			s.mu.Lock()
			delete(s.supervisors, entityID)
			s.mu.Unlock()
		})
		s.supervisors[entityID] = sv
	}
	return sv
}

func (s *Service[M]) handleHydrationRequest(msg *nats.Msg, entityID string) {
	s.mu.Lock()
	sv, ok := s.supervisors[entityID]
	s.mu.Unlock()

	if !ok {
		// No active supervisor — replay from JetStream.
		model, reducedThrough, err := s.replayFromJetStream(entityID)
		if err != nil {
			log.Printf("perhap: hydration replay %s/%s: %v", s.config.Context, entityID, err)
		}
		data, _ := json.Marshal(model)
		update := ModelUpdate{
			EntityID:       entityID,
			Context:        s.config.Context,
			Domain:         s.config.Domain,
			SnapshotUUID:   "",
			Window:         []string{},
			ReducedThrough: reducedThrough,
			Pending:        false,
			Valid:          true,
			Reduction:      data,
		}
		resp, _ := json.Marshal(update)
		msg.Respond(resp)
		return
	}

	resp := sv.currentModelUpdate()
	msg.Respond(resp)
}

// replayFromJetStream rebuilds entity state by consuming all stored events
// from the JetStream stream. Used for hydration when no supervisor is active.
func (s *Service[M]) replayFromJetStream(entityID string) (M, string, error) {
	var zero M
	filterSubject := fmt.Sprintf("perhap.%s.%s.*", s.config.Context, entityID)

	sub, err := s.js.SubscribeSync(filterSubject,
		nats.DeliverAll(),
		nats.AckNone(),
	)
	if err != nil {
		return zero, "", fmt.Errorf("subscribe for replay: %w", err)
	}
	defer sub.Drain()

	var events []Event
	for {
		// Use a short timeout: 200ms for first message (handles empty entity),
		// then rely on NumPending to detect the last message without extra wait.
		msg, err := sub.NextMsg(200 * time.Millisecond)
		if err != nil {
			break // timeout — no more messages
		}
		parts := strings.Split(msg.Subject, ".")
		if len(parts) < 4 {
			continue
		}
		eventType := parts[len(parts)-1]
		if strings.HasPrefix(eventType, "_") {
			// Skip _model, _model.get, _exception
			meta, _ := msg.Metadata()
			if meta != nil && meta.NumPending == 0 {
				break
			}
			continue
		}
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			continue
		}
		events = append(events, event)
		meta, _ := msg.Metadata()
		if meta != nil && meta.NumPending == 0 {
			break // consumed all stored messages
		}
	}

	if len(events) == 0 {
		return zero, "", nil
	}

	// Sort by UUID to guarantee correct reduction order.
	sort.Slice(events, func(i, j int) bool {
		return events[i].ID < events[j].ID
	})

	model := zero
	for _, e := range events {
		model, _ = s.config.Reducer(e.Type, model, e)
	}
	return model, events[len(events)-1].ID, nil
}
