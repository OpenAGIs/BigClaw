from dataclasses import dataclass, field
from datetime import datetime
from html import escape
from pathlib import Path
from typing import List, Optional

from .observability import TaskRun


@dataclass
class PilotMetric:
    name: str
    baseline: float
    current: float
    target: float
    unit: str = ""
    higher_is_better: bool = True

    @property
    def delta(self) -> float:
        return self.current - self.baseline

    @property
    def met_target(self) -> bool:
        if self.higher_is_better:
            return self.current >= self.target
        return self.current <= self.target


@dataclass
class PilotScorecard:
    issue_id: str
    customer: str
    period: str
    metrics: List[PilotMetric] = field(default_factory=list)
    monthly_benefit: float = 0.0
    monthly_cost: float = 0.0
    implementation_cost: float = 0.0
    benchmark_score: Optional[int] = None
    benchmark_passed: Optional[bool] = None

    @property
    def monthly_net_value(self) -> float:
        return self.monthly_benefit - self.monthly_cost

    @property
    def annualized_roi(self) -> float:
        total_cost = self.implementation_cost + (self.monthly_cost * 12)
        if total_cost <= 0:
            return 0.0
        annual_gain = (self.monthly_benefit * 12) - total_cost
        return (annual_gain / total_cost) * 100

    @property
    def payback_months(self) -> Optional[float]:
        if self.monthly_net_value <= 0:
            return None
        if self.implementation_cost <= 0:
            return 0.0
        return round(self.implementation_cost / self.monthly_net_value, 1)

    @property
    def metrics_met(self) -> int:
        return sum(1 for metric in self.metrics if metric.met_target)

    @property
    def recommendation(self) -> str:
        benchmark_ok = self.benchmark_passed is not False
        if self.metrics and self.metrics_met == len(self.metrics) and self.annualized_roi > 0 and benchmark_ok:
            return "go"
        if self.annualized_roi > 0 or self.metrics_met:
            return "iterate"
        return "hold"


@dataclass
class PilotPortfolio:
    name: str
    period: str
    scorecards: List[PilotScorecard] = field(default_factory=list)

    @property
    def total_monthly_net_value(self) -> float:
        return sum(scorecard.monthly_net_value for scorecard in self.scorecards)

    @property
    def average_roi(self) -> float:
        if not self.scorecards:
            return 0.0
        return round(
            sum(scorecard.annualized_roi for scorecard in self.scorecards) / len(self.scorecards),
            1,
        )

    @property
    def recommendation_counts(self) -> dict[str, int]:
        counts = {"go": 0, "iterate": 0, "hold": 0}
        for scorecard in self.scorecards:
            counts[scorecard.recommendation] += 1
        return counts

    @property
    def recommendation(self) -> str:
        counts = self.recommendation_counts
        if self.scorecards and counts["go"] == len(self.scorecards):
            return "scale"
        if counts["go"] or counts["iterate"]:
            return "continue"
        return "stop"


@dataclass
class IssueClosureDecision:
    issue_id: str
    allowed: bool
    reason: str
    report_path: str = ""


@dataclass
class DocumentationArtifact:
    name: str
    path: str

    @property
    def available(self) -> bool:
        return validation_report_exists(self.path)


@dataclass
class LaunchChecklistItem:
    name: str
    evidence: List[str] = field(default_factory=list)


@dataclass
class LaunchChecklist:
    issue_id: str
    documentation: List[DocumentationArtifact] = field(default_factory=list)
    items: List[LaunchChecklistItem] = field(default_factory=list)

    @property
    def documentation_status(self) -> dict[str, bool]:
        return {artifact.name: artifact.available for artifact in self.documentation}

    @property
    def completed_items(self) -> int:
        return sum(1 for item in self.items if self.item_completed(item))

    @property
    def missing_documentation(self) -> List[str]:
        return [artifact.name for artifact in self.documentation if not artifact.available]

    @property
    def ready(self) -> bool:
        if self.missing_documentation:
            return False
        return all(self.item_completed(item) for item in self.items)

    def item_completed(self, item: LaunchChecklistItem) -> bool:
        status = self.documentation_status
        if not item.evidence:
            return True
        return all(status.get(name, False) for name in item.evidence)


@dataclass
class TriageFinding:
    run_id: str
    task_id: str
    source: str
    severity: str
    owner: str
    status: str
    reason: str
    next_action: str


