package worker

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

type coordinatingRunner struct {
	started chan<- string
	release <-chan struct{}
}

func (r coordinatingRunner) Kind() domain.ExecutorKind { return domain.ExecutorLocal }

func (r coordinatingRunner) Capability() executor.Capability {
	return executor.Capability{Kind: domain.ExecutorLocal, MaxConcurrency: 1, SupportsShell: true}
}

func (r coordinatingRunner) Execute(ctx context.Context, task domain.Task) executor.Result {
	r.started <- task.ID

	select {
	case <-ctx.Done():
		return executor.Result{ShouldRetry: true, Message: ctx.Err().Error(), FinishedAt: time.Now()}
	case <-r.release:
		return executor.Result{Success: true, Message: "ok", FinishedAt: time.Now()}
	}
}

func TestPoolProcessesMultipleTasksAcrossWorkers(t *testing.T) {
	q := queue.NewMemoryQueue()
	for _, taskID := range []string{"task-1", "task-2"} {
		if err := q.Enqueue(context.Background(), domain.Task{ID: taskID, TraceID: taskID, Priority: 1, CreatedAt: time.Now()}); err != nil {
			t.Fatalf("enqueue %s: %v", taskID, err)
		}
	}

	started := make(chan string, 2)
	release := make(chan struct{})
	bus := events.NewBus()
	recorder := observability.NewRecorder()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	registry := executor.NewRegistry(coordinatingRunner{started: started, release: release})

	pool := NewPool(
		&Runtime{
			WorkerID:    "worker-1",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
		&Runtime{
			WorkerID:    "worker-2",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
	)

	done := make(chan bool, 1)
	go func() {
		done <- pool.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 2, BudgetRemaining: 1000})
	}()

	seen := map[string]bool{}
	for index := 0; index < 2; index++ {
		select {
		case taskID := <-started:
			seen[taskID] = true
		case <-time.After(500 * time.Millisecond):
			t.Fatal("expected both workers to start tasks before release")
		}
	}

	if !seen["task-1"] || !seen["task-2"] {
		t.Fatalf("expected both tasks to start, got %#v", seen)
	}

	close(release)

	select {
	case processed := <-done:
		if !processed {
			t.Fatal("expected pool tick to process tasks")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected pool tick to complete after releasing runners")
	}

	if got := q.Size(context.Background()); got != 0 {
		t.Fatalf("expected queue size 0 after ack, got %d", got)
	}

	snapshots := pool.Snapshots()
	if len(snapshots) != 2 {
		t.Fatalf("expected 2 worker snapshots, got %d", len(snapshots))
	}
	for _, snapshot := range snapshots {
		if snapshot.SuccessfulRuns != 1 || snapshot.State != "idle" {
			t.Fatalf("expected successful idle worker snapshot, got %+v", snapshot)
		}
	}

	summary := pool.Snapshot()
	if summary.WorkerID != "worker-pool" {
		t.Fatalf("expected worker-pool summary id, got %+v", summary)
	}
	if summary.SuccessfulRuns != 2 || summary.State != "idle" {
		t.Fatalf("expected aggregated successful idle summary, got %+v", summary)
	}
}

func TestPoolRespectsRemainingConcurrencyCapacity(t *testing.T) {
	q := queue.NewMemoryQueue()
	for _, taskID := range []string{"task-1", "task-2"} {
		if err := q.Enqueue(context.Background(), domain.Task{ID: taskID, TraceID: taskID, Priority: 1, CreatedAt: time.Now()}); err != nil {
			t.Fatalf("enqueue %s: %v", taskID, err)
		}
	}

	started := make(chan string, 2)
	release := make(chan struct{})
	bus := events.NewBus()
	recorder := observability.NewRecorder()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	registry := executor.NewRegistry(coordinatingRunner{started: started, release: release})

	pool := NewPool(
		&Runtime{
			WorkerID:    "worker-1",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
		&Runtime{
			WorkerID:    "worker-2",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
	)

	done := make(chan bool, 1)
	go func() {
		done <- pool.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 2, CurrentRunning: 1, BudgetRemaining: 1000})
	}()

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected one worker to start within remaining concurrency")
	}

	select {
	case taskID := <-started:
		t.Fatalf("expected only one worker to start, but saw extra task %s", taskID)
	case <-time.After(150 * time.Millisecond):
	}

	close(release)

	select {
	case processed := <-done:
		if !processed {
			t.Fatal("expected pool tick to process one task")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected pool tick to complete after release")
	}

	if got := q.Size(context.Background()); got != 1 {
		t.Fatalf("expected one task to remain queued after capacity-protected tick, got %d", got)
	}
}

func TestPoolLimitsPreemptibleOverflowAcrossWorkers(t *testing.T) {
	q := queue.NewMemoryQueue()
	base := time.Now()
	if err := q.Enqueue(context.Background(), domain.Task{ID: "task-low", TraceID: "task-low", Priority: 5, CreatedAt: base}); err != nil {
		t.Fatalf("enqueue low-priority task: %v", err)
	}

	bus := events.NewBus()
	recorder := observability.NewRecorder()
	bus.AddSink(events.RecorderSink{Recorder: recorder})

	lowRuntime := Runtime{
		WorkerID:    "worker-low",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorLocal, blockUntilContext: true}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    80 * time.Millisecond,
		TaskTimeout: 2 * time.Second,
	}
	lowDone := make(chan bool, 1)
	go func() {
		lowDone <- lowRuntime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 1, BudgetRemaining: 1000})
	}()

	deadline := time.Now().Add(time.Second)
	for {
		if state := lowRuntime.Snapshot().State; state == "running" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected low-priority runtime to start running before overflow test")
		}
		time.Sleep(10 * time.Millisecond)
	}

	for _, taskID := range []string{"task-urgent-1", "task-urgent-2"} {
		base = base.Add(time.Second)
		if err := q.Enqueue(context.Background(), domain.Task{ID: taskID, TraceID: taskID, Priority: 1, CreatedAt: base}); err != nil {
			t.Fatalf("enqueue %s: %v", taskID, err)
		}
	}

	started := make(chan string, 2)
	release := make(chan struct{})
	registry := executor.NewRegistry(coordinatingRunner{started: started, release: release})

	pool := NewPool(
		&Runtime{
			WorkerID:    "worker-1",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
		&Runtime{
			WorkerID:    "worker-2",
			Queue:       q,
			Scheduler:   scheduler.New(),
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			LeaseTTL:    time.Second,
			TaskTimeout: time.Second,
		},
	)

	done := make(chan bool, 1)
	go func() {
		done <- pool.RunOnce(context.Background(), scheduler.QuotaSnapshot{
			ConcurrentLimit:       1,
			CurrentRunning:        1,
			BudgetRemaining:       1000,
			PreemptibleExecutions: 1,
		})
	}()

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected one urgent worker to start using preemptible overflow")
	}

	select {
	case taskID := <-started:
		t.Fatalf("expected only one preemptible overflow worker to start, but saw extra task %s", taskID)
	case <-time.After(150 * time.Millisecond):
	}

	close(release)

	select {
	case processed := <-done:
		if !processed {
			t.Fatal("expected pool tick to process urgent overflow task")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected pool tick to complete after release")
	}

	select {
	case processed := <-lowDone:
		if !processed {
			t.Fatal("expected low-priority runtime to finish after being preempted")
		}
	case <-time.After(time.Second):
		t.Fatal("expected low-priority runtime to stop after urgent overflow preemption")
	}

	if got := q.Size(context.Background()); got != 1 {
		t.Fatalf("expected one urgent task to remain queued after capped overflow tick, got %d", got)
	}
}

