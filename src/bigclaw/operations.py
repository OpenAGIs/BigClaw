from dataclasses import dataclass, field
from datetime import datetime, timezone
from difflib import unified_diff
from pathlib import Path
from typing import Dict, List, Optional, Sequence

from .evaluation import BenchmarkSuiteResult
from .observability import Task
from .reports import (
    SharedViewContext,
    build_console_actions,
    render_console_actions,
    render_shared_view_context,
    write_report,
)
from .queue import PersistentTaskQueue


STATUS_COMPLETE = {"approved", "accepted", "completed", "succeeded"}
STATUS_ACTIONABLE = {"needs-approval", "failed", "rejected"}


@dataclass
class TriageCluster:
    reason: str
    run_ids: List[str] = field(default_factory=list)
    task_ids: List[str] = field(default_factory=list)
    statuses: List[str] = field(default_factory=list)

    @property
    def occurrences(self) -> int:
        return len(self.run_ids)


@dataclass
class RegressionFinding:
    case_id: str
    baseline_score: int
    current_score: int
    delta: int
    severity: str
    summary: str


@dataclass
class OperationsSnapshot:
    total_runs: int
    status_counts: Dict[str, int]
    success_rate: float
    approval_queue_depth: int
    sla_target_minutes: int
    sla_breach_count: int
    average_cycle_minutes: float
    top_blockers: List[TriageCluster] = field(default_factory=list)


@dataclass
class WeeklyOperationsReport:
    name: str
    period: str
    snapshot: OperationsSnapshot
    regressions: List[RegressionFinding] = field(default_factory=list)


@dataclass
class RegressionCenter:
    name: str
    baseline_version: str
    current_version: str
    regressions: List[RegressionFinding] = field(default_factory=list)
    improved_cases: List[str] = field(default_factory=list)
    unchanged_cases: List[str] = field(default_factory=list)

    @property
    def regression_count(self) -> int:
        return len(self.regressions)


@dataclass
class VersionedArtifact:
    artifact_type: str
    artifact_id: str
    version: str
    updated_at: str
    author: str
    summary: str
    content: str
    change_ticket: Optional[str] = None


@dataclass
class VersionChangeSummary:
    from_version: str
    to_version: str
    additions: int
    deletions: int
    changed_lines: int
    preview: List[str] = field(default_factory=list)

    @property
    def has_changes(self) -> bool:
        return self.changed_lines > 0


@dataclass
class VersionedArtifactHistory:
    artifact_type: str
    artifact_id: str
    current_version: str
    current_updated_at: str
    current_author: str
    current_summary: str
    revision_count: int
    revisions: List[VersionedArtifact] = field(default_factory=list)
    rollback_version: Optional[str] = None
    rollback_ready: bool = False
    change_summary: Optional[VersionChangeSummary] = None


@dataclass
class PolicyPromptVersionCenter:
    name: str
    generated_at: str
    histories: List[VersionedArtifactHistory] = field(default_factory=list)

    @property
    def artifact_count(self) -> int:
        return len(self.histories)

    @property
    def rollback_ready_count(self) -> int:
        return sum(1 for history in self.histories if history.rollback_ready)


@dataclass
class WeeklyOperationsArtifacts:
    root_dir: str
    weekly_report_path: str
    dashboard_path: str
    metric_spec_path: Optional[str] = None
    regression_center_path: Optional[str] = None
    queue_control_path: Optional[str] = None
    version_center_path: Optional[str] = None


@dataclass
class QueueControlCenter:
    queue_depth: int
    queued_by_priority: Dict[str, int]
    queued_by_risk: Dict[str, int]
    execution_media: Dict[str, int]
    waiting_approval_runs: int
    blocked_tasks: List[str] = field(default_factory=list)
    queued_tasks: List[str] = field(default_factory=list)
    actions: Dict[str, List] = field(default_factory=dict)


@dataclass
class EngineeringOverviewKPI:
    name: str
    value: float
    target: float
    unit: str = ""
    direction: str = "up"

    @property
    def healthy(self) -> bool:
        if self.direction == "down":
            return self.value <= self.target
        return self.value >= self.target


@dataclass
class EngineeringFunnelStage:
    name: str
    count: int
    share: float


@dataclass
class EngineeringOverviewBlocker:
    summary: str
    affected_runs: int
    affected_tasks: List[str] = field(default_factory=list)
    owner: str = "engineering"
    severity: str = "medium"


@dataclass
class EngineeringActivity:
    timestamp: str
    run_id: str
    task_id: str
    status: str
    summary: str


@dataclass
class EngineeringOverviewPermission:
    viewer_role: str
    allowed_modules: List[str] = field(default_factory=list)

    def can_view(self, module: str) -> bool:
        return module in self.allowed_modules


@dataclass
class EngineeringOverview:
    name: str
    period: str
    snapshot: OperationsSnapshot
    permissions: EngineeringOverviewPermission
    kpis: List[EngineeringOverviewKPI] = field(default_factory=list)
    funnel: List[EngineeringFunnelStage] = field(default_factory=list)
    blockers: List[EngineeringOverviewBlocker] = field(default_factory=list)
    activities: List[EngineeringActivity] = field(default_factory=list)


@dataclass(frozen=True)
class OperationsMetricDefinition:
    metric_id: str
    label: str
    unit: str
    direction: str
    formula: str
    description: str
    source_fields: List[str] = field(default_factory=list)


@dataclass(frozen=True)
class OperationsMetricValue:
    metric_id: str
    label: str
    value: float
    display_value: str
    numerator: float
    denominator: float
    unit: str
    evidence: List[str] = field(default_factory=list)


@dataclass(frozen=True)
class OperationsMetricSpec:
    name: str
    generated_at: str
    period_start: str
    period_end: str
    timezone_name: str
    definitions: List[OperationsMetricDefinition] = field(default_factory=list)
    values: List[OperationsMetricValue] = field(default_factory=list)


