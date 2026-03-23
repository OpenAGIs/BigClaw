#!/usr/bin/env python3
import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


SHADOW_COMPARE_REPORT = 'docs/reports/shadow-compare-report.json'
SHADOW_MATRIX_REPORT = 'docs/reports/shadow-matrix-report.json'
SHADOW_SCORECARD_REPORT = 'docs/reports/live-shadow-mirror-scorecard.json'
ROLLBACK_TRIGGER_REPORT = 'docs/reports/rollback-trigger-surface.json'
FOLLOWUP_INDEX_PATH = 'docs/reports/parallel-follow-up-index.md'
VALIDATION_MATRIX_PATH = 'docs/reports/parallel-validation-matrix.md'
DOC_LINKS = [
    ('docs/migration-shadow.md', 'Shadow helper workflow and bundle generation steps.'),
    ('docs/reports/migration-readiness-report.md', 'Migration readiness summary linked to the shadow bundle.'),
    ('docs/reports/migration-plan-review-notes.md', 'Review notes tied to the shadow bundle index.'),
    ('docs/reports/rollback-trigger-surface.json', 'Machine-readable rollback blockers, warnings, and manual-only paths linked from the shadow bundle.'),
]


def read_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding='utf-8'))


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def relpath(path: Path, root: Path) -> str:
    try:
        return str(path.relative_to(root))
    except ValueError:
        return str(path)


def parse_time(value: str) -> datetime:
    return datetime.fromisoformat(str(value).replace('Z', '+00:00'))


def utc_iso(moment: datetime) -> str:
    return moment.astimezone(timezone.utc).isoformat().replace('+00:00', 'Z')


def copy_json_artifact(source: Path, destination: Path) -> str:
    payload = read_json(source)
    write_json(destination, payload)
    return str(destination)


def resolve_go_root(value: str) -> Path:
    requested = Path(value)
    if requested.is_absolute():
        return requested

    candidate = requested.resolve()
    if candidate.exists():
        return candidate

    cwd = Path.cwd().resolve()
    if cwd.name == requested.name and (cwd / 'docs').exists():
        return cwd

    return candidate


def severity_rank(value: str) -> int:
    order = {
        'critical': 4,
        'high': 3,
        'medium': 2,
        'low': 1,
        'none': 0,
    }
    return order.get(value, 0)


def classify_severity(scorecard: dict[str, Any]) -> str:
    summary = scorecard.get('summary', {})
    if int(summary.get('stale_inputs', 0) or 0) > 0:
        return 'high'
    if int(summary.get('drift_detected_count', 0) or 0) > 0:
        return 'medium'
    if any(not checkpoint.get('passed', False) for checkpoint in scorecard.get('cutover_checkpoints', [])):
        return 'low'
    return 'none'


