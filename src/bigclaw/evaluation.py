from dataclasses import dataclass, field
from html import escape
import json
from pathlib import Path
from typing import List, Optional

from .models import Task
from .observability import ObservabilityLedger
from .scheduler import ExecutionRecord, Scheduler


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


def write_report(path: str, content: str) -> None:
    p = Path(path)
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(content)


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
            mismatches.append(
                f"approved expected {replay_record.approved} got {observed.approved}"
            )
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
            self._criterion(
                name="decision-medium",
                weight=40,
                expected=case.expected_medium,
                actual=record.decision.medium,
            ),
            self._criterion(
                name="approval-gate",
                weight=30,
                expected=case.expected_approved,
                actual=record.decision.approved,
            ),
            self._criterion(
                name="final-status",
                weight=20,
                expected=case.expected_status,
                actual=record.run.status,
            ),
            EvaluationCriterion(
                name="report-artifact",
                weight=10,
                passed=(not case.require_report) or bool(record.report_path),
                detail=(
                    "report emitted"
                    if (not case.require_report) or bool(record.report_path)
                    else "report missing"
                ),
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
                    details=[f"{key}={value}" for key, value in sorted(entry.context.items())] or ["No structured context recorded."],
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
                    details=[f"{key}={value}" for key, value in sorted(entry.attributes.items())] or ["No trace attributes recorded."],
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
                    details=[f"actor={entry.actor}", *[f"{key}={value}" for key, value in sorted(entry.details.items())]] or ["No audit details recorded."],
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
