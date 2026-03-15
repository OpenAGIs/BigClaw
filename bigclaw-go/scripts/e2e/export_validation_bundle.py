#!/usr/bin/env python3
import argparse
import json
import shutil
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Optional


SCHEMA_VERSION = 'live_validation_bundle/v1'
LATEST_REPORTS = {
    'local': 'docs/reports/sqlite-smoke-report.json',
    'kubernetes': 'docs/reports/kubernetes-live-smoke-report.json',
    'ray': 'docs/reports/ray-live-smoke-report.json',
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


def same_path(source: Path, destination: Path) -> bool:
    return source.expanduser().resolve() == destination.expanduser().resolve()


def copy_text_artifact(source: Path, destination: Path) -> str:
    if not source.exists():
        return ''
    if same_path(source, destination):
        return str(destination)
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, destination)
    return str(destination)


def copy_json_artifact(source: Path, destination: Path) -> str:
    if not source.exists():
        return ''
    if same_path(source, destination):
        return str(destination)
    payload = read_json(source)
    if payload is None:
        return ''
    write_json(destination, payload)
    return str(destination)


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


def first_artifact_list(report: dict[str, Any]) -> list[str]:
    status = report.get('status')
    if isinstance(status, dict):
        latest_event = status.get('latest_event')
        if isinstance(latest_event, dict):
            payload = latest_event.get('payload')
            if isinstance(payload, dict):
                artifacts = payload.get('artifacts')
                if isinstance(artifacts, list):
                    return [str(item) for item in artifacts]
    artifacts = report.get('artifacts')
    if isinstance(artifacts, list):
        return [str(item) for item in artifacts]
    return []


