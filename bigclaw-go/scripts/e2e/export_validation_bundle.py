#!/usr/bin/env python3
from __future__ import annotations
import argparse
import json
import shutil
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Optional


LATEST_REPORTS = {
    'local': 'docs/reports/sqlite-smoke-report.json',
    'kubernetes': 'docs/reports/kubernetes-live-smoke-report.json',
    'ray': 'docs/reports/ray-live-smoke-report.json',
}
BROKER_SUMMARY = 'docs/reports/broker-validation-summary.json'
BROKER_BOOTSTRAP_SUMMARY = 'docs/reports/broker-bootstrap-review-summary.json'
BROKER_VALIDATION_PACK = 'docs/reports/broker-failover-fault-injection-validation-pack.md'
SHARED_QUEUE_REPORT = 'docs/reports/multi-node-shared-queue-report.json'
SHARED_QUEUE_SUMMARY = 'docs/reports/shared-queue-companion-summary.json'

CONTINUATION_ARTIFACTS = [
    (
        'docs/reports/validation-bundle-continuation-scorecard.json',
        'summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof.',
    ),
    (
        'docs/reports/validation-bundle-continuation-policy-gate.json',
        'records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability.',
    ),
]

FOLLOWUP_DIGESTS = [
    (
        'docs/reports/validation-bundle-continuation-digest.md',
        'Validation bundle continuation caveats are consolidated here.',
    ),
]

LANE_ALIASES = {
    'local': 'local',
    'kubernetes': 'k8s',
    'ray': 'ray',
}

FAILURE_EVENT_TYPES = {
    'task.cancelled',
    'task.dead_letter',
    'task.failed',
    'task.retried',
}


def build_continuation_gate_summary(root: Path) -> Optional[dict[str, Any]]:
    gate_path = root / 'docs/reports/validation-bundle-continuation-policy-gate.json'
    gate = read_json(gate_path)
    if not isinstance(gate, dict):
        return None
    enforcement = gate.get('enforcement')
    summary = gate.get('summary')
    reviewer_path = gate.get('reviewer_path')
    next_actions = gate.get('next_actions')
    return {
        'path': relpath(gate_path, root),
        'status': gate.get('status', 'unknown'),
        'recommendation': gate.get('recommendation', 'unknown'),
        'failing_checks': gate.get('failing_checks', []),
        'enforcement': enforcement if isinstance(enforcement, dict) else {},
        'summary': summary if isinstance(summary, dict) else {},
        'reviewer_path': reviewer_path if isinstance(reviewer_path, dict) else {},
        'next_actions': next_actions if isinstance(next_actions, list) else [],
    }


def read_json(path: Path) -> Optional[Any]:
    if not path.exists() or path.stat().st_size == 0:
        return None
    return json.loads(path.read_text(encoding='utf-8'))


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')


def relpath(path: Path, root: Path) -> str:
    try:
        return str(path.relative_to(root))
    except ValueError:
        return str(path)


def copy_text_artifact(source: Path, destination: Path) -> str:
    if not source.exists():
        return ''
    if source.resolve() == destination.resolve():
        return str(destination)
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, destination)
    return str(destination)


def copy_json_artifact(source: Path, destination: Path) -> str:
    if not source.exists():
        return ''
    if source.resolve() == destination.resolve():
        return str(destination)
    payload = read_json(source)
    if payload is None:
        return ''
    write_json(destination, payload)
    return str(destination)


def first_text(*values: Any) -> str:
    for value in values:
        if isinstance(value, str):
            stripped = value.strip()
            if stripped:
                return stripped
    return ''


def collect_report_events(report: dict[str, Any]) -> list[dict[str, Any]]:
    events: list[dict[str, Any]] = []
    status = report.get('status')
    if isinstance(status, dict):
        status_events = status.get('events')
        if isinstance(status_events, list):
            events.extend(event for event in status_events if isinstance(event, dict))
        latest_event = status.get('latest_event')
        if isinstance(latest_event, dict):
            latest_id = latest_event.get('id')
            if latest_id and any(event.get('id') == latest_id for event in events):
                return events
            events.append(latest_event)
    report_events = report.get('events')
    if isinstance(report_events, list):
        for event in report_events:
            if isinstance(event, dict):
                event_id = event.get('id')
                if event_id and any(existing.get('id') == event_id for existing in events):
                    continue
                events.append(event)
    return events


