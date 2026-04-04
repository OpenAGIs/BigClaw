"""Legacy Python workflow surface frozen after Go mainline cutover."""

import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

from .models import RiskLevel, Task
from .observability import APPROVAL_RECORDED_EVENT, ObservabilityLedger, RepoSyncAudit, utc_now
from .orchestration import render_orchestration_plan
from .reports import (
    build_orchestration_canvas,
    PilotScorecard,
    render_orchestration_canvas,
    render_pilot_scorecard,
    render_repo_sync_audit_report,
    write_report,
)
from .runtime import LEGACY_RUNTIME_GUIDANCE
from .scheduler import ExecutionRecord, Scheduler


LEGACY_MAINLINE_STATUS = LEGACY_RUNTIME_GUIDANCE
GO_MAINLINE_REPLACEMENT = "bigclaw-go/internal/workflow/engine.go"

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


@dataclass
class JournalEntry:
    step: str
    status: str
    timestamp: str = field(default_factory=utc_now)
    details: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "step": self.step,
            "status": self.status,
            "timestamp": self.timestamp,
            "details": self.details,
        }


@dataclass
class WorkpadJournal:
    task_id: str
    run_id: str
    entries: List[JournalEntry] = field(default_factory=list)

    def record(self, step: str, status: str, **details: Any) -> None:
        self.entries.append(JournalEntry(step=step, status=status, details=details))

    def replay(self) -> List[str]:
        return [f"{entry.step}:{entry.status}" for entry in self.entries]

    def to_dict(self) -> Dict[str, Any]:
        return {
            "task_id": self.task_id,
            "run_id": self.run_id,
            "entries": [entry.to_dict() for entry in self.entries],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "WorkpadJournal":
        entries = [
            JournalEntry(
                step=item["step"],
                status=item["status"],
                timestamp=item.get("timestamp", utc_now()),
                details=item.get("details", {}),
            )
            for item in data.get("entries", [])
        ]
        return cls(task_id=data["task_id"], run_id=data["run_id"], entries=entries)

    @classmethod
    def read(cls, path: str) -> "WorkpadJournal":
        payload = json.loads(Path(path).read_text())
        return cls.from_dict(payload)

    def write(self, path: str) -> str:
        output = Path(path)
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps(self.to_dict(), ensure_ascii=False, indent=2))
        return str(output)


@dataclass
class AcceptanceDecision:
    passed: bool
    status: str
    summary: str
    missing_acceptance_criteria: List[str] = field(default_factory=list)
    missing_validation_steps: List[str] = field(default_factory=list)
    approvals: List[str] = field(default_factory=list)


class AcceptanceGate:
    def evaluate(
        self,
        task: Task,
        record: ExecutionRecord,
        validation_evidence: Optional[Sequence[str]] = None,
        approvals: Optional[Sequence[str]] = None,
        pilot_scorecard: Optional[PilotScorecard] = None,
    ) -> AcceptanceDecision:
        evidence = set(validation_evidence or [])
        approval_list = list(approvals or [])

        missing_acceptance = [item for item in task.acceptance_criteria if item not in evidence]
        missing_validation = [item for item in task.validation_plan if item not in evidence]

        needs_manual_approval = (
            task.risk_level == RiskLevel.HIGH or not record.decision.approved or record.run.status == "needs-approval"
        )
        if needs_manual_approval and not approval_list:
            return AcceptanceDecision(
                passed=False,
                status="needs-approval",
                summary="manual approval required before acceptance closure",
                missing_acceptance_criteria=missing_acceptance,
                missing_validation_steps=missing_validation,
                approvals=approval_list,
            )

        if pilot_scorecard is not None and pilot_scorecard.recommendation == "hold":
            return AcceptanceDecision(
                passed=False,
                status="rejected",
                summary="pilot scorecard indicates insufficient ROI or KPI progress",
                missing_acceptance_criteria=missing_acceptance,
                missing_validation_steps=missing_validation,
                approvals=approval_list,
            )

        if missing_acceptance or missing_validation:
            return AcceptanceDecision(
                passed=False,
                status="rejected",
                summary="acceptance evidence incomplete",
                missing_acceptance_criteria=missing_acceptance,
                missing_validation_steps=missing_validation,
                approvals=approval_list,
            )

        return AcceptanceDecision(
            passed=True,
            status="accepted",
            summary="acceptance criteria and validation plan satisfied",
            approvals=approval_list,
        )


