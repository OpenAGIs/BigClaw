package legacyruntime

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/workflow"
)

func TestQueueToSchedulerExecutionRecordsFullChain(t *testing.T) {
	tempDir := t.TempDir()
	queue := NewQueue(filepath.Join(tempDir, "queue.json"))
	ledger := NewLedger(filepath.Join(tempDir, "ledger.json"))
	reportPath := filepath.Join(tempDir, "reports", "run-1.md")

	if err := queue.Enqueue(domain.Task{
		ID:            "BIG-502",
		Source:        "linear",
		Title:         "Record execution",
		Description:   "full chain",
		Priority:      0,
		RiskLevel:     domain.RiskMedium,
		RequiredTools: []string{"browser"},
	}); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}

	task, err := queue.DequeueTask()
	if err != nil {
		t.Fatalf("dequeue task: %v", err)
	}
	if task == nil {
		t.Fatal("expected task")
	}

	record, err := (Scheduler{}).Execute(*task, "run-1", ledger, reportPath)
	if err != nil {
		t.Fatalf("execute task: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	if record.Decision.Medium != "browser" {
		t.Fatalf("unexpected medium: %s", record.Decision.Medium)
	}
	if !record.Decision.Approved {
		t.Fatal("expected approved decision")
	}
	if record.Run.Status != "approved" {
		t.Fatalf("unexpected run status: %s", record.Run.Status)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("stat report: %v", err)
	}
	if _, err := os.Stat(strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".html"); err != nil {
		t.Fatalf("stat html report: %v", err)
	}
	reportBody, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(reportBody), "Status: approved") {
		t.Fatalf("expected approved status in report, got %q", string(reportBody))
	}
	if len(entries) != 1 {
		t.Fatalf("expected one ledger entry, got %d", len(entries))
	}

	traces := mapSlice(t, entries[0]["traces"])
	artifacts := mapSlice(t, entries[0]["artifacts"])
	audits := mapSlice(t, entries[0]["audits"])
	if traces[0]["span"] != "scheduler.decide" {
		t.Fatalf("unexpected first trace: %+v", traces[0])
	}
	if artifacts[0]["kind"] != "page" || artifacts[1]["kind"] != "report" {
		t.Fatalf("unexpected artifacts: %+v", artifacts)
	}
	details := mapValue(t, audits[0]["details"])
	if details["reason"] != "browser automation task" {
		t.Fatalf("unexpected scheduler reason: %+v", details)
	}
}

func TestHighRiskExecutionRecordsPendingApproval(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	record, err := (Scheduler{}).Execute(domain.Task{
		ID:          "BIG-502-risk",
		Source:      "jira",
		Title:       "Prod change",
		Description: "manual review",
		RiskLevel:   domain.RiskHigh,
	}, "run-2", ledger, "")
	if err != nil {
		t.Fatalf("execute task: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	if record.Decision.Approved {
		t.Fatal("expected pending approval")
	}
	if record.Run.Status != "needs-approval" {
		t.Fatalf("unexpected run status: %s", record.Run.Status)
	}
	audits := mapSlice(t, entries[0]["audits"])
	if audits[0]["outcome"] != "pending" {
		t.Fatalf("unexpected audit outcome: %+v", audits[0])
	}
}

func TestCrossDepartmentOrchestratorRoutesSecurityDataAndCustomerWork(t *testing.T) {
	task := domain.Task{
		ID:                 "OPE-66",
		Source:             "linear",
		Title:              "Coordinate customer analytics rollout approval",
		Description:        "Need stakeholder sign-off for warehouse-backed browser workflow",
		Labels:             []string{"data", "customer", "premium"},
		Priority:           0,
		RiskLevel:          domain.RiskHigh,
		RequiredTools:      []string{"browser", "sql"},
		AcceptanceCriteria: []string{"approval recorded"},
		ValidationPlan:     []string{"customer signoff"},
	}

	plan := workflowPlan(task)
	if plan.CollaborationMode != "cross-functional" {
		t.Fatalf("unexpected collaboration mode: %s", plan.CollaborationMode)
	}
	if got := plan.Departments(); !reflect.DeepEqual(got, []string{"operations", "engineering", "security", "data", "customer-success"}) {
		t.Fatalf("unexpected departments: %+v", got)
	}
	if got := plan.RequiredApprovals(); !reflect.DeepEqual(got, []string{"security-review"}) {
		t.Fatalf("unexpected approvals: %+v", got)
	}
}

func TestStandardPolicyLimitsAdvancedCrossDepartmentRouting(t *testing.T) {
	task := domain.Task{
		ID:            "OPE-66-standard",
		Source:        "linear",
		Title:         "Coordinate customer analytics rollout approval",
		Description:   "Need stakeholder sign-off for warehouse-backed browser workflow",
		Labels:        []string{"data", "customer"},
		RequiredTools: []string{"browser", "sql"},
		RiskLevel:     domain.RiskHigh,
	}

	plan := workflowPlan(task)
	constrained, policy := workflowPolicy(task, plan)
	if constrained.CollaborationMode != "tier-limited" {
		t.Fatalf("unexpected collaboration mode: %s", constrained.CollaborationMode)
	}
	if got := constrained.Departments(); !reflect.DeepEqual(got, []string{"operations", "engineering"}) {
		t.Fatalf("unexpected constrained departments: %+v", got)
	}
	if !policy.UpgradeRequired || policy.EntitlementStatus != "upgrade-required" || policy.BillingModel != "standard-blocked" {
		t.Fatalf("unexpected policy decision: %+v", policy)
	}
	if policy.IncludedUsageUnits != 2 || policy.OverageUsageUnits != 3 || policy.OverageCostUSD != 12.0 || policy.EstimatedCostUSD != 15.0 {
		t.Fatalf("unexpected policy costs: %+v", policy)
	}
	if !reflect.DeepEqual(policy.BlockedDepartments, []string{"security", "data", "customer-success"}) {
		t.Fatalf("unexpected blocked departments: %+v", policy.BlockedDepartments)
	}
}

func TestRenderOrchestrationPlanListsHandoffsAndPolicy(t *testing.T) {
	task := domain.Task{
		ID:            "OPE-66-render",
		Source:        "jira",
		Title:         "Warehouse rollout",
		Description:   "Customer-ready release",
		Labels:        []string{"data", "customer"},
		RequiredTools: []string{"sql"},
	}

	plan := workflowPlan(task)
	constrained, policy := workflowPolicy(task, plan)
	content := RenderOrchestrationPlan(constrained, &policy, nil)

	for _, want := range []string{
		"# Cross-Department Orchestration Plan",
		"- Departments: operations, engineering",
		"- Tier: standard",
		"- Entitlement Status: upgrade-required",
		"- Billing Model: standard-blocked",
		"- Estimated Cost (USD): 11.00",
		"- Blocked Departments: data, customer-success",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in content:\n%s", want, content)
		}
	}
	if strings.Contains(content, "- Human Handoff Team:") {
		t.Fatalf("did not expect handoff section in content:\n%s", content)
	}
}

