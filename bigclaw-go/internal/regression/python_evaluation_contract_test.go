package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonEvaluationContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "evaluation_contract.py")
	script := `import json
import tempfile
import sys

from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.evaluation import (
    BenchmarkCase,
    BenchmarkRunner,
    BenchmarkSuiteResult,
    ReplayOutcome,
    ReplayRecord,
    render_benchmark_suite_report,
    render_replay_detail_page,
    render_run_replay_index_page,
)
from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.scheduler import Scheduler

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    runner = BenchmarkRunner(storage_dir=str(td))
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
    successful_case = {
        "score": result.score,
        "passed": result.passed,
        "replay_matched": result.replay.matched,
        "report_exists": (td / "browser-low-risk" / "task-run.md").exists(),
        "replay_exists": (td / "benchmark-browser-low-risk" / "replay.html").exists(),
        "detail_exists": (td / "browser-low-risk" / "run-detail.html").exists(),
        "detail_page_path": result.detail_page_path,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    runner = BenchmarkRunner(storage_dir=str(td))
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
    failed_case = {
        "passed": result.passed,
        "score": result.score,
        "failed_criteria": [item.name for item in result.criteria if not item.passed],
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    runner = BenchmarkRunner(scheduler=Scheduler(), storage_dir=str(td))
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
    replay_outcome = {
        "matched": outcome.matched,
        "mismatches": outcome.mismatches,
        "report_exists": bool(outcome.report_path) and Path(outcome.report_path).exists(),
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    runner = BenchmarkRunner(storage_dir=str(td))
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
    suite_comparison = {
        "delta": comparison[0].delta,
        "score": improved_suite.score,
        "has_version": "Version: v0.2" in report,
        "has_baseline": "Baseline Version: v0.1" in report,
        "has_delta": "Score Delta: 100" in report,
    }

task = Task(task_id="BIG-804", source="linear", title="Replay detail", description="")
expected = ReplayRecord(task=task, run_id="run-1", medium="docker", approved=True, status="approved")
observed = ReplayRecord(task=task, run_id="run-1", medium="browser", approved=False, status="needs-approval")
detail_page = render_replay_detail_page(
    expected,
    observed,
    ["medium expected docker got browser", "approved expected True got False"],
)
detail_report = {
    "title": "Replay Detail" in detail_page,
    "timeline": "Timeline / Log Sync" in detail_page,
    "split_view": "Split View" in detail_page,
    "reports": "Reports" in detail_page,
    "mismatch": "medium expected docker got browser" in detail_page,
    "status": "needs-approval" in detail_page,
}

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    runner = BenchmarkRunner(storage_dir=str(td))
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
    run_replay_index = {
        "title": "Run Detail Index" in page,
        "timeline": "Timeline / Log Sync" in page,
        "acceptance": "Acceptance" in page,
        "reports": "Reports" in page,
        "task_run": "task-run.md" in page,
        "replay_html": "replay.html" in page,
        "decision_medium": "decision-medium" in page,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    task = Task(task_id="BIG-804", source="linear", title="Run detail index", description="")
    replay = ReplayOutcome(
        matched=True,
        replay_record=ReplayRecord(task=task, run_id="run-1", medium="docker", approved=True, status="approved"),
        report_path=None,
    )
    record = Scheduler().execute(
        task,
        run_id="run-1",
        ledger=ObservabilityLedger(str(td / "ledger.json")),
    )
    page = render_run_replay_index_page("big-804-index", record, replay, [])
    missing_report_index = {
        "has_na": "n/a" in page,
        "has_replay": "Replay" in page,
    }

print(json.dumps({
    "successful_case": successful_case,
    "failed_case": failed_case,
    "replay_outcome": replay_outcome,
    "suite_comparison": suite_comparison,
    "detail_report": detail_report,
    "run_replay_index": run_replay_index,
    "missing_report_index": missing_report_index,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write evaluation contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run evaluation contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		SuccessfulCase struct {
			Score          int    `json:"score"`
			Passed         bool   `json:"passed"`
			ReplayMatched  bool   `json:"replay_matched"`
			ReportExists   bool   `json:"report_exists"`
			ReplayExists   bool   `json:"replay_exists"`
			DetailExists   bool   `json:"detail_exists"`
			DetailPagePath string `json:"detail_page_path"`
		} `json:"successful_case"`
		FailedCase struct {
			Passed         bool     `json:"passed"`
			Score          int      `json:"score"`
			FailedCriteria []string `json:"failed_criteria"`
		} `json:"failed_case"`
		ReplayOutcome struct {
			Matched      bool     `json:"matched"`
			Mismatches   []string `json:"mismatches"`
			ReportExists bool     `json:"report_exists"`
		} `json:"replay_outcome"`
		SuiteComparison struct {
			Delta       int  `json:"delta"`
			Score       int  `json:"score"`
			HasVersion  bool `json:"has_version"`
			HasBaseline bool `json:"has_baseline"`
			HasDelta    bool `json:"has_delta"`
		} `json:"suite_comparison"`
		DetailReport       map[string]bool `json:"detail_report"`
		RunReplayIndex     map[string]bool `json:"run_replay_index"`
		MissingReportIndex map[string]bool `json:"missing_report_index"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode evaluation contract output: %v\n%s", err, string(output))
	}

	if decoded.SuccessfulCase.Score != 100 || !decoded.SuccessfulCase.Passed || !decoded.SuccessfulCase.ReplayMatched || !decoded.SuccessfulCase.ReportExists || !decoded.SuccessfulCase.ReplayExists || !decoded.SuccessfulCase.DetailExists || decoded.SuccessfulCase.DetailPagePath == "" {
		t.Fatalf("unexpected successful case payload: %+v", decoded.SuccessfulCase)
	}
	if decoded.FailedCase.Passed || decoded.FailedCase.Score != 60 || len(decoded.FailedCase.FailedCriteria) == 0 || decoded.FailedCase.FailedCriteria[0] != "decision-medium" {
		t.Fatalf("unexpected failed case payload: %+v", decoded.FailedCase)
	}
	if decoded.ReplayOutcome.Matched || !decoded.ReplayOutcome.ReportExists || len(decoded.ReplayOutcome.Mismatches) != 1 || decoded.ReplayOutcome.Mismatches[0] != "medium expected docker got browser" {
		t.Fatalf("unexpected replay outcome payload: %+v", decoded.ReplayOutcome)
	}
	if decoded.SuiteComparison.Delta != 100 || decoded.SuiteComparison.Score != 100 || !decoded.SuiteComparison.HasVersion || !decoded.SuiteComparison.HasBaseline || !decoded.SuiteComparison.HasDelta {
		t.Fatalf("unexpected suite comparison payload: %+v", decoded.SuiteComparison)
	}
	for name, ok := range decoded.DetailReport {
		if !ok {
			t.Fatalf("expected detail report check %s to pass", name)
		}
	}
	for name, ok := range decoded.RunReplayIndex {
		if !ok {
			t.Fatalf("expected run replay index check %s to pass", name)
		}
	}
	for name, ok := range decoded.MissingReportIndex {
		if !ok {
			t.Fatalf("expected missing report index check %s to pass", name)
		}
	}
}
