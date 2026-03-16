#!/usr/bin/env python3
import argparse
import json
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


DEFAULT_REQUIRED_COMPONENTS = ("local", "kubernetes", "ray")


def read_json(path: Path) -> Any:
    if not path.exists() or path.stat().st_size == 0:
        return None
    return json.loads(path.read_text(encoding="utf-8"))


def write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def parse_timestamp(value: Any) -> datetime | None:
    if not isinstance(value, str) or not value:
        return None
    normalized = value.replace("Z", "+00:00")
    try:
        dt = datetime.fromisoformat(normalized)
    except ValueError:
        return None
    if dt.tzinfo is None:
        return dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(timezone.utc)


def relpath(path: Path, root: Path) -> str:
    try:
        return str(path.relative_to(root))
    except ValueError:
        return str(path)


def component_snapshot(summary: dict[str, Any], component: str) -> dict[str, Any]:
    section = summary.get(component)
    if not isinstance(section, dict):
        return {"enabled": False, "status": "missing_component"}
    enabled = bool(section.get("enabled", False))
    status = str(section.get("status", "unknown"))
    return {"enabled": enabled, "status": status}


def evaluate_run(
    *,
    summary: dict[str, Any],
    root: Path,
    stale_after_hours: float,
    required_components: tuple[str, ...],
    now: datetime,
) -> dict[str, Any]:
    generated_at = parse_timestamp(summary.get("generated_at"))
    age_hours = None
    stale_for_age = False
    if generated_at is not None:
        age_hours = round((now - generated_at).total_seconds() / 3600, 3)
        stale_for_age = age_hours > stale_after_hours

    components = {name: component_snapshot(summary, name) for name in required_components}
    missing_components = [name for name, component in components.items() if component["enabled"] and component["status"] == "missing_report"]
    failing_components = [
        name
        for name, component in components.items()
        if component["enabled"] and component["status"] not in {"succeeded", "skipped"}
    ]
    disabled_components = [name for name, component in components.items() if not component["enabled"]]

    stale_reasons: list[str] = []
    if stale_for_age:
        stale_reasons.append("bundle_age_exceeds_threshold")
    if str(summary.get("status")) != "succeeded":
        stale_reasons.append("bundle_status_not_succeeded")
    stale_reasons.extend(f"component_incomplete:{name}" for name in failing_components)
    stale_reasons.extend(f"component_disabled:{name}" for name in disabled_components)

    bundle_path = summary.get("bundle_path")
    if isinstance(bundle_path, str) and bundle_path:
        summary_path = str((root / bundle_path / "summary.json"))
    else:
        summary_path = ""

    return {
        "run_id": str(summary.get("run_id", "")),
        "generated_at": summary.get("generated_at", ""),
        "status": str(summary.get("status", "unknown")),
        "bundle_path": bundle_path or "",
        "summary_path": relpath(Path(summary_path), root) if summary_path else "",
        "age_hours": age_hours,
        "components": components,
        "missing_components": missing_components,
        "failing_components": failing_components,
        "disabled_components": disabled_components,
        "stale_bundle": bool(stale_reasons),
        "stale_reasons": stale_reasons,
        "ready_for_closeout": not stale_reasons,
    }


def load_recent_runs(bundle_root: Path, history_window: int) -> list[dict[str, Any]]:
    runs: list[tuple[datetime, dict[str, Any]]] = []
    if not bundle_root.exists():
        return []
    for child in bundle_root.iterdir():
        if not child.is_dir():
            continue
        summary = read_json(child / "summary.json")
        if not isinstance(summary, dict):
            continue
        generated_at = parse_timestamp(summary.get("generated_at")) or datetime.min.replace(tzinfo=timezone.utc)
        runs.append((generated_at, summary))
    runs.sort(key=lambda item: item[0], reverse=True)
    return [summary for _, summary in runs[:history_window]]


def build_scorecard(
    *,
    root: Path,
    bundle_root: Path,
    history_window: int,
    stale_after_hours: float,
    required_components: tuple[str, ...],
) -> dict[str, Any]:
    now = datetime.now(timezone.utc)
    recent_summaries = load_recent_runs(bundle_root, history_window)
    runs = [
        evaluate_run(
            summary=summary,
            root=root,
            stale_after_hours=stale_after_hours,
            required_components=required_components,
            now=now,
        )
        for summary in recent_summaries
    ]
    latest = runs[0] if runs else None
    return {
        "generated_at": now.isoformat(),
        "history_window": history_window,
        "stale_after_hours": stale_after_hours,
        "required_components": list(required_components),
        "latest_run_id": latest["run_id"] if latest else "",
        "latest_bundle_policy": latest
        or {
            "run_id": "",
            "status": "missing",
            "stale_bundle": True,
            "stale_reasons": ["no_validation_bundles_found"],
            "ready_for_closeout": False,
        },
        "window_runs": runs,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Score validation bundle continuation freshness across recent runs")
    parser.add_argument("--go-root", required=True)
    parser.add_argument("--bundle-root", default="docs/reports/live-validation-runs")
    parser.add_argument("--history-window", type=int, default=3)
    parser.add_argument("--stale-after-hours", type=float, default=24.0)
    parser.add_argument(
        "--required-components",
        nargs="*",
        default=list(DEFAULT_REQUIRED_COMPONENTS),
    )
    parser.add_argument(
        "--output-path",
        default="docs/reports/validation-bundle-continuation-scorecard.json",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = Path(args.go_root).resolve()
    scorecard = build_scorecard(
        root=root,
        bundle_root=root / args.bundle_root,
        history_window=max(args.history_window, 1),
        stale_after_hours=max(args.stale_after_hours, 0.0),
        required_components=tuple(args.required_components or DEFAULT_REQUIRED_COMPONENTS),
    )
    output_path = root / args.output_path
    write_json(output_path, scorecard)
    print(json.dumps(scorecard, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