@dataclass(frozen=True)
class DashboardWidgetSpec:
    widget_id: str
    title: str
    module: str
    data_source: str
    default_width: int = 4
    default_height: int = 3
    min_width: int = 2
    max_width: int = 12

    def to_dict(self) -> Dict[str, object]:
        return {
            "widget_id": self.widget_id,
            "title": self.title,
            "module": self.module,
            "data_source": self.data_source,
            "default_width": self.default_width,
            "default_height": self.default_height,
            "min_width": self.min_width,
            "max_width": self.max_width,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardWidgetSpec":
        return cls(
            widget_id=str(data["widget_id"]),
            title=str(data["title"]),
            module=str(data["module"]),
            data_source=str(data["data_source"]),
            default_width=int(data.get("default_width", 4)),
            default_height=int(data.get("default_height", 3)),
            min_width=int(data.get("min_width", 2)),
            max_width=int(data.get("max_width", 12)),
        )


@dataclass(frozen=True)
class DashboardWidgetPlacement:
    placement_id: str
    widget_id: str
    column: int
    row: int
    width: int
    height: int
    title_override: str = ""
    filters: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "placement_id": self.placement_id,
            "widget_id": self.widget_id,
            "column": self.column,
            "row": self.row,
            "width": self.width,
            "height": self.height,
            "title_override": self.title_override,
            "filters": list(self.filters),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardWidgetPlacement":
        return cls(
            placement_id=str(data["placement_id"]),
            widget_id=str(data["widget_id"]),
            column=int(data.get("column", 0)),
            row=int(data.get("row", 0)),
            width=int(data.get("width", 1)),
            height=int(data.get("height", 1)),
            title_override=str(data.get("title_override", "")),
            filters=[str(item) for item in data.get("filters", [])],
        )


@dataclass
class DashboardLayout:
    layout_id: str
    name: str
    columns: int = 12
    placements: List[DashboardWidgetPlacement] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "layout_id": self.layout_id,
            "name": self.name,
            "columns": self.columns,
            "placements": [placement.to_dict() for placement in self.placements],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardLayout":
        return cls(
            layout_id=str(data["layout_id"]),
            name=str(data["name"]),
            columns=int(data.get("columns", 12)),
            placements=[DashboardWidgetPlacement.from_dict(item) for item in data.get("placements", [])],
        )


@dataclass
class DashboardBuilder:
    name: str
    period: str
    owner: str
    permissions: EngineeringOverviewPermission
    widgets: List[DashboardWidgetSpec] = field(default_factory=list)
    layouts: List[DashboardLayout] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def widget_index(self) -> Dict[str, DashboardWidgetSpec]:
        return {widget.widget_id: widget for widget in self.widgets}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "period": self.period,
            "owner": self.owner,
            "permissions": {
                "viewer_role": self.permissions.viewer_role,
                "allowed_modules": list(self.permissions.allowed_modules),
            },
            "widgets": [widget.to_dict() for widget in self.widgets],
            "layouts": [layout.to_dict() for layout in self.layouts],
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardBuilder":
        permissions = dict(data.get("permissions", {}))
        return cls(
            name=str(data["name"]),
            period=str(data["period"]),
            owner=str(data["owner"]),
            permissions=EngineeringOverviewPermission(
                viewer_role=str(permissions.get("viewer_role", "contributor")),
                allowed_modules=[str(item) for item in permissions.get("allowed_modules", [])],
            ),
            widgets=[DashboardWidgetSpec.from_dict(item) for item in data.get("widgets", [])],
            layouts=[DashboardLayout.from_dict(item) for item in data.get("layouts", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


@dataclass
class DashboardBuilderAudit:
    name: str
    total_widgets: int
    layout_count: int
    placed_widgets: int
    duplicate_placement_ids: List[str] = field(default_factory=list)
    missing_widget_defs: List[str] = field(default_factory=list)
    inaccessible_widgets: List[str] = field(default_factory=list)
    overlapping_placements: List[str] = field(default_factory=list)
    out_of_bounds_placements: List[str] = field(default_factory=list)
    empty_layouts: List[str] = field(default_factory=list)
    documentation_complete: bool = False

    @property
    def release_ready(self) -> bool:
        return not (
            self.duplicate_placement_ids
            or self.missing_widget_defs
            or self.inaccessible_widgets
            or self.overlapping_placements
            or self.out_of_bounds_placements
            or self.empty_layouts
            or not self.documentation_complete
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "total_widgets": self.total_widgets,
            "layout_count": self.layout_count,
            "placed_widgets": self.placed_widgets,
            "duplicate_placement_ids": list(self.duplicate_placement_ids),
            "missing_widget_defs": list(self.missing_widget_defs),
            "inaccessible_widgets": list(self.inaccessible_widgets),
            "overlapping_placements": list(self.overlapping_placements),
            "out_of_bounds_placements": list(self.out_of_bounds_placements),
            "empty_layouts": list(self.empty_layouts),
            "documentation_complete": self.documentation_complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardBuilderAudit":
        return cls(
            name=str(data["name"]),
            total_widgets=int(data.get("total_widgets", 0)),
            layout_count=int(data.get("layout_count", 0)),
            placed_widgets=int(data.get("placed_widgets", 0)),
            duplicate_placement_ids=[str(item) for item in data.get("duplicate_placement_ids", [])],
            missing_widget_defs=[str(item) for item in data.get("missing_widget_defs", [])],
            inaccessible_widgets=[str(item) for item in data.get("inaccessible_widgets", [])],
            overlapping_placements=[str(item) for item in data.get("overlapping_placements", [])],
            out_of_bounds_placements=[str(item) for item in data.get("out_of_bounds_placements", [])],
            empty_layouts=[str(item) for item in data.get("empty_layouts", [])],
            documentation_complete=bool(data.get("documentation_complete", False)),
        )


class OperationsAnalytics:
    METRIC_DEFINITIONS = (
        OperationsMetricDefinition(
            metric_id="runs-today",
            label="Runs Today",
            unit="runs",
            direction="up",
            formula="count(run.started_at within [period_start, period_end])",
            description="Number of runs that started inside the reporting day window.",
            source_fields=["started_at"],
        ),
        OperationsMetricDefinition(
            metric_id="avg-lead-time",
            label="Avg Lead Time",
            unit="m",
            direction="down",
            formula="sum(cycle_minutes for runs with started_at and ended_at) / measured_runs",
            description="Average elapsed minutes from run start to run end for runs with complete timestamps.",
            source_fields=["started_at", "ended_at"],
        ),
        OperationsMetricDefinition(
            metric_id="intervention-rate",
            label="Intervention Rate",
            unit="%",
            direction="down",
            formula="100 * actionable_runs / total_runs",
            description="Share of runs that require operator intervention because they ended in an actionable status.",
            source_fields=["status"],
        ),
        OperationsMetricDefinition(
            metric_id="sla",
            label="SLA",
            unit="%",
            direction="up",
            formula="100 * compliant_runs / measured_runs where compliant_runs have cycle_minutes <= sla_target_minutes",
            description="Share of measured runs that met the SLA target.",
            source_fields=["started_at", "ended_at"],
        ),
        OperationsMetricDefinition(
            metric_id="regression",
            label="Regression",
            unit="cases",
            direction="down",
            formula="count(current.compare(baseline) deltas < 0 or pass->fail transitions)",
            description="Number of benchmark cases that regressed against the provided baseline suite.",
            source_fields=["benchmark.current", "benchmark.baseline"],
        ),
        OperationsMetricDefinition(
            metric_id="risk",
            label="Risk",
            unit="score",
            direction="down",
            formula="sum(resolved_run_risk_score) / runs_with_risk where risk_score.total wins over risk_level mapping low=25, medium=60, high=90",
            description="Average per-run risk score from explicit risk scores or normalized risk levels.",
            source_fields=["risk_score.total", "risk_level"],
        ),
        OperationsMetricDefinition(
            metric_id="spend",
            label="Spend",
            unit="USD",
            direction="down",
            formula="sum(first non-null of spend_usd, cost_usd, spend, cost across runs)",
            description="Total reported run spend in USD over the reporting window.",
            source_fields=["spend_usd", "cost_usd", "spend", "cost"],
        ),
    )

    def summarize_runs(
        self,
        runs: Sequence[dict],
        sla_target_minutes: int = 60,
        top_n_blockers: int = 3,
    ) -> OperationsSnapshot:
        status_counts: Dict[str, int] = {}
        total_cycle_minutes = 0.0
        cycle_count = 0
        completed = 0
        approval_queue_depth = 0
        sla_breach_count = 0

        for run in runs:
            status = str(run.get("status", "unknown"))
            status_counts[status] = status_counts.get(status, 0) + 1

            if status == "needs-approval":
                approval_queue_depth += 1

            cycle_minutes = self._cycle_minutes(run)
            if cycle_minutes is not None:
                total_cycle_minutes += cycle_minutes
                cycle_count += 1
                if cycle_minutes > sla_target_minutes:
                    sla_breach_count += 1

            if status in STATUS_COMPLETE:
                completed += 1

        success_rate = round((completed / len(runs)) * 100, 1) if runs else 0.0
        average_cycle_minutes = round(total_cycle_minutes / cycle_count, 1) if cycle_count else 0.0
        blockers = self.build_triage_clusters(runs)[:top_n_blockers]
        return OperationsSnapshot(
            total_runs=len(runs),
            status_counts=status_counts,
            success_rate=success_rate,
            approval_queue_depth=approval_queue_depth,
            sla_target_minutes=sla_target_minutes,
            sla_breach_count=sla_breach_count,
            average_cycle_minutes=average_cycle_minutes,
            top_blockers=blockers,
        )

    def build_metric_spec(
        self,
        runs: Sequence[dict],
        *,
        period_start: str,
        period_end: str,
        timezone_name: str = "UTC",
        generated_at: Optional[str] = None,
        sla_target_minutes: int = 60,
        current_suite: Optional[BenchmarkSuiteResult] = None,
        baseline_suite: Optional[BenchmarkSuiteResult] = None,
    ) -> OperationsMetricSpec:
        period_start_dt = self._parse_ts(period_start)
        period_end_dt = self._parse_ts(period_end)
        if period_start_dt is None or period_end_dt is None or period_end_dt < period_start_dt:
            raise ValueError("period_start and period_end must be valid ISO-8601 timestamps with period_end >= period_start")

        runs_today = 0
        lead_time_sum = 0.0
        lead_time_count = 0
        actionable_runs = 0
        sla_compliant_runs = 0
        risk_sum = 0.0
        risk_count = 0
        spend_total = 0.0

        for run in runs:
            started_at = self._parse_ts(str(run.get("started_at", "")))
            if started_at is not None and period_start_dt <= started_at <= period_end_dt:
                runs_today += 1

            cycle_minutes = self._cycle_minutes(run)
            if cycle_minutes is not None:
                lead_time_sum += cycle_minutes
                lead_time_count += 1
                if cycle_minutes <= sla_target_minutes:
                    sla_compliant_runs += 1

            if str(run.get("status", "unknown")) in STATUS_ACTIONABLE:
                actionable_runs += 1

            risk_score = self._resolve_run_risk_score(run)
            if risk_score is not None:
                risk_sum += risk_score
                risk_count += 1

            spend_total += self._resolve_run_spend(run)

        regression_findings = self.analyze_regressions(current_suite, baseline_suite) if current_suite is not None else []
        total_runs = len(runs)
        avg_lead = round(lead_time_sum / lead_time_count, 1) if lead_time_count else 0.0
        intervention_rate = round((actionable_runs / total_runs) * 100, 1) if total_runs else 0.0
        sla_value = round((sla_compliant_runs / lead_time_count) * 100, 1) if lead_time_count else 0.0
        avg_risk = round(risk_sum / risk_count, 1) if risk_count else 0.0
        spend_total = round(spend_total, 2)

        values = [
            OperationsMetricValue(
                metric_id="runs-today",
                label="Runs Today",
                value=float(runs_today),
                display_value=str(runs_today),
                numerator=float(runs_today),
                denominator=float(total_runs),
                unit="runs",
                evidence=[f"{runs_today} of {total_runs} runs started inside the reporting window."],
            ),
            OperationsMetricValue(
                metric_id="avg-lead-time",
                label="Avg Lead Time",
                value=avg_lead,
                display_value=f"{avg_lead:.1f}m",
                numerator=round(lead_time_sum, 1),
                denominator=float(lead_time_count),
                unit="m",
                evidence=[f"{lead_time_count} runs had valid start/end timestamps."],
            ),
            OperationsMetricValue(
                metric_id="intervention-rate",
                label="Intervention Rate",
                value=intervention_rate,
                display_value=f"{intervention_rate:.1f}%",
                numerator=float(actionable_runs),
                denominator=float(total_runs),
                unit="%",
                evidence=[f"Actionable statuses counted: {', '.join(sorted(STATUS_ACTIONABLE))}."],
            ),
            OperationsMetricValue(
                metric_id="sla",
                label="SLA",
                value=sla_value,
                display_value=f"{sla_value:.1f}%",
                numerator=float(sla_compliant_runs),
                denominator=float(lead_time_count),
                unit="%",
                evidence=[
                    f"SLA target: {sla_target_minutes} minutes.",
                    f"{sla_compliant_runs} of {lead_time_count} measured runs met target.",
                ],
            ),
            OperationsMetricValue(
                metric_id="regression",
                label="Regression",
                value=float(len(regression_findings)),
                display_value=str(len(regression_findings)),
                numerator=float(len(regression_findings)),
                denominator=float(len(current_suite.results)) if current_suite is not None else 0.0,
                unit="cases",
                evidence=[
                    f"Baseline provided: {baseline_suite is not None}.",
                    f"Current suite provided: {current_suite is not None}.",
                ],
            ),
            OperationsMetricValue(
                metric_id="risk",
                label="Risk",
                value=avg_risk,
                display_value=f"{avg_risk:.1f}",
                numerator=round(risk_sum, 1),
                denominator=float(risk_count),
                unit="score",
                evidence=["Risk score precedence: risk_score.total, then risk_level mapping low=25 medium=60 high=90."],
            ),
            OperationsMetricValue(
                metric_id="spend",
                label="Spend",
                value=spend_total,
                display_value=f"${spend_total:.2f}",
                numerator=spend_total,
                denominator=float(total_runs),
                unit="USD",
                evidence=["Spend field precedence: spend_usd, cost_usd, spend, cost."],
            ),
        ]

        return OperationsMetricSpec(
            name="Operations Metric Spec",
            generated_at=generated_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            period_start=period_start,
            period_end=period_end,
            timezone_name=timezone_name,
            definitions=list(self.METRIC_DEFINITIONS),
            values=values,
        )

    def build_triage_clusters(self, runs: Sequence[dict]) -> List[TriageCluster]:
        clusters: Dict[str, TriageCluster] = {}
        for run in runs:
            status = str(run.get("status", "unknown"))
            if status not in STATUS_ACTIONABLE:
                continue

            reason = self._primary_reason(run)
            cluster = clusters.setdefault(reason, TriageCluster(reason=reason))
            run_id = str(run.get("run_id", ""))
            task_id = str(run.get("task_id", ""))
            if run_id and run_id not in cluster.run_ids:
                cluster.run_ids.append(run_id)
            if task_id and task_id not in cluster.task_ids:
                cluster.task_ids.append(task_id)
            if status not in cluster.statuses:
                cluster.statuses.append(status)

        return sorted(
            clusters.values(),
            key=lambda cluster: (-cluster.occurrences, cluster.reason),
        )

    def analyze_regressions(
        self,
        current: BenchmarkSuiteResult,
        baseline: Optional[BenchmarkSuiteResult] = None,
    ) -> List[RegressionFinding]:
        if baseline is None:
            return []

        baseline_results = {result.case_id: result for result in baseline.results}
        findings: List[RegressionFinding] = []
        for comparison in current.compare(baseline):
            baseline_result = baseline_results.get(comparison.case_id)
            current_result = next(result for result in current.results if result.case_id == comparison.case_id)
            if comparison.delta >= 0 and not (baseline_result and baseline_result.passed and not current_result.passed):
                continue

            severity = "high" if comparison.delta <= -20 or (baseline_result and baseline_result.passed and not current_result.passed) else "medium"
            summary = (
                f"score dropped from {comparison.baseline_score} to {comparison.current_score}"
                if comparison.delta < 0
                else "case regressed from passing to failing"
            )
            findings.append(
                RegressionFinding(
                    case_id=comparison.case_id,
                    baseline_score=comparison.baseline_score,
                    current_score=comparison.current_score,
                    delta=comparison.delta,
                    severity=severity,
                    summary=summary,
                )
            )

        return sorted(findings, key=lambda finding: (finding.delta, finding.case_id))

    def build_regression_center(
        self,
        current: BenchmarkSuiteResult,
        baseline: BenchmarkSuiteResult,
        name: str = "Regression Analysis Center",
    ) -> RegressionCenter:
        regressions = self.analyze_regressions(current, baseline)
        comparisons = current.compare(baseline)
        improved_cases = sorted(comparison.case_id for comparison in comparisons if comparison.delta > 0)
        unchanged_cases = sorted(comparison.case_id for comparison in comparisons if comparison.delta == 0)
        return RegressionCenter(
            name=name,
            baseline_version=baseline.version,
            current_version=current.version,
            regressions=regressions,
            improved_cases=improved_cases,
            unchanged_cases=unchanged_cases,
        )

    def build_queue_control_center(
        self,
        queue: PersistentTaskQueue,
        runs: Sequence[dict],
    ) -> QueueControlCenter:
        queued_tasks = queue.peek_tasks()
        queued_by_priority = {"P0": 0, "P1": 0, "P2": 0}
        queued_by_risk = {"low": 0, "medium": 0, "high": 0}
        for task in queued_tasks:
            queued_by_priority[f"P{int(task.priority)}"] += 1
            queued_by_risk[task.risk_level.value] += 1

        execution_media: Dict[str, int] = {}
        waiting_approval_runs = 0
        blocked_tasks: List[str] = []
        for run in runs:
            medium = str(run.get("medium", "unknown"))
            execution_media[medium] = execution_media.get(medium, 0) + 1
            if run.get("status") == "needs-approval":
                waiting_approval_runs += 1
                task_id = str(run.get("task_id", ""))
                if task_id and task_id not in blocked_tasks:
                    blocked_tasks.append(task_id)

        return QueueControlCenter(
            queue_depth=queue.size(),
            queued_by_priority=queued_by_priority,
            queued_by_risk=queued_by_risk,
            execution_media=execution_media,
            waiting_approval_runs=waiting_approval_runs,
            blocked_tasks=blocked_tasks,
            queued_tasks=[task.task_id for task in queued_tasks],
            actions={
                task.task_id: build_console_actions(
                    task.task_id,
                    allow_retry=task.task_id in blocked_tasks,
                    retry_reason="" if task.task_id in blocked_tasks else "retry is reserved for blocked queue items",
                    allow_pause=task.task_id not in blocked_tasks,
                    pause_reason="" if task.task_id not in blocked_tasks else "approval-blocked tasks should be escalated instead of paused",
                    allow_escalate=task.task_id in blocked_tasks,
                    escalate_reason="" if task.task_id in blocked_tasks else "escalate is reserved for blocked queue items",
                )
                for task in queued_tasks
            },
        )

    def build_policy_prompt_version_center(
        self,
        artifacts: Sequence[VersionedArtifact],
        name: str = "Policy/Prompt Version Center",
        generated_at: Optional[str] = None,
        diff_preview_lines: int = 8,
    ) -> PolicyPromptVersionCenter:
        grouped: Dict[tuple[str, str], List[VersionedArtifact]] = {}
        for artifact in artifacts:
            key = (artifact.artifact_type, artifact.artifact_id)
            grouped.setdefault(key, []).append(artifact)

        histories: List[VersionedArtifactHistory] = []
        for artifact_type, artifact_id in sorted(grouped.keys()):
            revisions = sorted(
                grouped[(artifact_type, artifact_id)],
                key=lambda artifact: self._parse_ts(artifact.updated_at) or datetime.min.replace(tzinfo=timezone.utc),
                reverse=True,
            )
            current = revisions[0]
            previous = revisions[1] if len(revisions) > 1 else None
            change_summary = None
            rollback_version = None
            rollback_ready = False

            if previous is not None:
                change_summary = self._summarize_version_change(previous, current, preview_lines=diff_preview_lines)
                rollback_version = previous.version
                rollback_ready = bool(previous.content.strip())

            histories.append(
                VersionedArtifactHistory(
                    artifact_type=artifact_type,
                    artifact_id=artifact_id,
                    current_version=current.version,
                    current_updated_at=current.updated_at,
                    current_author=current.author,
                    current_summary=current.summary,
                    revision_count=len(revisions),
                    revisions=revisions,
                    rollback_version=rollback_version,
                    rollback_ready=rollback_ready,
                    change_summary=change_summary,
                )
            )

        return PolicyPromptVersionCenter(
            name=name,
            generated_at=generated_at or datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
            histories=histories,
        )

    def build_engineering_overview(
        self,
        name: str,
        period: str,
        runs: Sequence[dict],
        viewer_role: str,
        sla_target_minutes: int = 60,
        top_n_blockers: int = 3,
        recent_activity_limit: int = 5,
    ) -> EngineeringOverview:
        snapshot = self.summarize_runs(
            runs,
            sla_target_minutes=sla_target_minutes,
            top_n_blockers=top_n_blockers,
        )
        permissions = self._permissions_for_role(viewer_role)
        kpis = [
            EngineeringOverviewKPI(name="success-rate", value=snapshot.success_rate, target=90.0, unit="%"),
            EngineeringOverviewKPI(
                name="approval-queue-depth",
                value=float(snapshot.approval_queue_depth),
                target=2.0,
                direction="down",
            ),
            EngineeringOverviewKPI(
                name="sla-breaches",
                value=float(snapshot.sla_breach_count),
                target=0.0,
                direction="down",
            ),
            EngineeringOverviewKPI(
                name="average-cycle-minutes",
                value=snapshot.average_cycle_minutes,
                target=float(sla_target_minutes),
                unit="m",
                direction="down",
            ),
        ]
        blockers = [
            EngineeringOverviewBlocker(
                summary=cluster.reason,
                affected_runs=cluster.occurrences,
                affected_tasks=cluster.task_ids,
                owner=self._owner_for_cluster(cluster),
                severity=self._severity_for_cluster(cluster),
            )
            for cluster in snapshot.top_blockers
        ]
        return EngineeringOverview(
            name=name,
            period=period,
            snapshot=snapshot,
            permissions=permissions,
            kpis=kpis,
            funnel=self._build_funnel(snapshot.status_counts, snapshot.total_runs),
            blockers=blockers,
            activities=self._build_recent_activities(runs, recent_activity_limit),
        )

    def build_weekly_report(
        self,
        name: str,
        period: str,
        runs: Sequence[dict],
        current_suite: Optional[BenchmarkSuiteResult] = None,
        baseline_suite: Optional[BenchmarkSuiteResult] = None,
        sla_target_minutes: int = 60,
    ) -> WeeklyOperationsReport:
        snapshot = self.summarize_runs(runs, sla_target_minutes=sla_target_minutes)
        regressions = []
        if current_suite is not None:
            regressions = self.analyze_regressions(current_suite, baseline_suite)
        return WeeklyOperationsReport(
            name=name,
            period=period,
            snapshot=snapshot,
            regressions=regressions,
        )

    def build_dashboard_builder(
        self,
        name: str,
        period: str,
        owner: str,
        viewer_role: str,
        widgets: Sequence[DashboardWidgetSpec],
        layouts: Sequence[DashboardLayout],
        documentation_complete: bool = False,
    ) -> DashboardBuilder:
        return DashboardBuilder(
            name=name,
            period=period,
            owner=owner,
            permissions=self._permissions_for_role(viewer_role),
            widgets=list(widgets),
            layouts=[self.normalize_dashboard_layout(layout, widgets) for layout in layouts],
            documentation_complete=documentation_complete,
        )

    def normalize_dashboard_layout(
        self,
        layout: DashboardLayout,
        widgets: Sequence[DashboardWidgetSpec],
    ) -> DashboardLayout:
        widget_index = {widget.widget_id: widget for widget in widgets}
        normalized: List[DashboardWidgetPlacement] = []
        column_count = max(1, layout.columns)
        for placement in layout.placements:
            spec = widget_index.get(placement.widget_id)
            min_width = spec.min_width if spec is not None else 1
            max_width = min(spec.max_width, column_count) if spec is not None else column_count
            width = max(min_width, min(placement.width, max_width))
            column = max(0, placement.column)
            if column + width > column_count:
                column = max(0, column_count - width)
            normalized.append(
                DashboardWidgetPlacement(
                    placement_id=placement.placement_id,
                    widget_id=placement.widget_id,
                    column=column,
                    row=max(0, placement.row),
                    width=width,
                    height=max(1, placement.height),
                    title_override=placement.title_override,
                    filters=list(placement.filters),
                )
            )

        normalized.sort(key=lambda item: (item.row, item.column, item.placement_id))
        return DashboardLayout(
            layout_id=layout.layout_id,
            name=layout.name,
            columns=column_count,
            placements=normalized,
        )

    def audit_dashboard_builder(self, dashboard: DashboardBuilder) -> DashboardBuilderAudit:
        widget_index = dashboard.widget_index
        placement_counts: Dict[str, int] = {}
        missing_widget_defs: set[str] = set()
        inaccessible_widgets: set[str] = set()
        overlapping_placements: set[str] = set()
        out_of_bounds_placements: set[str] = set()
        empty_layouts: List[str] = []
        placed_widgets = 0

        for layout in dashboard.layouts:
            if not layout.placements:
                empty_layouts.append(layout.layout_id)
                continue

            placed_widgets += len(layout.placements)
            for placement in layout.placements:
                placement_counts[placement.placement_id] = placement_counts.get(placement.placement_id, 0) + 1
                spec = widget_index.get(placement.widget_id)
                if spec is None:
                    missing_widget_defs.add(placement.widget_id)
                else:
                    if not dashboard.permissions.can_view(spec.module):
                        inaccessible_widgets.add(placement.widget_id)
                if placement.column + placement.width > layout.columns:
                    out_of_bounds_placements.add(placement.placement_id)

            for index, placement in enumerate(layout.placements):
                for other in layout.placements[index + 1 :]:
                    if self._placements_overlap(placement, other):
                        overlapping_placements.add(
                            f"{layout.layout_id}:{placement.placement_id}<->{other.placement_id}"
                        )

        duplicate_ids = sorted(
            placement_id for placement_id, count in placement_counts.items() if count > 1
        )
        return DashboardBuilderAudit(
            name=dashboard.name,
            total_widgets=len(dashboard.widgets),
            layout_count=len(dashboard.layouts),
            placed_widgets=placed_widgets,
            duplicate_placement_ids=duplicate_ids,
            missing_widget_defs=sorted(missing_widget_defs),
            inaccessible_widgets=sorted(inaccessible_widgets),
            overlapping_placements=sorted(overlapping_placements),
            out_of_bounds_placements=sorted(out_of_bounds_placements),
            empty_layouts=sorted(empty_layouts),
            documentation_complete=dashboard.documentation_complete,
        )

    def _primary_reason(self, run: dict) -> str:
        for audit in run.get("audits", []):
            reason = audit.get("details", {}).get("reason")
            if reason:
                return str(reason)
        summary = str(run.get("summary", "")).strip()
        if summary:
            return summary
        return str(run.get("status", "unknown"))

    def _cycle_minutes(self, run: dict) -> Optional[float]:
        started_at = run.get("started_at")
        ended_at = run.get("ended_at")
        if not started_at or not ended_at:
            return None
        start = self._parse_ts(str(started_at))
        end = self._parse_ts(str(ended_at))
        if start is None or end is None or end < start:
            return None
        return round((end - start).total_seconds() / 60, 1)

    def _parse_ts(self, value: str) -> Optional[datetime]:
        try:
            return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)
        except ValueError:
            return None

    def _resolve_run_risk_score(self, run: dict) -> Optional[float]:
        risk_score = run.get("risk_score")
        if isinstance(risk_score, dict) and risk_score.get("total") is not None:
            try:
                return float(risk_score["total"])
            except (TypeError, ValueError):
                return None

        risk_level = str(run.get("risk_level", "")).strip().lower()
        risk_by_level = {"low": 25.0, "medium": 60.0, "high": 90.0}
        return risk_by_level.get(risk_level)

    def _resolve_run_spend(self, run: dict) -> float:
        for key in ("spend_usd", "cost_usd", "spend", "cost"):
            value = run.get(key)
            if value is None:
                continue
            try:
                return float(value)
            except (TypeError, ValueError):
                return 0.0
        return 0.0

    def _summarize_version_change(
        self,
        previous: VersionedArtifact,
        current: VersionedArtifact,
        preview_lines: int,
    ) -> VersionChangeSummary:
        diff_lines = list(
            unified_diff(
                previous.content.splitlines(),
                current.content.splitlines(),
                fromfile=previous.version,
                tofile=current.version,
                lineterm="",
            )
        )
        additions = sum(1 for line in diff_lines if line.startswith("+") and not line.startswith("+++"))
        deletions = sum(1 for line in diff_lines if line.startswith("-") and not line.startswith("---"))
        preview = [line for line in diff_lines if not line.startswith("@@")][:preview_lines]
        return VersionChangeSummary(
            from_version=previous.version,
            to_version=current.version,
            additions=additions,
            deletions=deletions,
            changed_lines=additions + deletions,
            preview=preview,
        )

    def _build_funnel(self, status_counts: Dict[str, int], total_runs: int) -> List[EngineeringFunnelStage]:
        funnel_counts = [
            ("queued", status_counts.get("queued", 0)),
            ("in-progress", status_counts.get("running", 0) + status_counts.get("in-progress", 0)),
            ("awaiting-approval", status_counts.get("needs-approval", 0)),
            ("completed", sum(count for status, count in status_counts.items() if status in STATUS_COMPLETE)),
        ]
        return [
            EngineeringFunnelStage(
                name=name,
                count=count,
                share=round((count / total_runs) * 100, 1) if total_runs else 0.0,
            )
            for name, count in funnel_counts
        ]

    def _build_recent_activities(self, runs: Sequence[dict], limit: int) -> List[EngineeringActivity]:
        dated_runs = []
        for run in runs:
            sort_key = self._parse_ts(str(run.get("ended_at", ""))) or self._parse_ts(str(run.get("started_at", "")))
            if sort_key is None:
                continue
            dated_runs.append((sort_key, run))

        activities: List[EngineeringActivity] = []
        for _, run in sorted(dated_runs, key=lambda item: item[0], reverse=True)[:limit]:
            activities.append(
                EngineeringActivity(
                    timestamp=str(run.get("ended_at") or run.get("started_at") or ""),
                    run_id=str(run.get("run_id", "")),
                    task_id=str(run.get("task_id", "")),
                    status=str(run.get("status", "unknown")),
                    summary=self._primary_reason(run),
                )
            )
        return activities

    def _permissions_for_role(self, viewer_role: str) -> EngineeringOverviewPermission:
        role = viewer_role.strip().lower() or "contributor"
        modules_by_role = {
            "executive": ["kpis", "funnel", "blockers"],
            "engineering-manager": ["kpis", "funnel", "blockers", "activity"],
            "operations": ["kpis", "funnel", "blockers", "activity"],
            "contributor": ["kpis", "activity"],
        }
        return EngineeringOverviewPermission(
            viewer_role=role,
            allowed_modules=modules_by_role.get(role, modules_by_role["contributor"]),
        )

    def _owner_for_cluster(self, cluster: TriageCluster) -> str:
        details = " ".join([cluster.reason, " ".join(cluster.statuses)]).lower()
        if "approval" in details:
            return "operations"
        if "security" in details:
            return "security"
        return "engineering"

    def _severity_for_cluster(self, cluster: TriageCluster) -> str:
        if cluster.occurrences >= 3 or "failed" in cluster.statuses:
            return "high"
        return "medium"

    @staticmethod
    def _placements_overlap(left: DashboardWidgetPlacement, right: DashboardWidgetPlacement) -> bool:
        return not (
            left.column + left.width <= right.column
            or right.column + right.width <= left.column
            or left.row + left.height <= right.row
            or right.row + right.height <= left.row
        )


def render_operations_dashboard(
    snapshot: OperationsSnapshot,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Operations Dashboard",
        "",
        f"- Total Runs: {snapshot.total_runs}",
        f"- Success Rate: {snapshot.success_rate:.1f}%",
        f"- Approval Queue Depth: {snapshot.approval_queue_depth}",
        f"- SLA Target: {snapshot.sla_target_minutes} minutes",
        f"- SLA Breaches: {snapshot.sla_breach_count}",
        f"- Average Cycle Time: {snapshot.average_cycle_minutes:.1f} minutes",
        "",
        "## Status Counts",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if snapshot.status_counts:
        for status, count in sorted(snapshot.status_counts.items()):
            lines.append(f"- {status}: {count}")
    else:
        lines.append("- None")

    lines.extend(["", "## Top Blockers", ""])
    if snapshot.top_blockers:
        for cluster in snapshot.top_blockers:
            statuses = ", ".join(cluster.statuses) if cluster.statuses else "unknown"
            lines.append(
                f"- {cluster.reason}: occurrences={cluster.occurrences} statuses={statuses} tasks={', '.join(cluster.task_ids)}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_weekly_operations_report(report: WeeklyOperationsReport) -> str:
    lines = [
        "# Weekly Operations Report",
        "",
        f"- Name: {report.name}",
        f"- Period: {report.period}",
        f"- Total Runs: {report.snapshot.total_runs}",
        f"- Success Rate: {report.snapshot.success_rate:.1f}%",
        f"- SLA Breaches: {report.snapshot.sla_breach_count}",
        f"- Approval Queue Depth: {report.snapshot.approval_queue_depth}",
        "",
        "## Blockers",
        "",
    ]

    if report.snapshot.top_blockers:
        for cluster in report.snapshot.top_blockers:
            lines.append(f"- {cluster.reason}: {cluster.occurrences} runs")
    else:
        lines.append("- None")

    lines.extend(["", "## Regressions", ""])
    if report.regressions:
        for finding in report.regressions:
            lines.append(
                f"- {finding.case_id}: severity={finding.severity} delta={finding.delta} summary={finding.summary}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_operations_metric_spec(spec: OperationsMetricSpec) -> str:
    lines = [
        "# Operations Metric Spec",
        "",
        f"- Name: {spec.name}",
        f"- Generated At: {spec.generated_at}",
        f"- Period Start: {spec.period_start}",
        f"- Period End: {spec.period_end}",
        f"- Timezone: {spec.timezone_name}",
        "",
        "## Definitions",
        "",
    ]

    for definition in spec.definitions:
        lines.extend(
            [
                f"### {definition.label}",
                "",
                f"- Metric ID: {definition.metric_id}",
                f"- Unit: {definition.unit}",
                f"- Direction: {definition.direction}",
                f"- Formula: {definition.formula}",
                f"- Description: {definition.description}",
                f"- Source Fields: {', '.join(definition.source_fields)}",
                "",
            ]
        )

    lines.extend(["## Values", ""])
    for value in spec.values:
        evidence = " | ".join(value.evidence) if value.evidence else "none"
        lines.append(
            f"- {value.label}: value={value.display_value} numerator={value.numerator:.1f} "
            f"denominator={value.denominator:.1f} unit={value.unit} evidence={evidence}"
        )

    return "\n".join(lines) + "\n"


def render_queue_control_center(
    center: QueueControlCenter,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Queue Control Center",
        "",
        f"- Queue Depth: {center.queue_depth}",
        f"- Waiting Approval Runs: {center.waiting_approval_runs}",
        f"- Queued Tasks: {', '.join(center.queued_tasks) if center.queued_tasks else 'none'}",
        "",
        "## Queue By Priority",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    for priority, count in center.queued_by_priority.items():
        lines.append(f"- {priority}: {count}")

    lines.extend(["", "## Queue By Risk", ""])
    for risk_level, count in center.queued_by_risk.items():
        lines.append(f"- {risk_level}: {count}")

    lines.extend(["", "## Execution Media", ""])
    if center.execution_media:
        for medium, count in sorted(center.execution_media.items()):
            lines.append(f"- {medium}: {count}")
    else:
        lines.append("- None")

    lines.extend(["", "## Blocked Tasks", ""])
    if center.blocked_tasks:
        for task_id in center.blocked_tasks:
            lines.append(f"- {task_id}")
    else:
        lines.append("- None")

    lines.extend(["", "## Actions", ""])
    if center.actions:
        for task_id in center.queued_tasks:
            actions = center.actions.get(task_id, [])
            lines.append(f"- {task_id}: {render_console_actions(actions)}")
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_policy_prompt_version_center(
    center: PolicyPromptVersionCenter,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Policy/Prompt Version Center",
        "",
        f"- Name: {center.name}",
        f"- Generated At: {center.generated_at}",
        f"- Versioned Artifacts: {center.artifact_count}",
        f"- Rollback Ready Artifacts: {center.rollback_ready_count}",
        "",
        "## Artifact Histories",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if not center.histories:
        lines.append("- None")
        return "\n".join(lines) + "\n"

    for history in center.histories:
        lines.extend(
            [
                f"### {history.artifact_type} / {history.artifact_id}",
                "",
                f"- Current Version: {history.current_version}",
                f"- Updated At: {history.current_updated_at}",
                f"- Updated By: {history.current_author}",
                f"- Summary: {history.current_summary}",
                f"- Revision Count: {history.revision_count}",
                f"- Rollback Version: {history.rollback_version or 'none'}",
                f"- Rollback Ready: {history.rollback_ready}",
            ]
        )
        if history.change_summary is not None:
            lines.append(
                f"- Diff Summary: {history.change_summary.additions} additions, "
                f"{history.change_summary.deletions} deletions"
            )
        lines.extend(["", "#### Revision History", ""])
        for revision in history.revisions:
            ticket = revision.change_ticket or "none"
            lines.append(
                f"- {revision.version}: updated_at={revision.updated_at} author={revision.author} "
                f"ticket={ticket} summary={revision.summary}"
            )
        lines.extend(["", "#### Diff Preview", ""])
        if history.change_summary is not None and history.change_summary.preview:
            lines.append("```diff")
            lines.extend(history.change_summary.preview)
            lines.append("```")
        else:
            lines.append("- None")
        lines.append("")

    return "\n".join(lines) + "\n"


def render_engineering_overview(overview: EngineeringOverview) -> str:
    lines = [
        "# Engineering Overview",
        "",
        f"- Name: {overview.name}",
        f"- Period: {overview.period}",
        f"- Viewer Role: {overview.permissions.viewer_role}",
        f"- Visible Modules: {', '.join(overview.permissions.allowed_modules)}",
    ]

    if overview.permissions.can_view("kpis"):
        lines.extend(["", "## KPI Modules", ""])
        for kpi in overview.kpis:
            lines.append(
                f"- {kpi.name}: value={kpi.value:.1f}{kpi.unit} target={kpi.target:.1f}{kpi.unit} healthy={kpi.healthy}"
            )

    if overview.permissions.can_view("funnel"):
        lines.extend(["", "## Funnel Modules", ""])
        for stage in overview.funnel:
            lines.append(f"- {stage.name}: count={stage.count} share={stage.share:.1f}%")

    if overview.permissions.can_view("blockers"):
        lines.extend(["", "## Blocker Modules", ""])
        if overview.blockers:
            for blocker in overview.blockers:
                lines.append(
                    f"- {blocker.summary}: severity={blocker.severity} owner={blocker.owner} "
                    f"affected_runs={blocker.affected_runs} tasks={', '.join(blocker.affected_tasks)}"
                )
        else:
            lines.append("- None")

    if overview.permissions.can_view("activity"):
        lines.extend(["", "## Activity Modules", ""])
        if overview.activities:
            for activity in overview.activities:
                lines.append(
                    f"- {activity.timestamp}: {activity.run_id} task={activity.task_id} "
                    f"status={activity.status} summary={activity.summary}"
                )
        else:
            lines.append("- None")

    return "\n".join(lines) + "\n"


def render_dashboard_builder_report(
    dashboard: DashboardBuilder,
    audit: DashboardBuilderAudit,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Dashboard Builder",
        "",
        f"- Name: {dashboard.name}",
        f"- Period: {dashboard.period}",
        f"- Owner: {dashboard.owner}",
        f"- Viewer Role: {dashboard.permissions.viewer_role}",
        f"- Available Widgets: {len(dashboard.widgets)}",
        f"- Layouts: {len(dashboard.layouts)}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Governance",
        "",
        f"- Documentation Complete: {audit.documentation_complete}",
        f"- Duplicate Placement IDs: {', '.join(audit.duplicate_placement_ids) if audit.duplicate_placement_ids else 'none'}",
        f"- Missing Widget Definitions: {', '.join(audit.missing_widget_defs) if audit.missing_widget_defs else 'none'}",
        f"- Inaccessible Widgets: {', '.join(audit.inaccessible_widgets) if audit.inaccessible_widgets else 'none'}",
        f"- Overlaps: {', '.join(audit.overlapping_placements) if audit.overlapping_placements else 'none'}",
        f"- Out Of Bounds: {', '.join(audit.out_of_bounds_placements) if audit.out_of_bounds_placements else 'none'}",
        f"- Empty Layouts: {', '.join(audit.empty_layouts) if audit.empty_layouts else 'none'}",
        "",
        "## Layouts",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if dashboard.layouts:
        for layout in dashboard.layouts:
            lines.append(f"- {layout.layout_id}: name={layout.name} columns={layout.columns} placements={len(layout.placements)}")
            for placement in layout.placements:
                widget = dashboard.widget_index.get(placement.widget_id)
                title = placement.title_override or (widget.title if widget is not None else placement.widget_id)
                filters = ", ".join(placement.filters) if placement.filters else "none"
                lines.append(
                    f"- {placement.placement_id}: widget={placement.widget_id} title={title} "
                    f"grid=({placement.column},{placement.row}) size={placement.width}x{placement.height} filters={filters}"
                )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def write_engineering_overview_bundle(root_dir: str, overview: EngineeringOverview) -> str:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    overview_path = str(base / "engineering-overview.md")
    write_report(overview_path, render_engineering_overview(overview))
    return overview_path


def write_dashboard_builder_bundle(
    root_dir: str,
    dashboard: DashboardBuilder,
    audit: DashboardBuilderAudit,
    view: Optional[SharedViewContext] = None,
) -> str:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    dashboard_path = str(base / "dashboard-builder.md")
    write_report(dashboard_path, render_dashboard_builder_report(dashboard, audit, view=view))
    return dashboard_path




def build_repo_collaboration_metrics(runs: Sequence[dict]) -> Dict[str, float]:
    total = len(runs)
    linked = 0
    accepted = 0
    discussion_posts = 0
    lineage_depth_sum = 0
    lineage_depth_count = 0

    for run in runs:
        links = run.get("closeout", {}).get("run_commit_links", [])
        if links:
            linked += 1
        if run.get("closeout", {}).get("accepted_commit_hash"):
            accepted += 1
        discussion_posts += int(run.get("repo_discussion_posts", 0))

        depth = run.get("accepted_lineage_depth")
        if depth is not None:
            lineage_depth_sum += float(depth)
            lineage_depth_count += 1

    return {
        "repo_link_coverage": round((linked / total) * 100, 1) if total else 0.0,
        "accepted_commit_rate": round((accepted / total) * 100, 1) if total else 0.0,
        "discussion_density": round(discussion_posts / total, 2) if total else 0.0,
        "accepted_lineage_depth_avg": round(lineage_depth_sum / lineage_depth_count, 2) if lineage_depth_count else 0.0,
    }


def write_weekly_operations_bundle(
    root_dir: str,
    report: WeeklyOperationsReport,
    metric_spec: Optional[OperationsMetricSpec] = None,
    regression_center: Optional[RegressionCenter] = None,
    queue_control_center: Optional[QueueControlCenter] = None,
    version_center: Optional[PolicyPromptVersionCenter] = None,
) -> WeeklyOperationsArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)

    weekly_report_path = str(base / "weekly-operations.md")
    dashboard_path = str(base / "operations-dashboard.md")
    write_report(weekly_report_path, render_weekly_operations_report(report))
    write_report(dashboard_path, render_operations_dashboard(report.snapshot))

    metric_spec_path = None
    if metric_spec is not None:
        metric_spec_path = str(base / "operations-metric-spec.md")
        write_report(metric_spec_path, render_operations_metric_spec(metric_spec))

    regression_center_path = None
    if regression_center is not None:
        regression_center_path = str(base / "regression-center.md")
        write_report(regression_center_path, render_regression_center(regression_center))

    queue_control_path = None
    if queue_control_center is not None:
        queue_control_path = str(base / "queue-control-center.md")
        write_report(queue_control_path, render_queue_control_center(queue_control_center))

    version_center_path = None
    if version_center is not None:
        version_center_path = str(base / "policy-prompt-version-center.md")
        write_report(version_center_path, render_policy_prompt_version_center(version_center))

    return WeeklyOperationsArtifacts(
        root_dir=str(base),
        weekly_report_path=weekly_report_path,
        dashboard_path=dashboard_path,
        metric_spec_path=metric_spec_path,
        regression_center_path=regression_center_path,
        queue_control_path=queue_control_path,
        version_center_path=version_center_path,
    )


def render_regression_center(
    center: RegressionCenter,
    view: Optional[SharedViewContext] = None,
) -> str:
    lines = [
        "# Regression Analysis Center",
        "",
        f"- Name: {center.name}",
        f"- Baseline Version: {center.baseline_version}",
        f"- Current Version: {center.current_version}",
        f"- Regressions: {center.regression_count}",
        f"- Improved Cases: {len(center.improved_cases)}",
        f"- Unchanged Cases: {len(center.unchanged_cases)}",
        "",
        "## Regressions",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if center.regressions:
        for finding in center.regressions:
            lines.append(
                f"- {finding.case_id}: severity={finding.severity} delta={finding.delta} summary={finding.summary}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Improved Cases", ""])
    if center.improved_cases:
        for case_id in center.improved_cases:
            lines.append(f"- {case_id}")
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"
