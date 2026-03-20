import json
import os
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
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


def test_refresh_continuation_artifacts_returns_stable_schema_when_disabled(tmp_path: Path):
    export_validation_bundle = load_export_validation_bundle_module()
    result = export_validation_bundle.refresh_continuation_artifacts(
        root=tmp_path,
        mode='off',
        scorecard_path='docs/reports/validation-bundle-continuation-scorecard.json',
        policy_gate_path='docs/reports/validation-bundle-continuation-policy-gate.json',
        shared_queue_report_path='docs/reports/multi-node-shared-queue-report.json',
    )

    assert result['refreshed'] is False
    assert result['reason'] == 'disabled'
    assert result['scorecard_status'] == 'not-refreshed'
    assert result['policy_gate_status'] == 'not-refreshed'
    assert result['policy_gate_recommendation'] == 'unknown'
    assert result['latest_bundle_age_hours'] is None
    assert result['failing_checks'] == []
    assert result['next_actions'] == []


def test_refresh_continuation_artifacts_returns_stable_schema_when_auto_skips(tmp_path: Path):
    export_validation_bundle = load_export_validation_bundle_module()
    result = export_validation_bundle.refresh_continuation_artifacts(
        root=tmp_path,
        mode='auto',
        scorecard_path='docs/reports/validation-bundle-continuation-scorecard.json',
        policy_gate_path='docs/reports/validation-bundle-continuation-policy-gate.json',
        shared_queue_report_path='docs/reports/multi-node-shared-queue-report.json',
    )

    assert result['refreshed'] is False
    assert 'missing shared queue report' in result['reason']
    assert result['scorecard_status'] == 'not-refreshed'
    assert result['policy_gate_status'] == 'skipped'
    assert result['policy_gate_recommendation'] == 'unknown'
    assert result['latest_bundle_age_hours'] is None
    assert result['failing_checks'] == []
    assert result['next_actions'] == []


def test_build_continuation_overview_and_index_show_skipped_state(tmp_path: Path):
    export_validation_bundle = load_export_validation_bundle_module()
    root = tmp_path / 'bigclaw-go'
    root.mkdir(parents=True, exist_ok=True)

    summary = {
        'run_id': '20260321T010203Z',
        'generated_at': '2026-03-21T01:02:03Z',
        'status': 'succeeded',
        'bundle_path': 'docs/reports/live-validation-runs/20260321T010203Z',
        'summary_path': 'docs/reports/live-validation-runs/20260321T010203Z/summary.json',
        'closeout_commands': ['cd bigclaw-go && ./scripts/e2e/run_all.sh'],
        'local': {'enabled': False, 'status': 'skipped', 'bundle_report_path': 'x', 'canonical_report_path': 'y'},
        'kubernetes': {'enabled': False, 'status': 'skipped', 'bundle_report_path': 'x', 'canonical_report_path': 'y'},
        'ray': {'enabled': False, 'status': 'skipped', 'bundle_report_path': 'x', 'canonical_report_path': 'y'},
        'continuation': export_validation_bundle.build_continuation_result(
            'auto',
            'docs/reports/validation-bundle-continuation-scorecard.json',
            'docs/reports/validation-bundle-continuation-policy-gate.json',
        ),
    }
    summary['continuation']['policy_gate_status'] = 'skipped'
    summary['continuation']['reason'] = 'missing shared queue report: docs/reports/multi-node-shared-queue-report.json'

    overview = export_validation_bundle.build_continuation_overview(root, summary['continuation'])
    assert overview is not None
    assert overview['status'] == 'skipped'
    assert overview['recommendation'] == 'unknown'
    assert 'missing shared queue report' in overview['reason']

    index_text = export_validation_bundle.render_index(
        summary,
        recent_runs=[],
        continuation_overview=overview,
        continuation_artifacts=[('docs/reports/validation-bundle-continuation-policy-gate.json', 'gate')],
        followup_digests=[],
    )
    assert 'Gate status: `skipped`' in index_text
    assert 'Recommendation: `unknown`' in index_text
    assert 'Refresh state: `missing shared queue report: docs/reports/multi-node-shared-queue-report.json`' in index_text


