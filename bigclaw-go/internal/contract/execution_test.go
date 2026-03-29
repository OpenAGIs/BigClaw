package contract

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func buildContract() ExecutionContract {
	return ExecutionContract{
		ContractID: "BIG-EPIC-18",
		Version:    "v4.0",
		Models: []ExecutionModel{
			{
				Name:  "ExecutionRequest",
				Owner: "runtime",
				Fields: []ExecutionField{
					requiredExecutionField("task_id", "string"),
					requiredExecutionField("actor", "string"),
					requiredExecutionField("requested_tools", "string[]"),
					optionalExecutionField("approval_token", "string"),
				},
			},
			{
				Name:  "ExecutionResponse",
				Owner: "runtime",
				Fields: []ExecutionField{
					requiredExecutionField("run_id", "string"),
					requiredExecutionField("status", "string"),
					requiredExecutionField("sandbox_profile", "string"),
				},
			},
		},
		APIs: []ExecutionAPISpec{
			{
				Name:               "start_execution",
				Method:             "POST",
				Path:               "/execution/runs",
				RequestModel:       "ExecutionRequest",
				ResponseModel:      "ExecutionResponse",
				RequiredPermission: "execution.run.write",
				EmittedAudits:      []string{"execution.run.started", "execution.permission.checked"},
				EmittedMetrics:     []string{"execution.request.count", "execution.duration.ms"},
			},
		},
		Permissions: []ExecutionPermission{
			{Name: "execution.run.write", Resource: "execution-run", Actions: []string{"create"}, Scopes: []string{"project", "workspace"}},
			{Name: "execution.run.approve", Resource: "execution-run", Actions: []string{"approve"}, Scopes: []string{"workspace"}},
			{Name: "execution.audit.read", Resource: "execution-audit", Actions: []string{"read"}, Scopes: []string{"workspace", "portfolio"}},
			{Name: "execution.orchestration.manage", Resource: "orchestration-plan", Actions: []string{"read", "update"}, Scopes: []string{"cross-team"}},
		},
		Roles: []ExecutionRole{
			{Name: "eng-lead", Personas: []string{"Eng Lead"}, GrantedPermissions: []string{"execution.run.write", "execution.run.approve"}, ScopeBindings: []string{"project"}, EscalationTarget: "vp-eng"},
			{Name: "platform-admin", Personas: []string{"Platform Admin"}, GrantedPermissions: []string{"execution.run.write", "execution.audit.read"}, ScopeBindings: []string{"workspace"}, EscalationTarget: "vp-eng"},
			{Name: "vp-eng", Personas: []string{"VP Eng"}, GrantedPermissions: []string{"execution.run.approve", "execution.audit.read"}, ScopeBindings: []string{"portfolio", "workspace"}, EscalationTarget: "none"},
			{Name: "cross-team-operator", Personas: []string{"Cross-Team Operator"}, GrantedPermissions: []string{"execution.run.write", "execution.orchestration.manage"}, ScopeBindings: []string{"cross-team", "project"}, EscalationTarget: "eng-lead"},
		},
		Metrics: []MetricDefinition{
			{Name: "execution.request.count", Unit: "count", Owner: "runtime"},
			{Name: "execution.duration.ms", Unit: "ms", Owner: "runtime"},
		},
		AuditPolicies: []AuditPolicy{
			{EventType: "execution.run.started", RequiredFields: []string{"task_id", "run_id", "actor"}, RetentionDays: 180, Severity: "info"},
			{EventType: "execution.permission.checked", RequiredFields: []string{"task_id", "actor", "permission", "allowed"}, RetentionDays: 180, Severity: "info"},
		},
	}
}

func TestExecutionContractAuditAcceptsWellFormedContract(t *testing.T) {
	contract := buildContract()
	audit := ExecutionContractLibrary{}.Audit(contract)
	report := RenderExecutionContractReport(contract, audit)
	if !audit.ReleaseReady() || audit.ReadinessScore() != 100 {
		t.Fatalf("expected release ready audit, got %+v", audit)
	}
	for _, want := range []string{"- Release Ready: True", "POST /execution/runs"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
		}
	}
}

