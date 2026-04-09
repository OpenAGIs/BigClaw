#!/usr/bin/env python3
import importlib.util
import unittest
from importlib.machinery import SourceFileLoader
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().with_name('multi_node_shared_queue.py')
SPEC = importlib.util.spec_from_loader(
    'multi_node_shared_queue',
    SourceFileLoader('multi_node_shared_queue', str(MODULE_PATH)),
)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError(f'Unable to load module from {MODULE_PATH}')
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class MultiNodeSharedQueueTest(unittest.TestCase):
    def test_build_live_takeover_report_summarizes_schema_parity(self) -> None:
        scenarios = [
            {
                'id': 'scenario-a',
                'all_assertions_passed': True,
                'duplicate_delivery_count': 1,
                'stale_write_rejections': 1,
            },
            {
                'id': 'scenario-b',
                'all_assertions_passed': False,
                'duplicate_delivery_count': 2,
                'stale_write_rejections': 0,
            },
        ]

        report = MODULE.build_live_takeover_report(
            scenarios,
            'docs/reports/multi-node-shared-queue-report.json',
        )

        self.assertEqual(report['ticket'], 'OPE-260')
        self.assertEqual(report['status'], 'live-multi-node-proof')
        self.assertEqual(report['summary']['scenario_count'], 2)
        self.assertEqual(report['summary']['passing_scenarios'], 1)
        self.assertEqual(report['summary']['failing_scenarios'], 1)
        self.assertEqual(report['summary']['duplicate_delivery_count'], 3)
        self.assertEqual(report['summary']['stale_write_rejections'], 1)
        self.assertIn('required_report_sections', report)
        self.assertIn('remaining_gaps', report)
        self.assertEqual(
            report['current_primitives']['shared_queue_evidence'][1],
            'docs/reports/multi-node-shared-queue-report.json',
        )


if __name__ == '__main__':
    unittest.main()