def test_export_validation_bundle_generates_latest_reports_and_index(tmp_path: Path):
    repo_root = tmp_path / 'repo'
    root = repo_root / 'bigclaw-go'
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
    write_json(
        root / 'docs' / 'reports' / 'multi-node-shared-queue-report.json',
        {
            'all_ok': True,
            'cross_node_completions': 12,
            'duplicate_completed_tasks': [],
            'duplicate_started_tasks': [],
        },
    )

    (root / 'docs' / 'reports').mkdir(parents=True, exist_ok=True)
    (root / 'docs' / 'reports' / 'validation-bundle-continuation-scorecard.json').write_text('{}\n', encoding='utf-8')
    (root / 'docs' / 'reports' / 'validation-bundle-continuation-policy-gate.json').write_text('{}\n', encoding='utf-8')
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
            '--shared-queue-source', 'existing-report',
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
    assert summary['shared_queue']['available'] is True
    assert summary['shared_queue']['bundle_report_path'].endswith('multi-node-shared-queue-report.json')
    assert summary['shared_queue']['cross_node_completions'] == 12
    assert summary['shared_queue']['source'] == 'existing-report'
    assert summary['continuation']['refreshed'] is True
    assert summary['continuation']['policy_gate_status'] == 'policy-hold'
    assert summary['continuation']['policy_gate_recommendation'] == 'hold'
    assert summary['continuation']['failing_checks']
    assert summary['continuation']['latest_bundle_age_hours'] == 0.0
    assert summary['continuation']['next_actions']

    latest_local = json.loads((root / 'docs' / 'reports' / 'sqlite-smoke-report.json').read_text(encoding='utf-8'))
    assert latest_local['task']['id'] == 'local-1'
    latest_shared_queue = json.loads((root / 'docs' / 'reports' / 'multi-node-shared-queue-report.json').read_text(encoding='utf-8'))
    assert latest_shared_queue['cross_node_completions'] == 12
    bundled_shared_queue = json.loads((bundle / 'multi-node-shared-queue-report.json').read_text(encoding='utf-8'))
    assert bundled_shared_queue['all_ok'] is True
    scorecard = json.loads((root / 'docs' / 'reports' / 'validation-bundle-continuation-scorecard.json').read_text(encoding='utf-8'))
    assert scorecard['summary']['latest_run_id'] == '20260315T120000Z'
    assert scorecard['shared_queue_companion']['mode'] == 'bundled-companion'

    index_text = (root / 'docs' / 'reports' / 'live-validation-index.md').read_text(encoding='utf-8')
    assert 'Live Validation Index' in index_text
    assert '20260315T120000Z' in index_text
    assert 'docs/reports/live-validation-runs/20260315T120000Z' in index_text
    assert 'docs/reports/validation-bundle-continuation-scorecard.json' in index_text
    assert 'docs/reports/validation-bundle-continuation-policy-gate.json' in index_text
    assert 'docs/reports/validation-bundle-continuation-digest.md' in index_text
    assert 'Cross-node completions: `12`' in index_text
    assert 'Source: `existing-report`' in index_text
    assert 'Gate status: `policy-hold`' in index_text
    assert 'Recommendation: `hold`' in index_text
    assert 'Failing checks: `' in index_text
    assert 'Next action:' in index_text

    manifest = json.loads((root / 'docs' / 'reports' / 'live-validation-index.json').read_text(encoding='utf-8'))
    assert manifest['latest']['run_id'] == '20260315T120000Z'
    assert manifest['recent_runs'][0]['run_id'] == '20260315T120000Z'


