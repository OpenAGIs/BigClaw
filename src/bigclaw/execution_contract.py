from dataclasses import dataclass, field
from typing import Dict, List, Optional


@dataclass(frozen=True)
class ExecutionField:
    name: str
    field_type: str
    required: bool = True
    description: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "field_type": self.field_type,
            "required": self.required,
            "description": self.description,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionField":
        return cls(
            name=str(data["name"]),
            field_type=str(data["field_type"]),
            required=bool(data.get("required", True)),
            description=str(data.get("description", "")),
        )


@dataclass
class ExecutionModel:
    name: str
    fields: List[ExecutionField] = field(default_factory=list)
    owner: str = ""

    @property
    def required_fields(self) -> List[str]:
        return [field.name for field in self.fields if field.required]

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "fields": [field.to_dict() for field in self.fields],
            "owner": self.owner,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionModel":
        return cls(
            name=str(data["name"]),
            fields=[ExecutionField.from_dict(field) for field in data.get("fields", [])],
            owner=str(data.get("owner", "")),
        )


@dataclass
class ExecutionApiSpec:
    name: str
    method: str
    path: str
    request_model: str
    response_model: str
    required_permission: str
    emitted_audits: List[str] = field(default_factory=list)
    emitted_metrics: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "method": self.method,
            "path": self.path,
            "request_model": self.request_model,
            "response_model": self.response_model,
            "required_permission": self.required_permission,
            "emitted_audits": list(self.emitted_audits),
            "emitted_metrics": list(self.emitted_metrics),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionApiSpec":
        return cls(
            name=str(data["name"]),
            method=str(data["method"]),
            path=str(data["path"]),
            request_model=str(data.get("request_model", "")),
            response_model=str(data.get("response_model", "")),
            required_permission=str(data.get("required_permission", "")),
            emitted_audits=[str(item) for item in data.get("emitted_audits", [])],
            emitted_metrics=[str(item) for item in data.get("emitted_metrics", [])],
        )


@dataclass(frozen=True)
class ExecutionPermission:
    name: str
    resource: str
    actions: List[str] = field(default_factory=list)
    scopes: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "resource": self.resource,
            "actions": list(self.actions),
            "scopes": list(self.scopes),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionPermission":
        return cls(
            name=str(data["name"]),
            resource=str(data.get("resource", "")),
            actions=[str(item) for item in data.get("actions", [])],
            scopes=[str(item) for item in data.get("scopes", [])],
        )


@dataclass(frozen=True)
class ExecutionRole:
    name: str
    personas: List[str] = field(default_factory=list)
    granted_permissions: List[str] = field(default_factory=list)
    scope_bindings: List[str] = field(default_factory=list)
    escalation_target: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "personas": list(self.personas),
            "granted_permissions": list(self.granted_permissions),
            "scope_bindings": list(self.scope_bindings),
            "escalation_target": self.escalation_target,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionRole":
        return cls(
            name=str(data["name"]),
            personas=[str(item) for item in data.get("personas", [])],
            granted_permissions=[str(item) for item in data.get("granted_permissions", [])],
            scope_bindings=[str(item) for item in data.get("scope_bindings", [])],
            escalation_target=str(data.get("escalation_target", "")),
        )


@dataclass
class PermissionCheckResult:
    allowed: bool
    granted_permissions: List[str] = field(default_factory=list)
    missing_permissions: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "allowed": self.allowed,
            "granted_permissions": list(self.granted_permissions),
            "missing_permissions": list(self.missing_permissions),
        }


class ExecutionPermissionMatrix:
    def __init__(self, permissions: List[ExecutionPermission], roles: Optional[List[ExecutionRole]] = None) -> None:
        self.permissions = {permission.name: permission for permission in permissions}
        self.roles = {role.name: role for role in roles or []}

    def evaluate(self, required_permissions: List[str], granted_permissions: List[str]) -> PermissionCheckResult:
        granted_set = {permission for permission in granted_permissions if permission in self.permissions}
        missing = [permission for permission in required_permissions if permission not in granted_set]
        return PermissionCheckResult(
            allowed=not missing,
            granted_permissions=sorted(granted_set),
            missing_permissions=missing,
        )

    def evaluate_roles(self, required_permissions: List[str], actor_roles: List[str]) -> PermissionCheckResult:
        granted_permissions = {
            permission
            for role_name in actor_roles
            for permission in self.roles.get(role_name, ExecutionRole(name=role_name)).granted_permissions
            if permission in self.permissions
        }
        return self.evaluate(required_permissions=required_permissions, granted_permissions=sorted(granted_permissions))


