import subprocess
import sys
from pathlib import Path


def test_scale_followup_digest_check_passes_with_aligned_reports(tmp_path: Path):
    root = tmp_path / "bigclaw-go"
    reports = root / "docs" / "reports"
    reports.mkdir(parents=True)

    files = {
        "scale-validation-follow-up-digest.md": """# Scale Validation Follow-Up Digest

- `docs/reports/queue-reliability-report.md`
- `docs/reports/benchmark-readiness-report.md`
- `docs/reports/long-duration-soak-report.md`
- `docs/reports/soak-local-2000x24.json`
- `docs/reports/review-readiness.md`
- `docs/reports/issue-coverage.md`
- A larger `10k` queue reliability matrix remains open.
- This remains below production-grade capacity certification.
- The current local `2000x24` soak run is the longest local proof.
""",
        "queue-reliability-report.md": "See `docs/reports/scale-validation-follow-up-digest.md`.\n",
        "benchmark-readiness-report.md": "See `docs/reports/scale-validation-follow-up-digest.md`.\n",
        "long-duration-soak-report.md": "See `docs/reports/scale-validation-follow-up-digest.md`.\n",
        "review-readiness.md": "See `docs/reports/scale-validation-follow-up-digest.md`.\n",
        "issue-coverage.md": "See `docs/reports/scale-validation-follow-up-digest.md`.\n",
        "soak-local-2000x24.json": "{}\n",
    }

    for name, content in files.items():
        (reports / name).write_text(content, encoding="utf-8")

    script = Path(__file__).resolve().parents[1] / "bigclaw-go" / "scripts" / "reports" / "check_scale_followup_digest.py"
    result = subprocess.run(
        [sys.executable, str(script), "--go-root", str(root)],
        capture_output=True,
        text=True,
        check=False,
    )

    assert result.returncode == 0, result.stderr
    assert "scale follow-up digest check passed" in result.stdout
