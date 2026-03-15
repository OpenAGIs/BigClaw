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

func TestHTTPEventLogReadsPersistedRetentionBoundaryFromService(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	store, err := NewSQLiteEventLogWithOptions(filepath.Join(t.TempDir(), "event-log.db"), SQLiteEventLogOptions{
		Retention: 2 * time.Second,
		Now:       func() time.Time { return base.Add(4 * time.Second) },
	})
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
	for _, event := range []domain.Event{
		{ID: "evt-http-retention-old", Type: domain.EventTaskQueued, TaskID: "task-http-retention", TraceID: "trace-http-retention", Timestamp: base},
		{ID: "evt-http-retention-new", Type: domain.EventTaskStarted, TaskID: "task-http-retention", TraceID: "trace-http-retention", Timestamp: base.Add(3 * time.Second)},
	} {
		if err := client.Write(context.Background(), event); err != nil {
			t.Fatalf("write remote event %s: %v", event.ID, err)
		}
	}
	watermark, err := client.RetentionWatermark()
	if err != nil {
		t.Fatalf("read remote retention watermark: %v", err)
	}
	if !watermark.HistoryTruncated || !watermark.PersistedBoundary || watermark.TrimmedThroughEventID != "evt-http-retention-old" {
		t.Fatalf("expected persisted remote retention boundary, got %+v", watermark)
	}
	if watermark.EventCount != 1 || watermark.OldestEventID != "evt-http-retention-new" {
		t.Fatalf("expected retained remote event window, got %+v", watermark)
	}
}

func TestHTTPEventLogResetsCheckpointThroughService(t *testing.T) {
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
	if err := client.Write(context.Background(), domain.Event{ID: "evt-reset-1", Type: domain.EventTaskQueued, TaskID: "task-reset", TraceID: "trace-reset", Timestamp: base}); err != nil {
		t.Fatalf("write reset event: %v", err)
	}
	checkpoint, err := client.Acknowledge("subscriber-reset", "evt-reset-1", base.Add(time.Second))
	if err != nil {
		t.Fatalf("ack checkpoint: %v", err)
	}
	if checkpoint.EventSequence == 0 {
		t.Fatalf("expected checkpoint sequence, got %+v", checkpoint)
	}
	if err := client.ResetCheckpoint("subscriber-reset"); err != nil {
		t.Fatalf("reset checkpoint: %v", err)
	}
	if _, err := client.Checkpoint("subscriber-reset"); !IsNoEventLog(err) {
		t.Fatalf("expected checkpoint to be cleared, got %v", err)
	}
}