@dataclass(frozen=True)
class MetricDefinition:
    name: str
    unit: str
    owner: str
    description: str = ""

    def to_dict(self) -> Dict[str, str]:
        return {
            "name": self.name,
            "unit": self.unit,
            "owner": self.owner,
            "description": self.description,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "MetricDefinition":
        return cls(
            name=str(data["name"]),
            unit=str(data.get("unit", "")),
            owner=str(data.get("owner", "")),
            description=str(data.get("description", "")),
        )


@dataclass(frozen=True)
class AuditPolicy:
    event_type: str
    required_fields: List[str] = field(default_factory=list)
    retention_days: int = 30
    severity: str = "info"

    def to_dict(self) -> Dict[str, object]:
        return {
            "event_type": self.event_type,
            "required_fields": list(self.required_fields),
            "retention_days": self.retention_days,
            "severity": self.severity,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AuditPolicy":
        return cls(
            event_type=str(data["event_type"]),
            required_fields=[str(item) for item in data.get("required_fields", [])],
            retention_days=int(data.get("retention_days", 30)),
            severity=str(data.get("severity", "info")),
        )


@dataclass
class ExecutionContract:
    contract_id: str
    version: str
    models: List[ExecutionModel] = field(default_factory=list)
    apis: List[ExecutionApiSpec] = field(default_factory=list)
    permissions: List[ExecutionPermission] = field(default_factory=list)
    roles: List[ExecutionRole] = field(default_factory=list)
    metrics: List[MetricDefinition] = field(default_factory=list)
    audit_policies: List[AuditPolicy] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "models": [model.to_dict() for model in self.models],
            "apis": [api.to_dict() for api in self.apis],
            "permissions": [permission.to_dict() for permission in self.permissions],
            "roles": [role.to_dict() for role in self.roles],
            "metrics": [metric.to_dict() for metric in self.metrics],
            "audit_policies": [policy.to_dict() for policy in self.audit_policies],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionContract":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            models=[ExecutionModel.from_dict(model) for model in data.get("models", [])],
            apis=[ExecutionApiSpec.from_dict(api) for api in data.get("apis", [])],
            permissions=[ExecutionPermission.from_dict(permission) for permission in data.get("permissions", [])],
            roles=[ExecutionRole.from_dict(role) for role in data.get("roles", [])],
            metrics=[MetricDefinition.from_dict(metric) for metric in data.get("metrics", [])],
            audit_policies=[AuditPolicy.from_dict(policy) for policy in data.get("audit_policies", [])],
        )


@dataclass
class ExecutionContractAudit:
    contract_id: str
    version: str
    models_missing_required_fields: Dict[str, List[str]] = field(default_factory=dict)
    apis_missing_permissions: List[str] = field(default_factory=list)
    apis_missing_audits: List[str] = field(default_factory=list)
    apis_missing_metrics: List[str] = field(default_factory=list)
    undefined_model_refs: Dict[str, List[str]] = field(default_factory=dict)
    undefined_permissions: Dict[str, str] = field(default_factory=dict)
    missing_roles: List[str] = field(default_factory=list)
    roles_missing_personas: List[str] = field(default_factory=list)
    roles_missing_scope_bindings: List[str] = field(default_factory=list)
    roles_missing_escalation_targets: List[str] = field(default_factory=list)
    roles_missing_permissions: List[str] = field(default_factory=list)
    undefined_role_permissions: Dict[str, List[str]] = field(default_factory=dict)
    permissions_without_roles: List[str] = field(default_factory=list)
    apis_without_role_coverage: List[str] = field(default_factory=list)
    undefined_metrics: Dict[str, List[str]] = field(default_factory=dict)
    undefined_audit_events: Dict[str, List[str]] = field(default_factory=dict)
    audit_policies_below_retention: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        api_count = max(1, len(self.apis_missing_permissions) + len(self.apis_missing_audits) + len(self.apis_missing_metrics))
        issue_count = (
            len(self.models_missing_required_fields)
            + len(self.apis_missing_permissions)
            + len(self.apis_missing_audits)
            + len(self.apis_missing_metrics)
            + len(self.undefined_model_refs)
            + len(self.undefined_permissions)
            + len(self.missing_roles)
            + len(self.roles_missing_personas)
            + len(self.roles_missing_scope_bindings)
            + len(self.roles_missing_escalation_targets)
            + len(self.roles_missing_permissions)
            + len(self.undefined_role_permissions)
            + len(self.permissions_without_roles)
            + len(self.apis_without_role_coverage)
            + len(self.undefined_metrics)
            + len(self.undefined_audit_events)
            + len(self.audit_policies_below_retention)
        )
        if issue_count == 0:
            return 100.0
        penalty = min(100.0, issue_count * (100.0 / api_count))
        return round(max(0.0, 100.0 - penalty), 1)

    @property
    def release_ready(self) -> bool:
        return self.readiness_score == 100.0

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "models_missing_required_fields": {
                name: list(fields) for name, fields in self.models_missing_required_fields.items()
            },
            "apis_missing_permissions": list(self.apis_missing_permissions),
            "apis_missing_audits": list(self.apis_missing_audits),
            "apis_missing_metrics": list(self.apis_missing_metrics),
            "undefined_model_refs": {name: list(values) for name, values in self.undefined_model_refs.items()},
            "undefined_permissions": dict(self.undefined_permissions),
            "missing_roles": list(self.missing_roles),
            "roles_missing_personas": list(self.roles_missing_personas),
            "roles_missing_scope_bindings": list(self.roles_missing_scope_bindings),
            "roles_missing_escalation_targets": list(self.roles_missing_escalation_targets),
            "roles_missing_permissions": list(self.roles_missing_permissions),
            "undefined_role_permissions": {name: list(values) for name, values in self.undefined_role_permissions.items()},
            "permissions_without_roles": list(self.permissions_without_roles),
            "apis_without_role_coverage": list(self.apis_without_role_coverage),
            "undefined_metrics": {name: list(values) for name, values in self.undefined_metrics.items()},
            "undefined_audit_events": {name: list(values) for name, values in self.undefined_audit_events.items()},
            "audit_policies_below_retention": list(self.audit_policies_below_retention),
            "readiness_score": self.readiness_score,
            "release_ready": self.release_ready,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ExecutionContractAudit":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            models_missing_required_fields={
                str(name): [str(field) for field in fields]
                for name, fields in dict(data.get("models_missing_required_fields", {})).items()
            },
            apis_missing_permissions=[str(item) for item in data.get("apis_missing_permissions", [])],
            apis_missing_audits=[str(item) for item in data.get("apis_missing_audits", [])],
            apis_missing_metrics=[str(item) for item in data.get("apis_missing_metrics", [])],
            undefined_model_refs={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_model_refs", {})).items()
            },
            undefined_permissions={str(name): str(value) for name, value in dict(data.get("undefined_permissions", {})).items()},
            missing_roles=[str(item) for item in data.get("missing_roles", [])],
            roles_missing_personas=[str(item) for item in data.get("roles_missing_personas", [])],
            roles_missing_scope_bindings=[str(item) for item in data.get("roles_missing_scope_bindings", [])],
            roles_missing_escalation_targets=[str(item) for item in data.get("roles_missing_escalation_targets", [])],
            roles_missing_permissions=[str(item) for item in data.get("roles_missing_permissions", [])],
            undefined_role_permissions={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_role_permissions", {})).items()
            },
            permissions_without_roles=[str(item) for item in data.get("permissions_without_roles", [])],
            apis_without_role_coverage=[str(item) for item in data.get("apis_without_role_coverage", [])],
            undefined_metrics={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_metrics", {})).items()
            },
            undefined_audit_events={
                str(name): [str(value) for value in values]
                for name, values in dict(data.get("undefined_audit_events", {})).items()
            },
            audit_policies_below_retention=[str(item) for item in data.get("audit_policies_below_retention", [])],
        )


