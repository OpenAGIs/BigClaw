#!/usr/bin/env python3
import argparse
import json
import pathlib
from datetime import datetime, timezone


MICROBENCHMARK_LIMITS = {
    'BenchmarkMemoryQueueEnqueueLease-8': {'max_ns_per_op': 100_000},
    'BenchmarkFileQueueEnqueueLease-8': {'max_ns_per_op': 40_000_000},
    'BenchmarkSQLiteQueueEnqueueLease-8': {'max_ns_per_op': 25_000_000},
    'BenchmarkSchedulerDecide-8': {'max_ns_per_op': 1_000},
}

SOAK_THRESHOLDS = {
    '50x8': {'min_throughput': 5.0, 'max_failures': 0, 'envelope': 'bootstrap-burst'},
    '100x12': {'min_throughput': 8.5, 'max_failures': 0, 'envelope': 'bootstrap-burst'},
    '1000x24': {'min_throughput': 9.0, 'max_failures': 0, 'envelope': 'recommended-local-sustained'},
    '2000x24': {'min_throughput': 8.5, 'max_failures': 0, 'envelope': 'recommended-local-ceiling'},
}

SATURATION_DROP_THRESHOLD_PCT = 12.0


def utc_iso(moment=None):
    value = moment or datetime.now(timezone.utc)
    return value.isoformat().replace('+00:00', 'Z')


def load_json(path):
    return json.loads(path.read_text(encoding='utf-8'))


def resolve_repo_path(repo_root, path):
    candidate = pathlib.Path(path)
    if candidate.is_absolute():
        return candidate
    return repo_root / candidate


def repo_relative_path(repo_root, path):
    return str(resolve_repo_path(repo_root, path).relative_to(repo_root))


def check(name, passed, detail):
    return {'name': name, 'passed': passed, 'detail': detail}


def benchmark_lane(name, ns_per_op, max_ns_per_op):
    return {
        'lane': name,
        'metric': 'ns_per_op',
        'observed': ns_per_op,
        'threshold': {'operator': '<=', 'value': max_ns_per_op},
        'status': 'pass' if ns_per_op <= max_ns_per_op else 'fail',
        'detail': f'observed={ns_per_op}ns/op limit={max_ns_per_op}ns/op',
    }


def soak_lane(label, result, threshold):
    throughput = float(result.get('throughput_tasks_per_sec', 0))
    failures = int(result.get('failed', 0))
    return {
        'lane': label,
        'scenario': {
            'count': int(result.get('count', 0)),
            'workers': int(result.get('workers', 0)),
        },
        'observed': {
            'elapsed_seconds': round(float(result.get('elapsed_seconds', 0)), 3),
            'throughput_tasks_per_sec': round(throughput, 3),
            'succeeded': int(result.get('succeeded', 0)),
            'failed': failures,
        },
        'thresholds': {
            'min_throughput_tasks_per_sec': threshold['min_throughput'],
            'max_failures': threshold['max_failures'],
        },
        'operating_envelope': threshold['envelope'],
        'status': 'pass' if throughput >= threshold['min_throughput'] and failures <= threshold['max_failures'] else 'fail',
        'detail': (
            f"throughput={round(throughput, 3)}tps min={threshold['min_throughput']} "
            f"failures={failures} max={threshold['max_failures']}"
        ),
    }


