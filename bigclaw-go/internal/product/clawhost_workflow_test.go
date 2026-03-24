package product

import (
	"slices"
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

func TestBuildClawHostWorkflowSurfaceOrdersReviewQueue(t *testing.T) {
	surface := BuildClawHostWorkflowSurface([]domain.Task{
		{
			ID:       "paired-zeta",
			Source:   "clawhost",
			TenantID: "tenant-z",
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"claw_name":               "zeta",
				"whatsapp_pairing_status": "paired",
			},
		},
		{
			ID:       "takeover-beta",
			Source:   "clawhost",
			TenantID: "tenant-b",
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"claw_name":                  "beta",
				"whatsapp_pairing_status":    "paired",
				"clawhost_takeover_required": "true",
			},
		},
		{
			ID:       "paired-alpha",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"claw_name":               "alpha",
				"whatsapp_pairing_status": "paired",
			},
		},
		{
			ID:       "waiting-gamma",
			Source:   "clawhost",
			TenantID: "tenant-g",
			Metadata: map[string]string{
				"control_plane":           "clawhost",
				"claw_name":               "gamma",
				"whatsapp_pairing_status": "waiting",
			},
		},
	})

	got := make([]string, 0, len(surface.ReviewQueue))
	for _, item := range surface.ReviewQueue {
		got = append(got, item.TaskID)
	}
	want := []string{"takeover-beta", "waiting-gamma", "paired-alpha", "paired-zeta"}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected workflow review queue ordering: got=%v want=%v queue=%+v", got, want, surface.ReviewQueue)
	}
}

func TestBuildClawHostWorkflowSurfaceIdleDefaults(t *testing.T) {
	surface := BuildClawHostWorkflowSurface([]domain.Task{
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

	if surface.Status != "idle" || surface.Summary.WorkflowItems != 0 || surface.Summary.Tenants != 0 {
		t.Fatalf("expected idle workflow surface, got %+v", surface)
	}
	if len(surface.ReviewQueue) != 0 {
		t.Fatalf("expected empty review queue for idle surface, got %+v", surface.ReviewQueue)
	}
	wantChannels := []string{"whatsapp", "telegram", "discord", "slack", "signal"}
	if !slices.Equal(surface.SupportedChannels, wantChannels) {
		t.Fatalf("unexpected supported channels: got=%v want=%v", surface.SupportedChannels, wantChannels)
	}
}

func TestBuildClawHostWorkflowSurfaceNormalizesParsingDefaults(t *testing.T) {
	surface := BuildClawHostWorkflowSurface([]domain.Task{
		{
			ID:       "claw-parsing",
			Source:   "clawhost",
			TenantID: "",
			Title:    "parsing-bot",
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"owner_user_id":             "owner-42",
				"channel_types":             " telegram, ,discord,telegram ",
				"whatsapp_pairing_status":   "paired",
				"admin_credentials_exposed": "not-a-bool",
				"clawhost_takeover_required": "not-a-bool",
				"skill_count":               "NaN",
				"agent_skill_count":         "oops",
			},
		},
	})

	if surface.Status != "active" || surface.Summary.WorkflowItems != 1 || surface.Summary.Tenants != 1 {
		t.Fatalf("expected normalized workflow surface to stay active with one tenant, got %+v", surface)
	}
	if surface.Summary.ChannelMutations != 1 || surface.Summary.SkillMutations != 0 || surface.Summary.CredentialReviews != 0 || surface.Summary.TakeoverRequired != 0 {
		t.Fatalf("unexpected normalized workflow summary counts: %+v", surface.Summary)
	}
	if len(surface.ReviewQueue) != 1 {
		t.Fatalf("expected one normalized workflow item, got %+v", surface.ReviewQueue)
	}

	item := surface.ReviewQueue[0]
	if item.TenantID != "owner-42" {
		t.Fatalf("expected tenant fallback from owner_user_id, got %+v", item)
	}
	if item.ClawID != "claw-parsing" || item.ClawName != "parsing-bot" {
		t.Fatalf("expected task id/title fallbacks, got %+v", item)
	}
	if !slices.Equal(item.Channels, []string{"discord", "telegram", "telegram"}) {
		t.Fatalf("expected normalized sorted channels, got %+v", item.Channels)
	}
	if item.SkillsEnabled != 0 || item.AgentSkillCount != 0 {
		t.Fatalf("expected invalid integer metadata to fall back to zero, got %+v", item)
	}
	if item.CredentialsExposed || item.TakeoverRequired {
		t.Fatalf("expected invalid bool metadata to stay false, got %+v", item)
	}
	if item.ReviewReason != "channel config requires review across active IM integrations" {
		t.Fatalf("expected channel-only review reason, got %+v", item)
	}
}
