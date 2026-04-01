package queue

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestPythonParityFileQueueCreatesParentDirectoryAndPreservesTaskPayload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state", "queue.json")

	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	task := domain.Task{
		ID:                 "t-meta",
		Source:             "linear",
		Title:              "Persist metadata",
		Description:        "keep fields",
		Labels:             []string{"platform"},
		RequiredTools:      []string{"browser"},
		AcceptanceCriteria: []string{"queue survives restart"},
		ValidationPlan:     []string{"pytest tests/test_queue.py"},
		Priority:           2,
		CreatedAt:          time.Now(),
	}
	if err := q.Enqueue(ctx, task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	leased, lease, err := reloaded.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || leased == nil || lease == nil {
		t.Fatalf("lease: %v task=%v lease=%v", err, leased, lease)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected queue file to exist: %v", err)
	}
	if got := leased.Labels; len(got) != 1 || got[0] != "platform" {
		t.Fatalf("unexpected labels: %+v", got)
	}
	if got := leased.RequiredTools; len(got) != 1 || got[0] != "browser" {
		t.Fatalf("unexpected required tools: %+v", got)
	}
	if got := leased.AcceptanceCriteria; len(got) != 1 || got[0] != "queue survives restart" {
		t.Fatalf("unexpected acceptance criteria: %+v", got)
	}
	if got := leased.ValidationPlan; len(got) != 1 || got[0] != "pytest tests/test_queue.py" {
		t.Fatalf("unexpected validation plan: %+v", got)
	}
}

func TestPythonParityFileQueueLoadsLegacyListStorage(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	contents := `[
  {
    "priority": 0,
    "task_id": "legacy",
    "task": {
      "task_id": "legacy",
      "source": "linear",
      "title": "legacy",
      "description": "legacy payload",
      "priority": 0
    }
  }
]`
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write legacy queue: %v", err)
	}

	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("load legacy queue: %v", err)
	}
	if got := q.Size(ctx); got != 1 {
		t.Fatalf("expected size 1, got %d", got)
	}
	task, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || task == nil || lease == nil {
		t.Fatalf("lease legacy task: %v task=%v lease=%v", err, task, lease)
	}
	if task.ID != "legacy" {
		t.Fatalf("expected legacy task id, got %+v", task)
	}
}
