import json
import subprocess
from pathlib import Path


def run_gate(scorecard_path: Path, output_path: Path, *extra_args: str) -> tuple[subprocess.CompletedProcess[str], dict]:
    result = subprocess.run(
        [
            "go",
            "run",
            "./scripts/e2e/validation_bundle_continuation_policy_gate",
            "--scorecard",
            str(scorecard_path),
            "--output",
            str(output_path),
            *extra_args,
        ],
        check=False,
        capture_output=True,
        text=True,
        cwd="bigclaw-go",
    )
    return result, json.loads(output_path.read_text())


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

    result, report = run_gate(scorecard_path, tmp_path / 'gate.json')

    assert result.returncode != 0
    assert report['status'] == 'policy-hold'
    assert report['recommendation'] == 'hold'
    assert 'repeated_lane_coverage_meets_policy' in report['failing_checks']
    assert report['summary']['failing_check_count'] == 1


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

    result, report = run_gate(scorecard_path, tmp_path / 'gate.json', '--allow-partial-lane-history')

    assert result.returncode == 0
    assert report['status'] == 'policy-go'
    assert report['recommendation'] == 'go'
    assert report['failing_checks'] == []


def test_checked_in_policy_gate_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json').read_text())

    assert report['status'] == 'policy-go'
    assert report['recommendation'] == 'go'
    assert report['summary']['latest_run_id'] == '20260316T140138Z'
    assert report['failing_checks'] == []


def test_policy_gate_cli_returns_zero_for_checked_in_go() -> None:
    output_path = Path('bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json')
    result = subprocess.run(
        [
            'go',
            'run',
            './scripts/e2e/validation_bundle_continuation_policy_gate',
            '--scorecard',
            'docs/reports/validation-bundle-continuation-scorecard.json',
            '--output',
            str(Path(output_path).resolve()),
            '--allow-partial-lane-history',
            '--enforcement-mode',
            'review',
        ],
        check=False,
        capture_output=True,
        text=True,
        cwd='bigclaw-go',
    )

    assert result.returncode == 0, result.stderr