def build_run_summary(
    *,
    root: Path,
    bundle_dir: Path,
    run_id: str,
    compare_report: dict[str, Any],
    matrix_report: dict[str, Any],
    scorecard_report: dict[str, Any],
    generated_at: datetime,
) -> dict[str, Any]:
    compare_bundle_path = Path(copy_json_artifact(root / SHADOW_COMPARE_REPORT, bundle_dir / 'shadow-compare-report.json'))
    matrix_bundle_path = Path(copy_json_artifact(root / SHADOW_MATRIX_REPORT, bundle_dir / 'shadow-matrix-report.json'))
    scorecard_bundle_path = Path(copy_json_artifact(root / SHADOW_SCORECARD_REPORT, bundle_dir / 'live-shadow-mirror-scorecard.json'))
    rollback_bundle_path = Path(copy_json_artifact(root / ROLLBACK_TRIGGER_REPORT, bundle_dir / 'rollback-trigger-surface.json'))
    rollback_report = read_json(root / ROLLBACK_TRIGGER_REPORT)

    scorecard_summary = scorecard_report.get('summary', {})
    freshness = scorecard_report.get('freshness', [])
    stale_inputs = int(scorecard_summary.get('stale_inputs', 0) or 0)
    drift_detected_count = int(scorecard_summary.get('drift_detected_count', 0) or 0)
    severity = classify_severity(scorecard_report)
    status = 'attention-needed' if severity_rank(severity) > 0 else 'parity-ok'

    return {
        'run_id': run_id,
        'generated_at': utc_iso(generated_at),
        'status': status,
        'severity': severity,
        'bundle_path': relpath(bundle_dir, root),
        'summary_path': relpath(bundle_dir / 'summary.json', root),
        'artifacts': {
            'shadow_compare_report_path': relpath(compare_bundle_path, root),
            'shadow_matrix_report_path': relpath(matrix_bundle_path, root),
            'live_shadow_scorecard_path': relpath(scorecard_bundle_path, root),
            'rollback_trigger_surface_path': relpath(rollback_bundle_path, root),
        },
        'latest_evidence_timestamp': scorecard_summary.get('latest_evidence_timestamp'),
        'freshness': freshness,
        'summary': {
            'total_evidence_runs': int(scorecard_summary.get('total_evidence_runs', 0) or 0),
            'parity_ok_count': int(scorecard_summary.get('parity_ok_count', 0) or 0),
            'drift_detected_count': drift_detected_count,
            'matrix_total': int(scorecard_summary.get('matrix_total', 0) or 0),
            'matrix_mismatched': int(scorecard_summary.get('matrix_mismatched', 0) or 0),
            'stale_inputs': stale_inputs,
            'fresh_inputs': int(scorecard_summary.get('fresh_inputs', 0) or 0),
        },
        'rollback_trigger_surface': {
            'status': rollback_report.get('summary', {}).get('status'),
            'automation_boundary': rollback_report.get('summary', {}).get('automation_boundary'),
            'automated_rollback_trigger': bool(rollback_report.get('summary', {}).get('automated_rollback_trigger', False)),
            'distinctions': rollback_report.get('summary', {}).get('distinctions', {}),
            'summary_path': relpath(root / ROLLBACK_TRIGGER_REPORT, root),
        },
        'compare_trace_id': compare_report.get('trace_id'),
        'matrix_trace_ids': [item.get('trace_id') for item in matrix_report.get('results', []) if item.get('trace_id')],
        'cutover_checkpoints': scorecard_report.get('cutover_checkpoints', []),
        'closeout_commands': [
            'cd bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --pretty',
            'cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py',
            'cd bigclaw-go && go test ./internal/regression -run TestRollbackDocsStayAligned',
            'git push origin <branch> && git log -1 --stat',
        ],
    }


def load_recent_runs(bundle_root: Path) -> list[dict[str, Any]]:
    recent_runs: list[dict[str, Any]] = []
    if not bundle_root.exists():
        return recent_runs
    for child in bundle_root.iterdir():
        summary_path = child / 'summary.json'
        if not child.is_dir() or not summary_path.exists():
            continue
        recent_runs.append(read_json(summary_path))
    recent_runs.sort(key=lambda item: str(item.get('generated_at', '')), reverse=True)
    return recent_runs


def build_rollup(recent_runs: list[dict[str, Any]], limit: int = 5) -> dict[str, Any]:
    window = recent_runs[:limit]
    highest_severity = 'none'
    status_counts = {'parity_ok': 0, 'attention_needed': 0}
    stale_runs = 0
    drift_detected_runs = 0

    entries: list[dict[str, Any]] = []
    for item in window:
        severity = str(item.get('severity', 'none'))
        if severity_rank(severity) > severity_rank(highest_severity):
            highest_severity = severity
        if item.get('status') == 'parity-ok':
            status_counts['parity_ok'] += 1
        else:
            status_counts['attention_needed'] += 1
        summary = item.get('summary', {})
        stale_inputs = int(summary.get('stale_inputs', 0) or 0)
        drift_detected_count = int(summary.get('drift_detected_count', 0) or 0)
        if stale_inputs > 0:
            stale_runs += 1
        if drift_detected_count > 0:
            drift_detected_runs += 1
        entries.append(
            {
                'run_id': item.get('run_id'),
                'generated_at': item.get('generated_at'),
                'status': item.get('status'),
                'severity': severity,
                'latest_evidence_timestamp': item.get('latest_evidence_timestamp'),
                'drift_detected_count': drift_detected_count,
                'stale_inputs': stale_inputs,
                'bundle_path': item.get('bundle_path'),
                'summary_path': item.get('summary_path'),
            }
        )

    overall_status = 'attention-needed' if severity_rank(highest_severity) > 0 else 'parity-ok'
    return {
        'generated_at': utc_iso(datetime.now(timezone.utc)),
        'status': overall_status,
        'window_size': limit,
        'summary': {
            'recent_run_count': len(window),
            'drift_detected_runs': drift_detected_runs,
            'stale_runs': stale_runs,
            'highest_severity': highest_severity,
            'status_counts': status_counts,
            'latest_run_id': window[0].get('run_id') if window else None,
        },
        'recent_runs': entries,
    }


