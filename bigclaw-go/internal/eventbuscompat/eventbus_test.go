package eventbuscompat

import (
	"path/filepath"
	"testing"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	run := RunRecord{RunID: "run-pr-1"}
	run.Finalize("needs-approval", "waiting for reviewer comment")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	seenStatuses := make([]string, 0, 1)
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current *RunRecord) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, err := bus.Publish(NewBusEvent(PullRequestCommentEvent, run.RunID, "reviewer", map[string]any{
		"decision": "approved",
		"body":     "LGTM, merge when green.",
		"mentions": []any{"ops"},
	}))
	if err != nil {
		t.Fatalf("publish comment event: %v", err)
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
	if persisted[0].Status != "approved" {
		t.Fatalf("expected approved persisted status, got %+v", persisted[0])
	}
	if !containsAudit(persisted[0].Audits, "collaboration.comment", "") || !containsAuditDetail(persisted[0].Audits, "event_bus.transition", "previous_status", "needs-approval") {
		t.Fatalf("expected collaboration comment and transition audit, got %+v", persisted[0].Audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	run := RunRecord{RunID: "run-ci-1"}
	run.Finalize("approved", "waiting for CI")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(NewBusEvent(CICompletedEvent, run.RunID, "github-actions", map[string]any{
		"workflow":   "pytest",
		"conclusion": "success",
	}))
	if err != nil {
		t.Fatalf("publish ci event: %v", err)
	}
	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if persisted[0].Status != "completed" || !containsAuditDetail(persisted[0].Audits, "event_bus.event", "event_type", CICompletedEvent) {
		t.Fatalf("unexpected persisted run: %+v", persisted[0])
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	run := RunRecord{RunID: "run-fail-1"}
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(NewBusEvent(TaskFailedEvent, run.RunID, "worker", map[string]any{
		"error": "sandbox command exited 137",
	}))
	if err != nil {
		t.Fatalf("publish failure event: %v", err)
	}
	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if persisted[0].Status != "failed" || !containsAuditDetail(persisted[0].Audits, "event_bus.transition", "status", "failed") {
		t.Fatalf("unexpected persisted run: %+v", persisted[0])
	}
}

func containsAudit(audits []AuditRecord, action, outcome string) bool {
	for _, audit := range audits {
		if audit.Action == action && (outcome == "" || audit.Outcome == outcome) {
			return true
		}
	}
	return false
}

func containsAuditDetail(audits []AuditRecord, action, key, want string) bool {
	for _, audit := range audits {
		if audit.Action != action {
			continue
		}
		if got, ok := audit.Details[key]; ok && got == want {
			return true
		}
	}
	return false
}
