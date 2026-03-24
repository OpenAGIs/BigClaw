#!/usr/bin/env python3
from __future__ import annotations

import argparse
import contextlib
import importlib.util
import io
import json
import shutil
import statistics
import tempfile
import time
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any


def load_export_module():
    script_path = Path(__file__).resolve().parent / 'export_validation_bundle.py'
    spec = importlib.util.spec_from_file_location('export_validation_bundle', script_path)
    if spec is None or spec.loader is None:
        raise RuntimeError(f'unable to load module from {script_path}')
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def create_component_report(task_id: str, state: str = 'succeeded') -> dict[str, Any]:
    return {
        'task': {'id': task_id, 'required_executor': 'local'},
        'status': {'state': state},
        'events': [{'id': f'{task_id}-done', 'type': 'task.completed', 'timestamp': '2026-03-16T14:00:00Z'}],
    }


def create_archive_summary(run_id: str) -> dict[str, Any]:
    return {
        'run_id': run_id,
        'generated_at': f'{run_id[0:4]}-{run_id[4:6]}-{run_id[6:8]}T{run_id[9:11]}:{run_id[11:13]}:{run_id[13:15]}+00:00',
        'status': 'succeeded',
        'bundle_path': f'docs/reports/live-validation-runs/{run_id}',
        'summary_path': f'docs/reports/live-validation-runs/{run_id}/summary.json',
    }


def seed_archive(root: Path, archive_runs: int) -> tuple[str, str]:
    reports = root / 'docs' / 'reports'
    bundle_root = reports / 'live-validation-runs'
    bundle_root.mkdir(parents=True, exist_ok=True)

    start = datetime(2026, 3, 16, 12, 0, 0, tzinfo=timezone.utc)
    latest_run_id = ''
    latest_bundle_rel = ''
    for index in range(archive_runs):
        run_id = (start + timedelta(minutes=index)).strftime('%Y%m%dT%H%M%SZ')
        bundle_dir = bundle_root / run_id
        bundle_dir.mkdir(parents=True, exist_ok=True)
        summary = create_archive_summary(run_id)
        write_json(bundle_dir / 'summary.json', summary)
        latest_run_id = run_id
        latest_bundle_rel = f'docs/reports/live-validation-runs/{run_id}'

    write_json(reports / 'multi-node-shared-queue-report.json', {'all_ok': True, 'nodes': []})
    write_json(reports / 'validation-bundle-continuation-scorecard.json', {'status': 'ok'})
    write_json(
        reports / 'validation-bundle-continuation-policy-gate.json',
        {'status': 'go', 'recommendation': 'go', 'failing_checks': [], 'enforcement': {'mode': 'review'}},
    )
    (reports / 'validation-bundle-continuation-digest.md').write_text('# digest\n', encoding='utf-8')
    return latest_run_id, latest_bundle_rel


def seed_latest_inputs(root: Path, run_id: str) -> dict[str, str]:
    reports = root / 'docs' / 'reports'
    bundle_dir = reports / 'live-validation-runs' / run_id
    bundle_dir.mkdir(parents=True, exist_ok=True)

    local_stdout = root / 'artifacts' / 'local.stdout'
    local_stderr = root / 'artifacts' / 'local.stderr'
    k8s_stdout = root / 'artifacts' / 'k8s.stdout'
    k8s_stderr = root / 'artifacts' / 'k8s.stderr'
    ray_stdout = root / 'artifacts' / 'ray.stdout'
    ray_stderr = root / 'artifacts' / 'ray.stderr'
    for path, content in [
        (local_stdout, 'local ok\n'),
        (local_stderr, ''),
        (k8s_stdout, 'k8s ok\n'),
        (k8s_stderr, ''),
        (ray_stdout, 'ray ok\n'),
        (ray_stderr, ''),
    ]:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(content, encoding='utf-8')

    write_json(bundle_dir / 'sqlite-smoke-report.json', create_component_report('local-bench'))
    write_json(bundle_dir / 'kubernetes-live-smoke-report.json', create_component_report('k8s-bench'))
    write_json(bundle_dir / 'ray-live-smoke-report.json', create_component_report('ray-bench'))

    return {
        'bundle_dir': f'docs/reports/live-validation-runs/{run_id}',
        'local_report_path': f'docs/reports/live-validation-runs/{run_id}/sqlite-smoke-report.json',
        'kubernetes_report_path': f'docs/reports/live-validation-runs/{run_id}/kubernetes-live-smoke-report.json',
        'ray_report_path': f'docs/reports/live-validation-runs/{run_id}/ray-live-smoke-report.json',
        'local_stdout_path': str(local_stdout),
        'local_stderr_path': str(local_stderr),
        'kubernetes_stdout_path': str(k8s_stdout),
        'kubernetes_stderr_path': str(k8s_stderr),
        'ray_stdout_path': str(ray_stdout),
        'ray_stderr_path': str(ray_stderr),
    }