@dataclass
class AutoTriageCenter:
    name: str
    period: str
    findings: List[TriageFinding] = field(default_factory=list)

    @property
    def flagged_runs(self) -> int:
        return len(self.findings)

    @property
    def severity_counts(self) -> dict[str, int]:
        counts = {"critical": 0, "high": 0, "medium": 0}
        for finding in self.findings:
            counts[finding.severity] += 1
        return counts

    @property
    def owner_counts(self) -> dict[str, int]:
        counts = {"security": 0, "engineering": 0, "operations": 0}
        for finding in self.findings:
            counts[finding.owner] = counts.get(finding.owner, 0) + 1
        return counts

    @property
    def recommendation(self) -> str:
        counts = self.severity_counts
        if counts["critical"]:
            return "immediate-attention"
        if counts["high"]:
            return "review-queue"
        return "monitor"


@dataclass
class TakeoverRequest:
    run_id: str
    task_id: str
    source: str
    target_team: str
    status: str
    reason: str
    required_approvals: List[str] = field(default_factory=list)


@dataclass
class TakeoverQueue:
    name: str
    period: str
    requests: List[TakeoverRequest] = field(default_factory=list)

    @property
    def pending_requests(self) -> int:
        return len(self.requests)

    @property
    def team_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for request in self.requests:
            counts[request.target_team] = counts.get(request.target_team, 0) + 1
        return counts

    @property
    def approval_count(self) -> int:
        return sum(len(request.required_approvals) for request in self.requests)

    @property
    def recommendation(self) -> str:
        if any(request.target_team == "security" for request in self.requests):
            return "expedite-security-review"
        if self.requests:
            return "staff-takeover-queue"
        return "monitor"


def render_issue_validation_report(issue_id: str, version: str, environment: str, summary: str) -> str:
    return f"""# Issue Validation Report\n\n- Issue ID: {issue_id}\n- 版本号: {version}\n- 测试环境: {environment}\n- 生成时间: {datetime.utcnow().isoformat()}Z\n\n## 结论\n\n{summary}\n"""


def build_launch_checklist(
    issue_id: str,
    documentation: List[DocumentationArtifact],
    items: List[LaunchChecklistItem],
) -> LaunchChecklist:
    return LaunchChecklist(issue_id=issue_id, documentation=documentation, items=items)


