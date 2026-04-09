#!/usr/bin/env python3
import importlib.util
import unittest
from importlib.machinery import SourceFileLoader
from pathlib import Path


MODULE_PATH = Path(__file__).resolve().with_name('broker_failover_stub_matrix.py')
SPEC = importlib.util.spec_from_loader(
    'broker_failover_stub_matrix',
    SourceFileLoader('broker_failover_stub_matrix', str(MODULE_PATH)),
)
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
        self.assertEqual(
            report['proof_artifacts'],
            {
                'checkpoint_fencing_summary': 'bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json',
                'retention_boundary_summary': 'bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json',
            },
        )

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

    def test_proof_summaries_project_scenario_evidence_into_rollout_gate_statuses(self) -> None:
        report = MODULE.build_report()

        checkpoint = MODULE.build_checkpoint_fencing_summary(report)
        self.assertEqual(checkpoint['ticket'], 'OPE-230')
        self.assertEqual(checkpoint['proof_family'], 'checkpoint_fencing')
        checkpoint_gates = {gate['name']: gate for gate in checkpoint['rollout_gate_statuses']}
        self.assertEqual(checkpoint_gates['replay_checkpoint_alignment']['status'], 'passed')
        self.assertEqual(checkpoint_gates['retention_boundary_visibility']['status'], 'unknown')
        self.assertEqual(checkpoint['summary']['stale_write_rejections'], 1)

        retention = MODULE.build_retention_boundary_summary(report)
        self.assertEqual(retention['ticket'], 'OPE-230')
        self.assertEqual(retention['proof_family'], 'retention_boundary')
        retention_gates = {gate['name']: gate for gate in retention['rollout_gate_statuses']}
        self.assertEqual(retention_gates['retention_boundary_visibility']['status'], 'passed')
        self.assertTrue(retention['focus_scenarios'][0]['reset_required'])
        self.assertEqual(retention['summary']['retention_floor'], 3)


if __name__ == '__main__':
    unittest.main()
