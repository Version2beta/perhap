package perhap

import "testing"

func TestSupervisorShouldSkipEvent(t *testing.T) {
	snapshotID := "00000000-0000-6000-8000-000000000010"
	sv := &supervisor[counterSnapshot]{
		entityID:     "cart-1",
		config:       ServiceConfig[counterSnapshot]{Context: "store"},
		snapshotUUID: snapshotID,
		windowUUIDs: map[string]bool{
			"00000000-0000-6000-8000-000000000020": true,
		},
	}

	tests := []struct {
		name    string
		eventID string
		want    bool
	}{
		{
			name:    "duplicate window event is skipped",
			eventID: "00000000-0000-6000-8000-000000000020",
			want:    true,
		},
		{
			name:    "event before snapshot is skipped",
			eventID: "00000000-0000-6000-8000-000000000009",
			want:    true,
		},
		{
			name:    "newer event is processed",
			eventID: "00000000-0000-6000-8000-000000000021",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skip := sv.shouldSkipEvent(Event{
				ID:       tt.eventID,
				EntityID: sv.entityID,
				Context:  sv.config.Context,
				Type:     "incremented",
			})
			if skip != tt.want {
				t.Fatalf("shouldSkipEvent(%q) = %v, want %v", tt.eventID, skip, tt.want)
			}
		})
	}
}

func TestSupervisorExceptionPayload(t *testing.T) {
	sv := &supervisor[counterSnapshot]{
		snapshotUUID: "00000000-0000-6000-8000-000000000001",
		window: []Event{
			{ID: "00000000-0000-6000-8000-000000000002"},
			{ID: "00000000-0000-6000-8000-000000000003"},
		},
	}

	payload := sv.exceptionPayload([]ValidationError{
		{Field: "count", Message: "must be positive"},
		{Message: "window inconsistent"},
	})

	if len(payload.WindowUUIDs) != 2 || payload.WindowUUIDs[0] != "00000000-0000-6000-8000-000000000002" {
		t.Fatalf("unexpected window UUIDs: %#v", payload.WindowUUIDs)
	}
	if len(payload.ValidationErrors) != 2 || payload.ValidationErrors[0] != "count: must be positive" {
		t.Fatalf("unexpected validation errors: %#v", payload.ValidationErrors)
	}
	if payload.LastValidUUID != "00000000-0000-6000-8000-000000000003" {
		t.Fatalf("unexpected last valid UUID: %q", payload.LastValidUUID)
	}
}