def render_launch_checklist_report(checklist: LaunchChecklist) -> str:
    lines = [
        "# Launch Checklist",
        "",
        f"- Issue ID: {checklist.issue_id}",
        f"- Linked Documentation: {len(checklist.documentation)}",
        f"- Completed Items: {checklist.completed_items}/{len(checklist.items)}",
        f"- Ready: {checklist.ready}",
        "",
        "## Documentation",
        "",
    ]

    if checklist.documentation:
        for artifact in checklist.documentation:
            lines.append(
                f"- {artifact.name}: available={artifact.available} path={artifact.path}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Checklist", ""])
    if checklist.items:
        for item in checklist.items:
            evidence = ", ".join(item.evidence) if item.evidence else "none"
            lines.append(
                f"- {item.name}: completed={checklist.item_completed(item)} evidence={evidence}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_pilot_scorecard(scorecard: PilotScorecard) -> str:
    lines = [
        "# Pilot Scorecard",
        "",
        f"- Issue ID: {scorecard.issue_id}",
        f"- Customer: {scorecard.customer}",
        f"- Period: {scorecard.period}",
        f"- Recommendation: {scorecard.recommendation}",
        f"- Metrics Met: {scorecard.metrics_met}/{len(scorecard.metrics)}",
        f"- Monthly Net Value: {scorecard.monthly_net_value:.2f}",
        f"- Annualized ROI: {scorecard.annualized_roi:.1f}%",
    ]

    if scorecard.payback_months is None:
        lines.append("- Payback Months: n/a")
    else:
        lines.append(f"- Payback Months: {scorecard.payback_months:.1f}")

    if scorecard.benchmark_score is not None:
        lines.append(f"- Benchmark Score: {scorecard.benchmark_score}")
    if scorecard.benchmark_passed is not None:
        lines.append(f"- Benchmark Passed: {scorecard.benchmark_passed}")

    lines.extend(["", "## KPI Progress", ""])
    if scorecard.metrics:
        for metric in scorecard.metrics:
            comparator = ">=" if metric.higher_is_better else "<="
            unit_suffix = f" {metric.unit}" if metric.unit else ""
            lines.append(
                f"- {metric.name}: baseline={metric.baseline}{unit_suffix} current={metric.current}{unit_suffix} "
                f"target{comparator}{metric.target}{unit_suffix} delta={metric.delta:+.2f}{unit_suffix} met={metric.met_target}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_pilot_portfolio_report(portfolio: PilotPortfolio) -> str:
    counts = portfolio.recommendation_counts
    lines = [
        "# Pilot Portfolio Report",
        "",
        f"- Portfolio: {portfolio.name}",
        f"- Period: {portfolio.period}",
        f"- Scorecards: {len(portfolio.scorecards)}",
        f"- Recommendation: {portfolio.recommendation}",
        f"- Total Monthly Net Value: {portfolio.total_monthly_net_value:.2f}",
        f"- Average ROI: {portfolio.average_roi:.1f}%",
        f"- Recommendation Mix: go={counts['go']} iterate={counts['iterate']} hold={counts['hold']}",
        "",
        "## Customers",
        "",
    ]

    if portfolio.scorecards:
        for scorecard in portfolio.scorecards:
            lines.append(
                f"- {scorecard.customer}: recommendation={scorecard.recommendation} roi={scorecard.annualized_roi:.1f}% "
                f"monthly-net={scorecard.monthly_net_value:.2f} benchmark={scorecard.benchmark_score if scorecard.benchmark_score is not None else 'n/a'}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def write_report(path: str, content: str) -> None:
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content)


def validation_report_exists(report_path: Optional[str]) -> bool:
    if not report_path:
        return False

    path = Path(report_path)
    if not path.exists() or not path.is_file():
        return False

    return bool(path.read_text().strip())


def evaluate_issue_closure(
    issue_id: str,
    report_path: Optional[str],
    validation_passed: bool = True,
    launch_checklist: Optional[LaunchChecklist] = None,
) -> IssueClosureDecision:
    resolved_path = str(Path(report_path)) if report_path else ""

    if not validation_report_exists(report_path):
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="validation report required before closing issue",
            report_path=resolved_path,
        )

    if not validation_passed:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="validation failed; issue must remain open",
            report_path=resolved_path,
        )

    if launch_checklist is not None and not launch_checklist.ready:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="launch checklist incomplete; linked documentation missing or empty",
            report_path=resolved_path,
        )

    return IssueClosureDecision(
        issue_id=issue_id,
        allowed=True,
        reason="validation report and launch checklist requirements satisfied; issue can be closed",
        report_path=resolved_path,
    )


def build_auto_triage_center(runs: List[TaskRun], name: str = "Auto Triage Center", period: str = "current") -> AutoTriageCenter:
    findings: List[TriageFinding] = []
    for run in runs:
        if not _run_requires_triage(run):
            continue

        severity = _triage_severity(run)
        owner = _triage_owner(run)
        findings.append(
            TriageFinding(
                run_id=run.run_id,
                task_id=run.task_id,
                source=run.source,
                severity=severity,
                owner=owner,
                status=run.status,
                reason=_triage_reason(run),
                next_action=_triage_next_action(severity, owner),
            )
        )

    severity_rank = {"critical": 0, "high": 1, "medium": 2}
    findings.sort(key=lambda finding: (severity_rank[finding.severity], finding.owner, finding.run_id))
    return AutoTriageCenter(name=name, period=period, findings=findings)


