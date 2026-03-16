import importlib.util
import json
import sys
from pathlib import Path


def load_shadow_matrix_module():
    script_path = Path('bigclaw-go/scripts/migration/shadow_matrix.py')
    sys.path.insert(0, str(script_path.parent))
    spec = importlib.util.spec_from_file_location('shadow_matrix', script_path)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


shadow_matrix = load_shadow_matrix_module()


def test_corpus_manifest_scorecard_marks_uncovered_shapes(tmp_path) -> None:
    fixture_basic = tmp_path / 'fixture-basic.json'
    fixture_basic.write_text(
        json.dumps(
            {
                'id': 'fixture-basic',
                'required_executor': 'local',
                'entrypoint': 'echo basic',
                'metadata': {'scenario': 'baseline'},
            }
        ),
        encoding='utf-8',
    )
    fixture_budget = tmp_path / 'fixture-budget.json'
    fixture_budget.write_text(
        json.dumps(
            {
                'id': 'fixture-budget',
                'required_executor': 'local',
                'entrypoint': 'echo budget',
                'budget_cents': 25,
                'labels': ['budget', 'shadow'],
                'metadata': {'scenario': 'budget'},
            }
        ),
        encoding='utf-8',
    )

    manifest = tmp_path / 'corpus.json'
    manifest.write_text(
        json.dumps(
            {
                'name': 'anon-corpus',
                'generated_at': '2026-03-16T10:00:00Z',
                'slices': [
                    {
                        'slice_id': 'baseline',
                        'title': 'Baseline slice',
                        'weight': 10,
                        'task_file': './fixture-basic.json',
                    },
                    {
                        'slice_id': 'validation',
                        'title': 'Validation slice',
                        'weight': 6,
                        'task': {
                            'id': 'validation-task',
                            'required_executor': 'local',
                            'entrypoint': 'echo validation',
                            'acceptance_criteria': ['ok'],
                            'validation_plan': ['compare output'],
                            'metadata': {'scenario': 'validation'},
                        },
                    },
                    {
                        'slice_id': 'browser-review',
                        'title': 'Browser review',
                        'weight': 3,
                        'task_shape': 'executor:browser|labels:browser,human-review|scenario:browser-review',
                        'tags': ['browser', 'human-review'],
                    },
                ],
            }
        ),
        encoding='utf-8',
    )

    fixture_entries = shadow_matrix.load_fixture_entries([str(fixture_basic), str(fixture_budget)])
    manifest_meta, replay_entries, corpus_slices = shadow_matrix.load_corpus_manifest_entries(str(manifest))
    coverage = shadow_matrix.build_corpus_coverage(fixture_entries, corpus_slices, manifest_meta)

    assert replay_entries == []
    assert coverage['manifest_name'] == 'anon-corpus'
    assert coverage['fixture_task_count'] == 2
    assert coverage['corpus_slice_count'] == 3
    assert coverage['corpus_replayable_slice_count'] == 2
    assert coverage['covered_corpus_slice_count'] == 1
    assert coverage['uncovered_corpus_slice_count'] == 2
    assert [item['slice_id'] for item in coverage['uncovered_slices']] == ['validation', 'browser-review']
    assert any(item['task_shape'] == 'executor:local|scenario:baseline' and item['covered_by_fixture'] for item in coverage['shape_scorecard'])


def test_corpus_manifest_can_promote_replayable_slices(tmp_path) -> None:
    task_file = tmp_path / 'task.json'
    task_file.write_text(
        json.dumps(
            {
                'id': 'replayable-task',
                'required_executor': 'local',
                'entrypoint': 'echo replay',
                'metadata': {'scenario': 'baseline'},
            }
        ),
        encoding='utf-8',
    )
    manifest = tmp_path / 'corpus.json'
    manifest.write_text(
        json.dumps(
            {
                'name': 'replay-pack',
                'slices': [
                    {
                        'slice_id': 'baseline',
                        'title': 'Baseline slice',
                        'weight': 2,
                        'task_file': './task.json',
                        'replay': True,
                    },
                    {
                        'slice_id': 'metadata-only',
                        'title': 'Metadata only',
                        'weight': 1,
                        'task_shape': 'executor:browser|scenario:review',
                    },
                ],
            }
        ),
        encoding='utf-8',
    )

    manifest_meta, replay_entries, corpus_slices = shadow_matrix.load_corpus_manifest_entries(
        str(manifest), replay_corpus_slices=True
    )
    report = shadow_matrix.build_report([], [], corpus_slices=corpus_slices, manifest_meta=manifest_meta)

    assert len(replay_entries) == 1
    assert replay_entries[0]['source_kind'] == 'corpus'
    assert replay_entries[0]['corpus_slice']['id'] == 'baseline'
    assert report['inputs']['manifest_name'] == 'replay-pack'
    assert report['corpus_coverage']['uncovered_corpus_slice_count'] == 2


def test_shadow_matrix_report_records_corpus_coverage_scorecard() -> None:
    report = json.loads(Path('bigclaw-go/docs/reports/shadow-matrix-report.json').read_text())

    assert 'corpus_coverage' in report
    assert report['corpus_coverage']['manifest_name'] == 'anonymized-production-corpus-v1'
    assert report['corpus_coverage']['uncovered_corpus_slice_count'] == 1
    assert report['corpus_coverage']['uncovered_slices'][0]['slice_id'] == 'browser-human-review'