def build_component_contract(
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
    artifacts = {
        'bundle_report_path': relpath(report_path, root) if enabled else '',
        'canonical_report_path': LATEST_REPORTS[name],
        'stdout_path': '',
        'stderr_path': '',
        'service_log_path': '',
        'audit_log_path': '',
    }
    section: dict[str, Any] = {
        'name': name,
        'enabled': enabled,
        'status': 'skipped',
        'artifacts': artifacts,
        'task': {
            'id': '',
            'trace_id': '',
            'state': '',
        },
        'control_plane': {
            'base_url': '',
        },
        'report_summary': {
            'latest_event_type': '',
        },
        'execution_artifacts': [],
    }
    if not enabled:
        return section

    report = read_json(report_path)
    section['status'] = component_status(report)
    if not isinstance(report, dict):
        return section

    copied_latest = copy_json_artifact(report_path, latest_report_path)
    if copied_latest:
        section['artifacts']['canonical_report_path'] = relpath(Path(copied_latest), root)

    stdout_copy = copy_text_artifact(stdout_path, bundle_dir / f'{name}.stdout.log')
    stderr_copy = copy_text_artifact(stderr_path, bundle_dir / f'{name}.stderr.log')
    if stdout_copy:
        section['artifacts']['stdout_path'] = relpath(Path(stdout_copy), root)
    if stderr_copy:
        section['artifacts']['stderr_path'] = relpath(Path(stderr_copy), root)

    task = report.get('task')
    if isinstance(task, dict):
        section['task']['id'] = str(task.get('id', ''))
        section['task']['trace_id'] = str(task.get('trace_id', ''))
        section['task']['state'] = str(task.get('state', ''))
    base_url = report.get('base_url')
    if base_url:
        section['control_plane']['base_url'] = str(base_url)

    status = report.get('status')
    if isinstance(status, dict):
        latest_event = status.get('latest_event')
        if isinstance(latest_event, dict):
            section['report_summary']['latest_event_type'] = str(latest_event.get('type', ''))
        task_state = status.get('state')
        if task_state:
            section['task']['state'] = str(task_state)

    section['execution_artifacts'] = first_artifact_list(report)

    state_dir = report.get('state_dir')
    if state_dir:
        audit_source = Path(state_dir) / 'audit.jsonl'
        audit_copy = copy_text_artifact(audit_source, bundle_dir / f'{name}.audit.jsonl')
        if audit_copy:
            section['artifacts']['audit_log_path'] = relpath(Path(audit_copy), root)
    existing_audit_copy = bundle_dir / f'{name}.audit.jsonl'
    if not section['artifacts']['audit_log_path'] and existing_audit_copy.exists():
        section['artifacts']['audit_log_path'] = relpath(existing_audit_copy, root)
    service_log = report.get('service_log')
    if service_log:
        service_copy = copy_text_artifact(Path(service_log), bundle_dir / f'{name}.service.log')
        if service_copy:
            section['artifacts']['service_log_path'] = relpath(Path(service_copy), root)
    existing_service_copy = bundle_dir / f'{name}.service.log'
    if not section['artifacts']['service_log_path'] and existing_service_copy.exists():
        section['artifacts']['service_log_path'] = relpath(existing_service_copy, root)

    return section


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
    return build_component_contract(
        name=name,
        enabled=enabled,
        root=root,
        bundle_dir=bundle_dir,
        report_path=report_path,
        stdout_path=stdout_path,
        stderr_path=stderr_path,
    )


def summary_bundle(summary: dict[str, Any]) -> dict[str, str]:
    bundle = summary.get('bundle')
    if isinstance(bundle, dict):
        return {
            'run_id': str(bundle.get('run_id', '')),
            'path': str(bundle.get('path', '')),
            'summary_path': str(bundle.get('summary_path', '')),
            'readme_path': str(bundle.get('readme_path', '')),
        }
    return {
        'run_id': str(summary.get('run_id', '')),
        'path': str(summary.get('bundle_path', '')),
        'summary_path': str(summary.get('summary_path', '')),
        'readme_path': str(summary.get('readme_path', '')),
    }


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
        bundle = summary_bundle(summary)
        items.append(
            {
                'run_id': bundle['run_id'],
                'generated_at': summary.get('generated_at', ''),
                'status': summary.get('status', 'unknown'),
                'bundle_path': bundle['path'],
                'summary_path': bundle['summary_path'],
                'readme_path': bundle['readme_path'],
            }
        )
    return items


def render_index(summary: dict[str, Any], recent_runs: list[dict[str, Any]]) -> str:
    bundle = summary['bundle']
    lines = [
        '# Live Validation Index',
        '',
        f"- Schema: `{summary['schema_version']}`",
        f"- Latest run: `{bundle['run_id']}`",
        f"- Generated at: `{summary['generated_at']}`",
        f"- Status: `{summary['status']}`",
        f"- Bundle: `{bundle['path']}`",
        f"- Bundle summary: `{bundle['summary_path']}`",
        f"- Bundle README: `{bundle['readme_path']}`",
        '',
        '## Component bundles',
        '',
    ]
    for name in ('local', 'kubernetes', 'ray'):
        section = summary['components'][name]
        artifacts = section['artifacts']
        lines.append(f"### {name}")
        lines.append(f"- Enabled: `{section['enabled']}`")
        lines.append(f"- Status: `{section['status']}`")
        lines.append(f"- Bundle report: `{artifacts['bundle_report_path']}`")
        lines.append(f"- Canonical report: `{artifacts['canonical_report_path']}`")
        if artifacts.get('stdout_path'):
            lines.append(f"- Stdout log: `{artifacts['stdout_path']}`")
        if artifacts.get('stderr_path'):
            lines.append(f"- Stderr log: `{artifacts['stderr_path']}`")
        if artifacts.get('service_log_path'):
            lines.append(f"- Service log: `{artifacts['service_log_path']}`")
        if artifacts.get('audit_log_path'):
            lines.append(f"- Audit log: `{artifacts['audit_log_path']}`")
        if section['task'].get('id'):
            lines.append(f"- Task ID: `{section['task']['id']}`")
        if section['task'].get('state'):
            lines.append(f"- Task state: `{section['task']['state']}`")
        if section['report_summary'].get('latest_event_type'):
            lines.append(f"- Latest event: `{section['report_summary']['latest_event_type']}`")
        if section['execution_artifacts']:
            lines.append(f"- Execution artifacts: `{', '.join(section['execution_artifacts'])}`")
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
        'schema_version': SCHEMA_VERSION,
        'run_id': args.run_id,
        'generated_at': datetime.now(timezone.utc).isoformat(),
        'status': 'succeeded' if args.validation_status == '0' else 'failed',
        'closeout_commands': [
            'cd bigclaw-go && ./scripts/e2e/run_all.sh',
            'cd bigclaw-go && go test ./...',
            'git push origin <branch> && git log -1 --stat',
        ],
    }
    bundle_summary_path = bundle_dir / 'summary.json'
    bundle_readme_path = bundle_dir / 'README.md'
    summary['bundle'] = {
        'run_id': args.run_id,
        'path': relpath(bundle_dir, root),
        'summary_path': relpath(bundle_summary_path, root),
        'readme_path': relpath(bundle_readme_path, root),
    }
    summary['components'] = {}
    summary['components']['local'] = build_component_section(
        name='local',
        enabled=args.run_local == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.local_report_path,
        stdout_path=Path(args.local_stdout_path),
        stderr_path=Path(args.local_stderr_path),
    )
    summary['components']['kubernetes'] = build_component_section(
        name='kubernetes',
        enabled=args.run_kubernetes == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.kubernetes_report_path,
        stdout_path=Path(args.kubernetes_stdout_path),
        stderr_path=Path(args.kubernetes_stderr_path),
    )
    summary['components']['ray'] = build_component_section(
        name='ray',
        enabled=args.run_ray == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.ray_report_path,
        stdout_path=Path(args.ray_stdout_path),
        stderr_path=Path(args.ray_stderr_path),
    )
    canonical_summary_path = root / args.summary_path
    write_json(bundle_summary_path, summary)
    write_json(canonical_summary_path, summary)

    bundle_root = bundle_dir.parent
    recent_runs = build_recent_runs(bundle_root, root)
    manifest = {
        'schema_version': SCHEMA_VERSION,
        'latest': summary,
        'recent_bundles': recent_runs,
    }
    write_json(root / args.manifest_path, manifest)

    index_text = render_index(summary, recent_runs)
    (root / args.index_path).parent.mkdir(parents=True, exist_ok=True)
    (root / args.index_path).write_text(index_text, encoding='utf-8')
    bundle_readme_path.write_text(index_text, encoding='utf-8')

    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary['status'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
