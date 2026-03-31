from __future__ import annotations

import json
import subprocess
from dataclasses import dataclass, field
from pathlib import Path
from typing import Dict, Iterable, List, Optional, Sequence, Set, Tuple

from .models import Task


@dataclass
class BudgetDecision:
    status: str
    estimated_cost: float
    remaining_budget: float
    reason: str


class CostController:
    def __init__(self, medium_hourly_costs: Optional[Dict[str, float]] = None):
        self.medium_hourly_costs = medium_hourly_costs or {
            "docker": 2.0,
            "browser": 4.0,
            "vm": 8.0,
            "none": 0.0,
        }

    def estimate_cost(self, medium: str, duration_minutes: int) -> float:
        hourly = self.medium_hourly_costs.get(medium, 0.0)
        return round(hourly * (max(0, duration_minutes) / 60.0), 2)

    def evaluate(self, task: Task, medium: str, duration_minutes: int, spent_so_far: float = 0.0) -> BudgetDecision:
        estimated = self.estimate_cost(medium, duration_minutes)
        effective_budget = float(task.budget + task.budget_override_amount)
        remaining = round(effective_budget - spent_so_far - estimated, 2)

        if effective_budget <= 0:
            return BudgetDecision(
                status="allow",
                estimated_cost=estimated,
                remaining_budget=remaining,
                reason="budget not set",
            )

        if remaining >= 0:
            return BudgetDecision(
                status="allow",
                estimated_cost=estimated,
                remaining_budget=remaining,
                reason="within budget",
            )

        downgraded_medium = "docker" if medium in {"browser", "vm"} else "none"
        if downgraded_medium != medium:
            downgraded_estimated = self.estimate_cost(downgraded_medium, duration_minutes)
            downgraded_remaining = round(effective_budget - spent_so_far - downgraded_estimated, 2)
            if downgraded_remaining >= 0:
                return BudgetDecision(
                    status="degrade",
                    estimated_cost=downgraded_estimated,
                    remaining_budget=downgraded_remaining,
                    reason=f"degrade to {downgraded_medium} to stay within budget",
                )

        return BudgetDecision(
            status="pause",
            estimated_cost=estimated,
            remaining_budget=remaining,
            reason="budget exceeded",
        )


@dataclass(frozen=True)
class EpicMilestone:
    epic_id: str
    title: str
    phase: str
    owner: str
    milestone: str


@dataclass
class ExecutionPackRoadmap:
    name: str
    epics: List[EpicMilestone] = field(default_factory=list)

    def phase_map(self) -> Dict[str, List[EpicMilestone]]:
        phases: Dict[str, List[EpicMilestone]] = {}
        for epic in self.epics:
            phases.setdefault(epic.phase, []).append(epic)
        return phases

    def epic_map(self) -> Dict[str, EpicMilestone]:
        return {epic.epic_id: epic for epic in self.epics}

    def validate_unique_owners(self) -> None:
        seen: Dict[str, str] = {}
        for epic in self.epics:
            owner = epic.owner.strip().lower()
            if owner in seen:
                raise ValueError(
                    f"Owner '{epic.owner}' is assigned to both {seen[owner]} and {epic.epic_id}"
                )
            seen[owner] = epic.epic_id


def build_execution_pack_roadmap() -> ExecutionPackRoadmap:
    roadmap = ExecutionPackRoadmap(
        name="BigClaw v4.0 Execution Pack",
        epics=[
            EpicMilestone(
                epic_id="BIG-EPIC-8",
                title="研发自治执行平台增强",
                phase="Phase 1",
                owner="engineering-platform",
                milestone="M1 Foundation uplift",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-9",
                title="工程运营系统",
                phase="Phase 2",
                owner="engineering-operations",
                milestone="M2 Operations control plane",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-10",
                title="跨部门 Agent Orchestration",
                phase="Phase 3",
                owner="orchestration-office",
                milestone="M3 Cross-team orchestration",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-11",
                title="产品化 UI 与控制台",
                phase="Phase 4",
                owner="product-experience",
                milestone="M4 Productized console",
            ),
            EpicMilestone(
                epic_id="BIG-EPIC-12",
                title="计费、套餐与商业化控制",
                phase="Phase 5",
                owner="commercialization",
                milestone="M5 Billing and packaging",
            ),
        ],
    )
    roadmap.validate_unique_owners()
    return roadmap


