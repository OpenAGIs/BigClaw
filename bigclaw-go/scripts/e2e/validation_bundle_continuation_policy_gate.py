#!/usr/bin/env python3
import argparse
import json
import pathlib
from datetime import datetime, timezone


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


def build_check(name, passed, detail):
    return {'name': name, 'passed': passed, 'detail': detail}


def normalize_enforcement_mode(enforcement_mode, legacy_enforce_continuation_gate=False):
    mode = str(enforcement_mode or '').strip().lower()
    if not mode:
        mode = 'fail' if legacy_enforce_continuation_gate else 'hold'
    if mode not in {'review', 'hold', 'fail'}:
        raise ValueError(
            f"unsupported enforcement mode {enforcement_mode!r}; expected one of review, hold, fail"
        )
    return mode


def build_enforcement_summary(recommendation, enforcement_mode):
    if recommendation == 'go':
        return {'mode': enforcement_mode, 'outcome': 'pass', 'exit_code': 0}
    if enforcement_mode == 'review':
        return {'mode': enforcement_mode, 'outcome': 'review-only', 'exit_code': 0}
    if enforcement_mode == 'hold':
        return {'mode': enforcement_mode, 'outcome': 'hold', 'exit_code': 2}
    return {'mode': enforcement_mode, 'outcome': 'fail', 'exit_code': 1}


def build_report(
    scorecard_path='bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json',
    max_latest_age_hours=72.0,
    min_recent_bundles=2,
    require_repeated_lane_coverage=True,
    enforcement_mode=None,
    legacy_enforce_continuation_gate=False,
):
    repo_root = pathlib.Path(__file__).resolve().parents[3]
    scorecard = load_json(resolve_repo_path(repo_root, scorecard_path))
    summary = scorecard['summary']
    shared_queue = scorecard['shared_queue_companion']
    normalized_mode = normalize_enforcement_mode(
        enforcement_mode,
        legacy_enforce_continuation_gate=legacy_enforce_continuation_gate,
    )

    checks = [
        build_check(
            'latest_bundle_age_within_threshold',
            float(summary.get('latest_bundle_age_hours', 0.0)) <= float(max_latest_age_hours),
            f"latest_bundle_age_hours={summary.get('latest_bundle_age_hours')} threshold={max_latest_age_hours}",
        ),
        build_check(
            'recent_bundle_count_meets_floor',
            int(summary.get('recent_bundle_count', 0)) >= int(min_recent_bundles),
            f"recent_bundle_count={summary.get('recent_bundle_count')} floor={min_recent_bundles}",
        ),
        build_check(
            'latest_bundle_all_executor_tracks_succeeded',
            bool(summary.get('latest_all_executor_tracks_succeeded')),
            f"latest_all_executor_tracks_succeeded={summary.get('latest_all_executor_tracks_succeeded')}",
        ),
        build_check(
            'recent_bundle_chain_has_no_failures',
            bool(summary.get('recent_bundle_chain_has_no_failures')),
            f"recent_bundle_chain_has_no_failures={summary.get('recent_bundle_chain_has_no_failures')}",
        ),
        build_check(
            'shared_queue_companion_available',
            bool(shared_queue.get('available')),
            f"cross_node_completions={shared_queue.get('cross_node_completions')}",
        ),
        build_check(
            'repeated_lane_coverage_meets_policy',
            (not require_repeated_lane_coverage) or bool(summary.get('all_executor_tracks_have_repeated_recent_coverage')),
            f"require_repeated_lane_coverage={require_repeated_lane_coverage} actual={summary.get('all_executor_tracks_have_repeated_recent_coverage')}",
        ),
    ]

    failing_checks = [item['name'] for item in checks if not item['passed']]
    recommendation = 'go' if not failing_checks else 'hold'
    enforcement = build_enforcement_summary(recommendation, normalized_mode)
    next_actions = []
    if 'latest_bundle_age_within_threshold' in failing_checks:
        next_actions.append('rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle')
    if 'recent_bundle_count_meets_floor' in failing_checks:
        next_actions.append('export additional validation bundles so the continuation window spans multiple indexed runs')
    if 'shared_queue_companion_available' in failing_checks:
        next_actions.append('rerun `cd bigclaw-go && BIGCLAW_E2E_REFRESH_SHARED_QUEUE=1 ./scripts/e2e/run_all.sh` so the shared-queue companion proof is refreshed with the bundle')
    if 'repeated_lane_coverage_meets_policy' in failing_checks:
        next_actions.append('refresh another full validation bundle with `ray` enabled so each executor lane has repeated indexed coverage')
    if not next_actions:
        next_actions.append('set BIGCLAW_E2E_CONTINUATION_GATE_MODE=fail when workflow closeout should stop on continuation regressions')

    return {
        'generated_at': utc_iso(),
        'ticket': 'OPE-262',
        'title': 'Validation workflow continuation gate',
        'status': 'policy-go' if recommendation == 'go' else 'policy-hold',
        'recommendation': recommendation,
        'evidence_inputs': {
            'scorecard_path': scorecard_path,
            'generator_script': 'scripts/e2e/validation_bundle_continuation_policy_gate.py',
        },
        'policy_inputs': {
            'max_latest_age_hours': float(max_latest_age_hours),
            'min_recent_bundles': int(min_recent_bundles),
            'require_repeated_lane_coverage': bool(require_repeated_lane_coverage),
        },
        'enforcement': enforcement,
        'summary': {
            'latest_run_id': summary.get('latest_run_id'),
            'latest_bundle_age_hours': summary.get('latest_bundle_age_hours'),
            'recent_bundle_count': summary.get('recent_bundle_count'),
            'latest_all_executor_tracks_succeeded': summary.get('latest_all_executor_tracks_succeeded'),
            'recent_bundle_chain_has_no_failures': summary.get('recent_bundle_chain_has_no_failures'),
            'all_executor_tracks_have_repeated_recent_coverage': summary.get('all_executor_tracks_have_repeated_recent_coverage'),
            'recommendation': recommendation,
            'enforcement_mode': enforcement['mode'],
            'workflow_outcome': enforcement['outcome'],
            'workflow_exit_code': enforcement['exit_code'],
            'passing_check_count': len([item for item in checks if item['passed']]),
            'failing_check_count': len(failing_checks),
        },
        'policy_checks': checks,
        'failing_checks': failing_checks,
        'reviewer_path': {
            'index_path': 'docs/reports/live-validation-index.md',
            'digest_path': 'docs/reports/validation-bundle-continuation-digest.md',
        },
        'shared_queue_companion': shared_queue,
        'next_actions': next_actions,
    }


