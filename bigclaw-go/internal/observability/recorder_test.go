package observability

import (
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestTraceSummaryAggregatesTimeline(t *testing.T) {
	recorder := NewRecorder()
	base := time.Now()
	recorder.Record(domain.Event{ID: "evt-1", Type: domain.EventTaskQueued, TaskID: "task-1", TraceID: "trace-1", Timestamp: base})
	recorder.Record(domain.Event{ID: "evt-2", Type: domain.EventTaskStarted, TaskID: "task-1", TraceID: "trace-1", Timestamp: base.Add(2 * time.Second)})
	recorder.Record(domain.Event{ID: "evt-3", Type: domain.EventTaskCompleted, TaskID: "task-2", TraceID: "trace-1", Timestamp: base.Add(5 * time.Second)})

	summary, ok := recorder.TraceSummary("trace-1")
	if !ok {
		t.Fatal("expected trace summary")
	}
	if summary.EventCount != 3 {
		t.Fatalf("expected 3 events, got %+v", summary)
	}
	if len(summary.TaskIDs) != 2 || summary.TaskIDs[0] != "task-1" || summary.TaskIDs[1] != "task-2" {
		t.Fatalf("unexpected task ids: %+v", summary)
	}
	if summary.LatestEventType != domain.EventTaskCompleted {
		t.Fatalf("unexpected latest event type: %+v", summary)
	}
	if summary.DurationSeconds != 5 {
		t.Fatalf("expected 5s duration, got %+v", summary)
	}
}

func TestTraceSummariesReturnsMostRecentFirst(t *testing.T) {
	recorder := NewRecorder()
	base := time.Now()
	recorder.Record(domain.Event{ID: "evt-a", Type: domain.EventTaskQueued, TaskID: "task-a", TraceID: "trace-a", Timestamp: base})
	recorder.Record(domain.Event{ID: "evt-b", Type: domain.EventTaskQueued, TaskID: "task-b", TraceID: "trace-b", Timestamp: base.Add(time.Second)})

	summaries := recorder.TraceSummaries(1)
	if len(summaries) != 1 || summaries[0].TraceID != "trace-b" {
		t.Fatalf("expected most recent trace first, got %+v", summaries)
	}
}

func TestRecorderStoresTaskSnapshotsAndAppliesEventStates(t *testing.T) {
	recorder := NewRecorder()
	base := time.Now()
	recorder.StoreTask(domain.Task{
		ID:                 "task-snapshot",
		TraceID:            "trace-snapshot",
		Title:              "Snapshot",
		State:              domain.TaskQueued,
		Metadata:           map[string]string{"team": "platform"},
		AcceptanceCriteria: []string{"merge PR"},
		CreatedAt:          base,
		UpdatedAt:          base,
	})
	recorder.Record(domain.Event{ID: "evt-running", Type: domain.EventTaskStarted, TaskID: "task-snapshot", TraceID: "trace-snapshot", Timestamp: base.Add(time.Second)})

	task, ok := recorder.Task("task-snapshot")
	if !ok {
		t.Fatal("expected stored task snapshot")
	}
	if task.Title != "Snapshot" || task.Metadata["team"] != "platform" {
		t.Fatalf("expected rich task snapshot, got %+v", task)
	}
	if task.State != domain.TaskRunning {
		t.Fatalf("expected running task state, got %+v", task)
	}
	if task.UpdatedAt.Before(base.Add(time.Second)) {
		t.Fatalf("expected updated timestamp to advance, got %+v", task)
	}

	tasks := recorder.Tasks(1)
	if len(tasks) != 1 || tasks[0].ID != "task-snapshot" {
		t.Fatalf("expected sorted task snapshots, got %+v", tasks)
	}
}
