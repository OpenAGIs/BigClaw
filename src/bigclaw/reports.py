from dataclasses import dataclass, field
from datetime import datetime, timezone
from difflib import SequenceMatcher
from html import escape
from pathlib import Path
from typing import List, Optional

from . import (
    FLOW_HANDOFF_EVENT,
    MANUAL_TAKEOVER_EVENT,
    CollaborationThread,
    RepoSyncAudit,
    TaskRun,
    build_collaboration_thread_from_audits,
    render_collaboration_lines,
    render_collaboration_panel_html,
)
from .orchestration import HandoffRequest, OrchestrationPlan, OrchestrationPolicyDecision


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
    .stat-card span, .meta, .resource-meta, .timeline-meta, .detail-list {{
      display: block;
      color: var(--muted);
      font: 500 0.78rem/1.5 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
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
    .tab-button.active {{
      background: var(--accent);
      border-color: var(--accent);
      color: #f8fffd;
    }}
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
      min-height: 100%;
    }}
    .detail-pane h3 {{
      margin-top: 0.35rem;
    }}
    .detail-list {{
      margin: 0.85rem 0 0;
      padding-left: 1rem;
    }}
    .detail-list li {{
      margin-bottom: 0.45rem;
      color: var(--ink);
    }}
    .kicker {{
      display: inline-flex;
      align-items: center;
      gap: 0.35rem;
      padding: 0.2rem 0.5rem;
      border-radius: 999px;
      background: rgba(19, 33, 47, 0.08);
      color: var(--muted);
      font: 600 0.72rem/1.2 "SFMono-Regular", Consolas, "Liberation Mono", monospace;
    }}
    .empty {{
      border: 1px dashed var(--line);
      border-radius: 16px;
      padding: 1rem;
      color: var(--muted);
      background: rgba(255, 255, 255, 0.6);
    }}
    @media (max-width: 880px) {{
      main {{
        width: min(100% - 1rem, 100%);
      }}
      .timeline-shell {{
        grid-template-columns: 1fr;
      }}
    }}
  </style>
</head>
<body>
  <main>
    <div class="shell">
      <section class="hero">
        <span class="eyebrow">{escape(eyebrow)}</span>
        <h1>{escape(hero_title)}</h1>
        <p>{escape(hero_summary)}</p>
        <div class="stats">{stat_cards}</div>
      </section>
      <div class="tabs">{tab_buttons}</div>
      <div class="panel-stack">{tab_panels}</div>
    </div>
  </main>
  <script id="timeline-data" type="application/json">{timeline_json}</script>
  <script>
    const tabButtons = Array.from(document.querySelectorAll(".tab-button"));
    const tabPanels = Array.from(document.querySelectorAll(".tab-panel"));
    const activateTab = (tabId) => {{
      tabButtons.forEach((button) => button.classList.toggle("active", button.dataset.tab === tabId));
      tabPanels.forEach((panel) => panel.classList.toggle("active", panel.dataset.panel === tabId));
    }};
    if (tabButtons.length > 0) {{
      activateTab(tabButtons[0].dataset.tab);
      tabButtons.forEach((button) => button.addEventListener("click", () => activateTab(button.dataset.tab)));
    }}

    const detailData = JSON.parse(document.getElementById("timeline-data").textContent || "[]");
    const detailIndex = Object.fromEntries(detailData.map((entry) => [entry.id, entry]));
    const timelineButtons = Array.from(document.querySelectorAll(".timeline-item"));
    const renderEvent = (eventId) => {{
      const event = detailIndex[eventId];
      if (!event) return;
      timelineButtons.forEach((button) => button.classList.toggle("active", button.dataset.eventId === eventId));
      document.querySelector('[data-detail="title"]').textContent = event.title;
      document.querySelector('[data-detail="meta"]').textContent = `${{event.lane}} / ${{event.status}} / ${{event.timestamp}}`;
      document.querySelector('[data-detail="summary"]').textContent = event.summary;
      const detailList = document.querySelector('[data-detail="list"]');
      detailList.innerHTML = "";
      const details = Array.isArray(event.details) && event.details.length > 0 ? event.details : ["No additional details."];
      details.forEach((detail) => {{
        const item = document.createElement("li");
        item.textContent = detail;
        detailList.appendChild(item);
      }});
    }};
    if (timelineButtons.length > 0) {{
      renderEvent(timelineButtons[0].dataset.eventId);
      timelineButtons.forEach((button) => button.addEventListener("click", () => renderEvent(button.dataset.eventId)));
    }}
  </script>
</body>
</html>
"""


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
