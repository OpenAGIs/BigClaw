import json
import sys
import types
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Protocol

from .models import (
    BillingInterval,
    BillingRate,
    BillingSummary,
    FlowRun,
    FlowRunStatus,
    FlowStepRun,
    FlowStepStatus,
    FlowTemplate,
    FlowTemplateStep,
    FlowTrigger,
    Priority,
    RiskAssessment,
    RiskLevel,
    RiskSignal,
    Task,
    TaskState,
    TriageLabel,
    TriageRecord,
    TriageStatus,
    UsageRecord,
)
from . import runtime as _legacy_runtime_surface


@dataclass
class SourceIssue:
    source: str
    source_id: str
    title: str
    description: str
    labels: List[str]
    priority: str
    state: str
    links: Dict[str, str]


class Connector(Protocol):
    name: str

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        ...


class GitHubConnector:
    name = "github"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="github",
                source_id=f"{project}#1",
                title="Fix flaky test",
                description="CI flaky on macOS",
                labels=["bug", "ci"],
                priority="P1",
                state=states[0] if states else "Todo",
                links={"issue": f"https://github.com/{project}/issues/1"},
            )
        ]


class LinearConnector:
    name = "linear"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="linear",
                source_id=f"{project}-101",
                title="Implement queue persistence",
                description="Need restart-safe queue",
                labels=["platform"],
                priority="P0",
                state=states[0] if states else "Todo",
                links={"issue": f"https://linear.app/{project}/issue/{project}-101"},
            )
        ]


class JiraConnector:
    name = "jira"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="jira",
                source_id=f"{project}-23",
                title="Runbook automation",
                description="Automate oncall runbook",
                labels=["ops"],
                priority="P2",
                state=states[0] if states else "Todo",
                links={"issue": f"https://jira.example.com/browse/{project}-23"},
            )
        ]


def map_priority(priority: str) -> Priority:
    priority = (priority or "").upper()
    if priority == "P0":
        return Priority.P0
    if priority == "P1":
        return Priority.P1
    return Priority.P2


def map_state(state: str) -> TaskState:
    normalized = (state or "").lower()
    if "progress" in normalized:
        return TaskState.IN_PROGRESS
    if "done" in normalized or "closed" in normalized:
        return TaskState.DONE
    if "block" in normalized:
        return TaskState.BLOCKED
    return TaskState.TODO


def map_source_issue_to_task(issue: SourceIssue) -> Task:
    risk_level = RiskLevel.HIGH if "prod" in issue.title.lower() else RiskLevel.LOW
    return Task(
        task_id=issue.source_id,
        source=issue.source,
        title=issue.title,
        description=issue.description,
        labels=issue.labels,
        priority=map_priority(issue.priority),
        state=map_state(issue.state),
        risk_level=risk_level,
        required_tools=["github" if issue.source == "github" else "connector"],
        acceptance_criteria=["Synced from source issue"],
        validation_plan=["mapping-test"],
    )


_VALID_WORKFLOW_STEP_KINDS = {
    "scheduler",
    "approval",
    "orchestration",
    "report",
    "closeout",
}


@dataclass
class WorkflowStep:
    name: str
    kind: str
    required: bool = True
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "kind": self.kind,
            "required": self.required,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WorkflowStep":
        return cls(
            name=data["name"],
            kind=data["kind"],
            required=data.get("required", True),
            metadata=data.get("metadata", {}),
        )


