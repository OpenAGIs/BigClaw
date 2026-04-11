package control

import (
	"testing"
	"time"
)

func TestControllerPauseAndTakeoverLifecycle(t *testing.T) {
	controller := New()
	now := time.Now()
	snapshot := controller.Pause("ops", "maintenance", now)
	if !snapshot.Paused || snapshot.PauseReason != "maintenance" {
		t.Fatalf("expected paused snapshot, got %+v", snapshot)
	}
	if !controller.IsPaused() {
		t.Fatal("expected controller paused")
	}

	takeover := controller.Takeover("task-1", "alice", "bob", "investigating", now.Add(time.Second))
	if !takeover.Active || takeover.Owner != "alice" || takeover.Reviewer != "bob" {
		t.Fatalf("unexpected takeover: %+v", takeover)
	}
	annotated := controller.Annotate("task-1", "alice", "added note", now.Add(2*time.Second))
	if len(annotated.Notes) != 2 {
		t.Fatalf("expected accumulated notes, got %+v", annotated)
	}
	reassigned, ok := controller.Reassign("task-1", "carol", "dave", "alice", "handoff to release captain", now.Add(2500*time.Millisecond))
	if !ok || reassigned.Owner != "carol" || reassigned.Reviewer != "dave" || len(reassigned.Notes) != 3 {
		t.Fatalf("expected reassigned takeover with preserved notes, got %+v ok=%v", reassigned, ok)
	}
	active := controller.ActiveTakeovers()
	if len(active) != 1 || active[0].TaskID != "task-1" {
		t.Fatalf("expected active takeover list, got %+v", active)
	}

	released, ok := controller.Release("task-1", "alice", "handoff complete", now.Add(3*time.Second))
	if !ok || released.Active {
		t.Fatalf("expected released takeover, got %+v ok=%v", released, ok)
	}
	if len(controller.ActiveTakeovers()) != 0 {
		t.Fatalf("expected no active takeovers after release, got %+v", controller.ActiveTakeovers())
	}

	snapshot = controller.Resume("ops", now.Add(4*time.Second))
	if snapshot.Paused {
		t.Fatalf("expected resumed snapshot, got %+v", snapshot)
	}
}

func TestControllerSnapshotAndTakeoverStatusHelpers(t *testing.T) {
	controller := New()
	now := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)

	if takeover, ok := controller.TakeoverStatus("missing"); ok || takeover.TaskID != "" || takeover.Active || len(takeover.Notes) != 0 {
		t.Fatalf("expected missing takeover lookup to fail, got %+v ok=%v", takeover, ok)
	}

	controller.Pause("ops", "incident", now)
	controller.Takeover("task-1", "alice", "bob", "investigating", now.Add(time.Second))

	snapshot := controller.Snapshot()
	if !snapshot.Paused || snapshot.PauseActor != "ops" || snapshot.PauseReason != "incident" || snapshot.ActiveTakeovers != 1 {
		t.Fatalf("unexpected controller snapshot: %+v", snapshot)
	}
	if snapshot.PausedAt != now {
		t.Fatalf("expected snapshot pause timestamp to be preserved, got %+v", snapshot)
	}

	takeover, ok := controller.TakeoverStatus("task-1")
	if !ok || !takeover.Active || takeover.TaskID != "task-1" || takeover.Owner != "alice" || takeover.Reviewer != "bob" {
		t.Fatalf("unexpected takeover status lookup: %+v ok=%v", takeover, ok)
	}
}
