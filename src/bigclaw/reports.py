from dataclasses import dataclass, field
from datetime import datetime, timezone
from difflib import SequenceMatcher, unified_diff
from html import escape
from pathlib import Path
import json
from typing import Dict, List, Optional, Sequence

from .models import Task
from .observability import (
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    CollaborationComment,
    CollaborationThread,
    DecisionNote,
    RepoSyncAudit,
    TaskRun,
    build_collaboration_thread,
    build_collaboration_thread_from_audits,
    render_collaboration_lines,
    render_collaboration_panel_html,
)
from .orchestration import HandoffRequest, OrchestrationPlan, OrchestrationPolicyDecision
from .queue import PersistentTaskQueue
from .scheduler import ExecutionRecord, Scheduler
from .observability import ObservabilityLedger


@dataclass
class RunDetailStat:
    label: str
    value: str
    tone: str = "default"


@dataclass
class RunDetailResource:
    name: str
    kind: str
    path: str
    meta: List[str] = field(default_factory=list)
    tone: str = "default"


@dataclass
class RunDetailEvent:
    event_id: str
    lane: str
    title: str
    timestamp: str
    status: str
    summary: str
    details: List[str] = field(default_factory=list)


@dataclass
class RunDetailTab:
    tab_id: str
    label: str
    body_html: str


