import importlib.util
import json
import subprocess
import sys
from pathlib import Path


def load_gate_module():
    script_path = Path('bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('validation_bundle_continuation_policy_gate', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


gate = load_gate_module()


def test_policy_gate_holds_for_partial_lane_history(tmp_path: Path) -> None:
    scorecard = {
        'summary': {
            'latest_run_id': 'synthetic-run',
            'latest_bundle_age_hours': 1.5,
            'recent_bundle_count': 2,
            'latest_all_executor_tracks_succeeded': True,
            'recent_bundle_chain_has_no_failures': True,
            'all_executor_tracks_have_repeated_recent_coverage': False,
        },
        'shared_queue_companion': {
            'available': True,
            'cross_node_completions': 99,
            'duplicate_completed_tasks': 0,
            'duplicate_started_tasks': 0,
            'mode': 'standalone-proof',
            'report_path': 'bigclaw-go/docs/reports/multi-node-shared-queue-report.json',
        },
    }
    scorecard_path = tmp_path / 'scorecard.json'
    scorecard_path.write_text(json.dumps(scorecard), encoding='utf-8')

    report = gate.build_report(scorecard_path=str(scorecard_path))

    assert report['status'] == 'policy-hold'
    assert report['recommendation'] == 'hold'
    assert 'repeated_lane_coverage_meets_policy' in report['failing_checks']
    assert report['summary']['failing_check_count'] == 1
    assert 'refresh another full validation bundle with `ray` enabled' in report['next_actions'][0]


def test_policy_gate_can_allow_partial_lane_history(tmp_path: Path) -> None:
    scorecard = {
        'summary': {
            'latest_run_id': 'synthetic-run',
            'latest_bundle_age_hours': 1.5,
            'recent_bundle_count': 2,
            'latest_all_executor_tracks_succeeded': True,
            'recent_bundle_chain_has_no_failures': True,
            'all_executor_tracks_have_repeated_recent_coverage': False,
        },
        'shared_queue_companion': {
            'available': True,
            'cross_node_completions': 99,
            'duplicate_completed_tasks': 0,
            'duplicate_started_tasks': 0,
            'mode': 'standalone-proof',
            'report_path': 'bigclaw-go/docs/reports/multi-node-shared-queue-report.json',
        },
    }
    scorecard_path = tmp_path / 'scorecard.json'
    scorecard_path.write_text(json.dumps(scorecard), encoding='utf-8')

    report = gate.build_report(scorecard_path=str(scorecard_path), require_repeated_lane_coverage=False)

    assert report['status'] == 'policy-go'
    assert report['recommendation'] == 'go'
    assert report['failing_checks'] == []
    assert 'standalone shared-queue companion proof' in report['next_actions'][0]


def test_policy_gate_recommends_inline_refresh_for_bundled_companion(tmp_path: Path) -> None:
    scorecard = {
        'summary': {
            'latest_run_id': 'synthetic-run',
            'latest_bundle_age_hours': 1.5,
            'recent_bundle_count': 2,
            'latest_all_executor_tracks_succeeded': True,
            'recent_bundle_chain_has_no_failures': False,
            'all_executor_tracks_have_repeated_recent_coverage': True,
        },
        'shared_queue_companion': {
            'available': False,
            'cross_node_completions': 0,
            'duplicate_completed_tasks': 0,
            'duplicate_started_tasks': 0,
            'mode': 'bundled-companion',
            'report_path': 'docs/reports/live-validation-runs/20260321T010203Z/multi-node-shared-queue-report.json',
        },
    }
    scorecard_path = tmp_path / 'scorecard.json'
    scorecard_path.write_text(json.dumps(scorecard), encoding='utf-8')

    report = gate.build_report(scorecard_path=str(scorecard_path))

    assert report['status'] == 'policy-hold'
    assert 'shared_queue_companion_available' in report['failing_checks']
    assert 'BIGCLAW_E2E_REFRESH_SHARED_QUEUE=1' in report['next_actions'][0]
    shared_queue_check = next(item for item in report['policy_checks'] if item['name'] == 'shared_queue_companion_available')
    assert 'mode=bundled-companion' in shared_queue_check['detail']


def test_checked_in_policy_gate_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json').read_text())

    assert report['status'] == 'policy-go'
    assert report['recommendation'] == 'go'
    assert report['summary']['latest_run_id'] == '20260316T140138Z'
    assert report['failing_checks'] == []


def test_policy_gate_cli_returns_zero_for_checked_in_go() -> None:
    script = Path('bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py')
    result = subprocess.run([sys.executable, str(script)], check=False, capture_output=True, text=True)

    assert result.returncode == 0
