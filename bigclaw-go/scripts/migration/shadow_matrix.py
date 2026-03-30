#!/usr/bin/env python3
import argparse
import json
import pathlib
import subprocess
import sys
import tempfile
from collections import defaultdict


def load_json(path):
    with open(path, 'r', encoding='utf-8') as fh:
        return json.load(fh)


def derive_task_shape(task):
    features = []
    executor = task.get('required_executor') or task.get('executor') or 'default'
    features.append(f'executor:{executor}')

    labels = sorted(str(label) for label in task.get('labels', []))
    if labels:
        features.append('labels:' + ','.join(labels))
    if task.get('budget_cents') is not None:
        features.append('budgeted')
    if task.get('acceptance_criteria'):
        features.append('acceptance')
    if task.get('validation_plan'):
        features.append('validation-plan')

    metadata = task.get('metadata') if isinstance(task.get('metadata'), dict) else {}
    scenario = metadata.get('scenario')
    if scenario:
        features.append(f'scenario:{scenario}')

    return '|'.join(features)


def make_execution_entry(task, source_kind, source_file, task_shape, slice_data=None):
    entry = {
        'task': dict(task),
        'source_kind': source_kind,
        'source_file': source_file,
        'task_shape': task_shape or derive_task_shape(task),
    }
    entry['task']['_source_file'] = source_file
    if slice_data:
        entry['corpus_slice'] = {
            'id': slice_data['slice_id'],
            'title': slice_data['title'],
            'weight': slice_data['weight'],
            'tags': slice_data['tags'],
        }
    return entry


def load_fixture_entries(task_files):
    entries = []
    for task_file in task_files or []:
        task = load_json(task_file)
        entries.append(
            make_execution_entry(
                task=task,
                source_kind='fixture',
                source_file=task_file,
                task_shape=derive_task_shape(task),
            )
        )
    return entries


def resolve_manifest_task_file(manifest_path, task_file):
    path = pathlib.Path(task_file)
    if path.is_absolute():
        return path
    return pathlib.Path(manifest_path).resolve().parent / path


def normalize_corpus_slice(slice_data, index, manifest_path):
    slice_id = slice_data.get('slice_id', f'corpus-slice-{index}')
    title = slice_data.get('title', slice_id)
    weight = int(slice_data.get('weight', 1))
    tags = [str(tag) for tag in slice_data.get('tags', [])]
    task = None
    source_file = None

    if 'task_file' in slice_data:
        source_file = slice_data['task_file']
        task = load_json(resolve_manifest_task_file(manifest_path, source_file))
    elif 'task' in slice_data:
        task = dict(slice_data['task'])
        source_file = f'{pathlib.Path(manifest_path).name}#{slice_id}'

    task_shape = slice_data.get('task_shape') or (derive_task_shape(task) if task else None)
    if not task_shape:
        raise ValueError(f'Corpus slice {slice_id} must define task_shape or provide task/task_file')

    return {
        'slice_id': slice_id,
        'title': title,
        'weight': weight,
        'tags': tags,
        'task_shape': task_shape,
        'task': task,
        'source_file': source_file,
        'replay': bool(slice_data.get('replay', False)),
        'notes': slice_data.get('notes', ''),
    }


def load_corpus_manifest_entries(manifest_path, replay_corpus_slices=False):
    manifest = load_json(manifest_path)
    slices = manifest.get('slices')
    if not isinstance(slices, list):
        raise ValueError('Corpus manifest must contain a top-level slices array')

    coverage_slices = []
    replay_entries = []
    for index, raw_slice in enumerate(slices, start=1):
        slice_data = normalize_corpus_slice(raw_slice, index, manifest_path)
        coverage_slices.append(slice_data)
        if replay_corpus_slices and slice_data.get('task') is not None:
            replay_entries.append(
                make_execution_entry(
                    task=slice_data['task'],
                    source_kind='corpus',
                    source_file=slice_data['source_file'],
                    task_shape=slice_data['task_shape'],
                    slice_data=slice_data,
                )
            )

    manifest_meta = {
        'name': manifest.get('name', pathlib.Path(manifest_path).stem),
        'generated_at': manifest.get('generated_at'),
        'source_file': str(manifest_path),
    }
    return manifest_meta, replay_entries, coverage_slices


def build_corpus_coverage(fixture_entries, corpus_slices, manifest_meta):
    fixture_by_shape = defaultdict(list)
    for entry in fixture_entries:
        fixture_by_shape[entry['task_shape']].append(entry)

    corpus_by_shape = defaultdict(
        lambda: {
            'slice_count': 0,
            'replayable_slice_count': 0,
            'corpus_weight': 0,
            'slice_ids': [],
            'titles': [],
        }
    )
    for slice_data in corpus_slices:
        aggregate = corpus_by_shape[slice_data['task_shape']]
        aggregate['slice_count'] += 1
        aggregate['replayable_slice_count'] += 1 if slice_data.get('task') is not None else 0
        aggregate['corpus_weight'] += slice_data['weight']
        aggregate['slice_ids'].append(slice_data['slice_id'])
        aggregate['titles'].append(slice_data['title'])

    shape_scorecard = []
    for task_shape in sorted(corpus_by_shape.keys(), key=lambda item: (-corpus_by_shape[item]['corpus_weight'], item)):
        aggregate = corpus_by_shape[task_shape]
        fixture_entries_for_shape = fixture_by_shape.get(task_shape, [])
        shape_scorecard.append(
            {
                'task_shape': task_shape,
                'fixture_task_count': len(fixture_entries_for_shape),
                'fixture_sources': [entry['source_file'] for entry in fixture_entries_for_shape],
                'corpus_slice_count': aggregate['slice_count'],
                'replayable_slice_count': aggregate['replayable_slice_count'],
                'corpus_weight': aggregate['corpus_weight'],
                'corpus_slice_ids': aggregate['slice_ids'],
                'corpus_titles': aggregate['titles'],
                'covered_by_fixture': bool(fixture_entries_for_shape),
            }
        )

    uncovered_slices = []
    for slice_data in corpus_slices:
        if fixture_by_shape.get(slice_data['task_shape']):
            continue
        uncovered_slices.append(
            {
                'slice_id': slice_data['slice_id'],
                'title': slice_data['title'],
                'task_shape': slice_data['task_shape'],
                'weight': slice_data['weight'],
                'replayable': slice_data.get('task') is not None,
                'source_file': slice_data['source_file'],
                'tags': slice_data['tags'],
                'notes': slice_data.get('notes', ''),
            }
        )

    return {
        'manifest_name': manifest_meta['name'],
        'manifest_source_file': manifest_meta['source_file'],
        'generated_at': manifest_meta.get('generated_at'),
        'fixture_task_count': len(fixture_entries),
        'corpus_slice_count': len(corpus_slices),
        'corpus_replayable_slice_count': sum(1 for slice_data in corpus_slices if slice_data.get('task') is not None),
        'covered_corpus_slice_count': len(corpus_slices) - len(uncovered_slices),
        'uncovered_corpus_slice_count': len(uncovered_slices),
        'shape_scorecard': shape_scorecard,
        'uncovered_slices': uncovered_slices,
    }


