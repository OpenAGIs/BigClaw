import json
import subprocess
import sys
from pathlib import Path

BIGCLAW_GO_ROOT = Path(__file__).resolve().parents[1] / 'bigclaw-go'


def write_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def test_export_live_shadow_bundle_generates_index_and_rollup(tmp_path: Path) -> None:
    root = tmp_path / 'go'
    reports = root / 'docs' / 'reports'
    reports.mkdir(parents=True, exist_ok=True)
    (root / 'docs').mkdir(exist_ok=True)
    (root / 'docs' / 'migration-shadow.md').write_text('# shadow\n', encoding='utf-8')
    (reports / 'migration-readiness-report.md').write_text('# readiness\n', encoding='utf-8')
    (reports / 'migration-plan-review-notes.md').write_text('# review\n', encoding='utf-8')
    (reports / 'live-shadow-comparison-follow-up-digest.md').write_text('# digest\n', encoding='utf-8')

    write_json(
        reports / 'shadow-compare-report.json',
        {
            'trace_id': 'compare-1',
            'primary': {'task_id': 'primary-1', 'events': [{'timestamp': '2026-03-10T10:00:00Z'}]},
            'shadow': {'task_id': 'shadow-1', 'events': [{'timestamp': '2026-03-10T10:00:01Z'}]},
            'diff': {
                'state_equal': True,
                'event_types_equal': True,
                'event_count_delta': 0,
                'primary_timeline_seconds': 0.1,
                'shadow_timeline_seconds': 0.15,
            },
        },
    )
    write_json(
        reports / 'shadow-matrix-report.json',
        {
            'total': 2,
            'matched': 2,
            'mismatched': 0,
            'results': [
                {
                    'trace_id': 'matrix-1',
                    'primary': {'task_id': 'matrix-primary-1', 'events': [{'timestamp': '2026-03-10T10:05:00Z'}]},
                    'shadow': {'task_id': 'matrix-shadow-1', 'events': [{'timestamp': '2026-03-10T10:05:01Z'}]},
                    'diff': {
                        'state_equal': True,
                        'event_types_equal': True,
                        'event_count_delta': 0,
                        'primary_timeline_seconds': 0.2,
                        'shadow_timeline_seconds': 0.22,
                    },
                },
                {
                    'trace_id': 'matrix-2',
                    'primary': {'task_id': 'matrix-primary-2', 'events': [{'timestamp': '2026-03-10T10:06:00Z'}]},
                    'shadow': {'task_id': 'matrix-shadow-2', 'events': [{'timestamp': '2026-03-10T10:06:01Z'}]},
                    'diff': {
                        'state_equal': True,
                        'event_types_equal': True,
                        'event_count_delta': 0,
                        'primary_timeline_seconds': 0.2,
                        'shadow_timeline_seconds': 0.23,
                    },
                },
            ],
        },
    )
    write_json(
        reports / 'rollback-trigger-surface.json',
        {
            'summary': {
                'status': 'manual-only',
                'automation_boundary': 'manual',
                'automated_rollback_trigger': False,
                'distinctions': {},
            },
        },
    )
    write_json(
        reports / 'live-shadow-mirror-scorecard.json',
        {
            'summary': {
                'latest_evidence_timestamp': '2026-03-10T10:06:01Z',
                'total_evidence_runs': 3,
                'parity_ok_count': 3,
                'drift_detected_count': 0,
                'matrix_total': 2,
                'matrix_mismatched': 0,
                'fresh_inputs': 2,
                'stale_inputs': 0,
            },
            'freshness': [{'status': 'fresh'}, {'status': 'fresh'}],
            'cutover_checkpoints': [{'name': 'ok', 'passed': True}],
        },
    )

    result = subprocess.run(
        [
            'go',
            'run',
            './cmd/bigclawctl',
            'automation',
            'migration',
            'export-live-shadow-bundle',
            '--go-root',
            str(root),
            '--run-id',
            '20260310T100601Z',
        ],
        cwd=BIGCLAW_GO_ROOT,
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr

    summary = json.loads((reports / 'live-shadow-summary.json').read_text(encoding='utf-8'))
    assert summary['run_id'] == '20260310T100601Z'
    assert summary['status'] == 'parity-ok'
    assert summary['severity'] == 'none'
    assert summary['artifacts']['shadow_compare_report_path'].endswith('shadow-compare-report.json')
    assert summary['compare_trace_id'] == 'compare-1'
    assert summary['matrix_trace_ids'] == ['matrix-1', 'matrix-2']

    manifest = json.loads((reports / 'live-shadow-index.json').read_text(encoding='utf-8'))
    assert manifest['latest']['run_id'] == '20260310T100601Z'
    assert manifest['recent_runs'][0]['run_id'] == '20260310T100601Z'
    assert manifest['drift_rollup']['summary']['highest_severity'] == 'none'
    assert manifest['drift_rollup']['summary']['drift_detected_runs'] == 0

    rollup = json.loads((reports / 'live-shadow-drift-rollup.json').read_text(encoding='utf-8'))
    assert rollup['status'] == 'parity-ok'
    assert rollup['summary']['recent_run_count'] == 1

    index_text = (reports / 'live-shadow-index.md').read_text(encoding='utf-8')
    assert 'Live Shadow Mirror Index' in index_text
    assert 'docs/reports/live-shadow-runs/20260310T100601Z' in index_text
    assert 'docs/migration-shadow.md' in index_text
    assert 'docs/reports/migration-readiness-report.md' in index_text
    assert 'docs/reports/migration-plan-review-notes.md' in index_text

    bundle_readme = (
        reports / 'live-shadow-runs' / '20260310T100601Z' / 'README.md'
    ).read_text(encoding='utf-8')
    assert 'Parity drift rollup' in bundle_readme


def test_export_live_shadow_bundle_supports_documented_bigclaw_go_cwd(tmp_path: Path) -> None:
    root = tmp_path / 'bigclaw-go'
    reports = root / 'docs' / 'reports'
    reports.mkdir(parents=True, exist_ok=True)
    (root / 'docs' / 'migration-shadow.md').write_text('# shadow\n', encoding='utf-8')
    (reports / 'migration-readiness-report.md').write_text('# readiness\n', encoding='utf-8')
    (reports / 'migration-plan-review-notes.md').write_text('# review\n', encoding='utf-8')
    (reports / 'live-shadow-comparison-follow-up-digest.md').write_text('# digest\n', encoding='utf-8')

    write_json(
        reports / 'shadow-compare-report.json',
        {
            'trace_id': 'compare-cwd',
            'primary': {'task_id': 'primary-cwd', 'events': [{'timestamp': '2026-03-10T10:00:00Z'}]},
            'shadow': {'task_id': 'shadow-cwd', 'events': [{'timestamp': '2026-03-10T10:00:01Z'}]},
            'diff': {
                'state_equal': True,
                'event_types_equal': True,
                'event_count_delta': 0,
                'primary_timeline_seconds': 0.1,
                'shadow_timeline_seconds': 0.15,
            },
        },
    )
    write_json(
        reports / 'shadow-matrix-report.json',
        {
            'total': 1,
            'matched': 1,
            'mismatched': 0,
            'results': [
                {
                    'trace_id': 'matrix-cwd',
                    'primary': {'task_id': 'matrix-primary-cwd', 'events': [{'timestamp': '2026-03-10T10:05:00Z'}]},
                    'shadow': {'task_id': 'matrix-shadow-cwd', 'events': [{'timestamp': '2026-03-10T10:05:01Z'}]},
                    'diff': {
                        'state_equal': True,
                        'event_types_equal': True,
                        'event_count_delta': 0,
                        'primary_timeline_seconds': 0.2,
                        'shadow_timeline_seconds': 0.22,
                    },
                }
            ],
        },
    )
    write_json(
        reports / 'rollback-trigger-surface.json',
        {
            'summary': {
                'status': 'manual-only',
                'automation_boundary': 'manual',
                'automated_rollback_trigger': False,
                'distinctions': {},
            },
        },
    )
    write_json(
        reports / 'live-shadow-mirror-scorecard.json',
        {
            'summary': {
                'latest_evidence_timestamp': '2026-03-10T10:05:01Z',
                'total_evidence_runs': 2,
                'parity_ok_count': 2,
                'drift_detected_count': 0,
                'matrix_total': 1,
                'matrix_mismatched': 0,
                'fresh_inputs': 2,
                'stale_inputs': 0,
            },
            'freshness': [{'status': 'fresh'}, {'status': 'fresh'}],
            'cutover_checkpoints': [{'name': 'ok', 'passed': True}],
        },
    )

    result = subprocess.run(
        [
            'go',
            'run',
            './cmd/bigclawctl',
            'automation',
            'migration',
            'export-live-shadow-bundle',
            '--go-root',
            str(root),
            '--run-id',
            '20260310T100501Z',
        ],
        cwd=BIGCLAW_GO_ROOT,
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr
    summary = json.loads((reports / 'live-shadow-summary.json').read_text(encoding='utf-8'))
    assert summary['run_id'] == '20260310T100501Z'
    assert summary['artifacts']['shadow_compare_report_path'].endswith('shadow-compare-report.json')


def test_checked_in_live_shadow_bundle_matches_expected_shape() -> None:
    manifest = json.loads(Path('bigclaw-go/docs/reports/live-shadow-index.json').read_text(encoding='utf-8'))
    latest = manifest['latest']
    rollup = manifest['drift_rollup']

    assert latest['run_id']
    assert latest['status'] == 'parity-ok'
    assert latest['severity'] == 'none'
    assert latest['summary']['drift_detected_count'] == 0
    assert latest['summary']['stale_inputs'] == 0
    assert latest['artifacts']['live_shadow_scorecard_path'].endswith('live-shadow-mirror-scorecard.json')
    assert rollup['status'] == 'parity-ok'
    assert rollup['summary']['highest_severity'] == 'none'
    assert rollup['summary']['latest_run_id'] == latest['run_id']
