from dataclasses import dataclass, field
from datetime import datetime
from html import escape
from pathlib import Path
from typing import Dict, List, Optional, Sequence

from .observability import TaskRun
from .orchestration import OrchestrationPlan, OrchestrationPolicyDecision


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
class CrossTeamFlowSnapshot:
    plan: OrchestrationPlan
    run: Optional[TaskRun] = None
    policy: Optional[OrchestrationPolicyDecision] = None
    source: str = "orchestration"


@dataclass
class CrossTeamFlow:
    task_id: str
    source: str
    collaboration_mode: str
    departments: List[str] = field(default_factory=list)
    required_approvals: List[str] = field(default_factory=list)
    blocked_departments: List[str] = field(default_factory=list)
    status: str = "planned"
    summary: str = ""
    next_action: str = "continue handoff execution"

    @property
    def is_cross_team(self) -> bool:
        return len(self.departments) > 1

    @property
    def is_blocked(self) -> bool:
        return bool(self.blocked_departments)

    @property
    def needs_attention(self) -> bool:
        return self.status in {"needs-approval", "failed", "rejected", "blocked"} or self.is_blocked


@dataclass
class CrossTeamFlowOverview:
    name: str
    period: str
    flows: List[CrossTeamFlow] = field(default_factory=list)

    @property
    def total_flows(self) -> int:
        return len(self.flows)

    @property
    def cross_team_flows(self) -> int:
        return sum(1 for flow in self.flows if flow.is_cross_team)

    @property
    def blocked_flows(self) -> int:
        return sum(1 for flow in self.flows if flow.is_blocked)

    @property
    def approval_queue_depth(self) -> int:
        return sum(1 for flow in self.flows if flow.status == "needs-approval")

    @property
    def department_counts(self) -> Dict[str, int]:
        counts: Dict[str, int] = {}
        for flow in self.flows:
            for department in flow.departments:
                counts[department] = counts.get(department, 0) + 1
        return dict(sorted(counts.items()))

    @property
    def status_counts(self) -> Dict[str, int]:
        counts: Dict[str, int] = {}
        for flow in self.flows:
            counts[flow.status] = counts.get(flow.status, 0) + 1
        return dict(sorted(counts.items()))

    @property
    def source_counts(self) -> Dict[str, int]:
        counts: Dict[str, int] = {}
        for flow in self.flows:
            counts[flow.source] = counts.get(flow.source, 0) + 1
        return dict(sorted(counts.items()))

    @property
    def at_risk_flows(self) -> List[CrossTeamFlow]:
        return [flow for flow in self.flows if flow.needs_attention]


def build_cross_team_flow_overview(
    snapshots: Sequence[CrossTeamFlowSnapshot],
    name: str = "Cross-Team Flow Overview",
    period: str = "current",
) -> CrossTeamFlowOverview:
    flows: List[CrossTeamFlow] = []
    for snapshot in snapshots:
        run = snapshot.run
        policy = snapshot.policy
        status = run.status if run is not None else "planned"
        summary = _cross_team_flow_summary(snapshot)
        flows.append(
            CrossTeamFlow(
                task_id=snapshot.plan.task_id,
                source=(run.source if run is not None else snapshot.source),
                collaboration_mode=snapshot.plan.collaboration_mode,
                departments=list(snapshot.plan.departments),
                required_approvals=list(snapshot.plan.required_approvals),
                blocked_departments=list(policy.blocked_departments if policy is not None else []),
                status=status,
                summary=summary,
                next_action=_cross_team_flow_next_action(snapshot, status),
            )
        )

    severity_rank = {"failed": 0, "rejected": 1, "needs-approval": 2, "blocked": 3, "planned": 4}
    flows.sort(key=lambda flow: (severity_rank.get(flow.status, 5), flow.task_id))
    return CrossTeamFlowOverview(name=name, period=period, flows=flows)


