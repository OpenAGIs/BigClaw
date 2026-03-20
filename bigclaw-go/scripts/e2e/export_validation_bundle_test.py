#!/usr/bin/env python3
import importlib.util
import tempfile
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().with_name('export_validation_bundle.py')
SPEC = importlib.util.spec_from_file_location('export_validation_bundle', MODULE_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError(f'Unable to load module from {MODULE_PATH}')
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ExportValidationBundleTest(unittest.TestCase):
    def test_render_index_surfaces_continuation_workflow_mode_and_outcome(self) -> None:
        summary = {
            'run_id': '20260316T140138Z',
            'generated_at': '2026-03-16T14:48:42.581505+00:00',
            'status': 'succeeded',
            'bundle_path': 'docs/reports/live-validation-runs/20260316T140138Z',
            'summary_path': 'docs/reports/live-validation-runs/20260316T140138Z/summary.json',
            'closeout_commands': ['cd bigclaw-go && ./scripts/e2e/run_all.sh'],
            'local': {
                'enabled': True,
                'status': 'succeeded',
                'bundle_report_path': 'docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json',
                'canonical_report_path': 'docs/reports/sqlite-smoke-report.json',
            },
            'kubernetes': {
                'enabled': False,
                'status': 'skipped',
                'bundle_report_path': 'docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json',
                'canonical_report_path': 'docs/reports/kubernetes-live-smoke-report.json',
            },
            'ray': {
                'enabled': False,
                'status': 'skipped',
                'bundle_report_path': 'docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json',
                'canonical_report_path': 'docs/reports/ray-live-smoke-report.json',
            },
            'broker': {
                'enabled': False,
                'status': 'skipped',
                'configuration_state': 'not_configured',
                'reason': 'not_configured',
                'bundle_summary_path': 'docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json',
                'canonical_summary_path': 'docs/reports/broker-validation-summary.json',
                'bundle_bootstrap_summary_path': 'docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json',
                'canonical_bootstrap_summary_path': 'docs/reports/broker-bootstrap-review-summary.json',
                'validation_pack_path': 'docs/reports/broker-failover-fault-injection-validation-pack.md',
                'backend': None,
                'bootstrap_ready': False,
                'runtime_posture': 'contract_only',
                'live_adapter_implemented': False,
                'proof_boundary': 'broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof',
                'config_completeness': {
                    'driver': False,
                    'urls': False,
                    'topic': False,
                    'consumer_group': False,
                },
                'validation_errors': ['broker event log config missing driver, urls, topic'],
            },
        }
        continuation_gate = {
            'path': 'docs/reports/validation-bundle-continuation-policy-gate.json',
            'status': 'policy-hold',
            'recommendation': 'hold',
            'enforcement': {'mode': 'hold', 'outcome': 'hold', 'exit_code': 2},
            'summary': {'latest_run_id': '20260316T140138Z', 'failing_check_count': 2, 'workflow_exit_code': 2},
            'reviewer_path': {'digest_path': 'docs/reports/validation-bundle-continuation-digest.md'},
            'next_actions': ['rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle'],
        }

        index_text = MODULE.render_index(summary, [], continuation_gate, [], [])

        self.assertIn('- Workflow mode: `hold`', index_text)
        self.assertIn('- Workflow outcome: `hold`', index_text)
        self.assertIn('- Workflow exit code on current evidence: `2`', index_text)
        self.assertIn('### broker', index_text)
        self.assertIn('- Configuration state: `not_configured`', index_text)
        self.assertIn('- Runtime posture: `contract_only`', index_text)
        self.assertIn('- Live adapter implemented: `False`', index_text)
        self.assertIn('- Validation error: `broker event log config missing driver, urls, topic`', index_text)
        self.assertIn('- Reason: `not_configured`', index_text)

    def test_build_broker_section_writes_not_configured_summary_when_disabled(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            bundle_dir = tmp_root / 'docs' / 'reports' / 'live-validation-runs' / 'run-1'
            bundle_dir.mkdir(parents=True, exist_ok=True)
            bootstrap_summary_path = tmp_root / 'docs' / 'reports' / 'broker-bootstrap-review-summary-source.json'
            bootstrap_summary_path.parent.mkdir(parents=True, exist_ok=True)
            bootstrap_summary_path.write_text(
                '{"ready": false, "runtime_posture": "contract_only", "live_adapter_implemented": false, '
                '"proof_boundary": "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof", '
                '"config_completeness": {"driver": false, "urls": false, "topic": false, "consumer_group": false}, '
                '"validation_errors": ["broker event log config missing driver, urls, topic"]}',
                encoding='utf-8',
            )

            section = MODULE.build_broker_section(
                enabled=False,
                backend='',
                root=tmp_root,
                bundle_dir=bundle_dir,
                bootstrap_summary_path=bootstrap_summary_path,
                report_path=None,
            )

            self.assertEqual(section['status'], 'skipped')
            self.assertEqual(section['configuration_state'], 'not_configured')
            self.assertEqual(section['reason'], 'not_configured')
            self.assertEqual(
                section['bundle_summary_path'],
                'docs/reports/live-validation-runs/run-1/broker-validation-summary.json',
            )
            self.assertEqual(section['runtime_posture'], 'contract_only')
            self.assertFalse(section['live_adapter_implemented'])
            self.assertFalse(section['config_completeness']['driver'])
            self.assertTrue((bundle_dir / 'broker-validation-summary.json').exists())
            self.assertTrue((tmp_root / 'docs/reports/broker-validation-summary.json').exists())
            self.assertTrue((bundle_dir / 'broker-bootstrap-review-summary.json').exists())
            self.assertTrue((tmp_root / 'docs/reports/broker-bootstrap-review-summary.json').exists())


if __name__ == '__main__':
    unittest.main()
