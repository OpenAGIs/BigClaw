import json
from dataclasses import dataclass, field
from typing import Dict, List, Sequence


@dataclass(frozen=True)
class SchemaField:
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
    def from_dict(cls, data: Dict[str, object]) -> "SchemaField":
        return cls(
            name=str(data["name"]),
            field_type=str(data["field_type"]),
            required=bool(data.get("required", True)),
            description=str(data.get("description", "")),
        )


@dataclass
class SurfaceSchema:
    name: str
    owner: str
    description: str = ""
    fields: List[SchemaField] = field(default_factory=list)
    sample: Dict[str, object] = field(default_factory=dict)

    @property
    def required_fields(self) -> List[str]:
        return [field.name for field in self.fields if field.required]

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "owner": self.owner,
            "description": self.description,
            "fields": [field.to_dict() for field in self.fields],
            "sample": self.sample,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceSchema":
        return cls(
            name=str(data["name"]),
            owner=str(data.get("owner", "")),
            description=str(data.get("description", "")),
            fields=[SchemaField.from_dict(field) for field in data.get("fields", [])],
            sample=dict(data.get("sample", {})),
        )


@dataclass
class DashboardRunContract:
    contract_id: str
    version: str
    dashboard_schema: SurfaceSchema
    run_detail_schema: SurfaceSchema

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "dashboard_schema": self.dashboard_schema.to_dict(),
            "run_detail_schema": self.run_detail_schema.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardRunContract":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            dashboard_schema=SurfaceSchema.from_dict(dict(data["dashboard_schema"])),
            run_detail_schema=SurfaceSchema.from_dict(dict(data["run_detail_schema"])),
        )


@dataclass
class DashboardRunContractAudit:
    contract_id: str
    version: str
    dashboard_missing_fields: List[str] = field(default_factory=list)
    dashboard_sample_gaps: List[str] = field(default_factory=list)
    run_detail_missing_fields: List[str] = field(default_factory=list)
    run_detail_sample_gaps: List[str] = field(default_factory=list)

    @property
    def release_ready(self) -> bool:
        return not (
            self.dashboard_missing_fields
            or self.dashboard_sample_gaps
            or self.run_detail_missing_fields
            or self.run_detail_sample_gaps
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "dashboard_missing_fields": list(self.dashboard_missing_fields),
            "dashboard_sample_gaps": list(self.dashboard_sample_gaps),
            "run_detail_missing_fields": list(self.run_detail_missing_fields),
            "run_detail_sample_gaps": list(self.run_detail_sample_gaps),
            "release_ready": self.release_ready,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardRunContractAudit":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            dashboard_missing_fields=[str(item) for item in data.get("dashboard_missing_fields", [])],
            dashboard_sample_gaps=[str(item) for item in data.get("dashboard_sample_gaps", [])],
            run_detail_missing_fields=[str(item) for item in data.get("run_detail_missing_fields", [])],
            run_detail_sample_gaps=[str(item) for item in data.get("run_detail_sample_gaps", [])],
        )