func TestExecutionContractAuditSurfacesContractGaps(t *testing.T) {
	contract := buildContract()
	contract.Models[0] = ExecutionModel{
		Name:  "ExecutionRequest",
		Owner: "runtime",
		Fields: []ExecutionField{
			requiredExecutionField("task_id", "string"),
		},
	}
	contract.APIs[0] = ExecutionAPISpec{
		Name:               "start_execution",
		Method:             "POST",
		Path:               "/execution/runs",
		RequestModel:       "ExecutionRequest",
		ResponseModel:      "MissingResponse",
		RequiredPermission: "execution.run.approve",
		EmittedAudits:      []string{"execution.run.finished"},
		EmittedMetrics:     []string{"execution.queue.depth"},
	}
	contract.AuditPolicies[0] = AuditPolicy{EventType: "execution.run.started", RequiredFields: []string{"task_id"}, RetentionDays: 7, Severity: "info"}
	contract.Roles = []ExecutionRole{
		{Name: "eng-lead"},
		{Name: "platform-admin", Personas: []string{"Platform Admin"}, GrantedPermissions: []string{"execution.audit.override"}, ScopeBindings: []string{"workspace"}, EscalationTarget: "vp-eng"},
	}
	audit := ExecutionContractLibrary{}.Audit(contract)
	if !reflect.DeepEqual(audit.ModelsMissingRequiredFields, map[string][]string{"ExecutionRequest": {"actor", "requested_tools"}}) {
		t.Fatalf("unexpected models missing fields: %+v", audit.ModelsMissingRequiredFields)
	}
	if !reflect.DeepEqual(audit.UndefinedModelRefs, map[string][]string{"start_execution": {"MissingResponse"}}) {
		t.Fatalf("unexpected undefined model refs: %+v", audit.UndefinedModelRefs)
	}
	if audit.UndefinedPermissions != nil {
		t.Fatalf("expected undefined permissions to be empty, got %+v", audit.UndefinedPermissions)
	}
	if !reflect.DeepEqual(audit.MissingRoles, []string{"cross-team-operator", "vp-eng"}) {
		t.Fatalf("unexpected missing roles: %+v", audit.MissingRoles)
	}
	if !reflect.DeepEqual(audit.RolesMissingPersonas, []string{"eng-lead"}) {
		t.Fatalf("unexpected roles missing personas: %+v", audit.RolesMissingPersonas)
	}
	if !reflect.DeepEqual(audit.RolesMissingScopeBindings, []string{"eng-lead"}) {
		t.Fatalf("unexpected roles missing scope bindings: %+v", audit.RolesMissingScopeBindings)
	}
	if !reflect.DeepEqual(audit.RolesMissingEscalations, []string{"eng-lead"}) {
		t.Fatalf("unexpected roles missing escalations: %+v", audit.RolesMissingEscalations)
	}
	if !reflect.DeepEqual(audit.RolesMissingPermissions, []string{"eng-lead"}) {
		t.Fatalf("unexpected roles missing permissions: %+v", audit.RolesMissingPermissions)
	}
	if !reflect.DeepEqual(audit.UndefinedRolePermissions, map[string][]string{"platform-admin": {"execution.audit.override"}}) {
		t.Fatalf("unexpected undefined role permissions: %+v", audit.UndefinedRolePermissions)
	}
	if !reflect.DeepEqual(audit.APIsWithoutRoleCoverage, []string{"start_execution"}) {
		t.Fatalf("unexpected api role coverage gaps: %+v", audit.APIsWithoutRoleCoverage)
	}
	if !reflect.DeepEqual(audit.PermissionsWithoutRoles, []string{"execution.audit.read", "execution.orchestration.manage", "execution.run.approve", "execution.run.write"}) {
		t.Fatalf("unexpected permissions without roles: %+v", audit.PermissionsWithoutRoles)
	}
	if !reflect.DeepEqual(audit.UndefinedMetrics, map[string][]string{"start_execution": {"execution.queue.depth"}}) {
		t.Fatalf("unexpected undefined metrics: %+v", audit.UndefinedMetrics)
	}
	if !reflect.DeepEqual(audit.UndefinedAuditEvents, map[string][]string{"start_execution": {"execution.run.finished"}}) {
		t.Fatalf("unexpected undefined audit events: %+v", audit.UndefinedAuditEvents)
	}
	if !reflect.DeepEqual(audit.AuditPoliciesBelowRetention, []string{"execution.run.started"}) {
		t.Fatalf("unexpected retention gaps: %+v", audit.AuditPoliciesBelowRetention)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected release not ready")
	}
}

