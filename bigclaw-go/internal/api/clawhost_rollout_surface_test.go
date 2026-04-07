package api

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestClawHostRolloutSurfacePayloadHandlesEmptyScope(t *testing.T) {
	surface := clawHostRolloutSurfacePayload(nil, "", "")

	if surface.Integration != "clawhost" || surface.Status != "idle" {
		t.Fatalf("unexpected empty rollout surface identity: %+v", surface)
	}
	if surface.Filters["team"] != "" || surface.Filters["project"] != "" {
		t.Fatalf("expected empty rollout filters to stay empty, got %+v", surface.Filters)
	}
	if len(surface.SupportedActions) != 5 {
		t.Fatalf("expected supported rollout actions to stay populated, got %+v", surface.SupportedActions)
	}
	if surface.Summary.ActivePlans != 0 || surface.Summary.TotalTargets != 0 || surface.Summary.CanaryTargets != 0 || surface.Summary.TakeoverRequired != 0 || surface.Summary.EvidenceArtifacts != 0 {
		t.Fatalf("expected zeroed rollout summary, got %+v", surface.Summary)
	}
	if len(surface.Plans) != 0 {
		t.Fatalf("expected no rollout plans in empty scope, got %+v", surface.Plans)
	}
}

func TestClawHostRolloutSurfacePayloadSummarizesPlans(t *testing.T) {
	surface := clawHostRolloutSurfacePayload([]domain.Task{
		{
			ID:       "claw-a",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":                      "clawhost",
				"inventory_kind":                     "claw",
				"claw_id":                            "claw-a",
				"claw_name":                          "sales-a",
				"provider_status":                    "running",
				"clawhost_rollout_action":            "restart",
				"clawhost_rollout_concurrency_limit": "2",
			},
		},
		{
			ID:       "claw-b",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"inventory_kind":          "claw",
				"claw_id":                 "claw-b",
				"claw_name":               "sales-b",
				"provider_status":         "running",
				"clawhost_rollout_action": "restart",
				"clawhost_takeover_required": "true",
			},
		},
		{
			ID:       "claw-c",
			Source:   "clawhost",
			TenantID: "tenant-b",
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"inventory_kind":            "claw",
				"claw_id":                   "claw-c",
				"claw_name":                 "ops-c",
				"provider_status":           "running",
				"clawhost_update_available": "true",
			},
		},
		{
			ID:       "other",
			Metadata: map[string]string{"control_plane": "other"},
		},
	}, "platform", "apollo")

	if surface.Status != "active" {
		t.Fatalf("expected rollout surface to become active, got %+v", surface)
	}
	if surface.Filters["team"] != "platform" || surface.Filters["project"] != "apollo" {
		t.Fatalf("expected rollout filters to preserve scope, got %+v", surface.Filters)
	}
	if surface.Summary.ActivePlans != 2 || surface.Summary.TotalTargets != 3 || surface.Summary.CanaryTargets != 2 || surface.Summary.TakeoverRequired != 1 || surface.Summary.EvidenceArtifacts != 10 {
		t.Fatalf("unexpected rollout summary: %+v", surface.Summary)
	}
	if len(surface.Plans) != 2 {
		t.Fatalf("expected two rollout plans, got %+v", surface.Plans)
	}
	if surface.Plans[0].TenantID != "tenant-a" || surface.Plans[0].Action != "restart" || surface.Plans[0].TargetCount != 2 {
		t.Fatalf("expected first rollout plan to cover tenant-a restart targets, got %+v", surface.Plans[0])
	}
	if surface.Plans[0].TakeoverHook != "required" {
		t.Fatalf("expected tenant-a rollout plan to require takeover, got %+v", surface.Plans[0])
	}
	if surface.Plans[1].TenantID != "tenant-b" || surface.Plans[1].Action != "upgrade" || surface.Plans[1].TargetCount != 1 {
		t.Fatalf("expected second rollout plan to cover tenant-b upgrade target, got %+v", surface.Plans[1])
	}
}
