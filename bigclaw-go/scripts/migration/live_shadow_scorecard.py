#!/usr/bin/env python3
import argparse
import json
import pathlib
from datetime import datetime, timezone


TIMELINE_DRIFT_TOLERANCE_SECONDS = 0.25
EVIDENCE_FRESHNESS_SLO_HOURS = 168


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


def parse_time(value):
    return datetime.fromisoformat(str(value).replace('Z', '+00:00'))


def collect_event_timestamps(events):
    timestamps = []
    for event in events:
        timestamp = event.get('timestamp')
        if not timestamp:
            continue
        timestamps.append(parse_time(timestamp))
    return timestamps


def latest_report_timestamp(report):
    timestamps = []
    if 'results' in report:
        for item in report['results']:
            timestamps.extend(collect_event_timestamps(item.get('primary', {}).get('events', [])))
            timestamps.extend(collect_event_timestamps(item.get('shadow', {}).get('events', [])))
    else:
        timestamps.extend(collect_event_timestamps(report.get('primary', {}).get('events', [])))
        timestamps.extend(collect_event_timestamps(report.get('shadow', {}).get('events', [])))
    return max(timestamps) if timestamps else None


def build_freshness_entry(name, path, report, generated_at):
    latest_timestamp = latest_report_timestamp(report)
    age_hours = None
    status = 'missing-timestamps'
    if latest_timestamp is not None:
        age_hours = round((generated_at - latest_timestamp).total_seconds() / 3600, 2)
        status = 'fresh' if age_hours <= EVIDENCE_FRESHNESS_SLO_HOURS else 'stale'
    return {
        'name': name,
        'report_path': path,
        'latest_evidence_timestamp': utc_iso(latest_timestamp) if latest_timestamp else None,
        'age_hours': age_hours,
        'freshness_slo_hours': EVIDENCE_FRESHNESS_SLO_HOURS,
        'status': status,
    }


def classify_parity(diff):
    reasons = []
    if not diff.get('state_equal'):
        reasons.append('terminal-state-mismatch')
    if not diff.get('event_types_equal'):
        reasons.append('event-sequence-mismatch')
    event_count_delta = int(diff.get('event_count_delta', 0))
    if event_count_delta != 0:
        reasons.append('event-count-drift')
    timeline_delta = round(
        abs(float(diff.get('primary_timeline_seconds', 0)) - float(diff.get('shadow_timeline_seconds', 0))),
        6,
    )
    if timeline_delta > TIMELINE_DRIFT_TOLERANCE_SECONDS:
        reasons.append('timeline-drift')
    return {
        'status': 'parity-ok' if not reasons else 'drift-detected',
        'timeline_delta_seconds': timeline_delta,
        'timeline_drift_tolerance_seconds': TIMELINE_DRIFT_TOLERANCE_SECONDS,
        'reasons': reasons,
    }


def build_compare_entry(report):
    parity = classify_parity(report.get('diff', {}))
    return {
        'entry_type': 'single-compare',
        'label': 'single fixture compare',
        'trace_id': report.get('trace_id'),
        'source_file': None,
        'source_kind': 'fixture',
        'parity': parity,
        'primary_task_id': report.get('primary', {}).get('task_id'),
        'shadow_task_id': report.get('shadow', {}).get('task_id'),
    }


def build_matrix_entries(report):
    entries = []
    for item in report.get('results', []):
        parity = classify_parity(item.get('diff', {}))
        entries.append(
            {
                'entry_type': 'matrix-row',
                'label': item.get('source_file'),
                'trace_id': item.get('trace_id'),
                'source_file': item.get('source_file'),
                'source_kind': item.get('source_kind'),
                'task_shape': item.get('task_shape'),
                'corpus_slice': item.get('corpus_slice'),
                'parity': parity,
                'primary_task_id': item.get('primary', {}).get('task_id'),
                'shadow_task_id': item.get('shadow', {}).get('task_id'),
            }
        )
    return entries


def check(name, passed, detail):
    return {'name': name, 'passed': passed, 'detail': detail}


