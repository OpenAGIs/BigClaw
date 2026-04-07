"""Compatibility coverage for the frozen Python evaluation surface.

Removal condition: delete this test file when `src/bigclaw/evaluation.py` is
retired or replaced by Go-native coverage.
"""

from pathlib import Path

from bigclaw.evaluation import (
    BenchmarkCase,
    BenchmarkRunner,
    BenchmarkSuiteResult,
    ReplayOutcome,
    ReplayRecord,
    render_benchmark_suite_report,
    render_run_replay_index_page,
    render_replay_detail_page,
)
from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.scheduler import Scheduler


def test_benchmark_runner_scores_and_replays_case(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    case = BenchmarkCase(
        case_id="browser-low-risk",
        task=Task(
            task_id="BIG-601",
            source="linear",
            title="Run browser benchmark",
            description="validate routing",
            risk_level=RiskLevel.LOW,
            required_tools=["browser"],
        ),
        expected_medium="browser",
        expected_approved=True,
        expected_status="approved",
        require_report=True,
    )

    result = runner.run_case(case)

    assert result.score == 100
    assert result.passed is True
    assert result.replay.matched is True
    assert (tmp_path / "browser-low-risk" / "task-run.md").exists()
    assert (tmp_path / "benchmark-browser-low-risk" / "replay.html").exists()
    assert (tmp_path / "browser-low-risk" / "run-detail.html").exists()
    assert result.detail_page_path == str(tmp_path / "browser-low-risk" / "run-detail.html")



def test_benchmark_runner_detects_failed_expectation(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    case = BenchmarkCase(
        case_id="high-risk-gate",
        task=Task(
            task_id="BIG-601-risk",
            source="jira",
            title="Prod change benchmark",
            description="must stop for approval",
            risk_level=RiskLevel.HIGH,
        ),
        expected_medium="docker",
        expected_approved=False,
        expected_status="needs-approval",
    )

    result = runner.run_case(case)

    assert result.passed is False
    assert result.score == 60
    assert any(item.name == "decision-medium" and item.passed is False for item in result.criteria)



def test_replay_outcome_reports_mismatch(tmp_path: Path):
    runner = BenchmarkRunner(scheduler=Scheduler(), storage_dir=str(tmp_path))
    replay_record = ReplayRecord(
        task=Task(
            task_id="BIG-601-replay",
            source="github",
            title="Replay browser route",
            description="compare deterministic scheduler behavior",
            required_tools=["browser"],
        ),
        run_id="run-1",
        medium="docker",
        approved=True,
        status="approved",
    )

    outcome = runner.replay(replay_record)

    assert outcome.matched is False
    assert outcome.mismatches == ["medium expected docker got browser"]
    assert outcome.report_path is not None
    assert Path(outcome.report_path).exists()



def test_suite_comparison_and_report(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    improved_suite = runner.run_suite(
        [
            BenchmarkCase(
                case_id="browser-low-risk",
                task=Task(
                    task_id="BIG-601-v2",
                    source="linear",
                    title="Run browser benchmark",
                    description="validate routing",
                    required_tools=["browser"],
                ),
                expected_medium="browser",
                expected_approved=True,
                expected_status="approved",
            )
        ],
        version="v0.2",
    )
    baseline_suite = BenchmarkSuiteResult(results=[], version="v0.1")

    comparison = improved_suite.compare(baseline_suite)
    report = render_benchmark_suite_report(improved_suite, baseline_suite)

    assert comparison[0].delta == 100
    assert improved_suite.score == 100
    assert "Version: v0.2" in report
    assert "Baseline Version: v0.1" in report
    assert "Score Delta: 100" in report


def test_render_replay_detail_page_lists_mismatches():
    task = Task(task_id="BIG-804", source="linear", title="Replay detail", description="")
    expected = ReplayRecord(task=task, run_id="run-1", medium="docker", approved=True, status="approved")
    observed = ReplayRecord(task=task, run_id="run-1", medium="browser", approved=False, status="needs-approval")

    page = render_replay_detail_page(
        expected,
        observed,
        ["medium expected docker got browser", "approved expected True got False"],
    )

    assert "Replay Detail" in page
    assert "Timeline / Log Sync" in page
    assert "Split View" in page
    assert "Reports" in page
    assert "medium expected docker got browser" in page
    assert "needs-approval" in page


def test_render_run_replay_index_page_links_outputs(tmp_path: Path):
    runner = BenchmarkRunner(storage_dir=str(tmp_path))
    case = BenchmarkCase(
        case_id="big-804-index",
        task=Task(
            task_id="BIG-804",
            source="linear",
            title="Run detail index",
            description="single landing page",
            required_tools=["browser"],
        ),
        expected_medium="browser",
        expected_approved=True,
        expected_status="approved",
        require_report=True,
    )

    result = runner.run_case(case)
    page = Path(result.detail_page_path).read_text()

    assert "Run Detail Index" in page
    assert "Timeline / Log Sync" in page
    assert "Acceptance" in page
    assert "Reports" in page
    assert "task-run.md" in page
    assert "replay.html" in page
    assert "decision-medium" in page


def test_render_run_replay_index_page_without_report_path(tmp_path: Path):
    task = Task(task_id="BIG-804", source="linear", title="Run detail index", description="")
    replay = ReplayOutcome(
        matched=True,
        replay_record=ReplayRecord(task=task, run_id="run-1", medium="docker", approved=True, status="approved"),
        report_path=None,
    )
    record = Scheduler().execute(
        task,
        run_id="run-1",
        ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
    )

    page = render_run_replay_index_page("big-804-index", record, replay, [])

    assert "n/a" in page
    assert "Replay" in page
