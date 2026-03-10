import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any, Dict, List, Optional, Sequence

from .dsl import WorkflowDefinition
from .models import RiskLevel, Task
from .observability import ObservabilityLedger, utc_now
from .reports import PilotScorecard, render_pilot_scorecard, write_report
from .scheduler import ExecutionRecord, Scheduler


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

    def to_dict(self) -> Dict[str, Any]:
        return {
            "task_id": self.task_id,
            "run_id": self.run_id,
            "entries": [entry.to_dict() for entry in self.entries],
        }

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
    pilot_report_path: Optional[str] = None


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
            ledger.append(execution.run)
            journal.record(
                "pilot-scorecard",
                pilot_scorecard.recommendation,
                metrics_met=pilot_scorecard.metrics_met,
                metrics_total=len(pilot_scorecard.metrics),
                annualized_roi=round(pilot_scorecard.annualized_roi, 1),
            )

        acceptance = self.gate.evaluate(
            task,
            execution,
            validation_evidence=validation_evidence,
            approvals=approvals,
            pilot_scorecard=pilot_scorecard,
        )
        journal.record(
            "acceptance",
            acceptance.status,
            passed=acceptance.passed,
            missing_acceptance_criteria=acceptance.missing_acceptance_criteria,
            missing_validation_steps=acceptance.missing_validation_steps,
        )

        resolved_journal_path = None
        if journal_path:
            resolved_journal_path = journal.write(journal_path)

        return WorkflowRunResult(
            execution=execution,
            acceptance=acceptance,
            journal=journal,
            journal_path=resolved_journal_path,
            pilot_report_path=resolved_pilot_report_path,
        )

    def run_definition(
        self,
        task: Task,
        definition: WorkflowDefinition,
        run_id: str,
        ledger: ObservabilityLedger,
    ) -> WorkflowRunResult:
        return self.run(
            task,
            run_id=run_id,
            ledger=ledger,
            report_path=definition.render_report_path(task, run_id),
            journal_path=definition.render_journal_path(task, run_id),
            validation_evidence=definition.validation_evidence,
            approvals=definition.approvals,
        )
