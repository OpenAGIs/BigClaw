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
