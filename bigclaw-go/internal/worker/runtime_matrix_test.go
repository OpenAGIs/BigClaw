package worker

import (
	"context"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
)

func TestRuntimeMatrixProcessesRequiredToolTasksToCompletion(t *testing.T) {
	q := queue.NewMemoryQueue()
	task := domain.Task{
		ID:            "matrix-worker-success",
		TraceID:       "trace-matrix-worker-success",
		RequiredTools: []string{"browser", "git"},
		CreatedAt:     time.Now(),
	}
	if err := q.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "matrix-worker",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorKubernetes, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	if !runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000}) {
		t.Fatal("expected task to be processed")
	}
	events := recorder.EventsByTask(task.ID, 10)
	if len(events) != 4 {
		t.Fatalf("expected leased, routed, started, completed events, got %+v", events)
	}
	if events[2].Type != domain.EventTaskStarted {
		t.Fatalf("expected started event, got %+v", events[2])
	}
	tools, ok := events[2].Payload["required_tools"].([]string)
	if !ok || len(tools) != 2 || tools[0] != "browser" || tools[1] != "git" {
		t.Fatalf("expected required tools on started event, got %+v", events[2].Payload)
	}
	if events[3].Type != domain.EventTaskCompleted {
		t.Fatalf("expected completed event, got %+v", events[3])
	}
}

func TestRuntimeMatrixSchedulerRoutesExpectedExecutorKinds(t *testing.T) {
	s := scheduler.New()

	low := s.Decide(domain.Task{ID: "matrix-low"}, scheduler.QuotaSnapshot{})
	if !low.Accepted || low.Assignment.Executor != domain.ExecutorLocal {
		t.Fatalf("expected low-risk task to route locally, got %+v", low)
	}

	high := s.Decide(domain.Task{ID: "matrix-high", RiskLevel: domain.RiskHigh}, scheduler.QuotaSnapshot{})
	if !high.Accepted || high.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected high-risk task to route to kubernetes, got %+v", high)
	}

	browser := s.Decide(domain.Task{ID: "matrix-browser", RequiredTools: []string{"browser"}}, scheduler.QuotaSnapshot{})
	if !browser.Accepted || browser.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected browser task to route to kubernetes, got %+v", browser)
	}
}

func TestRuntimeMatrixPublishesBlockedOperationsHandoff(t *testing.T) {
	q := queue.NewMemoryQueue()
	task := domain.Task{
		ID:            "matrix-operations-handoff",
		TraceID:       "trace-matrix-operations-handoff",
		Source:        "linear",
		Title:         "Customer analytics rollout",
		Description:   "Need customer stakeholder rollout and analytics validation.",
		Labels:        []string{"customer", "analytics"},
		RequiredTools: []string{"browser", "sql"},
		CreatedAt:     time.Now(),
	}
	if err := q.Enqueue(context.Background(), task); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	bus, recorder := newRuntimeRecorder()
	runtime := Runtime{
		WorkerID:    "matrix-worker",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(fakeRunner{kind: domain.ExecutorKubernetes, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    200 * time.Millisecond,
		TaskTimeout: time.Second,
	}

	if !runtime.RunOnce(context.Background(), scheduler.QuotaSnapshot{ConcurrentLimit: 10, BudgetRemaining: 1000}) {
		t.Fatal("expected task to be processed")
	}
	events := recorder.EventsByTask(task.ID, 10)
	if len(events) < 3 {
		t.Fatalf("expected routed and handoff events, got %+v", events)
	}
	if events[1].Type != domain.EventSchedulerRouted {
		t.Fatalf("expected routed event, got %+v", events[1])
	}
	if events[2].Type != domain.EventRunTakeover || events[2].Payload["target_team"] != "operations" || events[2].Payload["handoff_status"] != "blocked" {
		t.Fatalf("expected blocked operations handoff, got %+v", events[2])
	}
}