def event_payload_text(event: dict[str, Any], key: str) -> str:
    payload = event.get('payload')
    if not isinstance(payload, dict):
        return ''
    return first_text(payload.get(key))


def latest_report_event(report: dict[str, Any]) -> Optional[dict[str, Any]]:
    events = collect_report_events(report)
    for event in reversed(events):
        return event
    return None


def find_routing_reason(report: dict[str, Any]) -> str:
    events = collect_report_events(report)
    for event in reversed(events):
        if event.get('type') == 'scheduler.routed':
            return first_text(event_payload_text(event, 'reason'))
    return ''


def build_failure_root_cause(section: dict[str, Any], report: dict[str, Any]) -> dict[str, Any]:
    events = collect_report_events(report)
    latest_event = latest_report_event(report)
    latest_status = first_text(
        report.get('status', {}).get('state') if isinstance(report.get('status'), dict) else None,
        report.get('task', {}).get('state') if isinstance(report.get('task'), dict) else None,
        component_status(report),
    )

    cause_event: Optional[dict[str, Any]] = None
    for event in reversed(events):
        if first_text(event.get('type')) in FAILURE_EVENT_TYPES:
            cause_event = event
            break
    if cause_event is None and latest_status not in {'', 'succeeded'}:
        cause_event = latest_event

    location = first_text(
        section.get('stderr_path'),
        section.get('service_log_path'),
        section.get('audit_log_path'),
        section.get('bundle_report_path'),
    )
    if cause_event is None:
        return {
            'status': 'not_triggered',
            'event_type': first_text(latest_event.get('type')) if isinstance(latest_event, dict) else '',
            'message': '',
            'location': location,
            'event_id': '',
            'timestamp': '',
        }

    return {
        'status': 'captured',
        'event_type': first_text(cause_event.get('type')),
        'message': first_text(
            event_payload_text(cause_event, 'message'),
            event_payload_text(cause_event, 'reason'),
            report.get('error'),
            report.get('failure_reason'),
        ),
        'location': location,
        'event_id': first_text(cause_event.get('id')),
        'timestamp': first_text(cause_event.get('timestamp')),
    }


def build_validation_matrix_entry(name: str, section: dict[str, Any], report: Optional[dict[str, Any]]) -> dict[str, Any]:
    task = report.get('task') if isinstance(report, dict) else None
    task_id = task.get('id') if isinstance(task, dict) else section.get('task_id')
    executor = task.get('required_executor') if isinstance(task, dict) else None
    if not isinstance(executor, str) or not executor.strip():
        executor = name
    root_cause = section.get('failure_root_cause')
    return {
        'lane': LANE_ALIASES.get(name, name),
        'executor': executor,
        'enabled': section.get('enabled', False),
        'status': section.get('status', 'unknown'),
        'task_id': task_id,
        'canonical_report_path': section.get('canonical_report_path', ''),
        'bundle_report_path': section.get('bundle_report_path', ''),
        'latest_event_type': section.get('latest_event_type', ''),
        'routing_reason': section.get('routing_reason', ''),
        'root_cause_event_type': root_cause.get('event_type', '') if isinstance(root_cause, dict) else '',
        'root_cause_location': root_cause.get('location', '') if isinstance(root_cause, dict) else '',
        'root_cause_message': root_cause.get('message', '') if isinstance(root_cause, dict) else '',
    }


def build_validation_matrix(summary: dict[str, Any]) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    for name in ('local', 'kubernetes', 'ray'):
        section = summary.get(name)
        if not isinstance(section, dict):
            continue
        row = section.get('validation_matrix')
        if isinstance(row, dict):
            rows.append(row)
    return rows