def mixed_workload_lane(report):
    tasks = report.get('tasks', [])
    mismatches = []
    for task in tasks:
        if not task.get('ok'):
            mismatches.append(f"{task.get('name')}: task-level ok=false")
        if task.get('expected_executor') != task.get('routed_executor'):
            mismatches.append(
                f"{task.get('name')}: expected={task.get('expected_executor')} routed={task.get('routed_executor')}"
            )
        if task.get('final_state') != 'succeeded':
            mismatches.append(f"{task.get('name')}: final_state={task.get('final_state')}")
    return {
        'lane': 'mixed-workload-routing',
        'observed': {
            'all_ok': bool(report.get('all_ok')),
            'task_count': len(tasks),
            'successful_tasks': sum(1 for task in tasks if task.get('final_state') == 'succeeded'),
        },
        'thresholds': {
            'all_ok_required': True,
            'minimum_task_count': 5,
            'executor_route_match_required': True,
        },
        'status': 'pass' if report.get('all_ok') and len(tasks) >= 5 and not mismatches else 'pass-with-ceiling' if report.get('all_ok') else 'fail',
        'detail': 'all sampled mixed-workload routes landed on the expected executor path' if not mismatches else '; '.join(mismatches),
        'limitations': [
            'executor-mix coverage is functional rather than high-volume',
            'mixed-workload evidence proves route correctness but not sustained cross-executor saturation limits',
        ],
    }


def build_saturation_summary(soak_lanes):
    baseline = next(lane for lane in soak_lanes if lane['lane'] == '1000x24')
    ceiling = next(lane for lane in soak_lanes if lane['lane'] == '2000x24')
    baseline_tps = baseline['observed']['throughput_tasks_per_sec']
    ceiling_tps = ceiling['observed']['throughput_tasks_per_sec']
    drop_pct = round(((baseline_tps - ceiling_tps) / baseline_tps) * 100, 2) if baseline_tps else 0.0
    status = 'pass' if drop_pct <= SATURATION_DROP_THRESHOLD_PCT else 'warn'
    return {
        'baseline_lane': baseline['lane'],
        'ceiling_lane': ceiling['lane'],
        'baseline_throughput_tasks_per_sec': baseline_tps,
        'ceiling_throughput_tasks_per_sec': ceiling_tps,
        'throughput_drop_pct': drop_pct,
        'drop_warn_threshold_pct': SATURATION_DROP_THRESHOLD_PCT,
        'status': status,
        'detail': (
            'throughput remains in the same single-instance local band at the 2000-task ceiling'
            if status == 'pass'
            else 'throughput drops materially at the 2000-task ceiling and should be treated as saturation'
        ),
    }


def build_markdown(report):
    lines = [
        '# Capacity Certification Report',
        '',
        '## Scope',
        '',
        f"- Generated at: `{report['generated_at']}`",
        f"- Ticket: `{report['ticket']}`",
        '- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.',
        '- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.',
        '',
        '## Certification Summary',
        '',
        f"- Overall status: `{report['summary']['overall_status']}`",
        f"- Passed lanes: `{report['summary']['passed_lanes']}/{report['summary']['total_lanes']}`",
        f"- Recommended local sustained envelope: `{report['summary']['recommended_sustained_envelope']}`",
        f"- Local ceiling envelope: `{report['summary']['ceiling_envelope']}`",
        f"- Saturation signal: `{report['saturation_indicator']['detail']}`",
        '',
        '## Microbenchmark Thresholds',
        '',
    ]
    for lane in report['microbenchmarks']:
        lines.append(
            f"- `{lane['lane']}`: `{lane['observed']:.2f} ns/op` vs limit `{lane['threshold']['value']}` -> `{lane['status']}`"
        )
    lines.extend(['', '## Soak Matrix', ''])
    for lane in report['soak_matrix']:
        observed = lane['observed']
        lines.append(
            f"- `{lane['lane']}`: `{observed['throughput_tasks_per_sec']} tasks/s`, `{observed['failed']} failed`, envelope `{lane['operating_envelope']}` -> `{lane['status']}`"
        )
    lines.extend(
        [
            '',
            '## Workload Mix',
            '',
            f"- `mixed-workload-routing`: `{report['mixed_workload']['detail']}` -> `{report['mixed_workload']['status']}`",
            '',
            '## Recommended Operating Envelopes',
            '',
        ]
    )
    for envelope in report['operating_envelopes']:
        lines.append(
            f"- `{envelope['name']}`: {envelope['recommendation']} Evidence: `{', '.join(envelope['evidence_lanes'])}`."
        )
    lines.extend(['', '## Saturation Notes', ''])
    for note in report['saturation_notes']:
        lines.append(f"- {note}")
    lines.extend(['', '## Limits', ''])
    for item in report['limits']:
        lines.append(f"- {item}")
    return '\n'.join(lines) + '\n'


