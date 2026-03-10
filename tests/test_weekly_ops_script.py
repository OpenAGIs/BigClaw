import json
import subprocess
import sys
from pathlib import Path


def test_generate_weekly_ops_report_script(tmp_path: Path):
    ledger_path = tmp_path / "ledger.json"
    out_path = tmp_path / "reports" / "weekly.md"
    ledger_path.write_text(
        json.dumps(
            [
                {
                    "run_id": "run-1",
                    "task_id": "OPE-78",
                    "title": "Generate weekly report",
                    "source": "linear",
                    "medium": "docker",
                    "status": "completed",
                    "started_at": "2026-03-09T09:00:00Z",
                    "summary": "report generated",
                    "artifacts": [],
                    "audits": [],
                    "traces": [],
                },
                {
                    "run_id": "run-2",
                    "task_id": "OPE-75",
                    "title": "Manual approval queue",
                    "source": "jira",
                    "medium": "vm",
                    "status": "needs-approval",
                    "started_at": "2026-03-09T12:00:00Z",
                    "summary": "approval pending",
                    "artifacts": [],
                    "audits": [{"action": "scheduler.decision", "actor": "scheduler", "outcome": "pending", "details": {"reason": "manual review"}}],
                    "traces": [],
                },
            ]
        )
    )

    result = subprocess.run(
        [
            sys.executable,
            "scripts/generate_weekly_ops_report.py",
            "--ledger",
            str(ledger_path),
            "--out",
            str(out_path),
            "--team",
            "OpenAGI",
            "--period-start",
            "2026-03-09T00:00:00Z",
            "--period-end",
            "2026-03-09T23:59:59Z",
            "--generated-at",
            "2026-03-10T00:00:00Z",
        ],
        cwd=Path(__file__).resolve().parents[1],
        capture_output=True,
        text=True,
        check=True,
    )

    payload = json.loads(result.stdout)
    assert payload == {
        "out": str(out_path),
        "total_runs": 2,
        "successful_runs": 1,
        "approvals_pending": 1,
    }
    assert out_path.exists()
    content = out_path.read_text()
    assert "# Weekly Operations Report" in content
    assert "Total Runs: 2" in content
    assert "Approvals Pending: 1" in content
    assert "OPE-75/run-2: status=needs-approval" in content
