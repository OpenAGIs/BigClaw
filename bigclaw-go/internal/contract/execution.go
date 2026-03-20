package contract

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type ExecutionField struct {
	Name        string `json:"name"`
	FieldType   string `json:"field_type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

func (field *ExecutionField) UnmarshalJSON(data []byte) error {
	type rawExecutionField struct {
		Name        string `json:"name"`
		FieldType   string `json:"field_type"`
		Required    *bool  `json:"required"`
		Description string `json:"description,omitempty"`
	}
	var raw rawExecutionField
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	field.Name = raw.Name
	field.FieldType = raw.FieldType
	field.Required = true
	if raw.Required != nil {
		field.Required = *raw.Required
	}
	field.Description = raw.Description
	return nil
}

func requiredExecutionField(name string, fieldType string) ExecutionField {
	return ExecutionField{Name: name, FieldType: fieldType, Required: true}
}

func optionalExecutionField(name string, fieldType string) ExecutionField {
	return ExecutionField{Name: name, FieldType: fieldType, Required: false}
}

type ExecutionModel struct {
	Name   string           `json:"name"`
	Fields []ExecutionField `json:"fields,omitempty"`
	Owner  string           `json:"owner,omitempty"`
}

func (model ExecutionModel) RequiredFields() []string {
	out := make([]string, 0)
	for _, field := range model.Fields {
		if field.Required {
			out = append(out, field.Name)
		}
	}
	return out
}

type ExecutionAPISpec struct {
	Name               string   `json:"name"`
	Method             string   `json:"method"`
	Path               string   `json:"path"`
	RequestModel       string   `json:"request_model,omitempty"`
	ResponseModel      string   `json:"response_model,omitempty"`
	RequiredPermission string   `json:"required_permission,omitempty"`
	EmittedAudits      []string `json:"emitted_audits,omitempty"`
	EmittedMetrics     []string `json:"emitted_metrics,omitempty"`
}

type ExecutionPermission struct {
	Name     string   `json:"name"`
	Resource string   `json:"resource,omitempty"`
	Actions  []string `json:"actions,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

type ExecutionRole struct {
	Name               string   `json:"name"`
	Personas           []string `json:"personas,omitempty"`
	GrantedPermissions []string `json:"granted_permissions,omitempty"`
	ScopeBindings      []string `json:"scope_bindings,omitempty"`
	EscalationTarget   string   `json:"escalation_target,omitempty"`
}

type PermissionCheckResult struct {
	Allowed            bool     `json:"allowed"`
	GrantedPermissions []string `json:"granted_permissions,omitempty"`
	MissingPermissions []string `json:"missing_permissions,omitempty"`
}

type ExecutionPermissionMatrix struct {
	permissions map[string]ExecutionPermission
	roles       map[string]ExecutionRole
}

func NewExecutionPermissionMatrix(permissions []ExecutionPermission, roles ...[]ExecutionRole) ExecutionPermissionMatrix {
	matrix := ExecutionPermissionMatrix{
		permissions: make(map[string]ExecutionPermission, len(permissions)),
		roles:       make(map[string]ExecutionRole),
	}
	for _, permission := range permissions {
		matrix.permissions[permission.Name] = permission
	}
	if len(roles) > 0 {
		for _, role := range roles[0] {
			matrix.roles[role.Name] = role
		}
	}
	return matrix
}

func (matrix ExecutionPermissionMatrix) Evaluate(requiredPermissions []string, grantedPermissions []string) PermissionCheckResult {
	granted := make([]string, 0)
	grantedSet := make(map[string]struct{})
	for _, permission := range grantedPermissions {
		if _, ok := matrix.permissions[permission]; ok {
			grantedSet[permission] = struct{}{}
		}
	}
	for permission := range grantedSet {
		granted = append(granted, permission)
	}
	sort.Strings(granted)
	missing := make([]string, 0)
	for _, permission := range requiredPermissions {
		if _, ok := grantedSet[permission]; !ok {
			missing = append(missing, permission)
		}
	}
	return PermissionCheckResult{
		Allowed:            len(missing) == 0,
		GrantedPermissions: granted,
		MissingPermissions: missing,
	}
}

func (matrix ExecutionPermissionMatrix) EvaluateRoles(requiredPermissions []string, actorRoles []string) PermissionCheckResult {
	grantedSet := make(map[string]struct{})
	for _, roleName := range actorRoles {
		role, ok := matrix.roles[roleName]
		if !ok {
			continue
		}
		for _, permission := range role.GrantedPermissions {
			if _, ok := matrix.permissions[permission]; ok {
				grantedSet[permission] = struct{}{}
			}
		}
	}
	granted := make([]string, 0, len(grantedSet))
	for permission := range grantedSet {
		granted = append(granted, permission)
	}
	sort.Strings(granted)
	return matrix.Evaluate(requiredPermissions, granted)
}

type MetricDefinition struct {
	Name        string `json:"name"`
	Unit        string `json:"unit,omitempty"`
	Owner       string `json:"owner,omitempty"`
	Description string `json:"description,omitempty"`
}

type AuditPolicy struct {
	EventType      string   `json:"event_type"`
	RequiredFields []string `json:"required_fields,omitempty"`
	RetentionDays  int      `json:"retention_days"`
	Severity       string   `json:"severity"`
}

func (policy *AuditPolicy) UnmarshalJSON(data []byte) error {
	type rawAuditPolicy struct {
		EventType      string   `json:"event_type"`
		RequiredFields []string `json:"required_fields,omitempty"`
		RetentionDays  *int     `json:"retention_days"`
		Severity       *string  `json:"severity"`
	}
	var raw rawAuditPolicy
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	policy.EventType = raw.EventType
	policy.RequiredFields = append([]string(nil), raw.RequiredFields...)
	policy.RetentionDays = 30
	if raw.RetentionDays != nil {
		policy.RetentionDays = *raw.RetentionDays
	}
	policy.Severity = "info"
	if raw.Severity != nil {
		policy.Severity = *raw.Severity
	}
	return nil
}

type ExecutionContract struct {
	ContractID    string                `json:"contract_id"`
	Version       string                `json:"version"`
	Models        []ExecutionModel      `json:"models,omitempty"`
	APIs          []ExecutionAPISpec    `json:"apis,omitempty"`
	Permissions   []ExecutionPermission `json:"permissions,omitempty"`
	Roles         []ExecutionRole       `json:"roles,omitempty"`
	Metrics       []MetricDefinition    `json:"metrics,omitempty"`
	AuditPolicies []AuditPolicy         `json:"audit_policies,omitempty"`
}

type ExecutionContractAudit struct {
	ContractID                  string              `json:"contract_id"`
	Version                     string              `json:"version"`
	ModelsMissingRequiredFields map[string][]string `json:"models_missing_required_fields,omitempty"`
	APIsMissingPermissions      []string            `json:"apis_missing_permissions,omitempty"`
	APIsMissingAudits           []string            `json:"apis_missing_audits,omitempty"`
	APIsMissingMetrics          []string            `json:"apis_missing_metrics,omitempty"`
	UndefinedModelRefs          map[string][]string `json:"undefined_model_refs,omitempty"`
	UndefinedPermissions        map[string]string   `json:"undefined_permissions,omitempty"`
	MissingRoles                []string            `json:"missing_roles,omitempty"`
	RolesMissingPersonas        []string            `json:"roles_missing_personas,omitempty"`
	RolesMissingScopeBindings   []string            `json:"roles_missing_scope_bindings,omitempty"`
	RolesMissingEscalations     []string            `json:"roles_missing_escalation_targets,omitempty"`
	RolesMissingPermissions     []string            `json:"roles_missing_permissions,omitempty"`
	UndefinedRolePermissions    map[string][]string `json:"undefined_role_permissions,omitempty"`
	PermissionsWithoutRoles     []string            `json:"permissions_without_roles,omitempty"`
	APIsWithoutRoleCoverage     []string            `json:"apis_without_role_coverage,omitempty"`
	UndefinedMetrics            map[string][]string `json:"undefined_metrics,omitempty"`
	UndefinedAuditEvents        map[string][]string `json:"undefined_audit_events,omitempty"`
	AuditPoliciesBelowRetention []string            `json:"audit_policies_below_retention,omitempty"`
}

func (audit ExecutionContractAudit) ReadinessScore() float64 {
	apiCount := len(audit.APIsMissingPermissions) + len(audit.APIsMissingAudits) + len(audit.APIsMissingMetrics)
	if apiCount < 1 {
		apiCount = 1
	}
	issueCount := len(audit.ModelsMissingRequiredFields) +
		len(audit.APIsMissingPermissions) +
		len(audit.APIsMissingAudits) +
		len(audit.APIsMissingMetrics) +
		len(audit.UndefinedModelRefs) +
		len(audit.UndefinedPermissions) +
		len(audit.MissingRoles) +
		len(audit.RolesMissingPersonas) +
		len(audit.RolesMissingScopeBindings) +
		len(audit.RolesMissingEscalations) +
		len(audit.RolesMissingPermissions) +
		len(audit.UndefinedRolePermissions) +
		len(audit.PermissionsWithoutRoles) +
		len(audit.APIsWithoutRoleCoverage) +
		len(audit.UndefinedMetrics) +
		len(audit.UndefinedAuditEvents) +
		len(audit.AuditPoliciesBelowRetention)
	if issueCount == 0 {
		return 100
	}
	penalty := float64(issueCount) * (100 / float64(apiCount))
	if penalty > 100 {
		penalty = 100
	}
	return round1(100 - penalty)
}

func (audit ExecutionContractAudit) ReleaseReady() bool {
	return audit.ReadinessScore() == 100
}

type ExecutionContractLibrary struct{}

func (ExecutionContractLibrary) Audit(contract ExecutionContract) ExecutionContractAudit {
	requiredModelFields := map[string][]string{
		"ExecutionRequest":  {"task_id", "actor", "requested_tools"},
		"ExecutionResponse": {"run_id", "status", "sandbox_profile"},
	}
	requiredRoles := []string{"eng-lead", "platform-admin", "vp-eng", "cross-team-operator"}

	modelNames := make(map[string]struct{}, len(contract.Models))
	permissionNames := make(map[string]struct{}, len(contract.Permissions))
	metricNames := make(map[string]struct{}, len(contract.Metrics))
	auditEvents := make(map[string]struct{}, len(contract.AuditPolicies))
	roleNames := make(map[string]struct{}, len(contract.Roles))
	for _, model := range contract.Models {
		modelNames[model.Name] = struct{}{}
	}
	for _, permission := range contract.Permissions {
		permissionNames[permission.Name] = struct{}{}
	}
	for _, metric := range contract.Metrics {
		metricNames[metric.Name] = struct{}{}
	}
	for _, policy := range contract.AuditPolicies {
		auditEvents[policy.EventType] = struct{}{}
	}
	for _, role := range contract.Roles {
		roleNames[role.Name] = struct{}{}
	}

	modelsMissingRequiredFields := make(map[string][]string)
	for _, model := range contract.Models {
		expectedFields := requiredModelFields[model.Name]
		if len(expectedFields) == 0 {
			continue
		}
		required := make(map[string]struct{})
		for _, field := range model.RequiredFields() {
			required[field] = struct{}{}
		}
		missing := make([]string, 0)
		for _, field := range expectedFields {
			if _, ok := required[field]; !ok {
				missing = append(missing, field)
			}
		}
		if len(missing) > 0 {
			modelsMissingRequiredFields[model.Name] = missing
		}
	}

	undefinedModelRefs := make(map[string][]string)
	undefinedPermissions := make(map[string]string)
	missingRoles := make([]string, 0)
	for _, role := range requiredRoles {
		if _, ok := roleNames[role]; !ok {
			missingRoles = append(missingRoles, role)
		}
	}
	sort.Strings(missingRoles)

	rolesMissingPersonas := make([]string, 0)
	rolesMissingScopeBindings := make([]string, 0)
	rolesMissingEscalations := make([]string, 0)
	rolesMissingPermissions := make([]string, 0)
	undefinedRolePermissions := make(map[string][]string)
	permissionsGrantedByRoles := make(map[string]struct{})
	apisWithoutRoleCoverage := make([]string, 0)
	undefinedMetrics := make(map[string][]string)
	undefinedAuditEvents := make(map[string][]string)
	apisMissingPermissions := make([]string, 0)
	apisMissingAudits := make([]string, 0)
	apisMissingMetrics := make([]string, 0)

	for _, api := range contract.APIs {
		missingModels := make([]string, 0)
		for _, modelName := range []string{api.RequestModel, api.ResponseModel} {
			if strings.TrimSpace(modelName) == "" {
				continue
			}
			if _, ok := modelNames[modelName]; !ok {
				missingModels = append(missingModels, modelName)
			}
		}
		if len(missingModels) > 0 {
			undefinedModelRefs[api.Name] = missingModels
		}
		if strings.TrimSpace(api.RequiredPermission) == "" {
			apisMissingPermissions = append(apisMissingPermissions, api.Name)
		} else if _, ok := permissionNames[api.RequiredPermission]; !ok {
			undefinedPermissions[api.Name] = api.RequiredPermission
		}
		if len(api.EmittedAudits) == 0 {
			apisMissingAudits = append(apisMissingAudits, api.Name)
		} else {
			missingEvents := make([]string, 0)
			for _, event := range api.EmittedAudits {
				if _, ok := auditEvents[event]; !ok {
					missingEvents = append(missingEvents, event)
				}
			}
			if len(missingEvents) > 0 {
				undefinedAuditEvents[api.Name] = missingEvents
			}
		}
		if len(api.EmittedMetrics) == 0 {
			apisMissingMetrics = append(apisMissingMetrics, api.Name)
		} else {
			missingMetricDefs := make([]string, 0)
			for _, metric := range api.EmittedMetrics {
				if _, ok := metricNames[metric]; !ok {
					missingMetricDefs = append(missingMetricDefs, metric)
				}
			}
			if len(missingMetricDefs) > 0 {
				undefinedMetrics[api.Name] = missingMetricDefs
			}
		}
	}

	for _, role := range contract.Roles {
		if len(role.Personas) == 0 {
			rolesMissingPersonas = append(rolesMissingPersonas, role.Name)
		}
		if len(role.ScopeBindings) == 0 {
			rolesMissingScopeBindings = append(rolesMissingScopeBindings, role.Name)
		}
		if strings.TrimSpace(role.EscalationTarget) == "" {
			rolesMissingEscalations = append(rolesMissingEscalations, role.Name)
		}
		if len(role.GrantedPermissions) == 0 {
			rolesMissingPermissions = append(rolesMissingPermissions, role.Name)
			continue
		}
		missingPermissions := make([]string, 0)
		for _, permission := range role.GrantedPermissions {
			if _, ok := permissionNames[permission]; !ok {
				missingPermissions = append(missingPermissions, permission)
				continue
			}
			permissionsGrantedByRoles[permission] = struct{}{}
		}
		if len(missingPermissions) > 0 {
			undefinedRolePermissions[role.Name] = missingPermissions
		}
	}

	for _, api := range contract.APIs {
		if strings.TrimSpace(api.RequiredPermission) == "" {
			continue
		}
		if _, ok := permissionNames[api.RequiredPermission]; ok {
			if _, covered := permissionsGrantedByRoles[api.RequiredPermission]; !covered {
				apisWithoutRoleCoverage = append(apisWithoutRoleCoverage, api.Name)
			}
		}
	}
	sort.Strings(apisWithoutRoleCoverage)

	permissionsWithoutRoles := make([]string, 0)
	for _, permission := range contract.Permissions {
		if _, ok := permissionsGrantedByRoles[permission.Name]; !ok {
			permissionsWithoutRoles = append(permissionsWithoutRoles, permission.Name)
		}
	}
	sort.Strings(permissionsWithoutRoles)

	auditPoliciesBelowRetention := make([]string, 0)
	for _, policy := range contract.AuditPolicies {
		if policy.RetentionDays < 30 {
			auditPoliciesBelowRetention = append(auditPoliciesBelowRetention, policy.EventType)
		}
	}
	sort.Strings(auditPoliciesBelowRetention)

	return ExecutionContractAudit{
		ContractID:                  contract.ContractID,
		Version:                     contract.Version,
		ModelsMissingRequiredFields: emptySliceMapIfNil(modelsMissingRequiredFields),
		APIsMissingPermissions:      apisMissingPermissions,
		APIsMissingAudits:           apisMissingAudits,
		APIsMissingMetrics:          apisMissingMetrics,
		UndefinedModelRefs:          emptySliceMapIfNil(undefinedModelRefs),
		UndefinedPermissions:        emptyStringMapIfNil(undefinedPermissions),
		MissingRoles:                missingRoles,
		RolesMissingPersonas:        sortStrings(rolesMissingPersonas),
		RolesMissingScopeBindings:   sortStrings(rolesMissingScopeBindings),
		RolesMissingEscalations:     sortStrings(rolesMissingEscalations),
		RolesMissingPermissions:     sortStrings(rolesMissingPermissions),
		UndefinedRolePermissions:    emptySliceMapIfNil(undefinedRolePermissions),
		PermissionsWithoutRoles:     permissionsWithoutRoles,
		APIsWithoutRoleCoverage:     apisWithoutRoleCoverage,
		UndefinedMetrics:            emptySliceMapIfNil(undefinedMetrics),
		UndefinedAuditEvents:        emptySliceMapIfNil(undefinedAuditEvents),
		AuditPoliciesBelowRetention: auditPoliciesBelowRetention,
	}
}

func RenderExecutionContractReport(contract ExecutionContract, audit ExecutionContractAudit) string {
	lines := []string{
		"# Execution Layer Technical Contract",
		"",
		fmt.Sprintf("- Contract ID: %s", contract.ContractID),
		fmt.Sprintf("- Version: %s", contract.Version),
		fmt.Sprintf("- Models: %d", len(contract.Models)),
		fmt.Sprintf("- APIs: %d", len(contract.APIs)),
		fmt.Sprintf("- Permissions: %d", len(contract.Permissions)),
		fmt.Sprintf("- Roles: %d", len(contract.Roles)),
		fmt.Sprintf("- Metrics: %d", len(contract.Metrics)),
		fmt.Sprintf("- Audit Policies: %d", len(contract.AuditPolicies)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore()),
		fmt.Sprintf("- Release Ready: %t", audit.ReleaseReady()),
		"",
		"## APIs",
		"",
	}
	if len(contract.APIs) == 0 {
		lines = append(lines, "- APIs: none")
	} else {
		for _, api := range contract.APIs {
			audits := joinOrNone(api.EmittedAudits)
			metrics := joinOrNone(api.EmittedMetrics)
			permission := firstNonEmpty(api.RequiredPermission, "none")
			lines = append(lines, fmt.Sprintf(
				"- %s %s: request=%s response=%s permission=%s audits=%s metrics=%s",
				api.Method,
				api.Path,
				firstNonEmpty(api.RequestModel, "none"),
				firstNonEmpty(api.ResponseModel, "none"),
				permission,
				audits,
				metrics,
			))
		}
	}
	lines = append(lines, "", "## Roles", "")
	if len(contract.Roles) == 0 {
		lines = append(lines, "- Roles: none")
	} else {
		for _, role := range contract.Roles {
			lines = append(lines, fmt.Sprintf(
				"- %s: personas=%s permissions=%s scopes=%s escalation=%s",
				role.Name,
				joinOrNone(role.Personas),
				joinOrNone(role.GrantedPermissions),
				joinOrNone(role.ScopeBindings),
				firstNonEmpty(role.EscalationTarget, "none"),
			))
		}
	}
	lines = append(lines,
		"",
		"## Audit",
		"",
		fmt.Sprintf("- Models missing required fields: %s", formatSliceMap(audit.ModelsMissingRequiredFields)),
		fmt.Sprintf("- APIs missing permissions: %s", joinOrNone(audit.APIsMissingPermissions)),
		fmt.Sprintf("- APIs missing audits: %s", joinOrNone(audit.APIsMissingAudits)),
		fmt.Sprintf("- APIs missing metrics: %s", joinOrNone(audit.APIsMissingMetrics)),
		fmt.Sprintf("- Undefined model refs: %s", formatSliceMap(audit.UndefinedModelRefs)),
		fmt.Sprintf("- Undefined permissions: %s", formatStringMap(audit.UndefinedPermissions)),
		fmt.Sprintf("- Missing roles: %s", joinOrNone(audit.MissingRoles)),
		fmt.Sprintf("- Roles missing personas: %s", joinOrNone(audit.RolesMissingPersonas)),
		fmt.Sprintf("- Roles missing scope bindings: %s", joinOrNone(audit.RolesMissingScopeBindings)),
		fmt.Sprintf("- Roles missing escalation targets: %s", joinOrNone(audit.RolesMissingEscalations)),
		fmt.Sprintf("- Roles missing permissions: %s", joinOrNone(audit.RolesMissingPermissions)),
		fmt.Sprintf("- Undefined role permissions: %s", formatSliceMap(audit.UndefinedRolePermissions)),
		fmt.Sprintf("- Permissions without roles: %s", joinOrNone(audit.PermissionsWithoutRoles)),
		fmt.Sprintf("- APIs without role coverage: %s", joinOrNone(audit.APIsWithoutRoleCoverage)),
		fmt.Sprintf("- Undefined metrics: %s", formatSliceMap(audit.UndefinedMetrics)),
		fmt.Sprintf("- Undefined audit events: %s", formatSliceMap(audit.UndefinedAuditEvents)),
		fmt.Sprintf("- Audit retention gaps: %s", joinOrNone(audit.AuditPoliciesBelowRetention)),
	)
	return strings.Join(lines, "\n")
}

func BuildOperationsAPIContract() ExecutionContract {
	return buildOperationsAPIContract("OPE-131", "v4.0-draft1")
}

func buildOperationsAPIContract(contractID string, version string) ExecutionContract {
	return ExecutionContract{
		ContractID: contractID,
		Version:    version,
		Models: []ExecutionModel{
			{
				Name:  "OperationsDashboardResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("period", "string"),
					requiredExecutionField("total_runs", "int"),
					requiredExecutionField("success_rate", "float"),
					requiredExecutionField("approval_queue_depth", "int"),
					requiredExecutionField("sla_breach_count", "int"),
					optionalExecutionField("top_blockers", "string[]"),
				},
			},
			{
				Name:  "RunDetailResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("run_id", "string"),
					requiredExecutionField("task_id", "string"),
					requiredExecutionField("status", "string"),
					requiredExecutionField("timeline_events", "RunDetailEvent[]"),
					requiredExecutionField("resources", "RunDetailResource[]"),
					requiredExecutionField("audit_count", "int"),
				},
			},
			{
				Name:  "RunReplayResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("run_id", "string"),
					requiredExecutionField("replay_available", "bool"),
					optionalExecutionField("replay_path", "string"),
					optionalExecutionField("benchmark_case_ids", "string[]"),
				},
			},
			{
				Name:  "QueueControlCenterResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("queue_depth", "int"),
					requiredExecutionField("queued_by_priority", "map<string,int>"),
					requiredExecutionField("queued_by_risk", "map<string,int>"),
					requiredExecutionField("waiting_approval_runs", "int"),
					optionalExecutionField("blocked_tasks", "string[]"),
				},
			},
			{
				Name:  "QueueActionRequest",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("actor", "string"),
					requiredExecutionField("reason", "string"),
				},
			},
			{
				Name:  "QueueActionResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("task_id", "string"),
					requiredExecutionField("action", "string"),
					requiredExecutionField("accepted", "bool"),
					requiredExecutionField("queue_depth", "int"),
				},
			},
			{
				Name:  "RunApprovalRequest",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("actor", "string"),
					requiredExecutionField("approval_token", "string"),
					requiredExecutionField("decision", "string"),
					optionalExecutionField("reason", "string"),
				},
			},
			{
				Name:  "RunApprovalResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("run_id", "string"),
					requiredExecutionField("status", "string"),
					requiredExecutionField("approved", "bool"),
					optionalExecutionField("required_follow_up", "string[]"),
				},
			},
			{
				Name:  "RiskOverviewResponse",
				Owner: "risk",
				Fields: []ExecutionField{
					requiredExecutionField("period", "string"),
					requiredExecutionField("high_risk_runs", "int"),
					requiredExecutionField("approval_required_runs", "int"),
					requiredExecutionField("risk_factors", "string[]"),
					requiredExecutionField("recommendation", "string"),
				},
			},
			{
				Name:  "SlaOverviewResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("period", "string"),
					requiredExecutionField("sla_target_minutes", "int"),
					requiredExecutionField("average_cycle_minutes", "float"),
					requiredExecutionField("sla_breach_count", "int"),
					requiredExecutionField("approval_queue_depth", "int"),
				},
			},
			{
				Name:  "RegressionCenterResponse",
				Owner: "operations",
				Fields: []ExecutionField{
					requiredExecutionField("baseline_version", "string"),
					requiredExecutionField("current_version", "string"),
					requiredExecutionField("regression_count", "int"),
					optionalExecutionField("improved_cases", "string[]"),
					optionalExecutionField("regressions", "RegressionFinding[]"),
				},
			},
			{
				Name:  "FlowCanvasResponse",
				Owner: "orchestration",
				Fields: []ExecutionField{
					requiredExecutionField("run_id", "string"),
					requiredExecutionField("collaboration_mode", "string"),
					requiredExecutionField("departments", "string[]"),
					optionalExecutionField("required_approvals", "string[]"),
					requiredExecutionField("billing_model", "string"),
					requiredExecutionField("recommendation", "string"),
				},
			},
			{
				Name:  "BillingEntitlementsResponse",
				Owner: "orchestration",
				Fields: []ExecutionField{
					requiredExecutionField("period", "string"),
					requiredExecutionField("tier", "string"),
					requiredExecutionField("billing_model_counts", "map<string,int>"),
					requiredExecutionField("upgrade_required_runs", "int"),
					requiredExecutionField("estimated_cost_usd", "float"),
				},
			},
			{
				Name:  "BillingRunChargeResponse",
				Owner: "orchestration",
				Fields: []ExecutionField{
					requiredExecutionField("run_id", "string"),
					requiredExecutionField("billing_model", "string"),
					requiredExecutionField("estimated_cost_usd", "float"),
					requiredExecutionField("overage_cost_usd", "float"),
					requiredExecutionField("upgrade_required", "bool"),
				},
			},
		},
		APIs: []ExecutionAPISpec{
			{Name: "get_operations_dashboard", Method: "GET", Path: "/operations/dashboard", ResponseModel: "OperationsDashboardResponse", RequiredPermission: "operations.dashboard.read", EmittedAudits: []string{"operations.dashboard.viewed"}, EmittedMetrics: []string{"operations.dashboard.requests", "operations.dashboard.latency.ms"}},
			{Name: "get_run_detail", Method: "GET", Path: "/operations/runs/{run_id}", ResponseModel: "RunDetailResponse", RequiredPermission: "operations.run.read", EmittedAudits: []string{"operations.run_detail.viewed"}, EmittedMetrics: []string{"operations.run_detail.requests", "operations.run_detail.latency.ms"}},
			{Name: "get_run_replay", Method: "GET", Path: "/operations/runs/{run_id}/replay", ResponseModel: "RunReplayResponse", RequiredPermission: "operations.run.read", EmittedAudits: []string{"operations.run_replay.viewed"}, EmittedMetrics: []string{"operations.run_replay.requests", "operations.run_replay.latency.ms"}},
			{Name: "get_queue_control_center", Method: "GET", Path: "/operations/queue/control-center", ResponseModel: "QueueControlCenterResponse", RequiredPermission: "operations.queue.read", EmittedAudits: []string{"operations.queue.viewed"}, EmittedMetrics: []string{"operations.queue.requests", "operations.queue.depth"}},
			{Name: "retry_queue_task", Method: "POST", Path: "/operations/queue/{task_id}/retry", RequestModel: "QueueActionRequest", ResponseModel: "QueueActionResponse", RequiredPermission: "operations.queue.act", EmittedAudits: []string{"operations.queue.retry.requested"}, EmittedMetrics: []string{"operations.queue.retry.requests", "operations.queue.depth"}},
			{Name: "approve_run_execution", Method: "POST", Path: "/operations/runs/{run_id}/approve", RequestModel: "RunApprovalRequest", ResponseModel: "RunApprovalResponse", RequiredPermission: "operations.run.approve", EmittedAudits: []string{"operations.run.approval.recorded"}, EmittedMetrics: []string{"operations.run.approval.requests", "operations.approval.queue.depth"}},
			{Name: "get_risk_overview", Method: "GET", Path: "/operations/risk/overview", ResponseModel: "RiskOverviewResponse", RequiredPermission: "operations.risk.read", EmittedAudits: []string{"operations.risk.viewed"}, EmittedMetrics: []string{"operations.risk.requests", "operations.risk.high_runs"}},
			{Name: "get_sla_overview", Method: "GET", Path: "/operations/sla/overview", ResponseModel: "SlaOverviewResponse", RequiredPermission: "operations.sla.read", EmittedAudits: []string{"operations.sla.viewed"}, EmittedMetrics: []string{"operations.sla.requests", "operations.sla.breaches"}},
			{Name: "get_regression_center", Method: "GET", Path: "/operations/regressions", ResponseModel: "RegressionCenterResponse", RequiredPermission: "operations.regression.read", EmittedAudits: []string{"operations.regression.viewed"}, EmittedMetrics: []string{"operations.regression.requests", "operations.regression.count"}},
			{Name: "get_flow_canvas", Method: "GET", Path: "/operations/flows/{run_id}", ResponseModel: "FlowCanvasResponse", RequiredPermission: "operations.flow.read", EmittedAudits: []string{"operations.flow.viewed"}, EmittedMetrics: []string{"operations.flow.requests", "operations.flow.handoff_count"}},
			{Name: "get_billing_entitlements", Method: "GET", Path: "/operations/billing/entitlements", ResponseModel: "BillingEntitlementsResponse", RequiredPermission: "operations.billing.read", EmittedAudits: []string{"operations.billing.viewed"}, EmittedMetrics: []string{"operations.billing.requests", "operations.billing.estimated_cost_usd"}},
			{Name: "get_billing_run_charge", Method: "GET", Path: "/operations/billing/runs/{run_id}", ResponseModel: "BillingRunChargeResponse", RequiredPermission: "operations.billing.read", EmittedAudits: []string{"operations.billing.run_charge.viewed"}, EmittedMetrics: []string{"operations.billing.run_charge.requests", "operations.billing.overage_cost_usd"}},
		},
		Permissions: []ExecutionPermission{
			{Name: "operations.dashboard.read", Resource: "operations-dashboard", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.run.read", Resource: "run-detail", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.queue.read", Resource: "queue-control-center", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.queue.act", Resource: "queue-control-center", Actions: []string{"retry", "escalate"}, Scopes: []string{"team"}},
			{Name: "operations.run.approve", Resource: "run-approval", Actions: []string{"approve"}, Scopes: []string{"workspace"}},
			{Name: "operations.risk.read", Resource: "risk-overview", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.sla.read", Resource: "sla-overview", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.regression.read", Resource: "regression-center", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.flow.read", Resource: "flow-canvas", Actions: []string{"read"}, Scopes: []string{"team", "workspace"}},
			{Name: "operations.billing.read", Resource: "billing-entitlements", Actions: []string{"read"}, Scopes: []string{"workspace"}},
		},
		Roles: []ExecutionRole{
			{Name: "eng-lead", Personas: []string{"Eng Lead"}, GrantedPermissions: []string{"operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.run.approve", "operations.risk.read", "operations.sla.read", "operations.regression.read"}, ScopeBindings: []string{"team", "workspace"}, EscalationTarget: "vp-eng"},
			{Name: "platform-admin", Personas: []string{"Platform Admin"}, GrantedPermissions: []string{"operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.flow.read", "operations.billing.read"}, ScopeBindings: []string{"workspace"}, EscalationTarget: "vp-eng"},
			{Name: "vp-eng", Personas: []string{"VP Eng"}, GrantedPermissions: []string{"operations.dashboard.read", "operations.run.read", "operations.run.approve", "operations.risk.read", "operations.sla.read", "operations.regression.read", "operations.billing.read"}, ScopeBindings: []string{"portfolio", "workspace"}, EscalationTarget: "none"},
			{Name: "cross-team-operator", Personas: []string{"Cross-Team Operator"}, GrantedPermissions: []string{"operations.dashboard.read", "operations.run.read", "operations.queue.read", "operations.queue.act", "operations.flow.read", "operations.billing.read"}, ScopeBindings: []string{"cross-team", "team", "workspace"}, EscalationTarget: "eng-lead"},
		},
		Metrics: []MetricDefinition{
			{Name: "operations.dashboard.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.dashboard.latency.ms", Unit: "ms", Owner: "operations"},
			{Name: "operations.run_detail.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.run_detail.latency.ms", Unit: "ms", Owner: "operations"},
			{Name: "operations.run_replay.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.run_replay.latency.ms", Unit: "ms", Owner: "operations"},
			{Name: "operations.queue.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.queue.depth", Unit: "count", Owner: "operations"},
			{Name: "operations.queue.retry.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.run.approval.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.approval.queue.depth", Unit: "count", Owner: "operations"},
			{Name: "operations.risk.requests", Unit: "count", Owner: "risk"},
			{Name: "operations.risk.high_runs", Unit: "count", Owner: "risk"},
			{Name: "operations.sla.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.sla.breaches", Unit: "count", Owner: "operations"},
			{Name: "operations.regression.requests", Unit: "count", Owner: "operations"},
			{Name: "operations.regression.count", Unit: "count", Owner: "operations"},
			{Name: "operations.flow.requests", Unit: "count", Owner: "orchestration"},
			{Name: "operations.flow.handoff_count", Unit: "count", Owner: "orchestration"},
			{Name: "operations.billing.requests", Unit: "count", Owner: "finance"},
			{Name: "operations.billing.estimated_cost_usd", Unit: "usd", Owner: "finance"},
			{Name: "operations.billing.run_charge.requests", Unit: "count", Owner: "finance"},
			{Name: "operations.billing.overage_cost_usd", Unit: "usd", Owner: "finance"},
		},
		AuditPolicies: []AuditPolicy{
			{EventType: "operations.dashboard.viewed", RequiredFields: []string{"actor", "period"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.run_detail.viewed", RequiredFields: []string{"actor", "run_id"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.run_replay.viewed", RequiredFields: []string{"actor", "run_id"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.queue.viewed", RequiredFields: []string{"actor", "queue_depth"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.queue.retry.requested", RequiredFields: []string{"actor", "task_id", "reason"}, RetentionDays: 180, Severity: "warning"},
			{EventType: "operations.run.approval.recorded", RequiredFields: []string{"actor", "run_id", "decision"}, RetentionDays: 365, Severity: "warning"},
			{EventType: "operations.risk.viewed", RequiredFields: []string{"actor", "period"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.sla.viewed", RequiredFields: []string{"actor", "period"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.regression.viewed", RequiredFields: []string{"actor", "current_version"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.flow.viewed", RequiredFields: []string{"actor", "run_id"}, RetentionDays: 180, Severity: "info"},
			{EventType: "operations.billing.viewed", RequiredFields: []string{"actor", "period", "tier"}, RetentionDays: 365, Severity: "info"},
			{EventType: "operations.billing.run_charge.viewed", RequiredFields: []string{"actor", "run_id", "billing_model"}, RetentionDays: 365, Severity: "info"},
		},
	}
}

func sortStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sort.Strings(values)
	return values
}

func emptySliceMapIfNil(value map[string][]string) map[string][]string {
	if len(value) == 0 {
		return nil
	}
	return value
}

func emptyStringMapIfNil(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	return value
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func formatSliceMap(values map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func formatStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, values[key]))
	}
	return strings.Join(parts, ", ")
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
