#!/usr/bin/env python3
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
    'shared_queue': 'docs/reports/multi-node-shared-queue-report.json',
}

SUPPORTING_REPORTS = [
    'docs/reports/replay-retention-semantics-report.md',
    'docs/reports/replicated-event-log-durability-rollout-contract.md',
    'docs/reports/multi-subscriber-takeover-validation-report.md',
    'docs/reports/multi-subscriber-takeover-validation-report.json',
]


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
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(source, destination)
    return str(destination)


def copy_json_artifact(source: Path, destination: Path) -> str:
    if not source.exists():
        return ''
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
        nodes = report.get('nodes')
        if isinstance(nodes, list):
            copied_nodes: list[dict[str, Any]] = []
            for index, node in enumerate(nodes):
                if not isinstance(node, dict):
                    continue
                copied_node = dict(node)
                audit_path = node.get('audit_path')
                if isinstance(audit_path, str) and audit_path:
                    audit_copy = copy_text_artifact(Path(audit_path), bundle_dir / f'{name}.node-{index + 1}.audit.jsonl')
                    if audit_copy:
                        copied_node['bundle_audit_path'] = relpath(Path(audit_copy), root)
                service_log = node.get('service_log')
                if isinstance(service_log, str) and service_log:
                    service_copy = copy_text_artifact(Path(service_log), bundle_dir / f'{name}.node-{index + 1}.service.log')
                    if service_copy:
                        copied_node['bundle_service_log_path'] = relpath(Path(service_copy), root)
                copied_nodes.append(copied_node)
            if copied_nodes:
                section['nodes'] = copied_nodes
    return section


def supporting_report_inventory(root: Path, bundle_dir: Path) -> list[dict[str, str]]:
    reports: list[dict[str, str]] = []
    for relative in SUPPORTING_REPORTS:
        source = root / relative
        if not source.exists():
            continue
        destination = bundle_dir / Path(relative).name
        copied = copy_text_artifact(source, destination)
        reports.append(
            {
                'source_path': relative,
                'bundle_path': relpath(Path(copied), root) if copied else '',
            }
        )
    return reports


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


def render_index(summary: dict[str, Any], recent_runs: list[dict[str, Any]]) -> str:
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
    for name in ('local', 'kubernetes', 'ray', 'shared_queue'):
        section = summary[name]
        lines.append(f"### {name}")
        lines.append(f"- Enabled: `{section['enabled']}`")
        lines.append(f"- Status: `{section['status']}`")
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
        nodes = section.get('nodes')
        if isinstance(nodes, list):
            for node in nodes:
                if not isinstance(node, dict):
                    continue
                name_hint = node.get('name', 'node')
                if node.get('bundle_audit_path'):
                    lines.append(f"- {name_hint} audit log: `{node['bundle_audit_path']}`")
                if node.get('bundle_service_log_path'):
                    lines.append(f"- {name_hint} service log: `{node['bundle_service_log_path']}`")
        lines.append('')

    lines.extend(['## Supporting durability and takeover references', ''])
    for report in summary.get('supporting_reports', []):
        lines.append(f"- Source: `{report['source_path']}`")
        if report.get('bundle_path'):
            lines.append(f"- Bundle copy: `{report['bundle_path']}`")
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
    parser.add_argument('--run-shared-queue', default='1')
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
    parser.add_argument('--shared-queue-report-path', required=True)
    parser.add_argument('--shared-queue-stdout-path', required=True)
    parser.add_argument('--shared-queue-stderr-path', required=True)
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
    summary['shared_queue'] = build_component_section(
        name='shared_queue',
        enabled=args.run_shared_queue == '1',
        root=root,
        bundle_dir=bundle_dir,
        report_path=root / args.shared_queue_report_path,
        stdout_path=Path(args.shared_queue_stdout_path),
        stderr_path=Path(args.shared_queue_stderr_path),
    )
    summary['supporting_reports'] = supporting_report_inventory(root, bundle_dir)

    bundle_summary_path = bundle_dir / 'summary.json'
    canonical_summary_path = root / args.summary_path
    summary['summary_path'] = relpath(bundle_summary_path, root)
    write_json(bundle_summary_path, summary)
    write_json(canonical_summary_path, summary)

    bundle_root = bundle_dir.parent
    recent_runs = build_recent_runs(bundle_root, root)
    manifest = {'latest': summary, 'recent_runs': recent_runs}
    write_json(root / args.manifest_path, manifest)

    index_text = render_index(summary, recent_runs)
    (root / args.index_path).parent.mkdir(parents=True, exist_ok=True)
    (root / args.index_path).write_text(index_text, encoding='utf-8')
    (bundle_dir / 'README.md').write_text(index_text, encoding='utf-8')

    print(json.dumps(summary, ensure_ascii=False, indent=2))
    return 0 if summary['status'] == 'succeeded' else 1


if __name__ == '__main__':
    raise SystemExit(main())