def component_status(report: Optional[dict[str, Any]]) -> str:
    if not report or not isinstance(report, dict):
        return 'missing_report'
    status = report.get('status')
    if isinstance(status, dict):
        return str(status.get('state', 'unknown'))
    if isinstance(status, str):
        return status
    if report.get('all_ok') is True:
        return 'succeeded'
    if report.get('all_ok') is False:
        return 'failed'
    return 'unknown'


def build_shared_queue_companion(root: Path, bundle_dir: Path) -> dict[str, Any]:
    canonical_report_path = root / SHARED_QUEUE_REPORT
    canonical_summary_path = root / SHARED_QUEUE_SUMMARY
    bundle_report_path = bundle_dir / 'multi-node-shared-queue-report.json'
    bundle_summary_path = bundle_dir / 'shared-queue-companion-summary.json'
    report = read_json(canonical_report_path)

    summary: dict[str, Any] = {
        'available': isinstance(report, dict),
        'canonical_report_path': SHARED_QUEUE_REPORT,
        'canonical_summary_path': SHARED_QUEUE_SUMMARY,
        'bundle_report_path': relpath(bundle_report_path, root),
        'bundle_summary_path': relpath(bundle_summary_path, root),
    }
    if not isinstance(report, dict):
        summary['status'] = 'missing_report'
        return summary

    copied_report = copy_json_artifact(canonical_report_path, bundle_report_path)
    if copied_report:
        summary['bundle_report_path'] = relpath(Path(copied_report), root)

    summary.update(
        {
            'status': 'succeeded' if report.get('all_ok') else 'failed',
            'generated_at': report.get('generated_at'),
            'count': report.get('count'),
            'cross_node_completions': report.get('cross_node_completions'),
            'duplicate_started_tasks': len(report.get('duplicate_started_tasks') or []),
            'duplicate_completed_tasks': len(report.get('duplicate_completed_tasks') or []),
            'missing_completed_tasks': len(report.get('missing_completed_tasks') or []),
            'submitted_by_node': report.get('submitted_by_node', {}),
            'completed_by_node': report.get('completed_by_node', {}),
            'nodes': [node.get('name') for node in report.get('nodes', []) if isinstance(node, dict) and node.get('name')],
        }
    )
    write_json(bundle_summary_path, summary)
    write_json(canonical_summary_path, summary)
    return summary


def build_component_section(
    *,
    name: str,
    enabled: bool,
    root: Path,
    bundle_dir: Path,
    report_path: Path,
    stdout_path: Path,
    stderr_path: Path,
) -> dict[str, Any]:
    latest_report_path = root / LATEST_REPORTS[name]
    section: dict[str, Any] = {
        'enabled': enabled,
        'bundle_report_path': relpath(report_path, root),
        'canonical_report_path': LATEST_REPORTS[name],
    }
    if not enabled:
        section['status'] = 'skipped'
        return section

    report = read_json(report_path)
    section['report'] = report
    section['status'] = component_status(report)

    copied_latest = copy_json_artifact(report_path, latest_report_path)
    if copied_latest:
        section['canonical_report_path'] = relpath(Path(copied_latest), root)

    stdout_copy = copy_text_artifact(stdout_path, bundle_dir / f'{name}.stdout.log')
    stderr_copy = copy_text_artifact(stderr_path, bundle_dir / f'{name}.stderr.log')
    if stdout_copy:
        section['stdout_path'] = relpath(Path(stdout_copy), root)
    if stderr_copy:
        section['stderr_path'] = relpath(Path(stderr_copy), root)

    if isinstance(report, dict):
        task = report.get('task')
        if isinstance(task, dict) and task.get('id'):
            section['task_id'] = task['id']
        base_url = report.get('base_url')
        if base_url:
            section['base_url'] = base_url
        state_dir = report.get('state_dir')
        if state_dir:
            section['state_dir'] = state_dir
            audit_source = Path(state_dir) / 'audit.jsonl'
            audit_copy = copy_text_artifact(audit_source, bundle_dir / f'{name}.audit.jsonl')
            if audit_copy:
                section['audit_log_path'] = relpath(Path(audit_copy), root)
        service_log = report.get('service_log')
        if service_log:
            service_copy = copy_text_artifact(Path(service_log), bundle_dir / f'{name}.service.log')
            if service_copy:
                section['service_log_path'] = relpath(Path(service_copy), root)
        latest_event = latest_report_event(report)
        if isinstance(latest_event, dict):
            section['latest_event_type'] = first_text(latest_event.get('type'))
            section['latest_event_timestamp'] = first_text(latest_event.get('timestamp'))
            artifacts = latest_event.get('payload', {}).get('artifacts') if isinstance(latest_event.get('payload'), dict) else None
            if isinstance(artifacts, list):
                section['artifact_paths'] = [artifact for artifact in artifacts if isinstance(artifact, str)]
        routing_reason = find_routing_reason(report)
        if routing_reason:
            section['routing_reason'] = routing_reason
        section['failure_root_cause'] = build_failure_root_cause(section, report)
        section['validation_matrix'] = build_validation_matrix_entry(name, section, report)
    else:
        section['failure_root_cause'] = {
            'status': 'missing_report',
            'event_type': '',
            'message': '',
            'location': section['bundle_report_path'],
            'event_id': '',
            'timestamp': '',
        }
        section['validation_matrix'] = build_validation_matrix_entry(name, section, None)
    return section


