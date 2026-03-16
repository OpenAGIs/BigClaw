import json
import tempfile
import unittest
from pathlib import Path

from scripts.migration import shadow_compare


def sample_comparison(trace_id='shadow-sample', matched=True):
    primary_events = [
        {'type': 'task.queued', 'timestamp': '2026-03-16T00:00:00+00:00'},
        {'type': 'task.completed', 'timestamp': '2026-03-16T00:00:01+00:00'},
    ]
    shadow_events = list(primary_events if matched else primary_events[:-1] + [{'type': 'task.failed', 'timestamp': '2026-03-16T00:00:01+00:00'}])
    return {
        'trace_id': trace_id,
        'primary': {
            'task_id': f'{trace_id}-primary',
            'status': {'state': 'succeeded'},
            'events': primary_events,
        },
        'shadow': {
            'task_id': f'{trace_id}-shadow',
            'status': {'state': 'succeeded' if matched else 'failed'},
            'events': shadow_events,
        },
        'diff': {
            'state_equal': matched,
            'event_count_delta': len(primary_events) - len(shadow_events),
            'event_types_equal': matched,
            'primary_event_types': [event['type'] for event in primary_events],
            'shadow_event_types': [event['type'] for event in shadow_events],
            'primary_timeline_seconds': 1,
            'shadow_timeline_seconds': 1,
        },
    }


class ShadowReportTests(unittest.TestCase):
    def test_build_bundle_report_normalizes_single_compare(self):
        comparison = sample_comparison()
        report = shadow_compare.build_bundle_report(
            report_kind='shadow_compare',
            comparisons=[comparison],
            task_sources=['examples/shadow-task.json'],
            legacy_single_result=comparison,
        )

        self.assertEqual(report['artifact_type'], 'shadow_compare')
        self.assertEqual(report['status'], 'matched')
        self.assertEqual(report['comparison_totals'], {'total': 1, 'matched': 1, 'mismatched': 0})
        self.assertEqual(report['task_sources'], ['examples/shadow-task.json'])
        self.assertEqual(report['comparison']['trace_id'], 'shadow-sample')
        self.assertEqual(report['comparison_summaries'][0]['matched'], True)
        self.assertEqual(report['trace_id'], 'shadow-sample')

    def test_write_bundle_artifacts_writes_canonical_report_and_summary(self):
        report = shadow_compare.build_bundle_report(
            report_kind='shadow_matrix',
            comparisons=[sample_comparison('matched'), sample_comparison('mismatched', matched=False)],
            task_sources=['examples/a.json', 'examples/b.json'],
        )

        with tempfile.TemporaryDirectory() as tempdir:
            root = Path(tempdir)
            scripts_dir = root / 'scripts' / 'migration'
            scripts_dir.mkdir(parents=True)
            script_path = scripts_dir / 'shadow_compare.py'
            script_path.write_text('# test shim\n', encoding='utf-8')
            reports_dir = root / 'docs' / 'reports'
            report_path = reports_dir / 'shadow-matrix-report.json'
            bundle_dir = reports_dir / 'migration-shadow-bundle'

            original_file = shadow_compare.__file__
            shadow_compare.__file__ = str(script_path)
            try:
                shadow_compare.write_bundle_artifacts(
                    report,
                    report_path=report_path,
                    bundle_dir=bundle_dir,
                    bundle_summary_name='shadow-matrix-summary.json',
                )
            finally:
                shadow_compare.__file__ = original_file

            canonical = json.loads(report_path.read_text(encoding='utf-8'))
            bundle_report = json.loads((bundle_dir / 'shadow-matrix-report.json').read_text(encoding='utf-8'))
            summary = json.loads((bundle_dir / 'shadow-matrix-summary.json').read_text(encoding='utf-8'))

            self.assertEqual(canonical['bundle_path'], 'docs/reports/migration-shadow-bundle')
            self.assertEqual(canonical['summary_path'], 'docs/reports/migration-shadow-bundle/shadow-matrix-summary.json')
            self.assertEqual(canonical['bundle_report_path'], 'docs/reports/migration-shadow-bundle/shadow-matrix-report.json')
            self.assertEqual(bundle_report['comparison_totals']['mismatched'], 1)
            self.assertEqual(summary['comparison_totals']['matched'], 1)
            self.assertEqual(summary['task_sources'], ['examples/a.json', 'examples/b.json'])
            self.assertNotIn('comparisons', summary)


if __name__ == '__main__':
    unittest.main()
