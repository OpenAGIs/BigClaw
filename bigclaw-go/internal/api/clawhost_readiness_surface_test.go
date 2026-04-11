package api

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestClawHostReadinessSurfacePayloadHandlesEmptyScope(t *testing.T) {
	surface := clawHostReadinessSurfacePayload(nil, "", "")

	if surface.Integration != "clawhost" || surface.Status != "idle" {
		t.Fatalf("unexpected empty readiness surface identity: %+v", surface)
	}
	if surface.Filters["team"] != "" || surface.Filters["project"] != "" {
		t.Fatalf("expected empty readiness filters to stay empty, got %+v", surface.Filters)
	}
	if len(surface.SupportedChecks) != 5 {
		t.Fatalf("expected supported checks to stay populated, got %+v", surface.SupportedChecks)
	}
	if surface.Summary.Targets != 0 || surface.Summary.ReadyTargets != 0 || surface.Summary.DegradedTargets != 0 || surface.Summary.AdminReadyTargets != 0 || surface.Summary.WebSocketReadyTargets != 0 || surface.Summary.SubdomainReadyTargets != 0 || surface.Summary.UpgradeAvailableTargets != 0 {
		t.Fatalf("expected zeroed readiness summary, got %+v", surface.Summary)
	}
	if len(surface.Targets) != 0 {
		t.Fatalf("expected no readiness targets in empty scope, got %+v", surface.Targets)
	}
}

func TestClawHostReadinessSurfacePayloadSummarizesTargets(t *testing.T) {
	surface := clawHostReadinessSurfacePayload([]domain.Task{
		{
			ID:       "task-ready",
			Title:    "Ready bot",
			TenantID: "tenant-ready",
			Metadata: map[string]string{
				"control_plane":        "clawhost",
				"claw_id":              "claw-ready",
				"claw_name":            "Alpha Ready",
				"domain":               "ready.clawhost.local",
				"proxy_mode":           "subdomain",
				"gateway_port":         "443",
				"version_current":      "1.0.0",
				"version_latest":       "1.0.0",
				"version_status":       "current",
				"admin_ui_enabled":     "true",
				"websocket_reachable":  "true",
				"subdomain_ready":      "true",
				"reachable":            "true",
			},
		},
		{
			ID:    "task-degraded",
			Title: "Needs upgrade",
			Source: "clawhost",
			Metadata: map[string]string{
				"tenant_id":            "tenant-degraded",
				"claw_id":              "claw-degraded",
				"claw_name":            "Beta Degraded",
				"gateway_port":         "8080",
				"version_current":      "1.0.0",
				"version_latest":       "1.1.0",
				"version_status":       "upgrade_available",
				"admin_ui_enabled":     "false",
				"websocket_reachable":  "false",
				"subdomain_ready":      "false",
				"reachable":            "false",
			},
		},
		{
			ID:       "task-other",
			Title:    "Other control plane",
			Metadata: map[string]string{"control_plane": "other"},
		},
	}, "platform", "apollo")

	if surface.Status != "active" {
		t.Fatalf("expected readiness surface to become active, got %+v", surface)
	}
	if surface.Filters["team"] != "platform" || surface.Filters["project"] != "apollo" {
		t.Fatalf("expected readiness filters to preserve scope, got %+v", surface.Filters)
	}
	if surface.Summary.Targets != 2 || surface.Summary.ReadyTargets != 1 || surface.Summary.DegradedTargets != 1 || surface.Summary.AdminReadyTargets != 1 || surface.Summary.WebSocketReadyTargets != 1 || surface.Summary.SubdomainReadyTargets != 1 || surface.Summary.UpgradeAvailableTargets != 1 {
		t.Fatalf("unexpected readiness summary: %+v", surface.Summary)
	}
	if len(surface.Targets) != 2 {
		t.Fatalf("expected only clawhost readiness targets, got %+v", surface.Targets)
	}
	if surface.Targets[0].TaskID != "task-degraded" || surface.Targets[0].ReviewStatus != "degraded" {
		t.Fatalf("expected degraded target to sort first, got %+v", surface.Targets)
	}
	if surface.Targets[1].TaskID != "task-ready" || surface.Targets[1].ReviewStatus != "ready" {
		t.Fatalf("expected ready target to sort last, got %+v", surface.Targets)
	}
}

func TestClawHostReadinessTargetFromTaskNormalizesWarnings(t *testing.T) {
	target, ok := clawHostReadinessTargetFromTask(domain.Task{
		ID:       "task-1",
		Title:    "Fallback name",
		TenantID: "tenant-a",
		Metadata: map[string]string{
			"control_plane":       "clawhost",
			"gateway_port":        "9000",
			"version_status":      "upgrade_available",
			"version_current":     "1.0.0",
			"version_latest":      "1.2.0",
			"admin_ui_enabled":    "false",
			"websocket_reachable": "false",
			"subdomain_ready":     "false",
			"reachable":           "false",
		},
	})
	if !ok {
		t.Fatal("expected clawhost readiness target to parse")
	}
	if target.TaskID != "task-1" || target.TenantID != "tenant-a" || target.ClawID != "task-1" || target.ClawName != "Fallback name" {
		t.Fatalf("expected fallback identifiers to normalize, got %+v", target)
	}
	if target.GatewayPort != 9000 || target.ReviewStatus != "degraded" || target.VersionStatus != "upgrade_available" {
		t.Fatalf("expected degraded readiness target details, got %+v", target)
	}
	if len(target.Warnings) != 4 {
		t.Fatalf("expected all degraded warnings, got %+v", target.Warnings)
	}
}

func TestClawHostReadinessTargetFromTaskRejectsNonClawHost(t *testing.T) {
	if _, ok := clawHostReadinessTargetFromTask(domain.Task{Metadata: map[string]string{"control_plane": "other"}}); ok {
		t.Fatal("expected non-clawhost task to be ignored")
	}
}
