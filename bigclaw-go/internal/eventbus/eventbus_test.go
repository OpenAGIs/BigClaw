package eventbus

import (
	"path/filepath"
	"testing"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	if err := ledger.Append(Run{RunID: "run-pr-1", Status: "needs-approval", Summary: "waiting for reviewer comment"}); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := New(ledger)
	var seenStatuses []string
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
			"mentions": []string{"ops"},
		},
	})
	if err != nil {
		t.Fatalf("publish approval comment: %v", err)
	}
	if updated.Status != "approved" || updated.Summary != "LGTM, merge when green." {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("unexpected seen statuses: %+v", seenStatuses)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "approved" {
		t.Fatalf("unexpected persisted runs: %+v", persisted)
	}
	if persisted[0].Audits[0].Action != "collaboration.comment" || persisted[0].Audits[1].Details["previous_status"] != "needs-approval" {
		t.Fatalf("unexpected persisted audits: %+v", persisted[0].Audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	if err := ledger.Append(Run{RunID: "run-ci-1", Status: "approved", Summary: "waiting for CI"}); err != nil {
		t.Fatalf("append run: %v", err)
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
		t.Fatalf("publish ci completion: %v", err)
	}
	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if persisted[0].Audits[0].Action != "event_bus.event" || persisted[0].Audits[0].Details["event_type"] != CICompletedEvent {
		t.Fatalf("unexpected persisted audits: %+v", persisted[0].Audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	if err := ledger.Append(Run{RunID: "run-fail-1", Status: "queued"}); err != nil {
		t.Fatalf("append run: %v", err)
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
		t.Fatalf("publish task failure: %v", err)
	}
	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if persisted[0].Audits[0].Action != "event_bus.transition" || persisted[0].Audits[0].Details["status"] != "failed" {
		t.Fatalf("unexpected persisted audits: %+v", persisted[0].Audits)
	}
}