@dataclass
class MemoryPattern:
    task_id: str
    title: str
    labels: List[str] = field(default_factory=list)
    required_tools: List[str] = field(default_factory=list)
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)
    summary: str = ""

    def to_dict(self) -> Dict:
        return {
            "task_id": self.task_id,
            "title": self.title,
            "labels": list(self.labels),
            "required_tools": list(self.required_tools),
            "acceptance_criteria": list(self.acceptance_criteria),
            "validation_plan": list(self.validation_plan),
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict) -> "MemoryPattern":
        return cls(
            task_id=str(data.get("task_id", "")),
            title=str(data.get("title", "")),
            labels=[str(item) for item in data.get("labels", [])],
            required_tools=[str(item) for item in data.get("required_tools", [])],
            acceptance_criteria=[str(item) for item in data.get("acceptance_criteria", [])],
            validation_plan=[str(item) for item in data.get("validation_plan", [])],
            summary=str(data.get("summary", "")),
        )


class TaskMemoryStore:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)

    def _load_patterns(self) -> List[MemoryPattern]:
        if not self.storage_path.exists():
            return []
        payload = json.loads(self.storage_path.read_text())
        return [MemoryPattern.from_dict(item) for item in payload]

    def _write_patterns(self, patterns: List[MemoryPattern]) -> None:
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        self.storage_path.write_text(json.dumps([item.to_dict() for item in patterns], ensure_ascii=False, indent=2))

    def remember_success(self, task: Task, summary: str = "") -> None:
        patterns = [item for item in self._load_patterns() if item.task_id != task.task_id]
        patterns.append(
            MemoryPattern(
                task_id=task.task_id,
                title=task.title,
                labels=list(task.labels),
                required_tools=list(task.required_tools),
                acceptance_criteria=list(task.acceptance_criteria),
                validation_plan=list(task.validation_plan),
                summary=summary,
            )
        )
        self._write_patterns(patterns)

    def suggest_rules(self, task: Task, limit: int = 3) -> Dict[str, List[str]]:
        ranked: List[Tuple[float, MemoryPattern]] = []
        for pattern in self._load_patterns():
            score = self._score(task, pattern)
            if score <= 0:
                continue
            ranked.append((score, pattern))

        ranked.sort(key=lambda item: item[0], reverse=True)
        selected = [item[1] for item in ranked[: max(1, limit)]]

        acceptance = list(task.acceptance_criteria)
        validation = list(task.validation_plan)
        for pattern in selected:
            for item in pattern.acceptance_criteria:
                if item not in acceptance:
                    acceptance.append(item)
            for item in pattern.validation_plan:
                if item not in validation:
                    validation.append(item)

        return {
            "acceptance_criteria": acceptance,
            "validation_plan": validation,
            "matched_task_ids": [item.task_id for item in selected],
        }

    @staticmethod
    def _score(task: Task, pattern: MemoryPattern) -> float:
        label_overlap = len(set(task.labels) & set(pattern.labels))
        tool_overlap = len(set(task.required_tools) & set(pattern.required_tools))
        return float(label_overlap * 2 + tool_overlap)


@dataclass
class ValidationReportDecision:
    allowed_to_close: bool
    status: str
    summary: str
    missing_reports: List[str] = field(default_factory=list)


REQUIRED_REPORT_ARTIFACTS = [
    "task-run",
    "replay",
    "benchmark-suite",
]


def enforce_validation_report_policy(artifacts: List[str]) -> ValidationReportDecision:
    existing = set(artifacts)
    missing = [name for name in REQUIRED_REPORT_ARTIFACTS if name not in existing]
    if missing:
        return ValidationReportDecision(
            allowed_to_close=False,
            status="blocked",
            summary="validation report policy not satisfied",
            missing_reports=missing,
        )
    return ValidationReportDecision(
        allowed_to_close=True,
        status="ready",
        summary="validation report policy satisfied",
    )


