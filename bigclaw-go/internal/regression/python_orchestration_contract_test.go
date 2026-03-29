package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonOrchestrationContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "orchestration_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.orchestration import (
    CrossDepartmentOrchestrator,
    PremiumOrchestrationPolicy,
    render_orchestration_plan,
)
from bigclaw.scheduler import Scheduler

task = Task(
    task_id="OPE-66",
    source="linear",
    title="Coordinate customer analytics rollout approval",
    description="Need stakeholder sign-off for warehouse-backed browser workflow",
    labels=["data", "customer", "premium"],
    priority=Priority.P0,
    risk_level=RiskLevel.HIGH,
    required_tools=["browser", "sql"],
    acceptance_criteria=["approval recorded"],
    validation_plan=["customer signoff"],
)
plan = CrossDepartmentOrchestrator().plan(task)
cross_department = {
    "collaboration_mode": plan.collaboration_mode,
    "departments": plan.departments,
    "required_approvals": plan.required_approvals,
}

task = Task(
    task_id="OPE-66-standard",
    source="linear",
    title="Coordinate customer analytics rollout approval",
    description="Need stakeholder sign-off for warehouse-backed browser workflow",
    labels=["data", "customer"],
    required_tools=["browser", "sql"],
    risk_level=RiskLevel.HIGH,
)
raw_plan = CrossDepartmentOrchestrator().plan(task)
plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)
standard = {
    "collaboration_mode": plan.collaboration_mode,
    "departments": plan.departments,
    "upgrade_required": policy.upgrade_required,
    "entitlement_status": policy.entitlement_status,
    "billing_model": policy.billing_model,
    "included_usage_units": policy.included_usage_units,
    "overage_usage_units": policy.overage_usage_units,
    "overage_cost_usd": policy.overage_cost_usd,
    "estimated_cost_usd": policy.estimated_cost_usd,
    "blocked_departments": policy.blocked_departments,
}

task = Task(
    task_id="OPE-66-render",
    source="jira",
    title="Warehouse rollout",
    description="Customer-ready release",
    labels=["data", "customer"],
    required_tools=["sql"],
)
raw_plan = CrossDepartmentOrchestrator().plan(task)
plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)
content = render_orchestration_plan(plan, policy)
rendered = {
    "has_title": "# Cross-Department Orchestration Plan" in content,
    "has_departments": "- Departments: operations, engineering" in content,
    "has_tier": "- Tier: standard" in content,
    "has_entitlement": "- Entitlement Status: upgrade-required" in content,
    "has_billing": "- Billing Model: standard-blocked" in content,
    "has_estimated_cost": "- Estimated Cost (USD): 11.00" in content,
    "has_blocked_departments": "- Blocked Departments: data, customer-success" in content,
    "has_handoff": "- Human Handoff Team:" in content,
}

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="OPE-66-exec",
        source="linear",
        title="Cross-team browser change",
        description="Program-managed rollout",
        labels=["ops"],
        priority=Priority.P0,
        risk_level=RiskLevel.MEDIUM,
        required_tools=["browser"],
    )
    record = Scheduler().execute(task, run_id="run-ope-66", ledger=ledger)
    entry = ledger.load()[0]
    policy_audit = next(audit for audit in entry["audits"] if audit["action"] == "orchestration.policy")
    execution = {
        "has_plan": record.orchestration_plan is not None,
        "has_policy": record.orchestration_policy is not None,
        "departments": record.orchestration_plan.departments,
        "upgrade_required": record.orchestration_policy.upgrade_required,
        "entitlement_status": record.orchestration_policy.entitlement_status,
        "billing_model": record.orchestration_policy.billing_model,
        "estimated_cost_usd": record.orchestration_policy.estimated_cost_usd,
        "has_plan_trace": any(trace["span"] == "orchestration.plan" for trace in entry["traces"]),
        "has_policy_trace": any(trace["span"] == "orchestration.policy" for trace in entry["traces"]),
        "has_plan_audit": any(audit["action"] == "orchestration.plan" for audit in entry["audits"]),
        "has_policy_audit": any(audit["action"] == "orchestration.policy" for audit in entry["audits"]),
        "policy_audit_entitlement": policy_audit["details"]["entitlement_status"],
        "policy_audit_billing": policy_audit["details"]["billing_model"],
    }

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="OPE-66-handoff",
        source="linear",
        title="Customer analytics rollout",
        description="Need cross-team coordination",
        labels=["customer", "data"],
        required_tools=["browser", "sql"],
    )
    record = Scheduler().execute(task, run_id="run-ope-66-handoff", ledger=ledger)
    entry = ledger.load()[0]
    handoff = {
        "has_handoff": record.handoff_request is not None,
        "target_team": record.handoff_request.target_team if record.handoff_request is not None else "",
        "has_handoff_trace": any(trace["span"] == "orchestration.handoff" for trace in entry["traces"]),
        "has_handoff_audit": any(audit["action"] == "orchestration.handoff" for audit in entry["audits"]),
    }