@dataclass
class WorkflowRunResult:
    execution: ExecutionRecord
    acceptance: AcceptanceDecision
    journal: WorkpadJournal
    journal_path: Optional[str]
    orchestration_report_path: Optional[str] = None
    orchestration_canvas_path: Optional[str] = None
    pilot_report_path: Optional[str] = None
    repo_sync_report_path: Optional[str] = None


class WorkflowEngine:
    def __init__(self, scheduler: Optional[Scheduler] = None, gate: Optional[AcceptanceGate] = None):
        self.scheduler = scheduler or Scheduler()
        self.gate = gate or AcceptanceGate()

    def run(
        self,
        task: Task,
        run_id: str,
        ledger: ObservabilityLedger,
        report_path: Optional[str] = None,
        journal_path: Optional[str] = None,
        validation_evidence: Optional[Sequence[str]] = None,
        approvals: Optional[Sequence[str]] = None,
        pilot_scorecard: Optional[PilotScorecard] = None,
        pilot_report_path: Optional[str] = None,
        orchestration_report_path: Optional[str] = None,
        orchestration_canvas_path: Optional[str] = None,
        repo_sync_audit: Optional[RepoSyncAudit] = None,
        repo_sync_report_path: Optional[str] = None,
        git_push_succeeded: bool = False,
        git_push_output: str = "",
        git_log_stat_output: str = "",
    ) -> WorkflowRunResult:
        journal = WorkpadJournal(task_id=task.task_id, run_id=run_id)
        journal.record("intake", "recorded", source=task.source)

        execution = self.scheduler.execute(
            task,
            run_id=run_id,
            ledger=ledger,
            report_path=report_path,
            actor="workflow-engine",
        )
        journal.record(
            "execution",
            execution.run.status,
            medium=execution.decision.medium,
            approved=execution.decision.approved,
        )

        resolved_orchestration_report_path = None
        resolved_orchestration_canvas_path = None
        if execution.orchestration_plan is not None and orchestration_report_path:
            resolved_orchestration_report_path = str(Path(orchestration_report_path))
            write_report(
                resolved_orchestration_report_path,
                render_orchestration_plan(
                    execution.orchestration_plan,
                    execution.orchestration_policy,
                    execution.handoff_request,
                ),
            )
            execution.run.register_artifact(
                "cross-department-orchestration",
                "report",
                resolved_orchestration_report_path,
                format="markdown",
                collaboration_mode=execution.orchestration_plan.collaboration_mode,
                departments=execution.orchestration_plan.departments,
            )
            ledger.upsert(execution.run)
            if orchestration_canvas_path:
                resolved_orchestration_canvas_path = str(Path(orchestration_canvas_path))
                canvas = build_orchestration_canvas(
                    execution.run,
                    execution.orchestration_plan,
                    execution.orchestration_policy,
                    execution.handoff_request,
                )
                write_report(resolved_orchestration_canvas_path, render_orchestration_canvas(canvas))
                execution.run.register_artifact(
                    "orchestration-canvas",
                    "report",
                    resolved_orchestration_canvas_path,
                    format="markdown",
                    recommendation=canvas.recommendation,
                )
                ledger.upsert(execution.run)
            journal.record(
                "orchestration",
                execution.orchestration_plan.collaboration_mode,
                departments=execution.orchestration_plan.departments,
                approvals=execution.orchestration_plan.required_approvals,
                tier=execution.orchestration_policy.tier if execution.orchestration_policy else "standard",
                upgrade_required=execution.orchestration_policy.upgrade_required if execution.orchestration_policy else False,
                handoff_team=execution.handoff_request.target_team if execution.handoff_request else "none",
            )

        resolved_pilot_report_path = None
        if pilot_scorecard is not None and pilot_report_path:
            resolved_pilot_report_path = str(Path(pilot_report_path))
            write_report(resolved_pilot_report_path, render_pilot_scorecard(pilot_scorecard))
            execution.run.register_artifact(
                "pilot-scorecard",
                "report",
                resolved_pilot_report_path,
                format="markdown",
                recommendation=pilot_scorecard.recommendation,
            )
            ledger.upsert(execution.run)
            journal.record(
                "pilot-scorecard",
                pilot_scorecard.recommendation,
                metrics_met=pilot_scorecard.metrics_met,
                metrics_total=len(pilot_scorecard.metrics),
                annualized_roi=round(pilot_scorecard.annualized_roi, 1),
            )

        resolved_repo_sync_report_path = None
        if repo_sync_audit is not None:
            execution.run.audit(
                "repo.sync",
                "workflow-engine",
                repo_sync_audit.sync.status,
                failure_category=repo_sync_audit.sync.failure_category,
                summary=repo_sync_audit.sync.summary,
                branch=repo_sync_audit.sync.branch,
                remote_ref=repo_sync_audit.sync.remote_ref,
                ahead_by=repo_sync_audit.sync.ahead_by,
                behind_by=repo_sync_audit.sync.behind_by,
                dirty_paths=repo_sync_audit.sync.dirty_paths,
                auth_target=repo_sync_audit.sync.auth_target,
            )
            execution.run.audit(
                "repo.pr-freshness",
                "workflow-engine",
                "fresh" if repo_sync_audit.pull_request.fresh else "stale",
                pr_number=repo_sync_audit.pull_request.pr_number,
                pr_url=repo_sync_audit.pull_request.pr_url,
                branch_state=repo_sync_audit.pull_request.branch_state,
                body_state=repo_sync_audit.pull_request.body_state,
                branch_head_sha=repo_sync_audit.pull_request.branch_head_sha,
                pr_head_sha=repo_sync_audit.pull_request.pr_head_sha,
            )
            journal.record(
                "repo-sync",
                repo_sync_audit.sync.status,
                failure_category=repo_sync_audit.sync.failure_category,
                branch_state=repo_sync_audit.pull_request.branch_state,
                body_state=repo_sync_audit.pull_request.body_state,
            )
            if repo_sync_report_path:
                resolved_repo_sync_report_path = str(Path(repo_sync_report_path))
                write_report(resolved_repo_sync_report_path, render_repo_sync_audit_report(repo_sync_audit))
                execution.run.register_artifact(
                    "repo-sync-audit",
                    "report",
                    resolved_repo_sync_report_path,
                    format="markdown",
                    sync_status=repo_sync_audit.sync.status,
                    pr_branch_state=repo_sync_audit.pull_request.branch_state,
                    pr_body_state=repo_sync_audit.pull_request.body_state,
                )
                ledger.upsert(execution.run)

        acceptance = self.gate.evaluate(
            task,
            execution,
            validation_evidence=validation_evidence,
            approvals=approvals,
            pilot_scorecard=pilot_scorecard,
        )
        if acceptance.approvals:
            execution.run.audit_spec_event(
                APPROVAL_RECORDED_EVENT,
                "workflow-engine",
                "recorded",
                task_id=task.task_id,
                run_id=run_id,
                approvals=list(acceptance.approvals),
                approval_count=len(acceptance.approvals),
                acceptance_status=acceptance.status,
            )
        journal.record(
            "acceptance",
            acceptance.status,
            passed=acceptance.passed,
            missing_acceptance_criteria=acceptance.missing_acceptance_criteria,
            missing_validation_steps=acceptance.missing_validation_steps,
        )
        execution.run.record_closeout(
            validation_evidence=list(validation_evidence or []),
            git_push_succeeded=git_push_succeeded,
            git_push_output=git_push_output,
            git_log_stat_output=git_log_stat_output,
            repo_sync_audit=repo_sync_audit,
        )
        ledger.upsert(execution.run)
        journal.record(
            "closeout",
            "complete" if execution.run.closeout.complete else "pending",
            validation_evidence=list(validation_evidence or []),
            git_push_succeeded=git_push_succeeded,
            git_log_stat_captured=bool(git_log_stat_output.strip()),
            repo_sync_status=repo_sync_audit.sync.status if repo_sync_audit else "none",
            repo_sync_failure_category=repo_sync_audit.sync.failure_category if repo_sync_audit else "",
        )

        resolved_journal_path = None
        if journal_path:
            resolved_journal_path = journal.write(journal_path)

        return WorkflowRunResult(
            execution=execution,
            acceptance=acceptance,
            journal=journal,
            journal_path=resolved_journal_path,
            orchestration_report_path=resolved_orchestration_report_path,
            orchestration_canvas_path=resolved_orchestration_canvas_path,
            pilot_report_path=resolved_pilot_report_path,
            repo_sync_report_path=resolved_repo_sync_report_path,
        )

    def run_definition(
        self,
        task: Task,
        definition: WorkflowDefinition,
        run_id: str,
        ledger: ObservabilityLedger,
    ) -> WorkflowRunResult:
        definition.validate()
        return self.run(
            task,
            run_id=run_id,
            ledger=ledger,
            report_path=definition.render_report_path(task, run_id),
            journal_path=definition.render_journal_path(task, run_id),
            validation_evidence=definition.validation_evidence,
            approvals=definition.approvals,
        )