def render_run_detail_console(
    *,
    page_title: str,
    eyebrow: str,
    hero_title: str,
    hero_summary: str,
    stats: List[RunDetailStat],
    tabs: List[RunDetailTab],
    timeline_events: List[RunDetailEvent],
) -> str:
    stat_cards = "".join(
        f"""
        <article class="stat-card" data-tone="{escape(stat.tone)}">
          <span>{escape(stat.label)}</span>
          <strong>{escape(stat.value)}</strong>
        </article>
        """
        for stat in stats
    )
    tab_buttons = "".join(
        f'<button class="tab-button" type="button" data-tab="{escape(tab.tab_id)}">{escape(tab.label)}</button>'
        for tab in tabs
    )
    tab_panels = "".join(
        f'<section class="tab-panel" data-panel="{escape(tab.tab_id)}">{tab.body_html}</section>'
        for tab in tabs
    )
    event_payload = [
        {
            "id": event.event_id,
            "lane": event.lane,
            "title": event.title,
            "timestamp": event.timestamp,
            "status": event.status,
            "summary": event.summary,
            "details": event.details,
        }
        for event in timeline_events
    ]
    timeline_json = json.dumps(event_payload).replace("</", "<\\/")
    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{escape(page_title)}</title>
  <style>
    :root {{
      color-scheme: light;
      --paper: #f7f1e3;
      --ink: #13212f;
      --muted: #5f6c79;
      --line: rgba(19, 33, 47, 0.14);
      --panel: rgba(255, 251, 245, 0.92);
      --accent: #0f766e;
      --accent-soft: rgba(15, 118, 110, 0.14);
      --alert: #b45309;
      --danger: #b91c1c;
      --shadow: 0 18px 40px rgba(19, 33, 47, 0.12);
      font-family: "Iowan Old Style", "Palatino Linotype", "Book Antiqua", Georgia, serif;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top left, rgba(15, 118, 110, 0.12), transparent 28%),
        linear-gradient(180deg, #fcfaf4 0%, var(--paper) 100%);
    }}
    main {{
      width: min(1160px, calc(100% - 2rem));
      margin: 0 auto;
      padding: 2rem 0 3rem;
    }}
    .shell {{
      border: 1px solid var(--line);
      border-radius: 24px;
      background: rgba(255, 255, 255, 0.7);
      box-shadow: var(--shadow);
      overflow: hidden;
      backdrop-filter: blur(10px);
    }}
    .hero {{
      padding: 1.5rem;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(135deg, rgba(15, 118, 110, 0.08), rgba(255, 255, 255, 0.45));
    }}
    .eyebrow {{
      display: inline-block;
      margin-bottom: 0.65rem;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      font: 600 0.72rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
      color: var(--muted);
    }}
    h1, h2, h3, p {{ margin: 0; }}
    .hero p {{
      margin-top: 0.6rem;
      max-width: 70ch;
      color: var(--muted);
      font-size: 1rem;
      line-height: 1.6;
    }}
    .stats {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
      gap: 0.8rem;
      margin-top: 1.2rem;
    }}
    .stat-card, .surface {{
      border: 1px solid var(--line);
      border-radius: 18px;
      background: var(--panel);
      padding: 1rem;
    }}
    .stat-card span, .meta, .resource-meta, .timeline-meta, .detail-list {{ display: block; color: var(--muted); font: 500 0.78rem/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace; }}
    .stat-card strong {{
      display: block;
      margin-top: 0.45rem;
      font-size: 1.15rem;
    }}
    .stat-card[data-tone="danger"] strong {{ color: var(--danger); }}
    .stat-card[data-tone="warning"] strong {{ color: var(--alert); }}
    .stat-card[data-tone="accent"] strong {{ color: var(--accent); }}
    .tabs {{
      display: flex;
      flex-wrap: wrap;
      gap: 0.6rem;
      padding: 1rem 1.5rem 0;
    }}
    .tab-button {{
      border: 1px solid var(--line);
      border-radius: 999px;
      background: rgba(255, 255, 255, 0.72);
      color: var(--ink);
      padding: 0.55rem 0.95rem;
      cursor: pointer;
      font: 600 0.83rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
    .tab-button.active {{ background: var(--accent); border-color: var(--accent); color: #f8fffd; }}
    .panel-stack {{
      padding: 1rem 1.5rem 1.5rem;
    }}
    .tab-panel {{
      display: none;
    }}
    .tab-panel.active {{
      display: block;
    }}
    .surface h2, .surface h3 {{
      margin-bottom: 0.6rem;
    }}
    .surface p {{
      color: var(--muted);
      line-height: 1.6;
    }}
    .resource-grid {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
      gap: 0.8rem;
    }}
    .resource-card {{
      border: 1px solid var(--line);
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.8);
      padding: 0.95rem;
    }}
    .resource-card[data-tone="report"] {{
      background: linear-gradient(180deg, rgba(15, 118, 110, 0.08), rgba(255, 255, 255, 0.9));
    }}
    .resource-card[data-tone="page"] {{
      background: linear-gradient(180deg, rgba(180, 83, 9, 0.08), rgba(255, 255, 255, 0.92));
    }}
    .resource-card code, .detail-pane code {{
      font: 500 0.78rem/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
      word-break: break-all;
    }}
    .timeline-shell {{
      display: grid;
      grid-template-columns: minmax(280px, 0.95fr) minmax(0, 1.35fr);
      gap: 1rem;
    }}
    .timeline-list {{
      display: grid;
      gap: 0.65rem;
      max-height: 560px;
      overflow: auto;
      padding-right: 0.2rem;
    }}
    .timeline-item {{
      border: 1px solid var(--line);
      border-radius: 16px;
      background: rgba(255, 255, 255, 0.82);
      text-align: left;
      padding: 0.9rem;
      cursor: pointer;
    }}
    .timeline-item.active {{
      border-color: var(--accent);
      box-shadow: inset 0 0 0 1px var(--accent);
      background: var(--accent-soft);
    }}
    .timeline-item strong {{
      display: block;
      margin: 0.25rem 0 0.45rem;
    }}
    .timeline-item p {{
      margin-top: 0.35rem;
      color: var(--muted);
      line-height: 1.5;
    }}
    .detail-pane {{
      min-height: 420px;
    }}
    .detail-pane ul {{
      margin: 0.8rem 0 0;
      padding-left: 1.2rem;
      line-height: 1.6;
    }}
    .kicker {{
      display: inline-block;
      border-radius: 999px;
      padding: 0.25rem 0.6rem;
      background: rgba(19, 33, 47, 0.06);
      font: 600 0.72rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
    .empty {{
      padding: 1rem;
      border: 1px dashed var(--line);
      border-radius: 14px;
      color: var(--muted);
      background: rgba(255, 255, 255, 0.62);
    }}
    @media (max-width: 860px) {{
      .timeline-shell {{
        grid-template-columns: 1fr;
      }}
      main {{
        width: min(100%, calc(100% - 1rem));
      }}
    }}
  </style>
</head>
<body>
  <main>
    <section class="shell">
      <header class="hero">
        <span class="eyebrow">{escape(eyebrow)}</span>
        <h1>{escape(hero_title)}</h1>
        <p>{escape(hero_summary)}</p>
        <div class="stats">{stat_cards}</div>
      </header>
      <nav class="tabs" aria-label="Run detail tabs">{tab_buttons}</nav>
      <div class="panel-stack">{tab_panels}</div>
    </section>
  </main>
  <script id="timeline-data" type="application/json">{timeline_json}</script>
  <script>
    const tabs = Array.from(document.querySelectorAll(".tab-button"));
    const panels = Array.from(document.querySelectorAll(".tab-panel"));
    const activateTab = (tabId) => {{
      tabs.forEach((button) => button.classList.toggle("active", button.dataset.tab === tabId));
      panels.forEach((panel) => panel.classList.toggle("active", panel.dataset.panel === tabId));
    }};
    if (tabs.length > 0) {{
      activateTab(tabs[0].dataset.tab);
      tabs.forEach((button) => button.addEventListener("click", () => activateTab(button.dataset.tab)));
    }}

    const timelineData = JSON.parse(document.getElementById("timeline-data").textContent);
    const timelineButtons = Array.from(document.querySelectorAll(".timeline-item"));
    const detailTitle = document.querySelector("[data-detail='title']");
    const detailMeta = document.querySelector("[data-detail='meta']");
    const detailSummary = document.querySelector("[data-detail='summary']");
    const detailList = document.querySelector("[data-detail='list']");
    const renderEvent = (eventId) => {{
      const match = timelineData.find((item) => item.id === eventId);
      if (!match || !detailTitle || !detailMeta || !detailSummary || !detailList) {{
        return;
      }}
      timelineButtons.forEach((button) => button.classList.toggle("active", button.dataset.eventId === eventId));
      detailTitle.textContent = match.title;
      detailMeta.textContent = `${{match.lane}} / ${{match.status}} / ${{match.timestamp}}`;
      detailSummary.textContent = match.summary;
      detailList.innerHTML = "";
      const items = match.details.length ? match.details : ["No additional details."];
      items.forEach((detail) => {{
        const li = document.createElement("li");
        li.textContent = detail;
        detailList.appendChild(li);
      }});
    }};
    if (timelineButtons.length > 0) {{
      renderEvent(timelineButtons[0].dataset.eventId);
      timelineButtons.forEach((button) => button.addEventListener("click", () => renderEvent(button.dataset.eventId)));
    }}
  </script>
</body>
</html>"""


def render_resource_grid(title: str, description: str, resources: List[RunDetailResource]) -> str:
    if resources:
        cards = "".join(
            f"""
            <article class="resource-card" data-tone="{escape(resource.tone)}">
              <span class="kicker">{escape(resource.kind)}</span>
              <h3>{escape(resource.name)}</h3>
              <p><code>{escape(resource.path)}</code></p>
              <span class="resource-meta">{escape(" | ".join(resource.meta) if resource.meta else "No extra metadata")}</span>
            </article>
            """
            for resource in resources
        )
        body = f'<div class="resource-grid">{cards}</div>'
    else:
        body = '<div class="empty">No resources recorded.</div>'
    return f'<section class="surface"><h2>{escape(title)}</h2><p>{escape(description)}</p>{body}</section>'


def render_timeline_panel(title: str, description: str, timeline_events: List[RunDetailEvent]) -> str:
    if timeline_events:
        items = "".join(
            f"""
            <button class="timeline-item" type="button" data-event-id="{escape(event.event_id)}">
              <span class="kicker">{escape(event.lane)}</span>
              <strong>{escape(event.title)}</strong>
              <span class="timeline-meta">{escape(event.timestamp)} | {escape(event.status)}</span>
              <p>{escape(event.summary)}</p>
            </button>
            """
            for event in timeline_events
        )
    else:
        items = '<div class="empty">No timeline events recorded.</div>'
    return f"""
    <section class="surface">
      <h2>{escape(title)}</h2>
      <p>{escape(description)}</p>
      <div class="timeline-shell">
        <div class="timeline-list">{items}</div>
        <aside class="surface detail-pane">
          <span class="kicker">Inspector</span>
          <h3 data-detail="title">No event selected</h3>
          <span class="meta" data-detail="meta">timeline / idle / n/a</span>
          <p data-detail="summary">Select a timeline item to inspect the synced log, trace, audit, or artifact details.</p>
          <ul class="detail-list" data-detail="list"><li>No additional details.</li></ul>
        </aside>
      </div>
    </section>
    """


def _utc_now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


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
class PilotKPI:
    name: str
    target: float
    actual: float
    higher_is_better: bool = True

    @property
    def met(self) -> bool:
        return self.actual >= self.target if self.higher_is_better else self.actual <= self.target


@dataclass
class PilotImplementationResult:
    customer: str
    environment: str
    kpis: List[PilotKPI] = field(default_factory=list)
    production_runs: int = 0
    incidents: int = 0

    @property
    def kpi_pass_rate(self) -> float:
        if not self.kpis:
            return 0.0
        passed = len([kpi for kpi in self.kpis if kpi.met])
        return round((passed / len(self.kpis)) * 100, 1)

    @property
    def ready(self) -> bool:
        return self.production_runs > 0 and self.incidents == 0 and self.kpi_pass_rate >= 80.0


def render_pilot_implementation_report(result: PilotImplementationResult) -> str:
    lines = [
        "# Pilot Implementation Report",
        "",
        f"- Customer: {result.customer}",
        f"- Environment: {result.environment}",
        f"- Production Runs: {result.production_runs}",
        f"- Incidents: {result.incidents}",
        f"- KPI Pass Rate: {result.kpi_pass_rate}%",
        f"- Ready: {result.ready}",
        "",
        "## KPI Details",
        "",
    ]
    if result.kpis:
        for kpi in result.kpis:
            lines.append(f"- {kpi.name}: target={kpi.target} actual={kpi.actual} met={kpi.met}")
    else:
        lines.append("- none")
    return "\n".join(lines) + "\n"


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
class FinalDeliveryChecklist:
    issue_id: str
    required_outputs: List[DocumentationArtifact] = field(default_factory=list)
    recommended_documentation: List[DocumentationArtifact] = field(default_factory=list)

    @property
    def required_output_status(self) -> dict[str, bool]:
        return {artifact.name: artifact.available for artifact in self.required_outputs}

    @property
    def recommended_documentation_status(self) -> dict[str, bool]:
        return {artifact.name: artifact.available for artifact in self.recommended_documentation}

    @property
    def generated_required_outputs(self) -> int:
        return sum(1 for artifact in self.required_outputs if artifact.available)

    @property
    def generated_recommended_documentation(self) -> int:
        return sum(1 for artifact in self.recommended_documentation if artifact.available)

    @property
    def missing_required_outputs(self) -> List[str]:
        return [artifact.name for artifact in self.required_outputs if not artifact.available]

    @property
    def missing_recommended_documentation(self) -> List[str]:
        return [artifact.name for artifact in self.recommended_documentation if not artifact.available]

    @property
    def ready(self) -> bool:
        return not self.missing_required_outputs


@dataclass
class ValidationReportDecision:
    allowed_to_close: bool
    status: str
    summary: str
    missing_reports: List[str] = field(default_factory=list)


REQUIRED_REPORT_ARTIFACTS = ["task-run", "replay", "benchmark-suite"]


def enforce_validation_report_policy(artifacts: List[str]) -> ValidationReportDecision:
    existing = set(artifacts)
    missing = [name for name in REQUIRED_REPORT_ARTIFACTS if name not in existing]
    if missing:
        return ValidationReportDecision(
            allowed_to_close=False,
            status="blocked",
            summary="validation report policy not satisfied",
            missing_reports=missing,
        )
    return ValidationReportDecision(
        allowed_to_close=True,
        status="ready",
        summary="validation report policy satisfied",
    )


ALLOWED_ISSUE_CATEGORIES = {"ui", "ia", "permission", "metric"}
ALLOWED_ISSUE_PRIORITIES = {"P0", "P1", "P2"}


@dataclass(frozen=True)
class ArchivedIssue:
    finding_id: str
    summary: str
    category: str
    priority: str
    owner: str
    surface: str = ""
    impact: str = ""
    status: str = "open"
    evidence: List[str] = field(default_factory=list)

    @property
    def normalized_category(self) -> str:
        return self.category.strip().lower()

    @property
    def normalized_priority(self) -> str:
        return self.priority.strip().upper()

    @property
    def resolved(self) -> bool:
        return self.status.strip().lower() == "resolved"

    def to_dict(self) -> Dict[str, object]:
        return {
            "finding_id": self.finding_id,
            "summary": self.summary,
            "category": self.category,
            "priority": self.priority,
            "owner": self.owner,
            "surface": self.surface,
            "impact": self.impact,
            "status": self.status,
            "evidence": list(self.evidence),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "ArchivedIssue":
        return cls(
            finding_id=str(data["finding_id"]),
            summary=str(data["summary"]),
            category=str(data["category"]),
            priority=str(data["priority"]),
            owner=str(data.get("owner", "")),
            surface=str(data.get("surface", "")),
            impact=str(data.get("impact", "")),
            status=str(data.get("status", "open")),
            evidence=[str(item) for item in data.get("evidence", [])],
        )


@dataclass
class IssuePriorityArchive:
    issue_id: str
    title: str
    version: str
    findings: List[ArchivedIssue] = field(default_factory=list)

    def to_dict(self) -> Dict[str, object]:
        return {
            "issue_id": self.issue_id,
            "title": self.title,
            "version": self.version,
            "findings": [finding.to_dict() for finding in self.findings],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "IssuePriorityArchive":
        return cls(
            issue_id=str(data["issue_id"]),
            title=str(data["title"]),
            version=str(data["version"]),
            findings=[ArchivedIssue.from_dict(item) for item in data.get("findings", [])],
        )


@dataclass(frozen=True)
class IssuePriorityArchiveAudit:
    ready: bool
    finding_count: int
    priority_counts: Dict[str, int] = field(default_factory=dict)
    category_counts: Dict[str, int] = field(default_factory=dict)
    missing_owners: List[str] = field(default_factory=list)
    invalid_priorities: List[str] = field(default_factory=list)
    invalid_categories: List[str] = field(default_factory=list)
    unresolved_p0_findings: List[str] = field(default_factory=list)

    @property
    def summary(self) -> str:
        status = "READY" if self.ready else "HOLD"
        return (
            f"{status}: findings={self.finding_count} "
            f"missing_owners={len(self.missing_owners)} "
            f"invalid_priorities={len(self.invalid_priorities)} "
            f"invalid_categories={len(self.invalid_categories)} "
            f"unresolved_p0={len(self.unresolved_p0_findings)}"
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "ready": self.ready,
            "finding_count": self.finding_count,
            "priority_counts": dict(self.priority_counts),
            "category_counts": dict(self.category_counts),
            "missing_owners": list(self.missing_owners),
            "invalid_priorities": list(self.invalid_priorities),
            "invalid_categories": list(self.invalid_categories),
            "unresolved_p0_findings": list(self.unresolved_p0_findings),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "IssuePriorityArchiveAudit":
        return cls(
            ready=bool(data["ready"]),
            finding_count=int(data.get("finding_count", 0)),
            priority_counts={
                str(priority): int(count)
                for priority, count in dict(data.get("priority_counts", {})).items()
            },
            category_counts={
                str(category): int(count)
                for category, count in dict(data.get("category_counts", {})).items()
            },
            missing_owners=[str(item) for item in data.get("missing_owners", [])],
            invalid_priorities=[str(item) for item in data.get("invalid_priorities", [])],
            invalid_categories=[str(item) for item in data.get("invalid_categories", [])],
            unresolved_p0_findings=[str(item) for item in data.get("unresolved_p0_findings", [])],
        )


class IssuePriorityArchivist:
    def audit(self, archive: IssuePriorityArchive) -> IssuePriorityArchiveAudit:
        priority_counts = {priority: 0 for priority in sorted(ALLOWED_ISSUE_PRIORITIES)}
        category_counts = {category: 0 for category in sorted(ALLOWED_ISSUE_CATEGORIES)}
        missing_owners: List[str] = []
        invalid_priorities: List[str] = []
        invalid_categories: List[str] = []
        unresolved_p0_findings: List[str] = []

        for finding in archive.findings:
            if not finding.owner.strip():
                missing_owners.append(finding.finding_id)

            if finding.normalized_priority in ALLOWED_ISSUE_PRIORITIES:
                priority_counts[finding.normalized_priority] += 1
            else:
                invalid_priorities.append(finding.finding_id)

            if finding.normalized_category in ALLOWED_ISSUE_CATEGORIES:
                category_counts[finding.normalized_category] += 1
            else:
                invalid_categories.append(finding.finding_id)

            if finding.normalized_priority == "P0" and not finding.resolved:
                unresolved_p0_findings.append(finding.finding_id)

        ready = bool(archive.findings) and not (
            missing_owners or invalid_priorities or invalid_categories or unresolved_p0_findings
        )
        return IssuePriorityArchiveAudit(
            ready=ready,
            finding_count=len(archive.findings),
            priority_counts=priority_counts,
            category_counts=category_counts,
            missing_owners=sorted(missing_owners),
            invalid_priorities=sorted(invalid_priorities),
            invalid_categories=sorted(invalid_categories),
            unresolved_p0_findings=sorted(unresolved_p0_findings),
        )


def render_issue_priority_archive_report(
    archive: IssuePriorityArchive,
    audit: IssuePriorityArchiveAudit,
) -> str:
    lines = [
        "# Issue Priority Archive",
        "",
        f"- Issue: {archive.issue_id} {archive.title}",
        f"- Version: {archive.version}",
        f"- Audit: {audit.summary}",
        (
            "- Priority Counts: "
            f"P0={audit.priority_counts.get('P0', 0)} "
            f"P1={audit.priority_counts.get('P1', 0)} "
            f"P2={audit.priority_counts.get('P2', 0)}"
        ),
        (
            "- Category Counts: "
            f"ui={audit.category_counts.get('ui', 0)} "
            f"ia={audit.category_counts.get('ia', 0)} "
            f"permission={audit.category_counts.get('permission', 0)} "
            f"metric={audit.category_counts.get('metric', 0)}"
        ),
        "",
        "## Findings",
    ]
    for finding in archive.findings:
        lines.append(
            "- "
            f"{finding.finding_id}: {finding.summary} "
            f"category={finding.normalized_category} priority={finding.normalized_priority} "
            f"owner={finding.owner or 'none'} status={finding.status}"
        )
        lines.append(
            "  "
            f"surface={finding.surface or 'none'} impact={finding.impact or 'none'} "
            f"evidence={','.join(finding.evidence) or 'none'}"
        )

    lines.extend(
        [
            "",
            "## Audit Findings",
            f"- Missing owners: {', '.join(audit.missing_owners) or 'none'}",
            f"- Invalid priorities: {', '.join(audit.invalid_priorities) or 'none'}",
            f"- Invalid categories: {', '.join(audit.invalid_categories) or 'none'}",
            f"- Unresolved P0 findings: {', '.join(audit.unresolved_p0_findings) or 'none'}",
        ]
    )
    return "\n".join(lines)


@dataclass
class NarrativeSection:
    heading: str
    body: str
    evidence: List[str] = field(default_factory=list)
    callouts: List[str] = field(default_factory=list)

    @property
    def ready(self) -> bool:
        return bool(self.heading.strip()) and bool(self.body.strip())


@dataclass
class ReportStudio:
    name: str
    issue_id: str
    audience: str
    period: str
    summary: str
    sections: List[NarrativeSection] = field(default_factory=list)
    action_items: List[str] = field(default_factory=list)
    source_reports: List[str] = field(default_factory=list)

    @property
    def ready(self) -> bool:
        return bool(self.summary.strip()) and bool(self.sections) and all(section.ready for section in self.sections)

    @property
    def recommendation(self) -> str:
        return "publish" if self.ready else "draft"

    @property
    def export_slug(self) -> str:
        return _slugify(self.name) or "report-studio"


@dataclass
class ReportStudioArtifacts:
    root_dir: str
    markdown_path: str
    html_path: str
    text_path: str


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
    actions: List["ConsoleAction"] = field(default_factory=list)


@dataclass
class TriageSimilarityEvidence:
    related_run_id: str
    related_task_id: str
    score: float
    reason: str


@dataclass
class TriageSuggestion:
    label: str
    action: str
    owner: str
    confidence: float
    evidence: List[TriageSimilarityEvidence] = field(default_factory=list)
    feedback_status: str = "pending"


@dataclass
class TriageInboxItem:
    run_id: str
    task_id: str
    source: str
    status: str
    severity: str
    owner: str
    summary: str
    submitted_at: str
    suggestions: List[TriageSuggestion] = field(default_factory=list)


@dataclass
class TriageFeedbackRecord:
    run_id: str
    action: str
    decision: str
    actor: str
    notes: str = ""
    timestamp: str = field(default_factory=_utc_now_iso)


@dataclass
class AutoTriageCenter:
    name: str
    period: str
    findings: List[TriageFinding] = field(default_factory=list)
    inbox: List[TriageInboxItem] = field(default_factory=list)
    feedback: List[TriageFeedbackRecord] = field(default_factory=list)

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
        if self.feedback_counts["rejected"] > self.feedback_counts["accepted"]:
            return "retune-suggestions"
        if counts["high"]:
            return "review-queue"
        return "monitor"

    @property
    def inbox_size(self) -> int:
        return len(self.inbox)

    @property
    def feedback_counts(self) -> dict[str, int]:
        counts = {"accepted": 0, "rejected": 0, "pending": 0}
        for record in self.feedback:
            counts[record.decision] = counts.get(record.decision, 0) + 1
        pending = sum(
            1
            for item in self.inbox
            for suggestion in item.suggestions
            if suggestion.feedback_status == "pending"
        )
        counts["pending"] = pending
        return counts


@dataclass
class TakeoverRequest:
    run_id: str
    task_id: str
    source: str
    target_team: str
    status: str
    reason: str
    required_approvals: List[str] = field(default_factory=list)
    actions: List["ConsoleAction"] = field(default_factory=list)


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


@dataclass
class SharedViewFilter:
    label: str
    value: str


@dataclass
class SharedViewContext:
    filters: List[SharedViewFilter] = field(default_factory=list)
    result_count: Optional[int] = None
    loading: bool = False
    errors: List[str] = field(default_factory=list)
    partial_data: List[str] = field(default_factory=list)
    empty_message: str = "No records match the current filters."
    last_updated: str = ""
    collaboration: Optional[CollaborationThread] = None

    @property
    def state(self) -> str:
        if self.loading:
            return "loading"
        if self.errors and not self.result_count:
            return "error"
        if self.result_count == 0 and not self.partial_data:
            return "empty"
        if self.errors or self.partial_data:
            return "partial-data"
        return "ready"

    @property
    def summary(self) -> str:
        if self.state == "loading":
            return "Loading data for the current filters."
        if self.state == "error":
            return "Unable to load data for the current filters."
        if self.state == "empty":
            return self.empty_message
        if self.state == "partial-data":
            return "Showing partial data while one or more sources are unavailable."
        return "Data is current for the selected filters."



@dataclass
class OrchestrationCanvas:
    task_id: str
    run_id: str
    collaboration_mode: str
    departments: List[str] = field(default_factory=list)
    required_approvals: List[str] = field(default_factory=list)
    tier: str = "standard"
    upgrade_required: bool = False
    blocked_departments: List[str] = field(default_factory=list)
    handoff_team: str = "none"
    handoff_status: str = "none"
    handoff_reason: str = ""
    active_tools: List[str] = field(default_factory=list)
    entitlement_status: str = "included"
    billing_model: str = "standard-included"
    estimated_cost_usd: float = 0.0
    included_usage_units: int = 0
    overage_usage_units: int = 0
    overage_cost_usd: float = 0.0
    actions: List["ConsoleAction"] = field(default_factory=list)
    collaboration: Optional[CollaborationThread] = None

    @property
    def recommendation(self) -> str:
        if self.collaboration is not None and self.collaboration.open_comment_count:
            return "resolve-flow-comments"
        if self.handoff_team == "security":
            return "review-security-takeover"
        if self.upgrade_required:
            return "resolve-entitlement-gap"
        if self.overage_cost_usd > 0:
            return "review-billing-overage"
        if len(self.departments) > 1:
            return "continue-cross-team-execution"
        return "monitor"



@dataclass
class OrchestrationPortfolio:
    name: str
    period: str
    canvases: List[OrchestrationCanvas] = field(default_factory=list)
    takeover_queue: Optional[TakeoverQueue] = None

    @property
    def total_runs(self) -> int:
        return len(self.canvases)

    @property
    def collaboration_modes(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.collaboration_mode] = counts.get(canvas.collaboration_mode, 0) + 1
        return counts

    @property
    def tier_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.tier] = counts.get(canvas.tier, 0) + 1
        return counts

    @property
    def upgrade_required_count(self) -> int:
        return sum(1 for canvas in self.canvases if canvas.upgrade_required)

    @property
    def active_handoffs(self) -> int:
        return sum(1 for canvas in self.canvases if canvas.handoff_team != "none")

    @property
    def entitlement_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.entitlement_status] = counts.get(canvas.entitlement_status, 0) + 1
        return counts

    @property
    def billing_model_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for canvas in self.canvases:
            counts[canvas.billing_model] = counts.get(canvas.billing_model, 0) + 1
        return counts

    @property
    def total_estimated_cost_usd(self) -> float:
        return round(sum(canvas.estimated_cost_usd for canvas in self.canvases), 2)

    @property
    def total_overage_cost_usd(self) -> float:
        return round(sum(canvas.overage_cost_usd for canvas in self.canvases), 2)

    @property
    def recommendation(self) -> str:
        if self.takeover_queue is not None and self.takeover_queue.recommendation == "expedite-security-review":
            return "stabilize-security-takeovers"
        if self.upgrade_required_count:
            return "close-entitlement-gaps"
        if self.active_handoffs:
            return "manage-cross-team-flow"
        return "monitor"

@dataclass(frozen=True)
class ConsoleAction:
    action_id: str
    label: str
    target: str
    enabled: bool = True
    reason: str = ""

    @property
    def state(self) -> str:
        return "enabled" if self.enabled else "disabled"


@dataclass
class BillingRunCharge:
    run_id: str
    task_id: str
    billing_model: str
    entitlement_status: str
    estimated_cost_usd: float
    included_usage_units: int = 0
    overage_usage_units: int = 0
    overage_cost_usd: float = 0.0
    blocked_capabilities: List[str] = field(default_factory=list)
    handoff_team: str = "none"
    recommendation: str = "monitor"


@dataclass
class BillingEntitlementsPage:
    workspace_name: str
    plan_name: str
    billing_period: str
    charges: List[BillingRunCharge] = field(default_factory=list)

    @property
    def run_count(self) -> int:
        return len(self.charges)

    @property
    def total_estimated_cost_usd(self) -> float:
        return round(sum(charge.estimated_cost_usd for charge in self.charges), 2)

    @property
    def total_included_usage_units(self) -> int:
        return sum(charge.included_usage_units for charge in self.charges)

    @property
    def total_overage_usage_units(self) -> int:
        return sum(charge.overage_usage_units for charge in self.charges)

    @property
    def total_overage_cost_usd(self) -> float:
        return round(sum(charge.overage_cost_usd for charge in self.charges), 2)

    @property
    def upgrade_required_count(self) -> int:
        return sum(1 for charge in self.charges if charge.entitlement_status == "upgrade-required")

    @property
    def billing_model_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for charge in self.charges:
            counts[charge.billing_model] = counts.get(charge.billing_model, 0) + 1
        return counts

    @property
    def entitlement_counts(self) -> dict[str, int]:
        counts: dict[str, int] = {}
        for charge in self.charges:
            counts[charge.entitlement_status] = counts.get(charge.entitlement_status, 0) + 1
        return counts

    @property
    def blocked_capabilities(self) -> List[str]:
        capabilities: List[str] = []
        for charge in self.charges:
            for capability in charge.blocked_capabilities:
                if capability not in capabilities:
                    capabilities.append(capability)
        return capabilities

    @property
    def recommendation(self) -> str:
        if self.upgrade_required_count:
            return "resolve-plan-gaps"
        if self.total_overage_cost_usd > 0:
            return "optimize-billed-usage"
        if any(charge.handoff_team != "none" for charge in self.charges):
            return "monitor-shared-capacity"
        return "healthy"


def render_issue_validation_report(issue_id: str, version: str, environment: str, summary: str) -> str:
    return f"""# Issue Validation Report\n\n- Issue ID: {issue_id}\n- 版本号: {version}\n- 测试环境: {environment}\n- 生成时间: {_utc_now_iso()}\n\n## 结论\n\n{summary}\n"""


def render_report_studio_report(studio: ReportStudio) -> str:
    lines = [
        "# Report Studio",
        "",
        f"- Name: {studio.name}",
        f"- Issue ID: {studio.issue_id}",
        f"- Audience: {studio.audience}",
        f"- Period: {studio.period}",
        f"- Sections: {len(studio.sections)}",
        f"- Recommendation: {studio.recommendation}",
        "",
        "## Narrative Summary",
        "",
        studio.summary or "No summary drafted.",
        "",
        "## Sections",
        "",
    ]

    if studio.sections:
        for section in studio.sections:
            lines.append(f"### {section.heading}")
            lines.append("")
            lines.append(section.body or "No narrative drafted.")
            lines.append("")
            lines.append("- Evidence: " + (", ".join(section.evidence) if section.evidence else "None"))
            lines.append("- Callouts: " + (", ".join(section.callouts) if section.callouts else "None"))
            lines.append("")
    else:
        lines.append("- None")
        lines.append("")

    lines.append("## Action Items")
    lines.append("")
    if studio.action_items:
        lines.extend(f"- {item}" for item in studio.action_items)
    else:
        lines.append("- None")
    lines.append("")

    lines.append("## Sources")
    lines.append("")
    if studio.source_reports:
        lines.extend(f"- {path}" for path in studio.source_reports)
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_report_studio_plain_text(studio: ReportStudio) -> str:
    lines = [
        f"{studio.name} ({studio.issue_id})",
        f"Audience: {studio.audience}",
        f"Period: {studio.period}",
        f"Recommendation: {studio.recommendation}",
        "",
        studio.summary or "No summary drafted.",
        "",
    ]
    for section in studio.sections:
        lines.append(section.heading.upper())
        lines.append(section.body or "No narrative drafted.")
        if section.callouts:
            lines.append("Callouts: " + "; ".join(section.callouts))
        if section.evidence:
            lines.append("Evidence: " + "; ".join(section.evidence))
        lines.append("")

    if studio.action_items:
        lines.append("Action Items:")
        lines.extend(f"- {item}" for item in studio.action_items)
        lines.append("")

    return "\n".join(lines).rstrip() + "\n"


def render_report_studio_html(studio: ReportStudio) -> str:
    section_html = "".join(
        f"""
        <section class="section">
          <h2>{escape(section.heading)}</h2>
          <p>{escape(section.body)}</p>
          <p class="meta">Evidence: {escape(', '.join(section.evidence) if section.evidence else 'None')}</p>
          <p class="meta">Callouts: {escape(', '.join(section.callouts) if section.callouts else 'None')}</p>
        </section>
        """
        for section in studio.sections
    )
    action_html = "".join(f"<li>{escape(item)}</li>" for item in studio.action_items) or "<li>None</li>"
    source_html = "".join(f"<li>{escape(path)}</li>" for path in studio.source_reports) or "<li>None</li>"
    return f"""<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{escape(studio.name)}</title>
    <style>
      body {{ font-family: Georgia, 'Times New Roman', serif; margin: 40px auto; max-width: 840px; color: #1f2933; line-height: 1.6; }}
      h1, h2 {{ font-family: 'Avenir Next', 'Segoe UI', sans-serif; }}
      .meta {{ color: #52606d; font-size: 0.95rem; }}
      .summary {{ padding: 16px 20px; background: #f7f3e8; border-left: 4px solid #c58b32; }}
      .section {{ margin-top: 28px; }}
    </style>
  </head>
  <body>
    <header>
      <p class="meta">{escape(studio.issue_id)} · {escape(studio.audience)} · {escape(studio.period)}</p>
      <h1>{escape(studio.name)}</h1>
      <p class="meta">Recommendation: {escape(studio.recommendation)}</p>
    </header>
    <section class="summary">
      <h2>Narrative Summary</h2>
      <p>{escape(studio.summary or 'No summary drafted.')}</p>
    </section>
    {section_html or '<section class="section"><p>No sections drafted.</p></section>'}
    <section class="section">
      <h2>Action Items</h2>
      <ul>{action_html}</ul>
    </section>
    <section class="section">
      <h2>Sources</h2>
      <ul>{source_html}</ul>
    </section>
  </body>
</html>
"""


def build_launch_checklist(
    issue_id: str,
    documentation: List[DocumentationArtifact],
    items: List[LaunchChecklistItem],
) -> LaunchChecklist:
    return LaunchChecklist(issue_id=issue_id, documentation=documentation, items=items)


def build_final_delivery_checklist(
    issue_id: str,
    required_outputs: List[DocumentationArtifact],
    recommended_documentation: List[DocumentationArtifact],
) -> FinalDeliveryChecklist:
    return FinalDeliveryChecklist(
        issue_id=issue_id,
        required_outputs=required_outputs,
        recommended_documentation=recommended_documentation,
    )


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


def render_final_delivery_checklist_report(checklist: FinalDeliveryChecklist) -> str:
    lines = [
        "# Final Delivery Checklist",
        "",
        f"- Issue ID: {checklist.issue_id}",
        f"- Required Outputs Generated: {checklist.generated_required_outputs}/{len(checklist.required_outputs)}",
        f"- Recommended Docs Generated: {checklist.generated_recommended_documentation}/{len(checklist.recommended_documentation)}",
        f"- Ready: {checklist.ready}",
        "",
        "## Required Outputs",
        "",
    ]

    if checklist.required_outputs:
        for artifact in checklist.required_outputs:
            lines.append(
                f"- {artifact.name}: available={artifact.available} path={artifact.path}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Recommended Documentation", ""])
    if checklist.recommended_documentation:
        for artifact in checklist.recommended_documentation:
            lines.append(
                f"- {artifact.name}: available={artifact.available} path={artifact.path}"
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


def write_report_studio_bundle(root_dir: str, studio: ReportStudio) -> ReportStudioArtifacts:
    base = Path(root_dir)
    base.mkdir(parents=True, exist_ok=True)
    markdown_path = str(base / f"{studio.export_slug}.md")
    html_path = str(base / f"{studio.export_slug}.html")
    text_path = str(base / f"{studio.export_slug}.txt")
    write_report(markdown_path, render_report_studio_report(studio))
    write_report(html_path, render_report_studio_html(studio))
    write_report(text_path, render_report_studio_plain_text(studio))
    return ReportStudioArtifacts(
        root_dir=str(base),
        markdown_path=markdown_path,
        html_path=html_path,
        text_path=text_path,
    )


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
    final_delivery_checklist: Optional[FinalDeliveryChecklist] = None,
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

    if final_delivery_checklist is not None and not final_delivery_checklist.ready:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="final delivery checklist incomplete; required outputs missing",
            report_path=resolved_path,
        )

    if launch_checklist is not None and not launch_checklist.ready:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=False,
            reason="launch checklist incomplete; linked documentation missing or empty",
            report_path=resolved_path,
        )

    if final_delivery_checklist is not None:
        return IssueClosureDecision(
            issue_id=issue_id,
            allowed=True,
            reason="validation report and final delivery checklist requirements satisfied; issue can be closed",
            report_path=resolved_path,
        )

    return IssueClosureDecision(
        issue_id=issue_id,
        allowed=True,
        reason="validation report and launch checklist requirements satisfied; issue can be closed",
        report_path=resolved_path,
    )

def build_console_actions(
    target: str,
    *,
    allow_retry: bool = True,
    retry_reason: str = "",
    allow_pause: bool = True,
    pause_reason: str = "",
    allow_reassign: bool = True,
    reassign_reason: str = "",
    allow_escalate: bool = True,
    escalate_reason: str = "",
) -> List[ConsoleAction]:
    return [
        ConsoleAction("drill-down", "Drill Down", target),
        ConsoleAction("export", "Export", target),
        ConsoleAction("add-note", "Add Note", target),
        ConsoleAction("escalate", "Escalate", target, enabled=allow_escalate, reason=escalate_reason),
        ConsoleAction("retry", "Retry", target, enabled=allow_retry, reason=retry_reason),
        ConsoleAction("pause", "Pause", target, enabled=allow_pause, reason=pause_reason),
        ConsoleAction("reassign", "Reassign", target, enabled=allow_reassign, reason=reassign_reason),
        ConsoleAction("audit", "Audit Trail", target),
    ]


def render_console_actions(actions: List[ConsoleAction]) -> str:
    if not actions:
        return "none"

    rendered: List[str] = []
    for action in actions:
        detail = f"{action.label} [{action.action_id}] state={action.state} target={action.target}"
        if action.reason:
            detail += f" reason={action.reason}"
        rendered.append(detail)
    return "; ".join(rendered)


def _default_canvas_actions(canvas: OrchestrationCanvas) -> List[ConsoleAction]:
    return build_console_actions(
        canvas.run_id,
        allow_retry=canvas.handoff_status != "pending",
        retry_reason="" if canvas.handoff_status != "pending" else "pending handoff must be resolved before retry",
        allow_pause=canvas.handoff_status != "completed",
        pause_reason="" if canvas.handoff_status != "completed" else "completed handoff runs cannot be paused",
        allow_reassign=canvas.handoff_team != "none",
        reassign_reason="" if canvas.handoff_team != "none" else "reassign is available after a handoff exists",
        allow_escalate=canvas.upgrade_required,
        escalate_reason="" if canvas.upgrade_required else "escalate when policy requires an entitlement or approval upgrade",
    )


def build_auto_triage_center(
    runs: List[TaskRun],
    name: str = "Auto Triage Center",
    period: str = "current",
    feedback: Optional[List[TriageFeedbackRecord]] = None,
) -> AutoTriageCenter:
    findings: List[TriageFinding] = []
    inbox: List[TriageInboxItem] = []
    feedback = feedback or []
    for run in runs:
        if not _run_requires_triage(run):
            continue

        severity = _triage_severity(run)
        owner = _triage_owner(run)
        reason = _triage_reason(run)
        next_action = _triage_next_action(severity, owner)
        suggestions = _build_triage_suggestions(run, runs, severity, owner, feedback)
        findings.append(
            TriageFinding(
                run_id=run.run_id,
                task_id=run.task_id,
                source=run.source,
                severity=severity,
                owner=owner,
                status=run.status,
                reason=reason,
                next_action=next_action,
                actions=build_console_actions(
                    run.run_id,
                    allow_retry=severity == "critical" and owner != "security",
                    retry_reason="" if severity == "critical" and owner != "security" else "retry available after owner review",
                    allow_pause=run.status not in {"failed", "completed", "approved"},
                    pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
                    allow_reassign=owner != "security",
                    reassign_reason="" if owner != "security" else "security-owned findings stay with the security queue",
                ),
            )
        )
        inbox.append(
            TriageInboxItem(
                run_id=run.run_id,
                task_id=run.task_id,
                source=run.source,
                status=run.status,
                severity=severity,
                owner=owner,
                summary=reason,
                submitted_at=run.ended_at or run.started_at,
                suggestions=suggestions,
            )
        )

    severity_rank = {"critical": 0, "high": 1, "medium": 2}
    findings.sort(key=lambda finding: (severity_rank[finding.severity], finding.owner, finding.run_id))
    inbox.sort(key=lambda item: (severity_rank[item.severity], item.owner, item.run_id))
    return AutoTriageCenter(name=name, period=period, findings=findings, inbox=inbox, feedback=feedback)


def render_shared_view_context(view: Optional[SharedViewContext]) -> List[str]:
    if view is None:
        return []

    lines = [
        "## View State",
        "",
        f"- State: {view.state}",
        f"- Summary: {view.summary}",
    ]
    if view.result_count is not None:
        lines.append(f"- Result Count: {view.result_count}")
    if view.last_updated:
        lines.append(f"- Last Updated: {view.last_updated}")

    lines.extend(["", "## Filters", ""])
    if view.filters:
        lines.extend(f"- {item.label}: {item.value}" for item in view.filters)
    else:
        lines.append("- None")

    if view.errors:
        lines.extend(["", "## Errors", ""])
        lines.extend(f"- {message}" for message in view.errors)

    if view.partial_data:
        lines.extend(["", "## Partial Data", ""])
        lines.extend(f"- {message}" for message in view.partial_data)

    lines.extend(render_collaboration_lines(view.collaboration))
    lines.append("")
    return lines


def render_auto_triage_center_report(
    center: AutoTriageCenter,
    total_runs: Optional[int] = None,
    view: Optional[SharedViewContext] = None,
) -> str:
    severity = center.severity_counts
    owners = center.owner_counts
    feedback = center.feedback_counts
    lines = [
        "# Auto Triage Center",
        "",
        f"- Center: {center.name}",
        f"- Period: {center.period}",
        f"- Flagged Runs: {center.flagged_runs}",
        f"- Inbox Size: {center.inbox_size}",
        f"- Total Runs: {total_runs if total_runs is not None else center.flagged_runs}",
        f"- Recommendation: {center.recommendation}",
        f"- Severity Mix: critical={severity['critical']} high={severity['high']} medium={severity['medium']}",
        f"- Owner Mix: security={owners['security']} engineering={owners['engineering']} operations={owners['operations']}",
        f"- Feedback Loop: accepted={feedback['accepted']} rejected={feedback['rejected']} pending={feedback['pending']}",
        "",
        "## Queue",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if center.findings:
        for finding in center.findings:
            lines.append(
                f"- {finding.run_id}: severity={finding.severity} owner={finding.owner} status={finding.status} "
                f"task={finding.task_id} reason={finding.reason} next={finding.next_action} actions={render_console_actions(finding.actions)}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Inbox", ""])
    if center.inbox:
        for item in center.inbox:
            suggestion_summary = "; ".join(
                f"{suggestion.action}({suggestion.feedback_status}, confidence={suggestion.confidence:.2f})"
                for suggestion in item.suggestions
            ) or "none"
            evidence_summary = ", ".join(
                f"{e.related_run_id}:{e.score:.2f}" for suggestion in item.suggestions for e in suggestion.evidence
            ) or "none"
            lines.append(
                f"- {item.run_id}: severity={item.severity} owner={item.owner} status={item.status} "
                f"summary={item.summary} suggestions={suggestion_summary} similar={evidence_summary}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def build_orchestration_portfolio(
    canvases: List[OrchestrationCanvas],
    name: str = "Cross-Department Orchestration",
    period: str = "current",
    takeover_queue: Optional[TakeoverQueue] = None,
) -> OrchestrationPortfolio:
    normalized_canvases = [
        canvas
        if canvas.actions
        else OrchestrationCanvas(
            task_id=canvas.task_id,
            run_id=canvas.run_id,
            collaboration_mode=canvas.collaboration_mode,
            departments=canvas.departments,
            required_approvals=canvas.required_approvals,
            tier=canvas.tier,
            upgrade_required=canvas.upgrade_required,
            blocked_departments=canvas.blocked_departments,
            handoff_team=canvas.handoff_team,
            handoff_status=canvas.handoff_status,
            handoff_reason=canvas.handoff_reason,
            active_tools=canvas.active_tools,
            entitlement_status=canvas.entitlement_status,
            billing_model=canvas.billing_model,
            estimated_cost_usd=canvas.estimated_cost_usd,
            included_usage_units=canvas.included_usage_units,
            overage_usage_units=canvas.overage_usage_units,
            overage_cost_usd=canvas.overage_cost_usd,
            actions=_default_canvas_actions(canvas),
        )
        for canvas in canvases
    ]
    return OrchestrationPortfolio(
        name=name,
        period=period,
        canvases=sorted(normalized_canvases, key=lambda canvas: canvas.run_id),
        takeover_queue=takeover_queue,
    )


def build_billing_entitlements_page(
    portfolio: OrchestrationPortfolio,
    *,
    workspace_name: str = "BigClaw Cloud",
    plan_name: str = "Standard",
    billing_period: Optional[str] = None,
) -> BillingEntitlementsPage:
    return BillingEntitlementsPage(
        workspace_name=workspace_name,
        plan_name=plan_name,
        billing_period=billing_period or portfolio.period,
        charges=[
            BillingRunCharge(
                run_id=canvas.run_id,
                task_id=canvas.task_id,
                billing_model=canvas.billing_model,
                entitlement_status=canvas.entitlement_status,
                estimated_cost_usd=canvas.estimated_cost_usd,
                included_usage_units=canvas.included_usage_units,
                overage_usage_units=canvas.overage_usage_units,
                overage_cost_usd=canvas.overage_cost_usd,
                blocked_capabilities=list(canvas.blocked_departments),
                handoff_team=canvas.handoff_team,
                recommendation=canvas.recommendation,
            )
            for canvas in portfolio.canvases
        ],
    )


def render_orchestration_portfolio_report(
    portfolio: OrchestrationPortfolio,
    view: Optional[SharedViewContext] = None,
) -> str:
    collaboration = " ".join(
        f"{mode}={count}" for mode, count in sorted(portfolio.collaboration_modes.items())
    ) or "none"
    tiers = " ".join(
        f"{tier}={count}" for tier, count in sorted(portfolio.tier_counts.items())
    ) or "none"
    entitlements = " ".join(
        f"{status}={count}" for status, count in sorted(portfolio.entitlement_counts.items())
    ) or "none"
    billing_models = " ".join(
        f"{model}={count}" for model, count in sorted(portfolio.billing_model_counts.items())
    ) or "none"
    takeover_summary = (
        f"pending={portfolio.takeover_queue.pending_requests} recommendation={portfolio.takeover_queue.recommendation}"
        if portfolio.takeover_queue is not None
        else "none"
    )
    lines = [
        "# Orchestration Portfolio Report",
        "",
        f"- Portfolio: {portfolio.name}",
        f"- Period: {portfolio.period}",
        f"- Total Runs: {portfolio.total_runs}",
        f"- Recommendation: {portfolio.recommendation}",
        f"- Collaboration Mix: {collaboration}",
        f"- Tier Mix: {tiers}",
        f"- Entitlement Mix: {entitlements}",
        f"- Billing Models: {billing_models}",
        f"- Upgrade Required Count: {portfolio.upgrade_required_count}",
        f"- Estimated Cost (USD): {portfolio.total_estimated_cost_usd:.2f}",
        f"- Overage Cost (USD): {portfolio.total_overage_cost_usd:.2f}",
        f"- Active Handoffs: {portfolio.active_handoffs}",
        f"- Takeover Queue: {takeover_summary}",
        "",
        "## Runs",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if portfolio.canvases:
        for canvas in portfolio.canvases:
            collaboration_summary = (
                f"comments={len(canvas.collaboration.comments)} decisions={len(canvas.collaboration.decisions)}"
                if canvas.collaboration is not None
                else "comments=0 decisions=0"
            )
            lines.append(
                f"- {canvas.run_id}: mode={canvas.collaboration_mode} tier={canvas.tier} "
                f"entitlement={canvas.entitlement_status} billing={canvas.billing_model} "
                f"estimated_cost_usd={canvas.estimated_cost_usd:.2f} overage_cost_usd={canvas.overage_cost_usd:.2f} "
                f"upgrade_required={canvas.upgrade_required} handoff={canvas.handoff_team} "
                f"collaboration={collaboration_summary} recommendation={canvas.recommendation} "
                f"actions={render_console_actions(canvas.actions)}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_billing_entitlements_report(
    page: BillingEntitlementsPage,
    view: Optional[SharedViewContext] = None,
) -> str:
    entitlements = " ".join(
        f"{status}={count}" for status, count in sorted(page.entitlement_counts.items())
    ) or "none"
    billing_models = " ".join(
        f"{model}={count}" for model, count in sorted(page.billing_model_counts.items())
    ) or "none"
    blocked = ", ".join(page.blocked_capabilities) if page.blocked_capabilities else "none"
    lines = [
        "# Billing & Entitlements Report",
        "",
        f"- Workspace: {page.workspace_name}",
        f"- Plan: {page.plan_name}",
        f"- Billing Period: {page.billing_period}",
        f"- Runs: {page.run_count}",
        f"- Recommendation: {page.recommendation}",
        f"- Entitlement Mix: {entitlements}",
        f"- Billing Models: {billing_models}",
        f"- Included Usage Units: {page.total_included_usage_units}",
        f"- Overage Usage Units: {page.total_overage_usage_units}",
        f"- Estimated Cost (USD): {page.total_estimated_cost_usd:.2f}",
        f"- Overage Cost (USD): {page.total_overage_cost_usd:.2f}",
        f"- Upgrade Required Count: {page.upgrade_required_count}",
        f"- Blocked Capabilities: {blocked}",
        "",
        "## Charges",
        "",
    ]
    lines.extend(render_shared_view_context(view))

    if page.charges:
        for charge in page.charges:
            blocked_capabilities = ", ".join(charge.blocked_capabilities) if charge.blocked_capabilities else "none"
            lines.append(
                f"- {charge.run_id}: task={charge.task_id} entitlement={charge.entitlement_status} "
                f"billing={charge.billing_model} included_units={charge.included_usage_units} "
                f"overage_units={charge.overage_usage_units} estimated_cost_usd={charge.estimated_cost_usd:.2f} "
                f"overage_cost_usd={charge.overage_cost_usd:.2f} blocked={blocked_capabilities} "
                f"handoff={charge.handoff_team} recommendation={charge.recommendation}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def render_orchestration_overview_page(portfolio: OrchestrationPortfolio) -> str:
    def render_items(items: List[str]) -> str:
        if not items:
            return "<li>None</li>"
        return "".join(f"<li>{item}</li>" for item in items)

    collaboration = render_items(
        [f"<strong>{escape(mode)}</strong>: {count}" for mode, count in sorted(portfolio.collaboration_modes.items())]
    )
    tiers = render_items(
        [f"<strong>{escape(tier)}</strong>: {count}" for tier, count in sorted(portfolio.tier_counts.items())]
    )
    entitlements = render_items(
        [
            f"<strong>{escape(status)}</strong>: {count}"
            for status, count in sorted(portfolio.entitlement_counts.items())
        ]
    )
    billing_models = render_items(
        [
            f"<strong>{escape(model)}</strong>: {count}"
            for model, count in sorted(portfolio.billing_model_counts.items())
        ]
    )
    runs = render_items(
        [
            f"<strong>{escape(canvas.run_id)}</strong> · mode={escape(canvas.collaboration_mode)} · tier={escape(canvas.tier)} · entitlement={escape(canvas.entitlement_status)} · billing={escape(canvas.billing_model)} · cost=${canvas.estimated_cost_usd:.2f} · handoff={escape(canvas.handoff_team)} · comments={len(canvas.collaboration.comments) if canvas.collaboration is not None else 0} · decisions={len(canvas.collaboration.decisions) if canvas.collaboration is not None else 0} · recommendation={escape(canvas.recommendation)} · actions={escape(render_console_actions(canvas.actions or _default_canvas_actions(canvas)))}"
            for canvas in portfolio.canvases
        ]
    )
    takeover = "none"
    if portfolio.takeover_queue is not None:
        takeover = (
            f"pending={portfolio.takeover_queue.pending_requests} recommendation={portfolio.takeover_queue.recommendation}"
        )

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Orchestration Overview · {escape(portfolio.name)}</title>
  <style>
    :root {{ color-scheme: light dark; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }}
    body {{ margin: 2rem auto; max-width: 1080px; padding: 0 1rem 3rem; line-height: 1.5; }}
    .grid {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 0.75rem; margin: 1rem 0 1.5rem; }}
    .card {{ border: 1px solid #cbd5e1; border-radius: 10px; padding: 0.9rem; background: rgba(148, 163, 184, 0.08); }}
    h1, h2 {{ margin-bottom: 0.5rem; }}
    ul {{ padding-left: 1.2rem; }}
    code {{ font-size: 0.95em; }}
  </style>
</head>
<body>
  <h1>Orchestration Overview</h1>
  <p>{escape(portfolio.name)} · {escape(portfolio.period)}</p>
    <div class="grid">
      <div class="card"><strong>Total Runs</strong><br>{portfolio.total_runs}</div>
      <div class="card"><strong>Recommendation</strong><br>{escape(portfolio.recommendation)}</div>
      <div class="card"><strong>Upgrade Required</strong><br>{portfolio.upgrade_required_count}</div>
      <div class="card"><strong>Estimated Cost</strong><br>${portfolio.total_estimated_cost_usd:.2f}</div>
      <div class="card"><strong>Overage Cost</strong><br>${portfolio.total_overage_cost_usd:.2f}</div>
      <div class="card"><strong>Active Handoffs</strong><br>{portfolio.active_handoffs}</div>
      <div class="card"><strong>Takeover Queue</strong><br>{escape(takeover)}</div>
    </div>
  <h2>Collaboration Mix</h2>
  <ul>{collaboration}</ul>
  <h2>Tier Mix</h2>
  <ul>{tiers}</ul>
  <h2>Entitlement Mix</h2>
  <ul>{entitlements}</ul>
  <h2>Billing Models</h2>
  <ul>{billing_models}</ul>
  <h2>Runs</h2>
  <ul>{runs}</ul>
</body>
</html>
"""


def render_billing_entitlements_page(page: BillingEntitlementsPage) -> str:
    def render_items(items: List[str]) -> str:
        if not items:
            return "<li>None</li>"
        return "".join(f"<li>{item}</li>" for item in items)

    entitlements = render_items(
        [f"<strong>{escape(status)}</strong>: {count}" for status, count in sorted(page.entitlement_counts.items())]
    )
    billing_models = render_items(
        [f"<strong>{escape(model)}</strong>: {count}" for model, count in sorted(page.billing_model_counts.items())]
    )
    blocked = render_items([escape(capability) for capability in page.blocked_capabilities])
    charges = render_items(
        [
            f"<strong>{escape(charge.run_id)}</strong> · task={escape(charge.task_id)} · entitlement={escape(charge.entitlement_status)} · billing={escape(charge.billing_model)} · included={charge.included_usage_units} · overage={charge.overage_usage_units} · cost=${charge.estimated_cost_usd:.2f} · recommendation={escape(charge.recommendation)}"
            for charge in page.charges
        ]
    )

    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Billing & Entitlements · {escape(page.workspace_name)}</title>
  <style>
    :root {{
      color-scheme: light;
      --ink: #102033;
      --muted: #5c6876;
      --canvas: #f6f0e4;
      --panel: rgba(255, 252, 246, 0.9);
      --line: rgba(16, 32, 51, 0.12);
      --accent: #b45309;
      font-family: "Avenir Next", "Segoe UI", sans-serif;
    }}
    * {{ box-sizing: border-box; }}
    body {{
      margin: 0;
      color: var(--ink);
      background:
        radial-gradient(circle at top right, rgba(22, 101, 52, 0.12), transparent 24%),
        radial-gradient(circle at left center, rgba(180, 83, 9, 0.12), transparent 28%),
        linear-gradient(180deg, #fffaf2 0%, var(--canvas) 100%);
    }}
    main {{ width: min(1180px, calc(100% - 2rem)); margin: 0 auto; padding: 2rem 0 3rem; }}
    .hero {{
      border: 1px solid var(--line);
      border-radius: 28px;
      background: linear-gradient(135deg, rgba(255,255,255,0.82), rgba(255,247,237,0.94));
      box-shadow: 0 20px 48px rgba(16, 32, 51, 0.08);
      padding: 1.5rem;
    }}
    .eyebrow {{
      display: inline-block;
      font: 600 0.75rem/1.2 "SFMono-Regular", Consolas, monospace;
      letter-spacing: 0.12em;
      text-transform: uppercase;
      color: var(--muted);
      margin-bottom: 0.75rem;
    }}
    h1, h2, p {{ margin: 0; }}
    .hero p {{ color: var(--muted); line-height: 1.6; max-width: 70ch; margin-top: 0.55rem; }}
    .metrics {{
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
      gap: 0.85rem;
      margin-top: 1.35rem;
    }}
    .card, .surface {{
      border: 1px solid var(--line);
      border-radius: 20px;
      background: var(--panel);
      padding: 1rem;
    }}
    .card strong {{ display: block; font-size: 1.2rem; margin-top: 0.35rem; }}
    .card span {{
      color: var(--muted);
      font: 500 0.78rem/1.4 "SFMono-Regular", Consolas, monospace;
      text-transform: uppercase;
      letter-spacing: 0.08em;
    }}
    .layout {{
      display: grid;
      grid-template-columns: minmax(0, 1.35fr) minmax(280px, 0.85fr);
      gap: 1rem;
      margin-top: 1rem;
    }}
    .stack {{ display: grid; gap: 1rem; }}
    ul {{ margin: 0; padding-left: 1.15rem; }}
    li {{ margin: 0.35rem 0; }}
    .section-title {{ margin-bottom: 0.7rem; }}
    @media (max-width: 860px) {{
      .layout {{ grid-template-columns: 1fr; }}
    }}
  </style>
</head>
<body>
  <main>
    <section class="hero">
      <span class="eyebrow">Billing & Entitlements</span>
      <h1>{escape(page.workspace_name)}</h1>
      <p>{escape(page.plan_name)} plan for {escape(page.billing_period)}. Recommendation: {escape(page.recommendation)}.</p>
      <div class="metrics">
        <article class="card"><span>Runs</span><strong>{page.run_count}</strong></article>
        <article class="card"><span>Included Units</span><strong>{page.total_included_usage_units}</strong></article>
        <article class="card"><span>Overage Units</span><strong>{page.total_overage_usage_units}</strong></article>
        <article class="card"><span>Estimated Cost</span><strong>${page.total_estimated_cost_usd:.2f}</strong></article>
        <article class="card"><span>Overage Cost</span><strong>${page.total_overage_cost_usd:.2f}</strong></article>
        <article class="card"><span>Upgrade Required</span><strong>{page.upgrade_required_count}</strong></article>
      </div>
    </section>
    <section class="layout">
      <div class="stack">
        <section class="surface">
          <h2 class="section-title">Charge Feed</h2>
          <ul>{charges}</ul>
        </section>
      </div>
      <div class="stack">
        <section class="surface">
          <h2 class="section-title">Entitlement Mix</h2>
          <ul>{entitlements}</ul>
        </section>
        <section class="surface">
          <h2 class="section-title">Billing Models</h2>
          <ul>{billing_models}</ul>
        </section>
        <section class="surface">
          <h2 class="section-title">Blocked Capabilities</h2>
          <ul>{blocked}</ul>
        </section>
      </div>
    </section>
  </main>
</body>
</html>
"""


def build_orchestration_canvas_from_ledger_entry(entry: dict) -> OrchestrationCanvas:
    audits = entry.get("audits", [])

    plan_audit = _latest_named_audit(audits, "orchestration.plan")
    policy_audit = _latest_named_audit(audits, "orchestration.policy")
    handoff_audit = _latest_handoff_audit(audits)
    tool_audits = [audit for audit in audits if audit.get("action") == "tool.invoke"]

    plan_details = plan_audit.get("details", {}) if plan_audit is not None else {}
    policy_details = policy_audit.get("details", {}) if policy_audit is not None else {}
    handoff_details = handoff_audit.get("details", {}) if handoff_audit is not None else {}

    active_tools = sorted(
        {
            str(audit.get("details", {}).get("tool", ""))
            for audit in tool_audits
            if audit.get("details", {}).get("tool")
        }
    )

    return OrchestrationCanvas(
        task_id=str(entry.get("task_id", "")),
        run_id=str(entry.get("run_id", "")),
        collaboration_mode=str(plan_details.get("collaboration_mode", "single-team")),
        departments=[str(value) for value in plan_details.get("departments", [])],
        required_approvals=[str(value) for value in plan_details.get("approvals", [])],
        tier=str(policy_details.get("tier", "standard")),
        upgrade_required=bool(policy_details.get("tier") and policy_audit.get("outcome") == "upgrade-required") if policy_audit is not None else False,
        blocked_departments=[str(value) for value in policy_details.get("blocked_departments", [])],
        handoff_team=str(handoff_details.get("target_team", "none")) if handoff_audit is not None else "none",
        handoff_status=str(handoff_audit.get("outcome", "none")) if handoff_audit is not None else "none",
        handoff_reason=str(handoff_details.get("reason", "")),
        active_tools=active_tools,
        entitlement_status=str(policy_details.get("entitlement_status", "included")),
        billing_model=str(policy_details.get("billing_model", "standard-included")),
        estimated_cost_usd=float(policy_details.get("estimated_cost_usd", 0.0) or 0.0),
        included_usage_units=int(policy_details.get("included_usage_units", 0) or 0),
        overage_usage_units=int(policy_details.get("overage_usage_units", 0) or 0),
        overage_cost_usd=float(policy_details.get("overage_cost_usd", 0.0) or 0.0),
        actions=build_console_actions(
            str(entry.get("run_id", "")),
            allow_retry=bool(handoff_audit is None or handoff_audit.get("outcome") != "pending"),
            retry_reason="" if handoff_audit is None or handoff_audit.get("outcome") != "pending" else "pending handoff must be resolved before retry",
            allow_pause=bool(handoff_audit is None or handoff_audit.get("outcome") != "completed"),
            pause_reason="" if handoff_audit is None or handoff_audit.get("outcome") != "completed" else "completed handoff runs cannot be paused",
            allow_reassign=handoff_audit is not None,
            reassign_reason="" if handoff_audit is not None else "reassign is available after a handoff exists",
            allow_escalate=bool(policy_audit is not None and policy_audit.get("outcome") == "upgrade-required"),
            escalate_reason="" if policy_audit is not None and policy_audit.get("outcome") == "upgrade-required" else "escalate when policy requires an entitlement or approval upgrade",
        ),
        collaboration=build_collaboration_thread_from_audits(audits, surface="flow", target_id=str(entry.get("run_id", ""))),
    )


def build_orchestration_canvas(
    run: TaskRun,
    plan: OrchestrationPlan,
    policy: Optional[OrchestrationPolicyDecision] = None,
    handoff_request: Optional[HandoffRequest] = None,
) -> OrchestrationCanvas:
    return OrchestrationCanvas(
        task_id=run.task_id,
        run_id=run.run_id,
        collaboration_mode=plan.collaboration_mode,
        departments=plan.departments,
        required_approvals=plan.required_approvals,
        tier=policy.tier if policy is not None else "standard",
        upgrade_required=policy.upgrade_required if policy is not None else False,
        blocked_departments=policy.blocked_departments if policy is not None else [],
        handoff_team=handoff_request.target_team if handoff_request is not None else "none",
        handoff_status=handoff_request.status if handoff_request is not None else "none",
        handoff_reason=handoff_request.reason if handoff_request is not None else "",
        active_tools=sorted({str(entry.details.get("tool", "")) for entry in run.audits if entry.action == "tool.invoke" and entry.details.get("tool")}),
        entitlement_status=policy.entitlement_status if policy is not None else "included",
        billing_model=policy.billing_model if policy is not None else "standard-included",
        estimated_cost_usd=policy.estimated_cost_usd if policy is not None else 0.0,
        included_usage_units=policy.included_usage_units if policy is not None else 0,
        overage_usage_units=policy.overage_usage_units if policy is not None else 0,
        overage_cost_usd=policy.overage_cost_usd if policy is not None else 0.0,
        actions=build_console_actions(
            run.run_id,
            allow_retry=handoff_request is None or handoff_request.status != "pending",
            retry_reason="" if handoff_request is None or handoff_request.status != "pending" else "pending handoff must be resolved before retry",
            allow_pause=run.status not in {"failed", "completed", "approved"},
            pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
            allow_reassign=handoff_request is not None,
            reassign_reason="" if handoff_request is not None else "reassign is available after a handoff exists",
            allow_escalate=policy is not None and policy.upgrade_required,
            escalate_reason="" if policy is not None and policy.upgrade_required else "escalate when policy requires an entitlement or approval upgrade",
        ),
        collaboration=build_collaboration_thread_from_audits(
            [entry.to_dict() for entry in run.audits],
            surface="flow",
            target_id=run.run_id,
        ),
    )


def render_orchestration_canvas(canvas: OrchestrationCanvas) -> str:
    lines = [
        "# Orchestration Canvas",
        "",
        f"- Task ID: {canvas.task_id}",
        f"- Run ID: {canvas.run_id}",
        f"- Collaboration Mode: {canvas.collaboration_mode}",
        f"- Departments: {', '.join(canvas.departments) if canvas.departments else 'none'}",
        f"- Required Approvals: {', '.join(canvas.required_approvals) if canvas.required_approvals else 'none'}",
        f"- Tier: {canvas.tier}",
        f"- Upgrade Required: {canvas.upgrade_required}",
        f"- Entitlement Status: {canvas.entitlement_status}",
        f"- Billing Model: {canvas.billing_model}",
        f"- Blocked Departments: {', '.join(canvas.blocked_departments) if canvas.blocked_departments else 'none'}",
        f"- Handoff Team: {canvas.handoff_team}",
        f"- Handoff Status: {canvas.handoff_status}",
        f"- Recommendation: {canvas.recommendation}",
        "",
        "## Execution Context",
        "",
        f"- Active Tools: {', '.join(canvas.active_tools) if canvas.active_tools else 'none'}",
        f"- Estimated Cost (USD): {canvas.estimated_cost_usd:.2f}",
        f"- Included Usage Units: {canvas.included_usage_units}",
        f"- Overage Usage Units: {canvas.overage_usage_units}",
        f"- Overage Cost (USD): {canvas.overage_cost_usd:.2f}",
        f"- Handoff Reason: {canvas.handoff_reason or 'none'}",
        "",
        "## Actions",
        "",
        f"- {render_console_actions(canvas.actions)}",
    ]
    lines.extend(render_collaboration_lines(canvas.collaboration))
    return "\n".join(lines) + "\n"


def build_orchestration_portfolio_from_ledger(
    entries: List[dict],
    name: str = "Cross-Department Orchestration",
    period: str = "current",
) -> OrchestrationPortfolio:
    canvases = [
        build_orchestration_canvas_from_ledger_entry(entry)
        for entry in entries
        if _latest_named_audit(entry.get("audits", []), "orchestration.plan") is not None
    ]
    takeover_queue = build_takeover_queue_from_ledger(entries, name=f"{name} Takeovers", period=period)
    return build_orchestration_portfolio(
        canvases,
        name=name,
        period=period,
        takeover_queue=takeover_queue,
    )


def build_billing_entitlements_page_from_ledger(
    entries: List[dict],
    *,
    workspace_name: str = "BigClaw Cloud",
    plan_name: str = "Standard",
    billing_period: str = "current",
) -> BillingEntitlementsPage:
    portfolio = build_orchestration_portfolio_from_ledger(entries, name=workspace_name, period=billing_period)
    return build_billing_entitlements_page(
        portfolio,
        workspace_name=workspace_name,
        plan_name=plan_name,
        billing_period=billing_period,
    )


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
                actions=build_console_actions(
                    str(entry.get("run_id", "")),
                    allow_retry=False,
                    retry_reason="retry is blocked while takeover is pending",
                    allow_pause=str(handoff_audit.get("outcome", "pending")) == "pending",
                    pause_reason="" if str(handoff_audit.get("outcome", "pending")) == "pending" else "only pending takeovers can be paused",
                    allow_reassign=True,
                    allow_escalate=str(details.get("target_team", "")) != "security",
                    escalate_reason="" if str(details.get("target_team", "")) != "security" else "security takeovers are already escalated",
                ),
            )
        )

    requests.sort(key=lambda request: (request.target_team, request.run_id))
    return TakeoverQueue(name=name, period=period, requests=requests)


def render_takeover_queue_report(
    queue: TakeoverQueue,
    total_runs: Optional[int] = None,
    view: Optional[SharedViewContext] = None,
) -> str:
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
    lines.extend(render_shared_view_context(view))

    if queue.requests:
        for request in queue.requests:
            approvals = ",".join(request.required_approvals) if request.required_approvals else "none"
            lines.append(
                f"- {request.run_id}: team={request.target_team} status={request.status} task={request.task_id} "
                f"approvals={approvals} reason={request.reason} actions={render_console_actions(request.actions)}"
            )
    else:
        lines.append("- None")

    return "\n".join(lines) + "\n"


def _latest_named_audit(audits: List[dict], action: str) -> Optional[dict]:
    for audit in reversed(audits):
        if audit.get("action") == action:
            return audit
    return None


def _latest_handoff_audit(audits: List[dict]) -> Optional[dict]:
    for action in (MANUAL_TAKEOVER_EVENT, FLOW_HANDOFF_EVENT, "orchestration.handoff"):
        audit = _latest_named_audit(audits, action)
        if audit is not None:
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


def _build_triage_suggestions(
    run: TaskRun,
    runs: List[TaskRun],
    severity: str,
    owner: str,
    feedback: List[TriageFeedbackRecord],
) -> List[TriageSuggestion]:
    action = _triage_next_action(severity, owner)
    label = _triage_suggestion_label(run, severity, owner)
    evidence = _similarity_evidence(run, runs)
    confidence = _triage_suggestion_confidence(run, evidence)
    feedback_status = _feedback_status(run.run_id, action, feedback)
    return [
        TriageSuggestion(
            label=label,
            action=action,
            owner=owner,
            confidence=confidence,
            evidence=evidence,
            feedback_status=feedback_status,
        )
    ]


def _triage_suggestion_label(run: TaskRun, severity: str, owner: str) -> str:
    if severity == "critical" and owner == "engineering":
        return "replay candidate"
    if owner == "security":
        return "approval review"
    if run.status == "failed":
        return "incident review"
    return "workflow follow-up"


def _triage_suggestion_confidence(run: TaskRun, evidence: List[TriageSimilarityEvidence]) -> float:
    base = 0.55 if run.status in {"needs-approval", "failed"} else 0.45
    if evidence:
        base = max(base, min(0.95, 0.45 + evidence[0].score / 2))
    return round(base, 2)


def _feedback_status(run_id: str, action: str, feedback: List[TriageFeedbackRecord]) -> str:
    for record in reversed(feedback):
        if record.run_id == run_id and record.action == action:
            return record.decision
    return "pending"


def _slugify(value: str) -> str:
    normalized = "".join(char.lower() if char.isalnum() else "-" for char in value.strip())
    return "-".join(part for part in normalized.split("-") if part)


def _similarity_evidence(run: TaskRun, runs: List[TaskRun], limit: int = 2) -> List[TriageSimilarityEvidence]:
    scored_matches: List[tuple[float, TaskRun]] = []
    for candidate in runs:
        if candidate.run_id == run.run_id:
            continue
        score = _run_similarity_score(run, candidate)
        if score < 0.35:
            continue
        scored_matches.append((score, candidate))

    scored_matches.sort(key=lambda item: (-item[0], item[1].run_id))
    evidence: List[TriageSimilarityEvidence] = []
    for score, candidate in scored_matches[:limit]:
        evidence.append(
            TriageSimilarityEvidence(
                related_run_id=candidate.run_id,
                related_task_id=candidate.task_id,
                score=round(score, 2),
                reason=_similarity_reason(run, candidate),
            )
        )
    return evidence


def _run_similarity_score(run: TaskRun, candidate: TaskRun) -> float:
    haystack = " ".join(
        [
            run.title,
            run.summary,
            " ".join(trace.span for trace in run.traces),
            " ".join(audit.outcome for audit in run.audits),
        ]
    ).lower()
    needle = " ".join(
        [
            candidate.title,
            candidate.summary,
            " ".join(trace.span for trace in candidate.traces),
            " ".join(audit.outcome for audit in candidate.audits),
        ]
    ).lower()
    status_bonus = 0.15 if run.status == candidate.status else 0.0
    owner_bonus = 0.1 if _triage_owner(run) == _triage_owner(candidate) else 0.0
    return min(1.0, SequenceMatcher(a=haystack, b=needle).ratio() + status_bonus + owner_bonus)


def _similarity_reason(run: TaskRun, candidate: TaskRun) -> str:
    reasons: List[str] = []
    if run.status == candidate.status:
        reasons.append(f"shared status {run.status}")
    if _triage_owner(run) == _triage_owner(candidate):
        reasons.append(f"shared owner {_triage_owner(run)}")
    run_reason = _triage_reason(run)
    candidate_reason = _triage_reason(candidate)
    if run_reason == candidate_reason:
        reasons.append("matching failure reason")
    return ", ".join(reasons) or "similar execution trail"


def render_repo_sync_audit_report(audit: RepoSyncAudit) -> str:
    lines = [
        "# Repo Sync Audit",
        "",
        "## Sync Status",
        "",
        f"- Status: {audit.sync.status}",
        f"- Failure Category: {audit.sync.failure_category or 'none'}",
        f"- Summary: {audit.sync.summary or 'none'}",
        f"- Branch: {audit.sync.branch or 'unknown'}",
        f"- Remote: {audit.sync.remote}",
        f"- Remote Ref: {audit.sync.remote_ref or 'unknown'}",
        f"- Ahead By: {audit.sync.ahead_by}",
        f"- Behind By: {audit.sync.behind_by}",
        f"- Dirty Paths: {', '.join(audit.sync.dirty_paths) if audit.sync.dirty_paths else 'none'}",
        f"- Auth Target: {audit.sync.auth_target or 'none'}",
        f"- Checked At: {audit.sync.timestamp}",
        "",
        "## Pull Request Freshness",
        "",
        f"- PR Number: {audit.pull_request.pr_number if audit.pull_request.pr_number is not None else 'unknown'}",
        f"- PR URL: {audit.pull_request.pr_url or 'none'}",
        f"- Branch State: {audit.pull_request.branch_state}",
        f"- Body State: {audit.pull_request.body_state}",
        f"- Branch Head SHA: {audit.pull_request.branch_head_sha or 'unknown'}",
        f"- PR Head SHA: {audit.pull_request.pr_head_sha or 'unknown'}",
        f"- Expected Body Digest: {audit.pull_request.expected_body_digest or 'unknown'}",
        f"- Actual Body Digest: {audit.pull_request.actual_body_digest or 'unknown'}",
        f"- Checked At: {audit.pull_request.checked_at}",
        "",
        "## Summary",
        "",
        f"- {audit.summary}",
    ]
    return "\n".join(lines) + "\n"


def render_task_run_report(run: TaskRun) -> str:
    actions = build_console_actions(
        run.run_id,
        allow_retry=run.status in {"failed", "needs-approval"},
        retry_reason="" if run.status in {"failed", "needs-approval"} else "retry is available for failed or approval-blocked runs",
        allow_pause=run.status not in {"failed", "completed", "approved"},
        pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
    )
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in run.audits],
        surface="run",
        target_id=run.run_id,
    )
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

    lines.extend(["", "## Closeout", ""])
    lines.append(f"- Complete: {run.closeout.complete}")
    lines.append(
        "- Validation Evidence: "
        + (", ".join(run.closeout.validation_evidence) if run.closeout.validation_evidence else "None")
    )
    lines.append(f"- Git Push Succeeded: {run.closeout.git_push_succeeded}")
    lines.append(f"- Git Push Output: {run.closeout.git_push_output or 'None'}")
    lines.append(f"- Git Log -1 --stat Output: {run.closeout.git_log_stat_output or 'None'}")
    if run.closeout.repo_sync_audit is not None:
        lines.append(f"- Repo Sync Status: {run.closeout.repo_sync_audit.sync.status}")
        lines.append(
            "- Repo Sync Failure Category: "
            + (run.closeout.repo_sync_audit.sync.failure_category or "none")
        )
        lines.append(f"- PR Branch State: {run.closeout.repo_sync_audit.pull_request.branch_state}")
        lines.append(f"- PR Body State: {run.closeout.repo_sync_audit.pull_request.body_state}")
    lines.extend(["", "## Actions", "", f"- {render_console_actions(actions)}"])
    lines.extend(render_collaboration_lines(collaboration))

    return "\n".join(lines) + "\n"


def render_task_run_detail_page(run: TaskRun) -> str:
    status_tone = "accent" if run.status in {"approved", "completed", "succeeded"} else "warning"
    if run.status in {"failed", "rejected"}:
        status_tone = "danger"

    actions = build_console_actions(
        run.run_id,
        allow_retry=run.status in {"failed", "needs-approval"},
        retry_reason="" if run.status in {"failed", "needs-approval"} else "retry is available for failed or approval-blocked runs",
        allow_pause=run.status not in {"failed", "completed", "approved"},
        pause_reason="" if run.status not in {"failed", "completed", "approved"} else "completed or failed runs cannot be paused",
    )
    timeline_events = sorted(
        [
            *[
                RunDetailEvent(
                    event_id=f"log-{index}",
                    lane="log",
                    title=entry.message,
                    timestamp=entry.timestamp,
                    status=entry.level,
                    summary=f"log entry at {entry.timestamp}",
                    details=[f"{key}={value}" for key, value in sorted(entry.context.items())] or ["No structured context recorded."],
                )
                for index, entry in enumerate(run.logs)
            ],
            *[
                RunDetailEvent(
                    event_id=f"trace-{index}",
                    lane="trace",
                    title=entry.span,
                    timestamp=entry.timestamp,
                    status=entry.status,
                    summary=f"trace span {entry.span}",
                    details=[f"{key}={value}" for key, value in sorted(entry.attributes.items())] or ["No trace attributes recorded."],
                )
                for index, entry in enumerate(run.traces)
            ],
            *[
                RunDetailEvent(
                    event_id=f"audit-{index}",
                    lane="audit",
                    title=entry.action,
                    timestamp=entry.timestamp,
                    status=entry.outcome,
                    summary=f"audit by {entry.actor}",
                    details=[f"actor={entry.actor}", *[f"{key}={value}" for key, value in sorted(entry.details.items())]] or ["No audit details recorded."],
                )
                for index, entry in enumerate(run.audits)
            ],
            *[
                RunDetailEvent(
                    event_id=f"artifact-{index}",
                    lane="artifact",
                    title=entry.name,
                    timestamp=entry.timestamp,
                    status=entry.kind,
                    summary=f"artifact emitted at {entry.path}",
                    details=[
                        f"path={entry.path}",
                        f"sha256={entry.sha256 or 'n/a'}",
                        *[f"{key}={value}" for key, value in sorted(entry.metadata.items())],
                    ],
                )
                for index, entry in enumerate(run.artifacts)
            ],
        ],
        key=lambda event: event.timestamp,
    )
    artifacts = [
        RunDetailResource(
            name=entry.name,
            kind=entry.kind,
            path=entry.path,
            meta=[f"sha256={entry.sha256 or 'n/a'}", *[f"{key}={value}" for key, value in sorted(entry.metadata.items())]],
            tone="report" if entry.kind == "report" else "page" if entry.kind == "page" else "default",
        )
        for entry in run.artifacts
    ]
    report_resources = [resource for resource in artifacts if resource.kind == "report"]
    collaboration = build_collaboration_thread_from_audits(
        [entry.to_dict() for entry in run.audits],
        surface="run",
        target_id=run.run_id,
    )

    repo_link_resources = [
        RunDetailResource(
            name=link.commit_hash,
            kind=link.role,
            path=f"repo:{link.repo_space_id}",
            meta=[f"actor={link.actor or 'unknown'}", *[f"{key}={value}" for key, value in sorted(link.metadata.items())]],
            tone="accent" if link.role == "accepted" else "default",
        )
        for link in run.closeout.run_commit_links
    ]

    overview_html = f"""
    <section class="surface">
      <h2>Overview</h2>
      <p>{escape(run.summary or 'No summary recorded.')}</p>
      <p class="meta">Task {escape(run.task_id)} from {escape(run.source)} started at {escape(run.started_at)} and ended at {escape(run.ended_at or 'n/a')}.</p>
    </section>
    <section class="surface">
      <h2>Closeout</h2>
      <p>Validation evidence: {escape(', '.join(run.closeout.validation_evidence) if run.closeout.validation_evidence else 'None recorded.')}</p>
      <p class="meta">git push succeeded={escape(str(run.closeout.git_push_succeeded))} | git log captured={escape(str(bool(run.closeout.git_log_stat_output.strip())))} | complete={escape(str(run.closeout.complete))}</p>
      <p class="meta">accepted_commit_hash={escape(run.closeout.accepted_commit_hash or 'none')} | commit_links={escape(str(len(run.closeout.run_commit_links)))}</p>
    </section>
    <section class="surface">
      <h2>Actions</h2>
      <p>{escape(render_console_actions(actions))}</p>
    </section>
    """

    return render_run_detail_console(
        page_title=f"Task Run Detail · {run.run_id}",
        eyebrow="Run Detail",
        hero_title=run.title,
        hero_summary=run.summary or "Operational detail page with synced logs, traces, audits, and artifacts.",
        stats=[
            RunDetailStat("Run ID", run.run_id),
            RunDetailStat("Task ID", run.task_id),
            RunDetailStat("Medium", run.medium, tone="accent" if run.medium == "browser" else "default"),
            RunDetailStat("Status", run.status, tone=status_tone),
            RunDetailStat("Artifacts", str(len(run.artifacts))),
            RunDetailStat("Reports", str(len(report_resources)), tone="accent" if report_resources else "default"),
            RunDetailStat("Closeout", "complete" if run.closeout.complete else "pending", tone="accent" if run.closeout.complete else "warning"),
            RunDetailStat("Repo Links", str(len(run.closeout.run_commit_links)), tone="accent" if run.closeout.run_commit_links else "default"),
        ],
        tabs=[
            RunDetailTab("overview", "Overview", overview_html),
            RunDetailTab(
                "timeline",
                "Timeline / Log Sync",
                render_timeline_panel(
                    "Timeline / Log Sync",
                    "Unified execution timeline for logs, traces, audits, and emitted artifacts. Selecting an item updates the inspector in the split view.",
                    timeline_events,
                ),
            ),
            RunDetailTab(
                "artifacts",
                "Artifacts",
                render_resource_grid(
                    "Artifacts",
                    "Execution artifacts and generated outputs attached to this run.",
                    artifacts,
                ),
            ),
            RunDetailTab(
                "reports",
                "Reports",
                render_resource_grid(
                    "Reports",
                    "Report artifacts emitted for this run, including markdown summaries and linked detail pages when present.",
                    report_resources,
                ),
            ),
            RunDetailTab(
                "repo-evidence",
                "Repo Evidence",
                render_resource_grid(
                    "Repo Evidence",
                    "Commit links, roles, and accepted lineage hints bound at closeout.",
                    repo_link_resources,
                ),
            ),
            RunDetailTab(
                "collaboration",
                "Collaboration",
                render_collaboration_panel_html(
                    "Collaboration",
                    "Comments, mentions, and decision notes recorded against this run.",
                    collaboration,
                ),
            ),
        ],
        timeline_events=timeline_events,
    )


def render_weekly_repo_evidence_section(
    *,
    experiment_volume: int,
    converged_tasks: int,
    accepted_commits: int,
    hottest_threads: List[str],
) -> str:
    lines = [
        "## Repo Evidence Summary",
        f"- Experiment Volume: {experiment_volume}",
        f"- Converged Tasks: {converged_tasks}",
        f"- Accepted Commits: {accepted_commits}",
        f"- Hottest Threads: {', '.join(hottest_threads) if hottest_threads else 'none'}",
    ]
    return "\n".join(lines)


def render_repo_narrative_exports(
    *,
    experiment_volume: int,
    converged_tasks: int,
    accepted_commits: int,
    hottest_threads: List[str],
) -> dict:
    markdown_text = render_weekly_repo_evidence_section(
        experiment_volume=experiment_volume,
        converged_tasks=converged_tasks,
        accepted_commits=accepted_commits,
        hottest_threads=hottest_threads,
    )
    plain_text = markdown_text.replace("## ", "")
    html = (
        "<section><h2>Repo Evidence Summary</h2>"
        f"<p>Experiment Volume: {experiment_volume}</p>"
        f"<p>Converged Tasks: {converged_tasks}</p>"
        f"<p>Accepted Commits: {accepted_commits}</p>"
        f"<p>Hottest Threads: {escape(', '.join(hottest_threads) if hottest_threads else 'none')}</p>"
        "</section>"
    )
    return {"markdown": markdown_text, "text": plain_text, "html": html}

# Folded operations surface from legacy operations.py

STATUS_COMPLETE = {"approved", "accepted", "completed", "succeeded"}
STATUS_ACTIONABLE = {"needs-approval", "failed", "rejected"}


@dataclass
class EvaluationCriterion:
    name: str
    weight: int
    passed: bool
    detail: str


@dataclass
class BenchmarkCase:
    case_id: str
    task: Task
    expected_medium: Optional[str] = None
    expected_approved: Optional[bool] = None
    expected_status: Optional[str] = None
    require_report: bool = False


@dataclass
class ReplayRecord:
    task: Task
    run_id: str
    medium: str
    approved: bool
    status: str

    @classmethod
    def from_execution(cls, task: Task, run_id: str, record: ExecutionRecord) -> "ReplayRecord":
        return cls(
            task=task,
            run_id=run_id,
            medium=record.decision.medium,
            approved=record.decision.approved,
            status=record.run.status,
        )


@dataclass
class ReplayOutcome:
    matched: bool
    replay_record: ReplayRecord
    mismatches: List[str] = field(default_factory=list)
    report_path: Optional[str] = None


@dataclass
class BenchmarkResult:
    case_id: str
    score: int
    passed: bool
    criteria: List[EvaluationCriterion]
    record: ExecutionRecord
    replay: ReplayOutcome
    detail_page_path: Optional[str] = None


@dataclass
class BenchmarkComparison:
    case_id: str
    baseline_score: int
    current_score: int
    delta: int
    changed: bool


@dataclass
class BenchmarkSuiteResult:
    results: List[BenchmarkResult]
    version: str = "current"

    @property
    def score(self) -> int:
        if not self.results:
            return 0
        return round(sum(result.score for result in self.results) / len(self.results))

    @property
    def passed(self) -> bool:
        return all(result.passed for result in self.results)

    def compare(self, baseline: "BenchmarkSuiteResult") -> List[BenchmarkComparison]:
        baseline_by_case = {result.case_id: result for result in baseline.results}
        comparisons = []
        for result in self.results:
            baseline_result = baseline_by_case.get(result.case_id)
            baseline_score = baseline_result.score if baseline_result else 0
            delta = result.score - baseline_score
            comparisons.append(
                BenchmarkComparison(
                    case_id=result.case_id,
                    baseline_score=baseline_score,
                    current_score=result.score,
                    delta=delta,
                    changed=delta != 0,
                )
            )
        return comparisons


class BenchmarkRunner:
    def __init__(self, scheduler: Optional[Scheduler] = None, storage_dir: Optional[str] = None):
        self.scheduler = scheduler or Scheduler()
        self.storage_dir = Path(storage_dir) if storage_dir else None

    def run_case(self, case: BenchmarkCase) -> BenchmarkResult:
        ledger = ObservabilityLedger(str(self._case_path(case.case_id, "ledger.json")))
        report_path = None
        if case.require_report:
            report_path = str(self._case_path(case.case_id, "task-run.md"))

        run_id = f"benchmark-{case.case_id}"
        record = self.scheduler.execute(
            case.task,
            run_id=run_id,
            ledger=ledger,
            report_path=report_path,
            actor="benchmark-runner",
        )
        criteria = self._evaluate(case, record)
        replay = self.replay(ReplayRecord.from_execution(case.task, run_id, record))
        total_weight = sum(item.weight for item in criteria)
        earned_weight = sum(item.weight for item in criteria if item.passed)
        score = round((earned_weight / total_weight) * 100) if total_weight else 0
        passed = all(item.passed for item in criteria) and replay.matched
        detail_page_path = None
        if self.storage_dir is not None:
            detail_page_path = str(self._case_path(case.case_id, "run-detail.html"))
            write_report(detail_page_path, render_run_replay_index_page(case.case_id, record, replay, criteria))
        return BenchmarkResult(
            case_id=case.case_id,
            score=score,
            passed=passed,
            criteria=criteria,
            record=record,
            replay=replay,
            detail_page_path=detail_page_path,
        )

    def run_suite(self, cases: List[BenchmarkCase], version: str = "current") -> BenchmarkSuiteResult:
        return BenchmarkSuiteResult(
            results=[self.run_case(case) for case in cases],
            version=version,
        )

    def replay(self, replay_record: ReplayRecord) -> ReplayOutcome:
        ledger = ObservabilityLedger(str(self._case_path(replay_record.run_id, "replay-ledger.json")))
        replayed = self.scheduler.execute(
            replay_record.task,
            run_id=f"{replay_record.run_id}-replay",
            ledger=ledger,
            actor="benchmark-replay",
        )
        observed = ReplayRecord.from_execution(
            replay_record.task,
            replay_record.run_id,
            replayed,
        )
        mismatches = []
        if observed.medium != replay_record.medium:
            mismatches.append(f"medium expected {replay_record.medium} got {observed.medium}")
        if observed.approved != replay_record.approved:
            mismatches.append(f"approved expected {replay_record.approved} got {observed.approved}")
        if observed.status != replay_record.status:
            mismatches.append(f"status expected {replay_record.status} got {observed.status}")
        report_path = None
        if self.storage_dir is not None:
            report_path = str(self._case_path(replay_record.run_id, "replay.html"))
            write_report(report_path, render_replay_detail_page(replay_record, observed, mismatches))
        return ReplayOutcome(
            matched=not mismatches,
            replay_record=observed,
            mismatches=mismatches,
            report_path=report_path,
        )

    def _evaluate(self, case: BenchmarkCase, record: ExecutionRecord) -> List[EvaluationCriterion]:
        return [
            self._criterion("decision-medium", 40, case.expected_medium, record.decision.medium),
            self._criterion("approval-gate", 30, case.expected_approved, record.decision.approved),
            self._criterion("final-status", 20, case.expected_status, record.run.status),
            EvaluationCriterion(
                name="report-artifact",
                weight=10,
                passed=(not case.require_report) or bool(record.report_path),
                detail=("report emitted" if (not case.require_report) or bool(record.report_path) else "report missing"),
            ),
        ]

    def _criterion(self, name: str, weight: int, expected: Optional[object], actual: object) -> EvaluationCriterion:
        if expected is None:
            return EvaluationCriterion(name=name, weight=weight, passed=True, detail="not asserted")
        passed = expected == actual
        detail = f"expected {expected} got {actual}"
        return EvaluationCriterion(name=name, weight=weight, passed=passed, detail=detail)

    def _case_path(self, case_id: str, file_name: str) -> Path:
        if self.storage_dir is None:
            return Path(file_name)
        return self.storage_dir / case_id / file_name


def render_benchmark_suite_report(
    suite: BenchmarkSuiteResult,
    baseline: Optional[BenchmarkSuiteResult] = None,
) -> str:
    lines = [
        "# Benchmark Suite Report",
        "",
        f"- Version: {suite.version}",
        f"- Cases: {len(suite.results)}",
        f"- Passed: {suite.passed}",
        f"- Score: {suite.score}",
        "",
        "## Cases",
        "",
    ]

    if suite.results:
        lines.extend(
            f"- {result.case_id}: score={result.score} passed={result.passed} replay={result.replay.matched}"
            for result in suite.results
        )
    else:
        lines.append("- None")

    lines.extend(["", "## Comparison", ""])
    if baseline is None:
        lines.append("- No baseline provided")
    else:
        lines.append(f"- Baseline Version: {baseline.version}")
        lines.append(f"- Score Delta: {suite.score - baseline.score}")
        comparisons = suite.compare(baseline)
        if comparisons:
            lines.extend(
                f"- {comparison.case_id}: baseline={comparison.baseline_score} current={comparison.current_score} delta={comparison.delta}"
                for comparison in comparisons
            )
        else:
            lines.append("- No comparable cases")

    return "\n".join(lines) + "\n"


def render_replay_detail_page(expected: ReplayRecord, observed: ReplayRecord, mismatches: List[str]) -> str:
    tone = "accent" if not mismatches else "danger"
    timeline_events = [
        RunDetailEvent(
            event_id="compare-medium",
            lane="comparison",
            title="Medium",
            timestamp="compare-1",
            status="matched" if expected.medium == observed.medium else "mismatch",
            summary=f"expected {expected.medium} | observed {observed.medium}",
            details=[f"expected={expected.medium}", f"observed={observed.medium}"],
        ),
        RunDetailEvent(
            event_id="compare-approved",
            lane="comparison",
            title="Approval",
            timestamp="compare-2",
            status="matched" if expected.approved == observed.approved else "mismatch",
            summary=f"expected {expected.approved} | observed {observed.approved}",
            details=[f"expected={expected.approved}", f"observed={observed.approved}"],
        ),
        RunDetailEvent(
            event_id="compare-status",
            lane="comparison",
            title="Status",
            timestamp="compare-3",
            status="matched" if expected.status == observed.status else "mismatch",
            summary=f"expected {expected.status} | observed {observed.status}",
            details=[f"expected={expected.status}", f"observed={observed.status}"],
        ),
        *[
            RunDetailEvent(
                event_id=f"mismatch-{index}",
                lane="replay",
                title=f"Mismatch {index + 1}",
                timestamp=f"compare-{index + 4}",
                status="mismatch",
                summary=item,
                details=[item],
            )
            for index, item in enumerate(mismatches)
        ],
    ]
    comparison_html = f"""
    <section class="surface">
      <h2>Split Comparison</h2>
      <p>Side-by-side replay comparison for task <strong>{escape(expected.task.task_id)}</strong> against baseline run <code>{escape(expected.run_id)}</code>.</p>
      <div class="resource-grid">
        <article class="resource-card">
          <span class="kicker">Baseline</span>
          <h3>Expected</h3>
          <p><code>medium={escape(expected.medium)}</code></p>
          <span class="resource-meta">approved={escape(str(expected.approved))} | status={escape(expected.status)}</span>
        </article>
        <article class="resource-card">
          <span class="kicker">Replay</span>
          <h3>Observed</h3>
          <p><code>medium={escape(observed.medium)}</code></p>
          <span class="resource-meta">approved={escape(str(observed.approved))} | status={escape(observed.status)}</span>
        </article>
      </div>
    </section>
    """
    mismatch_html = f"""
    <section class="surface">
      <h2>Replay Mismatches</h2>
      <p>Detailed mismatch list for the replay execution.</p>
      <ul>{''.join(f'<li>{escape(item)}</li>' for item in mismatches) or '<li>None</li>'}</ul>
    </section>
    """
    return render_run_detail_console(
        page_title=f"Replay Detail · {expected.run_id}",
        eyebrow="Replay Detail",
        hero_title=f"Replay Detail · {expected.task.task_id}",
        hero_summary="High-fidelity replay inspection with synced comparison timeline and split-view baseline versus observed execution state.",
        stats=[
            RunDetailStat("Run ID", expected.run_id),
            RunDetailStat("Task ID", expected.task.task_id),
            RunDetailStat("Expected Medium", expected.medium),
            RunDetailStat("Observed Medium", observed.medium, tone=tone),
            RunDetailStat("Replay", "matched" if not mismatches else "mismatch", tone=tone),
            RunDetailStat("Mismatches", str(len(mismatches)), tone=tone),
        ],
        tabs=[
            RunDetailTab("overview", "Overview", comparison_html),
            RunDetailTab(
                "timeline",
                "Timeline / Log Sync",
                render_timeline_panel(
                    "Timeline / Log Sync",
                    "Field-by-field replay comparison with a synced inspector for each expectation and mismatch.",
                    timeline_events,
                ),
            ),
            RunDetailTab("comparison", "Split View", comparison_html),
            RunDetailTab("replay", "Replay", mismatch_html),
            RunDetailTab(
                "reports",
                "Reports",
                render_resource_grid(
                    "Reports",
                    "Replay detail pages do not emit standalone report files beyond the generated HTML page unless the caller persists additional artifacts.",
                    [],
                ),
            ),
        ],
        timeline_events=timeline_events,
    )


def render_run_replay_index_page(
    case_id: str,
    record: ExecutionRecord,
    replay: ReplayOutcome,
    criteria: List[EvaluationCriterion],
) -> str:
    status_tone = "accent" if record.run.status == "approved" else "warning"
    if replay.mismatches:
        status_tone = "danger"

    report_path = record.report_path or "n/a"
    detail_path = str(Path(record.report_path).with_suffix(".html")) if record.report_path else "n/a"
    replay_path = replay.report_path or "n/a"

    criteria_events = [
        RunDetailEvent(
            event_id=f"criterion-{index}",
            lane="acceptance",
            title=item.name,
            timestamp=f"step-{index + 1}",
            status="passed" if item.passed else "failed",
            summary=item.detail,
            details=[f"weight={item.weight}", f"passed={item.passed}"],
        )
        for index, item in enumerate(criteria)
    ]
    mismatch_events = [
        RunDetailEvent(
            event_id=f"mismatch-{index}",
            lane="replay",
            title=f"Replay mismatch {index + 1}",
            timestamp=f"replay-{index + 1}",
            status="mismatch",
            summary=item,
            details=[item],
        )
        for index, item in enumerate(replay.mismatches)
    ]
    run_events = sorted(
        [
            *[
                RunDetailEvent(
                    event_id=f"log-{index}",
                    lane="log",
                    title=entry.message,
                    timestamp=entry.timestamp,
                    status=entry.level,
                    summary=f"log entry at {entry.timestamp}",
                    details=[f"{key}={value}" for key, value in sorted(entry.context.items())]
                    or ["No structured context recorded."],
                )
                for index, entry in enumerate(record.run.logs)
            ],
            *[
                RunDetailEvent(
                    event_id=f"trace-{index}",
                    lane="trace",
                    title=entry.span,
                    timestamp=entry.timestamp,
                    status=entry.status,
                    summary=f"trace span {entry.span}",
                    details=[f"{key}={value}" for key, value in sorted(entry.attributes.items())]
                    or ["No trace attributes recorded."],
                )
                for index, entry in enumerate(record.run.traces)
            ],
            *[
                RunDetailEvent(
                    event_id=f"audit-{index}",
                    lane="audit",
                    title=entry.action,
                    timestamp=entry.timestamp,
                    status=entry.outcome,
                    summary=f"audit by {entry.actor}",
                    details=[f"actor={entry.actor}", *[f"{key}={value}" for key, value in sorted(entry.details.items())]]
                    or ["No audit details recorded."],
                )
                for index, entry in enumerate(record.run.audits)
            ],
            *criteria_events,
            *mismatch_events,
        ],
        key=lambda event: event.timestamp,
    )

    execution_resources = [
        RunDetailResource(
            name="Markdown report",
            kind="report",
            path=report_path,
            meta=["execution report"],
            tone="report",
        ),
        RunDetailResource(
            name="Run detail page",
            kind="page",
            path=detail_path,
            meta=["task run detail"],
            tone="page",
        ),
        RunDetailResource(
            name="Replay page",
            kind="page",
            path=replay_path,
            meta=[f"matched={replay.matched}"],
            tone="page",
        ),
    ]

    overview_html = f"""
    <section class="surface">
      <h2>Overview</h2>
      <p>Benchmark case <strong>{escape(case_id)}</strong> executed task <strong>{escape(record.run.task_id)}</strong> with scheduler medium <strong>{escape(record.decision.medium)}</strong>.</p>
      <p class="meta">Replay matched={escape(str(replay.matched))} | mismatches={escape(str(len(replay.mismatches)))}</p>
    </section>
    """
    acceptance_html = f"""
    <section class="surface">
      <h2>Acceptance Criteria</h2>
      <p>Scored checks used to grade the run detail and replay execution path.</p>
      <ul>
        {''.join(f'<li><strong>{escape(item.name)}</strong>: {escape(item.detail)} | weight={item.weight} | passed={item.passed}</li>' for item in criteria) or '<li>None</li>'}
      </ul>
    </section>
    """
    replay_html = f"""
    <section class="surface">
      <h2>Replay</h2>
      <p>Replay status <strong>{escape('matched' if replay.matched else 'mismatch')}</strong> for baseline run <code>{escape(replay.replay_record.run_id)}</code>.</p>
      <ul>
        {''.join(f'<li>{escape(item)}</li>' for item in replay.mismatches) or '<li>None</li>'}
      </ul>
    </section>
    """

    return render_run_detail_console(
        page_title=f"Run Detail Index · {case_id}",
        eyebrow="Replay Console",
        hero_title=f"Run Detail Index · {case_id}",
        hero_summary="Benchmark execution, replay evidence, and acceptance criteria in a single operator-facing run console.",
        stats=[
            RunDetailStat("Task ID", record.run.task_id),
            RunDetailStat("Status", record.run.status, tone=status_tone),
            RunDetailStat("Medium", record.decision.medium, tone="accent" if record.decision.medium == "browser" else "default"),
            RunDetailStat("Replay", "matched" if replay.matched else "mismatch", tone="accent" if replay.matched else "danger"),
            RunDetailStat("Criteria", str(len(criteria))),
            RunDetailStat("Mismatches", str(len(replay.mismatches)), tone="danger" if replay.mismatches else "default"),
        ],
        tabs=[
            RunDetailTab("overview", "Overview", overview_html),
            RunDetailTab(
                "timeline",
                "Timeline / Log Sync",
                render_timeline_panel(
                    "Timeline / Log Sync",
                    "Run logs, trace spans, audits, acceptance checks, and replay mismatches are merged into one synced timeline inspector.",
                    run_events,
                ),
            ),
            RunDetailTab("acceptance", "Acceptance", acceptance_html),
            RunDetailTab(
                "artifacts",
                "Artifacts",
                render_resource_grid(
                    "Artifacts",
                    "Generated reports and pages emitted for benchmark review and replay inspection.",
                    execution_resources,
                ),
            ),
            RunDetailTab(
                "reports",
                "Reports",
                render_resource_grid(
                    "Reports",
                    "Report-first view for markdown output and linked run/replay pages.",
                    [resource for resource in execution_resources if resource.kind == "report" or resource.name.endswith("page")],
                ),
            ),
            RunDetailTab("replay", "Replay", replay_html),
        ],
        timeline_events=run_events,
    )


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