def render_issue_validation_report(issue_id: str, version: str, environment: str, summary: str) -> str:
    return f"""# Issue Validation Report\n\n- Issue ID: {issue_id}\n- 版本号: {version}\n- 测试环境: {environment}\n- 生成时间: {datetime.utcnow().isoformat()}Z\n\n## 结论\n\n{summary}\n"""


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

    return IssueClosureDecision(
        issue_id=issue_id,
        allowed=True,
        reason="validation report present; issue can be closed",
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


def render_cross_team_flow_overview_report(overview: CrossTeamFlowOverview) -> str:
    department_mix = _format_counts(overview.department_counts)
    source_mix = _format_counts(overview.source_counts)
    status_mix = _format_counts(overview.status_counts)
    lines = [
        "# Cross-Team Flow Overview",
        "",
        f"- Overview: {overview.name}",
        f"- Period: {overview.period}",
        f"- Total Flows: {overview.total_flows}",
        f"- Cross-Team Flows: {overview.cross_team_flows}",
        f"- Approval Queue Depth: {overview.approval_queue_depth}",
        f"- Blocked Flows: {overview.blocked_flows}",
        f"- Department Mix: {department_mix}",
        f"- Source Mix: {source_mix}",
        f"- Status Mix: {status_mix}",
        "",
        "## Active Flows",
        "",
    ]

    if overview.flows:
        for flow in overview.flows:
            approvals = ", ".join(flow.required_approvals) if flow.required_approvals else "none"
            blocked = ", ".join(flow.blocked_departments) if flow.blocked_departments else "none"
            departments = " -> ".join(flow.departments) if flow.departments else "none"
            lines.append(
                f"- {flow.task_id}: source={flow.source} mode={flow.collaboration_mode} departments={departments} "
                f"status={flow.status} approvals={approvals} blocked={blocked} next={flow.next_action}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Attention Queue", ""])
    if overview.at_risk_flows:
        for flow in overview.at_risk_flows:
            lines.append(f"- {flow.task_id}: {flow.summary}")
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_cross_team_flow_overview_page(overview: CrossTeamFlowOverview) -> str:
    cards = [
        ("Total Flows", str(overview.total_flows)),
        ("Cross-Team Flows", str(overview.cross_team_flows)),
        ("Approval Queue", str(overview.approval_queue_depth)),
        ("Blocked Flows", str(overview.blocked_flows)),
        ("Department Mix", _format_counts(overview.department_counts)),
        ("Status Mix", _format_counts(overview.status_counts)),
    ]
    card_html = "".join(
        f'<div class="card"><strong>{escape(label)}</strong><br>{escape(value)}</div>' for label, value in cards
    )
    rows = "".join(
        "<tr>"
        f"<td><strong>{escape(flow.task_id)}</strong></td>"
        f"<td>{escape(flow.source)}</td>"
        f"<td>{escape(flow.collaboration_mode)}</td>"
        f"<td>{escape(' → '.join(flow.departments) if flow.departments else 'none')}</td>"
        f"<td>{escape(', '.join(flow.required_approvals) if flow.required_approvals else 'none')}</td>"
        f"<td>{escape(flow.status)}</td>"
        f"<td>{escape(', '.join(flow.blocked_departments) if flow.blocked_departments else 'none')}</td>"
        f"<td>{escape(flow.next_action)}</td>"
        "</tr>"
        for flow in overview.flows
    ) or '<tr><td colspan="8">None</td></tr>'
    attention = "".join(
        f"<li><strong>{escape(flow.task_id)}</strong> · {escape(flow.summary)}</li>" for flow in overview.at_risk_flows
    ) or "<li>None</li>"

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Cross-Team Flow Overview · {escape(overview.name)}</title>
  <style>
    :root {{ color-scheme: light dark; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }}
    body {{ margin: 2rem auto; max-width: 1180px; padding: 0 1rem 3rem; line-height: 1.5; }}
    .grid {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.75rem; margin: 1rem 0 1.5rem; }}
    .card {{ border: 1px solid #cbd5e1; border-radius: 10px; padding: 0.9rem; background: rgba(148, 163, 184, 0.08); }}
    table {{ width: 100%; border-collapse: collapse; margin-top: 1rem; }}
    th, td {{ text-align: left; padding: 0.75rem; border-bottom: 1px solid #cbd5e1; vertical-align: top; }}
    h1, h2 {{ margin-bottom: 0.5rem; }}
    ul {{ padding-left: 1.2rem; }}
  </style>
</head>
<body>
  <h1>Cross-Team Flow Overview</h1>
  <p>{escape(overview.name)} · <code>{escape(overview.period)}</code></p>
  <div class="grid">{card_html}</div>
  <h2>Flow Table</h2>
  <table>
    <thead>
      <tr>
        <th>Task</th>
        <th>Source</th>
        <th>Mode</th>
        <th>Departments</th>
        <th>Approvals</th>
        <th>Status</th>
        <th>Blocked</th>
        <th>Next Action</th>
      </tr>
    </thead>
    <tbody>{rows}</tbody>
  </table>
  <h2>Attention Queue</h2>
  <ul>{attention}</ul>
</body>
</html>
"""


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


def _cross_team_flow_summary(snapshot: CrossTeamFlowSnapshot) -> str:
    if snapshot.policy is not None and snapshot.policy.upgrade_required:
        blocked = ", ".join(snapshot.policy.blocked_departments) if snapshot.policy.blocked_departments else "additional teams"
        return f"premium tier required to unblock {blocked}"
    if snapshot.run is not None and snapshot.run.summary:
        return snapshot.run.summary
    approvals = ", ".join(snapshot.plan.required_approvals)
    if approvals:
        return f"waiting for {approvals} approval"
    return "handoffs aligned and ready to proceed"


def _cross_team_flow_next_action(snapshot: CrossTeamFlowSnapshot, status: str) -> str:
    if snapshot.policy is not None and snapshot.policy.upgrade_required:
        blocked = ", ".join(snapshot.policy.blocked_departments) if snapshot.policy.blocked_departments else "advanced departments"
        return f"upgrade tier to unlock {blocked}"
    if status == "needs-approval" and snapshot.plan.required_approvals:
        return f"collect {', '.join(snapshot.plan.required_approvals)} approval"
    if status in {"failed", "rejected", "blocked"}:
        return "review the failed handoff and replay safely"
    if snapshot.plan.required_approvals:
        return f"confirm handoff owners and prepare {', '.join(snapshot.plan.required_approvals)} approval"
    return "continue handoff execution"


def _format_counts(counts: Dict[str, int]) -> str:
    if not counts:
        return "none"
    return " ".join(f"{key}={value}" for key, value in counts.items())


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
