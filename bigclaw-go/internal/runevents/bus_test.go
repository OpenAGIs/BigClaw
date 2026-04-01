package runevents

import (
	"path/filepath"
	"testing"
)

func TestPullRequestCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	run := RunRecord{RunID: "run-pr-1", Status: "needs-approval", Summary: "waiting for reviewer comment"}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewBus(&ledger)
	var seen []string
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current RunRecord) {
		seen = append(seen, current.Status)
	})

	updated, err := bus.Publish(NewBusEvent(PullRequestCommentEvent, "run-pr-1", "reviewer", map[string]string{
		"decision": "approved",
		"body":     "LGTM, merge when green.",
		"mentions": "ops",
	}))
	if err != nil {
		t.Fatalf("publish pull request comment: %v", err)
	}

	if updated.Status != "approved" {
		t.Fatalf("expected approved status, got %+v", updated)
	}
	if updated.Summary != "LGTM, merge when green." {
		t.Fatalf("expected comment summary, got %+v", updated)
	}
	if len(seen) != 1 || seen[0] != "approved" {
		t.Fatalf("unexpected subscriber statuses: %+v", seen)
	}

	records, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(records) != 1 || records[0].Status != "approved" {
		t.Fatalf("unexpected persisted records: %+v", records)
	}
	if !hasAudit(records[0].Audits, "collaboration.comment", "recorded", "") {
		t.Fatalf("expected collaboration comment audit, got %+v", records[0].Audits)
	}
	if !hasAudit(records[0].Audits, "event_bus.transition", "approved", "previous_status") {
		t.Fatalf("expected transition audit with previous status, got %+v", records[0].Audits)
	}
}

func TestCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	run := RunRecord{RunID: "run-ci-1", Status: "approved", Summary: "waiting for CI"}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewBus(&ledger)
	updated, err := bus.Publish(NewBusEvent(CICompletedEvent, "run-ci-1", "github-actions", map[string]string{
		"workflow":   "pytest",
		"conclusion": "success",
	}))
	if err != nil {
		t.Fatalf("publish ci completion: %v", err)
	}

	if updated.Status != "completed" {
		t.Fatalf("expected completed status, got %+v", updated)
	}
	if updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected summary: %+v", updated)
	}

	records, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(records) != 1 || records[0].Status != "completed" {
		t.Fatalf("unexpected persisted records: %+v", records)
	}
	if !hasAuditValue(records[0].Audits, "event_bus.event", "event_type", CICompletedEvent) {
		t.Fatalf("expected event audit for ci completion, got %+v", records[0].Audits)
	}
}

func TestTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	run := RunRecord{RunID: "run-fail-1", Status: "running"}
	if err := ledger.Upsert(run); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}

	bus := NewBus(&ledger)
	updated, err := bus.Publish(NewBusEvent(TaskFailedEvent, "run-fail-1", "worker", map[string]string{
		"error": "sandbox command exited 137",
	}))
	if err != nil {
		t.Fatalf("publish task failed: %v", err)
	}

	if updated.Status != "failed" {
		t.Fatalf("expected failed status, got %+v", updated)
	}
	if updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected summary: %+v", updated)
	}

	records, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(records) != 1 || records[0].Status != "failed" {
		t.Fatalf("unexpected persisted records: %+v", records)
	}
	if !hasAuditValue(records[0].Audits, "event_bus.transition", "status", "failed") {
		t.Fatalf("expected failed transition audit, got %+v", records[0].Audits)
	}
}

func hasAudit(audits []AuditEntry, action string, outcome string, detailKey string) bool {
	for _, audit := range audits {
		if audit.Action != action || audit.Outcome != outcome {
			continue
		}
		if detailKey == "" {
			return true
		}
		if _, ok := audit.Details[detailKey]; ok {
			return true
		}
	}
	return false
}

func hasAuditValue(audits []AuditEntry, action string, detailKey string, want string) bool {
	for _, audit := range audits {
		if audit.Action != action {
			continue
		}
		if audit.Details[detailKey] == want {
			return true
		}
	}
	return false
}