func TestPoolSnapshotSummarizesWorkerState(t *testing.T) {
	workerA := &Runtime{WorkerID: "worker-a"}
	workerA.updateStatus(func(status *Status) {
		status.State = "running"
		status.CurrentTaskID = "task-urgent"
		status.CurrentTraceID = "trace-urgent"
		status.CurrentExecutor = domain.ExecutorLocal
		status.LeaseRenewals = 2
		status.LeaseRenewalFailures = 1
		status.LeaseLostRuns = 2
		status.SuccessfulRuns = 3
		status.PreemptionActive = true
		status.CurrentPreemptionTaskID = "task-low"
		status.CurrentPreemptionWorkerID = "worker-low"
		status.LastPreemptedTaskID = "task-low"
		status.LastPreemptionAt = time.Unix(1_700_000_100, 0)
		status.LastPreemptionReason = "preempted by urgent task"
		status.PreemptionsIssued = 1
	})

	workerB := &Runtime{WorkerID: "worker-b"}
	workerB.updateStatus(func(status *Status) {
		status.State = "idle"
		status.LeaseRenewalFailures = 3
		status.LeaseLostRuns = 1
		status.SuccessfulRuns = 4
		status.LastResult = "ok"
		status.LastFinishedAt = time.Unix(1_700_000_200, 0)
		status.LastTransition = string(domain.EventTaskCompleted)
	})

	pool := NewPool(workerA, workerB)
	summary := pool.Snapshot()

	if summary.State != "running" || summary.CurrentTaskID != "task-urgent" {
		t.Fatalf("expected running summary with active task, got %+v", summary)
	}
	if summary.SuccessfulRuns != 7 || summary.LeaseRenewals != 2 {
		t.Fatalf("expected aggregated counters, got %+v", summary)
	}
	if summary.LeaseRenewalFailures != 4 || summary.LeaseLostRuns != 3 {
		t.Fatalf("expected aggregated lease-safety counters, got %+v", summary)
	}
	if !summary.PreemptionActive || summary.LastPreemptedTaskID != "task-low" {
		t.Fatalf("expected aggregated preemption state, got %+v", summary)
	}
	if len(pool.Snapshots()) != 2 {
		t.Fatalf("expected 2 snapshots in pool")
	}
}