@dataclass
class WorkflowDefinition:
    name: str
    steps: List[WorkflowStep] = field(default_factory=list)
    report_path_template: Optional[str] = None
    journal_path_template: Optional[str] = None
    validation_evidence: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "steps": [step.to_dict() for step in self.steps],
            "report_path_template": self.report_path_template,
            "journal_path_template": self.journal_path_template,
            "validation_evidence": self.validation_evidence,
            "approvals": self.approvals,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WorkflowDefinition":
        return cls(
            name=data["name"],
            steps=[WorkflowStep.from_dict(item) for item in data.get("steps", [])],
            report_path_template=data.get("report_path_template"),
            journal_path_template=data.get("journal_path_template"),
            validation_evidence=data.get("validation_evidence", []),
            approvals=data.get("approvals", []),
        )

    @classmethod
    def from_json(cls, text: str) -> "WorkflowDefinition":
        return cls.from_dict(json.loads(text))

    def render_path(self, template: Optional[str], task: Task, run_id: str) -> Optional[str]:
        if template is None:
            return None
        return template.format(
            workflow=self.name,
            task_id=task.task_id,
            source=task.source,
            run_id=run_id,
        )

    def render_report_path(self, task: Task, run_id: str) -> Optional[str]:
        return self.render_path(self.report_path_template, task, run_id)

    def render_journal_path(self, task: Task, run_id: str) -> Optional[str]:
        return self.render_path(self.journal_path_template, task, run_id)

    def validate(self) -> None:
        invalid_steps = [step.kind for step in self.steps if step.kind not in _VALID_WORKFLOW_STEP_KINDS]
        if invalid_steps:
            joined = ", ".join(sorted(set(invalid_steps)))
            raise ValueError(f"invalid workflow step kind(s): {joined}")