def render_index(
    latest: dict[str, Any],
    recent_runs: list[dict[str, Any]],
    rollup: dict[str, Any],
) -> str:
    lines = [
        '# Live Shadow Mirror Index',
        '',
        f"- Latest run: `{latest['run_id']}`",
        f"- Generated at: `{latest['generated_at']}`",
        f"- Status: `{latest['status']}`",
        f"- Severity: `{latest['severity']}`",
        f"- Bundle: `{latest['bundle_path']}`",
        f"- Summary JSON: `{latest['summary_path']}`",
        '',
        '## Latest bundle artifacts',
        '',
    ]
    for key, label in [
        ('shadow_compare_report_path', 'Shadow compare report'),
        ('shadow_matrix_report_path', 'Shadow matrix report'),
        ('live_shadow_scorecard_path', 'Parity scorecard'),
        ('rollback_trigger_surface_path', 'Rollback trigger surface'),
    ]:
        lines.append(f"- {label}: `{latest['artifacts'][key]}`")
    lines.append('')
    lines.extend(['## Latest run summary', ''])
    lines.append(f"- Compare trace: `{latest['compare_trace_id']}`")
    lines.append(f"- Matrix trace count: `{len(latest['matrix_trace_ids'])}`")
    for key, label in [
        ('total_evidence_runs', 'Evidence runs'),
        ('parity_ok_count', 'Parity-ok entries'),
        ('drift_detected_count', 'Drift-detected entries'),
        ('matrix_total', 'Matrix total'),
        ('matrix_mismatched', 'Matrix mismatched'),
        ('fresh_inputs', 'Fresh inputs'),
        ('stale_inputs', 'Stale inputs'),
    ]:
        lines.append(f"- {label}: `{latest['summary'][key]}`")
    lines.append(f"- Rollback trigger surface status: `{latest['rollback_trigger_surface']['status']}`")
    lines.append(f"- Rollback automation boundary: `{latest['rollback_trigger_surface']['automation_boundary']}`")
    lines.append(
        f"- Rollback trigger distinctions: `{latest['rollback_trigger_surface']['distinctions']}`"
    )
    lines.append('')
    lines.extend(['## Parity drift rollup', ''])
    lines.append(f"- Status: `{rollup['status']}`")
    lines.append(f"- Latest run: `{rollup['summary']['latest_run_id']}`")
    lines.append(f"- Highest severity: `{rollup['summary']['highest_severity']}`")
    lines.append(f"- Drift-detected runs in window: `{rollup['summary']['drift_detected_runs']}`")
    lines.append(f"- Stale runs in window: `{rollup['summary']['stale_runs']}`")
    lines.append('')
    lines.extend(['## Workflow closeout commands', ''])
    for command in latest['closeout_commands']:
        lines.append(f"- `{command}`")
    lines.append('')
    lines.extend(['## Recent bundles', ''])
    for item in recent_runs:
        lines.append(
            f"- `{item['run_id']}` · `{item['status']}` · `{item['severity']}` · `{item['generated_at']}` · `{item['bundle_path']}`"
        )
    lines.append('')
    lines.extend(['## Linked migration docs', ''])
    for path, description in DOC_LINKS:
        lines.append(f"- `{path}` {description}")
    lines.append('')
    lines.extend(['## Parallel Follow-up Index', ''])
    lines.append(
        f"- `{FOLLOWUP_INDEX_PATH}` is the canonical index for the remaining live-shadow, rollback, and corpus-coverage follow-up digests."
    )
    lines.append(
        f"- Use `{VALIDATION_MATRIX_PATH}` first when a shadow review needs the checked-in local/Kubernetes/Ray validation entrypoint alongside the shadow evidence bundle."
    )
    lines.append('')
    return '\n'.join(lines)


