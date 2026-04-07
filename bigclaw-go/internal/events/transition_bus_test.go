package events

import "testing"

func TestTransitionBusPRCommentApprovesWaitingRun(t *testing.T) {
	bus := NewTransitionBus()
	run := &TransitionRun{RunID: "run-pr-1", Status: "needs-approval", Summary: "waiting for reviewer comment"}
	bus.RegisterRun(run)

	seenStatuses := []string{}
	bus.Subscribe(PullRequestCommentEvent, func(_ TransitionEvent, current *TransitionRun) {
		seenStatuses = append(seenStatuses, current.Status)
	})

	updated, ok := bus.Publish(TransitionEvent{
		EventType: PullRequestCommentEvent,
		RunID:     "run-pr-1",
		Actor:     "reviewer",
		Details: map[string]any{
			"decision": "approved",
			"body":     "LGTM, merge when green.",
			"mentions": []string{"ops"},
		},
	})
	if !ok {
		t.Fatal("expected registered run")
	}
	if updated.Status != "approved" || updated.Summary != "LGTM, merge when green." {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if len(seenStatuses) != 1 || seenStatuses[0] != "approved" {
		t.Fatalf("unexpected subscriber statuses: %+v", seenStatuses)
	}
	if len(updated.Comments) != 1 || updated.Comments[0].Body != "LGTM, merge when green." {
		t.Fatalf("expected recorded PR comment, got %+v", updated.Comments)
	}
	if len(updated.Audits) < 2 || updated.Audits[1].Action != "event_bus.transition" || updated.Audits[1].Details["previous_status"] != "needs-approval" {
		t.Fatalf("expected transition audit, got %+v", updated.Audits)
	}
}

func TestTransitionBusCICompletedMarksRunCompleted(t *testing.T) {
	bus := NewTransitionBus()
	run := &TransitionRun{RunID: "run-ci-1", Status: "approved", Summary: "waiting for CI"}
	bus.RegisterRun(run)

	updated, ok := bus.Publish(TransitionEvent{
		EventType: CICompletedEvent,
		RunID:     "run-ci-1",
		Actor:     "github-actions",
		Details: map[string]any{
			"workflow":   "pytest",
			"conclusion": "success",
		},
	})
	if !ok {
		t.Fatal("expected registered run")
	}
	if updated.Status != "completed" || updated.Summary != "CI workflow pytest completed with success" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if updated.Audits[0].Action != "event_bus.event" || updated.Audits[0].Details["event_type"] != CICompletedEvent {
		t.Fatalf("expected event audit, got %+v", updated.Audits)
	}
}

func TestTransitionBusTaskFailedMarksRunFailed(t *testing.T) {
	bus := NewTransitionBus()
	run := &TransitionRun{RunID: "run-fail-1", Status: "running"}
	bus.RegisterRun(run)

	updated, ok := bus.Publish(TransitionEvent{
		EventType: TaskFailedEvent,
		RunID:     "run-fail-1",
		Actor:     "worker",
		Details: map[string]any{
			"error": "sandbox command exited 137",
		},
	})
	if !ok {
		t.Fatal("expected registered run")
	}
	if updated.Status != "failed" || updated.Summary != "sandbox command exited 137" {
		t.Fatalf("unexpected updated run: %+v", updated)
	}
	if updated.Audits[1].Action != "event_bus.transition" || updated.Audits[1].Details["status"] != "failed" {
		t.Fatalf("expected failure transition audit, got %+v", updated.Audits)
	}
}