class DashboardRunContractLibrary:
    DASHBOARD_REQUIRED_FIELDS = [
        "dashboard_id",
        "generated_at",
        "period.label",
        "period.start",
        "period.end",
        "filters.team",
        "summary.total_runs",
        "summary.success_rate",
        "summary.approval_queue_depth",
        "summary.sla_breach_count",
        "kpis",
        "kpis[].name",
        "kpis[].value",
        "kpis[].target",
        "funnel",
        "blockers",
        "activity",
    ]
    RUN_DETAIL_REQUIRED_FIELDS = [
        "run_id",
        "task_id",
        "status",
        "started_at",
        "ended_at",
        "summary",
        "timeline",
        "timeline[].event_id",
        "timeline[].status",
        "artifacts",
        "closeout.validation_evidence",
        "closeout.git_push_succeeded",
        "closeout.git_log_stat_output",
    ]

    def audit(self, contract: DashboardRunContract) -> DashboardRunContractAudit:
        return DashboardRunContractAudit(
            contract_id=contract.contract_id,
            version=contract.version,
            dashboard_missing_fields=self._missing_field_defs(
                self.DASHBOARD_REQUIRED_FIELDS,
                contract.dashboard_schema.fields,
            ),
            dashboard_sample_gaps=self._missing_sample_paths(
                self.DASHBOARD_REQUIRED_FIELDS,
                contract.dashboard_schema.sample,
            ),
            run_detail_missing_fields=self._missing_field_defs(
                self.RUN_DETAIL_REQUIRED_FIELDS,
                contract.run_detail_schema.fields,
            ),
            run_detail_sample_gaps=self._missing_sample_paths(
                self.RUN_DETAIL_REQUIRED_FIELDS,
                contract.run_detail_schema.sample,
            ),
        )

    def build_default_contract(self) -> DashboardRunContract:
        return DashboardRunContract(
            contract_id="BIG-4301",
            version="v1",
            dashboard_schema=SurfaceSchema(
                name="DashboardKpiAggregate",
                owner="operations",
                description="Team dashboard response for KPI cards, funnel, blockers, and recent activity.",
                fields=[
                    SchemaField("dashboard_id", "string", description="Stable dashboard identifier."),
                    SchemaField("generated_at", "datetime", description="UTC generation timestamp."),
                    SchemaField("period.label", "string", description="Human-readable period label."),
                    SchemaField("period.start", "date", description="Inclusive reporting window start."),
                    SchemaField("period.end", "date", description="Inclusive reporting window end."),
                    SchemaField("filters.team", "string", description="Team or org filter applied."),
                    SchemaField("filters.viewer_role", "string", required=False, description="Persona used for visibility rules."),
                    SchemaField("summary.total_runs", "integer", description="Total runs in the period."),
                    SchemaField("summary.success_rate", "number", description="Completed-success ratio in percent."),
                    SchemaField("summary.approval_queue_depth", "integer", description="Runs waiting for approval."),
                    SchemaField("summary.sla_breach_count", "integer", description="Runs over SLA."),
                    SchemaField("kpis", "DashboardKpi[]", description="Ordered KPI cards shown in the hero grid."),
                    SchemaField("kpis[].name", "string", description="Metric identifier."),
                    SchemaField("kpis[].value", "number", description="Observed metric value."),
                    SchemaField("kpis[].target", "number", description="Target threshold."),
                    SchemaField("kpis[].unit", "string", required=False, description="Display suffix or unit."),
                    SchemaField("kpis[].direction", "string", required=False, description="Healthy trend direction."),
                    SchemaField("kpis[].healthy", "boolean", required=False, description="Precomputed health state."),
                    SchemaField("funnel", "DashboardFunnelStage[]", description="Pipeline distribution."),
                    SchemaField("blockers", "DashboardBlocker[]", description="Highest impact blockers."),
                    SchemaField("activity", "DashboardActivity[]", description="Most recent run activity."),
                ],
                sample={
                    "dashboard_id": "eng-overview-core-product",
                    "generated_at": "2026-03-11T09:30:00Z",
                    "period": {
                        "label": "2026-W11",
                        "start": "2026-03-09",
                        "end": "2026-03-15",
                    },
                    "filters": {
                        "team": "core-product",
                        "viewer_role": "engineering-manager",
                    },
                    "summary": {
                        "total_runs": 42,
                        "success_rate": 88.1,
                        "approval_queue_depth": 3,
                        "sla_breach_count": 2,
                    },
                    "kpis": [
                        {
                            "name": "success-rate",
                            "value": 88.1,
                            "target": 90.0,
                            "unit": "%",
                            "direction": "up",
                            "healthy": False,
                        },
                        {
                            "name": "average-cycle-minutes",
                            "value": 47.3,
                            "target": 60.0,
                            "unit": "m",
                            "direction": "down",
                            "healthy": True,
                        },
                    ],
                    "funnel": [
                        {"name": "queued", "count": 5, "share": 11.9},
                        {"name": "in-progress", "count": 7, "share": 16.7},
                        {"name": "awaiting-approval", "count": 3, "share": 7.1},
                        {"name": "completed", "count": 27, "share": 64.3},
                    ],
                    "blockers": [
                        {
                            "summary": "Security scan failures on release branch",
                            "affected_runs": 2,
                            "affected_tasks": ["OPE-121", "OPE-127"],
                            "owner": "security",
                            "severity": "high",
                        }
                    ],
                    "activity": [
                        {
                            "timestamp": "2026-03-11T09:20:00Z",
                            "run_id": "run-204",
                            "task_id": "OPE-127",
                            "status": "failed",
                            "summary": "Security scan failed after dependency bump",
                        }
                    ],
                },
            ),
            run_detail_schema=SurfaceSchema(
                name="RunDetail",
                owner="runtime",
                description="Canonical run detail payload for replay, timeline inspection, artifacts, and closeout evidence.",
                fields=[
                    SchemaField("run_id", "string", description="Stable run identifier."),
                    SchemaField("task_id", "string", description="Parent task identifier."),
                    SchemaField("status", "string", description="Current run status."),
                    SchemaField("started_at", "datetime", description="Run start timestamp."),
                    SchemaField("ended_at", "datetime", description="Run end timestamp."),
                    SchemaField("summary", "string", description="Operator-readable summary."),
                    SchemaField("medium", "string", required=False, description="Execution medium."),
                    SchemaField("timeline", "RunTimelineEvent[]", description="Chronological execution events."),
                    SchemaField("timeline[].event_id", "string", description="Stable event identifier."),
                    SchemaField("timeline[].lane", "string", required=False, description="UI lane or track."),
                    SchemaField("timeline[].timestamp", "datetime", required=False, description="Event timestamp."),
                    SchemaField("timeline[].status", "string", description="Event outcome."),
                    SchemaField("artifacts", "RunArtifact[]", description="Artifacts emitted by the run."),
                    SchemaField("closeout.validation_evidence", "string[]", description="Validation proof captured before finish."),
                    SchemaField("closeout.git_push_succeeded", "boolean", description="Whether push completed successfully."),
                    SchemaField("closeout.git_push_output", "string", required=False, description="Push command output."),
                    SchemaField("closeout.git_log_stat_output", "string", description="Captured git log --stat output."),
                ],
                sample={
                    "run_id": "run-204",
                    "task_id": "OPE-127",
                    "status": "completed",
                    "started_at": "2026-03-11T08:58:00Z",
                    "ended_at": "2026-03-11T09:24:00Z",
                    "medium": "codex-cli",
                    "summary": "Shipped release-branch scan hardening and captured closeout evidence.",
                    "timeline": [
                        {
                            "event_id": "evt-1",
                            "lane": "analysis",
                            "timestamp": "2026-03-11T08:59:00Z",
                            "status": "completed",
                            "title": "Inspected release branch failures",
                            "summary": "Reviewed the failing dependency diff and scan output.",
                            "details": ["Focused on lockfile drift and transitive CVE triage."],
                        },
                        {
                            "event_id": "evt-2",
                            "lane": "validation",
                            "timestamp": "2026-03-11T09:22:00Z",
                            "status": "completed",
                            "title": "Validated fix",
                            "summary": "Executed targeted tests and recorded the validation report.",
                            "details": ["pytest passed", "report stored under reports/OPE-127-validation.md"],
                        },
                    ],
                    "artifacts": [
                        {
                            "name": "validation-report",
                            "kind": "report",
                            "path": "reports/OPE-127-validation.md",
                            "timestamp": "2026-03-11T09:23:00Z",
                            "sha256": "abc123",
                            "metadata": {"ticket": "OPE-127"},
                        }
                    ],
                    "closeout": {
                        "validation_evidence": [
                            "bash scripts/ops/legacy_python_smoke.sh -> legacy_python_smoke_ok",
                            "BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh -> legacy Python migration smoke suite validated from source",
                        ],
                        "git_push_succeeded": True,
                        "git_push_output": "To github.com:OpenAGIs/BigClaw.git\\n   abc123..def456  main -> main",
                        "git_log_stat_output": "commit def456\\n src/bigclaw/security.py | 12 ++++++++++--",
                    },
                },
            ),
        )

    def _missing_field_defs(self, required_fields: Sequence[str], fields: Sequence[SchemaField]) -> List[str]:
        defined = {field.name for field in fields}
        return [field_name for field_name in required_fields if field_name not in defined]

    def _missing_sample_paths(self, required_fields: Sequence[str], payload: Dict[str, object]) -> List[str]:
        return [field_name for field_name in required_fields if not self._path_exists(payload, field_name)]

    def _path_exists(self, payload: object, path: str) -> bool:
        current_items = [payload]
        for part in path.split("."):
            next_items: List[object] = []
            is_list = part.endswith("[]")
            key = part[:-2] if is_list else part
            for item in current_items:
                if not isinstance(item, dict) or key not in item:
                    continue
                value = item[key]
                if is_list:
                    if isinstance(value, list) and value:
                        next_items.extend(value)
                    else:
                        continue
                else:
                    next_items.append(value)
            if not next_items:
                return False
            current_items = next_items
        return True


