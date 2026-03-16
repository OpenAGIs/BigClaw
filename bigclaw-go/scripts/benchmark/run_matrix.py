#!/usr/bin/env python3
import argparse
import json
import pathlib
import re
import subprocess
import time


READINESS_REPORT_PATH = 'docs/reports/benchmark-readiness-report.md'
LEGACY_BENCHMARK_REPORT_PATH = 'docs/reports/benchmark-report.md'


def parse_benchmark_stdout(stdout):
    parsed = {}
    for line in stdout.splitlines():
        line = line.strip()
        match = re.match(r'^(Benchmark\S+)\s+\d+\s+([0-9.]+)\s+ns/op$', line)
        if match:
            parsed[match.group(1)] = {'ns_per_op': float(match.group(2))}
    return parsed


def run_benchmarks(go_root):
    result = subprocess.run(
        ['go', 'test', '-bench', '.', './internal/queue', './internal/scheduler'],
        cwd=go_root,
        capture_output=True,
        text=True,
        check=True,
    )
    return {'stdout': result.stdout, 'parsed': parse_benchmark_stdout(result.stdout)}


def run_soak(go_root, count, workers, timeout_seconds, report_path):
    subprocess.run(
        [
            'python3', 'scripts/benchmark/soak_local.py',
            '--autostart',
            '--count', str(count),
            '--workers', str(workers),
            '--timeout-seconds', str(timeout_seconds),
            '--report-path', str(report_path),
        ],
        cwd=go_root,
        check=True,
        text=True,
    )
    return json.loads((go_root / report_path).read_text(encoding='utf-8'))


def scenario_id(count, workers):
    return f'{count}x{workers}'


def benchmark_cases(parsed):
    return [
        {
            'name': name,
            'ns_per_op': values['ns_per_op'],
        }
        for name, values in parsed.items()
    ]


def build_report(report_path, benchmark_result, soak_results):
    soak_artifacts = []
    throughput_values = []
    all_zero_failures = True
    all_completed = True
    for item in soak_results:
        result = item['result']
        throughput = result.get('throughput_tasks_per_sec', 0)
        throughput_values.append(throughput)
        failed = result.get('failed', 0)
        succeeded = result.get('succeeded', 0)
        count = item['scenario']['count']
        workers = item['scenario']['workers']
        zero_failures = failed == 0
        completed = succeeded == count
        all_zero_failures = all_zero_failures and zero_failures
        all_completed = all_completed and completed
        soak_artifacts.append({
            'artifact_type': 'soak_report',
            'scenario_id': scenario_id(count, workers),
            'count': count,
            'workers': workers,
            'report_path': item['report_path'],
            'elapsed_seconds': result.get('elapsed_seconds'),
            'throughput_tasks_per_sec': throughput,
            'succeeded': succeeded,
            'failed': failed,
            'zero_failures': zero_failures,
            'all_tasks_completed': completed,
        })

    readiness_status = 'ready' if all_zero_failures and all_completed else 'needs-attention'
    readiness_highlights = [
        f"{len(benchmark_result['parsed'])} benchmark cases captured in one canonical matrix report.",
        f"{len(soak_artifacts)} soak scenarios exported as stable scenario artifacts.",
    ]
    if throughput_values:
        readiness_highlights.append(
            'Observed soak throughput band: '
            f"{min(throughput_values):.3f}-{max(throughput_values):.3f} tasks/s."
        )
    readiness_highlights.append(
        'All soak artifacts completed without failures.'
        if all_zero_failures and all_completed
        else 'One or more soak artifacts recorded failures or incomplete completion.'
    )

    return {
        'report_version': 2,
        'generated_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
        'artifacts': {
            'canonical_report_path': report_path,
            'readiness_report_path': READINESS_REPORT_PATH,
            'legacy_benchmark_report_path': LEGACY_BENCHMARK_REPORT_PATH,
            'benchmark_cases': benchmark_cases(benchmark_result['parsed']),
            'soak_reports': soak_artifacts,
        },
        'readiness': {
            'status': readiness_status,
            'all_soak_zero_failures': all_zero_failures,
            'all_soak_completed': all_completed,
            'highlights': readiness_highlights,
        },
        'benchmark': benchmark_result,
        'soak_matrix': soak_results,
    }