def build_broker_section(
    *,
    enabled: bool,
    backend: str,
    root: Path,
    bundle_dir: Path,
    bootstrap_summary_path: Path | None,
    report_path: Path | None,
) -> dict[str, Any]:
    bundle_summary_path = bundle_dir / 'broker-validation-summary.json'
    bundle_bootstrap_summary_path = bundle_dir / 'broker-bootstrap-review-summary.json'
    section: dict[str, Any] = {
        'enabled': enabled,
        'backend': backend or None,
        'bundle_summary_path': relpath(bundle_summary_path, root),
        'canonical_summary_path': BROKER_SUMMARY,
        'bundle_bootstrap_summary_path': relpath(bundle_bootstrap_summary_path, root),
        'canonical_bootstrap_summary_path': BROKER_BOOTSTRAP_SUMMARY,
        'validation_pack_path': BROKER_VALIDATION_PACK,
    }
    configuration_state = 'configured' if enabled and backend else 'not_configured'
    section['configuration_state'] = configuration_state
    bootstrap_summary = read_json(bootstrap_summary_path) if bootstrap_summary_path else None
    if isinstance(bootstrap_summary, dict):
        copied_bootstrap = copy_json_artifact(bootstrap_summary_path, bundle_bootstrap_summary_path)
        if copied_bootstrap:
            section['bundle_bootstrap_summary_path'] = relpath(Path(copied_bootstrap), root)
        canonical_bootstrap = copy_json_artifact(bootstrap_summary_path, root / BROKER_BOOTSTRAP_SUMMARY)
        if canonical_bootstrap:
            section['canonical_bootstrap_summary_path'] = relpath(Path(canonical_bootstrap), root)
        section['bootstrap_summary'] = bootstrap_summary
        section['bootstrap_ready'] = bool(bootstrap_summary.get('ready'))
        section['runtime_posture'] = bootstrap_summary.get('runtime_posture')
        section['live_adapter_implemented'] = bool(bootstrap_summary.get('live_adapter_implemented'))
        section['proof_boundary'] = bootstrap_summary.get('proof_boundary')
        validation_errors = bootstrap_summary.get('validation_errors')
        if isinstance(validation_errors, list):
            section['validation_errors'] = validation_errors
        completeness = bootstrap_summary.get('config_completeness')
        if isinstance(completeness, dict):
            section['config_completeness'] = completeness

    if not enabled or not backend:
        section['status'] = 'skipped'
        section['reason'] = 'not_configured'
        write_json(bundle_summary_path, section)
        write_json(root / BROKER_SUMMARY, section)
        return section

    if report_path is None:
        section['status'] = 'skipped'
        section['reason'] = 'missing_report_path'
        write_json(bundle_summary_path, section)
        write_json(root / BROKER_SUMMARY, section)
        return section

    report = read_json(report_path)
    report_relpath = relpath(report_path, root)
    section['canonical_report_path'] = report_relpath
    section['bundle_report_path'] = relpath(bundle_dir / report_path.name, root)
    if not isinstance(report, dict):
        section['status'] = 'skipped'
        section['reason'] = 'not_configured'
        write_json(bundle_summary_path, section)
        write_json(root / BROKER_SUMMARY, section)
        return section

    copied_report = copy_json_artifact(report_path, bundle_dir / report_path.name)
    if copied_report:
        section['bundle_report_path'] = relpath(Path(copied_report), root)
    section['report'] = report
    section['status'] = component_status(report)
    write_json(bundle_summary_path, section)
    write_json(root / BROKER_SUMMARY, section)
    return section


