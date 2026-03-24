package queue

import (
	"context"
	"errors"
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

func TestFileQueueRejectsRenewalAfterLeaseExpiry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
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

func TestFileQueueRejectsExpiredLeaseMutations(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-expired-mutations", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", 25*time.Millisecond)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	time.Sleep(40 * time.Millisecond)
	if err := q.Ack(ctx, lease); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected expired ack to return ErrLeaseExpired, got %v", err)
	}
	if err := q.Requeue(ctx, lease, time.Now()); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected expired requeue to return ErrLeaseExpired, got %v", err)
	}
	if err := q.DeadLetter(ctx, lease, "expired"); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected expired dead-letter to return ErrLeaseExpired, got %v", err)
	}
}

func TestFileQueueRejectsStaleLeaseAfterReacquire(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.json")
	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-stale", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, firstLease, err := q.LeaseNext(ctx, "worker-a", 25*time.Millisecond)
	if err != nil || firstLease == nil {
		t.Fatalf("first lease: %v lease=%v", err, firstLease)
	}
	time.Sleep(40 * time.Millisecond)
	_, secondLease, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || secondLease == nil {
		t.Fatalf("second lease: %v lease=%v", err, secondLease)
	}
	if err := q.Ack(ctx, firstLease); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale ack to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.Requeue(ctx, firstLease, time.Now()); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale requeue to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.DeadLetter(ctx, firstLease, "stale"); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale dead-letter to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.RenewLease(ctx, firstLease, time.Minute); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale renew to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.Ack(ctx, secondLease); err != nil {
		t.Fatalf("ack second lease: %v", err)
	}
}
