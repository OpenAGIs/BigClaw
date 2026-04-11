package api

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestClawHostWorkflowSurfacePayloadHandlesDefaultScope(t *testing.T) {
	surface := clawHostWorkflowSurfacePayload([]domain.Task{
		{
			ID:       "other-task",
			Source:   "github",
			TenantID: "tenant-other",
			Metadata: map[string]string{
				"control_plane": "github",
				"channel_types": "slack",
			},
		},
	}, "", "", "")

	if surface.Integration != "clawhost" || surface.Status != "idle" {
		t.Fatalf("unexpected empty workflow payload identity: %+v", surface)
	}
	if surface.Filters["actor"] != "workflow-operator" || surface.Filters["team"] != "" || surface.Filters["project"] != "" {
		t.Fatalf("expected workflow payload default scope to stay intact, got %+v", surface.Filters)
	}
	if surface.Summary.WorkflowItems != 0 || surface.Summary.Tenants != 0 || surface.Summary.PairingApprovals != 0 || surface.Summary.CredentialReviews != 0 || surface.Summary.TakeoverRequired != 0 {
		t.Fatalf("expected zeroed workflow payload summary, got %+v", surface.Summary)
	}
	if len(surface.ReviewQueue) != 0 {
		t.Fatalf("expected no workflow payload review items, got %+v", surface.ReviewQueue)
	}
}

func TestClawHostWorkflowSurfacePayloadPreservesExplicitScope(t *testing.T) {
	surface := clawHostWorkflowSurfacePayload([]domain.Task{
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
				"channel_types":           "telegram",
				"whatsapp_pairing_status": "paired",
			},
		},
	}, "alice", "platform", "apollo")

	if surface.Status != "active" {
		t.Fatalf("expected workflow payload to become active, got %+v", surface)
	}
	if surface.Filters["actor"] != "alice" || surface.Filters["team"] != "platform" || surface.Filters["project"] != "apollo" {
		t.Fatalf("expected workflow payload to preserve explicit scope, got %+v", surface.Filters)
	}
	if surface.Summary.WorkflowItems != 2 || surface.Summary.Tenants != 2 || surface.Summary.PairingApprovals != 1 || surface.Summary.CredentialReviews != 1 || surface.Summary.TakeoverRequired != 1 || surface.Summary.ChannelMutations != 2 || surface.Summary.SkillMutations != 1 {
		t.Fatalf("unexpected workflow payload summary: %+v", surface.Summary)
	}
	if len(surface.ReviewQueue) != 2 {
		t.Fatalf("expected two workflow payload items, got %+v", surface.ReviewQueue)
	}
	if surface.ReviewQueue[0].TaskID != "claw-a" || !surface.ReviewQueue[0].TakeoverRequired {
		t.Fatalf("expected takeover-required workflow item first, got %+v", surface.ReviewQueue)
	}
}
