package product

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBuildClawHostWorkflowSurface(t *testing.T) {
	surface := BuildClawHostWorkflowSurface([]domain.Task{
		{
			ID:       "claw-a",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"claw_id":                   "claw-a",
				"claw_name":                 "sales-a",
				"skill_count":               "3",
				"agent_skill_count":         "4",
				"channel_types":             "telegram,discord,whatsapp",
				"whatsapp_pairing_status":   "waiting",
				"admin_credentials_exposed": "true",
				"admin_surface_path":        "/credentials",
			},
		},
		{
			ID:       "claw-b",
			Source:   "clawhost",
			TenantID: "tenant-b",
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"claw_id":                 "claw-b",
				"claw_name":               "sales-b",
				"skill_count":             "2",
				"agent_skill_count":       "2",
				"channel_types":           "telegram",
				"whatsapp_pairing_status": "paired",
			},
		},
	})
	if surface.Status != "active" || surface.Summary.WorkflowItems != 2 || surface.Summary.Tenants != 2 {
		t.Fatalf("unexpected workflow surface summary: %+v", surface)
	}
	if surface.Summary.PairingApprovals != 1 || surface.Summary.CredentialReviews != 1 || surface.Summary.TakeoverRequired != 1 {
		t.Fatalf("unexpected workflow approval counts: %+v", surface.Summary)
	}
	if len(surface.ReviewQueue) != 2 || !surface.ReviewQueue[0].TakeoverRequired {
		t.Fatalf("expected takeover-required item first, got %+v", surface.ReviewQueue)
	}
}

func TestBuildClawHostWorkflowSurfaceIgnoresNonClawHostTasksAndUsesFallbackReason(t *testing.T) {
	surface := BuildClawHostWorkflowSurface([]domain.Task{
		{
			ID:       "claw-fallback",
			Source:   "clawhost",
			TenantID: "tenant-fallback",
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_name":                  "fallback-bot",
				"whatsapp_pairing_status":    "paired",
				"clawhost_takeover_required": "true",
			},
		},
		{
			ID:       "other-task",
			Source:   "github",
			TenantID: "tenant-other",
			Metadata: map[string]string{
				"control_plane": "github",
				"channel_types": "slack",
			},
		},
	})

	if surface.Status != "active" || surface.Summary.WorkflowItems != 1 || surface.Summary.Tenants != 1 {
		t.Fatalf("expected only clawhost workflow item to count, got %+v", surface)
	}
	if surface.Summary.PairingApprovals != 0 || surface.Summary.CredentialReviews != 0 || surface.Summary.TakeoverRequired != 1 {
		t.Fatalf("unexpected workflow summary counts: %+v", surface.Summary)
	}
	if len(surface.ReviewQueue) != 1 {
		t.Fatalf("expected non-clawhost task to be filtered out, got %+v", surface.ReviewQueue)
	}
	if surface.ReviewQueue[0].TaskID != "claw-fallback" || surface.ReviewQueue[0].ReviewReason != "workflow requires explicit human takeover before mutating bot config" {
		t.Fatalf("expected fallback clawhost item and reason, got %+v", surface.ReviewQueue[0])
	}
}