def render_dashboard_run_contract_report(
    contract: DashboardRunContract,
    audit: DashboardRunContractAudit,
) -> str:
    sections = [
        "# Dashboard and Run Contract",
        "",
        f"- Contract ID: {contract.contract_id}",
        f"- Version: {contract.version}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Dashboard KPI Aggregate",
        f"- Name: {contract.dashboard_schema.name}",
        f"- Owner: {contract.dashboard_schema.owner}",
        f"- Required Fields: {', '.join(contract.dashboard_schema.required_fields)}",
        f"- Missing Required Fields: {', '.join(audit.dashboard_missing_fields) or 'none'}",
        f"- Sample Gaps: {', '.join(audit.dashboard_sample_gaps) or 'none'}",
        "",
        "```json",
        json.dumps(contract.dashboard_schema.sample, indent=2, sort_keys=True),
        "```",
        "",
        "## Run Detail",
        f"- Name: {contract.run_detail_schema.name}",
        f"- Owner: {contract.run_detail_schema.owner}",
        f"- Required Fields: {', '.join(contract.run_detail_schema.required_fields)}",
        f"- Missing Required Fields: {', '.join(audit.run_detail_missing_fields) or 'none'}",
        f"- Sample Gaps: {', '.join(audit.run_detail_sample_gaps) or 'none'}",
        "",
        "```json",
        json.dumps(contract.run_detail_schema.sample, indent=2, sort_keys=True),
        "```",
    ]
    return "\n".join(sections)
