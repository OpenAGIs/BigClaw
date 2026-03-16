import json
import subprocess
import sys
from datetime import datetime, timedelta, timezone
from pathlib import Path


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def test_export_validation_bundle_generates_latest_reports_and_index(tmp_path: Path):
    root = tmp_path / 'repo'
    bundle = root / 'docs' / 'reports' / 'live-validation-runs' / '20260315T120000Z'
    previous_bundle = root / 'docs' / 'reports' / 'live-validation-runs' / '20260314T120000Z'
    state_dir = tmp_path / 'state-local'
    state_dir.mkdir(parents=True)
    (state_dir / 'audit.jsonl').write_text('{"event":"ok"}\n', encoding='utf-8')
    service_log = tmp_path / 'local-service.log'
    service_log.write_text('local service ready\n', encoding='utf-8')

    local_report = {
        'base_url': 'http://127.0.0.1:19090',
        'task': {'id': 'local-1'},
        'status': {'state': 'succeeded'},
        'state_dir': str(state_dir),
        'service_log': str(service_log),
    }
    write_json(bundle / 'sqlite-smoke-report.json', local_report)
    write_json(bundle / 'kubernetes-live-smoke-report.json', {'task': {'id': 'k8s-1'}, 'status': {'state': 'succeeded'}})
    write_json(
        previous_bundle / 'summary.json',
        {
            'run_id': '20260314T120000Z',
            'generated_at': (datetime.now(timezone.utc) - timedelta(hours=30)).isoformat(),
            'status': 'succeeded',
            'bundle_path': 'docs/reports/live-validation-runs/20260314T120000Z',
            'local': {'enabled': True, 'status': 'succeeded'},
            'kubernetes': {'enabled': True, 'status': 'succeeded'},
            'ray': {'enabled': True, 'status': 'succeeded'},
        },
    )

    local_stdout = tmp_path / 'local.stdout'
    local_stderr = tmp_path / 'local.stderr'
    k8s_stdout = tmp_path / 'k8s.stdout'
    k8s_stderr = tmp_path / 'k8s.stderr'
    ray_stdout = tmp_path / 'ray.stdout'
    ray_stderr = tmp_path / 'ray.stderr'
    for path, content in [
        (local_stdout, 'local ok\n'),
        (local_stderr, ''),
        (k8s_stdout, 'k8s ok\n'),
        (k8s_stderr, ''),
        (ray_stdout, 'ray ok\n'),
        (ray_stderr, ''),
    ]:
        path.write_text(content, encoding='utf-8')

    script = Path(__file__).resolve().parents[1] / 'bigclaw-go' / 'scripts' / 'e2e' / 'export_validation_bundle.py'
    result = subprocess.run(
        [
            sys.executable,
            str(script),
            '--go-root', str(root),
            '--run-id', '20260315T120000Z',
            '--bundle-dir', 'docs/reports/live-validation-runs/20260315T120000Z',
            '--summary-path', 'docs/reports/live-validation-summary.json',
            '--index-path', 'docs/reports/live-validation-index.md',
            '--manifest-path', 'docs/reports/live-validation-index.json',
            '--run-local', '1',
            '--run-kubernetes', '1',
            '--run-ray', '1',
            '--continuation-history-window', '2',
            '--continuation-stale-after-hours', '24',
            '--validation-status', '0',
            '--local-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/sqlite-smoke-report.json',
            '--local-stdout-path', str(local_stdout),
            '--local-stderr-path', str(local_stderr),
            '--kubernetes-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/kubernetes-live-smoke-report.json',
            '--kubernetes-stdout-path', str(k8s_stdout),
            '--kubernetes-stderr-path', str(k8s_stderr),
            '--ray-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/ray-live-smoke-report.json',
            '--ray-stdout-path', str(ray_stdout),
            '--ray-stderr-path', str(ray_stderr),
        ],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr

    summary = json.loads((root / 'docs' / 'reports' / 'live-validation-summary.json').read_text(encoding='utf-8'))
    assert summary['status'] == 'succeeded'
    assert summary['local']['canonical_report_path'] == 'docs/reports/sqlite-smoke-report.json'
    assert summary['local']['audit_log_path'].endswith('local.audit.jsonl')
    assert summary['local']['service_log_path'].endswith('local.service.log')
    assert summary['continuation_policy']['history_window'] == 2
    assert summary['continuation_policy']['stale_bundle'] is True
    assert 'component_incomplete:ray' in summary['continuation_policy']['stale_reasons']

    latest_local = json.loads((root / 'docs' / 'reports' / 'sqlite-smoke-report.json').read_text(encoding='utf-8'))
    assert latest_local['task']['id'] == 'local-1'

    index_text = (root / 'docs' / 'reports' / 'live-validation-index.md').read_text(encoding='utf-8')
    assert 'Live Validation Index' in index_text
    assert '20260315T120000Z' in index_text
    assert 'docs/reports/live-validation-runs/20260315T120000Z' in index_text
    assert 'Continuation policy' in index_text

    manifest = json.loads((root / 'docs' / 'reports' / 'live-validation-index.json').read_text(encoding='utf-8'))
    assert manifest['latest']['run_id'] == '20260315T120000Z'
    assert manifest['recent_runs'][0]['run_id'] == '20260315T120000Z'

    scorecard = json.loads(
        (root / 'docs' / 'reports' / 'validation-bundle-continuation-scorecard.json').read_text(encoding='utf-8')
    )
    assert scorecard['history_window'] == 2
    assert len(scorecard['window_runs']) == 2
    assert scorecard['latest_bundle_policy']['run_id'] == '20260315T120000Z'
    assert scorecard['latest_bundle_policy']['stale_bundle'] is True
    assert scorecard['latest_bundle_policy']['ready_for_closeout'] is False

    rerun = subprocess.run(
        [
            sys.executable,
            str(script),
            '--go-root', str(root),
            '--run-id', '20260315T120000Z',
            '--bundle-dir', 'docs/reports/live-validation-runs/20260315T120000Z',
            '--summary-path', 'docs/reports/live-validation-summary.json',
            '--index-path', 'docs/reports/live-validation-index.md',
            '--manifest-path', 'docs/reports/live-validation-index.json',
            '--run-local', '1',
            '--run-kubernetes', '1',
            '--run-ray', '1',
            '--continuation-history-window', '2',
            '--continuation-stale-after-hours', '24',
            '--validation-status', '0',
            '--local-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/sqlite-smoke-report.json',
            '--local-stdout-path', str(bundle / 'local.stdout.log'),
            '--local-stderr-path', str(bundle / 'local.stderr.log'),
            '--kubernetes-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/kubernetes-live-smoke-report.json',
            '--kubernetes-stdout-path', str(bundle / 'kubernetes.stdout.log'),
            '--kubernetes-stderr-path', str(bundle / 'kubernetes.stderr.log'),
            '--ray-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/ray-live-smoke-report.json',
            '--ray-stdout-path', str(bundle / 'ray.stdout.log'),
            '--ray-stderr-path', str(bundle / 'ray.stderr.log'),
        ],
        check=False,
        capture_output=True,
        text=True,
    )
    assert rerun.returncode == 0, rerun.stderr
