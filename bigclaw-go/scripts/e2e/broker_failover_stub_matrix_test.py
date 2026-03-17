#!/usr/bin/env python3
import importlib.util
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().with_name('broker_failover_stub_matrix.py')
SPEC = importlib.util.spec_from_file_location('broker_failover_stub_matrix', MODULE_PATH)
if SPEC is None or SPEC.loader is None:
    raise RuntimeError(f'Unable to load module from {MODULE_PATH}')
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class BrokerFailoverStubMatrixTest(unittest.TestCase):
    def test_build_report_covers_all_scenarios_and_schema(self) -> None:
        report = MODULE.build_report()

        self.assertEqual(report['ticket'], 'OPE-272')
        self.assertEqual(report['summary']['scenario_count'], 8)
        self.assertEqual(report['summary']['passing_scenarios'], 8)
        self.assertEqual(report['summary']['failing_scenarios'], 0)

        required_keys = {
            'scenario_id',
            'backend',
            'topology',
            'fault_window',
            'published_count',
            'committed_count',
            'replayed_count',
            'duplicate_count',
            'missing_event_ids',
            'checkpoint_before_fault',
            'checkpoint_after_recovery',
            'lease_transitions',
            'publish_outcomes',
            'replay_resume_cursor',
            'artifacts',
            'result',
            'assertions',
        }
        for scenario in report['scenarios']:
            self.assertTrue(required_keys.issubset(scenario.keys()))
            self.assertEqual(scenario['result'], 'passed')

    def test_checkpoint_and_durable_sequence_assertions_hold_for_fencing_scenarios(self) -> None:
        report = MODULE.build_report()
        scenarios = {scenario['scenario_id']: scenario for scenario in report['scenarios']}

        bf04 = scenarios['BF-04']
        self.assertGreaterEqual(
            bf04['checkpoint_after_recovery']['durable_sequence'],
            bf04['checkpoint_before_fault']['durable_sequence'],
        )
        self.assertTrue(
            any(
                entry['owner_id'] == 'consumer-a' and entry['transition'] == 'fenced'
                for entry in report['raw_artifacts']['BF-04']['checkpoint_transition_log']
            )
        )

        bf08 = scenarios['BF-08']
        self.assertEqual(bf08['duplicate_count'], 1)
        self.assertEqual(bf08['missing_event_ids'], [])


if __name__ == '__main__':
    unittest.main()