class ParallelIssueQueue:
    def __init__(self, queue_path: str):
        self.queue_path = Path(queue_path)
        self.payload = json.loads(self.queue_path.read_text())

    def project_slug(self) -> str:
        return str(self.payload["project"]["slug_id"])

    def activate_state_id(self) -> str:
        return str(self.payload["policy"]["activate_state_id"])

    def target_in_progress(self) -> int:
        return int(self.payload["policy"]["target_in_progress"])

    def refill_states(self) -> Set[str]:
        return {str(name) for name in self.payload["policy"].get("refill_states", [])}

    def issue_order(self) -> List[str]:
        return [str(identifier) for identifier in self.payload.get("issue_order", [])]

    def issue_records(self) -> List[dict]:
        return list(self.payload.get("issues", []))

    def issue_identifiers(self) -> List[str]:
        return [str(record["identifier"]) for record in self.issue_records()]

    def select_candidates(
        self,
        active_identifiers: Set[str],
        issue_states: Dict[str, str],
        target_in_progress: Optional[int] = None,
    ) -> List[str]:
        target = self.target_in_progress() if target_in_progress is None else int(target_in_progress)
        needed = max(target - len(active_identifiers), 0)
        if needed == 0:
            return []
        candidates: List[str] = []
        refill_states = self.refill_states()
        for identifier in self.issue_order():
            if needed == 0:
                break
            if identifier in active_identifiers:
                continue
            if issue_states.get(identifier) in refill_states:
                candidates.append(identifier)
                needed -= 1
        return candidates


def issue_state_map(issues: Sequence[dict]) -> Dict[str, str]:
    state_map: Dict[str, str] = {}
    for issue in issues:
        identifier = str(issue.get("identifier", "")).strip()
        state = issue.get("state") or {}
        state_name = str(state.get("name", issue.get("state_name", ""))).strip()
        if identifier and state_name:
            state_map[identifier] = state_name
    return state_map


LEGACY_PYTHON_WRAPPER_NOTICE = (
    "Legacy Python operator wrapper: use scripts/ops/bigclawctl for the Go mainline. "
    "This Python path remains only as a compatibility shim during migration."
)


def append_missing_flag(args: Sequence[str], flag: str, value: str) -> List[str]:
    flag_prefix = flag + "="
    if any(arg == flag or arg.startswith(flag_prefix) for arg in args):
        return list(args)
    return [*args, flag, value]


def build_bigclawctl_exec_args(repo_root: Path, command: Iterable[str], forwarded: Sequence[str]) -> List[str]:
    return ["bash", str(repo_root / "scripts/ops/bigclawctl"), *command, *forwarded]


def repo_root_from_script(script_path: str) -> Path:
    return Path(script_path).resolve().parents[2]


def run_bigclawctl_shim(script_path: str, command: Iterable[str], forwarded: Sequence[str]) -> int:
    repo_root = repo_root_from_script(script_path)
    argv = build_bigclawctl_exec_args(repo_root, command, forwarded)
    return subprocess.call(argv, cwd=repo_root)


def build_workspace_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    args = list(forwarded)
    args = append_missing_flag(args, "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
    args = append_missing_flag(args, "--cache-key", "openagis-bigclaw")
    return build_bigclawctl_exec_args(repo_root, ["workspace", "bootstrap"], args)


def translate_workspace_validate_args(forwarded: Sequence[str]) -> List[str]:
    translated: List[str] = []
    i = 0
    while i < len(forwarded):
        arg = forwarded[i]
        if arg == "--report-file":
            translated.extend(["--report", forwarded[i + 1]])
            i += 2
            continue
        if arg.startswith("--report-file="):
            translated.append("--report=" + arg.split("=", 1)[1])
            i += 1
            continue
        if arg == "--no-cleanup":
            translated.append("--cleanup=false")
            i += 1
            continue
        if arg == "--issues":
            issues: List[str] = []
            i += 1
            while i < len(forwarded) and not forwarded[i].startswith("-"):
                issues.append(forwarded[i])
                i += 1
            translated.extend(["--issues", ",".join(issues)])
            continue
        translated.append(arg)
        i += 1
    return translated


def build_workspace_validate_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["workspace", "validate"], translate_workspace_validate_args(forwarded))


def build_github_sync_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["github-sync"], list(forwarded))


def build_refill_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["refill"], list(forwarded))


def build_workspace_runtime_bootstrap_args(repo_root: Path, forwarded: Sequence[str]) -> List[str]:
    return build_bigclawctl_exec_args(repo_root, ["workspace"], list(forwarded))
