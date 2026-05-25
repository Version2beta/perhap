package perhap

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type counterSnapshot struct {
	Count int `json:"count"`
}

type counterEventData struct {
	By int `json:"by"`
}

func TestService_ReplayFromJetStreamSortsAndSkipsInternalSubjects(t *testing.T) {
	srv := runJetStreamServer(t)
	nc := connectTestClient(t, srv)
	defer nc.Close()

	service, err := NewService(nc, ServiceConfig[counterSnapshot]{
		Context: "store",
		Domain:  "counter",
		Reducer: testCounterReducer,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	entityID := "cart-1"
	first := testCounterEvent(t, entityID, "incremented", 1)
	second := testCounterEvent(t, entityID, "incremented", 2)
	if first.ID >= second.ID {
		t.Fatalf("expected first event ID to sort before second: %q >= %q", first.ID, second.ID)
	}
	publishStoredEvent(t, nc, first)
	publishInternalModelUpdate(t, nc, service.config.Context, entityID)
	publishStoredEvent(t, nc, second)
	publishStoredException(t, nc, service.config.Context, entityID)

	model, reducedThrough, err := service.replayFromJetStream(entityID)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if model.Count != 3 {
		t.Fatalf("expected replayed count 3, got %d", model.Count)
	}
	if reducedThrough != second.ID {
		t.Fatalf("expected reducedThrough %q, got %q", second.ID, reducedThrough)
	}
}

func TestService_GetOrCreateSupervisorReplaysHistoryForRecreatedSupervisor(t *testing.T) {
	srv := runJetStreamServer(t)
	nc := connectTestClient(t, srv)
	defer nc.Close()

	service, err := NewService(nc, ServiceConfig[counterSnapshot]{
		Context: "store",
		Domain:  "counter",
		Reducer: testCounterReducer,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	entityID := "cart-2"
	first := testCounterEvent(t, entityID, "incremented", 1)
	second := testCounterEvent(t, entityID, "incremented", 2)
	publishStoredEvent(t, nc, first)

	supervisor := service.getOrCreateSupervisor(entityID)
	if got := supervisor.currentModel().Count; got != 1 {
		t.Fatalf("expected first supervisor to replay count 1, got %d", got)
	}

	service.mu.Lock()
	delete(service.supervisors, entityID)
	service.mu.Unlock()

	publishStoredEvent(t, nc, second)
	recreated := service.getOrCreateSupervisor(entityID)
	if got := recreated.currentModel().Count; got != 3 {
		t.Fatalf("expected recreated supervisor to replay count 3, got %d", got)
	}
	if got := recreated.reducedThrough(); got != second.ID {
		t.Fatalf("expected reducedThrough %q, got %q", second.ID, got)
	}
}

func TestSupervisor_EmitExceptionPublishesToJetStream(t *testing.T) {
	srv := runJetStreamServer(t)
	nc := connectTestClient(t, srv)
	defer nc.Close()

	service, err := NewService(nc, ServiceConfig[counterSnapshot]{
		Context: "store",
		Domain:  "counter",
		Reducer: testCounterReducer,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	supervisor := &supervisor[counterSnapshot]{
		entityID: "cart-3",
		nc:       nc,
		config:   service.config,
		window:   []Event{},
	}

	supervisor.emitException([]ValidationError{{Field: "count", Message: "boom"}})

	sub, err := service.js.SubscribeSync("perhap.store.cart-3._exception", nats.DeliverAll(), nats.AckNone())
	if err != nil {
		t.Fatalf("subscribe exception: %v", err)
	}
	defer sub.Drain()

	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("next exception msg: %v", err)
	}

	var event Event
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		t.Fatalf("unmarshal exception event: %v", err)
	}
	if event.Type != "_exception" {
		t.Fatalf("expected _exception event, got %q", event.Type)
	}
}

func runJetStreamServer(t *testing.T) *server.Server {
	t.Helper()
	srv, err := server.NewServer(&server.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("nats server not ready")
	}
	t.Cleanup(func() {
		srv.Shutdown()
		srv.WaitForShutdown()
	})
	return srv
}

func connectTestClient(t *testing.T, srv *server.Server) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	return nc
}

func testCounterReducer(_ string, model counterSnapshot, event Event) (counterSnapshot, []Event, error) {
	var data counterEventData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return model, nil, err
	}
	model.Count += data.By
	return model, nil, nil
}

func testCounterEvent(t *testing.T, entityID, eventType string, by int) Event {
	t.Helper()
	data, err := json.Marshal(counterEventData{By: by})
	if err != nil {
		t.Fatalf("marshal event data: %v", err)
	}
	return Event{
		ID:            NewEventID(),
		EntityID:      entityID,
		Context:       "store",
		Type:          eventType,
		CorrelationID: NewEventID(),
		Data:          data,
	}
}

func publishStoredEvent(t *testing.T, nc *nats.Conn, event Event) {
	t.Helper()
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	msg := nats.NewMsg(fmt.Sprintf("perhap.%s.%s.%s", event.Context, event.EntityID, event.Type))
	msg.Data = payload
	msg.Header.Set("Nats-Msg-Id", event.ID)
	if _, err := js.PublishMsg(msg); err != nil {
		t.Fatalf("publish event: %v", err)
	}
}

func publishInternalModelUpdate(t *testing.T, nc *nats.Conn, context, entityID string) {
	t.Helper()
	js, err := nc.JetStream()
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	update, err := json.Marshal(ModelUpdate{
		EntityID:       entityID,
		Context:        context,
		Domain:         "counter",
		ReducedThrough: "",
		Window:         []string{},
		Reduction:      json.RawMessage(`{"count":99}`),
	})
	if err != nil {
		t.Fatalf("marshal model update: %v", err)
	}
	msg := nats.NewMsg(fmt.Sprintf("perhap.%s.%s._model", context, entityID))
	msg.Data = update
	msg.Header.Set("Nats-Msg-Id", NewEventID())
	if _, err := js.PublishMsg(msg); err != nil {
		t.Fatalf("publish model update: %v", err)
	}
}

func publishStoredException(t *testing.T, nc *nats.Conn, context, entityID string) {
	t.Helper()
	exception := Event{
		ID:            NewEventID(),
		EntityID:      entityID,
		Context:       context,
		Type:          "_exception",
		CorrelationID: NewEventID(),
		Data:          json.RawMessage(`{"validation_errors":["bad"]}`),
	}
	publishStoredEvent(t, nc, exception)
}
