from __future__ import annotations

import json
from pathlib import Path
from typing import Any, Sequence

from bigclaw.workspace_bootstrap import bootstrap_workspace, cleanup_workspace


def build_validation_report(
    *,
    repo_url: str,
    workspace_root: str | Path,
    issue_identifiers: Sequence[str],
    default_branch: str = "main",
    cache_root: str | Path | None = None,
    cache_base: str | Path | None = None,
    cache_key: str | None = None,
    cleanup: bool = True,
) -> dict[str, Any]:
    workspace_root_path = Path(workspace_root).expanduser().resolve()
    workspace_root_path.mkdir(parents=True, exist_ok=True)

    bootstrap_results = []
    for issue_identifier in issue_identifiers:
        workspace_path = workspace_root_path / issue_identifier
        status = bootstrap_workspace(
            workspace=workspace_path,
            issue_identifier=issue_identifier,
            repo_url=repo_url,
            default_branch=default_branch,
            cache_root=cache_root,
            cache_base=cache_base,
            cache_key=cache_key,
        )
        bootstrap_results.append(status.to_dict())

    cache_roots = sorted({result["cache_root"] for result in bootstrap_results})
    mirror_paths = sorted({result["mirror_path"] for result in bootstrap_results})
    seed_paths = sorted({result["seed_path"] for result in bootstrap_results})
    cleanup_results = []

    if cleanup:
        for issue_identifier in issue_identifiers:
            workspace_path = workspace_root_path / issue_identifier
            status = cleanup_workspace(
                workspace=workspace_path,
                issue_identifier=issue_identifier,
                repo_url=repo_url,
                default_branch=default_branch,
                cache_root=cache_root,
                cache_base=cache_base,
                cache_key=cache_key,
            )
            cleanup_results.append(status.to_dict())

    report = {
        "repo_url": repo_url,
        "default_branch": default_branch,
        "workspace_root": str(workspace_root_path),
        "issue_identifiers": list(issue_identifiers),
        "bootstrap_results": bootstrap_results,
        "cleanup_results": cleanup_results,
        "summary": {
            "workspace_count": len(bootstrap_results),
            "unique_cache_roots": cache_roots,
            "unique_mirror_paths": mirror_paths,
            "unique_seed_paths": seed_paths,
            "single_cache_root_reused": len(cache_roots) == 1,
            "single_mirror_reused": len(mirror_paths) == 1,
            "single_seed_reused": len(seed_paths) == 1,
            "mirror_creations": sum(1 for result in bootstrap_results if result["mirror_created"]),
            "seed_creations": sum(1 for result in bootstrap_results if result["seed_created"]),
            "clone_suppressed_after_first": all(result["clone_suppressed"] for result in bootstrap_results[1:]),
            "cache_reused_after_first": all(result["cache_reused"] for result in bootstrap_results[1:]),
            "all_workspaces_created_via_worktree": all(
                result["workspace_mode"] in {"worktree_created", "workspace_reused"}
                for result in bootstrap_results
            ),
            "cleanup_preserved_cache": bool(bootstrap_results)
            and Path(bootstrap_results[0]["mirror_path"]).exists()
            and Path(bootstrap_results[0]["seed_path"]).joinpath(".git").exists(),
        },
    }
    return report


def render_validation_markdown(report: dict[str, Any]) -> str:
    summary = report["summary"]
    lines = [
        "# Symphony bootstrap cache validation",
        "",
        f"- Repo: `{report['repo_url']}`",
        f"- Workspace root: `{report['workspace_root']}`",
        f"- Workspaces: `{summary['workspace_count']}`",
        f"- Single cache root reused: `{summary['single_cache_root_reused']}`",
        f"- Mirror creations: `{summary['mirror_creations']}`",
        f"- Seed creations: `{summary['seed_creations']}`",
        f"- Clone suppressed after first workspace: `{summary['clone_suppressed_after_first']}`",
        f"- Cleanup preserved cache: `{summary['cleanup_preserved_cache']}`",
        "",
        "## Bootstrap Results",
        "",
    ]

    for result in report["bootstrap_results"]:
        lines.extend(
            [
                f"- `{result['workspace']}`",
                f"  - `cache_root={result['cache_root']}`",
                f"  - `cache_key={result['cache_key']}`",
                f"  - `workspace_mode={result['workspace_mode']}`",
                f"  - `cache_reused={result['cache_reused']}`",
                f"  - `clone_suppressed={result['clone_suppressed']}`",
                f"  - `mirror_created={result['mirror_created']}`",
                f"  - `seed_created={result['seed_created']}`",
            ]
        )

    return "\n".join(lines) + "\n"


def write_validation_report(report: dict[str, Any], path: str | Path) -> Path:
    target = Path(path).expanduser().resolve()
    target.parent.mkdir(parents=True, exist_ok=True)

    if target.suffix.lower() == ".md":
        target.write_text(render_validation_markdown(report))
    else:
        target.write_text(json.dumps(report, ensure_ascii=False, indent=2))
    return target