class ExecutionContractLibrary:
    REQUIRED_MODEL_FIELDS = {
        "ExecutionRequest": ["task_id", "actor", "requested_tools"],
        "ExecutionResponse": ["run_id", "status", "sandbox_profile"],
    }
    REQUIRED_ROLES = ["eng-lead", "platform-admin", "vp-eng", "cross-team-operator"]

    def audit(self, contract: ExecutionContract) -> ExecutionContractAudit:
        model_names = {model.name for model in contract.models}
        permission_names = {permission.name for permission in contract.permissions}
        metric_names = {metric.name for metric in contract.metrics}
        audit_events = {policy.event_type for policy in contract.audit_policies}
        role_names = {role.name for role in contract.roles}

        models_missing_required_fields: Dict[str, List[str]] = {}
        for model in contract.models:
            expected_fields = self.REQUIRED_MODEL_FIELDS.get(model.name, [])
            missing = [field for field in expected_fields if field not in model.required_fields]
            if missing:
                models_missing_required_fields[model.name] = missing

        undefined_model_refs: Dict[str, List[str]] = {}
        undefined_permissions: Dict[str, str] = {}
        missing_roles = sorted(role for role in self.REQUIRED_ROLES if role not in role_names)
        roles_missing_personas: List[str] = []
        roles_missing_scope_bindings: List[str] = []
        roles_missing_escalation_targets: List[str] = []
        roles_missing_permissions: List[str] = []
        undefined_role_permissions: Dict[str, List[str]] = {}
        permissions_granted_by_roles: set[str] = set()
        apis_without_role_coverage: List[str] = []
        undefined_metrics: Dict[str, List[str]] = {}
        undefined_audit_events: Dict[str, List[str]] = {}
        apis_missing_permissions: List[str] = []
        apis_missing_audits: List[str] = []
        apis_missing_metrics: List[str] = []

        for api in contract.apis:
            missing_models = [
                model_name
                for model_name in [api.request_model, api.response_model]
                if model_name and model_name not in model_names
            ]
            if missing_models:
                undefined_model_refs[api.name] = missing_models

            if not api.required_permission:
                apis_missing_permissions.append(api.name)
            elif api.required_permission not in permission_names:
                undefined_permissions[api.name] = api.required_permission

            if not api.emitted_audits:
                apis_missing_audits.append(api.name)
            else:
                missing_events = [event for event in api.emitted_audits if event not in audit_events]
                if missing_events:
                    undefined_audit_events[api.name] = missing_events

            if not api.emitted_metrics:
                apis_missing_metrics.append(api.name)
            else:
                missing_metric_defs = [metric for metric in api.emitted_metrics if metric not in metric_names]
                if missing_metric_defs:
                    undefined_metrics[api.name] = missing_metric_defs

        for role in contract.roles:
            if not role.personas:
                roles_missing_personas.append(role.name)
            if not role.scope_bindings:
                roles_missing_scope_bindings.append(role.name)
            if not role.escalation_target.strip():
                roles_missing_escalation_targets.append(role.name)
            if not role.granted_permissions:
                roles_missing_permissions.append(role.name)
                continue
            missing_permissions = [permission for permission in role.granted_permissions if permission not in permission_names]
            if missing_permissions:
                undefined_role_permissions[role.name] = missing_permissions
            permissions_granted_by_roles.update(
                permission for permission in role.granted_permissions if permission in permission_names
            )

        for api in contract.apis:
            if api.required_permission and api.required_permission in permission_names and api.required_permission not in permissions_granted_by_roles:
                apis_without_role_coverage.append(api.name)

        permissions_without_roles = sorted(permission for permission in permission_names if permission not in permissions_granted_by_roles)

        audit_policies_below_retention = sorted(
            policy.event_type for policy in contract.audit_policies if policy.retention_days < 30
        )

        return ExecutionContractAudit(
            contract_id=contract.contract_id,
            version=contract.version,
            models_missing_required_fields=models_missing_required_fields,
            apis_missing_permissions=sorted(apis_missing_permissions),
            apis_missing_audits=sorted(apis_missing_audits),
            apis_missing_metrics=sorted(apis_missing_metrics),
            undefined_model_refs=undefined_model_refs,
            undefined_permissions=undefined_permissions,
            missing_roles=missing_roles,
            roles_missing_personas=sorted(roles_missing_personas),
            roles_missing_scope_bindings=sorted(roles_missing_scope_bindings),
            roles_missing_escalation_targets=sorted(roles_missing_escalation_targets),
            roles_missing_permissions=sorted(roles_missing_permissions),
            undefined_role_permissions=undefined_role_permissions,
            permissions_without_roles=permissions_without_roles,
            apis_without_role_coverage=sorted(apis_without_role_coverage),
            undefined_metrics=undefined_metrics,
            undefined_audit_events=undefined_audit_events,
            audit_policies_below_retention=audit_policies_below_retention,
        )


