package queue

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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
	task := domain.Task{
		ID:                 "task-meta",
		Source:             "linear",
		Title:              "Persist metadata",
		Description:        "keep fields",
		Labels:             []string{"platform"},
		RequiredTools:      []string{"browser"},
		AcceptanceCriteria: []string{"queue survives restart"},
		ValidationPlan:     []string{"go test ./internal/queue"},
		Priority:           1,
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
		t.Fatalf("lease: task=%v lease=%v err=%v", leased, lease, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected queue file to exist: %v", err)
	}
	if !reflect.DeepEqual(leased.Labels, []string{"platform"}) ||
		!reflect.DeepEqual(leased.RequiredTools, []string{"browser"}) ||
		!reflect.DeepEqual(leased.AcceptanceCriteria, []string{"queue survives restart"}) ||
		!reflect.DeepEqual(leased.ValidationPlan, []string{"go test ./internal/queue"}) {
		t.Fatalf("expected task payload to persist across reload, got %+v", leased)
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
	legacyTask := domain.Task{
		ID:          "legacy",
		Source:      "linear",
		Title:       "legacy",
		Description: "legacy payload",
		Priority:    0,
	}
	payload, err := json.Marshal([]item{
		{
			Task:        legacyTask,
			AvailableAt: time.Now(),
		},
	})
	if err != nil {
		t.Fatalf("marshal legacy queue: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write legacy queue: %v", err)
	}

	q, err := NewFileQueue(path)
	if err != nil {
		t.Fatalf("new queue from legacy storage: %v", err)
	}
	if got := q.Size(ctx); got != 1 {
		t.Fatalf("expected size 1 from legacy storage, got %d", got)
	}
	task, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || task == nil || lease == nil {
		t.Fatalf("lease legacy task: task=%v lease=%v err=%v", task, lease, err)
	}
	if task.ID != "legacy" {
		t.Fatalf("expected legacy task id, got %+v", task)
	}
}
