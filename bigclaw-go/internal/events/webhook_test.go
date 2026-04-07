package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestWebhookSinkWritesEvent(t *testing.T) {
	received := make(chan domain.Event, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var event domain.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("decode event: %v", err)
		}
		received <- event
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sink := NewWebhookSink(WebhookConfig{URLs: []string{server.URL}, Timeout: time.Second})
	event := domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", Timestamp: time.Now()}
	if err := sink.Write(context.Background(), event); err != nil {
		t.Fatalf("write webhook: %v", err)
	}
	got := <-received
	if got.ID != event.ID {
		t.Fatalf("expected %s, got %s", event.ID, got.ID)
	}
}
