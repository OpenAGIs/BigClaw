package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

type testRunner struct {
	kind   domain.ExecutorKind
	result executor.Result
}

func (runner testRunner) Kind() domain.ExecutorKind { return runner.kind }

func (runner testRunner) Capability() executor.Capability {
	return executor.Capability{Kind: runner.kind, MaxConcurrency: 1, SupportsShell: true}
}

func (runner testRunner) Execute(_ context.Context, _ domain.Task) executor.Result {
	result := runner.result
	if result.FinishedAt.IsZero() {
		result.FinishedAt = time.Unix(1700000002, 0).UTC()
	}
	return result
}

func TestEngineRunDefinitionWritesReportAndJournal(t *testing.T) {
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	runtime := &worker.Runtime{
		WorkerID:    "worker-1",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(testRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	engine := &Engine{
		Runtime:  runtime,
		Recorder: recorder,
		Queue:    q,
		Quota:    scheduler.QuotaSnapshot{ConcurrentLimit: 4, BudgetRemaining: 5000},
		Now:      func() time.Time { return time.Unix(1700000000, 0).UTC() },
	}

	tempDir := t.TempDir()
	definition, err := ParseDefinition(`{"name":"release-closeout","steps":[{"name":"execute","kind":"scheduler"}],"report_path_template":"` + filepath.Join(tempDir, `reports`, `{task_id}`, `{run_id}.md`) + `","journal_path_template":"` + filepath.Join(tempDir, `journals`, `{workflow}`, `{run_id}.json`) + `","validation_evidence":["go test ./..."]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	result, err := engine.RunDefinition(context.Background(), domain.Task{
		ID:                 "task-1",
		Title:              "Ship runtime closeout",
		AcceptanceCriteria: []string{"go test ./..."},
		ValidationPlan:     []string{"go test ./..."},
	}, definition, "run-1", RunOptions{
		ValidationEvidence: []string{"go test ./..."},
		GitPushSucceeded:   true,
		GitLogStatOutput:   " cmd/file.go | 10 +++++-----",
		RepoSyncAudit: &observability.RepoSyncAudit{
			Sync: observability.GitSyncTelemetry{Status: "synced"},
			PullRequest: observability.PullRequestFreshness{
				BranchState: "in-sync",
				BodyState:   "fresh",
			},
		},
	})
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Acceptance.Status != "accepted" || !result.Acceptance.Passed {
		t.Fatalf("expected accepted result, got %+v", result.Acceptance)
	}
	if result.Quota.ConcurrentLimit != 4 || result.Quota.BudgetRemaining != 5000 {
		t.Fatalf("expected explicit quota preserved in result, got %+v", result.Quota)
	}
	if result.WorkflowRun.Status != WorkflowRunSucceeded {
		t.Fatalf("expected succeeded workflow run, got %+v", result.WorkflowRun)
	}
	if len(result.WorkflowRun.Steps) != 1 || result.WorkflowRun.Steps[0].Status != WorkflowStepSucceeded {
		t.Fatalf("expected execute step succeeded, got %+v", result.WorkflowRun.Steps)
	}
	if !result.Closeout.Complete {
		t.Fatalf("expected complete closeout, got %+v", result.Closeout)
	}
	if result.Task.State != domain.TaskSucceeded {
		t.Fatalf("expected succeeded task, got %+v", result.Task)
	}
	if len(result.Events) != 4 {
		t.Fatalf("expected 4 events, got %d", len(result.Events))
	}
	reportContents, err := os.ReadFile(filepath.Join(tempDir, "reports", "task-1", "run-1.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportContents), "Acceptance: accepted") {
		t.Fatalf("expected acceptance in report, got %s", string(reportContents))
	}
	if !strings.Contains(string(reportContents), "Scheduler Quota: concurrency=4 queue_depth=64 budget_remaining=5000") {
		t.Fatalf("expected scheduler quota in report, got %s", string(reportContents))
	}
	if !strings.Contains(string(reportContents), "Repo Sync Status: synced") {
		t.Fatalf("expected repo sync status in report, got %s", string(reportContents))
	}
	if !strings.Contains(string(reportContents), "Repo Sync Verified: true") {
		t.Fatalf("expected repo sync verification in report, got %s", string(reportContents))
	}
	journalContents, err := os.ReadFile(filepath.Join(tempDir, "journals", "release-closeout", "run-1.json"))
	if err != nil {
		t.Fatalf("read journal: %v", err)
	}
	if !strings.Contains(string(journalContents), `"step": "closeout"`) {
		t.Fatalf("expected closeout journal entry, got %s", string(journalContents))
	}
	if !strings.Contains(string(journalContents), `"repo_sync_status": "synced"`) {
		t.Fatalf("expected repo sync journal details, got %s", string(journalContents))
	}
	if !strings.Contains(string(journalContents), `"repo_sync_verified": true`) {
		t.Fatalf("expected repo sync verification in journal, got %s", string(journalContents))
	}
}

func TestEngineRunDefinitionRequiresApprovalForHighRiskTask(t *testing.T) {
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	runtime := &worker.Runtime{
		WorkerID:    "worker-2",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(testRunner{kind: domain.ExecutorKubernetes, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	engine := &Engine{
		Runtime:  runtime,
		Recorder: recorder,
		Queue:    q,
		Quota:    scheduler.QuotaSnapshot{ConcurrentLimit: 4, BudgetRemaining: 5000},
	}

	definition, err := ParseDefinition(`{"name":"risk-closeout","steps":[{"name":"security-review","kind":"approval"}],"validation_evidence":["go test ./..."]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	result, err := engine.RunDefinition(context.Background(), domain.Task{
		ID:                 "task-risk",
		Title:              "Sensitive rollout",
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"go test ./..."},
		ValidationPlan:     []string{"go test ./..."},
	}, definition, "run-risk", RunOptions{ValidationEvidence: []string{"go test ./..."}})
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Acceptance.Status != "needs-approval" || result.Acceptance.Passed {
		t.Fatalf("expected needs-approval, got %+v", result.Acceptance)
	}
	if result.Task.State != domain.TaskBlocked {
		t.Fatalf("expected workflow task parked in blocked state pending approval, got %+v", result.Task)
	}
	snapshot, err := q.GetTask(context.Background(), "task-risk")
	if err != nil {
		t.Fatalf("get blocked workflow task: %v", err)
	}
	if snapshot.Task.State != domain.TaskBlocked || snapshot.Task.Metadata["blocked_reason"] != "requires approval for high-risk task" {
		t.Fatalf("expected blocked workflow queue task, got %+v", snapshot)
	}
	if result.WorkflowRun.Status != WorkflowRunRunning {
		t.Fatalf("expected running workflow run awaiting approval, got %+v", result.WorkflowRun)
	}
	if len(result.WorkflowRun.Steps) != 1 || result.WorkflowRun.Steps[0].Status != WorkflowStepPending {
		t.Fatalf("expected pending approval step, got %+v", result.WorkflowRun.Steps)
	}
}

func TestEngineRunDefinitionRequiresApprovalForComputedHighRiskTask(t *testing.T) {
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	runtime := &worker.Runtime{
		WorkerID:    "worker-approval",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(testRunner{kind: domain.ExecutorKubernetes, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	engine := &Engine{
		Runtime:  runtime,
		Recorder: recorder,
		Queue:    q,
		Quota:    scheduler.QuotaSnapshot{ConcurrentLimit: 4, BudgetRemaining: 5000},
	}

	definition, err := ParseDefinition(`{"name":"security-closeout","steps":[{"name":"security-review","kind":"approval"}],"validation_evidence":["go test ./..."]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	result, err := engine.RunDefinition(context.Background(), domain.Task{
		ID:                 "task-computed-risk",
		Title:              "Production deploy",
		Priority:           1,
		Labels:             []string{"security", "prod"},
		RequiredTools:      []string{"deploy"},
		AcceptanceCriteria: []string{"go test ./..."},
		ValidationPlan:     []string{"go test ./..."},
	}, definition, "run-computed-risk", RunOptions{ValidationEvidence: []string{"go test ./..."}})
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Acceptance.Status != "needs-approval" || result.Task.State != domain.TaskBlocked {
		t.Fatalf("expected computed high-risk workflow to block pending approval, got acceptance=%+v task=%+v", result.Acceptance, result.Task)
	}
	if result.WorkflowRun.Status != WorkflowRunRunning {
		t.Fatalf("expected computed high-risk workflow to remain running pending approval, got %+v", result.WorkflowRun)
	}
}

func TestEngineRunDefinitionKeepsCloseoutPendingWithoutVerifiedRepoSyncAudit(t *testing.T) {
	tempDir := t.TempDir()
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	runtime := &worker.Runtime{
		WorkerID:    "worker-3",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(testRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	engine := &Engine{
		Runtime:  runtime,
		Recorder: recorder,
		Queue:    q,
		Quota:    scheduler.QuotaSnapshot{ConcurrentLimit: 4, BudgetRemaining: 5000},
		Now:      func() time.Time { return time.Unix(0, 0).UTC() },
	}
	definition, err := ParseDefinition(`{"name":"release-closeout","report_path_template":"` + filepath.Join(tempDir, `reports`, `{task_id}`, `{run_id}.md`) + `","journal_path_template":"` + filepath.Join(tempDir, `journals`, `{workflow}`, `{run_id}.json`) + `","validation_evidence":["go test ./..."]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	result, err := engine.RunDefinition(context.Background(), domain.Task{
		ID:                 "task-2",
		TraceID:            "trace-2",
		Title:              "Release build",
		State:              domain.TaskQueued,
		AcceptanceCriteria: []string{"go test ./..."},
		ValidationPlan:     []string{"go test ./..."},
	}, definition, "run-2", RunOptions{
		ValidationEvidence: []string{"go test ./..."},
		GitPushSucceeded:   true,
		GitLogStatOutput:   " cmd/file.go | 10 +++++-----",
		RepoSyncAudit: &observability.RepoSyncAudit{
			Sync: observability.GitSyncTelemetry{Status: "synced"},
			PullRequest: observability.PullRequestFreshness{
				BranchState: "stale",
				BodyState:   "fresh",
			},
		},
	})
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.Closeout.Complete {
		t.Fatalf("expected closeout to remain pending without verified repo sync audit: %+v", result.Closeout)
	}
}

func TestEngineRunDefinitionUsesPolicyQuotaDefaults(t *testing.T) {
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	runtime := &worker.Runtime{
		WorkerID:    "worker-policy",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(testRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	engine := &Engine{
		Runtime:  runtime,
		Recorder: recorder,
		Queue:    q,
	}

	definition, err := ParseDefinition(`{"name":"policy-closeout","steps":[{"name":"execute","kind":"scheduler"}]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	result, err := engine.RunDefinition(context.Background(), domain.Task{
		ID:          "task-policy-budget",
		TraceID:     "trace-policy-budget",
		Title:       "Over-budget rollout",
		BudgetCents: 15000,
		Metadata:    map[string]string{"team": "growth"},
	}, definition, "run-policy-budget", RunOptions{})
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	snapshot, err := q.GetTask(context.Background(), "task-policy-budget")
	if err != nil {
		t.Fatalf("get queued task: %v", err)
	}
	if snapshot.Task.State != domain.TaskQueued || snapshot.Leased {
		t.Fatalf("expected task requeued by policy budget default, got %+v", snapshot)
	}
	if result.Quota.ConcurrentLimit != 8 || result.Quota.MaxQueueDepth != 64 || result.Quota.BudgetRemaining != 10000 {
		t.Fatalf("expected standard policy quota defaults, got %+v", result.Quota)
	}
	if len(result.Events) == 0 || result.Events[len(result.Events)-1].Type != domain.EventTaskRetried {
		t.Fatalf("expected retry event after policy budget rejection, got %+v", result.Events)
	}
	if result.Status.LastResult != "budget exceeded" {
		t.Fatalf("expected runtime status to report budget rejection, got %+v", result.Status)
	}
}

func TestAcceptanceGateRejectsMissingEvidence(t *testing.T) {
	decision := AcceptanceGate{}.Evaluate(domain.Task{
		ID:                 "task-evidence",
		AcceptanceCriteria: []string{"go test ./..."},
		ValidationPlan:     []string{"git log -1 --stat"},
	}, []string{"go test ./..."}, nil)
	if decision.Status != "rejected" || decision.Passed {
		t.Fatalf("expected rejected decision, got %+v", decision)
	}
	if len(decision.MissingValidationSteps) != 1 || decision.MissingValidationSteps[0] != "git log -1 --stat" {
		t.Fatalf("unexpected missing validation steps: %+v", decision)
	}
}

func TestEngineRunDefinitionMarksValidationAndCloseoutStepsFailedWhenIncomplete(t *testing.T) {
	q := queue.NewMemoryQueue()
	recorder := observability.NewRecorder()
	bus := events.NewBus()
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	runtime := &worker.Runtime{
		WorkerID:    "worker-4",
		Queue:       q,
		Scheduler:   scheduler.New(),
		Registry:    executor.NewRegistry(testRunner{kind: domain.ExecutorLocal, result: executor.Result{Success: true, Message: "ok"}}),
		Bus:         bus,
		Recorder:    recorder,
		LeaseTTL:    100 * time.Millisecond,
		TaskTimeout: time.Second,
	}
	engine := &Engine{
		Runtime:  runtime,
		Recorder: recorder,
		Queue:    q,
		Quota:    scheduler.QuotaSnapshot{ConcurrentLimit: 4, BudgetRemaining: 5000},
		Now:      func() time.Time { return time.Unix(1700000200, 0).UTC() },
	}
	definition, err := ParseDefinition(`{"name":"release-closeout","steps":[{"name":"execute","kind":"scheduler"},{"name":"validate","kind":"validation"},{"name":"closeout","kind":"closeout"}],"validation_evidence":["go test ./..."]}`)
	if err != nil {
		t.Fatalf("parse definition: %v", err)
	}
	result, err := engine.RunDefinition(context.Background(), domain.Task{
		ID:                 "task-3",
		Title:              "Release build",
		AcceptanceCriteria: []string{"go test ./..."},
		ValidationPlan:     []string{"git log -1 --stat"},
	}, definition, "run-3", RunOptions{
		ValidationEvidence: []string{"go test ./..."},
		GitPushSucceeded:   false,
	})
	if err != nil {
		t.Fatalf("run definition: %v", err)
	}
	if result.WorkflowRun.Status != WorkflowRunFailed {
		t.Fatalf("expected failed workflow run, got %+v", result.WorkflowRun)
	}
	if len(result.WorkflowRun.Steps) != 3 {
		t.Fatalf("expected three workflow steps, got %+v", result.WorkflowRun.Steps)
	}
	if result.WorkflowRun.Steps[1].Status != WorkflowStepFailed || result.WorkflowRun.Steps[2].Status != WorkflowStepFailed {
		t.Fatalf("expected validation and closeout steps failed, got %+v", result.WorkflowRun.Steps)
	}
}
