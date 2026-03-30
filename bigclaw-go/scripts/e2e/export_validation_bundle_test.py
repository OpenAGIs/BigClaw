#!/usr/bin/env python3
import importlib.util
import json
import subprocess
import sys
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
    def write_json(self, path: Path, payload: dict) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')

    def test_export_validation_bundle_generates_latest_reports_and_index(self) -> None:
        with tempfile.TemporaryDirectory() as tmpdir:
            tmp_root = Path(tmpdir)
            root = tmp_root / 'repo'
            bundle = root / 'docs' / 'reports' / 'live-validation-runs' / '20260315T120000Z'
            state_dir = tmp_root / 'state-local'
            state_dir.mkdir(parents=True)
            (state_dir / 'audit.jsonl').write_text('{"event":"ok"}\n', encoding='utf-8')
            service_log = tmp_root / 'local-service.log'
            service_log.write_text('local service ready\n', encoding='utf-8')

            local_report = {
                'base_url': 'http://127.0.0.1:19090',
                'task': {'id': 'local-1'},
                'status': {'state': 'succeeded'},
                'state_dir': str(state_dir),
                'service_log': str(service_log),
            }
            self.write_json(bundle / 'sqlite-smoke-report.json', local_report)
            self.write_json(bundle / 'kubernetes-live-smoke-report.json', {'task': {'id': 'k8s-1'}, 'status': {'state': 'succeeded'}})
            self.write_json(bundle / 'ray-live-smoke-report.json', {'task': {'id': 'ray-1'}, 'status': {'state': 'succeeded'}})

            (root / 'docs' / 'reports').mkdir(parents=True, exist_ok=True)
            self.write_json(
                root / 'docs' / 'reports' / 'multi-node-shared-queue-report.json',
                {
                    'generated_at': '2026-03-15T11:55:00Z',
                    'count': 12,
                    'submitted_by_node': {'node-a': 6, 'node-b': 6},
                    'completed_by_node': {'node-a': 5, 'node-b': 7},
                    'cross_node_completions': 4,
                    'duplicate_started_tasks': [],
                    'duplicate_completed_tasks': [],
                    'missing_completed_tasks': [],
                    'all_ok': True,
                    'nodes': [{'name': 'node-a'}, {'name': 'node-b'}],
                },
            )
            (root / 'docs' / 'reports' / 'validation-bundle-continuation-scorecard.json').write_text('{}\n', encoding='utf-8')
            self.write_json(
                root / 'docs' / 'reports' / 'validation-bundle-continuation-policy-gate.json',
                {
                    'status': 'policy-hold',
                    'recommendation': 'hold',
                    'failing_checks': ['latest_bundle_all_executor_tracks_succeeded'],
                    'enforcement': {'mode': 'review'},
                    'summary': {'latest_bundle_age_hours': 12.5},
                    'reviewer_path': {'digest': 'docs/reports/validation-bundle-continuation-digest.md'},
                    'next_actions': ['rerun ./scripts/e2e/run_all.sh'],
                },
            )
            (root / 'docs' / 'reports' / 'validation-bundle-continuation-digest.md').write_text('# digest\n', encoding='utf-8')

            local_stdout = tmp_root / 'local.stdout'
            local_stderr = tmp_root / 'local.stderr'
            k8s_stdout = tmp_root / 'k8s.stdout'
            k8s_stderr = tmp_root / 'k8s.stderr'
            ray_stdout = tmp_root / 'ray.stdout'
            ray_stderr = tmp_root / 'ray.stderr'
            for path, content in [
                (local_stdout, 'local ok\n'),
                (local_stderr, ''),
                (k8s_stdout, 'k8s ok\n'),
                (k8s_stderr, ''),
                (ray_stdout, 'ray ok\n'),
                (ray_stderr, ''),
            ]:
                path.write_text(content, encoding='utf-8')

            result = subprocess.run(
                [
                    sys.executable,
                    str(MODULE_PATH),
                    '--go-root', str(root),
                    '--run-id', '20260315T120000Z',
                    '--bundle-dir', 'docs/reports/live-validation-runs/20260315T120000Z',
                    '--summary-path', 'docs/reports/live-validation-summary.json',
                    '--index-path', 'docs/reports/live-validation-index.md',
                    '--manifest-path', 'docs/reports/live-validation-index.json',
                    '--run-local', '1',
                    '--run-kubernetes', '1',
                    '--run-ray', '1',
                    '--validation-status', '0',
                    '--local-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/sqlite-smoke-report.json',
                    '--local-stdout-path', str(local_stdout),
                    '--local-stderr-path', str(local_stderr),
                    '--kubernetes-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/kubernetes-live-smoke-report.json',
                    '--kubernetes-stdout-path', str(k8s_stdout),
                    '--kubernetes-stderr-path', str(k8s_stderr),
                    '--ray-report-path', 'docs/reports/live-validation-runs/20260315T120000Z/ray-live-smoke-report.json',
                    '--ray-stdout-path', str(ray_stdout),
                    '--ray-stderr-path', str(ray_stderr),
                ],
                check=False,
                capture_output=True,
                text=True,
            )

            self.assertEqual(result.returncode, 0, result.stderr)

            summary = json.loads((root / 'docs' / 'reports' / 'live-validation-summary.json').read_text(encoding='utf-8'))
            self.assertEqual(summary['status'], 'succeeded')
            self.assertEqual(summary['local']['canonical_report_path'], 'docs/reports/sqlite-smoke-report.json')
            self.assertTrue(summary['local']['audit_log_path'].endswith('local.audit.jsonl'))
            self.assertTrue(summary['local']['service_log_path'].endswith('local.service.log'))
            self.assertEqual(summary['shared_queue_companion']['status'], 'succeeded')
            self.assertEqual(summary['shared_queue_companion']['cross_node_completions'], 4)
            self.assertEqual(summary['shared_queue_companion']['canonical_summary_path'], 'docs/reports/shared-queue-companion-summary.json')
            self.assertTrue(summary['shared_queue_companion']['bundle_summary_path'].endswith('shared-queue-companion-summary.json'))
            self.assertEqual(summary['continuation_gate']['status'], 'policy-hold')
            self.assertEqual(summary['continuation_gate']['recommendation'], 'hold')
            self.assertEqual(summary['continuation_gate']['summary']['latest_bundle_age_hours'], 12.5)
            self.assertEqual(summary['continuation_gate']['failing_checks'], ['latest_bundle_all_executor_tracks_succeeded'])
            self.assertEqual(summary['continuation_gate']['next_actions'], ['rerun ./scripts/e2e/run_all.sh'])

            latest_local = json.loads((root / 'docs' / 'reports' / 'sqlite-smoke-report.json').read_text(encoding='utf-8'))
            self.assertEqual(latest_local['task']['id'], 'local-1')
            shared_queue_summary = json.loads((root / 'docs' / 'reports' / 'shared-queue-companion-summary.json').read_text(encoding='utf-8'))
            self.assertEqual(shared_queue_summary['nodes'], ['node-a', 'node-b'])
            self.assertTrue(shared_queue_summary['bundle_report_path'].endswith('multi-node-shared-queue-report.json'))

            index_text = (root / 'docs' / 'reports' / 'live-validation-index.md').read_text(encoding='utf-8')
            for expected in [
                'Live Validation Index',
                '20260315T120000Z',
                'docs/reports/live-validation-runs/20260315T120000Z',
                'shared-queue companion',
                'docs/reports/shared-queue-companion-summary.json',
                'docs/reports/multi-node-shared-queue-report.json',
                'docs/reports/validation-bundle-continuation-scorecard.json',
                'docs/reports/validation-bundle-continuation-policy-gate.json',
                'docs/reports/validation-bundle-continuation-digest.md',
            ]:
                self.assertIn(expected, index_text)

            manifest = json.loads((root / 'docs' / 'reports' / 'live-validation-index.json').read_text(encoding='utf-8'))
            self.assertEqual(manifest['latest']['run_id'], '20260315T120000Z')
            self.assertEqual(manifest['latest']['shared_queue_companion']['nodes'], ['node-a', 'node-b'])
            self.assertEqual(manifest['recent_runs'][0]['run_id'], '20260315T120000Z')

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
