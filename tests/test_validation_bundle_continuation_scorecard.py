import importlib.util
import json
import sys
from pathlib import Path


def load_continuation_module():
    script_path = Path('bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('validation_bundle_continuation_scorecard', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


continuation = load_continuation_module()


def test_continuation_scorecard_summarizes_recent_bundle_chain() -> None:
    report = continuation.build_report()

    assert report['status'] == 'local-continuation-scorecard'
    assert report['summary']['recent_bundle_count'] == 2
    assert report['summary']['latest_run_id'] == '20260314T164647Z'
    assert report['summary']['latest_all_executor_tracks_succeeded'] is True
    assert report['summary']['recent_bundle_chain_has_no_failures'] is True
    assert report['summary']['all_executor_tracks_have_repeated_recent_coverage'] is False
    assert report['shared_queue_companion']['cross_node_completions'] == 99


def test_continuation_scorecard_marks_lane_success_and_manual_boundary() -> None:
    report = continuation.build_report()
    lanes = {item['lane']: item for item in report['executor_lanes']}
    manual_boundary = next(item for item in report['continuation_checks'] if item['name'] == 'continuation_surface_is_manual')
    repeated_coverage = next(item for item in report['continuation_checks'] if item['name'] == 'all_executor_tracks_have_repeated_recent_coverage')

    assert set(lanes) == {'local', 'kubernetes', 'ray'}
    assert all(item['latest_status'] == 'succeeded' for item in lanes.values())
    assert lanes['local']['consecutive_successes'] == 2
    assert lanes['kubernetes']['consecutive_successes'] == 2
    assert lanes['ray']['consecutive_successes'] == 1
    assert lanes['ray']['enabled_runs'] == 1
    assert all(item['all_recent_runs_succeeded'] for item in lanes.values())
    assert repeated_coverage['passed'] is False
    assert "'ray': 1" in repeated_coverage['detail']
    assert manual_boundary['passed'] is True
    assert 'manual' in manual_boundary['detail']


def test_checked_in_continuation_scorecard_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json').read_text())

    assert report['status'] == 'local-continuation-scorecard'
    assert report['summary']['latest_run_id'] == '20260314T164647Z'
    assert report['summary']['all_executor_tracks_have_repeated_recent_coverage'] is False
    assert report['shared_queue_companion']['cross_node_completions'] == 99
    assert report['shared_queue_companion']['duplicate_completed_tasks'] == 0
    assert report['executor_lanes'][0]['lane'] == 'local'
