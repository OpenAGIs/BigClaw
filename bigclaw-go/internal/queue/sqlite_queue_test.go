package queue

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestSQLiteQueuePersistsAndLeases(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()

	if err := q.Enqueue(ctx, domain.Task{ID: "task-1", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if got := q.Size(ctx); got != 1 {
		t.Fatalf("expected size 1, got %d", got)
	}
	leasedTask, lease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	if leasedTask == nil || lease == nil || leasedTask.ID != "task-1" {
		t.Fatalf("unexpected lease result: %#v %#v", leasedTask, lease)
	}
	if err := q.Ack(ctx, lease); err != nil {
		t.Fatalf("ack: %v", err)
	}
	if got := q.Size(ctx); got != 0 {
		t.Fatalf("expected size 0, got %d", got)
	}

	reopened, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	if got := reopened.Size(ctx); got != 0 {
		t.Fatalf("expected persisted size 0, got %d", got)
	}
}

func TestSQLiteQueueLeaseExpiresAndCanBeReacquired(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()
	if err := q.Enqueue(ctx, domain.Task{ID: "task-expire", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	firstTask, firstLease, err := q.LeaseNext(ctx, "worker-a", 50*time.Millisecond)
	if err != nil || firstTask == nil || firstLease == nil {
		t.Fatalf("first lease: %v task=%v lease=%v", err, firstTask, firstLease)
	}
	time.Sleep(80 * time.Millisecond)
	secondTask, secondLease, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || secondTask == nil || secondLease == nil {
		t.Fatalf("second lease: %v task=%v lease=%v", err, secondTask, secondLease)
	}
	if secondTask.ID != "task-expire" {
		t.Fatalf("expected reacquired task task-expire, got %s", secondTask.ID)
	}
	if secondLease.WorkerID != "worker-b" {
		t.Fatalf("expected second lease worker-b, got %s", secondLease.WorkerID)
	}
}

func TestSQLiteQueueDoesNotDoubleLeaseAcrossClients(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	primary, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer primary.Close()
	secondary, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new secondary sqlite queue: %v", err)
	}
	defer secondary.Close()

	if err := primary.Enqueue(ctx, domain.Task{ID: "task-double-lease", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	type leaseResult struct {
		task  *domain.Task
		lease *Lease
		err   error
	}

	start := make(chan struct{})
	results := make(chan leaseResult, 2)
	go func() {
		<-start
		task, lease, err := primary.LeaseNext(ctx, "worker-a", time.Minute)
		results <- leaseResult{task: task, lease: lease, err: err}
	}()
	go func() {
		<-start
		task, lease, err := secondary.LeaseNext(ctx, "worker-b", time.Minute)
		results <- leaseResult{task: task, lease: lease, err: err}
	}()
	close(start)

	first := <-results
	second := <-results
	if first.err != nil {
		t.Fatalf("first lease: %v", first.err)
	}
	if second.err != nil {
		t.Fatalf("second lease: %v", second.err)
	}

	leasedCount := 0
	if first.task != nil || first.lease != nil {
		if first.task == nil || first.lease == nil {
			t.Fatalf("expected first result to include both task and lease, got task=%v lease=%v", first.task, first.lease)
		}
		leasedCount++
	}
	if second.task != nil || second.lease != nil {
		if second.task == nil || second.lease == nil {
			t.Fatalf("expected second result to include both task and lease, got task=%v lease=%v", second.task, second.lease)
		}
		leasedCount++
	}
	if leasedCount != 1 {
		t.Fatalf("expected exactly one lease across clients, got %d (first=%v second=%v)", leasedCount, first.lease, second.lease)
	}
}

func TestSQLiteQueueDeadLetterReplayPersistsAcrossReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
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
	deadLetters, err := q.ListDeadLetters(ctx, 10)
	if err != nil {
		t.Fatalf("list dead letters: %v", err)
	}
	if len(deadLetters) != 1 || deadLetters[0].ID != "task-dead" {
		t.Fatalf("unexpected dead letters: %+v", deadLetters)
	}
	if err := q.Close(); err != nil {
		t.Fatalf("close queue: %v", err)
	}

	reopened, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("reopen queue: %v", err)
	}
	defer reopened.Close()
	deadLetters, err = reopened.ListDeadLetters(ctx, 10)
	if err != nil {
		t.Fatalf("list dead letters after reopen: %v", err)
	}
	if len(deadLetters) != 1 || deadLetters[0].ID != "task-dead" {
		t.Fatalf("unexpected dead letters after reopen: %+v", deadLetters)
	}
	if err := reopened.ReplayDeadLetter(ctx, "task-dead"); err != nil {
		t.Fatalf("replay dead letter: %v", err)
	}
	deadLetters, err = reopened.ListDeadLetters(ctx, 10)
	if err != nil {
		t.Fatalf("list dead letters after replay: %v", err)
	}
	if len(deadLetters) != 0 {
		t.Fatalf("expected dead letters to be empty after replay, got %+v", deadLetters)
	}
	replayed, replayLease, err := reopened.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || replayed == nil || replayLease == nil {
		t.Fatalf("lease replayed task: %v task=%v lease=%v", err, replayed, replayLease)
	}
	if replayed.ID != "task-dead" {
		t.Fatalf("expected replayed task task-dead, got %s", replayed.ID)
	}
}

func TestSQLiteQueueRecoverExpiredLeasesRequeuesAndPurges(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()
	base := time.Now()
	if err := q.Enqueue(ctx, domain.Task{ID: "task-recover", Priority: 1, CreatedAt: base}); err != nil {
		t.Fatalf("enqueue recover task: %v", err)
	}
	if err := q.Enqueue(ctx, domain.Task{ID: "task-purge", Priority: 2, CreatedAt: base.Add(time.Second)}); err != nil {
		t.Fatalf("enqueue purge task: %v", err)
	}
	_, recoverLease, err := q.LeaseNext(ctx, "worker-a", time.Minute)
	if err != nil || recoverLease == nil {
		t.Fatalf("lease recover task: %v lease=%v", err, recoverLease)
	}
	_, purgeLease, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || purgeLease == nil {
		t.Fatalf("lease purge task: %v lease=%v", err, purgeLease)
	}
	if _, err := q.CancelTask(ctx, "task-purge", "stop"); err != nil {
		t.Fatalf("cancel purge task: %v", err)
	}

	expiredAt := time.Now().Add(2 * time.Minute)
	if _, err := q.db.Exec(`UPDATE tasks SET lease_expires_ns=? WHERE task_id IN (?, ?)`, expiredAt.Add(-time.Second).UnixNano(), "task-recover", "task-purge"); err != nil {
		t.Fatalf("mark leases expired: %v", err)
	}

	result, err := q.RecoverExpiredLeases(ctx, expiredAt)
	if err != nil {
		t.Fatalf("recover expired leases: %v", err)
	}
	if result.Recovered != 1 || result.Purged != 1 {
		t.Fatalf("expected one recovered and one purged lease, got %+v", result)
	}

	snapshot, err := q.GetTask(ctx, "task-recover")
	if err != nil {
		t.Fatalf("get recovered task: %v", err)
	}
	if snapshot.Leased || snapshot.Task.State != domain.TaskQueued {
		t.Fatalf("expected recovered task to be queued and unleased, got %+v", snapshot)
	}
	if _, err := q.GetTask(ctx, "task-purge"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected purged task to be removed, got %v", err)
	}
}

func TestSQLiteQueueProcessesTaskScaleWithoutDuplicateLease(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		totalTasks int
		workers    int
		leaseTTL   time.Duration
	}{
		{name: "1k", totalTasks: 1000, workers: 8, leaseTTL: time.Second},
		{name: "10k", totalTasks: 10000, workers: 16, leaseTTL: 30 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "queue.db")
			q, err := NewSQLiteQueue(path)
			if err != nil {
				t.Fatalf("new sqlite queue: %v", err)
			}
			defer q.Close()

			for i := 0; i < tc.totalTasks; i++ {
				if err := q.Enqueue(ctx, domain.Task{ID: fmt.Sprintf("task-%d", i), Priority: i % 5, CreatedAt: time.Now()}); err != nil {
					t.Fatalf("enqueue %d: %v", i, err)
				}
			}

			var (
				mu        sync.Mutex
				processed = make(map[string]struct{}, tc.totalTasks)
				wg        sync.WaitGroup
				errCh     = make(chan error, tc.workers*2)
			)
			for worker := 0; worker < tc.workers; worker++ {
				wg.Add(1)
				go func(workerID string) {
					defer wg.Done()
					for {
						task, lease, err := q.LeaseNext(ctx, workerID, tc.leaseTTL)
						if err != nil {
							errCh <- err
							return
						}
						if task == nil || lease == nil {
							if q.Size(ctx) == 0 {
								return
							}
							time.Sleep(time.Millisecond)
							continue
						}
						mu.Lock()
						if _, exists := processed[task.ID]; exists {
							mu.Unlock()
							errCh <- fmt.Errorf("duplicate lease for %s", task.ID)
							return
						}
						processed[task.ID] = struct{}{}
						mu.Unlock()
						if err := q.Ack(ctx, lease); err != nil {
							errCh <- err
							return
						}
					}
				}(fmt.Sprintf("worker-%d", worker))
			}
			wg.Wait()
			close(errCh)
			for err := range errCh {
				if err != nil {
					t.Fatal(err)
				}
			}
			if len(processed) != tc.totalTasks {
				t.Fatalf("expected %d processed tasks, got %d", tc.totalTasks, len(processed))
			}
			if got := q.Size(ctx); got != 0 {
				t.Fatalf("expected queue size 0, got %d", got)
			}
		})
	}
}