print(json.dumps({
    "cross_department": cross_department,
    "standard": standard,
    "rendered": rendered,
    "execution": execution,
    "handoff": handoff,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write orchestration contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run orchestration contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		CrossDepartment struct {
			CollaborationMode string   `json:"collaboration_mode"`
			Departments       []string `json:"departments"`
			RequiredApprovals []string `json:"required_approvals"`
		} `json:"cross_department"`
		Standard struct {
			CollaborationMode string    `json:"collaboration_mode"`
			Departments       []string  `json:"departments"`
			UpgradeRequired   bool      `json:"upgrade_required"`
			EntitlementStatus string    `json:"entitlement_status"`
			BillingModel      string    `json:"billing_model"`
			IncludedUsage     int       `json:"included_usage_units"`
			OverageUsage      int       `json:"overage_usage_units"`
			OverageCostUSD    float64   `json:"overage_cost_usd"`
			EstimatedCostUSD  float64   `json:"estimated_cost_usd"`
			BlockedDepartments []string `json:"blocked_departments"`
		} `json:"standard"`
		Rendered struct {
			HasTitle              bool `json:"has_title"`
			HasDepartments        bool `json:"has_departments"`
			HasTier               bool `json:"has_tier"`
			HasEntitlement        bool `json:"has_entitlement"`
			HasBilling            bool `json:"has_billing"`
			HasEstimatedCost      bool `json:"has_estimated_cost"`
			HasBlockedDepartments bool `json:"has_blocked_departments"`
			HasHandoff            bool `json:"has_handoff"`
		} `json:"rendered"`
		Execution struct {
			HasPlan               bool     `json:"has_plan"`
			HasPolicy             bool     `json:"has_policy"`
			Departments           []string `json:"departments"`
			UpgradeRequired       bool     `json:"upgrade_required"`
			EntitlementStatus     string   `json:"entitlement_status"`
			BillingModel          string   `json:"billing_model"`
			EstimatedCostUSD      float64  `json:"estimated_cost_usd"`
			HasPlanTrace          bool     `json:"has_plan_trace"`
			HasPolicyTrace        bool     `json:"has_policy_trace"`
			HasPlanAudit          bool     `json:"has_plan_audit"`
			HasPolicyAudit        bool     `json:"has_policy_audit"`
			PolicyAuditEntitlement string  `json:"policy_audit_entitlement"`
			PolicyAuditBilling    string   `json:"policy_audit_billing"`
		} `json:"execution"`
		Handoff struct {
			HasHandoff      bool   `json:"has_handoff"`
			TargetTeam      string `json:"target_team"`
			HasHandoffTrace bool   `json:"has_handoff_trace"`
			HasHandoffAudit bool   `json:"has_handoff_audit"`
		} `json:"handoff"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode orchestration contract output: %v\n%s", err, string(output))
	}

	if decoded.CrossDepartment.CollaborationMode != "cross-functional" ||
		len(decoded.CrossDepartment.Departments) != 5 ||
		decoded.CrossDepartment.Departments[0] != "operations" ||
		decoded.CrossDepartment.Departments[1] != "engineering" ||
		decoded.CrossDepartment.Departments[2] != "security" ||
		decoded.CrossDepartment.Departments[3] != "data" ||
		decoded.CrossDepartment.Departments[4] != "customer-success" ||
		len(decoded.CrossDepartment.RequiredApprovals) != 1 ||
		decoded.CrossDepartment.RequiredApprovals[0] != "security-review" {
		t.Fatalf("unexpected cross-department payload: %+v", decoded.CrossDepartment)
	}
	if decoded.Standard.CollaborationMode != "tier-limited" ||
		len(decoded.Standard.Departments) != 2 ||
		decoded.Standard.Departments[0] != "operations" ||
		decoded.Standard.Departments[1] != "engineering" ||
		!decoded.Standard.UpgradeRequired ||
		decoded.Standard.EntitlementStatus != "upgrade-required" ||
		decoded.Standard.BillingModel != "standard-blocked" ||
		decoded.Standard.IncludedUsage != 2 ||
		decoded.Standard.OverageUsage != 3 ||
		decoded.Standard.OverageCostUSD != 12.0 ||
		decoded.Standard.EstimatedCostUSD != 15.0 ||
		len(decoded.Standard.BlockedDepartments) != 3 ||
		decoded.Standard.BlockedDepartments[0] != "security" ||
		decoded.Standard.BlockedDepartments[1] != "data" ||
		decoded.Standard.BlockedDepartments[2] != "customer-success" {
		t.Fatalf("unexpected standard policy payload: %+v", decoded.Standard)
	}
	if !decoded.Rendered.HasTitle || !decoded.Rendered.HasDepartments || !decoded.Rendered.HasTier || !decoded.Rendered.HasEntitlement || !decoded.Rendered.HasBilling || !decoded.Rendered.HasEstimatedCost || !decoded.Rendered.HasBlockedDepartments || decoded.Rendered.HasHandoff {
		t.Fatalf("unexpected rendered orchestration payload: %+v", decoded.Rendered)
	}
	if !decoded.Execution.HasPlan || !decoded.Execution.HasPolicy ||
		len(decoded.Execution.Departments) != 2 ||
		decoded.Execution.Departments[0] != "operations" ||
		decoded.Execution.Departments[1] != "engineering" ||
		decoded.Execution.UpgradeRequired ||
		decoded.Execution.EntitlementStatus != "included" ||
		decoded.Execution.BillingModel != "standard-included" ||
		decoded.Execution.EstimatedCostUSD != 3.0 ||
		!decoded.Execution.HasPlanTrace || !decoded.Execution.HasPolicyTrace ||
		!decoded.Execution.HasPlanAudit || !decoded.Execution.HasPolicyAudit ||
		decoded.Execution.PolicyAuditEntitlement != "included" ||
		decoded.Execution.PolicyAuditBilling != "standard-included" {
		t.Fatalf("unexpected orchestration execution payload: %+v", decoded.Execution)
	}
	if !decoded.Handoff.HasHandoff || decoded.Handoff.TargetTeam != "operations" || !decoded.Handoff.HasHandoffTrace || !decoded.Handoff.HasHandoffAudit {
		t.Fatalf("unexpected orchestration handoff payload: %+v", decoded.Handoff)
	}
}
