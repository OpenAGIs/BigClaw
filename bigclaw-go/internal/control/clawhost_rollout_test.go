package control

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBuildClawHostRolloutPlansCreatesCanaryAndBatchWaves(t *testing.T) {
	tasks := []domain.Task{
		{
			ID:       "claw-a",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":                      "clawhost",
				"inventory_kind":                     "claw",
				"claw_id":                            "claw-a",
				"claw_name":                          "sales-a",
				"provider":                           "hetzner",
				"provider_status":                    "running",
				"domain":                             "sales-a.clawhost.cloud",
				"agent_count":                        "2",
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
				"provider":                "hetzner",
				"provider_status":         "running",
				"domain":                  "sales-b.clawhost.cloud",
				"agent_count":             "1",
				"clawhost_rollout_action": "restart",
			},
		},
		{
			ID:       "claw-c",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":              "clawhost",
				"inventory_kind":             "claw",
				"claw_id":                    "claw-c",
				"claw_name":                  "sales-c",
				"provider":                   "hetzner",
				"provider_status":            "running",
				"domain":                     "sales-c.clawhost.cloud",
				"agent_count":                "1",
				"clawhost_rollout_action":    "restart",
				"clawhost_takeover_required": "true",
			},
		},
	}
	plans := BuildClawHostRolloutPlans(tasks)
	if len(plans) != 1 {
		t.Fatalf("expected one rollout plan, got %+v", plans)
	}
	plan := plans[0]
	if plan.Action != "restart" || plan.TargetCount != 3 || plan.WaveCount != 2 || plan.CanaryCount != 1 || plan.ConcurrencyLimit != 2 {
		t.Fatalf("unexpected rollout plan: %+v", plan)
	}
	if plan.TakeoverHook != "required" {
		t.Fatalf("expected takeover hook required, got %+v", plan)
	}
	if len(plan.Waves[0].Targets) != 1 || plan.Waves[0].Kind != "canary" {
		t.Fatalf("expected first wave to be canary, got %+v", plan.Waves)
	}
	if len(plan.Waves[1].Targets) != 2 || plan.Waves[1].Kind != "batch" {
		t.Fatalf("expected second wave to batch remaining claws, got %+v", plan.Waves)
	}
}

func TestBuildClawHostRolloutPlansChoosesStartAndUpgradeDefaults(t *testing.T) {
	tasks := []domain.Task{
		{
			ID:       "stopped-claw",
			Source:   "clawhost",
			TenantID: "tenant-a",
			Metadata: map[string]string{
				"control_plane":   "clawhost",
				"inventory_kind":  "claw",
				"claw_id":         "stopped-claw",
				"claw_name":       "support-a",
				"provider_status": "stopped",
			},
		},
		{
			ID:       "upgrade-claw",
			Source:   "clawhost",
			TenantID: "tenant-b",
			Metadata: map[string]string{
				"control_plane":             "clawhost",
				"inventory_kind":            "claw",
				"claw_id":                   "upgrade-claw",
				"claw_name":                 "ops-a",
				"provider_status":           "running",
				"clawhost_update_available": "true",
			},
		},
	}
	plans := BuildClawHostRolloutPlans(tasks)
	if len(plans) != 2 {
		t.Fatalf("expected two rollout plans, got %+v", plans)
	}
	if plans[0].Action != "start" || plans[1].Action != "upgrade" {
		t.Fatalf("expected start and upgrade plans, got %+v", plans)
	}
	if plans[1].ConcurrencyLimit != 1 {
		t.Fatalf("expected upgrade concurrency limit 1, got %+v", plans[1])
	}
}
