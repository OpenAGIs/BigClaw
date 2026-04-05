import json
import subprocess
import sys
from pathlib import Path


def run_scorecard(output_path: Path) -> dict:
    result = subprocess.run(
        [
            "go",
            "run",
            "./scripts/e2e/validation_bundle_continuation_scorecard",
            "--output",
            str(output_path),
        ],
        check=False,
        capture_output=True,
        text=True,
        cwd="bigclaw-go",
    )
    assert result.returncode == 0, result.stderr
    return json.loads(output_path.read_text())


def test_continuation_scorecard_summarizes_recent_bundle_chain(tmp_path: Path) -> None:
    report = run_scorecard(tmp_path / "scorecard.json")

    assert report['status'] == 'local-continuation-scorecard'
    assert report['summary']['recent_bundle_count'] == 3
    assert report['summary']['latest_run_id'] == '20260316T140138Z'
    assert report['summary']['latest_all_executor_tracks_succeeded'] is True
    assert report['summary']['recent_bundle_chain_has_no_failures'] is True
    assert report['summary']['all_executor_tracks_have_repeated_recent_coverage'] is True
    assert report['shared_queue_companion']['cross_node_completions'] == 99
    assert report['shared_queue_companion']['mode'] == 'bundle-companion-summary'
    assert report['shared_queue_companion']['summary_path'] == 'docs/reports/shared-queue-companion-summary.json'


def test_continuation_scorecard_marks_lane_success_and_manual_boundary(tmp_path: Path) -> None:
    report = run_scorecard(tmp_path / "scorecard.json")
    lanes = {item['lane']: item for item in report['executor_lanes']}
    manual_boundary = next(item for item in report['continuation_checks'] if item['name'] == 'continuation_surface_is_workflow_triggered')
    repeated_coverage = next(item for item in report['continuation_checks'] if item['name'] == 'all_executor_tracks_have_repeated_recent_coverage')

    assert set(lanes) == {'local', 'kubernetes', 'ray'}
    assert all(item['latest_status'] == 'succeeded' for item in lanes.values())
    assert lanes['local']['consecutive_successes'] == 3
    assert lanes['kubernetes']['consecutive_successes'] == 3
    assert lanes['ray']['consecutive_successes'] == 2
    assert lanes['ray']['enabled_runs'] == 2
    assert all(item['all_recent_runs_succeeded'] for item in lanes.values())
    assert repeated_coverage['passed'] is True
    assert "ray:2" in repeated_coverage['detail']
    assert manual_boundary['passed'] is True
    assert 'workflow execution' in manual_boundary['detail']


def test_checked_in_continuation_scorecard_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json').read_text())

    assert report['status'] == 'local-continuation-scorecard'
    assert report['summary']['latest_run_id'] == '20260316T140138Z'
    assert report['summary']['all_executor_tracks_have_repeated_recent_coverage'] is True
    assert report['shared_queue_companion']['cross_node_completions'] == 99
    assert report['shared_queue_companion']['duplicate_completed_tasks'] == 0
    assert report['shared_queue_companion']['mode'] == 'bundle-companion-summary'
    assert report['executor_lanes'][0]['lane'] == 'local'
