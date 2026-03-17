#!/usr/bin/env python3
import importlib.util
import json
import tempfile
import unittest
import unittest.mock
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().with_name('validation_bundle_continuation_policy_gate.py')
SPEC = importlib.util.spec_from_file_location('validation_bundle_continuation_policy_gate', MODULE_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError(f'Unable to load module from {MODULE_PATH}')
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ValidationBundleContinuationPolicyGateTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmpdir = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmpdir.cleanup)
        self.repo_root = Path(self.tmpdir.name)
        reports_dir = self.repo_root / 'docs' / 'reports'
        reports_dir.mkdir(parents=True, exist_ok=True)
        self.scorecard_path = reports_dir / 'validation-bundle-continuation-scorecard.json'

    def write_scorecard(
        self,
        *,
        latest_bundle_age_hours: float = 1.0,
        recent_bundle_count: int = 3,
        latest_all_executor_tracks_succeeded: bool = True,
        recent_bundle_chain_has_no_failures: bool = True,
        repeated_lane_coverage: bool = True,
        shared_queue_available: bool = True,
    ) -> None:
        payload = {
            'summary': {
                'latest_run_id': '20260316T140138Z',
                'latest_bundle_age_hours': latest_bundle_age_hours,
                'recent_bundle_count': recent_bundle_count,
                'latest_all_executor_tracks_succeeded': latest_all_executor_tracks_succeeded,
                'recent_bundle_chain_has_no_failures': recent_bundle_chain_has_no_failures,
                'all_executor_tracks_have_repeated_recent_coverage': repeated_lane_coverage,
            },
            'shared_queue_companion': {
                'available': shared_queue_available,
                'cross_node_completions': 99,
                'duplicate_completed_tasks': 0,
                'duplicate_started_tasks': 0,
                'mode': 'standalone-proof',
            },
        }
        self.scorecard_path.write_text(json.dumps(payload), encoding='utf-8')

    def build_report(self) -> dict:
        fake_script_path = self.repo_root / 'bigclaw-go' / 'scripts' / 'e2e' / 'validation_bundle_continuation_policy_gate.py'
        fake_script_path.parent.mkdir(parents=True, exist_ok=True)
        fake_script_path.write_text('# test shim\n', encoding='utf-8')
        with unittest.mock.patch.object(MODULE, '__file__', str(fake_script_path)):
            return MODULE.build_report(
                scorecard_path='docs/reports/validation-bundle-continuation-scorecard.json',
            )

    def test_build_report_returns_policy_go_when_inputs_pass(self) -> None:
        self.write_scorecard()

        report = self.build_report()

        self.assertEqual(report['ticket'], 'OPE-262')
        self.assertEqual(report['status'], 'policy-go')
        self.assertEqual(report['recommendation'], 'go')
        self.assertEqual(report['enforcement']['mode'], 'hold')
        self.assertEqual(report['enforcement']['outcome'], 'pass')
        self.assertEqual(report['failing_checks'], [])
        self.assertEqual(report['reviewer_path']['index_path'], 'docs/reports/live-validation-index.md')
        self.assertIn(
            'BIGCLAW_E2E_CONTINUATION_GATE_MODE=review',
            report['next_actions'][0],
        )

    def test_build_report_returns_policy_hold_with_actionable_failures(self) -> None:
        self.write_scorecard(
            latest_bundle_age_hours=96.0,
            recent_bundle_count=1,
            repeated_lane_coverage=False,
            shared_queue_available=False,
        )

        report = self.build_report()

        self.assertEqual(report['status'], 'policy-hold')
        self.assertEqual(report['recommendation'], 'hold')
        self.assertIn('latest_bundle_age_within_threshold', report['failing_checks'])
        self.assertIn('recent_bundle_count_meets_floor', report['failing_checks'])
        self.assertIn('shared_queue_companion_available', report['failing_checks'])
        self.assertIn('repeated_lane_coverage_meets_policy', report['failing_checks'])
        self.assertTrue(
            any('./scripts/e2e/run_all.sh' in action for action in report['next_actions'])
        )

    def test_build_report_supports_hold_mode_without_failing_generation(self) -> None:
        self.write_scorecard(recent_bundle_count=1)

        fake_script_path = self.repo_root / 'bigclaw-go' / 'scripts' / 'e2e' / 'validation_bundle_continuation_policy_gate.py'
        fake_script_path.parent.mkdir(parents=True, exist_ok=True)
        fake_script_path.write_text('# test shim\n', encoding='utf-8')
        with unittest.mock.patch.object(MODULE, '__file__', str(fake_script_path)):
            report = MODULE.build_report(
                scorecard_path='docs/reports/validation-bundle-continuation-scorecard.json',
                enforcement_mode='hold',
            )

        self.assertEqual(report['recommendation'], 'hold')
        self.assertEqual(report['enforcement']['mode'], 'hold')
        self.assertEqual(report['enforcement']['outcome'], 'hold')
        self.assertEqual(report['enforcement']['exit_code'], 2)

    def test_legacy_enforce_flag_maps_to_fail_mode(self) -> None:
        self.write_scorecard(recent_bundle_count=1)

        fake_script_path = self.repo_root / 'bigclaw-go' / 'scripts' / 'e2e' / 'validation_bundle_continuation_policy_gate.py'
        fake_script_path.parent.mkdir(parents=True, exist_ok=True)
        fake_script_path.write_text('# test shim\n', encoding='utf-8')
        with unittest.mock.patch.object(MODULE, '__file__', str(fake_script_path)):
            report = MODULE.build_report(
                scorecard_path='docs/reports/validation-bundle-continuation-scorecard.json',
                legacy_enforce_continuation_gate=True,
            )

        self.assertEqual(report['enforcement']['mode'], 'fail')
        self.assertEqual(report['enforcement']['outcome'], 'fail')
        self.assertEqual(report['enforcement']['exit_code'], 1)


if __name__ == '__main__':
    unittest.main()
