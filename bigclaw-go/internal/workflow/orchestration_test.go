package workflow

import (
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestCrossDepartmentOrchestratorPlansHandoffs(t *testing.T) {
	orchestrator := CrossDepartmentOrchestrator{}
	task := domain.Task{
		ID:                 "BIG-500",
		Source:             "linear",
		Title:              "Rollout with security review",
		Description:        "Customer-facing rollout needs stakeholder approval and analytics checks.",
		Labels:             []string{"release", "security", "customer"},
		RequiredTools:      []string{"browser", "sql"},
		RiskLevel:          domain.RiskHigh,
		AcceptanceCriteria: []string{"customer signoff"},
		ValidationPlan:     []string{"approval evidence"},
	}

	plan := orchestrator.Plan(task)
	if plan.TaskID != "BIG-500" || plan.CollaborationMode != "cross-functional" {
		t.Fatalf("unexpected plan header: %+v", plan)
	}
	if got := plan.Departments(); !reflect.DeepEqual(got, []string{"operations", "engineering", "security", "data", "customer-success"}) {
		t.Fatalf("unexpected departments: %+v", got)
	}
	if got := plan.RequiredApprovals(); !reflect.DeepEqual(got, []string{"security-review"}) {
		t.Fatalf("unexpected approvals: %+v", got)
	}
}

func TestPremiumOrchestrationPolicyConstrainsStandardTier(t *testing.T) {
	policy := PremiumOrchestrationPolicy{}
	plan := OrchestrationPlan{
		TaskID:            "BIG-501",
		CollaborationMode: "cross-functional",
		Handoffs: []DepartmentHandoff{
			{Department: "operations", Reason: "ops"},
			{Department: "engineering", Reason: "eng"},
			{Department: "security", Reason: "security", Approvals: []string{"security-review"}},
		},
	}
	constrained, decision := policy.Apply(domain.Task{ID: "BIG-501"}, plan)
	if !decision.UpgradeRequired || decision.Tier != "standard" || decision.EntitlementStatus != "upgrade-required" {
		t.Fatalf("unexpected standard-tier decision: %+v", decision)
	}
	if constrained.CollaborationMode != "tier-limited" || !reflect.DeepEqual(constrained.Departments(), []string{"operations", "engineering"}) {
		t.Fatalf("unexpected constrained plan: %+v", constrained)
	}
	if !reflect.DeepEqual(decision.BlockedDepartments, []string{"security"}) || decision.OverageUsageUnits != 1 || decision.OverageCostUSD != 4.0 {
		t.Fatalf("unexpected blocked decision payload: %+v", decision)
	}
}

func TestPremiumOrchestrationPolicyKeepsPremiumPlan(t *testing.T) {
	policy := PremiumOrchestrationPolicy{}
	plan := OrchestrationPlan{
		TaskID:            "BIG-502",
		CollaborationMode: "cross-functional",
		Handoffs: []DepartmentHandoff{
			{Department: "operations", Reason: "ops"},
			{Department: "engineering", Reason: "eng"},
			{Department: "security", Reason: "security"},
		},
	}
	full, decision := policy.Apply(domain.Task{ID: "BIG-502", Labels: []string{"enterprise"}}, plan)
	if decision.UpgradeRequired || decision.Tier != "premium" || decision.BillingModel != "premium-included" {
		t.Fatalf("unexpected premium decision: %+v", decision)
	}
	if !reflect.DeepEqual(full.Departments(), []string{"operations", "engineering", "security"}) {
		t.Fatalf("unexpected premium plan: %+v", full)
	}
}

func TestBuildHandoffRequest(t *testing.T) {
	plan := OrchestrationPlan{
		TaskID:            "BIG-503",
		CollaborationMode: "cross-functional",
		Handoffs: []DepartmentHandoff{
			{Department: "operations", Reason: "ops"},
			{Department: "security", Reason: "security", Approvals: []string{"security-review"}},
		},
	}
	request := BuildHandoffRequest(false, plan, OrchestrationPolicyDecision{UpgradeRequired: true, Reason: "premium tier required"})
	if request == nil {
		t.Fatal("expected handoff request")
	}
	if request.TargetTeam != "security" || request.Status != "blocked" || request.Reason != "premium tier required" {
		t.Fatalf("unexpected handoff request: %+v", request)
	}
	if !reflect.DeepEqual(request.RequiredApprovals, []string{"security-review"}) {
		t.Fatalf("unexpected required approvals: %+v", request.RequiredApprovals)
	}
	if got := BuildHandoffRequest(true, plan, OrchestrationPolicyDecision{}); got != nil {
		t.Fatalf("expected no request when execution accepted, got %+v", got)
	}
}
