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

    def test_build_component_section_emits_k8s_matrix_and_failure_root_cause(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            bundle_dir = tmp_root / 'docs' / 'reports' / 'live-validation-runs' / 'run-k8s'
            bundle_dir.mkdir(parents=True, exist_ok=True)

            report_path = tmp_root / 'tmp' / 'kubernetes-smoke-report.json'
            report_path.parent.mkdir(parents=True, exist_ok=True)
            report_path.write_text(
                """
{
  "task": {
    "id": "kubernetes-smoke-failed",
    "required_executor": "kubernetes"
  },
  "status": {
    "state": "dead_letter",
    "latest_event": {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {
        "message": "lease lost during replay"
      }
    }
  },
  "events": [
    {
      "id": "evt-routed",
      "type": "scheduler.routed",
      "timestamp": "2026-03-23T11:00:00Z",
      "payload": {
        "reason": "browser workloads default to kubernetes executor"
      }
    },
    {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {
        "message": "lease lost during replay"
      }
    }
  ]
}
                """.strip(),
                encoding='utf-8',
            )
            stdout_path = tmp_root / 'tmp' / 'kubernetes.stdout.log'
            stderr_path = tmp_root / 'tmp' / 'kubernetes.stderr.log'
            stdout_path.write_text('starting kubernetes smoke\n', encoding='utf-8')
            stderr_path.write_text('lease lost during replay\n', encoding='utf-8')

            section = MODULE.build_component_section(
                name='kubernetes',
                enabled=True,
                root=tmp_root,
                bundle_dir=bundle_dir,
                report_path=report_path,
                stdout_path=stdout_path,
                stderr_path=stderr_path,
            )

            self.assertEqual(section['status'], 'dead_letter')
            self.assertEqual(section['validation_matrix']['lane'], 'k8s')
            self.assertEqual(section['validation_matrix']['executor'], 'kubernetes')
            self.assertEqual(section['routing_reason'], 'browser workloads default to kubernetes executor')
            self.assertEqual(section['failure_root_cause']['status'], 'captured')
            self.assertEqual(section['failure_root_cause']['event_type'], 'task.dead_letter')
            self.assertEqual(section['failure_root_cause']['message'], 'lease lost during replay')
            self.assertEqual(
                section['failure_root_cause']['location'],
                'docs/reports/live-validation-runs/run-k8s/kubernetes.stderr.log',
            )

    def test_render_index_surfaces_validation_matrix_and_failure_root_cause(self) -> None:
        summary = {
            'run_id': '20260323T030000Z',
            'generated_at': '2026-03-23T03:10:00+00:00',
            'status': 'failed',
            'bundle_path': 'docs/reports/live-validation-runs/20260323T030000Z',
            'summary_path': 'docs/reports/live-validation-runs/20260323T030000Z/summary.json',
            'closeout_commands': ['cd bigclaw-go && ./scripts/e2e/run_all.sh'],
            'local': {
                'enabled': True,
                'status': 'succeeded',
                'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
                'canonical_report_path': 'docs/reports/sqlite-smoke-report.json',
                'validation_matrix': {
                    'lane': 'local',
                    'executor': 'local',
                    'enabled': True,
                    'status': 'succeeded',
                    'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
                },
                'failure_root_cause': {
                    'status': 'not_triggered',
                    'event_type': 'task.completed',
                    'message': '',
                    'location': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
                },
            },
            'kubernetes': {
                'enabled': True,
                'status': 'dead_letter',
                'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json',
                'canonical_report_path': 'docs/reports/kubernetes-live-smoke-report.json',
                'validation_matrix': {
                    'lane': 'k8s',
                    'executor': 'kubernetes',
                    'enabled': True,
                    'status': 'dead_letter',
                    'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json',
                    'root_cause_event_type': 'task.dead_letter',
                    'root_cause_location': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log',
                    'root_cause_message': 'lease lost during replay',
                },
                'failure_root_cause': {
                    'status': 'captured',
                    'event_type': 'task.dead_letter',
                    'message': 'lease lost during replay',
                    'location': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log',
                },
            },
            'ray': {
                'enabled': True,
                'status': 'succeeded',
                'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
                'canonical_report_path': 'docs/reports/ray-live-smoke-report.json',
                'validation_matrix': {
                    'lane': 'ray',
                    'executor': 'ray',
                    'enabled': True,
                    'status': 'succeeded',
                    'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
                },
                'failure_root_cause': {
                    'status': 'not_triggered',
                    'event_type': 'task.completed',
                    'message': '',
                    'location': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
                },
            },
            'validation_matrix': [
                {
                    'lane': 'local',
                    'executor': 'local',
                    'enabled': True,
                    'status': 'succeeded',
                    'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json',
                },
                {
                    'lane': 'k8s',
                    'executor': 'kubernetes',
                    'enabled': True,
                    'status': 'dead_letter',
                    'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json',
                    'root_cause_event_type': 'task.dead_letter',
                    'root_cause_location': 'docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log',
                    'root_cause_message': 'lease lost during replay',
                },
                {
                    'lane': 'ray',
                    'executor': 'ray',
                    'enabled': True,
                    'status': 'succeeded',
                    'bundle_report_path': 'docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json',
                },
            ],
        }

        index_text = MODULE.render_index(summary, [], None, [], [])

        self.assertIn('## Validation matrix', index_text)
        self.assertIn(
            '- Lane `k8s` executor=`kubernetes` status=`dead_letter` enabled=`True` report=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json`',
            index_text,
        )
        self.assertIn(
            '- Lane `k8s` root cause: event=`task.dead_letter` location=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log` message=`lease lost during replay`',
            index_text,
        )
        self.assertIn(
            '- Failure root cause: status=`captured` event=`task.dead_letter` location=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log`',
            index_text,
        )


if __name__ == '__main__':
    unittest.main()