def render_execution_contract_report(contract: ExecutionContract, audit: ExecutionContractAudit) -> str:
    lines = [
        "# Execution Layer Technical Contract",
        "",
        f"- Contract ID: {contract.contract_id}",
        f"- Version: {contract.version}",
        f"- Models: {len(contract.models)}",
        f"- APIs: {len(contract.apis)}",
        f"- Permissions: {len(contract.permissions)}",
        f"- Roles: {len(contract.roles)}",
        f"- Metrics: {len(contract.metrics)}",
        f"- Audit Policies: {len(contract.audit_policies)}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## APIs",
        "",
    ]

    if contract.apis:
        for api in contract.apis:
            audits = ", ".join(api.emitted_audits) if api.emitted_audits else "none"
            metrics = ", ".join(api.emitted_metrics) if api.emitted_metrics else "none"
            permission = api.required_permission or "none"
            lines.append(
                f"- {api.method} {api.path}: request={api.request_model or 'none'} "
                f"response={api.response_model or 'none'} permission={permission} audits={audits} metrics={metrics}"
            )
    else:
        lines.append("- APIs: none")

    lines.extend(["", "## Roles", ""])
    if contract.roles:
        for role in contract.roles:
            personas = ", ".join(role.personas) if role.personas else "none"
            permissions = ", ".join(role.granted_permissions) if role.granted_permissions else "none"
            scopes = ", ".join(role.scope_bindings) if role.scope_bindings else "none"
            escalation_target = role.escalation_target or "none"
            lines.append(
                f"- {role.name}: personas={personas} permissions={permissions} scopes={scopes} escalation={escalation_target}"
            )
    else:
        lines.append("- Roles: none")

    lines.extend(
        [
            "",
            "## Audit",
            "",
            f"- Models missing required fields: {', '.join(f'{name}={fields}' for name, fields in sorted(audit.models_missing_required_fields.items())) if audit.models_missing_required_fields else 'none'}",
            f"- APIs missing permissions: {', '.join(audit.apis_missing_permissions) if audit.apis_missing_permissions else 'none'}",
            f"- APIs missing audits: {', '.join(audit.apis_missing_audits) if audit.apis_missing_audits else 'none'}",
            f"- APIs missing metrics: {', '.join(audit.apis_missing_metrics) if audit.apis_missing_metrics else 'none'}",
            f"- Undefined model refs: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_model_refs.items())) if audit.undefined_model_refs else 'none'}",
            f"- Undefined permissions: {', '.join(f'{name}={value}' for name, value in sorted(audit.undefined_permissions.items())) if audit.undefined_permissions else 'none'}",
            f"- Missing roles: {', '.join(audit.missing_roles) if audit.missing_roles else 'none'}",
            f"- Roles missing personas: {', '.join(audit.roles_missing_personas) if audit.roles_missing_personas else 'none'}",
            f"- Roles missing scope bindings: {', '.join(audit.roles_missing_scope_bindings) if audit.roles_missing_scope_bindings else 'none'}",
            f"- Roles missing escalation targets: {', '.join(audit.roles_missing_escalation_targets) if audit.roles_missing_escalation_targets else 'none'}",
            f"- Roles missing permissions: {', '.join(audit.roles_missing_permissions) if audit.roles_missing_permissions else 'none'}",
            f"- Undefined role permissions: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_role_permissions.items())) if audit.undefined_role_permissions else 'none'}",
            f"- Permissions without roles: {', '.join(audit.permissions_without_roles) if audit.permissions_without_roles else 'none'}",
            f"- APIs without role coverage: {', '.join(audit.apis_without_role_coverage) if audit.apis_without_role_coverage else 'none'}",
            f"- Undefined metrics: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_metrics.items())) if audit.undefined_metrics else 'none'}",
            f"- Undefined audit events: {', '.join(f'{name}={values}' for name, values in sorted(audit.undefined_audit_events.items())) if audit.undefined_audit_events else 'none'}",
            f"- Audit retention gaps: {', '.join(audit.audit_policies_below_retention) if audit.audit_policies_below_retention else 'none'}",
        ]
    )
    return "\n".join(lines)