def validate_report(report):
    if not isinstance(report, dict):
        raise ValueError('report must be a JSON object')

    for required_key in ('report_version', 'generated_at', 'artifacts', 'readiness', 'benchmark', 'soak_matrix'):
        if required_key not in report:
            raise ValueError(f'missing top-level key: {required_key}')

    artifacts = report['artifacts']
    readiness = report['readiness']
    benchmark = report['benchmark']
    soak_matrix = report['soak_matrix']
    if not isinstance(artifacts, dict):
        raise ValueError('artifacts must be an object')
    if not isinstance(readiness, dict):
        raise ValueError('readiness must be an object')
    if not isinstance(benchmark, dict):
        raise ValueError('benchmark must be an object')
    if not isinstance(soak_matrix, list):
        raise ValueError('soak_matrix must be a list')

    if 'canonical_report_path' not in artifacts:
        raise ValueError('artifacts.canonical_report_path is required')
    if 'benchmark_cases' not in artifacts or not isinstance(artifacts['benchmark_cases'], list):
        raise ValueError('artifacts.benchmark_cases must be a list')
    if 'soak_reports' not in artifacts or not isinstance(artifacts['soak_reports'], list):
        raise ValueError('artifacts.soak_reports must be a list')
    if len(artifacts['soak_reports']) != len(soak_matrix):
        raise ValueError('artifacts.soak_reports must align with soak_matrix length')

    parsed = benchmark.get('parsed')
    if not isinstance(parsed, dict):
        raise ValueError('benchmark.parsed must be an object')
    if len(artifacts['benchmark_cases']) != len(parsed):
        raise ValueError('artifacts.benchmark_cases must align with benchmark.parsed')

    for case in artifacts['benchmark_cases']:
        if not isinstance(case, dict) or 'name' not in case or 'ns_per_op' not in case:
            raise ValueError('each artifacts.benchmark_cases entry must include name and ns_per_op')

    for artifact, soak in zip(artifacts['soak_reports'], soak_matrix):
        if artifact.get('report_path') != soak.get('report_path'):
            raise ValueError('artifact report_path must match soak_matrix report_path')
        scenario = soak.get('scenario', {})
        expected_id = scenario_id(scenario.get('count'), scenario.get('workers'))
        if artifact.get('scenario_id') != expected_id:
            raise ValueError('artifact scenario_id must match soak_matrix scenario')

    if readiness.get('status') not in {'ready', 'needs-attention'}:
        raise ValueError('readiness.status must be ready or needs-attention')
    if not isinstance(readiness.get('highlights'), list) or not readiness['highlights']:
        raise ValueError('readiness.highlights must be a non-empty list')


def main():
    parser = argparse.ArgumentParser(description='Run a local benchmark/soak matrix for BigClaw Go')
    parser.add_argument('--go-root', default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument('--report-path', default='docs/reports/benchmark-matrix-report.json')
    parser.add_argument('--timeout-seconds', type=int, default=180)
    parser.add_argument('--scenario', action='append', help='count:workers; default scenarios are 50:8 and 100:12')
    parser.add_argument('--validate-report', help='Validate an existing benchmark matrix report and exit')
    args = parser.parse_args()

    go_root = pathlib.Path(args.go_root)
    if args.validate_report:
        report = json.loads((go_root / args.validate_report).read_text(encoding='utf-8'))
        validate_report(report)
        print(f'validated {args.validate_report}')
        return 0

    scenarios = args.scenario or ['50:8', '100:12']
    benchmark_result = run_benchmarks(go_root)

    soak_results = []
    for scenario in scenarios:
        count_str, workers_str = scenario.split(':', 1)
        count = int(count_str)
        workers = int(workers_str)
        report_path = pathlib.Path(f'docs/reports/soak-local-{count}x{workers}.json')
        soak_report = run_soak(go_root, count, workers, args.timeout_seconds, report_path)
        soak_results.append({
            'scenario': {'count': count, 'workers': workers},
            'report_path': str(report_path),
            'result': soak_report,
        })

    report = build_report(args.report_path, benchmark_result, soak_results)
    validate_report(report)
    output_path = go_root / args.report_path
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
