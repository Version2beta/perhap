package perhap

import (
	"encoding/json"
	"testing"

	"github.com/nats-io/nats.go"
)

func TestDecodeReplayEvent(t *testing.T) {
	valid := Event{
		ID:       NewEventID(),
		EntityID: "cart-1",
		Context:  "store",
		Type:     "incremented",
		Data:     json.RawMessage(`{"by":1}`),
	}
	payload, err := json.Marshal(valid)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	tests := []struct {
		name     string
		msg      *nats.Msg
		wantOK   bool
		wantType string
	}{
		{
			name: "domain event is kept",
			msg: &nats.Msg{
				Subject: "perhap.store.cart-1.incremented",
				Data:    payload,
			},
			wantOK:   true,
			wantType: "incremented",
		},
		{
			name: "internal subject is skipped",
			msg: &nats.Msg{
				Subject: "perhap.store.cart-1._model",
				Data:    payload,
			},
			wantOK: false,
		},
		{
			name: "short subject is skipped",
			msg: &nats.Msg{
				Subject: "perhap.store.cart-1",
				Data:    payload,
			},
			wantOK: false,
		},
		{
			name: "bad payload is skipped",
			msg: &nats.Msg{
				Subject: "perhap.store.cart-1.incremented",
				Data:    []byte(`not json`),
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, ok := decodeReplayEvent(tt.msg)
			if ok != tt.wantOK {
				t.Fatalf("decodeReplayEvent() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if event.Type != tt.wantType {
				t.Fatalf("decodeReplayEvent() type = %q, want %q", event.Type, tt.wantType)
			}
			if event.ID != valid.ID {
				t.Fatalf("decodeReplayEvent() id = %q, want %q", event.ID, valid.ID)
			}
		})
	}
}