func TestSQLiteQueueCancelAndInspect(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()
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
	if err := q.Requeue(ctx, lease, time.Now()); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected cancelled task to reject requeue with ErrLeaseNotOwned, got %v", err)
	}
	if _, err := q.CancelTask(ctx, "task-1", "duplicate"); err != nil {
		t.Fatalf("cancel queued task: %v", err)
	}
	if got := q.Size(ctx); got != 0 {
		t.Fatalf("expected actionable size 0 after cancels, got %d", got)
	}
	snapshot, err := q.GetTask(ctx, "task-2")
	if err != nil {
		t.Fatalf("get leased cancelled task: %v", err)
	}
	if snapshot.Task.State != domain.TaskCancelled || !snapshot.Leased {
		t.Fatalf("expected leased cancelled snapshot, got %+v", snapshot)
	}
	snapshots, err := q.ListTasks(ctx, 10)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].Task.ID != "task-2" {
		t.Fatalf("expected only leased cancelled task to remain, got %+v", snapshots)
	}
}

func TestSQLiteQueueRejectsStaleLeaseAfterReacquire(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()
	if err := q.Enqueue(ctx, domain.Task{ID: "task-stale", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, firstLease, err := q.LeaseNext(ctx, "worker-a", 50*time.Millisecond)
	if err != nil || firstLease == nil {
		t.Fatalf("first lease: %v lease=%v", err, firstLease)
	}
	time.Sleep(80 * time.Millisecond)
	_, secondLease, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || secondLease == nil {
		t.Fatalf("second lease: %v lease=%v", err, secondLease)
	}
	if firstLease.Attempt == secondLease.Attempt {
		t.Fatalf("expected attempt to advance across reacquire, got first=%d second=%d", firstLease.Attempt, secondLease.Attempt)
	}
	if err := q.Ack(ctx, firstLease); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale ack to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.Requeue(ctx, firstLease, time.Now()); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale requeue to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.DeadLetter(ctx, firstLease, "stale"); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale dead letter to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.RenewLease(ctx, firstLease, time.Minute); !errors.Is(err, ErrLeaseNotOwned) {
		t.Fatalf("expected stale renew to fail with ErrLeaseNotOwned, got %v", err)
	}
	if err := q.Ack(ctx, secondLease); err != nil {
		t.Fatalf("ack second lease: %v", err)
	}
	if got := q.Size(ctx); got != 0 {
		t.Fatalf("expected queue size 0 after ack, got %d", got)
	}
}

func TestSQLiteQueueRejectsRenewalAfterLeaseExpiry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()

	if err := q.Enqueue(ctx, domain.Task{ID: "task-expired-renew", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", 30*time.Millisecond)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	time.Sleep(40 * time.Millisecond)
	if err := q.RenewLease(ctx, lease, time.Minute); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("expected ErrLeaseExpired, got %v", err)
	}
}

func TestSQLiteQueuePurgesCancelledLeaseAfterExpiry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "queue.db")
	q, err := NewSQLiteQueue(path)
	if err != nil {
		t.Fatalf("new sqlite queue: %v", err)
	}
	defer q.Close()

	if err := q.Enqueue(ctx, domain.Task{ID: "task-expire-cancel", Priority: 1, CreatedAt: time.Now()}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	_, lease, err := q.LeaseNext(ctx, "worker-a", 25*time.Millisecond)
	if err != nil || lease == nil {
		t.Fatalf("lease: %v lease=%v", err, lease)
	}
	if _, err := q.CancelTask(ctx, "task-expire-cancel", "stop"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	time.Sleep(40 * time.Millisecond)

	// LeaseNext runs the cancelled-lease cleanup.
	task, lease2, err := q.LeaseNext(ctx, "worker-b", time.Minute)
	if err != nil || task != nil || lease2 != nil {
		t.Fatalf("expected no lease after purge, got task=%v lease=%v err=%v", task, lease2, err)
	}
	if _, err := q.GetTask(ctx, "task-expire-cancel"); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected purged task, got %v", err)
	}
}