def render_auto_triage_center_report(center: AutoTriageCenter, total_runs: Optional[int] = None) -> str:
    severity = center.severity_counts
    owners = center.owner_counts
    lines = [
        "# Auto Triage Center",
        "",
        f"- Center: {center.name}",
        f"- Period: {center.period}",
        f"- Flagged Runs: {center.flagged_runs}",
        f"- Total Runs: {total_runs if total_runs is not None else center.flagged_runs}",
        f"- Recommendation: {center.recommendation}",
        f"- Severity Mix: critical={severity['critical']} high={severity['high']} medium={severity['medium']}",
        f"- Owner Mix: security={owners['security']} engineering={owners['engineering']} operations={owners['operations']}",
        "",
        "## Queue",
        "",
    ]

    if center.findings:
        for finding in center.findings:
            lines.append(
                f"- {finding.run_id}: severity={finding.severity} owner={finding.owner} status={finding.status} "
                f"task={finding.task_id} reason={finding.reason} next={finding.next_action}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def build_takeover_queue_from_ledger(
    entries: List[dict],
    name: str = "Human Takeover Queue",
    period: str = "current",
) -> TakeoverQueue:
    requests: List[TakeoverRequest] = []
    for entry in entries:
        handoff_audit = _latest_handoff_audit(entry.get("audits", []))
        if handoff_audit is None:
            continue

        details = handoff_audit.get("details", {})
        requests.append(
            TakeoverRequest(
                run_id=str(entry.get("run_id", "")),
                task_id=str(entry.get("task_id", "")),
                source=str(entry.get("source", "")),
                target_team=str(details.get("target_team", "operations")),
                status=str(handoff_audit.get("outcome", "pending")),
                reason=str(details.get("reason", entry.get("summary", "handoff requested"))),
                required_approvals=[str(value) for value in details.get("required_approvals", [])],
            )
        )

    requests.sort(key=lambda request: (request.target_team, request.run_id))
    return TakeoverQueue(name=name, period=period, requests=requests)


