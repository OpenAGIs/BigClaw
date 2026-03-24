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

func TestRenderClawHostRolloutPlannerReport(t *testing.T) {
	plan := BuildDefaultClawHostRolloutPlanner(nil, "platform", "apollo")
	audit := AuditClawHostRolloutPlanner(plan)
	report := RenderClawHostRolloutPlannerReport(plan, audit)

	for _, want := range []string{
		"# ClawHost Rollout Planner",
		"Plan ID: BIG-PAR-288",
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

func TestClawHostRolloutHelpers(t *testing.T) {
	if got := clawHostFirstNonEmpty("   ", "", " app-1 ", "app-2"); got != "app-1" {
		t.Fatalf("clawHostFirstNonEmpty = %q, want %q", got, "app-1")
	}
	if got := clawHostFirstNonEmpty(" ", ""); got != "" {
		t.Fatalf("clawHostFirstNonEmpty empty fallback = %q, want empty", got)
	}
	if got := clawHostMin(2, 5); got != 2 {
		t.Fatalf("clawHostMin(2, 5) = %d, want 2", got)
	}
	if got := clawHostMin(7, 3); got != 3 {
		t.Fatalf("clawHostMin(7, 3) = %d, want 3", got)
	}
	if got := clawHostMax(2, 5); got != 5 {
		t.Fatalf("clawHostMax(2, 5) = %d, want 5", got)
	}
	if got := clawHostMax(7, 3); got != 7 {
		t.Fatalf("clawHostMax(7, 3) = %d, want 7", got)
	}
}

func TestAuditClawHostRolloutPlannerEmptyPlanAndSortedValues(t *testing.T) {
	if got := clawHostSortedValues([]domain.Task{
		{TenantID: " tenant-b "},
		{TenantID: "tenant-a"},
		{TenantID: "tenant-b"},
	}, func(task domain.Task) string {
		return task.TenantID
	}); strings.Join(got, ",") != "tenant-a,tenant-b" {
		t.Fatalf("unexpected sorted rollout values: %+v", got)
	}

	audit := AuditClawHostRolloutPlanner(ClawHostRolloutPlanner{PlanID: "plan-empty", Version: "v1"})
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected empty rollout plan readiness score to stay at zero, got %+v", audit)
	}
	if audit.ReleaseReady {
		t.Fatalf("expected empty rollout plan to remain not release-ready, got %+v", audit)
	}
}
