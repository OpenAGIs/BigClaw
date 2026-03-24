import json
import subprocess
import sys
from pathlib import Path


def test_validation_bundle_benchmark_report_shape(tmp_path: Path) -> None:
    script = Path(__file__).resolve().parents[1] / 'bigclaw-go' / 'scripts' / 'e2e' / 'benchmark_validation_bundle_export.py'
    output_path = tmp_path / 'benchmark-report.json'
    result = subprocess.run(
        [
            sys.executable,
            str(script),
            '--iterations',
            '2',
            '--archive-runs',
            '16',
            '--output',
            str(output_path),
        ],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr
    report = json.loads(output_path.read_text(encoding='utf-8'))
    assert report['status'] == 'ok'
    assert report['benchmark']['iterations'] == 2
    assert report['benchmark']['archive_runs'] == 16
    assert len(report['timings_ms']['samples']) == 2
    assert report['timings_ms']['p95'] >= report['timings_ms']['min']
    assert report['artifacts']['index_path'] == 'docs/reports/live-validation-index.md'
