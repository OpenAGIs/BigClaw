package events

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestHTTPEventLogRoundTripsThroughService(t *testing.T) {
	store, err := NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	server := httptest.NewServer(NewEventLogServiceHandler(store))
	defer server.Close()

	client, err := NewHTTPEventLog(server.URL, "")
	if err != nil {
		t.Fatalf("new http event log: %v", err)
	}
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-remote-1", Type: domain.EventTaskQueued, TaskID: "task-remote", TraceID: "trace-remote", Timestamp: base},
		{ID: "evt-remote-2", Type: domain.EventTaskStarted, TaskID: "task-remote", TraceID: "trace-remote", Timestamp: base.Add(time.Second)},
	} {
		if err := client.Write(context.Background(), event); err != nil {
			t.Fatalf("write remote event %s: %v", event.ID, err)
		}
	}
	replayed, err := client.EventsByTraceAfter("trace-remote", "evt-remote-1", 10)
	if err != nil {
		t.Fatalf("events by trace after via remote log: %v", err)
	}
	if len(replayed) != 1 || replayed[0].ID != "evt-remote-2" {
		t.Fatalf("unexpected remote replay-after events: %+v", replayed)
	}
	checkpoint, err := client.Acknowledge("subscriber-remote", "evt-remote-2", base.Add(2*time.Second))
	if err != nil {
		t.Fatalf("acknowledge remote checkpoint: %v", err)
	}
	if checkpoint.EventID != "evt-remote-2" {
		t.Fatalf("unexpected remote checkpoint after ack: %+v", checkpoint)
	}
	checkpoint, err = client.Checkpoint("subscriber-remote")
	if err != nil {
		t.Fatalf("read remote checkpoint: %v", err)
	}
	if checkpoint.EventID != "evt-remote-2" || checkpoint.SubscriberID != "subscriber-remote" {
		t.Fatalf("unexpected remote checkpoint payload: %+v", checkpoint)
	}
	watermark := client.RetentionWatermark(context.Background())
	if !watermark.Available || watermark.OldestEventID != "evt-remote-1" || watermark.NewestEventID != "evt-remote-2" {
		t.Fatalf("unexpected remote retention watermark: %+v", watermark)
	}
	if capabilities := client.Capabilities(); capabilities.RetentionWatermark.NewestEventID != "evt-remote-2" {
		t.Fatalf("expected remote capabilities watermark, got %+v", capabilities.RetentionWatermark)
	}
}

func TestEventLogServiceEventsResponseIncludesRetentionWatermark(t *testing.T) {
	store, err := NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = store.Close() }()
	base := time.Unix(1700000200, 0).UTC()
	for _, event := range []domain.Event{
		{ID: "evt-service-1", Type: domain.EventTaskQueued, TaskID: "task-service", TraceID: "trace-service", Timestamp: base},
		{ID: "evt-service-2", Type: domain.EventTaskStarted, TaskID: "task-service", TraceID: "trace-service", Timestamp: base.Add(time.Second)},
	} {
		if err := store.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/events?trace_id=trace-service&limit=10", nil)
	response := httptest.NewRecorder()
	NewEventLogServiceHandler(store).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", response.Code, response.Body.String())
	}
	var decoded struct {
		Events             []domain.Event     `json:"events"`
		RetentionWatermark RetentionWatermark `json:"retention_watermark"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode event-log service response: %v", err)
	}
	if len(decoded.Events) != 2 || decoded.RetentionWatermark.OldestEventID != "evt-service-1" || decoded.RetentionWatermark.NewestEventID != "evt-service-2" {
		t.Fatalf("unexpected service watermark payload: %+v", decoded)
	}
}