def build_recent_runs(bundle_root: Path, root: Path, limit: int = 8) -> list[dict[str, Any]]:
    runs: list[tuple[str, dict[str, Any]]] = []
    if not bundle_root.exists():
        return []
    for child in bundle_root.iterdir():
        if not child.is_dir():
            continue
        summary_path = child / 'summary.json'
        summary = read_json(summary_path)
        if isinstance(summary, dict):
            generated_at = str(summary.get('generated_at', ''))
            runs.append((generated_at, summary))
    runs.sort(key=lambda item: item[0], reverse=True)
    items: list[dict[str, Any]] = []
    for _, summary in runs[:limit]:
        items.append(
            {
                'run_id': summary.get('run_id', ''),
                'generated_at': summary.get('generated_at', ''),
                'status': summary.get('status', 'unknown'),
                'bundle_path': summary.get('bundle_path', ''),
                'summary_path': summary.get('summary_path', ''),
            }
        )
    return items


def build_continuation_artifacts(root: Path) -> list[tuple[str, str]]:
    items: list[tuple[str, str]] = []
    for relpath_value, description in CONTINUATION_ARTIFACTS:
        if (root / relpath_value).exists():
            items.append((relpath_value, description))
    return items


def build_followup_digests(root: Path) -> list[tuple[str, str]]:
    items: list[tuple[str, str]] = []
    for relpath_value, description in FOLLOWUP_DIGESTS:
        if (root / relpath_value).exists():
            items.append((relpath_value, description))
    return items


