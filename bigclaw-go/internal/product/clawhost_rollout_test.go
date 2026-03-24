package product

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBuildDefaultClawHostRolloutPlannerUsesTaskTenantsAndApps(t *testing.T) {
	plan := BuildDefaultClawHostRolloutPlanner([]domain.Task{
		{ID: "task-1", TenantID: "tenant-a", Metadata: map[string]string{"app": "alpha-app"}},
		{ID: "task-2", TenantID: "tenant-b", Metadata: map[string]string{"app": "beta-app"}},
		{ID: "task-3", TenantID: "tenant-a", Metadata: map[string]string{"app": "alpha-app"}},
	}, "platform", "apollo")

	if plan.PlanID != "BIG-PAR-288" || plan.Version != "go-v1" {
		t.Fatalf("unexpected plan metadata: %+v", plan)
	}
	if plan.Summary.WaveCount != 3 || plan.Summary.TenantCount != 2 || plan.Summary.AppCount != 2 || plan.Summary.RequiresApprovalWaves != 2 || plan.Summary.CanaryWaves != 1 || plan.Summary.MaxParallelBots < 5 {
		t.Fatalf("unexpected rollout summary: %+v", plan.Summary)
	}
	if plan.Filters["team"] != "platform" || plan.Filters["project"] != "apollo" {
		t.Fatalf("unexpected filters: %+v", plan.Filters)
	}
	if len(plan.Waves) != 3 || plan.Waves[0].WaveID == "" || len(plan.Waves[0].HealthChecks) == 0 || len(plan.Waves[1].TakeoverTriggers) == 0 {
		t.Fatalf("expected populated waves, got %+v", plan.Waves)
	}
}

func TestBuildDefaultClawHostRolloutPlannerFallsBackToProjectAndDefaults(t *testing.T) {
	t.Run("project fallback", func(t *testing.T) {
		plan := BuildDefaultClawHostRolloutPlanner(nil, "platform", "apollo")
		if plan.Filters["team"] != "platform" || plan.Filters["project"] != "apollo" {
			t.Fatalf("unexpected filters: %+v", plan.Filters)
		}
		if plan.Summary.TenantCount != 3 || plan.Summary.AppCount != 1 {
			t.Fatalf("unexpected fallback rollout summary: %+v", plan.Summary)
		}
		if len(plan.Waves) != 3 || len(plan.Waves[0].TargetApps) != 1 || plan.Waves[0].TargetApps[0] != "apollo" {
			t.Fatalf("expected project fallback app in rollout waves, got %+v", plan.Waves)
		}
		if len(plan.Waves[2].TargetApps) != 1 || plan.Waves[2].TargetApps[0] != "apollo" {
			t.Fatalf("expected app fanout to use project fallback app, got %+v", plan.Waves[2])
		}
	})

	t.Run("empty project fallback", func(t *testing.T) {
		plan := BuildDefaultClawHostRolloutPlanner(nil, "", "")
		if plan.Filters["team"] != "" || plan.Filters["project"] != "" {
			t.Fatalf("expected empty rollout filters to stay empty, got %+v", plan.Filters)
		}
		if len(plan.Waves) != 3 || len(plan.Waves[0].TargetApps) != 1 || plan.Waves[0].TargetApps[0] != "clawhost-app" {
			t.Fatalf("expected default app fallback in rollout waves, got %+v", plan.Waves)
		}
		if len(plan.Waves[2].TargetApps) != 1 || plan.Waves[2].TargetApps[0] != "clawhost-app" {
			t.Fatalf("expected default app fallback in app fanout wave, got %+v", plan.Waves[2])
		}
	})
}

func TestAuditClawHostRolloutPlannerDetectsGaps(t *testing.T) {
	plan := BuildDefaultClawHostRolloutPlanner(nil, "", "")
	plan.Waves[0].WaveID = plan.Waves[1].WaveID
	plan.Waves[0].HealthChecks = nil
	plan.Waves[1].RollbackActions = nil
	plan.Waves[2].MaxParallelBots = 0
	plan.Waves[2].TakeoverTriggers = nil

	audit := AuditClawHostRolloutPlanner(plan)
	if audit.ReleaseReady {
		t.Fatalf("expected rollout planner to fail readiness, got %+v", audit)
	}
	if len(audit.DuplicateWaveIDs) != 1 || len(audit.WavesMissingChecks) != 1 || len(audit.WavesMissingRollback) != 1 || len(audit.InvalidParallelism) != 1 || len(audit.MissingTakeoverSignals) != 1 {
		t.Fatalf("expected gap detection, got %+v", audit)
	}
	if audit.ReadinessScore >= 100 {
		t.Fatalf("expected readiness penalty, got %+v", audit)
	}
}

func TestAuditClawHostRolloutPlannerHandlesEmptyPlan(t *testing.T) {
	plan := BuildDefaultClawHostRolloutPlanner(nil, "", "")
	plan.Waves = nil
	plan.Summary = ClawHostRolloutSummary{}

	audit := AuditClawHostRolloutPlanner(plan)
	if audit.PlanID != plan.PlanID || audit.Version != plan.Version {
		t.Fatalf("expected audit metadata to mirror empty plan, got %+v", audit)
	}
	if plan.Filters["team"] != "" || plan.Filters["project"] != "" {
		t.Fatalf("expected empty rollout plan filters to stay empty, got %+v", plan.Filters)
	}
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected empty rollout plan readiness score to be zero, got %+v", audit)
	}
	if audit.ReleaseReady {
		t.Fatalf("expected empty rollout plan to stay not release ready, got %+v", audit)
	}
	if len(audit.DuplicateWaveIDs) != 0 || len(audit.WavesMissingChecks) != 0 || len(audit.WavesMissingRollback) != 0 || len(audit.InvalidParallelism) != 0 || len(audit.MissingTakeoverSignals) != 0 {
		t.Fatalf("expected empty rollout plan to have no per-wave gaps, got %+v", audit)
	}
}

func TestRenderClawHostRolloutPlannerReport(t *testing.T) {
	plan := BuildDefaultClawHostRolloutPlanner(nil, "platform", "apollo")
	audit := AuditClawHostRolloutPlanner(plan)
	report := RenderClawHostRolloutPlannerReport(plan, audit)

	for _, want := range []string{
		"# ClawHost Rollout Planner",
		"Plan ID: BIG-PAR-288",
		"## Filters",
		"- project: apollo",
		"- team: platform",
		"Canary Upgrade Wave",
		"Tenant Ring 1",
		"GET /proxy/:bot_id/",
		"Duplicate wave IDs: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestRenderClawHostRolloutPlannerReportHandlesFallbackPlanner(t *testing.T) {
	plan := BuildDefaultClawHostRolloutPlanner(nil, "", "")
	audit := AuditClawHostRolloutPlanner(plan)
	report := RenderClawHostRolloutPlannerReport(plan, audit)

	for _, want := range []string{
		"# ClawHost Rollout Planner",
		"## Filters",
		"- project: none",
		"- team: none",
		"Tenants: 3",
		"Apps: 1",
		"Canary Upgrade Wave",
		"apps=clawhost-app",
		"Missing takeover signals: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in fallback rollout report, got %s", want, report)
		}
	}
}