def render_bundle_readme(
    latest: dict[str, Any],
    recent_runs: list[dict[str, Any]],
    rollup: dict[str, Any],
) -> str:
    lines = render_index(latest, recent_runs, rollup).splitlines()
    lines.extend(
        [
            '## Primary caveat tracks',
            '',
            f"- `{FOLLOWUP_INDEX_PATH}` is the canonical index for the remaining live-shadow, rollback, and corpus-coverage follow-up digests behind this run bundle.",
            '- For the two primary caveat tracks referenced by this bundle, see',
            '  `OPE-266` / `BIG-PAR-092` in',
            '  `docs/reports/live-shadow-comparison-follow-up-digest.md` and',
            '  `OPE-254` / `BIG-PAR-088` in',
            '  `docs/reports/rollback-safeguard-follow-up-digest.md`.',
            '',
        ]
    )
    lines.append('')
    return '\n'.join(lines)


def derive_run_id(scorecard_report: dict[str, Any], generated_at: datetime) -> str:
    latest_evidence_timestamp = scorecard_report.get('summary', {}).get('latest_evidence_timestamp')
    if latest_evidence_timestamp:
        latest_evidence = parse_time(str(latest_evidence_timestamp))
        return latest_evidence.astimezone(timezone.utc).strftime('%Y%m%dT%H%M%SZ')
    return generated_at.astimezone(timezone.utc).strftime('%Y%m%dT%H%M%SZ')


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Export live shadow mirror evidence into a repo-native bundle')
    parser.add_argument('--go-root', default='bigclaw-go')
    parser.add_argument('--shadow-compare-report', default=SHADOW_COMPARE_REPORT)
    parser.add_argument('--shadow-matrix-report', default=SHADOW_MATRIX_REPORT)
    parser.add_argument('--scorecard-report', default=SHADOW_SCORECARD_REPORT)
    parser.add_argument('--bundle-root', default='docs/reports/live-shadow-runs')
    parser.add_argument('--summary-path', default='docs/reports/live-shadow-summary.json')
    parser.add_argument('--index-path', default='docs/reports/live-shadow-index.md')
    parser.add_argument('--manifest-path', default='docs/reports/live-shadow-index.json')
    parser.add_argument('--rollup-path', default='docs/reports/live-shadow-drift-rollup.json')
    parser.add_argument('--run-id', default='')
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = resolve_go_root(args.go_root)
    compare_report = read_json(root / args.shadow_compare_report)
    matrix_report = read_json(root / args.shadow_matrix_report)
    scorecard_report = read_json(root / args.scorecard_report)
    generated_at = datetime.now(timezone.utc)

    run_id = args.run_id or derive_run_id(scorecard_report, generated_at)
    bundle_dir = root / args.bundle_root / run_id
    bundle_dir.mkdir(parents=True, exist_ok=True)

    latest = build_run_summary(
        root=root,
        bundle_dir=bundle_dir,
        run_id=run_id,
        compare_report=compare_report,
        matrix_report=matrix_report,
        scorecard_report=scorecard_report,
        generated_at=generated_at,
    )
    write_json(bundle_dir / 'summary.json', latest)
    write_json(root / args.summary_path, latest)

    recent_runs = load_recent_runs(root / args.bundle_root)
    rollup = build_rollup(recent_runs)
    manifest = {
        'latest': latest,
        'recent_runs': [
            {
                'run_id': item.get('run_id'),
                'generated_at': item.get('generated_at'),
                'status': item.get('status'),
                'severity': item.get('severity'),
                'bundle_path': item.get('bundle_path'),
                'summary_path': item.get('summary_path'),
            }
            for item in recent_runs
        ],
        'drift_rollup': rollup,
    }
    write_json(root / args.rollup_path, rollup)
    write_json(root / args.manifest_path, manifest)

    index_text = render_index(latest, manifest['recent_runs'], rollup)
    (root / args.index_path).write_text(index_text, encoding='utf-8')
    readme_text = render_bundle_readme(latest, manifest['recent_runs'], rollup)
    (bundle_dir / 'README.md').write_text(readme_text, encoding='utf-8')

    print(json.dumps(manifest, ensure_ascii=False, indent=2))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())
