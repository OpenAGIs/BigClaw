#!/usr/bin/env python3
import argparse
import json
import pathlib
import re
import subprocess


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


def main():
    parser = argparse.ArgumentParser(description='Run a local benchmark/soak matrix for BigClaw Go')
    parser.add_argument('--go-root', default=str(pathlib.Path(__file__).resolve().parents[2]))
    parser.add_argument('--report-path', default='docs/reports/benchmark-matrix-report.json')
    parser.add_argument('--timeout-seconds', type=int, default=180)
    parser.add_argument('--scenario', action='append', help='count:workers; default scenarios are 50:8 and 100:12')
    args = parser.parse_args()

    go_root = pathlib.Path(args.go_root)
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

    report = {
        'benchmark': benchmark_result,
        'soak_matrix': soak_results,
    }
    output_path = go_root / args.report_path
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding='utf-8')
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
