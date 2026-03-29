package queue

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestFileQueuePersistsAcrossReload(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "queue.json")
	ctx := context.Background()

	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-1", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	if got := reloaded.Size(ctx); got != 1 {
		t.Fatalf("expected size 1, got %d", got)
	}
}

func TestFileQueueCreatesParentDirectoryAndPreservesTaskPayload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "state", "queue.json")

	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{
		ID:                 "task-meta",
		Source:             "linear",
		Title:              "Persist metadata",
		Description:        "keep fields",
		Labels:             []string{"platform"},
		RequiredTools:      []string{"browser"},
		AcceptanceCriteria: []string{"queue survives restart"},
		ValidationPlan:     []string{"go test ./internal/queue"},
		Priority:           2,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	task, lease, err := reloaded.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || task == nil || lease == nil {
		t.Fatalf("lease reloaded task: %v task=%v lease=%v", err, task, lease)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected queue file to exist: %v", err)
	}
	if got := task.Labels; len(got) != 1 || got[0] != "platform" {
		t.Fatalf("expected labels to persist, got %+v", got)
	}
	if got := task.RequiredTools; len(got) != 1 || got[0] != "browser" {
		t.Fatalf("expected required tools to persist, got %+v", got)
	}
	if got := task.AcceptanceCriteria; len(got) != 1 || got[0] != "queue survives restart" {
		t.Fatalf("expected acceptance criteria to persist, got %+v", got)
	}
	if got := task.ValidationPlan; len(got) != 1 || got[0] != "go test ./internal/queue" {
		t.Fatalf("expected validation plan to persist, got %+v", got)
	}
}

func TestFileQueueDeadLetterReplayPersistsAcrossReload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-dead", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	if err := q.DeadLetter(ctx, lease, "boom"); err != nil {
		t.Fatalf("dead letter: %v", err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	deadLetters, err := reloaded.ListDeadLetters(ctx, 10)
	if err != nil {
		t.Fatalf("list dead letters: %v", err)
	}
	if len(deadLetters) != 1 || deadLetters[0].ID != "task-dead" {
		t.Fatalf("unexpected dead letters: %+v", deadLetters)
	}
	if err := reloaded.ReplayDeadLetter(ctx, "task-dead"); err != nil {
		t.Fatalf("replay dead letter: %v", err)
	}
	replayed, replayLease, err := reloaded.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || replayed == nil || replayLease == nil {
		t.Fatalf("lease replayed task: %v task=%v lease=%v", err, replayed, replayLease)
	}
	if replayed.ID != "task-dead" {
		t.Fatalf("expected replayed task task-dead, got %s", replayed.ID)
	}
}

func TestFileQueueCancelPersistsAcrossReload(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-1", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue task-1: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-2", Priority: 0, CreatedAt: time.Now().Add(time.Second)}); err != nil {
		t.Fatalf("enqueue task-2: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	if _, err := q.CancelTask(ctx, "task-2", "manual stop"); err != nil {
		t.Fatalf("cancel leased task: %v", err)
	}
	if _, err := q.CancelTask(ctx, "task-1", "duplicate"); err != nil {
		t.Fatalf("cancel queued task: %v", err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	if got := reloaded.Size(ctx); got != 0 {
		t.Fatalf("expected actionable size 0 after cancels, got %d", got)
	}
	snapshot, err := reloaded.GetTask(ctx, "task-2")
	if err != nil {
		t.Fatalf("get leased cancelled task: %v", err)
	}
	if snapshot.Task.State != domain.TaskCancelled || !snapshot.Leased {
		t.Fatalf("expected leased cancelled snapshot after reload, got %+v", snapshot)
	}
	if _, err := reloaded.GetTask(ctx, "task-1"); err == nil {
		t.Fatal("expected unleased cancelled task to be removed after reload")
	}
}

func TestFileQueuePurgesCancelledLeaseAfterExpiry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-expire", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", 25*time.Millisecond)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	if _, err := q.CancelTask(ctx, "task-expire", "stop"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	time.Sleep(40 * time.Millisecond)

	// Trigger expiry recovery + purge.
	task, newLease, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || task != nil || newLease != nil {
		t.Fatalf("expected no lease after purge, got task=%v lease=%v err=%v", task, newLease, err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	if _, err := reloaded.GetTask(ctx, "task-expire"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected task to be purged after reload, got %v", err)
	}
}

func TestFileQueueLoadsLegacyListStorage(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	legacy := []map[string]any{
		{
			"priority": 1,
			"task_id":  "legacy",
			"task": map[string]any{
				"task_id":     "legacy",
				"source":      "linear",
				"title":       "legacy",
				"description": "legacy payload",
				"priority":    1,
			},
		},
	}
	payload, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy queue: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write legacy queue: %v", err)
	}

	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("load legacy queue: %v", err)
	}
	if got := q.Size(ctx); got != 1 {
		t.Fatalf("expected legacy queue size 1, got %d", got)
	}
	task, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || task == nil || lease == nil {
		t.Fatalf("lease legacy task: %v task=%v lease=%v", err, task, lease)
	}
	if task.ID != "legacy" || task.Description != "legacy payload" {
		t.Fatalf("unexpected legacy task payload: %+v", task)
	}
}
