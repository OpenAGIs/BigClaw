package eventbus

import (
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{ID: "BIG-203-pr", Source: "github", Title: "PR approval"}
	run := NewTaskRun(task, "run-pr-1", "vm")
	run.Finalize("needs-approval", "waiting for reviewer comment")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(&ledger)
	seenStatuses := []string{}
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current *TaskRun) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, err := bus.Publish(NewBusEvent(PullRequestCommentEvent, run.RunID, "reviewer", map[string]any{
		"decision": "approved",
		"body":     "LGTM, merge when green.",
		"mentions": []string{"ops"},
	}))
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}
	if updated.Status != "approved" || updated.Summary != "LGTM, merge when green." {
		t.Fatalf("unexpected updated run %+v", updated)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("unexpected seen statuses %+v", seenStatuses)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0]["status"] != "approved" {
		t.Fatalf("unexpected persisted ledger %+v", persisted)
	}
	assertAuditAction(t, persisted[0], "collaboration.comment")
	assertTransitionPreviousStatus(t, persisted[0], "needs-approval")
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{ID: "BIG-203-ci", Source: "github", Title: "CI completion"}
	run := NewTaskRun(task, "run-ci-1", "docker")
	run.Finalize("approved", "waiting for CI")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(&ledger)
	updated, err := bus.Publish(NewBusEvent(CICompletedEvent, run.RunID, "github-actions", map[string]any{
		"workflow":   "pytest",
		"conclusion": "success",
	}))
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}
	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run %+v", updated)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0]["status"] != "completed" {
		t.Fatalf("unexpected persisted ledger %+v", persisted)
	}
	assertEventTypeAudit(t, persisted[0], CICompletedEvent)
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{ID: "BIG-203-fail", Source: "scheduler", Title: "Task failure"}
	run := NewTaskRun(task, "run-fail-1", "docker")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(&ledger)
	updated, err := bus.Publish(NewBusEvent(TaskFailedEvent, run.RunID, "worker", map[string]any{
		"error": "sandbox command exited 137",
	}))
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}
	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run %+v", updated)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0]["status"] != "failed" {
		t.Fatalf("unexpected persisted ledger %+v", persisted)
	}
	assertTransitionStatus(t, persisted[0], "failed")
}

func assertAuditAction(t *testing.T, entry map[string]any, want string) {
	t.Helper()
	for _, item := range entry["audits"].([]any) {
		audit := item.(map[string]any)
		if audit["action"] == want {
			return
		}
	}
	t.Fatalf("expected audit action %q in %+v", want, entry["audits"])
}

func assertTransitionPreviousStatus(t *testing.T, entry map[string]any, want string) {
	t.Helper()
	for _, item := range entry["audits"].([]any) {
		audit := item.(map[string]any)
		if audit["action"] != "event_bus.transition" {
			continue
		}
		details := audit["details"].(map[string]any)
		if details["previous_status"] == want {
			return
		}
	}
	t.Fatalf("expected transition previous_status=%q in %+v", want, entry["audits"])
}

func assertEventTypeAudit(t *testing.T, entry map[string]any, want string) {
	t.Helper()
	for _, item := range entry["audits"].([]any) {
		audit := item.(map[string]any)
		if audit["action"] != "event_bus.event" {
			continue
		}
		details := audit["details"].(map[string]any)
		if details["event_type"] == want {
			return
		}
	}
	t.Fatalf("expected event_bus.event with event_type=%q in %+v", want, entry["audits"])
}

func assertTransitionStatus(t *testing.T, entry map[string]any, want string) {
	t.Helper()
	for _, item := range entry["audits"].([]any) {
		audit := item.(map[string]any)
		if audit["action"] != "event_bus.transition" {
			continue
		}
		details := audit["details"].(map[string]any)
		if details["status"] == want {
			return
		}
	}
	t.Fatalf("expected transition status=%q in %+v", want, entry["audits"])
}
