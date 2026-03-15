package events

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestSQLiteEventLogPersistsReplayAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := NewSQLiteEventLog(path)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	base := time.Now()
	if err := log1.Write(context.Background(), domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: base}); err != nil {
		t.Fatalf("write evt-1: %v", err)
	}
	if err := log1.Write(context.Background(), domain.Event{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-1", TraceID: "trace-1", Timestamp: base.Add(time.Second)}); err != nil {
		t.Fatalf("write evt-2: %v", err)
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}
	log2, err := NewSQLiteEventLog(path)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	replayed, err := log2.Replay(10)
	if err != nil {
		t.Fatalf("replay durable event log: %v", err)
	}
	if len(replayed) != 2 || replayed[0].ID != "evt-1" || replayed[1].ID != "evt-2" {
		t.Fatalf("unexpected replayed events: %+v", replayed)
	}
	byTrace, err := log2.EventsByTrace("trace-1", 10)
	if err != nil {
		t.Fatalf("events by trace: %v", err)
	}
	if len(byTrace) != 2 {
		t.Fatalf("expected 2 trace events, got %+v", byTrace)
	}
}

func TestSQLiteEventLogReplayAfterCursor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-log.db")
	log, err := NewSQLiteEventLog(path)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = log.Close() }()
	base := time.Now()
	eventsToWrite := []domain.Event{
		{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: base},
		{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-1", TraceID: "trace-1", Timestamp: base.Add(time.Second)},
		{ID: "evt-3", Type: domain.EventTaskCompleted, TaskID: "task-2", TraceID: "trace-2", Timestamp: base.Add(2 * time.Second)},
	}
	for _, event := range eventsToWrite {
		if err := log.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	replayed, err := log.ReplayAfter("evt-1", 10)
	if err != nil {
		t.Fatalf("replay after: %v", err)
	}
	if len(replayed) != 2 || replayed[0].ID != "evt-2" || replayed[1].ID != "evt-3" {
		t.Fatalf("unexpected replay-after events: %+v", replayed)
	}
	byTask, err := log.EventsByTaskAfter("task-1", "evt-1", 10)
	if err != nil {
		t.Fatalf("events by task after: %v", err)
	}
	if len(byTask) != 1 || byTask[0].ID != "evt-2" {
		t.Fatalf("unexpected task replay-after events: %+v", byTask)
	}
	missingCursor, err := log.ReplayAfter("missing-event", 2)
	if err != nil {
		t.Fatalf("replay missing cursor: %v", err)
	}
	if len(missingCursor) != 2 || missingCursor[0].ID != "evt-1" || missingCursor[1].ID != "evt-2" {
		t.Fatalf("unexpected replay when cursor missing: %+v", missingCursor)
	}
}

func TestSQLiteEventLogCheckpointPersistsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := NewSQLiteEventLog(path)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	base := time.Now()
	if err := log1.Write(context.Background(), domain.Event{ID: "evt-check-1", Type: domain.EventTaskQueued, TaskID: "task-check", TraceID: "trace-check", Timestamp: base}); err != nil {
		t.Fatalf("write event: %v", err)
	}
	checkpoint, err := log1.Acknowledge("subscriber-a", "evt-check-1", base.Add(time.Second))
	if err != nil {
		t.Fatalf("acknowledge checkpoint: %v", err)
	}
	if checkpoint.EventID != "evt-check-1" {
		t.Fatalf("unexpected checkpoint after ack: %+v", checkpoint)
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}

	log2, err := NewSQLiteEventLog(path)
	if err != nil {
		t.Fatalf("reopen sqlite event log: %v", err)
	}
	defer func() { _ = log2.Close() }()
	checkpoint, err = log2.Checkpoint("subscriber-a")
	if err != nil {
		t.Fatalf("read checkpoint after reopen: %v", err)
	}
	if checkpoint.EventID != "evt-check-1" || checkpoint.SubscriberID != "subscriber-a" {
		t.Fatalf("unexpected reopened checkpoint: %+v", checkpoint)
	}
}