def build_report(results, fixture_entries, corpus_slices=None, manifest_meta=None):
    matched = sum(1 for item in results if item['diff']['state_equal'] and item['diff']['event_types_equal'])
    report = {
        'total': len(results),
        'matched': matched,
        'mismatched': len(results) - matched,
        'inputs': {
            'fixture_task_count': len(fixture_entries),
            'corpus_slice_count': len(corpus_slices or []),
            'manifest_name': manifest_meta['name'] if manifest_meta else None,
        },
        'results': results,
    }
    if corpus_slices and manifest_meta:
        report['corpus_coverage'] = build_corpus_coverage(fixture_entries, corpus_slices, manifest_meta)
    return report


def compare_task(primary, shadow, task, timeout_seconds, health_timeout_seconds):
    go_root = pathlib.Path(__file__).resolve().parents[2]
    with tempfile.NamedTemporaryFile('w', encoding='utf-8', suffix='.json', delete=False) as handle:
        json.dump(task, handle, ensure_ascii=False, indent=2)
        handle.write('\n')
        task_file = pathlib.Path(handle.name)
    with tempfile.NamedTemporaryFile('w', encoding='utf-8', suffix='.json', delete=False) as handle:
        report_file = pathlib.Path(handle.name)
    try:
        command = [
            'go',
            'run',
            './cmd/bigclawctl',
            'automation',
            'migration',
            'shadow-compare',
            '--primary',
            primary,
            '--shadow',
            shadow,
            '--task-file',
            str(task_file),
            '--timeout-seconds',
            str(timeout_seconds),
            '--health-timeout-seconds',
            str(health_timeout_seconds),
            '--report-path',
            str(report_file),
        ]
        completed = subprocess.run(
            command,
            cwd=go_root,
            capture_output=True,
            text=True,
            check=False,
        )
        if completed.returncode not in (0, 1):
            raise RuntimeError(completed.stderr.strip() or completed.stdout.strip() or 'shadow compare failed')
        report_payload = report_file.read_text(encoding='utf-8').strip()
        if report_payload:
            return json.loads(report_payload)
        if completed.stdout.strip():
            return json.loads(completed.stdout)
        raise RuntimeError('shadow compare produced no JSON report output')
    finally:
        task_file.unlink(missing_ok=True)
        report_file.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description='Run a shadow-comparison matrix across multiple task files')
    parser.add_argument('--primary', required=True)
    parser.add_argument('--shadow', required=True)
    parser.add_argument('--task-file', action='append')
    parser.add_argument('--corpus-manifest')
    parser.add_argument('--replay-corpus-slices', action='store_true')
    parser.add_argument('--timeout-seconds', type=int, default=180)
    parser.add_argument('--health-timeout-seconds', type=int, default=60)
    parser.add_argument('--report-path')
    args = parser.parse_args()

    if not args.task_file and not args.corpus_manifest:
        parser.error('at least one --task-file or --corpus-manifest must be provided')

    fixture_entries = load_fixture_entries(args.task_file)
    manifest_meta = None
    corpus_slices = []
    replay_entries = []
    if args.corpus_manifest:
        manifest_meta, replay_entries, corpus_slices = load_corpus_manifest_entries(
            args.corpus_manifest,
            replay_corpus_slices=args.replay_corpus_slices,
        )

    execution_entries = fixture_entries + replay_entries
    results = []
    for index, entry in enumerate(execution_entries, start=1):
        task = dict(entry['task'])
        base_id = task.get('id', f'matrix-task-{index}')
        task['id'] = f'{base_id}-m{index}'
        result = compare_task(
            args.primary,
            args.shadow,
            task,
            args.timeout_seconds,
            args.health_timeout_seconds,
        )
        result['source_file'] = entry['source_file']
        result['source_kind'] = entry['source_kind']
        result['task_shape'] = entry['task_shape']
        if 'corpus_slice' in entry:
            result['corpus_slice'] = entry['corpus_slice']
        results.append(result)

    report = build_report(results, fixture_entries, corpus_slices=corpus_slices, manifest_meta=manifest_meta)
    if args.report_path:
        path = pathlib.Path(args.report_path)
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if matched_all(results) else 1


def matched_all(results):
    return all(item['diff']['state_equal'] and item['diff']['event_types_equal'] for item in results)


if __name__ == '__main__':
    sys.exit(main())
