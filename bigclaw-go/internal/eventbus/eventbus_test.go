package eventbus

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(t.TempDir() + "/ledger.json")
	task := domain.Task{ID: "BIG-203-pr", Source: "github", Title: "PR approval"}
	run := NewTaskRun(task, "run-pr-1", "vm")
	run.Finalize("needs-approval", "waiting for reviewer comment")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewEventBus(ledger)
	var seenStatuses []string
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current *TaskRun) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, err := bus.Publish(NewBusEvent(PullRequestCommentEvent, run.RunID, "reviewer", map[string]any{
		"decision": "approved",
		"body":     "LGTM, merge when green.",
		"mentions": []any{"ops"},
	}))
	if err != nil {
		t.Fatalf("publish pr comment: %v", err)
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
	if got := persisted[0]["status"]; got != "approved" {
		t.Fatalf("expected persisted approved status, got %+v", persisted[0])
	}
	audits, ok := persisted[0]["audits"].([]any)
	if !ok {
		t.Fatalf("expected persisted audits, got %+v", persisted[0])
	}
	if !hasAuditAction(audits, "collaboration.comment") {
		t.Fatalf("expected collaboration.comment audit, got %+v", audits)
	}
	if !hasTransition(audits, "needs-approval") {
		t.Fatalf("expected event_bus.transition with previous_status needs-approval, got %+v", audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(t.TempDir() + "/ledger.json")
	task := domain.Task{ID: "BIG-203-ci", Source: "github", Title: "CI completion"}
	run := NewTaskRun(task, "run-ci-1", "docker")
	run.Finalize("approved", "waiting for CI")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(NewBusEvent(CICompletedEvent, run.RunID, "github-actions", map[string]any{
		"workflow":   "pytest",
		"conclusion": "success",
	}))
	if err != nil {
		t.Fatalf("publish ci completed: %v", err)
	}

	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if got := persisted[0]["status"]; got != "completed" {
		t.Fatalf("expected completed status, got %+v", persisted[0])
	}
	audits := persisted[0]["audits"].([]any)
	if !hasEventType(audits, CICompletedEvent) {
		t.Fatalf("expected ci.completed event audit, got %+v", audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(t.TempDir() + "/ledger.json")
	task := domain.Task{ID: "BIG-203-fail", Source: "scheduler", Title: "Task failure"}
	run := NewTaskRun(task, "run-fail-1", "docker")
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(NewBusEvent(TaskFailedEvent, run.RunID, "worker", map[string]any{
		"error": "sandbox command exited 137",
	}))
	if err != nil {
		t.Fatalf("publish task failed: %v", err)
	}

	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if got := persisted[0]["status"]; got != "failed" {
		t.Fatalf("expected failed status, got %+v", persisted[0])
	}
	audits := persisted[0]["audits"].([]any)
	if !hasTransitionStatus(audits, "failed") {
		t.Fatalf("expected failed transition audit, got %+v", audits)
	}
}

func hasAuditAction(audits []any, action string) bool {
	for _, raw := range audits {
		entry, ok := raw.(map[string]any)
		if ok && entry["action"] == action {
			return true
		}
	}
	return false
}

func hasTransition(audits []any, previousStatus string) bool {
	for _, raw := range audits {
		entry, ok := raw.(map[string]any)
		if !ok || entry["action"] != "event_bus.transition" {
			continue
		}
		details, ok := entry["details"].(map[string]any)
		if ok && details["previous_status"] == previousStatus {
			return true
		}
	}
	return false
}

func hasEventType(audits []any, eventType string) bool {
	for _, raw := range audits {
		entry, ok := raw.(map[string]any)
		if !ok || entry["action"] != "event_bus.event" {
			continue
		}
		details, ok := entry["details"].(map[string]any)
		if ok && details["event_type"] == eventType {
			return true
		}
	}
	return false
}

func hasTransitionStatus(audits []any, status string) bool {
	for _, raw := range audits {
		entry, ok := raw.(map[string]any)
		if !ok || entry["action"] != "event_bus.transition" {
			continue
		}
		details, ok := entry["details"].(map[string]any)
		if ok && details["status"] == status {
			return true
		}
	}
	return false
}

func TestBuildSummaryDefaults(t *testing.T) {
	if got := buildSummary(NewBusEvent(TaskFailedEvent, "run", "worker", map[string]any{}), "failed"); !strings.Contains(got, "task failed") {
		t.Fatalf("unexpected default failed summary: %q", got)
	}
}
