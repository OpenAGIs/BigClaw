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

func TestSQLiteEventLogCheckpointAcknowledgeIsMonotonic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "event-log.db")
	log, err := NewSQLiteEventLog(path)
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = log.Close() }()
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-monotonic-1", Type: domain.EventTaskQueued, TaskID: "task-monotonic", TraceID: "trace-monotonic", Timestamp: base},
		{ID: "evt-monotonic-2", Type: domain.EventTaskStarted, TaskID: "task-monotonic", TraceID: "trace-monotonic", Timestamp: base.Add(time.Second)},
	} {
		if err := log.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	if _, err := log.Acknowledge("subscriber-monotonic", "evt-monotonic-2", base.Add(2*time.Second)); err != nil {
		t.Fatalf("ack latest event: %v", err)
	}
	checkpoint, err := log.Acknowledge("subscriber-monotonic", "evt-monotonic-1", base.Add(3*time.Second))
	if err != nil {
		t.Fatalf("ack stale event: %v", err)
	}
	if checkpoint.EventID != "evt-monotonic-2" {
		t.Fatalf("expected monotonic checkpoint to stay on newer event, got %+v", checkpoint)
	}
}

func TestSQLiteEventLogRetentionWatermark(t *testing.T) {
	log, err := NewSQLiteEventLog(filepath.Join(t.TempDir(), "event-log.db"))
	if err != nil {
		t.Fatalf("new sqlite event log: %v", err)
	}
	defer func() { _ = log.Close() }()
	base := time.Now()
	for _, event := range []domain.Event{
		{ID: "evt-watermark-1", Type: domain.EventTaskQueued, TaskID: "task-watermark", TraceID: "trace-watermark", Timestamp: base},
		{ID: "evt-watermark-2", Type: domain.EventTaskStarted, TaskID: "task-watermark", TraceID: "trace-watermark", Timestamp: base.Add(time.Second)},
	} {
		if err := log.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	watermark, err := log.RetentionWatermark()
	if err != nil {
		t.Fatalf("retention watermark: %v", err)
	}
	if watermark.Backend != "sqlite" || watermark.EventCount != 2 || watermark.OldestEventID != "evt-watermark-1" || watermark.NewestEventID != "evt-watermark-2" {
		t.Fatalf("unexpected retention watermark: %+v", watermark)
	}
	if watermark.OldestSequence == 0 || watermark.NewestSequence == 0 || watermark.NewestSequence < watermark.OldestSequence {
		t.Fatalf("expected ordered watermark sequences, got %+v", watermark)
	}
}

func TestSQLiteEventLogRetentionBoundaryPersistsAcrossInstances(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	path := filepath.Join(t.TempDir(), "event-log.db")
	log1, err := NewSQLiteEventLogWithOptions(path, SQLiteEventLogOptions{
		Retention: 2 * time.Second,
		Now:       func() time.Time { return base.Add(4 * time.Second) },
	})
	if err != nil {
		t.Fatalf("new sqlite event log with retention: %v", err)
	}
	for _, event := range []domain.Event{
		{ID: "evt-retention-old", Type: domain.EventTaskQueued, TaskID: "task-retention", TraceID: "trace-retention", Timestamp: base},
		{ID: "evt-retention-new", Type: domain.EventTaskStarted, TaskID: "task-retention", TraceID: "trace-retention", Timestamp: base.Add(3 * time.Second)},
	} {
		if err := log1.Write(context.Background(), event); err != nil {
			t.Fatalf("write %s: %v", event.ID, err)
		}
	}
	watermark, err := log1.RetentionWatermark()
	if err != nil {
		t.Fatalf("retention watermark after trim: %v", err)
	}
	if !watermark.HistoryTruncated || !watermark.PersistedBoundary || watermark.TrimmedThroughEventID != "evt-retention-old" {
		t.Fatalf("expected persisted trimmed boundary, got %+v", watermark)
	}
	if watermark.OldestEventID != "evt-retention-new" || watermark.EventCount != 1 {
		t.Fatalf("expected only retained event to remain, got %+v", watermark)
	}
	if err := log1.Close(); err != nil {
		t.Fatalf("close first sqlite event log: %v", err)
	}
	log2, err := NewSQLiteEventLogWithOptions(path, SQLiteEventLogOptions{Retention: 2 * time.Second, Now: func() time.Time { return base.Add(4 * time.Second) }})
	if err != nil {
		t.Fatalf("reopen sqlite event log with retention: %v", err)
	}
	defer func() { _ = log2.Close() }()
	watermark, err = log2.RetentionWatermark()
	if err != nil {
		t.Fatalf("retention watermark after reopen: %v", err)
	}
	if !watermark.HistoryTruncated || watermark.TrimmedThroughSequence == 0 || watermark.TrimmedThroughEventID != "evt-retention-old" {
		t.Fatalf("expected persisted boundary after reopen, got %+v", watermark)
	}
	replayed, err := log2.Replay(10)
	if err != nil {
		t.Fatalf("replay retained events: %v", err)
	}
	if len(replayed) != 1 || replayed[0].ID != "evt-retention-new" {
		t.Fatalf("expected retained replay window after reopen, got %+v", replayed)
	}
}

func TestSQLiteEventLogCheckpointDiagnosticExpiresTrimmedCheckpoint(t *testing.T) {
	base := time.Unix(1_700_000_000, 0).UTC()
	log, err := NewSQLiteEventLogWithOptions(filepath.Join(t.TempDir(), "event-log.db"), SQLiteEventLogOptions{
		Retention: 2 * time.Second,
		Now:       func() time.Time { return base },
	})
	if err != nil {
		t.Fatalf("new sqlite event log with retention: %v", err)
	}
	defer func() { _ = log.Close() }()
	if err := log.Write(context.Background(), domain.Event{ID: "evt-diagnostic-old", Type: domain.EventTaskQueued, TaskID: "task-diagnostic", TraceID: "trace-diagnostic", Timestamp: base}); err != nil {
		t.Fatalf("write old event: %v", err)
	}
	if _, err := log.Acknowledge("subscriber-diagnostic", "evt-diagnostic-old", base.Add(500*time.Millisecond)); err != nil {
		t.Fatalf("acknowledge old event: %v", err)
	}
	log.now = func() time.Time { return base.Add(4 * time.Second) }
	if err := log.Write(context.Background(), domain.Event{ID: "evt-diagnostic-new", Type: domain.EventTaskStarted, TaskID: "task-diagnostic", TraceID: "trace-diagnostic", Timestamp: base.Add(4 * time.Second)}); err != nil {
		t.Fatalf("write new event: %v", err)
	}
	diagnostic, err := log.CheckpointDiagnostic("subscriber-diagnostic")
	if err != nil {
		t.Fatalf("checkpoint diagnostic: %v", err)
	}
	if diagnostic.Status != "expired" || diagnostic.Reason != "checkpoint_expired" {
		t.Fatalf("expected expired checkpoint diagnostic, got %+v", diagnostic)
	}
	if diagnostic.Checkpoint == nil || diagnostic.Checkpoint.EventID != "evt-diagnostic-old" || diagnostic.Checkpoint.EventSequence == 0 {
		t.Fatalf("expected checkpoint metadata in diagnostic, got %+v", diagnostic.Checkpoint)
	}
	if diagnostic.RetentionWatermark == nil || diagnostic.RetentionWatermark.OldestEventID != "evt-diagnostic-new" {
		t.Fatalf("expected retained watermark context, got %+v", diagnostic.RetentionWatermark)
	}
	if diagnostic.ResetAction == nil || diagnostic.ResetAction.Action != "reset_checkpoint" || diagnostic.ResetAction.EarliestRetainedEventID != "evt-diagnostic-new" {
		t.Fatalf("expected reset guidance in diagnostic, got %+v", diagnostic.ResetAction)
	}
}
