package api

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestClawHostRecoverySurfacePayloadHandlesEmptyScope(t *testing.T) {
	surface := clawHostRecoverySurfacePayload(nil, "", "")

	if surface.Integration != "clawhost" || surface.Status != "idle" {
		t.Fatalf("unexpected empty recovery surface identity: %+v", surface)
	}
	if surface.Filters["team"] != "" || surface.Filters["project"] != "" {
		t.Fatalf("expected empty recovery filters to stay empty, got %+v", surface.Filters)
	}
	if len(surface.SupportedActions) != 6 {
		t.Fatalf("expected supported actions to stay populated, got %+v", surface.SupportedActions)
	}
	if surface.Summary.Targets != 0 || surface.Summary.RecoverableTargets != 0 || surface.Summary.DegradedTargets != 0 || surface.Summary.IsolatedTargets != 0 || surface.Summary.TakeoverRequired != 0 || surface.Summary.EvidenceArtifacts != 0 {
		t.Fatalf("expected zeroed recovery summary, got %+v", surface.Summary)
	}
	if len(surface.Targets) != 0 {
		t.Fatalf("expected no recovery targets in empty scope, got %+v", surface.Targets)
	}
}

func TestClawHostRecoverySurfacePayloadSummarizesTargets(t *testing.T) {
	surface := clawHostRecoverySurfacePayload([]domain.Task{
		{
			ID:       "task-ready",
			Title:    "Ready recovery target",
			TenantID: "tenant-ready",
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_id":                    "claw-ready",
				"claw_name":                  "Alpha Ready",
				"clawhost_lifecycle_actions": "start,restart",
				"clawhost_takeover_triggers": "gateway timeout, websocket drop",
				"clawhost_recovery_evidence": "runbook, drill",
				"clawhost_pod_isolation":     "true",
				"clawhost_service_isolation": "true",
				"clawhost_takeover_required": "false",
			},
		},
		{
			ID:    "task-degraded",
			Title: "Needs recovery work",
			Source: "clawhost",
			Metadata: map[string]string{
				"tenant_id":                 "tenant-degraded",
				"claw_id":                   "claw-degraded",
				"claw_name":                 "Beta Degraded",
				"clawhost_lifecycle_actions": "",
				"clawhost_takeover_triggers": "",
				"clawhost_recovery_evidence": "",
				"pod_isolation":             "false",
				"service_isolation":         "false",
				"clawhost_takeover_required":"true",
			},
		},
		{
			ID:       "task-other",
			Title:    "Other control plane",
			Metadata: map[string]string{"control_plane": "other"},
		},
	}, "platform", "apollo")

	if surface.Status != "active" {
		t.Fatalf("expected recovery surface to become active, got %+v", surface)
	}
	if surface.Filters["team"] != "platform" || surface.Filters["project"] != "apollo" {
		t.Fatalf("expected recovery filters to preserve scope, got %+v", surface.Filters)
	}
	if surface.Summary.Targets != 2 || surface.Summary.RecoverableTargets != 1 || surface.Summary.DegradedTargets != 1 || surface.Summary.IsolatedTargets != 1 || surface.Summary.TakeoverRequired != 1 || surface.Summary.EvidenceArtifacts != 2 {
		t.Fatalf("unexpected recovery summary: %+v", surface.Summary)
	}
	if len(surface.Targets) != 2 {
		t.Fatalf("expected only clawhost recovery targets, got %+v", surface.Targets)
	}
	if surface.Targets[0].TaskID != "task-degraded" || surface.Targets[0].RecoveryStatus != "degraded" {
		t.Fatalf("expected degraded recovery target to sort first, got %+v", surface.Targets)
	}
	if surface.Targets[1].TaskID != "task-ready" || surface.Targets[1].RecoveryStatus != "ready" {
		t.Fatalf("expected ready recovery target to sort last, got %+v", surface.Targets)
	}
}

func TestClawHostRecoveryTargetFromTaskNormalizesWarnings(t *testing.T) {
	target, ok := clawHostRecoveryTargetFromTask(domain.Task{
		ID:       "task-1",
		Title:    "Fallback name",
		TenantID: "tenant-a",
		Metadata: map[string]string{
			"control_plane": "clawhost",
		},
	})
	if !ok {
		t.Fatal("expected clawhost recovery target to parse")
	}
	if target.TaskID != "task-1" || target.TenantID != "tenant-a" || target.ClawID != "task-1" || target.ClawName != "Fallback name" {
		t.Fatalf("expected fallback identifiers to normalize, got %+v", target)
	}
	if target.RecoveryStatus != "degraded" || target.TakeoverRequired {
		t.Fatalf("expected degraded recovery target defaults, got %+v", target)
	}
	if len(target.Warnings) != 4 {
		t.Fatalf("expected all degraded recovery warnings, got %+v", target.Warnings)
	}
}

func TestClawHostRecoveryTargetFromTaskRejectsNonClawHost(t *testing.T) {
	if _, ok := clawHostRecoveryTargetFromTask(domain.Task{Metadata: map[string]string{"control_plane": "other"}}); ok {
		t.Fatal("expected non-clawhost recovery task to be ignored")
	}
}
