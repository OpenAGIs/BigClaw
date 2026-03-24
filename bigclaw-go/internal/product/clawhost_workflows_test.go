package product

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBuildClawHostWorkflowSurfaceIncludesParallelLanesAndSignals(t *testing.T) {
	tasks := []domain.Task{
		{
			ID:        "task-1",
			State:     domain.TaskBlocked,
			RiskLevel: domain.RiskHigh,
			Metadata: map[string]string{
				"channel":  "telegram",
				"device":   "wechat",
				"provider": "openai",
			},
			CreatedAt: time.Now(),
		},
		{
			ID: "task-2",
			Metadata: map[string]string{
				"channel":  "slack",
				"provider": "anthropic",
			},
			CreatedAt: time.Now(),
		},
	}

	surface := BuildDefaultClawHostWorkflowSurface(tasks, "alice", "platform", "apollo")
	if surface.Name != "clawhost-workflow-surface" || surface.Version != "go-v1" {
		t.Fatalf("unexpected workflow surface identity: %+v", surface)
	}
	if surface.SourceRepo != "https://github.com/fastclaw-ai/clawhost" {
		t.Fatalf("unexpected source repo: %s", surface.SourceRepo)
	}
	if surface.Filters["actor"] != "alice" || surface.Filters["team"] != "platform" || surface.Filters["project"] != "apollo" {
		t.Fatalf("unexpected filters: %+v", surface.Filters)
	}
	if len(surface.Lanes) != 5 {
		t.Fatalf("expected 5 lanes, got %d", len(surface.Lanes))
	}
	if got := surface.OperationalSignals["total_tasks"]; got != 2 {
		t.Fatalf("expected total_tasks=2, got %d", got)
	}
	if got := surface.OperationalSignals["blocked_tasks"]; got != 1 {
		t.Fatalf("expected blocked_tasks=1, got %d", got)
	}
	if got := surface.OperationalSignals["high_risk_tasks"]; got != 1 {
		t.Fatalf("expected high_risk_tasks=1, got %d", got)
	}

	tokenSessionLanes := 0
	approvalLanes := 0
	for _, lane := range surface.Lanes {
		if lane.TokenSessionGating {
			tokenSessionLanes++
		}
		if len(lane.ParallelBatch.RequiredApprovals) > 0 {
			approvalLanes++
		}
		if lane.ParallelBatch.MaxConcurrency <= 0 || lane.ParallelBatch.CanarySize <= 0 {
			t.Fatalf("expected positive batch controls for lane %s: %+v", lane.LaneID, lane.ParallelBatch)
		}
	}
	if tokenSessionLanes == 0 {
		t.Fatalf("expected at least one token/session-gated lane, got %+v", surface.Lanes)
	}
	if approvalLanes != len(surface.Lanes) {
		t.Fatalf("expected all lanes to include approvals, got %d/%d", approvalLanes, len(surface.Lanes))
	}
}

func TestAuditClawHostWorkflowSurfaceFlagsWorkflowGaps(t *testing.T) {
	surface := BuildDefaultClawHostWorkflowSurface(nil, "", "", "")
	if len(surface.Lanes) == 0 {
		t.Fatal("expected base lanes from builder")
	}

	surface.Lanes[0].Route = ""
	surface.Lanes[0].Owner = ""
	surface.Lanes[0].TokenSessionGating = false
	surface.Lanes[0].SupportsHumanTakeover = false
	surface.Lanes[0].ParallelBatch.RequiredApprovals = nil
	surface.Lanes[0].Stage = "unknown"
	surface.Lanes[0].AutomationBoundary = "manual-only"

	audit := AuditClawHostWorkflowSurface(surface)
	if audit.LaneCount != len(surface.Lanes) {
		t.Fatalf("unexpected lane count in audit: %+v", audit)
	}
	if len(audit.MissingRouteLanes) != 1 || audit.MissingRouteLanes[0] != surface.Lanes[0].LaneID {
		t.Fatalf("expected missing route lane for %s, got %+v", surface.Lanes[0].LaneID, audit.MissingRouteLanes)
	}
	if len(audit.MissingOwnerLanes) != 1 || audit.MissingOwnerLanes[0] != surface.Lanes[0].LaneID {
		t.Fatalf("expected missing owner lane for %s, got %+v", surface.Lanes[0].LaneID, audit.MissingOwnerLanes)
	}
	if len(audit.LanesWithoutTokenSessionGating) == 0 || len(audit.LanesWithoutTakeover) == 0 {
		t.Fatalf("expected token/takeover gaps, got %+v", audit)
	}
	if len(audit.LanesWithoutRequiredApprovals) == 0 {
		t.Fatalf("expected required-approvals gap, got %+v", audit)
	}
	if len(audit.LanesWithInvalidStage) == 0 || len(audit.LanesWithInvalidAutomationPolicy) == 0 {
		t.Fatalf("expected invalid stage/policy gaps, got %+v", audit)
	}
	if audit.ReadinessScore >= 100 {
		t.Fatalf("expected reduced readiness score, got %.1f", audit.ReadinessScore)
	}
}

func TestRenderClawHostWorkflowReportIncludesKeySections(t *testing.T) {
	surface := BuildDefaultClawHostWorkflowSurface(nil, "ops-bot", "platform", "apollo")
	audit := AuditClawHostWorkflowSurface(surface)

	report := RenderClawHostWorkflowReport(surface, audit)
	for _, want := range []string{
		"# ClawHost Workflow Surface",
		"## Summary",
		"## Lanes",
		"## Gaps",
		"clawhost-parallel-rollout-control",
		"token_session",
		"Readiness Score",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
		}
	}
	if !strings.HasSuffix(report, "\n") {
		t.Fatalf("expected markdown report to end with newline, got %q", report)
	}
}

func TestBuildClawHostWorkflowSurfaceEncodesScopedLaneRoutes(t *testing.T) {
	surface := BuildDefaultClawHostWorkflowSurface(nil, "alice", "platform & ops", "apollo/mobile")
	if len(surface.Lanes) == 0 {
		t.Fatal("expected workflow lanes from builder")
	}
	for _, lane := range surface.Lanes {
		parsed, err := url.Parse(lane.Route)
		if err != nil {
			t.Fatalf("parse lane route %s: %v", lane.LaneID, err)
		}
		if parsed.Query().Get("team") != "platform & ops" || parsed.Query().Get("project") != "apollo/mobile" {
			t.Fatalf("expected encoded scope filters in lane %s route, got %s", lane.LaneID, lane.Route)
		}
		if strings.Contains(lane.Route, "team=platform & ops") || strings.Contains(lane.Route, "project=apollo/mobile") {
			t.Fatalf("expected lane %s route to encode reserved characters, got %s", lane.LaneID, lane.Route)
		}
	}
}