def build_operations_api_contract(contract_id: str = "OPE-131", version: str = "v4.0-draft1") -> ExecutionContract:
    return ExecutionContract(
        contract_id=contract_id,
        version=version,
        models=[
            ExecutionModel(
                name="OperationsDashboardResponse",
                owner="operations",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("total_runs", "int"),
                    ExecutionField("success_rate", "float"),
                    ExecutionField("approval_queue_depth", "int"),
                    ExecutionField("sla_breach_count", "int"),
                    ExecutionField("top_blockers", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="RunDetailResponse",
                owner="operations",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("task_id", "string"),
                    ExecutionField("status", "string"),
                    ExecutionField("timeline_events", "RunDetailEvent[]"),
                    ExecutionField("resources", "RunDetailResource[]"),
                    ExecutionField("audit_count", "int"),
                ],
            ),
            ExecutionModel(
                name="RunReplayResponse",
                owner="operations",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("replay_available", "bool"),
                    ExecutionField("replay_path", "string", required=False),
                    ExecutionField("benchmark_case_ids", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="QueueControlCenterResponse",
                owner="operations",
                fields=[
                    ExecutionField("queue_depth", "int"),
                    ExecutionField("queued_by_priority", "map<string,int>"),
                    ExecutionField("queued_by_risk", "map<string,int>"),
                    ExecutionField("execution_media", "map<string,int>"),
                    ExecutionField("waiting_approval_runs", "int"),
                    ExecutionField("blocked_tasks", "string[]", required=False),
                    ExecutionField("bulk_retry_tasks", "string[]", required=False),
                    ExecutionField("bulk_retry_blockers", "map<string,string>", required=False),
                    ExecutionField("failure_attribution", "map<string,string[]>", required=False),
                    ExecutionField("failure_attribution_counts", "map<string,int>", required=False),
                    ExecutionField("manual_takeover_tasks", "string[]", required=False),
                    ExecutionField("manual_takeover_reasons", "map<string,string>", required=False),
                ],
            ),
            ExecutionModel(
                name="QueueActionRequest",
                owner="operations",
                fields=[
                    ExecutionField("actor", "string"),
                    ExecutionField("reason", "string"),
                ],
            ),
            ExecutionModel(
                name="QueueActionResponse",
                owner="operations",
                fields=[
                    ExecutionField("task_id", "string"),
                    ExecutionField("action", "string"),
                    ExecutionField("accepted", "bool"),
                    ExecutionField("queue_depth", "int"),
                ],
            ),
            ExecutionModel(
                name="QueueBulkActionRequest",
                owner="operations",
                fields=[
                    ExecutionField("actor", "string"),
                    ExecutionField("task_ids", "string[]"),
                    ExecutionField("reason", "string"),
                ],
            ),
            ExecutionModel(
                name="QueueBulkActionResponse",
                owner="operations",
                fields=[
                    ExecutionField("action", "string"),
                    ExecutionField("accepted", "bool"),
                    ExecutionField("requested_task_ids", "string[]"),
                    ExecutionField("accepted_task_ids", "string[]"),
                    ExecutionField("rejected_task_ids", "map<string,string>", required=False),
                    ExecutionField("queue_depth", "int"),
                ],
            ),
            ExecutionModel(
                name="RunApprovalRequest",
                owner="operations",
                fields=[
                    ExecutionField("actor", "string"),
                    ExecutionField("approval_token", "string"),
                    ExecutionField("decision", "string"),
                    ExecutionField("reason", "string", required=False),
                ],
            ),
            ExecutionModel(
                name="RunApprovalResponse",
                owner="operations",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("status", "string"),
                    ExecutionField("approved", "bool"),
                    ExecutionField("required_follow_up", "string[]", required=False),
                ],
            ),
            ExecutionModel(
                name="RiskOverviewResponse",
                owner="risk",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("high_risk_runs", "int"),
                    ExecutionField("approval_required_runs", "int"),
                    ExecutionField("risk_factors", "string[]"),
                    ExecutionField("recommendation", "string"),
                ],
            ),
            ExecutionModel(
                name="SlaOverviewResponse",
                owner="operations",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("sla_target_minutes", "int"),
                    ExecutionField("average_cycle_minutes", "float"),
                    ExecutionField("sla_breach_count", "int"),
                    ExecutionField("approval_queue_depth", "int"),
                ],
            ),
            ExecutionModel(
                name="RegressionCenterResponse",
                owner="operations",
                fields=[
                    ExecutionField("baseline_version", "string"),
                    ExecutionField("current_version", "string"),
                    ExecutionField("regression_count", "int"),
                    ExecutionField("improved_cases", "string[]", required=False),
                    ExecutionField("regressions", "RegressionFinding[]", required=False),
                ],
            ),
            ExecutionModel(
                name="FlowCanvasResponse",
                owner="orchestration",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("collaboration_mode", "string"),
                    ExecutionField("departments", "string[]"),
                    ExecutionField("required_approvals", "string[]", required=False),
                    ExecutionField("billing_model", "string"),
                    ExecutionField("recommendation", "string"),
                ],
            ),
            ExecutionModel(
                name="BillingEntitlementsResponse",
                owner="orchestration",
                fields=[
                    ExecutionField("period", "string"),
                    ExecutionField("tier", "string"),
                    ExecutionField("billing_model_counts", "map<string,int>"),
                    ExecutionField("upgrade_required_runs", "int"),
                    ExecutionField("estimated_cost_usd", "float"),
                ],
            ),
            ExecutionModel(
                name="BillingRunChargeResponse",
                owner="orchestration",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("billing_model", "string"),
                    ExecutionField("estimated_cost_usd", "float"),
                    ExecutionField("overage_cost_usd", "float"),
                    ExecutionField("upgrade_required", "bool"),
                ],
            ),
        ],
        apis=[
            ExecutionApiSpec(
                name="get_operations_dashboard",
                method="GET",
                path="/operations/dashboard",
                request_model="",
                response_model="OperationsDashboardResponse",
                required_permission="operations.dashboard.read",
                emitted_audits=["operations.dashboard.viewed"],
                emitted_metrics=["operations.dashboard.requests", "operations.dashboard.latency.ms"],
            ),
            ExecutionApiSpec(
                name="get_run_detail",
                method="GET",
                path="/operations/runs/{run_id}",
                request_model="",
                response_model="RunDetailResponse",
                required_permission="operations.run.read",
                emitted_audits=["operations.run_detail.viewed"],
                emitted_metrics=["operations.run_detail.requests", "operations.run_detail.latency.ms"],
            ),
            ExecutionApiSpec(
                name="get_run_replay",
                method="GET",
                path="/operations/runs/{run_id}/replay",
                request_model="",
                response_model="RunReplayResponse",
                required_permission="operations.run.read",
                emitted_audits=["operations.run_replay.viewed"],
                emitted_metrics=["operations.run_replay.requests", "operations.run_replay.latency.ms"],
            ),
            ExecutionApiSpec(
                name="get_queue_control_center",
                method="GET",
                path="/operations/queue/control-center",
                request_model="",
                response_model="QueueControlCenterResponse",
                required_permission="operations.queue.read",
                emitted_audits=["operations.queue.viewed"],
                emitted_metrics=["operations.queue.requests", "operations.queue.depth"],
            ),
            ExecutionApiSpec(
                name="retry_queue_task",
                method="POST",
                path="/operations/queue/{task_id}/retry",
                request_model="QueueActionRequest",
                response_model="QueueActionResponse",
                required_permission="operations.queue.act",
                emitted_audits=["operations.queue.retry.requested"],
                emitted_metrics=["operations.queue.retry.requests", "operations.queue.depth"],
            ),
            ExecutionApiSpec(
                name="bulk_retry_queue_tasks",
                method="POST",
                path="/operations/queue/bulk-retry",
                request_model="QueueBulkActionRequest",
                response_model="QueueBulkActionResponse",
                required_permission="operations.queue.act",
                emitted_audits=["operations.queue.bulk_retry.requested"],
                emitted_metrics=["operations.queue.bulk_retry.requests", "operations.queue.depth"],
            ),
            ExecutionApiSpec(
                name="request_manual_queue_takeover",
                method="POST",
                path="/operations/queue/{task_id}/manual-takeover",
                request_model="QueueActionRequest",
                response_model="QueueActionResponse",
                required_permission="operations.queue.act",
                emitted_audits=["operations.queue.manual_takeover.requested"],
                emitted_metrics=["operations.queue.manual_takeover.requests", "operations.queue.depth"],
            ),
            ExecutionApiSpec(
                name="approve_run_execution",
                method="POST",
                path="/operations/runs/{run_id}/approve",
                request_model="RunApprovalRequest",
                response_model="RunApprovalResponse",
                required_permission="operations.run.approve",
                emitted_audits=["operations.run.approval.recorded"],
                emitted_metrics=["operations.run.approval.requests", "operations.approval.queue.depth"],
            ),
            ExecutionApiSpec(
                name="get_risk_overview",
                method="GET",
                path="/operations/risk/overview",
                request_model="",
                response_model="RiskOverviewResponse",
                required_permission="operations.risk.read",
                emitted_audits=["operations.risk.viewed"],
                emitted_metrics=["operations.risk.requests", "operations.risk.high_runs"],
            ),
            ExecutionApiSpec(
                name="get_sla_overview",
                method="GET",
                path="/operations/sla/overview",
                request_model="",
                response_model="SlaOverviewResponse",
                required_permission="operations.sla.read",
                emitted_audits=["operations.sla.viewed"],
                emitted_metrics=["operations.sla.requests", "operations.sla.breaches"],
            ),
            ExecutionApiSpec(
                name="get_regression_center",
                method="GET",
                path="/operations/regressions",
                request_model="",
                response_model="RegressionCenterResponse",
                required_permission="operations.regression.read",
                emitted_audits=["operations.regression.viewed"],
                emitted_metrics=["operations.regression.requests", "operations.regression.count"],
            ),
            ExecutionApiSpec(
                name="get_flow_canvas",
                method="GET",
                path="/operations/flows/{run_id}",
                request_model="",
                response_model="FlowCanvasResponse",
                required_permission="operations.flow.read",
                emitted_audits=["operations.flow.viewed"],
                emitted_metrics=["operations.flow.requests", "operations.flow.handoff_count"],
            ),
            ExecutionApiSpec(
                name="get_billing_entitlements",
                method="GET",
                path="/operations/billing/entitlements",
                request_model="",
                response_model="BillingEntitlementsResponse",
                required_permission="operations.billing.read",
                emitted_audits=["operations.billing.viewed"],
                emitted_metrics=["operations.billing.requests", "operations.billing.estimated_cost_usd"],
            ),
            ExecutionApiSpec(
                name="get_billing_run_charge",
                method="GET",
                path="/operations/billing/runs/{run_id}",
                request_model="",
                response_model="BillingRunChargeResponse",
                required_permission="operations.billing.read",
                emitted_audits=["operations.billing.run_charge.viewed"],
                emitted_metrics=["operations.billing.run_charge.requests", "operations.billing.overage_cost_usd"],
            ),
        ],
        permissions=[
            ExecutionPermission("operations.dashboard.read", "operations-dashboard", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.run.read", "run-detail", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.queue.read", "queue-control-center", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission(
                "operations.queue.act",
                "queue-control-center",
                actions=["retry", "retry-bulk", "escalate", "manual-takeover"],
                scopes=["team"],
            ),
            ExecutionPermission("operations.run.approve", "run-approval", actions=["approve"], scopes=["workspace"]),
            ExecutionPermission("operations.risk.read", "risk-overview", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.sla.read", "sla-overview", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.regression.read", "regression-center", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.flow.read", "flow-canvas", actions=["read"], scopes=["team", "workspace"]),
            ExecutionPermission("operations.billing.read", "billing-entitlements", actions=["read"], scopes=["workspace"]),
        ],
        roles=[
            ExecutionRole(
                name="eng-lead",
                personas=["Eng Lead"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.queue.read",
                    "operations.run.approve",
                    "operations.risk.read",
                    "operations.sla.read",
                    "operations.regression.read",
                ],
                scope_bindings=["team", "workspace"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="platform-admin",
                personas=["Platform Admin"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.queue.read",
                    "operations.queue.act",
                    "operations.risk.read",
                    "operations.sla.read",
                    "operations.regression.read",
                    "operations.flow.read",
                    "operations.billing.read",
                ],
                scope_bindings=["workspace"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="vp-eng",
                personas=["VP Eng"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.run.approve",
                    "operations.risk.read",
                    "operations.sla.read",
                    "operations.regression.read",
                    "operations.billing.read",
                ],
                scope_bindings=["portfolio", "workspace"],
                escalation_target="none",
            ),
            ExecutionRole(
                name="cross-team-operator",
                personas=["Cross-Team Operator"],
                granted_permissions=[
                    "operations.dashboard.read",
                    "operations.run.read",
                    "operations.queue.read",
                    "operations.queue.act",
                    "operations.flow.read",
                    "operations.billing.read",
                ],
                scope_bindings=["cross-team", "team", "workspace"],
                escalation_target="eng-lead",
            ),
        ],
        metrics=[
            MetricDefinition("operations.dashboard.requests", "count", owner="operations"),
            MetricDefinition("operations.dashboard.latency.ms", "ms", owner="operations"),
            MetricDefinition("operations.run_detail.requests", "count", owner="operations"),
            MetricDefinition("operations.run_detail.latency.ms", "ms", owner="operations"),
            MetricDefinition("operations.run_replay.requests", "count", owner="operations"),
            MetricDefinition("operations.run_replay.latency.ms", "ms", owner="operations"),
            MetricDefinition("operations.queue.requests", "count", owner="operations"),
            MetricDefinition("operations.queue.depth", "count", owner="operations"),
            MetricDefinition("operations.queue.retry.requests", "count", owner="operations"),
            MetricDefinition("operations.queue.bulk_retry.requests", "count", owner="operations"),
            MetricDefinition("operations.queue.manual_takeover.requests", "count", owner="operations"),
            MetricDefinition("operations.run.approval.requests", "count", owner="operations"),
            MetricDefinition("operations.approval.queue.depth", "count", owner="operations"),
            MetricDefinition("operations.risk.requests", "count", owner="risk"),
            MetricDefinition("operations.risk.high_runs", "count", owner="risk"),
            MetricDefinition("operations.sla.requests", "count", owner="operations"),
            MetricDefinition("operations.sla.breaches", "count", owner="operations"),
            MetricDefinition("operations.regression.requests", "count", owner="operations"),
            MetricDefinition("operations.regression.count", "count", owner="operations"),
            MetricDefinition("operations.flow.requests", "count", owner="orchestration"),
            MetricDefinition("operations.flow.handoff_count", "count", owner="orchestration"),
            MetricDefinition("operations.billing.requests", "count", owner="finance"),
            MetricDefinition("operations.billing.estimated_cost_usd", "usd", owner="finance"),
            MetricDefinition("operations.billing.run_charge.requests", "count", owner="finance"),
            MetricDefinition("operations.billing.overage_cost_usd", "usd", owner="finance"),
        ],
        audit_policies=[
            AuditPolicy("operations.dashboard.viewed", required_fields=["actor", "period"], retention_days=180, severity="info"),
            AuditPolicy("operations.run_detail.viewed", required_fields=["actor", "run_id"], retention_days=180, severity="info"),
            AuditPolicy("operations.run_replay.viewed", required_fields=["actor", "run_id"], retention_days=180, severity="info"),
            AuditPolicy("operations.queue.viewed", required_fields=["actor", "queue_depth"], retention_days=180, severity="info"),
            AuditPolicy("operations.queue.retry.requested", required_fields=["actor", "task_id", "reason"], retention_days=180, severity="warning"),
            AuditPolicy(
                "operations.queue.bulk_retry.requested",
                required_fields=["actor", "task_ids", "reason"],
                retention_days=180,
                severity="warning",
            ),
            AuditPolicy(
                "operations.queue.manual_takeover.requested",
                required_fields=["actor", "task_id", "reason"],
                retention_days=180,
                severity="warning",
            ),
            AuditPolicy("operations.run.approval.recorded", required_fields=["actor", "run_id", "decision"], retention_days=365, severity="warning"),
            AuditPolicy("operations.risk.viewed", required_fields=["actor", "period"], retention_days=180, severity="info"),
            AuditPolicy("operations.sla.viewed", required_fields=["actor", "period"], retention_days=180, severity="info"),
            AuditPolicy("operations.regression.viewed", required_fields=["actor", "current_version"], retention_days=180, severity="info"),
            AuditPolicy("operations.flow.viewed", required_fields=["actor", "run_id"], retention_days=180, severity="info"),
            AuditPolicy("operations.billing.viewed", required_fields=["actor", "period", "tier"], retention_days=365, severity="info"),
            AuditPolicy("operations.billing.run_charge.viewed", required_fields=["actor", "run_id", "billing_model"], retention_days=365, severity="info"),
        ],
    )