def test_run_all_can_refresh_shared_queue_before_bundle_export(tmp_path: Path):
    repo_root = tmp_path / 'repo'
    root = repo_root / 'bigclaw-go'
    scripts_dir = root / 'scripts' / 'e2e'
    scripts_dir.mkdir(parents=True, exist_ok=True)

    source_run_all = Path(__file__).resolve().parents[1] / 'bigclaw-go' / 'scripts' / 'e2e' / 'run_all.sh'
    run_all = scripts_dir / 'run_all.sh'
    run_all.write_text(source_run_all.read_text(encoding='utf-8'), encoding='utf-8')
    run_all.chmod(0o755)

    exporter = scripts_dir / 'export_validation_bundle.py'
    exporter.write_text(
        """#!/usr/bin/env python3
import argparse
import json
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('--go-root', required=True)
parser.add_argument('--summary-path', required=True)
parser.add_argument('--index-path', required=True)
parser.add_argument('--manifest-path', required=True)
parser.add_argument('--run-id', required=True)
parser.add_argument('--bundle-dir', required=True)
parser.add_argument('--refresh-continuation', required=True)
parser.add_argument('--shared-queue-source', required=True)
parser.add_argument('--local-report-path', required=True)
parser.add_argument('--local-stdout-path', required=True)
parser.add_argument('--local-stderr-path', required=True)
parser.add_argument('--kubernetes-report-path', required=True)
parser.add_argument('--kubernetes-stdout-path', required=True)
parser.add_argument('--kubernetes-stderr-path', required=True)
parser.add_argument('--ray-report-path', required=True)
parser.add_argument('--ray-stdout-path', required=True)
parser.add_argument('--ray-stderr-path', required=True)
parser.add_argument('--run-local', default='1')
parser.add_argument('--run-kubernetes', default='1')
parser.add_argument('--run-ray', default='1')
parser.add_argument('--validation-status', default='0')
args = parser.parse_args()

root = Path(args.go_root)
shared_queue_path = root / 'docs' / 'reports' / 'multi-node-shared-queue-report.json'
shared_queue = json.loads(shared_queue_path.read_text(encoding='utf-8'))
summary = {
    'run_id': args.run_id,
    'status': 'succeeded' if args.validation_status == '0' else 'failed',
    'continuation': {
        'mode': args.refresh_continuation,
    },
    'shared_queue': {
        'available': shared_queue.get('all_ok', False),
        'cross_node_completions': shared_queue.get('cross_node_completions', 0),
        'source': args.shared_queue_source,
    },
}
for rel in (args.summary_path, args.manifest_path):
    path = root / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {'latest': summary, 'recent_runs': [summary]} if rel == args.manifest_path else summary
    path.write_text(json.dumps(payload, indent=2) + '\\n', encoding='utf-8')
(root / args.index_path).write_text('shared queue refreshed\\n', encoding='utf-8')
print(json.dumps(summary))
""",
        encoding='utf-8',
    )
    exporter.chmod(0o755)

    shared_queue = scripts_dir / 'multi_node_shared_queue.py'
    shared_queue.write_text(
        """#!/usr/bin/env python3
import argparse
import json
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument('--go-root', required=True)
parser.add_argument('--report-path', required=True)
parser.add_argument('--count', type=int, required=True)
parser.add_argument('--submit-workers', type=int, required=True)
parser.add_argument('--timeout-seconds', type=int, required=True)
args = parser.parse_args()

root = Path(args.go_root)
report_path = root / args.report_path
report_path.parent.mkdir(parents=True, exist_ok=True)
report_path.write_text(json.dumps({
    'all_ok': True,
    'cross_node_completions': args.count - 1,
    'duplicate_completed_tasks': [],
    'duplicate_started_tasks': [],
    'missing_completed_tasks': [],
}) + '\\n', encoding='utf-8')
print(json.dumps({'status': 'ok', 'report_path': str(report_path)}))
""",
        encoding='utf-8',
    )
    shared_queue.chmod(0o755)

    env = os.environ.copy()
    env.update(
        {
            'BIGCLAW_E2E_RUN_LOCAL': '0',
            'BIGCLAW_E2E_RUN_KUBERNETES': '0',
            'BIGCLAW_E2E_RUN_RAY': '0',
            'BIGCLAW_E2E_REFRESH_SHARED_QUEUE': '1',
            'BIGCLAW_E2E_REFRESH_CONTINUATION': '0',
            'BIGCLAW_E2E_SHARED_QUEUE_COUNT': '17',
            'BIGCLAW_E2E_SHARED_QUEUE_SUBMIT_WORKERS': '3',
            'BIGCLAW_E2E_SHARED_QUEUE_TIMEOUT_SECONDS': '9',
            'BIGCLAW_E2E_RUN_ID': '20260321T010203Z',
        }
    )
    result = subprocess.run(
        ['bash', str(run_all)],
        cwd=root,
        env=env,
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr
    summary = json.loads((root / 'docs' / 'reports' / 'live-validation-summary.json').read_text(encoding='utf-8'))
    assert summary['shared_queue']['available'] is True
    assert summary['shared_queue']['cross_node_completions'] == 16
    assert summary['shared_queue']['source'] == 'inline-workflow-refresh'
    assert summary['continuation']['mode'] == 'off'
