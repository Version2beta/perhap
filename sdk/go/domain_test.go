package perhap

import (
	"encoding/json"
	"testing"
)

// Simple counter model for testing.
type counterModel struct {
	Count int `json:"count"`
}

type incrementData struct {
	By int `json:"by"`
}

type resetData struct{}

func TestDomain_OnDispatchesTypedHandlers(t *testing.T) {
	d := NewDomain[counterModel]("test", "counter")
	On(d, "incremented", func(m counterModel, e incrementData) counterModel {
		m.Count += e.By
		return m
	})
	On(d, "reset", func(m counterModel, _ resetData) counterModel {
		m.Count = 0
		return m
	})

	reducer := d.buildReducer()

	incData, _ := json.Marshal(incrementData{By: 5})
	model, _, _ := reducer("incremented", counterModel{Count: 0}, Event{Data: incData})
	if model.Count != 5 {
		t.Fatalf("expected count 5, got %d", model.Count)
	}

	model, _, _ = reducer("incremented", model, Event{Data: incData})
	if model.Count != 10 {
		t.Fatalf("expected count 10, got %d", model.Count)
	}

	resetRaw, _ := json.Marshal(resetData{})
	model, _, _ = reducer("reset", model, Event{Data: resetRaw})
	if model.Count != 0 {
		t.Fatalf("expected count 0 after reset, got %d", model.Count)
	}
}

func TestDomain_UnknownEventPassesThrough(t *testing.T) {
	d := NewDomain[counterModel]("test", "counter")
	reducer := d.buildReducer()

	start := counterModel{Count: 42}
	result, newEvents, _ := reducer("unknown_event", start, Event{Data: json.RawMessage(`{}`)})
	if result.Count != 42 {
		t.Fatalf("model should be unchanged for unknown event, got count %d", result.Count)
	}
	if len(newEvents) != 0 {
		t.Fatalf("expected no new events, got %d", len(newEvents))
	}
}

func TestDomain_BadJSONLeavesModelUnchanged(t *testing.T) {
	d := NewDomain[counterModel]("test", "counter")
	On(d, "incremented", func(m counterModel, e incrementData) counterModel {
		m.Count += e.By
		return m
	})
	reducer := d.buildReducer()

	start := counterModel{Count: 7}
	result, _, err := reducer("incremented", start, Event{Data: json.RawMessage(`not json`)})
	if result.Count != 7 {
		t.Fatalf("bad JSON should leave model unchanged, got count %d", result.Count)
	}
	if err == nil {
		t.Fatal("bad JSON should return an error")
	}
}

func TestDomain_ServiceConfigIncludesReducerAndValidator(t *testing.T) {
	validated := false
	d := NewDomain[counterModel]("ctx", "dom")
	On(d, "incremented", func(m counterModel, e incrementData) counterModel {
		m.Count += e.By
		return m
	})
	d.Validate(func(m counterModel) []ValidationError {
		validated = true
		return nil
	})

	cfg := d.ServiceConfig()
	if cfg.Context != "ctx" || cfg.Domain != "dom" {
		t.Fatalf("unexpected context/domain: %s/%s", cfg.Context, cfg.Domain)
	}
	if cfg.Reducer == nil {
		t.Fatal("ServiceConfig.Reducer should not be nil")
	}
	if cfg.Validator == nil {
		t.Fatal("ServiceConfig.Validator should not be nil")
	}

	// Invoke validator to confirm it's the right one.
	cfg.Validator(counterModel{})
	if !validated {
		t.Fatal("validator was not called")
	}
}

func TestDomain_HandlersSnapshotted(t *testing.T) {
	// Handlers registered after buildReducer() should not affect existing reducers.
	d := NewDomain[counterModel]("test", "counter")
	On(d, "incremented", func(m counterModel, e incrementData) counterModel {
		m.Count += e.By
		return m
	})
	reducer := d.buildReducer()

	// Register a new handler AFTER building the reducer.
	On(d, "reset", func(m counterModel, _ resetData) counterModel {
		m.Count = 0
		return m
	})

	resetRaw, _ := json.Marshal(resetData{})
	result, _, _ := reducer("reset", counterModel{Count: 99}, Event{Data: resetRaw})
	// The snapshotted reducer should NOT know about "reset".
	if result.Count != 99 {
		t.Fatalf("post-build handler should not affect snapshotted reducer, got count %d", result.Count)
	}
}
