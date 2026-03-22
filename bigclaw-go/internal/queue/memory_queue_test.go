package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestMemoryQueueLeasesByPriority(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()
	now := time.Now()

	_ = q.Enqueue(ctx, domain.Task{ID: "p1", Priority: 2, CreatedAt: now})
	_ = q.Enqueue(ctx, domain.Task{ID: "p0", Priority: 1, CreatedAt: now.Add(time.Second)})

	task, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lease == nil || task == nil {
		t.Fatalf("expected lease and task")
	}
	if task.ID != "p0" {
		t.Fatalf("expected p0 first, got %s", task.ID)
	}
}

func TestMemoryQueueDeadLetterAndReplay(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()
	if err := q.Enqueue(ctx, domain.Task{ID: "task-dead", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	task, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || task == nil || lease == nil {
		t.Fatalf("lease: %v task=%v lease=%v", err, task, lease)
	}
	if err := q.DeadLetter(ctx, lease, "boom"); err != nil {
		t.Fatalf("dead letter: %v", err)
	}
	deadLetters, err := q.ListDeadLetters(ctx, 10)
	if err != nil {
		t.Fatalf("list dead letters: %v", err)
	}
	if len(deadLetters) != 1 || deadLetters[0].ID != "task-dead" {
		t.Fatalf("unexpected dead letters: %+v", deadLetters)
	}
	if err := q.ReplayDeadLetter(ctx, "task-dead"); err != nil {
		t.Fatalf("replay dead letter: %v", err)
	}
	deadLetters, err = q.ListDeadLetters(ctx, 10)
	if err != nil {
		t.Fatalf("list dead letters after replay: %v", err)
	}
	if len(deadLetters) != 0 {
		t.Fatalf("expected dead letters to be empty after replay, got %+v", deadLetters)
	}
	replayed, replayLease, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || replayed == nil || replayLease == nil {
		t.Fatalf("lease replayed task: %v task=%v lease=%v", err, replayed, replayLease)
	}
	if replayed.ID != "task-dead" {
		t.Fatalf("expected replayed task task-dead, got %s", replayed.ID)
	}
}

func TestMemoryQueueListAndCancelTask(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()
	base := time.Now()
	if err := q.Enqueue(ctx, domain.Task{ID: "queued-task", Priority: 1, CreatedAt: base}); err != nil {
		t.Fatalf("enqueue queued task: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "leased-task", Priority: 0, CreatedAt: base.Add(time.Second)}); err != nil {
		t.Fatalf("enqueue leased task: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || lease == nil {
		t.Fatalf("lease task: %v lease=%v", err, lease)
	}
	if lease.TaskID != "leased-task" {
		t.Fatalf("expected leased-task to lease first, got %s", lease.TaskID)
	}

	snapshots, err := q.ListTasks(ctx, 10)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 queue snapshots, got %+v", snapshots)
	}

	cancelled, err := q.CancelTask(ctx, "leased-task", "manual stop")
	if err != nil {
		t.Fatalf("cancel leased task: %v", err)
	}
	if cancelled.Task.State != domain.TaskCancelled || !cancelled.Leased {
		t.Fatalf("expected leased cancelled snapshot, got %+v", cancelled)
	}
	if got := q.Size(ctx); got != 1 {
		t.Fatalf("expected actionable size 1 after leased cancel, got %d", got)
	}

	_, err = q.CancelTask(ctx, "queued-task", "duplicate")
	if err != nil {
		t.Fatalf("cancel queued task: %v", err)
	}
	if got := q.Size(ctx); got != 0 {
		t.Fatalf("expected actionable size 0 after queued cancel, got %d", got)
	}
	if _, err := q.GetTask(ctx, "queued-task"); err == nil {
		t.Fatal("expected queued cancelled task to be removed from queue")
	}
}

func TestMemoryQueueRejectsRenewalAfterLeaseExpiry(t *testing.T) {
	q := NewMemoryQueue()
	ctx := context.Background()
	if err := q.Enqueue(ctx, domain.Task{ID: "task-expired-renew", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", 25*time.Millisecond)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	time.Sleep(40 * time.Millisecond)
	if err := q.RenewLease(ctx, lease, time.Minute); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected ErrLeaseExpired, got %v", err)
	}
}
