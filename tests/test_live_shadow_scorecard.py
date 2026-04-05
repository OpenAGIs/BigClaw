import json
import subprocess
from pathlib import Path


def test_live_shadow_scorecard_detects_drift_and_stale_inputs(tmp_path) -> None:
    compare_report = {
        'trace_id': 'compare-trace',
        'primary': {'task_id': 'compare-primary', 'events': [{'timestamp': '2026-03-01T10:00:00Z'}]},
        'shadow': {'task_id': 'compare-shadow', 'events': [{'timestamp': '2026-03-01T10:00:05Z'}]},
        'diff': {
            'state_equal': True,
            'event_types_equal': True,
            'event_count_delta': 0,
            'primary_timeline_seconds': 0.1,
            'shadow_timeline_seconds': 0.15,
        },
    }
    matrix_report = {
        'total': 1,
        'matched': 0,
        'mismatched': 1,
        'results': [
            {
                'trace_id': 'matrix-trace',
                'source_file': './examples/drift.json',
                'source_kind': 'fixture',
                'task_shape': 'executor:local|scenario:drift',
                'primary': {'task_id': 'matrix-primary', 'events': [{'timestamp': '2026-02-20T10:00:00Z'}]},
                'shadow': {'task_id': 'matrix-shadow', 'events': [{'timestamp': '2026-02-20T10:00:03Z'}]},
                'diff': {
                    'state_equal': True,
                    'event_types_equal': True,
                    'event_count_delta': 1,
                    'primary_timeline_seconds': 0.1,
                    'shadow_timeline_seconds': 0.6,
                },
            }
        ],
        'corpus_coverage': {
            'corpus_slice_count': 1,
            'uncovered_corpus_slice_count': 1,
        },
    }

    compare_path = tmp_path / 'shadow-compare.json'
    matrix_path = tmp_path / 'shadow-matrix.json'
    compare_path.write_text(json.dumps(compare_report), encoding='utf-8')
    matrix_path.write_text(json.dumps(matrix_report), encoding='utf-8')

    output_path = tmp_path / 'live-shadow-scorecard.json'
    repo_root = Path(__file__).resolve().parents[1] / 'bigclaw-go'
    result = subprocess.run(
        [
            'go',
            'run',
            str(repo_root / 'scripts' / 'migration' / 'live_shadow_scorecard'),
            '--shadow-compare-report', str(compare_path),
            '--shadow-matrix-report', str(matrix_path),
            '--output', str(output_path),
        ],
        check=False,
        capture_output=True,
        text=True,
        cwd=repo_root,
    )
    assert result.returncode == 0, result.stderr
    report = json.loads(output_path.read_text(encoding='utf-8'))

    assert report['ticket'] == 'BIG-PAR-092'
    assert report['summary']['total_evidence_runs'] == 2
    assert report['summary']['drift_detected_count'] == 1
    assert report['summary']['stale_inputs'] == 1
    assert report['parity_entries'][0]['parity']['status'] == 'parity-ok'
    assert report['parity_entries'][1]['parity']['status'] == 'drift-detected'
    assert report['parity_entries'][1]['parity']['reasons'] == ['event-count-drift', 'timeline-drift']
    assert report['cutover_checkpoints'][1]['passed'] is False
    assert report['cutover_checkpoints'][3]['passed'] is False


def test_checked_in_live_shadow_scorecard_matches_expected_shape() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json').read_text())

    assert report['ticket'] == 'BIG-PAR-092'
    assert report['status'] == 'repo-native-live-shadow-scorecard'
    assert report['summary']['matrix_mismatched'] == 0
    assert report['summary']['corpus_coverage_present'] is True
    assert report['summary']['stale_inputs'] == 0
    assert len(report['freshness']) == 2
    assert all(item['status'] == 'fresh' for item in report['freshness'])
    assert all(item['parity']['status'] == 'parity-ok' for item in report['parity_entries'])