def build_report(
    shadow_compare_report_path='bigclaw-go/docs/reports/shadow-compare-report.json',
    shadow_matrix_report_path='bigclaw-go/docs/reports/shadow-matrix-report.json',
):
    repo_root = pathlib.Path(__file__).resolve().parents[3]
    compare_path = resolve_repo_path(repo_root, shadow_compare_report_path)
    matrix_path = resolve_repo_path(repo_root, shadow_matrix_report_path)
    compare_report = load_json(compare_path)
    matrix_report = load_json(matrix_path)
    generated_at = datetime.now(timezone.utc)

    parity_entries = [build_compare_entry(compare_report)] + build_matrix_entries(matrix_report)
    parity_ok_count = sum(1 for item in parity_entries if item['parity']['status'] == 'parity-ok')
    drift_detected_count = len(parity_entries) - parity_ok_count

    freshness = [
        build_freshness_entry('shadow-compare-report', shadow_compare_report_path, compare_report, generated_at),
        build_freshness_entry('shadow-matrix-report', shadow_matrix_report_path, matrix_report, generated_at),
    ]
    stale_inputs = [item for item in freshness if item['status'] != 'fresh']
    latest_evidence_timestamp = max(
        (item['latest_evidence_timestamp'] for item in freshness if item['latest_evidence_timestamp']),
        default=None,
    )

    matrix_corpus_coverage = matrix_report.get('corpus_coverage', {})
    cutover_checkpoints = [
        check(
            'single_compare_matches_terminal_state_and_event_sequence',
            compare_report.get('diff', {}).get('state_equal') and compare_report.get('diff', {}).get('event_types_equal'),
            f"trace_id={compare_report.get('trace_id')}",
        ),
        check(
            'matrix_reports_no_state_or_event_sequence_mismatches',
            matrix_report.get('mismatched', 0) == 0,
            f"matched={matrix_report.get('matched')} mismatched={matrix_report.get('mismatched')}",
        ),
        check(
            'scorecard_detects_no_parity_drift',
            drift_detected_count == 0,
            f"parity_ok={parity_ok_count} drift_detected={drift_detected_count}",
        ),
        check(
            'checked_in_evidence_is_fresh_enough_for_review',
            not stale_inputs,
            f"freshness_statuses={[item['status'] for item in freshness]}",
        ),
        check(
            'matrix_includes_corpus_coverage_overlay',
            bool(matrix_corpus_coverage),
            f"corpus_slice_count={matrix_corpus_coverage.get('corpus_slice_count', 0)}",
        ),
    ]

    limitations = [
        'repo-native only: this scorecard summarizes checked-in shadow artifacts rather than an always-on production traffic mirror',
        'parity drift is measured from fixture-backed compare/matrix runs and optional corpus slices, not mirrored tenant traffic',
        'freshness is derived from the latest artifact event timestamps and should be treated as evidence recency, not live service health',
    ]
    future_work = [
        'replace offline fixture submission with a real ingress mirror or tenant-scoped shadow routing control before treating this as cutover-proof traffic parity',
        'promote parity drift review from checked-in artifacts into a continuously refreshed operational surface',
        'pair this scorecard with rollback automation only after tenant-scoped rollback guardrails exist',
    ]

    return {
        'generated_at': utc_iso(generated_at),
        'ticket': 'BIG-PAR-092',
        'title': 'Live shadow mirror parity drift scorecard',
        'status': 'repo-native-live-shadow-scorecard',
        'evidence_inputs': {
            'shadow_compare_report_path': shadow_compare_report_path,
            'shadow_matrix_report_path': shadow_matrix_report_path,
            'generator_script': 'bigclaw-go/scripts/migration/live_shadow_scorecard',
        },
        'summary': {
            'total_evidence_runs': len(parity_entries),
            'parity_ok_count': parity_ok_count,
            'drift_detected_count': drift_detected_count,
            'matrix_total': matrix_report.get('total', 0),
            'matrix_matched': matrix_report.get('matched', 0),
            'matrix_mismatched': matrix_report.get('mismatched', 0),
            'corpus_coverage_present': bool(matrix_corpus_coverage),
            'corpus_uncovered_slice_count': matrix_corpus_coverage.get('uncovered_corpus_slice_count'),
            'latest_evidence_timestamp': latest_evidence_timestamp,
            'fresh_inputs': sum(1 for item in freshness if item['status'] == 'fresh'),
            'stale_inputs': len(stale_inputs),
        },
        'freshness': freshness,
        'parity_entries': parity_entries,
        'cutover_checkpoints': cutover_checkpoints,
        'limitations': limitations,
        'future_work': future_work,
    }


def main():
    parser = argparse.ArgumentParser(description='Generate a repo-native live shadow mirror parity drift scorecard')
    parser.add_argument('--shadow-compare-report', default='bigclaw-go/docs/reports/shadow-compare-report.json')
    parser.add_argument('--shadow-matrix-report', default='bigclaw-go/docs/reports/shadow-matrix-report.json')
    parser.add_argument('--output', default='bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json')
    parser.add_argument('--pretty', action='store_true')
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[3]
    report = build_report(
        shadow_compare_report_path=args.shadow_compare_report,
        shadow_matrix_report_path=args.shadow_matrix_report,
    )
    output_path = resolve_repo_path(repo_root, args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
    if args.pretty:
        print(json.dumps(report, indent=2))


if __name__ == '__main__':
    main()