def render_takeover_queue_report(queue: TakeoverQueue, total_runs: Optional[int] = None) -> str:
    team_counts = queue.team_counts
    team_mix = " ".join(f"{team}={count}" for team, count in sorted(team_counts.items())) or "none"
    lines = [
        "# Human Takeover Queue",
        "",
        f"- Queue: {queue.name}",
        f"- Period: {queue.period}",
        f"- Pending Requests: {queue.pending_requests}",
        f"- Total Runs: {total_runs if total_runs is not None else queue.pending_requests}",
        f"- Recommendation: {queue.recommendation}",
        f"- Team Mix: {team_mix}",
        f"- Required Approvals: {queue.approval_count}",
        "",
        "## Requests",
        "",
    ]

    if queue.requests:
        for request in queue.requests:
            approvals = ",".join(request.required_approvals) if request.required_approvals else "none"
            lines.append(
                f"- {request.run_id}: team={request.target_team} status={request.status} task={request.task_id} "
                f"approvals={approvals} reason={request.reason}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def _latest_handoff_audit(audits: List[dict]) -> Optional[dict]:
    for audit in reversed(audits):
        if audit.get("action") == "orchestration.handoff":
            return audit
    return None


def _run_requires_triage(run: TaskRun) -> bool:
    if run.status in {"failed", "needs-approval"}:
        return True
    if any(entry.status in {"pending", "error", "failed"} for entry in run.traces):
        return True
    return any(entry.outcome in {"pending", "failed", "rejected"} for entry in run.audits)


def _triage_severity(run: TaskRun) -> str:
    if run.status == "failed":
        return "critical"
    if any(entry.status in {"error", "failed"} for entry in run.traces):
        return "critical"
    if any(entry.outcome in {"failed", "rejected"} for entry in run.audits):
        return "critical"
    if run.status == "needs-approval":
        return "high"
    if any(entry.status == "pending" for entry in run.traces):
        return "high"
    if any(entry.outcome == "pending" for entry in run.audits):
        return "high"
    return "medium"


def _triage_owner(run: TaskRun) -> str:
    evidence = " ".join(
        [run.summary, run.title, run.source, run.medium]
        + [entry.status for entry in run.traces]
        + [entry.span for entry in run.traces]
        + [entry.outcome for entry in run.audits]
        + [str(entry.details.get("reason", "")) for entry in run.audits]
        + [str(entry.details.get("approvals", [])) for entry in run.audits]
    ).lower()

    if "security" in evidence or "high-risk" in evidence or "security-review" in evidence:
        return "security"
    if run.medium == "browser" or any(artifact.kind == "page" for artifact in run.artifacts):
        return "engineering"
    return "operations"


def _triage_reason(run: TaskRun) -> str:
    for audit in run.audits:
        if audit.outcome in {"failed", "rejected", "pending"} and audit.details.get("reason"):
            return str(audit.details["reason"])
    for trace in run.traces:
        if trace.status in {"error", "failed", "pending"}:
            return f"{trace.span} is {trace.status}"
    return run.summary or run.status


def _triage_next_action(severity: str, owner: str) -> str:
    if severity == "critical":
        if owner == "engineering":
            return "replay run and inspect tool failures"
        if owner == "security":
            return "page security reviewer and block rollout"
        return "open incident review and coordinate response"
    if owner == "security":
        return "request approval and queue security review"
    if owner == "engineering":
        return "inspect execution evidence and retry when safe"
    return "confirm owner and clear pending workflow gate"


def render_task_run_report(run: TaskRun) -> str:
    lines = [
        "# Task Run Report",
        "",
        f"- Run ID: {run.run_id}",
        f"- Task ID: {run.task_id}",
        f"- Source: {run.source}",
        f"- Medium: {run.medium}",
        f"- Status: {run.status}",
        f"- Started At: {run.started_at}",
        f"- Ended At: {run.ended_at or 'n/a'}",
        "",
        "## Summary",
        "",
        run.summary or "No summary recorded.",
        "",
        "## Logs",
        "",
    ]

    if run.logs:
        lines.extend(
            f"- [{entry.level}] {entry.timestamp} {entry.message}" for entry in run.logs
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Trace", ""])
    if run.traces:
        lines.extend(
            f"- {entry.span}: {entry.status} @ {entry.timestamp}" for entry in run.traces
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Artifacts", ""])
    if run.artifacts:
        lines.extend(
            f"- {entry.name} ({entry.kind}): {entry.path}" for entry in run.artifacts
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Audit", ""])
    if run.audits:
        lines.extend(
            f"- {entry.action} by {entry.actor}: {entry.outcome}" for entry in run.audits
        )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_task_run_detail_page(run: TaskRun) -> str:
    def render_items(items: List[str]) -> str:
        if not items:
            return "<li>None</li>"
        return "".join(f"<li>{item}</li>" for item in items)

    summary = escape(run.summary or "No summary recorded.")
    logs = render_items(
        [
            f"<strong>[{escape(entry.level)}]</strong> <code>{escape(entry.timestamp)}</code> {escape(entry.message)}"
            for entry in run.logs
        ]
    )
    traces = render_items(
        [
            f"<strong>{escape(entry.span)}</strong> · {escape(entry.status)} · <code>{escape(entry.timestamp)}</code>"
            for entry in run.traces
        ]
    )
    artifacts = render_items(
        [
            f"<strong>{escape(entry.name)}</strong> ({escape(entry.kind)}) · <code>{escape(entry.path)}</code>"
            for entry in run.artifacts
        ]
    )
    audits = render_items(
        [
            f"<strong>{escape(entry.action)}</strong> by {escape(entry.actor)} · {escape(entry.outcome)}"
            for entry in run.audits
        ]
    )

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Task Run Detail · {escape(run.run_id)}</title>
  <style>
    :root {{ color-scheme: light dark; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }}
    body {{ margin: 2rem auto; max-width: 960px; padding: 0 1rem 3rem; line-height: 1.5; }}
    .grid {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.75rem; margin: 1rem 0 1.5rem; }}
    .card {{ border: 1px solid #cbd5e1; border-radius: 10px; padding: 0.9rem; background: rgba(148, 163, 184, 0.08); }}
    h1, h2 {{ margin-bottom: 0.5rem; }}
    ul {{ padding-left: 1.2rem; }}
    code {{ font-size: 0.95em; }}
  </style>
</head>
<body>
  <h1>Task Run Detail</h1>
  <p>{escape(run.title)}</p>
  <div class="grid">
    <div class="card"><strong>Run ID</strong><br>{escape(run.run_id)}</div>
    <div class="card"><strong>Task ID</strong><br>{escape(run.task_id)}</div>
    <div class="card"><strong>Source</strong><br>{escape(run.source)}</div>
    <div class="card"><strong>Medium</strong><br>{escape(run.medium)}</div>
    <div class="card"><strong>Status</strong><br>{escape(run.status)}</div>
    <div class="card"><strong>Started</strong><br><code>{escape(run.started_at)}</code></div>
    <div class="card"><strong>Ended</strong><br><code>{escape(run.ended_at or 'n/a')}</code></div>
  </div>
  <h2>Summary</h2>
  <p>{summary}</p>
  <h2>Logs</h2>
  <ul>{logs}</ul>
  <h2>Trace</h2>
  <ul>{traces}</ul>
  <h2>Artifacts</h2>
  <ul>{artifacts}</ul>
  <h2>Audit</h2>
  <ul>{audits}</ul>
</body>
</html>
"""