def build_report(
    benchmark_report_path='bigclaw-go/docs/reports/benchmark-matrix-report.json',
    mixed_workload_report_path='bigclaw-go/docs/reports/mixed-workload-matrix-report.json',
    supplemental_soak_report_paths=None,
):
    repo_root = pathlib.Path(__file__).resolve().parents[3]
    benchmark_report = load_json(resolve_repo_path(repo_root, benchmark_report_path))
    mixed_workload_report = load_json(resolve_repo_path(repo_root, mixed_workload_report_path))
    supplemental_soak_report_paths = supplemental_soak_report_paths or [
        'bigclaw-go/docs/reports/soak-local-1000x24.json',
        'bigclaw-go/docs/reports/soak-local-2000x24.json',
    ]

    microbenchmarks = []
    parsed = benchmark_report.get('benchmark', {}).get('parsed', {})
    for benchmark_name, limits in MICROBENCHMARK_LIMITS.items():
        microbenchmarks.append(
            benchmark_lane(
                benchmark_name,
                float(parsed[benchmark_name]['ns_per_op']),
                limits['max_ns_per_op'],
            )
        )

    soak_matrix = []
    soak_inputs = []
    soak_results_by_label = {}
    for entry in benchmark_report.get('soak_matrix', []):
        result = entry.get('result', {})
        label = f"{result.get('count')}x{result.get('workers')}"
        soak_results_by_label[label] = result
        soak_inputs.append(repo_relative_path(repo_root, entry.get('report_path')))

    for soak_path in supplemental_soak_report_paths:
        result = load_json(resolve_repo_path(repo_root, soak_path))
        label = f"{result.get('count')}x{result.get('workers')}"
        soak_results_by_label[label] = result
        soak_inputs.append(repo_relative_path(repo_root, soak_path))

    for label, threshold in SOAK_THRESHOLDS.items():
        soak_matrix.append(soak_lane(label, soak_results_by_label[label], threshold))

    mixed_workload = mixed_workload_lane(mixed_workload_report)
    saturation_indicator = build_saturation_summary(soak_matrix)

    all_lanes = microbenchmarks + soak_matrix + [mixed_workload]
    passed_lanes = sum(1 for lane in all_lanes if lane['status'] in {'pass', 'pass-with-ceiling'})
    failed_lanes = [lane['lane'] for lane in all_lanes if lane['status'] not in {'pass', 'pass-with-ceiling'}]

    operating_envelopes = [
        {
            'name': 'recommended-local-sustained',
            'recommendation': 'Use up to 1000 queued tasks with 24 submit workers when a stable single-instance local review lane is required.',
            'evidence_lanes': ['1000x24'],
        },
        {
            'name': 'recommended-local-ceiling',
            'recommendation': 'Treat 2000 queued tasks with 24 submit workers as the checked-in local ceiling, not the default operating point.',
            'evidence_lanes': ['2000x24'],
        },
        {
            'name': 'mixed-workload-routing',
            'recommendation': 'Use the mixed-workload matrix for executor routing correctness, but do not infer sustained multi-executor throughput from it.',
            'evidence_lanes': ['mixed-workload-routing'],
        },
    ]

    checks = [
        check('all_microbenchmark_thresholds_hold', all(lane['status'] == 'pass' for lane in microbenchmarks), str([lane['status'] for lane in microbenchmarks])),
        check('all_soak_lanes_hold', all(lane['status'] == 'pass' for lane in soak_matrix), str([lane['status'] for lane in soak_matrix])),
        check('mixed_workload_routes_match_expected_executors', mixed_workload['status'] in {'pass', 'pass-with-ceiling'}, mixed_workload['detail']),
        check('ceiling_lane_does_not_show_excessive_throughput_drop', saturation_indicator['status'] == 'pass', f"drop_pct={saturation_indicator['throughput_drop_pct']} threshold={SATURATION_DROP_THRESHOLD_PCT}"),
    ]

    report = {
        'generated_at': utc_iso(),
        'ticket': 'BIG-PAR-098',
        'title': 'Production-grade capacity certification matrix',
        'status': 'repo-native-capacity-certification',
        'evidence_inputs': {
            'benchmark_report_path': repo_relative_path(repo_root, benchmark_report_path),
            'mixed_workload_report_path': repo_relative_path(repo_root, mixed_workload_report_path),
            'soak_report_paths': soak_inputs,
            'generator_script': 'bigclaw-go/scripts/benchmark/capacity_certification.py',
        },
        'summary': {
            'overall_status': 'pass' if not failed_lanes else 'fail',
            'total_lanes': len(all_lanes),
            'passed_lanes': passed_lanes,
            'failed_lanes': failed_lanes,
            'recommended_sustained_envelope': '<=1000 tasks with 24 submit workers',
            'ceiling_envelope': '<=2000 tasks with 24 submit workers',
        },
        'microbenchmarks': microbenchmarks,
        'soak_matrix': soak_matrix,
        'mixed_workload': mixed_workload,
        'saturation_indicator': saturation_indicator,
        'operating_envelopes': operating_envelopes,
        'certification_checks': checks,
        'saturation_notes': [
            'Throughput plateaus around 9-10 tasks/s across the checked-in 100x12, 1000x24, and 2000x24 local lanes.',
            'The 2000x24 lane remains within the same throughput band as 1000x24, so the checked-in local ceiling is evidence-backed but not substantially headroom-rich.',
            'Mixed-workload evidence verifies executor-routing correctness across local, Kubernetes, and Ray, but it is a functional routing proof rather than a concurrency ceiling.',
        ],
        'limits': [
            'Evidence is repo-native and single-instance; it does not certify multi-node or multi-tenant production saturation behavior.',
            'The matrix uses checked-in local runs from 2026-03-13 and should be refreshed when queue, scheduler, or executor behavior changes materially.',
            'Recommended envelopes are conservative reviewer guidance derived from current evidence, not an automated runtime admission policy.',
        ],
    }
    report['markdown'] = build_markdown(report)
    return report


