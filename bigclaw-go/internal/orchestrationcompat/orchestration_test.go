package orchestrationcompat

import (
	"strings"
	"testing"
)

func TestCrossDepartmentOrchestratorRoutesSecurityDataAndCustomerWork(t *testing.T) {
	task := Task{
		TaskID:             "OPE-66",
		Source:             "linear",
		Title:              "Coordinate customer analytics rollout approval",
		Description:        "Need stakeholder sign-off for warehouse-backed browser workflow",
		Labels:             []string{"data", "customer", "premium"},
		Priority:           0,
		RiskLevel:          RiskHigh,
		RequiredTools:      []string{"browser", "sql"},
		AcceptanceCriteria: []string{"approval recorded"},
		ValidationPlan:     []string{"customer signoff"},
	}

	plan := CrossDepartmentOrchestrator{}.Plan(task)
	if plan.CollaborationMode != "cross-functional" {
		t.Fatalf("unexpected collaboration mode: %+v", plan)
	}
	wantDepartments := []string{"operations", "engineering", "security", "data", "customer-success"}
	for i, want := range wantDepartments {
		if plan.Departments()[i] != want {
			t.Fatalf("unexpected departments: %+v", plan.Departments())
		}
	}
	if approvals := plan.RequiredApprovals(); len(approvals) != 1 || approvals[0] != "security-review" {
		t.Fatalf("unexpected approvals: %+v", approvals)
	}
}

func TestStandardPolicyLimitsAdvancedCrossDepartmentRouting(t *testing.T) {
	task := Task{
		TaskID:        "OPE-66-standard",
		Source:        "linear",
		Title:         "Coordinate customer analytics rollout approval",
		Description:   "Need stakeholder sign-off for warehouse-backed browser workflow",
		Labels:        []string{"data", "customer"},
		RequiredTools: []string{"browser", "sql"},
		RiskLevel:     RiskHigh,
	}
	rawPlan := CrossDepartmentOrchestrator{}.Plan(task)
	plan, policy := PremiumOrchestrationPolicy{}.Apply(task, rawPlan)

	if plan.CollaborationMode != "tier-limited" {
		t.Fatalf("unexpected constrained plan: %+v", plan)
	}
	if got := strings.Join(plan.Departments(), ", "); got != "operations, engineering" {
		t.Fatalf("unexpected constrained departments: %s", got)
	}
	if !policy.UpgradeRequired || policy.EntitlementStatus != "upgrade-required" || policy.BillingModel != "standard-blocked" || policy.IncludedUsageUnits != 2 || policy.OverageUsageUnits != 3 || policy.OverageCostUSD != 12.0 || policy.EstimatedCostUSD != 15.0 {
		t.Fatalf("unexpected policy: %+v", policy)
	}
	if got := strings.Join(policy.BlockedDepartments, ", "); got != "security, data, customer-success" {
		t.Fatalf("unexpected blocked departments: %s", got)
	}
}

func TestRenderOrchestrationPlanListsHandoffsAndPolicy(t *testing.T) {
	task := Task{
		TaskID:        "OPE-66-render",
		Source:        "jira",
		Title:         "Warehouse rollout",
		Description:   "Customer-ready release",
		Labels:        []string{"data", "customer"},
		RequiredTools: []string{"sql"},
	}

	rawPlan := CrossDepartmentOrchestrator{}.Plan(task)
	plan, policy := PremiumOrchestrationPolicy{}.Apply(task, rawPlan)
	content := RenderOrchestrationPlan(plan, &policy, nil)
	for _, want := range []string{"# Cross-Department Orchestration Plan", "- Departments: operations, engineering", "- Tier: standard", "- Entitlement Status: upgrade-required", "- Billing Model: standard-blocked", "- Estimated Cost (USD): 11.00", "- Blocked Departments: data, customer-success"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in report, got %s", want, content)
		}
	}
	if strings.Contains(content, "- Human Handoff Team:") {
		t.Fatalf("did not expect handoff section in report: %s", content)
	}
}

func TestSchedulerExecutionRecordsOrchestrationPlanAndPolicy(t *testing.T) {
	ledger := &ObservabilityLedger{}
	task := Task{
		TaskID:        "OPE-66-exec",
		Source:        "linear",
		Title:         "Cross-team browser change",
		Description:   "Program-managed rollout",
		Labels:        []string{"ops"},
		Priority:      0,
		RiskLevel:     RiskMedium,
		RequiredTools: []string{"browser"},
	}

	record := Scheduler{}.Execute(task, "run-ope-66", ledger)
	entry := ledger.Load()[0]
	if got := strings.Join(record.OrchestrationPlan.Departments(), ", "); got != "operations, engineering" {
		t.Fatalf("unexpected orchestration plan: %+v", record.OrchestrationPlan)
	}
	if record.OrchestrationPolicy.UpgradeRequired || record.OrchestrationPolicy.EntitlementStatus != "included" || record.OrchestrationPolicy.BillingModel != "standard-included" || record.OrchestrationPolicy.EstimatedCostUSD != 3.0 {
		t.Fatalf("unexpected orchestration policy: %+v", record.OrchestrationPolicy)
	}
	if !hasTrace(entry["traces"].([]map[string]any), "orchestration.plan") || !hasTrace(entry["traces"].([]map[string]any), "orchestration.policy") {
		t.Fatalf("expected orchestration traces, got %+v", entry["traces"])
	}
	if !hasAudit(entry["audits"].([]map[string]any), "orchestration.plan") || !hasAudit(entry["audits"].([]map[string]any), "orchestration.policy") {
		t.Fatalf("expected orchestration audits, got %+v", entry["audits"])
	}
	policyAudit := findAudit(entry["audits"].([]map[string]any), "orchestration.policy")
	if policyAudit["details"].(map[string]any)["entitlement_status"] != "included" || policyAudit["details"].(map[string]any)["billing_model"] != "standard-included" {
		t.Fatalf("unexpected policy audit: %+v", policyAudit)
	}
}

func TestSchedulerCreatesHandoffForPolicyOrApprovalBlockers(t *testing.T) {
	ledger := &ObservabilityLedger{}
	task := Task{
		TaskID:        "OPE-66-handoff",
		Source:        "linear",
		Title:         "Customer analytics rollout",
		Description:   "Need cross-team coordination",
		Labels:        []string{"customer", "data"},
		RequiredTools: []string{"browser", "sql"},
	}

	record := Scheduler{}.Execute(task, "run-ope-66-handoff", ledger)
	entry := ledger.Load()[0]
	if record.HandoffRequest == nil || record.HandoffRequest.TargetTeam != "operations" {
		t.Fatalf("unexpected handoff request: %+v", record.HandoffRequest)
	}
	if !hasTrace(entry["traces"].([]map[string]any), "orchestration.handoff") || !hasAudit(entry["audits"].([]map[string]any), "orchestration.handoff") {
		t.Fatalf("expected handoff trace and audit, got %+v", entry)
	}
}

func hasTrace(items []map[string]any, want string) bool {
	for _, item := range items {
		if item["span"] == want {
			return true
		}
	}
	return false
}

func hasAudit(items []map[string]any, want string) bool {
	for _, item := range items {
		if item["action"] == want {
			return true
		}
	}
	return false
}

func findAudit(items []map[string]any, want string) map[string]any {
	for _, item := range items {
		if item["action"] == want {
			return item
		}
	}
	return nil
}
