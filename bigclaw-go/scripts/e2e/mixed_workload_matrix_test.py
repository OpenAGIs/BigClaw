#!/usr/bin/env python3
import pathlib
import sys
import tempfile
import unittest

SCRIPT_DIR = pathlib.Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

import mixed_workload_matrix


class MixedWorkloadMatrixBundleTest(unittest.TestCase):
    def test_materialize_bundle_writes_summary_and_task_drilldowns(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            root = pathlib.Path(tmpdir)
            state_dir = root / 'state'
            state_dir.mkdir()
            (state_dir / 'audit.jsonl').write_text('{"event":"audit"}\n', encoding='utf-8')
            service_log = root / 'service.log'
            service_log.write_text('service log\n', encoding='utf-8')

            report = {
                'generated_at': '2026-03-16T02:00:00Z',
                'run_id': '20260316T020000Z',
                'latest_report_path': 'docs/reports/mixed-workload-matrix-report.json',
                'all_ok': True,
                'tasks': [
                    {
                        'name': 'browser-auto',
                        'task_id': 'task-1',
                        'trace_id': 'trace-1',
                        'expectation': {
                            'executor': 'kubernetes',
                            'label': 'browser tool route',
                            'rule': 'browser-required workloads should route to kubernetes',
                        },
                        'observed': {
                            'routed_executor': 'kubernetes',
                            'routed_reason': 'browser workloads default to kubernetes executor',
                            'final_state': 'succeeded',
                            'latest_event_type': 'task.completed',
                        },
                        'event_types': ['task.queued', 'scheduler.routed', 'task.completed'],
                        'event_count': 3,
                        'ok': True,
                        'task': {'id': 'task-1', 'trace_id': 'trace-1'},
                        'events': [{'type': 'task.completed'}],
                    }
                ],
            }

            materialized = mixed_workload_matrix.materialize_bundle(
                str(root),
                'docs/reports/mixed-workload-runs',
                report['run_id'],
                report,
                str(state_dir),
                str(service_log),
            )

            summary_path = root / 'docs/reports/mixed-workload-runs/20260316T020000Z/summary.json'
            task_path = root / 'docs/reports/mixed-workload-runs/20260316T020000Z/tasks/browser-auto.json'
            readme_path = root / 'docs/reports/mixed-workload-runs/20260316T020000Z/README.md'
            self.assertTrue(summary_path.exists())
            self.assertTrue(task_path.exists())
            self.assertTrue(readme_path.exists())
            self.assertEqual(
                materialized['tasks'][0]['drilldown']['json_path'],
                'docs/reports/mixed-workload-runs/20260316T020000Z/tasks/browser-auto.json',
            )
            self.assertNotIn('task', materialized['tasks'][0])
            self.assertNotIn('events', materialized['tasks'][0])
            self.assertEqual(
                materialized['bundle']['audit_log_path'],
                'docs/reports/mixed-workload-runs/20260316T020000Z/audit.jsonl',
            )
            self.assertEqual(
                materialized['bundle']['service_log_path'],
                'docs/reports/mixed-workload-runs/20260316T020000Z/service.log',
            )


if __name__ == '__main__':
    unittest.main()
