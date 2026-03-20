package queue

import (
	"context"
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

func TestFileQueueBlockedTaskPersistsAndCanBeReleased(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-block", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if _, err := q.UpdateTaskState(ctx, "task-block", domain.TaskBlocked, time.Now(), "awaiting operator"); err != nil {
		t.Fatalf("block task: %v", err)
	}

	reloaded, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	snapshot, err := reloaded.GetTask(ctx, "task-block")
	if err != nil {
		t.Fatalf("get blocked task: %v", err)
	}
	if snapshot.Task.State != domain.TaskBlocked || snapshot.Task.Metadata["blocked_reason"] != "awaiting operator" {
		t.Fatalf("expected blocked task to persist, got %+v", snapshot)
	}
	if got := reloaded.Size(ctx); got != 0 {
		t.Fatalf("expected blocked task to be excluded from actionable size, got %d", got)
	}
	if task, lease, err := reloaded.LeaseNext(ctx, "worker-a", time.Minute); err != nil || task != nil || lease != nil {
		t.Fatalf("expected blocked task to stay parked after reload, got task=%v lease=%v err=%v", task, lease, err)
	}
	if _, err := reloaded.UpdateTaskState(ctx, "task-block", domain.TaskQueued, time.Now(), ""); err != nil {
		t.Fatalf("release blocked task: %v", err)
	}
	task, lease, err := reloaded.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || task == nil || lease == nil || task.ID != "task-block" {
		t.Fatalf("expected released blocked task to lease, got task=%v lease=%v err=%v", task, lease, err)
	}
}
