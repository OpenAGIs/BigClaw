package runbus

import (
	"path/filepath"
	"testing"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	t.Parallel()

	ledger := &Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	run := Run{
		RunID:   "run-pr-1",
		TaskID:  "BIG-203-pr",
		Status:  "needs-approval",
		Summary: "waiting for reviewer comment",
	}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert seed run: %v", err)
	}

	bus := NewBus(ledger)
	seenStatuses := make([]string, 0, 1)
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current *Run) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, err := bus.Publish(BusEvent{
		EventType: PullRequestCommentEvent,
		RunID:     run.RunID,
		Actor:     "reviewer",
		Details: map[string]any{
			"decision": "approved",
			"body":     "LGTM, merge when green.",
			"mentions": []any{"ops"},
		},
	})
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}

	if updated.Status != "approved" {
		t.Fatalf("status = %q, want approved", updated.Status)
	}
	if updated.Summary != "LGTM, merge when green." {
		t.Fatalf("summary = %q", updated.Summary)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("subscriber statuses = %v", seenStatuses)
	}

	runs, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("ledger runs = %d, want 1", len(runs))
	}
	persisted := runs[0]
	if persisted.Status != "approved" {
		t.Fatalf("persisted status = %q, want approved", persisted.Status)
	}
	if !hasAudit(persisted.Audits, "collaboration.comment", nil) {
		t.Fatalf("expected collaboration.comment audit, got %#v", persisted.Audits)
	}
	if !hasAudit(persisted.Audits, "event_bus.transition", func(details map[string]any) bool {
		return toString(details["previous_status"]) == "needs-approval"
	}) {
		t.Fatalf("expected transition audit with previous_status needs-approval, got %#v", persisted.Audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	t.Parallel()

	ledger := &Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	run := Run{
		RunID:   "run-ci-1",
		TaskID:  "BIG-203-ci",
		Status:  "approved",
		Summary: "waiting for CI",
	}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert seed run: %v", err)
	}

	bus := NewBus(ledger)
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
		t.Fatalf("publish event: %v", err)
	}

	if updated.Status != "completed" {
		t.Fatalf("status = %q, want completed", updated.Status)
	}
	if updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("summary = %q", updated.Summary)
	}

	runs, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("ledger runs = %d, want 1", len(runs))
	}
	persisted := runs[0]
	if persisted.Status != "completed" {
		t.Fatalf("persisted status = %q, want completed", persisted.Status)
	}
	if !hasAudit(persisted.Audits, "event_bus.event", func(details map[string]any) bool {
		return toString(details["event_type"]) == CICompletedEvent
	}) {
		t.Fatalf("expected event_bus.event audit for ci.completed, got %#v", persisted.Audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	t.Parallel()

	ledger := &Ledger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	run := Run{
		RunID:   "run-fail-1",
		TaskID:  "BIG-203-fail",
		Status:  "queued",
		Summary: "awaiting worker execution",
	}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("upsert seed run: %v", err)
	}

	bus := NewBus(ledger)
	updated, err := bus.Publish(BusEvent{
		EventType: TaskFailedEvent,
		RunID:     run.RunID,
		Actor:     "worker",
		Details: map[string]any{
			"error": "sandbox command exited 137",
		},
	})
	if err != nil {
		t.Fatalf("publish event: %v", err)
	}

	if updated.Status != "failed" {
		t.Fatalf("status = %q, want failed", updated.Status)
	}
	if updated.Summary != "sandbox command exited 137" {
		t.Fatalf("summary = %q", updated.Summary)
	}

	runs, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("ledger runs = %d, want 1", len(runs))
	}
	persisted := runs[0]
	if persisted.Status != "failed" {
		t.Fatalf("persisted status = %q, want failed", persisted.Status)
	}
	if !hasAudit(persisted.Audits, "event_bus.transition", func(details map[string]any) bool {
		return toString(details["status"]) == "failed"
	}) {
		t.Fatalf("expected event_bus.transition audit with failed status, got %#v", persisted.Audits)
	}
}

func hasAudit(audits []Audit, action string, match func(map[string]any) bool) bool {
	for _, audit := range audits {
		if audit.Action != action {
			continue
		}
		if match == nil || match(audit.Details) {
			return true
		}
	}
	return false
}
