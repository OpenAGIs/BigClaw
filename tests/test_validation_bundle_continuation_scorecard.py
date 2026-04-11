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


def load_cross_process_coordination_surface_module():
    script_path = Path('bigclaw-go/scripts/e2e/cross_process_coordination_surface.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('cross_process_coordination_surface', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


def load_subscriber_takeover_fault_matrix_module():
    script_path = Path('bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('subscriber_takeover_fault_matrix', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


coordination_surface = load_cross_process_coordination_surface_module()
subscriber_takeover = load_subscriber_takeover_fault_matrix_module()


def test_continuation_scorecard_summarizes_recent_bundle_chain() -> None:
    report = continuation.build_report()

    assert report['status'] == 'local-continuation-scorecard'
    assert report['summary']['recent_bundle_count'] == 3
    assert report['summary']['latest_run_id'] == '20260316T140138Z'
    assert report['summary']['latest_all_executor_tracks_succeeded'] is True
    assert report['summary']['recent_bundle_chain_has_no_failures'] is True
    assert report['summary']['all_executor_tracks_have_repeated_recent_coverage'] is True
    assert report['shared_queue_companion']['cross_node_completions'] == 99
    assert report['shared_queue_companion']['mode'] == 'bundle-companion-summary'
    assert report['shared_queue_companion']['summary_path'] == 'docs/reports/shared-queue-companion-summary.json'


def test_continuation_scorecard_marks_lane_success_and_manual_boundary() -> None:
    report = continuation.build_report()
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
    assert "'ray': 2" in repeated_coverage['detail']
    assert manual_boundary['passed'] is True
    assert 'workflow execution' in manual_boundary['detail']


def test_coordination_surface_summarizes_current_live_and_local_proofs() -> None:
    report = coordination_surface.build_report()

    assert report['status'] == 'local-capability-surface'
    assert report['runtime_readiness_levels']['live_proven'].startswith('Shipped runtime behavior')
    assert report['summary']['shared_queue_cross_node_completions'] == 99
    assert report['summary']['takeover_passing_scenarios'] == 3
    assert 'no partitioned topic model' in report['current_ceiling']


def test_coordination_surface_marks_partitioned_and_broker_models_unavailable() -> None:
    report = coordination_surface.build_report()
    by_capability = {item['capability']: item for item in report['capabilities']}

    assert by_capability['partitioned_topic_routing']['current_state'] == 'not_available'
    assert by_capability['partitioned_topic_routing']['runtime_readiness'] == 'contract_only'
    assert by_capability['broker_backed_subscriber_ownership']['current_state'] == 'not_available'
    assert by_capability['broker_backed_subscriber_ownership']['runtime_readiness'] == 'contract_only'
    assert by_capability['shared_queue_task_coordination']['runtime_readiness'] == 'live_proven'
    assert by_capability['subscriber_takeover_semantics']['deterministic_local_harness'] is True
    assert by_capability['subscriber_takeover_semantics']['runtime_readiness'] == 'live_proven'


def test_checked_in_coordination_surface_matches_expected_shape() -> None:
    report = json.loads(
        Path('bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json').read_text()
    )

    assert report['status'] == 'local-capability-surface'
    assert sorted(report['runtime_readiness_levels']) == [
        'contract_only',
        'harness_proven',
        'live_proven',
        'supporting_surface',
    ]
    assert report['summary']['shared_queue_duplicate_completed_tasks'] == 0
    assert report['summary']['takeover_stale_write_rejections'] == 2
    assert report['capabilities'][0]['capability'] == 'shared_queue_task_coordination'
    assert report['capabilities'][0]['runtime_readiness'] == 'live_proven'


def test_takeover_harness_report_has_three_passing_scenarios() -> None:
    report = subscriber_takeover.build_report()

    assert report['status'] == 'local-executable'
    assert report['summary']['scenario_count'] == 3
    assert report['summary']['passing_scenarios'] == 3
    assert report['summary']['failing_scenarios'] == 0
    assert report['summary']['stale_write_rejections'] == 2
    assert report['summary']['duplicate_delivery_count'] == 4


def test_stale_writer_scenario_records_rejection_and_final_owner() -> None:
    report = subscriber_takeover.build_report()
    scenario = next(item for item in report['scenarios'] if item['id'] == 'lease-expiry-stale-writer-rejected')

    assert scenario['stale_write_rejections'] == 1
    assert scenario['checkpoint_after']['owner'] == scenario['takeover_subscriber']
    assert 'evt-81' in scenario['duplicate_events']
    assert scenario['all_assertions_passed'] is True


def test_checked_in_takeover_report_matches_local_harness_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json').read_text())

    assert report['status'] == 'local-executable'
    assert report['summary']['scenario_count'] == 3
    assert all(item['all_assertions_passed'] for item in report['scenarios'])
    split_brain = next(item for item in report['scenarios'] if item['id'] == 'split-brain-dual-replay-window')
    assert split_brain['duplicate_delivery_count'] == 2
    assert split_brain['checkpoint_after']['owner'] == split_brain['takeover_subscriber']


def test_checked_in_continuation_scorecard_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json').read_text())

    assert report['status'] == 'local-continuation-scorecard'
    assert report['summary']['latest_run_id'] == '20260316T140138Z'
    assert report['summary']['all_executor_tracks_have_repeated_recent_coverage'] is True
    assert report['shared_queue_companion']['cross_node_completions'] == 99
    assert report['shared_queue_companion']['duplicate_completed_tasks'] == 0
    assert report['shared_queue_companion']['mode'] == 'bundle-companion-summary'
    assert report['executor_lanes'][0]['lane'] == 'local'