func TestSchedulerExecutionRecordsOrchestrationPlanAndPolicy(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	record, err := (Scheduler{}).Execute(domain.Task{
		ID:            "OPE-66-exec",
		Source:        "linear",
		Title:         "Cross-team browser change",
		Description:   "Program-managed rollout",
		Labels:        []string{"ops"},
		Priority:      0,
		RiskLevel:     domain.RiskMedium,
		RequiredTools: []string{"browser"},
	}, "run-ope-66", ledger, "")
	if err != nil {
		t.Fatalf("execute task: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	if record.OrchestrationPlan == nil || record.OrchestrationPolicy == nil {
		t.Fatalf("expected orchestration data: %+v", record)
	}
	if got := record.OrchestrationPlan.Departments(); !reflect.DeepEqual(got, []string{"operations", "engineering"}) {
		t.Fatalf("unexpected departments: %+v", got)
	}
	if record.OrchestrationPolicy.UpgradeRequired || record.OrchestrationPolicy.EntitlementStatus != "included" || record.OrchestrationPolicy.BillingModel != "standard-included" || record.OrchestrationPolicy.EstimatedCostUSD != 3.0 {
		t.Fatalf("unexpected policy: %+v", record.OrchestrationPolicy)
	}
	assertTraceAndAudit(t, entries[0], "orchestration.plan")
	assertTraceAndAudit(t, entries[0], "orchestration.policy")
	policyAudit := findAudit(t, entries[0], "orchestration.policy")
	details := mapValue(t, policyAudit["details"])
	if details["entitlement_status"] != "included" || details["billing_model"] != "standard-included" {
		t.Fatalf("unexpected policy audit: %+v", policyAudit)
	}
}

func TestSchedulerCreatesHandoffForPolicyOrApprovalBlockers(t *testing.T) {
	ledger := NewLedger(filepath.Join(t.TempDir(), "ledger.json"))
	record, err := (Scheduler{}).Execute(domain.Task{
		ID:            "OPE-66-handoff",
		Source:        "linear",
		Title:         "Customer analytics rollout",
		Description:   "Need cross-team coordination",
		Labels:        []string{"customer", "data"},
		RequiredTools: []string{"browser", "sql"},
	}, "run-ope-66-handoff", ledger, "")
	if err != nil {
		t.Fatalf("execute task: %v", err)
	}
	entries, err := ledger.Load()
	if err != nil {
		t.Fatalf("load ledger: %v", err)
	}

	if record.HandoffRequest == nil || record.HandoffRequest.TargetTeam != "operations" {
		t.Fatalf("unexpected handoff request: %+v", record.HandoffRequest)
	}
	assertTraceAndAudit(t, entries[0], "orchestration.handoff")
}

func workflowPlan(task domain.Task) workflow.OrchestrationPlan {
	return workflow.CrossDepartmentOrchestrator{}.Plan(task)
}

func workflowPolicy(task domain.Task, plan workflow.OrchestrationPlan) (workflow.OrchestrationPlan, workflow.OrchestrationPolicyDecision) {
	return workflow.PremiumOrchestrationPolicy{}.Apply(task, plan)
}

func assertTraceAndAudit(t *testing.T, entry map[string]any, name string) {
	t.Helper()
	traces := mapSlice(t, entry["traces"])
	foundTrace := false
	for _, trace := range traces {
		if trace["span"] == name {
			foundTrace = true
			break
		}
	}
	if !foundTrace {
		t.Fatalf("missing trace %q in %+v", name, traces)
	}
	_ = findAudit(t, entry, name)
}

func findAudit(t *testing.T, entry map[string]any, name string) map[string]any {
	t.Helper()
	audits := mapSlice(t, entry["audits"])
	for _, audit := range audits {
		if audit["action"] == name {
			return audit
		}
	}
	t.Fatalf("missing audit %q in %+v", name, audits)
	return nil
}

func mapSlice(t *testing.T, raw any) []map[string]any {
	t.Helper()
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", raw)
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, mapValue(t, item))
	}
	return out
}

func mapValue(t *testing.T, raw any) map[string]any {
	t.Helper()
	value, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", raw)
	}
	return value
}