def percentile(values: list[float], pct: float) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    index = min(len(ordered) - 1, max(0, int(round((len(ordered) - 1) * pct))))
    return ordered[index]


def run_benchmark(iterations: int, archive_runs: int, keep_fixture: bool) -> dict[str, Any]:
    export_module = load_export_module()
    root_path: Path | None = None
    temp_dir: tempfile.TemporaryDirectory[str] | None = None
    if keep_fixture:
        root_path = Path(tempfile.mkdtemp(prefix='bigclaw-validation-benchmark-')).resolve()
    else:
        temp_dir = tempfile.TemporaryDirectory(prefix='bigclaw-validation-benchmark-')
        root_path = Path(temp_dir.name).resolve()

    try:
        run_id, _ = seed_archive(root_path, archive_runs)
        seeded = seed_latest_inputs(root_path, run_id)
        samples_ms: list[float] = []
        for _ in range(iterations):
            start = time.perf_counter()
            args = argparse.Namespace(
                go_root=str(root_path),
                run_id=run_id,
                bundle_dir=seeded['bundle_dir'],
                summary_path='docs/reports/live-validation-summary.json',
                index_path='docs/reports/live-validation-index.md',
                manifest_path='docs/reports/live-validation-index.json',
                run_local='1',
                run_kubernetes='1',
                run_ray='1',
                validation_status='0',
                run_broker='0',
                broker_backend='',
                broker_report_path='',
                broker_bootstrap_summary_path='',
                local_report_path=seeded['local_report_path'],
                local_stdout_path=seeded['local_stdout_path'],
                local_stderr_path=seeded['local_stderr_path'],
                kubernetes_report_path=seeded['kubernetes_report_path'],
                kubernetes_stdout_path=seeded['kubernetes_stdout_path'],
                kubernetes_stderr_path=seeded['kubernetes_stderr_path'],
                ray_report_path=seeded['ray_report_path'],
                ray_stdout_path=seeded['ray_stdout_path'],
                ray_stderr_path=seeded['ray_stderr_path'],
            )
            original_parse_args = export_module.parse_args
            export_module.parse_args = lambda: args
            try:
                with contextlib.redirect_stdout(io.StringIO()):
                    exit_code = export_module.main()
            finally:
                export_module.parse_args = original_parse_args
            if exit_code != 0:
                raise RuntimeError(f'export_validation_bundle.py exited with {exit_code}')
            samples_ms.append(round((time.perf_counter() - start) * 1000, 3))

        report = {
            'generated_at': datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z'),
            'status': 'ok',
            'title': 'Parallel validation bundle archive export benchmark',
            'benchmark': {
                'iterations': iterations,
                'archive_runs': archive_runs,
                'latest_run_id': run_id,
                'fixture_root': str(root_path),
                'keep_fixture': keep_fixture,
            },
            'timings_ms': {
                'samples': samples_ms,
                'min': round(min(samples_ms), 3),
                'max': round(max(samples_ms), 3),
                'mean': round(statistics.fmean(samples_ms), 3),
                'median': round(statistics.median(samples_ms), 3),
                'p95': round(percentile(samples_ms, 0.95), 3),
            },
            'artifacts': {
                'summary_path': 'docs/reports/live-validation-summary.json',
                'index_path': 'docs/reports/live-validation-index.md',
                'manifest_path': 'docs/reports/live-validation-index.json',
            },
        }
        return report
    finally:
        if temp_dir is not None:
            temp_dir.cleanup()
        elif root_path is not None and not keep_fixture and root_path.exists():
            shutil.rmtree(root_path)


def main() -> int:
    parser = argparse.ArgumentParser(description='Benchmark live validation bundle archive export/index generation')
    parser.add_argument('--iterations', type=int, default=5)
    parser.add_argument('--archive-runs', type=int, default=250)
    parser.add_argument('--output', default='')
    parser.add_argument('--pretty', action='store_true')
    parser.add_argument('--keep-fixture', action='store_true')
    args = parser.parse_args()

    report = run_benchmark(args.iterations, args.archive_runs, args.keep_fixture)
    output = json.dumps(report, ensure_ascii=False, indent=2 if args.pretty or args.output else None)
    if args.output:
        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(output)
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
