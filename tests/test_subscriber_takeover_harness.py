import importlib.util
import json
import sys
from pathlib import Path


def load_takeover_module():
    script_path = Path('bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('subscriber_takeover_fault_matrix', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


takeover = load_takeover_module()


def test_takeover_harness_report_has_three_passing_scenarios() -> None:
    report = takeover.build_report()

    assert report['status'] == 'local-executable'
    assert report['summary']['scenario_count'] == 3
    assert report['summary']['passing_scenarios'] == 3
    assert report['summary']['failing_scenarios'] == 0
    assert report['summary']['stale_write_rejections'] == 2
    assert report['summary']['duplicate_delivery_count'] == 4


def test_stale_writer_scenario_records_rejection_and_final_owner() -> None:
    report = takeover.build_report()
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
