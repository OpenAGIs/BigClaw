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


def test_policy_gate_holds_when_repeated_lane_coverage_is_missing() -> None:
    report = gate.build_report()

    assert report['status'] == 'policy-hold'
    assert report['recommendation'] == 'hold'
    assert 'repeated_lane_coverage_meets_policy' in report['failing_checks']
    assert report['summary']['failing_check_count'] == 1


def test_policy_gate_can_allow_partial_lane_history() -> None:
    report = gate.build_report(require_repeated_lane_coverage=False)

    assert report['status'] == 'policy-go'
    assert report['recommendation'] == 'go'
    assert report['failing_checks'] == []


def test_checked_in_policy_gate_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json').read_text())

    assert report['status'] == 'policy-hold'
    assert report['recommendation'] == 'hold'
    assert report['summary']['latest_run_id'] == '20260314T164647Z'
    assert 'repeated_lane_coverage_meets_policy' in report['failing_checks']


def test_policy_gate_cli_returns_nonzero_for_hold() -> None:
    script = Path('bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py')
    result = subprocess.run([sys.executable, str(script)], check=False, capture_output=True, text=True)

    assert result.returncode == 1