def main():
    parser = argparse.ArgumentParser(description='Evaluate validation bundle continuation policy checks')
    parser.add_argument('--scorecard', default='bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json')
    parser.add_argument('--output', default='bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json')
    parser.add_argument('--max-latest-age-hours', type=float, default=72.0)
    parser.add_argument('--min-recent-bundles', type=int, default=2)
    parser.add_argument('--require-repeated-lane-coverage', action='store_true', default=True)
    parser.add_argument('--allow-partial-lane-history', action='store_true')
    parser.add_argument('--enforcement-mode', choices=('review', 'hold', 'fail'))
    parser.add_argument('--enforce', action='store_true')
    parser.add_argument('--pretty', action='store_true')
    args = parser.parse_args()

    repo_root = pathlib.Path(__file__).resolve().parents[3]
    report = build_report(
        scorecard_path=args.scorecard,
        max_latest_age_hours=args.max_latest_age_hours,
        min_recent_bundles=args.min_recent_bundles,
        require_repeated_lane_coverage=not args.allow_partial_lane_history,
        enforcement_mode=args.enforcement_mode,
        legacy_enforce_continuation_gate=args.enforce,
    )
    output_path = resolve_repo_path(repo_root, args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(report, indent=2) + '\n', encoding='utf-8')
    if args.pretty:
        print(json.dumps(report, indent=2))
    return int(report['enforcement']['exit_code'])


if __name__ == '__main__':
    raise SystemExit(main())
