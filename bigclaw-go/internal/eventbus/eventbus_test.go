package eventbus

import (
	"reflect"
	"testing"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(t.TempDir() + "/ledger.json")
	if err := ledger.Append(RunRecord{
		RunID:   "run-pr-1",
		TaskID:  "BIG-203-pr",
		Status:  "needs-approval",
		Summary: "waiting for reviewer comment",
	}); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	var seen []string
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current RunRecord) {
		seen = append(seen, current.Status)
	})

	updated, err := bus.Publish(BusEvent{
		EventType: PullRequestCommentEvent,
		RunID:     "run-pr-1",
		Actor:     "reviewer",
		Details: map[string]any{
			"decision": "approved",
			"body":     "LGTM, merge when green.",
			"mentions": []string{"ops"},
		},
	})
	if err != nil {
		t.Fatalf("publish pr comment: %v", err)
	}
	if updated.Status != "approved" || updated.Summary != "LGTM, merge when green." {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if !reflect.DeepEqual(seen, []string{"approved"}) {
		t.Fatalf("unexpected seen statuses: %+v", seen)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if persisted[0].Status != "approved" {
		t.Fatalf("expected approved persisted status, got %+v", persisted[0])
	}
	if !hasAuditAction(persisted[0].Audits, "collaboration.comment") {
		t.Fatalf("expected collaboration comment audit, got %+v", persisted[0].Audits)
	}
	if !hasTransitionWithPrevious(persisted[0].Audits, "needs-approval") {
		t.Fatalf("expected transition from needs-approval, got %+v", persisted[0].Audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(t.TempDir() + "/ledger.json")
	if err := ledger.Append(RunRecord{RunID: "run-ci-1", TaskID: "BIG-203-ci", Status: "approved", Summary: "waiting for CI"}); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(BusEvent{
		EventType: CICompletedEvent,
		RunID:     "run-ci-1",
		Actor:     "github-actions",
		Details: map[string]any{
			"workflow":   "pytest",
			"conclusion": "success",
		},
	})
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
	if persisted[0].Status != "completed" {
		t.Fatalf("expected completed persisted status, got %+v", persisted[0])
	}
	if !hasEventTypeAudit(persisted[0].Audits, CICompletedEvent) {
		t.Fatalf("expected ci completed event audit, got %+v", persisted[0].Audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(t.TempDir() + "/ledger.json")
	if err := ledger.Append(RunRecord{RunID: "run-fail-1", TaskID: "BIG-203-fail", Status: "queued"}); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	updated, err := bus.Publish(BusEvent{
		EventType: TaskFailedEvent,
		RunID:     "run-fail-1",
		Actor:     "worker",
		Details: map[string]any{
			"error": "sandbox command exited 137",
		},
	})
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
	if persisted[0].Status != "failed" {
		t.Fatalf("expected failed persisted status, got %+v", persisted[0])
	}
	if !hasTransitionStatus(persisted[0].Audits, "failed") {
		t.Fatalf("expected failed transition audit, got %+v", persisted[0].Audits)
	}
}

func hasAuditAction(audits []AuditRecord, action string) bool {
	for _, audit := range audits {
		if audit.Action == action {
			return true
		}
	}
	return false
}

func hasTransitionWithPrevious(audits []AuditRecord, previous string) bool {
	for _, audit := range audits {
		if audit.Action == "event_bus.transition" && audit.Details["previous_status"] == previous {
			return true
		}
	}
	return false
}

func hasTransitionStatus(audits []AuditRecord, status string) bool {
	for _, audit := range audits {
		if audit.Action == "event_bus.transition" && audit.Details["status"] == status {
			return true
		}
	}
	return false
}

func hasEventTypeAudit(audits []AuditRecord, eventType string) bool {
	for _, audit := range audits {
		if audit.Action == "event_bus.event" && audit.Details["event_type"] == eventType {
			return true
		}
	}
	return false
}
