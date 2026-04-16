package perhap

import (
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"time"
)

// Domain builds a ServiceConfig from typed event handlers.
//
// Instead of writing a raw Reducer with a type switch over json.RawMessage,
// the developer registers a handler per event type using the package-level
// On function. Each handler receives an already-unmarshaled, strongly-typed
// data struct, keeping domain logic clean and individually testable.
//
// Usage:
//
//	d := perhap.NewDomain[ChildProfile]("development", "child")
//	perhap.On(d, "skill_observed",  onSkillObserved)  // func(ChildProfile, SkillObserved) ChildProfile
//	perhap.On(d, "skill_confirmed", onSkillConfirmed)
//	perhap.On(d, "skill_regressed", onSkillRegressed)
//	d.Validate(validateProfile)
//	svc, err := perhap.NewService(nc, d.ServiceConfig())
//
// Why is On a package-level function and not a method?
// Go does not allow methods to introduce type parameters beyond those on the
// receiver. A method on *Domain[M] cannot be generic in D. The package-level
// function is the idiomatic Go workaround.
type Domain[M any] struct {
	context  string
	domain   string
	handlers map[string]typedHandler[M]
	sealed   bool // true after ServiceConfig() is called; On() panics if sealed

	// Optional overrides — zero values use ServiceConfig defaults.
	windowSize       int
	windowIdleClose  time.Duration
	passivateTimeout time.Duration
	lateEventPolicy  LateEventPolicy
	validator        Validator[M]
}

// typedHandler is an internal adapter that handles JSON unmarshaling
// and calls the user's strongly-typed handler function.
type typedHandler[M any] func(model M, raw json.RawMessage) (M, error)

// NewDomain creates a Domain builder for model type M.
// context and domain identify where events are routed in NATS:
// events arrive on perhap.{context}.{entityId}.{eventType}.
func NewDomain[M any](context, domain string) *Domain[M] {
	return &Domain[M]{
		context:  context,
		domain:   domain,
		handlers: make(map[string]typedHandler[M]),
	}
}

// On registers a typed event handler on d for the given eventType.
//
// fn receives the current model and an already-unmarshaled value of type D.
// It must return the new model — pure, no side effects.
//
// On panics if called after ServiceConfig() has been called, or if a handler
// for eventType is already registered.
//
// On is a package-level function (not a method) because Go does not allow
// generic methods to introduce type parameters beyond those on the receiver.
// Type parameters M and D are inferred from d and fn:
//
//	perhap.On(d, "skill_observed", onSkillObserved)  // M, D inferred
func On[M any, D any](d *Domain[M], eventType string, fn func(M, D) M) *Domain[M] {
	if d.sealed {
		panic(fmt.Sprintf("perhap: On(%q) called after ServiceConfig() — handlers are sealed", eventType))
	}
	if _, exists := d.handlers[eventType]; exists {
		panic(fmt.Sprintf("perhap: duplicate handler registered for event type %q", eventType))
	}
	d.handlers[eventType] = func(model M, raw json.RawMessage) (M, error) {
		var data D
		if err := json.Unmarshal(raw, &data); err != nil {
			return model, fmt.Errorf("perhap: unmarshal %q data: %w", eventType, err)
		}
		return fn(model, data), nil
	}
	return d
}

// Validate sets the domain invariant checker. It is called after every
// reduction; returning one or more ValidationErrors marks the model invalid
// and emits an _exception event. Passing nil clears a previously set validator.
func (d *Domain[M]) Validate(v Validator[M]) *Domain[M] {
	d.validator = v
	return d
}

// WithWindowSize overrides the default window size (100 events).
func (d *Domain[M]) WithWindowSize(n int) *Domain[M] {
	d.windowSize = n
	return d
}

// WithWindowIdleClose overrides how long the supervisor waits after the
// last event before closing the window and marking pending=false (default 500ms).
func (d *Domain[M]) WithWindowIdleClose(dur time.Duration) *Domain[M] {
	d.windowIdleClose = dur
	return d
}

// WithPassivateTimeout overrides how long an idle supervisor lives after its
// window closes before being evicted from memory (default 5m).
func (d *Domain[M]) WithPassivateTimeout(dur time.Duration) *Domain[M] {
	d.passivateTimeout = dur
	return d
}

// WithLateEventPolicy sets how out-of-window events are handled.
func (d *Domain[M]) WithLateEventPolicy(p LateEventPolicy) *Domain[M] {
	d.lateEventPolicy = p
	return d
}

// RegisteredEventTypes returns the event types that have handlers registered,
// sorted alphabetically. Useful for documentation, health endpoints, and tooling.
func (d *Domain[M]) RegisteredEventTypes() []string {
	types := make([]string, 0, len(d.handlers))
	for t := range d.handlers {
		types = append(types, t)
	}
	sort.Strings(types)
	return types
}

// ServiceConfig returns a ServiceConfig ready for use with NewService.
// After this call the Domain is sealed — calling On() will panic.
// Zero-value optional fields use the defaults defined in withDefaults.
func (d *Domain[M]) ServiceConfig() ServiceConfig[M] {
	d.sealed = true

	validator := d.validator
	if validator == nil {
		validator = func(M) []ValidationError { return nil }
	}

	return ServiceConfig[M]{
		Context:          d.context,
		Domain:           d.domain,
		Reducer:          d.buildReducer(),
		Validator:        validator,
		WindowSize:       d.windowSize,
		WindowIdleClose:  d.windowIdleClose,
		PassivateTimeout: d.passivateTimeout,
		LateEventPolicy:  d.lateEventPolicy,
	}
}

// buildReducer constructs the raw Reducer[M] that dispatches to typed handlers.
// Handlers are snapshotted at call time — mutations to d.handlers after this
// point do not affect the returned reducer.
//
// Unknown event types are passed through with the model unchanged. This is
// intentional: services only handle events they know about; unrecognised events
// may belong to a future version of the domain.
//
// Unmarshal errors are returned as errors so the supervisor can emit an
// _exception event rather than silently dropping state changes.
func (d *Domain[M]) buildReducer() Reducer[M] {
	handlers := maps.Clone(d.handlers)

	return func(eventType string, model M, event Event) (M, []Event, error) {
		handler, ok := handlers[eventType]
		if !ok {
			return model, nil, nil
		}
		newModel, err := handler(model, event.Data)
		if err != nil {
			return model, nil, err
		}
		return newModel, nil, nil
	}
}
