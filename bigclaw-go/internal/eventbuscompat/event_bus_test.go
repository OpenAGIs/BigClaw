package eventbuscompat

import (
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{ID: "BIG-203-pr", Source: "github", Title: "PR approval"}
	run := NewRun(task, "run-pr-1", "vm")
	run.Finalize("needs-approval", "waiting for reviewer comment")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewEventBus(ledger)
	seenStatuses := make([]string, 0, 1)
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current *TaskRun) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, err := bus.Publish(BusEvent{
		EventType: PullRequestCommentEvent,
		RunID:     run.RunID,
		Actor:     "reviewer",
		Details: map[string]any{
			"decision": "approved",
			"body":     "LGTM, merge when green.",
			"mentions": []string{"ops"},
		},
	})
	if err != nil {
		t.Fatalf("publish PR comment event: %v", err)
	}

	if updated.Status != "approved" || updated.Summary != "LGTM, merge when green." {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("unexpected subscriber statuses: %+v", seenStatuses)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0]["status"] != "approved" {
		t.Fatalf("unexpected persisted entries: %+v", persisted)
	}
	audits := persisted[0]["audits"].([]any)
	if !hasAuditAction(audits, "collaboration.comment") {
		t.Fatalf("expected collaboration.comment audit, got %+v", audits)
	}
	if !hasTransitionPreviousStatus(audits, "needs-approval") {
		t.Fatalf("expected transition audit with previous_status needs-approval, got %+v", audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{ID: "BIG-203-ci", Source: "github", Title: "CI completion"}
	run := NewRun(task, "run-ci-1", "docker")
	run.Finalize("approved", "waiting for CI")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(BusEvent{
		EventType: CICompletedEvent,
		RunID:     run.RunID,
		Actor:     "github-actions",
		Details: map[string]any{
			"workflow":   "pytest",
			"conclusion": "success",
		},
	})
	if err != nil {
		t.Fatalf("publish CI event: %v", err)
	}
	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	audits := persisted[0]["audits"].([]any)
	if !hasEventTypeAudit(audits, CICompletedEvent) {
		t.Fatalf("expected event_bus.event audit for CI event, got %+v", audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	task := domain.Task{ID: "BIG-203-fail", Source: "scheduler", Title: "Task failure"}
	run := NewRun(task, "run-fail-1", "docker")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(BusEvent{
		EventType: TaskFailedEvent,
		RunID:     run.RunID,
		Actor:     "worker",
		Details: map[string]any{
			"error": "sandbox command exited 137",
		},
	})
	if err != nil {
		t.Fatalf("publish task failed event: %v", err)
	}
	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	audits := persisted[0]["audits"].([]any)
	if !hasTransitionStatus(audits, "failed") {
		t.Fatalf("expected failed transition audit, got %+v", audits)
	}
}

func hasAuditAction(audits []any, want string) bool {
	for _, item := range audits {
		audit, ok := item.(map[string]any)
		if ok && audit["action"] == want {
			return true
		}
	}
	return false
}

func hasEventTypeAudit(audits []any, want string) bool {
	for _, item := range audits {
		audit, ok := item.(map[string]any)
		if !ok || audit["action"] != "event_bus.event" {
			continue
		}
		details, ok := audit["details"].(map[string]any)
		if ok && details["event_type"] == want {
			return true
		}
	}
	return false
}

func hasTransitionPreviousStatus(audits []any, want string) bool {
	for _, item := range audits {
		audit, ok := item.(map[string]any)
		if !ok || audit["action"] != "event_bus.transition" {
			continue
		}
		details, ok := audit["details"].(map[string]any)
		if ok && details["previous_status"] == want {
			return true
		}
	}
	return false
}

func hasTransitionStatus(audits []any, want string) bool {
	for _, item := range audits {
		audit, ok := item.(map[string]any)
		if !ok || audit["action"] != "event_bus.transition" {
			continue
		}
		details, ok := audit["details"].(map[string]any)
		if ok && details["status"] == want {
			return true
		}
	}
	return false
}
