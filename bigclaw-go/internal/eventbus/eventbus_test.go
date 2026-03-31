package eventbus

import (
	"path/filepath"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestEventBusPRCommentApprovesWaitingRunAndPersistsLedger(t *testing.T) {
	t.Parallel()

	ledger := &ObservabilityLedger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	task := domain.Task{ID: "BIG-203-pr", Source: "github", Title: "PR approval"}
	run := TaskRunFromTask(task, "run-pr-1", "vm")
	run.Finalize("needs-approval", "waiting for reviewer comment")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
	}

	bus := NewEventBus(ledger)
	var seenStatuses []string
	bus.Subscribe(PullRequestCommentEvent, func(_ BusEvent, current TaskRun) {
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
		t.Fatalf("publish: %v", err)
	}

	if updated.Status != "approved" {
		t.Fatalf("status = %q, want approved", updated.Status)
	}
	if updated.Summary != "LGTM, merge when green." {
		t.Fatalf("summary = %q", updated.Summary)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("seen statuses = %#v", seenStatuses)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "approved" {
		t.Fatalf("persisted = %#v", persisted)
	}
	if !hasAudit(persisted[0].Audits, "collaboration.comment", nil) {
		t.Fatalf("missing collaboration comment audit: %#v", persisted[0].Audits)
	}
	if !hasAudit(persisted[0].Audits, "event_bus.transition", map[string]any{"previous_status": "needs-approval"}) {
		t.Fatalf("missing transition audit: %#v", persisted[0].Audits)
	}
}

func TestEventBusCICompletedMarksRunCompleted(t *testing.T) {
	t.Parallel()

	ledger := &ObservabilityLedger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	task := domain.Task{ID: "BIG-203-ci", Source: "github", Title: "CI completion"}
	run := TaskRunFromTask(task, "run-ci-1", "docker")
	run.Finalize("approved", "waiting for CI")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
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
		t.Fatalf("publish: %v", err)
	}

	if updated.Status != "completed" {
		t.Fatalf("status = %q, want completed", updated.Status)
	}
	if updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("summary = %q", updated.Summary)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "completed" {
		t.Fatalf("persisted = %#v", persisted)
	}
	if !hasAudit(persisted[0].Audits, "event_bus.event", map[string]any{"event_type": CICompletedEvent}) {
		t.Fatalf("missing event audit: %#v", persisted[0].Audits)
	}
}

func TestEventBusTaskFailedMarksRunFailed(t *testing.T) {
	t.Parallel()

	ledger := &ObservabilityLedger{Path: filepath.Join(t.TempDir(), "ledger.json")}
	task := domain.Task{ID: "BIG-203-fail", Source: "scheduler", Title: "Task failure"}
	run := TaskRunFromTask(task, "run-fail-1", "docker")
	if err := ledger.Append(run); err != nil {
		t.Fatalf("append run: %v", err)
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
		t.Fatalf("publish: %v", err)
	}

	if updated.Status != "failed" {
		t.Fatalf("status = %q, want failed", updated.Status)
	}
	if updated.Summary != "sandbox command exited 137" {
		t.Fatalf("summary = %q", updated.Summary)
	}

	persisted, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}
	if len(persisted) != 1 || persisted[0].Status != "failed" {
		t.Fatalf("persisted = %#v", persisted)
	}
	if !hasAudit(persisted[0].Audits, "event_bus.transition", map[string]any{"status": "failed"}) {
		t.Fatalf("missing transition audit: %#v", persisted[0].Audits)
	}
}

func hasAudit(audits []AuditRecord, action string, expected map[string]any) bool {
	for _, audit := range audits {
		if audit.Action != action {
			continue
		}
		matched := true
		for key, want := range expected {
			if audit.Details[key] != want {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