def _install_legacy_surface_module(name: str, export_names: list[str], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    for export_name in export_names:
        module.__dict__[export_name] = getattr(_legacy_runtime_surface, export_name)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


def _install_compatibility_module(name: str, exports: Dict[str, object], **extra_attrs: object) -> None:
    module = types.ModuleType(f"{__name__}.{name}")
    module.__dict__.update(exports)
    module.__dict__.update(extra_attrs)
    sys.modules[module.__name__] = module
    globals()[name] = module


_install_legacy_surface_module(
    "queue",
    ["DeadLetterEntry", "PersistentTaskQueue"],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/queue/queue.go",
)
_install_legacy_surface_module(
    "orchestration",
    [
        "CrossDepartmentOrchestrator",
        "DepartmentHandoff",
        "HandoffRequest",
        "OrchestrationPlan",
        "OrchestrationPolicyDecision",
        "PremiumOrchestrationPolicy",
        "render_orchestration_plan",
    ],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/orchestration.go",
)
_install_legacy_surface_module(
    "scheduler",
    ["ExecutionRecord", "Scheduler", "SchedulerDecision"],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/scheduler/scheduler.go",
)
_install_legacy_surface_module(
    "workflow",
    ["AcceptanceDecision", "AcceptanceGate", "JournalEntry", "WorkflowEngine", "WorkflowRunResult", "WorkpadJournal"],
    LEGACY_MAINLINE_STATUS=_legacy_runtime_surface.LEGACY_MAINLINE_STATUS,
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/engine.go",
)
_install_legacy_surface_module(
    "service",
    [
        "RepoGovernanceEnforcer",
        "RepoGovernancePolicy",
        "RepoGovernanceResult",
        "ServerMonitoring",
        "create_server",
        "run_server",
        "warn_legacy_service_surface",
    ],
    LEGACY_MAINLINE_STATUS=(
        "bigclaw-go is the sole implementation mainline for active development; "
        "service.py remains migration-only compatibility scaffolding."
    ),
    GO_MAINLINE_REPLACEMENT="bigclaw-go/cmd/bigclawd/main.go",
)
_install_compatibility_module(
    "connectors",
    {
        "SourceIssue": SourceIssue,
        "Connector": Connector,
        "GitHubConnector": GitHubConnector,
        "LinearConnector": LinearConnector,
        "JiraConnector": JiraConnector,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/connector.go",
)
_install_compatibility_module(
    "mapping",
    {
        "map_priority": map_priority,
        "map_state": map_state,
        "map_source_issue_to_task": map_source_issue_to_task,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/intake/mapping.go",
)
_install_compatibility_module(
    "dsl",
    {
        "WorkflowDefinition": WorkflowDefinition,
        "WorkflowStep": WorkflowStep,
    },
    GO_MAINLINE_REPLACEMENT="bigclaw-go/internal/workflow/definition.go",
)

from .runtime import (
    AcceptanceDecision,
    AcceptanceGate,
    ClawWorkerRuntime,
    CrossDepartmentOrchestrator,
    DeadLetterEntry,
    DepartmentHandoff,
    ExecutionRecord,
    HandoffRequest,
    JournalEntry,
    OrchestrationPlan,
    OrchestrationPolicyDecision,
    PersistentTaskQueue,
    PremiumOrchestrationPolicy,
    RepoGovernanceEnforcer,
    RepoGovernancePolicy,
    RepoGovernanceResult,
    SandboxProfile,
    SandboxRouter,
    Scheduler,
    SchedulerDecision,
    ServerMonitoring,
    ToolCallResult,
    ToolPolicy,
    ToolRuntime,
    WorkerExecutionResult,
    WorkflowEngine,
    WorkflowRunResult,
    WorkpadJournal,
    create_server,
    render_orchestration_plan,
    run_server,
    warn_legacy_service_surface,
)
from .design_system import (
    AuditRequirement,
    CommandAction,
    ComponentLibrary,
    ComponentSpec,
    ComponentVariant,
    ConsoleChromeLibrary,
    ConsoleCommandEntry,
    ConsoleTopBar,
    ConsoleTopBarAudit,
    DataAccuracyCheck,
    DesignSystem,
    DesignSystemAudit,
    DesignToken,
    InformationArchitecture,
    InformationArchitectureAudit,
    NavigationEntry,
    NavigationNode,
    NavigationRoute,
    PerformanceBudget,
    RolePermissionScenario,
    UIAcceptanceAudit,
    UIAcceptanceLibrary,
    UIAcceptanceSuite,
    UsabilityJourney,
    render_console_top_bar_report,
    render_design_system_report,
    render_information_architecture_report,
    render_ui_acceptance_report,
)
from .console_ia import (
    ConsoleIA,
    ConsoleIAAudit,
    ConsoleIAAuditor,
    ConsoleInteractionAudit,
    ConsoleInteractionAuditor,
    ConsoleInteractionDraft,
    ConsoleSurface,
    FilterDefinition,
    GlobalAction,
    NavigationItem,
    SurfaceInteractionContract,
    SurfacePermissionRule,
    SurfaceState,
    build_big_4203_console_interaction_draft,
    render_console_interaction_report,
    render_console_ia_report,
)
from .collaboration import (
    CollaborationComment,
    CollaborationThread,
    DecisionNote,
    build_collaboration_thread,
    build_collaboration_thread_from_audits,
)
from .saved_views import (
    AlertDigestSubscription,
    SavedView,
    SavedViewCatalog,
    SavedViewCatalogAudit,
    SavedViewFilter,
    SavedViewLibrary,
    render_saved_view_report,
)
from .governance import (
    FreezeException,
    GovernanceBacklogItem,
    ScopeFreezeAudit,
    ScopeFreezeBoard,
    ScopeFreezeGovernance,
    render_scope_freeze_report,
)
from .issue_archive import (
    ArchivedIssue,
    IssuePriorityArchive,
    IssuePriorityArchiveAudit,
    IssuePriorityArchivist,
    render_issue_priority_archive_report,
)
from .risk import RiskFactor, RiskScore, RiskScorer
from .roadmap import EpicMilestone, ExecutionPackRoadmap, build_execution_pack_roadmap
from .audit_events import (
    APPROVAL_RECORDED_EVENT,
    BUDGET_OVERRIDE_EVENT,
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    P0_AUDIT_EVENT_SPECS,
    SCHEDULER_DECISION_EVENT,
    AuditEventSpec,
    get_audit_event_spec,
    missing_required_fields,
)
from .event_bus import (
    CI_COMPLETED_EVENT,
    PULL_REQUEST_COMMENT_EVENT,
    TASK_FAILED_EVENT,
    BusEvent,
    EventBus,
)
from .observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit, RunCloseout, TaskRun
from .execution_contract import (
    AuditPolicy,
    build_operations_api_contract,
    ExecutionApiSpec,
    ExecutionContract,
    ExecutionContractAudit,
    ExecutionContractLibrary,
    ExecutionField,
    ExecutionModel,
    ExecutionPermission,
    ExecutionPermissionMatrix,
    ExecutionRole,
    MetricDefinition,
    PermissionCheckResult,
    render_execution_contract_report,
)
from .dashboard_run_contract import (
    DashboardRunContract,
    DashboardRunContractAudit,
    DashboardRunContractLibrary,
    SchemaField,
    SurfaceSchema,
    render_dashboard_run_contract_report,
)
from .reports import (
    AutoTriageCenter,
    ConsoleAction,
    BillingEntitlementsPage,
    BillingRunCharge,
    DocumentationArtifact,
    FinalDeliveryChecklist,
    LaunchChecklist,
    LaunchChecklistItem,
    NarrativeSection,
    IssueClosureDecision,
    OrchestrationCanvas,
    OrchestrationPortfolio,
    PilotMetric,
    PilotPortfolio,
    PilotScorecard,
    ReportStudio,
    ReportStudioArtifacts,
    SharedViewContext,
    SharedViewFilter,
    TakeoverQueue,
    TakeoverRequest,
    TriageFeedbackRecord,
    TriageFinding,
    TriageInboxItem,
    TriageSimilarityEvidence,
    TriageSuggestion,
    build_auto_triage_center,
    build_console_actions,
    build_billing_entitlements_page,
    build_billing_entitlements_page_from_ledger,
    build_final_delivery_checklist,
    build_launch_checklist,
    build_orchestration_canvas,
    build_orchestration_canvas_from_ledger_entry,
    build_orchestration_portfolio,
    build_orchestration_portfolio_from_ledger,
    build_takeover_queue_from_ledger,
    evaluate_issue_closure,
    render_auto_triage_center_report,
    render_console_actions,
    render_billing_entitlements_page,
    render_billing_entitlements_report,
    render_final_delivery_checklist_report,
    render_launch_checklist_report,
    render_orchestration_canvas,
    render_orchestration_overview_page,
    render_orchestration_portfolio_report,
    render_issue_validation_report,
    render_report_studio_html,
    render_report_studio_plain_text,
    render_report_studio_report,
    render_takeover_queue_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    render_repo_sync_audit_report,
    render_task_run_detail_page,
    render_task_run_report,
    validation_report_exists,
    write_report,
    write_report_studio_bundle,
)
from .operations import (
    DashboardBuilder,
    DashboardBuilderAudit,
    DashboardLayout,
    DashboardWidgetPlacement,
    DashboardWidgetSpec,
    EngineeringActivity,
    EngineeringFunnelStage,
    EngineeringOverview,
    EngineeringOverviewBlocker,
    EngineeringOverviewKPI,
    EngineeringOverviewPermission,
    OperationsAnalytics,
    OperationsMetricDefinition,
    OperationsMetricSpec,
    OperationsMetricValue,
    OperationsSnapshot,
    PolicyPromptVersionCenter,
    RegressionFinding,
    RegressionCenter,
    TriageCluster,
    QueueControlCenter,
    VersionChangeSummary,
    VersionedArtifact,
    VersionedArtifactHistory,
    WeeklyOperationsArtifacts,
    WeeklyOperationsReport,
    render_dashboard_builder_report,
    render_engineering_overview,
    render_operations_metric_spec,
    render_operations_dashboard,
    render_policy_prompt_version_center,
    render_queue_control_center,
    render_regression_center,
    render_weekly_operations_report,
    write_dashboard_builder_bundle,
    write_engineering_overview_bundle,
    write_weekly_operations_bundle,
)
from .evaluation import (
    BenchmarkCase,
    BenchmarkComparison,
    BenchmarkResult,
    BenchmarkRunner,
    BenchmarkSuiteResult,
    EvaluationCriterion,
    ReplayOutcome,
    ReplayRecord,
    render_run_replay_index_page,
    render_replay_detail_page,
    render_benchmark_suite_report,
)
from .planning import (
    FourWeekExecutionPlan,
    CandidateBacklog,
    CandidateEntry,
    CandidatePlanner,
    EvidenceLink,
    EntryGate,
    EntryGateDecision,
    WeeklyExecutionPlan,
    WeeklyGoal,
    build_big_4701_execution_plan,
    build_v3_candidate_backlog,
    build_v3_entry_gate,
    render_candidate_backlog_report,
    render_four_week_execution_report,
)
from .ui_review import (
    InteractionFlow,
    OpenQuestion,
    ReviewBlocker,
    ReviewBlockerEvent,
    ReviewDecision,
    ReviewObjective,
    ReviewRoleAssignment,
    ReviewSignoff,
    ReviewerChecklistItem,
    UIReviewPack,
    UIReviewPackArtifacts,
    UIReviewPackAudit,
    UIReviewPackAuditor,
    WireframeSurface,
    build_big_4204_review_pack,
    render_ui_review_blocker_log,
    render_ui_review_blocker_timeline,
    render_ui_review_blocker_timeline_summary,
    render_ui_review_escalation_dashboard,
    render_ui_review_escalation_handoff_ledger,
    render_ui_review_exception_log,
    render_ui_review_exception_matrix,
    render_ui_review_freeze_approval_trail,
    render_ui_review_freeze_exception_board,
    render_ui_review_freeze_renewal_tracker,
    render_ui_review_handoff_ack_ledger,
    render_ui_review_interaction_coverage_board,
    render_ui_review_objective_coverage_board,
    render_ui_review_open_question_tracker,
    render_ui_review_owner_escalation_digest,
    render_ui_review_persona_readiness_board,
    render_ui_review_review_summary_board,
    render_ui_review_owner_review_queue,
    render_ui_review_owner_workload_board,
    render_ui_review_checklist_traceability_board,
    render_ui_review_decision_followup_tracker,
    render_ui_review_audit_density_board,
    render_ui_review_reminder_cadence_board,
    render_ui_review_role_coverage_board,
    render_ui_review_wireframe_readiness_board,
    render_ui_review_signoff_breach_board,
    render_ui_review_signoff_dependency_board,
    render_ui_review_signoff_reminder_queue,
    render_ui_review_signoff_sla_dashboard,
    render_ui_review_decision_log,
    render_ui_review_pack_html,
    render_ui_review_role_matrix,
    render_ui_review_signoff_log,
    render_ui_review_pack_report,
    write_ui_review_pack_bundle,
)

__all__ = [
    "Task",
    "TaskState",
    "RiskLevel",
    "Priority",
    "RiskSignal",
    "RiskAssessment",
    "TriageStatus",
    "TriageLabel",
    "TriageRecord",
    "FlowTrigger",
    "FlowRunStatus",
    "FlowStepStatus",
    "FlowTemplateStep",
    "FlowTemplate",
    "FlowStepRun",
    "FlowRun",
    "BillingInterval",
    "BillingRate",
    "UsageRecord",
    "BillingSummary",
    "PersistentTaskQueue",
    "Scheduler",
    "SchedulerDecision",
    "ExecutionRecord",
    "SourceIssue",
    "GitHubConnector",
    "LinearConnector",
    "JiraConnector",
    "CommandAction",
    "AuditRequirement",
    "ComponentLibrary",
    "ComponentSpec",
    "ComponentVariant",
    "ConsoleChromeLibrary",
    "ConsoleCommandEntry",
    "ConsoleTopBar",
    "ConsoleTopBarAudit",
    "DataAccuracyCheck",
    "DesignSystem",
    "DesignSystemAudit",
    "DesignToken",
    "InformationArchitecture",
    "InformationArchitectureAudit",
    "NavigationEntry",
    "NavigationNode",
    "NavigationRoute",
    "PerformanceBudget",
    "RolePermissionScenario",
    "UIAcceptanceAudit",
    "UIAcceptanceLibrary",
    "UIAcceptanceSuite",
    "UsabilityJourney",
    "render_console_top_bar_report",
    "render_design_system_report",
    "render_information_architecture_report",
    "render_ui_acceptance_report",
    "ConsoleIA",
    "ConsoleIAAudit",
    "ConsoleIAAuditor",
    "ConsoleInteractionAudit",
    "ConsoleInteractionAuditor",
    "ConsoleInteractionDraft",
    "ConsoleSurface",
    "FilterDefinition",
    "GlobalAction",
    "NavigationItem",
    "SurfaceInteractionContract",
    "SurfacePermissionRule",
    "SurfaceState",
    "build_big_4203_console_interaction_draft",
    "render_console_interaction_report",
    "render_console_ia_report",
    "AlertDigestSubscription",
    "SavedView",
    "SavedViewCatalog",
    "SavedViewCatalogAudit",
    "SavedViewFilter",
    "SavedViewLibrary",
    "render_saved_view_report",
    "FreezeException",
    "GovernanceBacklogItem",
    "ScopeFreezeAudit",
    "ScopeFreezeBoard",
    "ScopeFreezeGovernance",
    "render_scope_freeze_report",
    "RiskFactor",
    "RiskScore",
    "RiskScorer",
    "CollaborationComment",
    "CollaborationThread",
    "DecisionNote",
    "build_collaboration_thread",
    "build_collaboration_thread_from_audits",
    "WorkflowDefinition",
    "WorkflowStep",
    "map_source_issue_to_task",
    "EpicMilestone",
    "ExecutionPackRoadmap",
    "build_execution_pack_roadmap",
    "APPROVAL_RECORDED_EVENT",
    "BUDGET_OVERRIDE_EVENT",
    "FLOW_HANDOFF_EVENT",
    "MANUAL_TAKEOVER_EVENT",
    "P0_AUDIT_EVENT_SPECS",
    "SCHEDULER_DECISION_EVENT",
    "AuditEventSpec",
    "get_audit_event_spec",
    "missing_required_fields",
    "BusEvent",
    "EventBus",
    "PULL_REQUEST_COMMENT_EVENT",
    "CI_COMPLETED_EVENT",
    "TASK_FAILED_EVENT",
    "ObservabilityLedger",
    "GitSyncTelemetry",
    "PullRequestFreshness",
    "RepoSyncAudit",
    "RunCloseout",
    "TaskRun",
    "CrossDepartmentOrchestrator",
    "DepartmentHandoff",
    "HandoffRequest",
    "OrchestrationPlan",
    "OrchestrationPolicyDecision",
    "PremiumOrchestrationPolicy",
    "render_orchestration_plan",
    "ClawWorkerRuntime",
    "SandboxProfile",
    "SandboxRouter",
    "ToolCallResult",
    "ToolPolicy",
    "ToolRuntime",
    "WorkerExecutionResult",
    "AuditPolicy",
    "ExecutionApiSpec",
    "ExecutionContract",
    "ExecutionContractAudit",
    "ExecutionContractLibrary",
    "ExecutionField",
    "ExecutionModel",
    "ExecutionPermission",
    "ExecutionPermissionMatrix",
    "ExecutionRole",
    "MetricDefinition",
    "PermissionCheckResult",
    "render_execution_contract_report",
    "build_operations_api_contract",
    "SchemaField",
    "SurfaceSchema",
    "DashboardRunContract",
    "DashboardRunContractAudit",
    "DashboardRunContractLibrary",
    "render_dashboard_run_contract_report",
    "AutoTriageCenter",
    "ConsoleAction",
    "BillingEntitlementsPage",
    "BillingRunCharge",
    "DocumentationArtifact",
    "FinalDeliveryChecklist",
    "LaunchChecklist",
    "LaunchChecklistItem",
    "NarrativeSection",
    "IssueClosureDecision",
    "OrchestrationCanvas",
    "OrchestrationPortfolio",
    "PilotMetric",
    "PilotPortfolio",
    "PilotScorecard",
    "ReportStudio",
    "ReportStudioArtifacts",
    "SharedViewContext",
    "SharedViewFilter",
    "TakeoverQueue",
    "TakeoverRequest",
    "TriageFeedbackRecord",
    "TriageFinding",
    "TriageInboxItem",
    "TriageSimilarityEvidence",
    "TriageSuggestion",
    "build_auto_triage_center",
    "build_console_actions",
    "build_billing_entitlements_page",
    "build_billing_entitlements_page_from_ledger",
    "build_final_delivery_checklist",
    "build_launch_checklist",
    "build_orchestration_canvas",
    "build_orchestration_canvas_from_ledger_entry",
    "build_orchestration_portfolio",
    "build_orchestration_portfolio_from_ledger",
    "build_takeover_queue_from_ledger",
    "evaluate_issue_closure",
    "render_auto_triage_center_report",
    "render_console_actions",
    "render_billing_entitlements_page",
    "render_billing_entitlements_report",
    "render_final_delivery_checklist_report",
    "render_launch_checklist_report",
    "render_orchestration_canvas",
    "render_orchestration_overview_page",
    "render_orchestration_portfolio_report",
    "render_issue_validation_report",
    "render_report_studio_html",
    "render_report_studio_plain_text",
    "render_report_studio_report",
    "render_takeover_queue_report",
    "render_pilot_portfolio_report",
    "render_pilot_scorecard",
    "render_repo_sync_audit_report",
    "render_task_run_detail_page",
    "render_task_run_report",
    "validation_report_exists",
    "write_report",
    "write_report_studio_bundle",
    "AcceptanceDecision",
    "AcceptanceGate",
    "WorkflowEngine",
    "WorkflowRunResult",
    "WorkpadJournal",
    "DashboardBuilder",
    "DashboardBuilderAudit",
    "DashboardLayout",
    "DashboardWidgetPlacement",
    "DashboardWidgetSpec",
    "EngineeringActivity",
    "EngineeringFunnelStage",
    "EngineeringOverview",
    "EngineeringOverviewBlocker",
    "EngineeringOverviewKPI",
    "EngineeringOverviewPermission",
    "OperationsAnalytics",
    "OperationsMetricDefinition",
    "OperationsMetricSpec",
    "OperationsMetricValue",
    "OperationsSnapshot",
    "PolicyPromptVersionCenter",
    "RegressionCenter",
    "RegressionFinding",
    "TriageCluster",
    "WeeklyOperationsArtifacts",
    "WeeklyOperationsReport",
    "QueueControlCenter",
    "render_dashboard_builder_report",
    "VersionChangeSummary",
    "VersionedArtifact",
    "VersionedArtifactHistory",
    "render_engineering_overview",
    "render_operations_metric_spec",
    "render_operations_dashboard",
    "render_policy_prompt_version_center",
    "render_queue_control_center",
    "render_regression_center",
    "render_weekly_operations_report",
    "write_dashboard_builder_bundle",
    "write_engineering_overview_bundle",
    "write_weekly_operations_bundle",
    "BenchmarkCase",
    "BenchmarkComparison",
    "BenchmarkResult",
    "BenchmarkRunner",
    "BenchmarkSuiteResult",
    "EvaluationCriterion",
    "ReplayOutcome",
    "ReplayRecord",
    "render_run_replay_index_page",
    "render_replay_detail_page",
    "render_benchmark_suite_report",
    "CandidateBacklog",
    "CandidateEntry",
    "CandidatePlanner",
    "EvidenceLink",
    "EntryGate",
    "EntryGateDecision",
    "FourWeekExecutionPlan",
    "WeeklyExecutionPlan",
    "WeeklyGoal",
    "build_big_4701_execution_plan",
    "build_v3_candidate_backlog",
    "build_v3_entry_gate",
    "render_candidate_backlog_report",
    "render_four_week_execution_report",
    "InteractionFlow",
    "OpenQuestion",
    "ReviewBlocker",
    "ReviewBlockerEvent",
    "ReviewDecision",
    "ReviewObjective",
    "ReviewRoleAssignment",
    "ReviewSignoff",
    "ReviewerChecklistItem",
    "UIReviewPack",
    "UIReviewPackArtifacts",
    "UIReviewPackAudit",
    "UIReviewPackAuditor",
    "WireframeSurface",
    "build_big_4204_review_pack",
    "render_ui_review_blocker_log",
    "render_ui_review_blocker_timeline",
    "render_ui_review_blocker_timeline_summary",
    "render_ui_review_escalation_dashboard",
    "render_ui_review_escalation_handoff_ledger",
    "render_ui_review_exception_log",
    "render_ui_review_exception_matrix",
    "render_ui_review_freeze_approval_trail",
    "render_ui_review_freeze_exception_board",
    "render_ui_review_freeze_renewal_tracker",
    "render_ui_review_handoff_ack_ledger",
    "render_ui_review_interaction_coverage_board",
    "render_ui_review_objective_coverage_board",
    "render_ui_review_open_question_tracker",
    "render_ui_review_owner_escalation_digest",
    "render_ui_review_persona_readiness_board",
    "render_ui_review_review_summary_board",
    "render_ui_review_owner_review_queue",
    "render_ui_review_owner_workload_board",
    "render_ui_review_checklist_traceability_board",
    "render_ui_review_decision_followup_tracker",
    "render_ui_review_audit_density_board",
    "render_ui_review_reminder_cadence_board",
    "render_ui_review_role_coverage_board",
    "render_ui_review_wireframe_readiness_board",
    "render_ui_review_signoff_breach_board",
    "render_ui_review_signoff_dependency_board",
    "render_ui_review_signoff_reminder_queue",
    "render_ui_review_signoff_sla_dashboard",
    "render_ui_review_decision_log",
    "render_ui_review_pack_html",
    "render_ui_review_role_matrix",
    "render_ui_review_signoff_log",
    "render_ui_review_pack_report",
    "write_ui_review_pack_bundle",
]