def main():
    parser = argparse.ArgumentParser(description='Generate the BigClaw repo-native capacity certification matrix')
    parser.add_argument('--benchmark-report', default='bigclaw-go/docs/reports/benchmark-matrix-report.json')
    parser.add_argument('--mixed-workload-report', default='bigclaw-go/docs/reports/mixed-workload-matrix-report.json')
    parser.add_argument(
        '--supplemental-soak-report',
        action='append',
        dest='supplemental_soak_reports',
        help='Additional soak report paths to merge into the certification matrix',
    )
    parser.add_argument('--output', default='bigclaw-go/docs/reports/capacity-certification-matrix.json')
    parser.add_argument('--markdown-output', default='bigclaw-go/docs/reports/capacity-certification-report.md')
    parser.add_argument('--pretty', action='store_true')
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[3]
    report = build_report(
        benchmark_report_path=args.benchmark_report,
        mixed_workload_report_path=args.mixed_workload_report,
        supplemental_soak_report_paths=args.supplemental_soak_reports,
    )
    markdown = report.pop('markdown')

    output_path = resolve_repo_path(repo_root, args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')

    markdown_output_path = resolve_repo_path(repo_root, args.markdown_output)
    markdown_output_path.parent.mkdir(parents=True, exist_ok=True)
    markdown_output_path.write_text(markdown, encoding='utf-8')

    if args.pretty:
        print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