def render_index(
    summary: dict[str, Any],
    recent_runs: list[dict[str, Any]],
    continuation_gate: dict[str, Any] | None = None,
    continuation_artifacts: list[tuple[str, str]] | None = None,
    followup_digests: list[tuple[str, str]] | None = None,
) -> str:
    lines = [
        '# Live Validation Index',
        '',
        f"- Latest run: `{summary['run_id']}`",
        f"- Generated at: `{summary['generated_at']}`",
        f"- Status: `{summary['status']}`",
        f"- Bundle: `{summary['bundle_path']}`",
        f"- Summary JSON: `{summary['summary_path']}`",
        '',
        '## Latest bundle artifacts',
        '',
    ]
    for name in ('local', 'kubernetes', 'ray'):
        section = summary[name]
        lines.append(f"### {name}")
        matrix = section.get('validation_matrix', {})
        lines.append(f"- Enabled: `{section['enabled']}`")
        lines.append(f"- Status: `{section['status']}`")
        if matrix.get('lane'):
            lines.append(f"- Validation lane: `{matrix['lane']}`")
        lines.append(f"- Bundle report: `{section['bundle_report_path']}`")
        lines.append(f"- Latest report: `{section['canonical_report_path']}`")
        if section.get('stdout_path'):
            lines.append(f"- Stdout log: `{section['stdout_path']}`")
        if section.get('stderr_path'):
            lines.append(f"- Stderr log: `{section['stderr_path']}`")
        if section.get('service_log_path'):
            lines.append(f"- Service log: `{section['service_log_path']}`")
        if section.get('audit_log_path'):
            lines.append(f"- Audit log: `{section['audit_log_path']}`")
        if section.get('task_id'):
            lines.append(f"- Task ID: `{section['task_id']}`")
        if section.get('latest_event_type'):
            lines.append(f"- Latest event: `{section['latest_event_type']}`")
        if section.get('routing_reason'):
            lines.append(f"- Routing reason: `{section['routing_reason']}`")
        root_cause = section.get('failure_root_cause')
        if isinstance(root_cause, dict):
            lines.append(
                f"- Failure root cause: status=`{root_cause.get('status', 'unknown')}` "
                f"event=`{root_cause.get('event_type', 'unknown')}` "
                f"location=`{root_cause.get('location', 'n/a')}`"
            )
            if root_cause.get('message'):
                lines.append(f"- Failure detail: `{root_cause['message']}`")
        lines.append('')

    validation_matrix = summary.get('validation_matrix')
    if isinstance(validation_matrix, list) and validation_matrix:
        lines.extend(['## Validation matrix', ''])
        for row in validation_matrix:
            if not isinstance(row, dict):
                continue
            lines.append(
                f"- Lane `{row.get('lane', 'unknown')}` executor=`{row.get('executor', 'unknown')}` "
                f"status=`{row.get('status', 'unknown')}` enabled=`{row.get('enabled', False)}` "
                f"report=`{row.get('bundle_report_path', '')}`"
            )
            if row.get('root_cause_event_type') or row.get('root_cause_message'):
                lines.append(
                    f"- Lane `{row.get('lane', 'unknown')}` root cause: "
                    f"event=`{row.get('root_cause_event_type', 'unknown')}` "
                    f"location=`{row.get('root_cause_location', 'n/a')}` "
                    f"message=`{row.get('root_cause_message', '')}`"
                )
        lines.append('')

    broker = summary.get('broker')
    if isinstance(broker, dict):
        lines.append('### broker')
        lines.append(f"- Enabled: `{broker['enabled']}`")
        lines.append(f"- Status: `{broker['status']}`")
        lines.append(f"- Configuration state: `{broker['configuration_state']}`")
        lines.append(f"- Bundle summary: `{broker['bundle_summary_path']}`")
        lines.append(f"- Canonical summary: `{broker['canonical_summary_path']}`")
        lines.append(f"- Bundle bootstrap summary: `{broker['bundle_bootstrap_summary_path']}`")
        lines.append(f"- Canonical bootstrap summary: `{broker['canonical_bootstrap_summary_path']}`")
        lines.append(f"- Validation pack: `{broker['validation_pack_path']}`")
        if broker.get('backend'):
            lines.append(f"- Backend: `{broker['backend']}`")
        if broker.get('bootstrap_ready') is not None:
            lines.append(f"- Bootstrap ready: `{broker['bootstrap_ready']}`")
        if broker.get('runtime_posture'):
            lines.append(f"- Runtime posture: `{broker['runtime_posture']}`")
        if broker.get('live_adapter_implemented') is not None:
            lines.append(f"- Live adapter implemented: `{broker['live_adapter_implemented']}`")
        completeness = broker.get('config_completeness')
        if isinstance(completeness, dict):
            lines.append(
                "- Config completeness: "
                f"driver=`{completeness.get('driver', False)}` "
                f"urls=`{completeness.get('urls', False)}` "
                f"topic=`{completeness.get('topic', False)}` "
                f"consumer_group=`{completeness.get('consumer_group', False)}`"
            )
        if broker.get('proof_boundary'):
            lines.append(f"- Proof boundary: `{broker['proof_boundary']}`")
        for error in broker.get('validation_errors', []):
            lines.append(f"- Validation error: `{error}`")
        if broker.get('bundle_report_path'):
            lines.append(f"- Bundle report: `{broker['bundle_report_path']}`")
        if broker.get('canonical_report_path'):
            lines.append(f"- Canonical report: `{broker['canonical_report_path']}`")
        if broker.get('reason'):
            lines.append(f"- Reason: `{broker['reason']}`")
        lines.append('')

    shared_queue = summary.get('shared_queue_companion')
    if isinstance(shared_queue, dict):
        lines.append('### shared-queue companion')
        lines.append(f"- Available: `{shared_queue['available']}`")
        lines.append(f"- Status: `{shared_queue['status']}`")
        lines.append(f"- Bundle summary: `{shared_queue['bundle_summary_path']}`")
        lines.append(f"- Canonical summary: `{shared_queue['canonical_summary_path']}`")
        lines.append(f"- Bundle report: `{shared_queue['bundle_report_path']}`")
        lines.append(f"- Canonical report: `{shared_queue['canonical_report_path']}`")
        if shared_queue.get('cross_node_completions') is not None:
            lines.append(f"- Cross-node completions: `{shared_queue['cross_node_completions']}`")
        if shared_queue.get('duplicate_started_tasks') is not None:
            lines.append(f"- Duplicate `task.started`: `{shared_queue['duplicate_started_tasks']}`")
        if shared_queue.get('duplicate_completed_tasks') is not None:
            lines.append(f"- Duplicate `task.completed`: `{shared_queue['duplicate_completed_tasks']}`")
        if shared_queue.get('missing_completed_tasks') is not None:
            lines.append(f"- Missing terminal completions: `{shared_queue['missing_completed_tasks']}`")
        lines.append('')

    lines.extend(['## Workflow closeout commands', ''])
    for command in summary['closeout_commands']:
        lines.append(f'- `{command}`')
    lines.append('')
    lines.extend(['## Recent bundles', ''])
    if not recent_runs:
        lines.append('- No previous bundles found')
    else:
        for run in recent_runs:
            lines.append(
                f"- `{run['run_id']}` · `{run['status']}` · `{run['generated_at']}` · `{run['bundle_path']}`"
            )
    lines.append('')
    if continuation_gate:
        lines.extend(['## Continuation gate', ''])
        lines.append(f"- Status: `{continuation_gate['status']}`")
        lines.append(f"- Recommendation: `{continuation_gate['recommendation']}`")
        lines.append(f"- Report: `{continuation_gate['path']}`")
        enforcement = continuation_gate.get('enforcement', {})
        if enforcement.get('mode'):
            lines.append(f"- Workflow mode: `{enforcement['mode']}`")
        if enforcement.get('outcome'):
            lines.append(f"- Workflow outcome: `{enforcement['outcome']}`")
        gate_summary = continuation_gate.get('summary', {})
        if gate_summary.get('latest_run_id'):
            lines.append(f"- Latest reviewed run: `{gate_summary['latest_run_id']}`")
        if 'failing_check_count' in gate_summary:
            lines.append(f"- Failing checks: `{gate_summary['failing_check_count']}`")
        if 'workflow_exit_code' in gate_summary:
            lines.append(f"- Workflow exit code on current evidence: `{gate_summary['workflow_exit_code']}`")
        reviewer_path = continuation_gate.get('reviewer_path', {})
        if reviewer_path.get('digest_path'):
            lines.append(f"- Reviewer digest: `{reviewer_path['digest_path']}`")
        if reviewer_path.get('index_path'):
            lines.append(f"- Reviewer index: `{reviewer_path['index_path']}`")
        for action in continuation_gate.get('next_actions', []):
            lines.append(f"- Next action: `{action}`")
        lines.append('')
    if continuation_artifacts:
        lines.extend(['## Continuation artifacts', ''])
        for artifact_path, description in continuation_artifacts:
            lines.append(f'- `{artifact_path}` {description}')
        lines.append('')
    if followup_digests:
        lines.extend(['## Parallel follow-up digests', ''])
        for digest_path, description in followup_digests:
            lines.append(f'- `{digest_path}` {description}')
        lines.append('')
    return '\n'.join(lines)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Export live validation evidence into a repo-native bundle')
    parser.add_argument('--go-root', required=True)
    parser.add_argument('--run-id', required=True)
    parser.add_argument('--bundle-dir', required=True)
    parser.add_argument('--summary-path', default='docs/reports/live-validation-summary.json')
    parser.add_argument('--index-path', default='docs/reports/live-validation-index.md')
    parser.add_argument('--manifest-path', default='docs/reports/live-validation-index.json')
    parser.add_argument('--run-local', default='1')
    parser.add_argument('--run-kubernetes', default='1')
    parser.add_argument('--run-ray', default='1')
    parser.add_argument('--validation-status', default='0')
    parser.add_argument('--run-broker', default='0')
    parser.add_argument('--broker-backend', default='')
    parser.add_argument('--broker-report-path', default='')
    parser.add_argument('--broker-bootstrap-summary-path', default='')
    parser.add_argument('--local-report-path', required=True)
    parser.add_argument('--local-stdout-path', required=True)
    parser.add_argument('--local-stderr-path', required=True)
    parser.add_argument('--kubernetes-report-path', required=True)
    parser.add_argument('--kubernetes-stdout-path', required=True)
    parser.add_argument('--kubernetes-stderr-path', required=True)
    parser.add_argument('--ray-report-path', required=True)
    parser.add_argument('--ray-stdout-path', required=True)
    parser.add_argument('--ray-stderr-path', required=True)
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = Path(args.go_root).resolve()
    bundle_dir = (root / args.bundle_dir).resolve()
    bundle_dir.mkdir(parents=True, exist_ok=True)

    summary = {
        'run_id': args.run_id,
        'generated_at': datetime.now(timezone.utc).isoformat(),
        'status': 'succeeded' if args.validation_status == '0' else 'failed',
        'bundle_path': relpath(bundle_dir, root),
        'closeout_commands': [
            'cd bigclaw-go && ./scripts/e2e/run_all.sh',
            'cd bigclaw-go && go test ./...',
            'git push origin <branch> && git log -1 --stat',
        ],
    }
    summary['local'] = build_component_section(
        name='local',
        enabled=args.run_local == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.local_report_path,
        stdout_path=Path(args.local_stdout_path),
        stderr_path=Path(args.local_stderr_path),
    )
    summary['kubernetes'] = build_component_section(
        name='kubernetes',
        enabled=args.run_kubernetes == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.kubernetes_report_path,
        stdout_path=Path(args.kubernetes_stdout_path),
        stderr_path=Path(args.kubernetes_stderr_path),
    )
    summary['ray'] = build_component_section(
        name='ray',
        enabled=args.run_ray == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.ray_report_path,
        stdout_path=Path(args.ray_stdout_path),
        stderr_path=Path(args.ray_stderr_path),
    )
    summary['broker'] = build_broker_section(
        enabled=args.run_broker == '1',
        backend=args.broker_backend.strip(),
        root=root,
        bundle_dir=bundle_dir,
        bootstrap_summary_path=(root / args.broker_bootstrap_summary_path) if args.broker_bootstrap_summary_path else None,
        report_path=(root / args.broker_report_path) if args.broker_report_path else None,
    )
    summary['shared_queue_companion'] = build_shared_queue_companion(root, bundle_dir)
    summary['validation_matrix'] = build_validation_matrix(summary)

    continuation_gate = build_continuation_gate_summary(root)
    if continuation_gate:
        summary['continuation_gate'] = continuation_gate

    bundle_summary_path = bundle_dir / 'summary.json'
    canonical_summary_path = root / args.summary_path
    summary['summary_path'] = relpath(bundle_summary_path, root)
    write_json(bundle_summary_path, summary)
    write_json(canonical_summary_path, summary)

    bundle_root = bundle_dir.parent
    recent_runs = build_recent_runs(bundle_root, root)
    manifest = {'latest': summary, 'recent_runs': recent_runs}
    if continuation_gate:
        manifest['continuation_gate'] = continuation_gate
    write_json(root / args.manifest_path, manifest)

    continuation_artifacts = build_continuation_artifacts(root)
    followup_digests = build_followup_digests(root)
    index_text = render_index(summary, recent_runs, continuation_gate, continuation_artifacts, followup_digests)
    (root / args.index_path).parent.mkdir(parents=True, exist_ok=True)
    (root / args.index_path).write_text(index_text, encoding='utf-8')
    (bundle_dir / 'README.md').write_text(index_text, encoding='utf-8')

    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary['status'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
