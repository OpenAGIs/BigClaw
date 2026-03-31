from dataclasses import dataclass, field
from html import escape
from pathlib import Path
from typing import List, Optional

from .models import Task
from .observability import ObservabilityLedger
from .support_surfaces import (
    RunDetailEvent,
    RunDetailResource,
    RunDetailStat,
    RunDetailTab,
    render_resource_grid,
    render_run_detail_console,
    render_timeline_panel,
)
from .scheduler import ExecutionRecord, Scheduler
from .reports import write_report


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
