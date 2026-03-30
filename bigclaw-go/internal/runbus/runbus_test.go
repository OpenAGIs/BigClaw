package runbus

import (
	"path/filepath"
	"testing"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	run := Run{RunID: "run-pr-1", Status: "needs-approval", Summary: "waiting for reviewer comment"}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := New(ledger)
	seenStatuses := make([]string, 0, 1)
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current Run) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, err := bus.Publish(BusEvent{
		EventType: PullRequestCommentEvent,
		RunID:     "run-pr-1",
		Actor:     "reviewer",
		Details: map[string]any{
			"decision": "approved",
			"body":     "LGTM, merge when green.",
			"mentions": []any{"ops"},
		},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	if updated.Status != "approved" || updated.Summary != "LGTM, merge when green." {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("unexpected subscriber statuses: %+v", seenStatuses)
	}

	persisted, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "approved" {
		t.Fatalf("unexpected persisted runs: %+v", persisted)
	}
	if !hasAuditAction(persisted[0], "collaboration.comment") {
		t.Fatalf("expected collaboration comment audit, got %+v", persisted[0].Audits)
	}
	if !hasTransitionWithPrevious(persisted[0], "needs-approval") {
		t.Fatalf("expected transition audit with previous status, got %+v", persisted[0].Audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	run := Run{RunID: "run-ci-1", Status: "approved", Summary: "waiting for CI"}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := New(ledger)
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
		t.Fatalf("publish: %v", err)
	}
	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}

	persisted, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "completed" {
		t.Fatalf("unexpected persisted runs: %+v", persisted)
	}
	if !hasEventTypeAudit(persisted[0], CICompletedEvent) {
		t.Fatalf("expected ci event audit, got %+v", persisted[0].Audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	run := Run{RunID: "run-fail-1"}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := New(ledger)
	updated, err := bus.Publish(BusEvent{
		EventType: TaskFailedEvent,
		RunID:     "run-fail-1",
		Actor:     "worker",
		Details: map[string]any{
			"error": "sandbox command exited 137",
		},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}

	persisted, err := ledger.LoadRuns()
	if err != nil {
		t.Fatalf("load runs: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "failed" {
		t.Fatalf("unexpected persisted runs: %+v", persisted)
	}
	if !hasTransitionStatus(persisted[0], "failed") {
		t.Fatalf("expected failed transition audit, got %+v", persisted[0].Audits)
	}
}

func hasAuditAction(run Run, action string) bool {
	for _, audit := range run.Audits {
		if audit.Action == action {
			return true
		}
	}
	return false
}

func hasTransitionWithPrevious(run Run, previous string) bool {
	for _, audit := range run.Audits {
		if audit.Action == "event_bus.transition" && audit.Details["previous_status"] == previous {
			return true
		}
	}
	return false
}

func hasTransitionStatus(run Run, status string) bool {
	for _, audit := range run.Audits {
		if audit.Action == "event_bus.transition" && audit.Details["status"] == status {
			return true
		}
	}
	return false
}

func hasEventTypeAudit(run Run, eventType string) bool {
	for _, audit := range run.Audits {
		if audit.Action == "event_bus.event" && audit.Details["event_type"] == eventType {
			return true
		}
	}
	return false
}
