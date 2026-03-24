import json
import subprocess
import sys
import importlib.util
from pathlib import Path


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def load_export_validation_bundle_module():
    script = Path(__file__).resolve().parents[1] / 'bigclaw-go' / 'scripts' / 'e2e' / 'export_validation_bundle.py'
    spec = importlib.util.spec_from_file_location('export_validation_bundle', script)
    assert spec is not None and spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_export_validation_bundle_generates_latest_reports_and_index(tmp_path: Path):
    root = tmp_path / 'repo'
    bundle = root / 'docs' / 'reports' / 'live-validation-runs' / '20260315T120000Z'
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
    write_json(bundle / 'ray-live-smoke-report.json', {'task': {'id': 'ray-1'}, 'status': {'state': 'succeeded'}})

    (root / 'docs' / 'reports').mkdir(parents=True, exist_ok=True)
    write_json(
        root / 'docs' / 'reports' / 'multi-node-shared-queue-report.json',
        {
            'generated_at': '2026-03-15T11:55:00Z',
            'count': 12,
            'submitted_by_node': {'node-a': 6, 'node-b': 6},
            'completed_by_node': {'node-a': 5, 'node-b': 7},
            'cross_node_completions': 4,
            'duplicate_started_tasks': [],
            'duplicate_completed_tasks': [],
            'missing_completed_tasks': [],
            'all_ok': True,
            'nodes': [{'name': 'node-a'}, {'name': 'node-b'}],
        },
    )
    (root / 'docs' / 'reports' / 'validation-bundle-continuation-scorecard.json').write_text('{}\n', encoding='utf-8')
    write_json(
        root / 'docs' / 'reports' / 'validation-bundle-continuation-policy-gate.json',
        {
            'status': 'policy-hold',
            'recommendation': 'hold',
            'failing_checks': ['latest_bundle_all_executor_tracks_succeeded'],
            'enforcement': {'mode': 'review'},
            'summary': {'latest_bundle_age_hours': 12.5},
            'reviewer_path': {'digest': 'docs/reports/validation-bundle-continuation-digest.md'},
            'next_actions': ['rerun ./scripts/e2e/run_all.sh'],
        },
    )
    (root / 'docs' / 'reports' / 'validation-bundle-continuation-digest.md').write_text('# digest\n', encoding='utf-8')

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
    assert summary['shared_queue_companion']['status'] == 'succeeded'
    assert summary['shared_queue_companion']['cross_node_completions'] == 4
    assert summary['shared_queue_companion']['canonical_summary_path'] == 'docs/reports/shared-queue-companion-summary.json'
    assert summary['shared_queue_companion']['bundle_summary_path'].endswith('shared-queue-companion-summary.json')
    assert summary['continuation_gate']['status'] == 'policy-hold'
    assert summary['continuation_gate']['recommendation'] == 'hold'
    assert summary['continuation_gate']['summary']['latest_bundle_age_hours'] == 12.5
    assert summary['continuation_gate']['failing_checks'] == ['latest_bundle_all_executor_tracks_succeeded']
    assert summary['continuation_gate']['next_actions'] == ['rerun ./scripts/e2e/run_all.sh']

    latest_local = json.loads((root / 'docs' / 'reports' / 'sqlite-smoke-report.json').read_text(encoding='utf-8'))
    assert latest_local['task']['id'] == 'local-1'
    shared_queue_summary = json.loads(
        (root / 'docs' / 'reports' / 'shared-queue-companion-summary.json').read_text(encoding='utf-8')
    )
    assert shared_queue_summary['nodes'] == ['node-a', 'node-b']
    assert shared_queue_summary['bundle_report_path'].endswith('multi-node-shared-queue-report.json')

    index_text = (root / 'docs' / 'reports' / 'live-validation-index.md').read_text(encoding='utf-8')
    assert 'Live Validation Index' in index_text
    assert '20260315T120000Z' in index_text
    assert 'docs/reports/live-validation-runs/20260315T120000Z' in index_text
    assert 'shared-queue companion' in index_text
    assert 'docs/reports/shared-queue-companion-summary.json' in index_text
    assert 'docs/reports/multi-node-shared-queue-report.json' in index_text
    assert 'docs/reports/validation-bundle-continuation-scorecard.json' in index_text
    assert 'docs/reports/validation-bundle-continuation-policy-gate.json' in index_text
    assert 'docs/reports/validation-bundle-continuation-digest.md' in index_text

    manifest = json.loads((root / 'docs' / 'reports' / 'live-validation-index.json').read_text(encoding='utf-8'))
    assert manifest['latest']['run_id'] == '20260315T120000Z'
    assert manifest['latest']['shared_queue_companion']['nodes'] == ['node-a', 'node-b']
    assert manifest['recent_runs'][0]['run_id'] == '20260315T120000Z'


def test_build_recent_runs_stops_after_limit_for_timestamped_archives(tmp_path, monkeypatch) -> None:
    module = load_export_validation_bundle_module()
    bundle_root = tmp_path / 'docs' / 'reports' / 'live-validation-runs'
    bundle_root.mkdir(parents=True, exist_ok=True)

    for minute in range(12):
        run_id = f'20260316T12{minute:02d}00Z'
        write_json(
            bundle_root / run_id / 'summary.json',
            {
                'run_id': run_id,
                'generated_at': f'2026-03-16T12:{minute:02d}:00+00:00',
                'status': 'succeeded',
                'bundle_path': f'docs/reports/live-validation-runs/{run_id}',
                'summary_path': f'docs/reports/live-validation-runs/{run_id}/summary.json',
            },
        )

    original_read_json = module.read_json
    calls: list[str] = []

    def counting_read_json(path):
        calls.append(Path(path).parent.name)
        return original_read_json(path)

    monkeypatch.setattr(module, 'read_json', counting_read_json)

    recent_runs = module.build_recent_runs(bundle_root, tmp_path, limit=4)

    assert [item['run_id'] for item in recent_runs] == [
        '20260316T121100Z',
        '20260316T121000Z',
        '20260316T120900Z',
        '20260316T120800Z',
    ]
    assert len(calls) == 4