func TestExecutionContractRoundTripAndPermissionMatrix(t *testing.T) {
	contract := buildContract()
	auditPayload, err := json.Marshal(ExecutionContractLibrary{}.Audit(contract))
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}
	var audit ExecutionContractAudit
	if err := json.Unmarshal(auditPayload, &audit); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	contractPayload, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	var restored ExecutionContract
	if err := json.Unmarshal(contractPayload, &restored); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	matrix := NewExecutionPermissionMatrix(restored.Permissions, restored.Roles)
	decision := matrix.Evaluate([]string{"execution.run.write", "missing.permission"}, []string{"execution.run.write", "unknown.permission"})
	roleDecision := matrix.EvaluateRoles([]string{"execution.run.write", "execution.orchestration.manage"}, []string{"cross-team-operator", "unknown-role"})
	if !reflect.DeepEqual(restored, contract) {
		t.Fatalf("restored contract mismatch: got %+v want %+v", restored, contract)
	}
	if !audit.ReleaseReady() {
		t.Fatalf("expected ready audit, got %+v", audit)
	}
	if decision.Allowed || !reflect.DeepEqual(decision.GrantedPermissions, []string{"execution.run.write"}) || !reflect.DeepEqual(decision.MissingPermissions, []string{"missing.permission"}) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if !roleDecision.Allowed || !reflect.DeepEqual(roleDecision.GrantedPermissions, []string{"execution.orchestration.manage", "execution.run.write"}) || len(roleDecision.MissingPermissions) > 0 {
		t.Fatalf("unexpected role decision: %+v", roleDecision)
	}
}

func TestRenderExecutionContractReportIncludesRoleMatrix(t *testing.T) {
	contract := buildContract()
	report := RenderExecutionContractReport(contract, ExecutionContractLibrary{}.Audit(contract))
	for _, want := range []string{
		"- Roles: 4",
		"## Roles",
		"- eng-lead: personas=Eng Lead permissions=execution.run.write, execution.run.approve",
		"- Missing roles: none",
		"- Roles missing personas: none",
		"- Roles missing scope bindings: none",
		"- Roles missing escalation targets: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
		}
	}
}

func TestOperationsAPIContractDraftIsReleaseReady(t *testing.T) {
	contract := BuildOperationsAPIContract()
	audit := ExecutionContractLibrary{}.Audit(contract)
	report := RenderExecutionContractReport(contract, audit)
	if contract.ContractID != "OPE-131" || !audit.ReleaseReady() || len(contract.APIs) != 12 {
		t.Fatalf("unexpected operations contract: %+v audit=%+v", contract, audit)
	}
	for _, want := range []string{
		"GET /operations/dashboard",
		"GET /operations/runs/{run_id}",
		"GET /operations/queue/control-center",
		"GET /operations/risk/overview",
		"GET /operations/sla/overview",
		"GET /operations/regressions",
		"GET /operations/flows/{run_id}",
		"GET /operations/billing/entitlements",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
		}
	}
}

func TestOperationsAPIContractPermissionsCoverReadAndActionPaths(t *testing.T) {
	contract := BuildOperationsAPIContract()
	matrix := NewExecutionPermissionMatrix(contract.Permissions)
	viewer := matrix.Evaluate(
		[]string{"operations.dashboard.read", "operations.queue.read", "operations.run.read"},
		[]string{"operations.dashboard.read", "operations.queue.read", "operations.run.read"},
	)
	operator := matrix.Evaluate(
		[]string{"operations.queue.act", "operations.run.approve", "operations.billing.read"},
		[]string{"operations.queue.act", "operations.billing.read"},
	)
	if !viewer.Allowed {
		t.Fatalf("expected viewer to be allowed, got %+v", viewer)
	}
	if operator.Allowed || !reflect.DeepEqual(operator.MissingPermissions, []string{"operations.run.approve"}) {
		t.Fatalf("expected operator to miss approve permission, got %+v", operator)
	}
}

func TestExecutionContractAuditRequiresPersonaScopeAndEscalationMetadata(t *testing.T) {
	contract := buildContract()
	contract.Roles[0] = ExecutionRole{
		Name:               "eng-lead",
		Personas:           nil,
		GrantedPermissions: []string{"execution.run.write"},
		ScopeBindings:      nil,
		EscalationTarget:   "",
	}
	audit := ExecutionContractLibrary{}.Audit(contract)
	if !reflect.DeepEqual(audit.RolesMissingPersonas, []string{"eng-lead"}) {
		t.Fatalf("unexpected missing personas: %+v", audit.RolesMissingPersonas)
	}
	if !reflect.DeepEqual(audit.RolesMissingScopeBindings, []string{"eng-lead"}) {
		t.Fatalf("unexpected missing scope bindings: %+v", audit.RolesMissingScopeBindings)
	}
	if !reflect.DeepEqual(audit.RolesMissingEscalations, []string{"eng-lead"}) {
		t.Fatalf("unexpected missing escalations: %+v", audit.RolesMissingEscalations)
	}
	if audit.ReleaseReady() {
		t.Fatalf("expected release not ready")
	}
}
