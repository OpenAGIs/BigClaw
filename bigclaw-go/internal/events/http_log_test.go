package events

import (
	"context"
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
}

func TestHTTPEventLogReadsRetentionWatermarkFromService(t *testing.T) {
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
		{ID: "evt-http-watermark-1", Type: domain.EventTaskQueued, TaskID: "task-http-watermark", TraceID: "trace-http-watermark", Timestamp: base},
		{ID: "evt-http-watermark-2", Type: domain.EventTaskStarted, TaskID: "task-http-watermark", TraceID: "trace-http-watermark", Timestamp: base.Add(time.Second)},
	} {
		if err := client.Write(context.Background(), event); err != nil {
			t.Fatalf("write remote event %s: %v", event.ID, err)
		}
	}
	watermark, err := client.RetentionWatermark()
	if err != nil {
		t.Fatalf("read remote retention watermark: %v", err)
	}
	if watermark.Backend != "sqlite" || watermark.EventCount != 2 || watermark.OldestEventID != "evt-http-watermark-1" || watermark.NewestEventID != "evt-http-watermark-2" {
		t.Fatalf("unexpected remote watermark: %+v", watermark)
	}
}
