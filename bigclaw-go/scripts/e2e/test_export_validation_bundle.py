import json
import subprocess
import tempfile
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).with_name('export_validation_bundle.py')


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload) + '\n', encoding='utf-8')


def report_payload(name: str) -> dict:
    return {
        'base_url': f'http://127.0.0.1/{name}',
        'task': {
            'id': f'{name}-task',
            'trace_id': f'{name}-trace',
            'state': 'succeeded',
        },
        'status': {
            'state': 'succeeded',
            'latest_event': {
                'type': 'task.completed',
                'payload': {
                    'artifacts': [f'{name}://artifact/1'],
                },
            },
        },
        'state_dir': '/tmp/should-not-leak',
        'service_log': '/tmp/should-not-leak/service.log',
    }


class ExportValidationBundleTest(unittest.TestCase):
    def test_cli_emits_stable_bundle_contract(self) -> None:
        with tempfile.TemporaryDirectory() as tmp_dir:
            root = Path(tmp_dir)
            bundle_dir = root / 'docs/reports/live-validation-runs/20260316T000000Z'
            bundle_dir.mkdir(parents=True, exist_ok=True)

            previous_bundle = root / 'docs/reports/live-validation-runs/20260315T000000Z'
            write_json(
                previous_bundle / 'summary.json',
                {
                    'run_id': '20260315T000000Z',
                    'generated_at': '2026-03-15T00:00:00+00:00',
                    'status': 'succeeded',
                    'bundle_path': 'docs/reports/live-validation-runs/20260315T000000Z',
                    'summary_path': 'docs/reports/live-validation-runs/20260315T000000Z/summary.json',
                },
            )

            local_report = bundle_dir / 'sqlite-smoke-report.json'
            k8s_report = bundle_dir / 'kubernetes-live-smoke-report.json'
            ray_report = bundle_dir / 'ray-live-smoke-report.json'
            write_json(local_report, report_payload('local'))
            write_json(k8s_report, report_payload('kubernetes'))
            write_json(ray_report, report_payload('ray'))

            local_out = root / 'tmp/local.out'
            local_err = root / 'tmp/local.err'
            k8s_out = root / 'tmp/k8s.out'
            k8s_err = root / 'tmp/k8s.err'
            ray_out = root / 'tmp/ray.out'
            ray_err = root / 'tmp/ray.err'
            for path in (local_out, local_err, k8s_out, k8s_err, ray_out, ray_err):
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(f'{path.name}\n', encoding='utf-8')

            completed = subprocess.run(
                [
                    'python3',
                    str(SCRIPT_PATH),
                    '--go-root',
                    str(root),
                    '--run-id',
                    '20260316T000000Z',
                    '--bundle-dir',
                    'docs/reports/live-validation-runs/20260316T000000Z',
                    '--validation-status',
                    '0',
                    '--local-report-path',
                    str(local_report.relative_to(root)),
                    '--local-stdout-path',
                    str(local_out),
                    '--local-stderr-path',
                    str(local_err),
                    '--kubernetes-report-path',
                    str(k8s_report.relative_to(root)),
                    '--kubernetes-stdout-path',
                    str(k8s_out),
                    '--kubernetes-stderr-path',
                    str(k8s_err),
                    '--ray-report-path',
                    str(ray_report.relative_to(root)),
                    '--ray-stdout-path',
                    str(ray_out),
                    '--ray-stderr-path',
                    str(ray_err),
                ],
                check=True,
                capture_output=True,
                text=True,
            )

            summary = json.loads((root / 'docs/reports/live-validation-summary.json').read_text(encoding='utf-8'))
            manifest = json.loads((root / 'docs/reports/live-validation-index.json').read_text(encoding='utf-8'))
            index_text = (root / 'docs/reports/live-validation-index.md').read_text(encoding='utf-8')

            self.assertEqual(summary['schema_version'], 'live_validation_bundle/v1')
            self.assertEqual(summary['bundle']['run_id'], '20260316T000000Z')
            self.assertEqual(
                summary['bundle']['summary_path'],
                'docs/reports/live-validation-runs/20260316T000000Z/summary.json',
            )
            self.assertEqual(
                summary['components']['local']['artifacts']['bundle_report_path'],
                'docs/reports/live-validation-runs/20260316T000000Z/sqlite-smoke-report.json',
            )
            self.assertEqual(summary['components']['local']['task']['id'], 'local-task')
            self.assertEqual(summary['components']['local']['report_summary']['latest_event_type'], 'task.completed')
            self.assertEqual(summary['components']['local']['execution_artifacts'], ['local://artifact/1'])
            self.assertNotIn('/tmp/should-not-leak', json.dumps(summary))

            self.assertEqual(manifest['schema_version'], 'live_validation_bundle/v1')
            self.assertEqual(manifest['latest']['bundle']['readme_path'], 'docs/reports/live-validation-runs/20260316T000000Z/README.md')
            self.assertEqual(manifest['recent_bundles'][0]['run_id'], '20260316T000000Z')
            self.assertEqual(manifest['recent_bundles'][1]['run_id'], '20260315T000000Z')
            self.assertIn('- Schema: `live_validation_bundle/v1`', index_text)
            self.assertIn('- Bundle README: `docs/reports/live-validation-runs/20260316T000000Z/README.md`', index_text)
            self.assertIn('local://artifact/1', index_text)
            self.assertIn('"schema_version": "live_validation_bundle/v1"', completed.stdout)


if __name__ == '__main__':
    unittest.main()
