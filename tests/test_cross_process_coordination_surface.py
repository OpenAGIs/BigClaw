import importlib.util
import json
import sys
from pathlib import Path


def load_surface_module():
    script_path = Path('bigclaw-go/scripts/e2e/cross_process_coordination_surface.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('cross_process_coordination_surface', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


surface = load_surface_module()


def test_coordination_surface_summarizes_current_live_and_local_proofs() -> None:
    report = surface.build_report()

    assert report['status'] == 'local-capability-surface'
    assert report['runtime_readiness_levels']['live_proven'].startswith('Shipped runtime behavior')
    assert report['summary']['shared_queue_cross_node_completions'] == 99
    assert report['summary']['takeover_passing_scenarios'] == 3
    assert 'no partitioned topic model' in report['current_ceiling']


def test_coordination_surface_marks_partitioned_and_broker_models_unavailable() -> None:
    report = surface.build_report()
    by_capability = {item['capability']: item for item in report['capabilities']}

    assert by_capability['partitioned_topic_routing']['current_state'] == 'not_available'
    assert by_capability['partitioned_topic_routing']['runtime_readiness'] == 'contract_only'
    assert by_capability['broker_backed_subscriber_ownership']['current_state'] == 'not_available'
    assert by_capability['broker_backed_subscriber_ownership']['runtime_readiness'] == 'contract_only'
    assert by_capability['shared_queue_task_coordination']['runtime_readiness'] == 'live_proven'
    assert by_capability['subscriber_takeover_semantics']['deterministic_local_harness'] is True
    assert by_capability['subscriber_takeover_semantics']['runtime_readiness'] == 'live_proven'


def test_checked_in_coordination_surface_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json').read_text())

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
