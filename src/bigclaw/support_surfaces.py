from __future__ import annotations

import json
import stat
import subprocess
from collections import defaultdict
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any, Callable, DefaultDict, Dict, List, Optional, Protocol, Sequence

from .models import Priority, RiskLevel, Task, TaskState
from .observability import ObservabilityLedger, TaskRun, utc_now


@dataclass
class SourceIssue:
    source: str
    source_id: str
    title: str
    description: str
    labels: List[str]
    priority: str
    state: str
    links: Dict[str, str]


class Connector(Protocol):
    name: str

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        ...


class GitHubConnector:
    name = "github"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="github",
                source_id=f"{project}#1",
                title="Fix flaky test",
                description="CI flaky on macOS",
                labels=["bug", "ci"],
                priority="P1",
                state=states[0] if states else "Todo",
                links={"issue": f"https://github.com/{project}/issues/1"},
            )
        ]


class LinearConnector:
    name = "linear"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="linear",
                source_id=f"{project}-101",
                title="Implement queue persistence",
                description="Need restart-safe queue",
                labels=["platform"],
                priority="P0",
                state=states[0] if states else "Todo",
                links={"issue": f"https://linear.app/{project}/issue/{project}-101"},
            )
        ]


class JiraConnector:
    name = "jira"

    def fetch_issues(self, project: str, states: List[str]) -> List[SourceIssue]:
        return [
            SourceIssue(
                source="jira",
                source_id=f"{project}-23",
                title="Runbook automation",
                description="Automate oncall runbook",
                labels=["ops"],
                priority="P2",
                state=states[0] if states else "Todo",
                links={"issue": f"https://jira.example.com/browse/{project}-23"},
            )
        ]


def map_priority(p: str) -> Priority:
    p = (p or "").upper()
    if p == "P0":
        return Priority.P0
    if p == "P1":
        return Priority.P1
    return Priority.P2


def map_state(s: str) -> TaskState:
    normalized = (s or "").lower()
    if "progress" in normalized:
        return TaskState.IN_PROGRESS
    if "done" in normalized or "closed" in normalized:
        return TaskState.DONE
    if "block" in normalized:
        return TaskState.BLOCKED
    return TaskState.TODO


def map_source_issue_to_task(issue: SourceIssue) -> Task:
    risk = RiskLevel.HIGH if "prod" in issue.title.lower() else RiskLevel.LOW
    return Task(
        task_id=issue.source_id,
        source=issue.source,
        title=issue.title,
        description=issue.description,
        labels=issue.labels,
        priority=map_priority(issue.priority),
        state=map_state(issue.state),
        risk_level=risk,
        required_tools=["github" if issue.source == "github" else "connector"],
        acceptance_criteria=["Synced from source issue"],
        validation_plan=["mapping-test"],
    )


VALID_VIEW_VISIBILITY = {"private", "team", "organization"}
VALID_DIGEST_CHANNELS = {"email", "slack", "webhook"}
VALID_DIGEST_CADENCES = {"hourly", "daily", "weekly"}


@dataclass(frozen=True)
class SavedViewFilter:
    field: str
    operator: str
    value: str

    def to_dict(self) -> Dict[str, str]:
        return {
            "field": self.field,
            "operator": self.operator,
            "value": self.value,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewFilter":
        return cls(
            field=str(data["field"]),
            operator=str(data["operator"]),
            value=str(data["value"]),
        )


@dataclass
class SavedView:
    view_id: str
    name: str
    route: str
    owner: str
    visibility: str = "private"
    filters: List[SavedViewFilter] = field(default_factory=list)
    sort_by: str = ""
    pinned: bool = False
    is_default: bool = False

    @property
    def filter_count(self) -> int:
        return len(self.filters)

    def to_dict(self) -> Dict[str, object]:
        return {
            "view_id": self.view_id,
            "name": self.name,
            "route": self.route,
            "owner": self.owner,
            "visibility": self.visibility,
            "filters": [view_filter.to_dict() for view_filter in self.filters],
            "sort_by": self.sort_by,
            "pinned": self.pinned,
            "is_default": self.is_default,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedView":
        return cls(
            view_id=str(data["view_id"]),
            name=str(data["name"]),
            route=str(data["route"]),
            owner=str(data["owner"]),
            visibility=str(data.get("visibility", "private")),
            filters=[SavedViewFilter.from_dict(item) for item in data.get("filters", [])],
            sort_by=str(data.get("sort_by", "")),
            pinned=bool(data.get("pinned", False)),
            is_default=bool(data.get("is_default", False)),
        )


@dataclass
class AlertDigestSubscription:
    subscription_id: str
    saved_view_id: str
    channel: str
    cadence: str
    recipients: List[str] = field(default_factory=list)
    include_empty_results: bool = False
    muted: bool = False

    def to_dict(self) -> Dict[str, object]:
        return {
            "subscription_id": self.subscription_id,
            "saved_view_id": self.saved_view_id,
            "channel": self.channel,
            "cadence": self.cadence,
            "recipients": list(self.recipients),
            "include_empty_results": self.include_empty_results,
            "muted": self.muted,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "AlertDigestSubscription":
        return cls(
            subscription_id=str(data["subscription_id"]),
            saved_view_id=str(data["saved_view_id"]),
            channel=str(data["channel"]),
            cadence=str(data["cadence"]),
            recipients=[str(recipient) for recipient in data.get("recipients", [])],
            include_empty_results=bool(data.get("include_empty_results", False)),
            muted=bool(data.get("muted", False)),
        )


@dataclass
class SavedViewCatalog:
    name: str
    version: str
    views: List[SavedView] = field(default_factory=list)
    subscriptions: List[AlertDigestSubscription] = field(default_factory=list)

    @property
    def view_index(self) -> Dict[str, SavedView]:
        return {view.view_id: view for view in self.views}

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "version": self.version,
            "views": [view.to_dict() for view in self.views],
            "subscriptions": [subscription.to_dict() for subscription in self.subscriptions],
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewCatalog":
        return cls(
            name=str(data["name"]),
            version=str(data["version"]),
            views=[SavedView.from_dict(item) for item in data.get("views", [])],
            subscriptions=[
                AlertDigestSubscription.from_dict(item) for item in data.get("subscriptions", [])
            ],
        )


@dataclass
class SavedViewCatalogAudit:
    catalog_name: str
    version: str
    view_count: int
    subscription_count: int
    duplicate_view_names: Dict[str, List[str]] = field(default_factory=dict)
    invalid_visibility_views: List[str] = field(default_factory=list)
    views_missing_filters: List[str] = field(default_factory=list)
    duplicate_default_views: Dict[str, List[str]] = field(default_factory=dict)
    orphan_subscriptions: List[str] = field(default_factory=list)
    subscriptions_missing_recipients: List[str] = field(default_factory=list)
    subscriptions_with_invalid_channel: List[str] = field(default_factory=list)
    subscriptions_with_invalid_cadence: List[str] = field(default_factory=list)

    @property
    def readiness_score(self) -> float:
        if self.view_count == 0:
            return 0.0
        penalties = (
            len(self.duplicate_view_names)
            + len(self.invalid_visibility_views)
            + len(self.views_missing_filters)
            + len(self.duplicate_default_views)
            + len(self.orphan_subscriptions)
            + len(self.subscriptions_missing_recipients)
            + len(self.subscriptions_with_invalid_channel)
            + len(self.subscriptions_with_invalid_cadence)
        )
        score = max(0.0, 100 - ((penalties * 100) / self.view_count))
        return round(score, 1)

    def to_dict(self) -> Dict[str, object]:
        return {
            "catalog_name": self.catalog_name,
            "version": self.version,
            "view_count": self.view_count,
            "subscription_count": self.subscription_count,
            "duplicate_view_names": {
                key: list(values) for key, values in self.duplicate_view_names.items()
            },
            "invalid_visibility_views": list(self.invalid_visibility_views),
            "views_missing_filters": list(self.views_missing_filters),
            "duplicate_default_views": {
                key: list(values) for key, values in self.duplicate_default_views.items()
            },
            "orphan_subscriptions": list(self.orphan_subscriptions),
            "subscriptions_missing_recipients": list(self.subscriptions_missing_recipients),
            "subscriptions_with_invalid_channel": list(self.subscriptions_with_invalid_channel),
            "subscriptions_with_invalid_cadence": list(self.subscriptions_with_invalid_cadence),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SavedViewCatalogAudit":
        return cls(
            catalog_name=str(data["catalog_name"]),
            version=str(data["version"]),
            view_count=int(data.get("view_count", 0)),
            subscription_count=int(data.get("subscription_count", 0)),
            duplicate_view_names={
                str(key): [str(value) for value in values]
                for key, values in dict(data.get("duplicate_view_names", {})).items()
            },
            invalid_visibility_views=[
                str(name) for name in data.get("invalid_visibility_views", [])
            ],
            views_missing_filters=[str(name) for name in data.get("views_missing_filters", [])],
            duplicate_default_views={
                str(key): [str(value) for value in values]
                for key, values in dict(data.get("duplicate_default_views", {})).items()
            },
            orphan_subscriptions=[str(name) for name in data.get("orphan_subscriptions", [])],
            subscriptions_missing_recipients=[
                str(name) for name in data.get("subscriptions_missing_recipients", [])
            ],
            subscriptions_with_invalid_channel=[
                str(name) for name in data.get("subscriptions_with_invalid_channel", [])
            ],
            subscriptions_with_invalid_cadence=[
                str(name) for name in data.get("subscriptions_with_invalid_cadence", [])
            ],
        )


class SavedViewLibrary:
    def audit(self, catalog: SavedViewCatalog) -> SavedViewCatalogAudit:
        duplicate_view_names: Dict[str, List[str]] = {}
        invalid_visibility_views: List[str] = []
        views_missing_filters: List[str] = []
        duplicate_default_views: Dict[str, List[str]] = {}
        orphan_subscriptions: List[str] = []
        subscriptions_missing_recipients: List[str] = []
        subscriptions_with_invalid_channel: List[str] = []
        subscriptions_with_invalid_cadence: List[str] = []
        names_by_scope: Dict[str, List[str]] = {}
        defaults_by_scope: Dict[str, List[str]] = {}

        for view in catalog.views:
            scope = f"{view.route}:{view.owner}"
            names_by_scope.setdefault(scope, []).append(view.name)
            if view.is_default:
                defaults_by_scope.setdefault(scope, []).append(view.name)
            if view.visibility not in VALID_VIEW_VISIBILITY:
                invalid_visibility_views.append(view.name)
            if not view.filters:
                views_missing_filters.append(view.name)

        for scope, names in sorted(names_by_scope.items()):
            unique_names = sorted({name for name in names if names.count(name) > 1})
            if unique_names:
                duplicate_view_names[scope] = unique_names

        for scope, names in sorted(defaults_by_scope.items()):
            if len(names) > 1:
                duplicate_default_views[scope] = sorted(names)

        view_index = catalog.view_index
        for subscription in catalog.subscriptions:
            if subscription.saved_view_id not in view_index:
                orphan_subscriptions.append(subscription.subscription_id)
            if not subscription.recipients:
                subscriptions_missing_recipients.append(subscription.subscription_id)
            if subscription.channel not in VALID_DIGEST_CHANNELS:
                subscriptions_with_invalid_channel.append(subscription.subscription_id)
            if subscription.cadence not in VALID_DIGEST_CADENCES:
                subscriptions_with_invalid_cadence.append(subscription.subscription_id)

        return SavedViewCatalogAudit(
            catalog_name=catalog.name,
            version=catalog.version,
            view_count=len(catalog.views),
            subscription_count=len(catalog.subscriptions),
            duplicate_view_names=duplicate_view_names,
            invalid_visibility_views=sorted(invalid_visibility_views),
            views_missing_filters=sorted(views_missing_filters),
            duplicate_default_views=duplicate_default_views,
            orphan_subscriptions=sorted(orphan_subscriptions),
            subscriptions_missing_recipients=sorted(subscriptions_missing_recipients),
            subscriptions_with_invalid_channel=sorted(subscriptions_with_invalid_channel),
            subscriptions_with_invalid_cadence=sorted(subscriptions_with_invalid_cadence),
        )


def render_saved_view_report(catalog: SavedViewCatalog, audit: SavedViewCatalogAudit) -> str:
    lines = [
        "# Saved Views & Alert Digests Report",
        "",
        f"- Name: {catalog.name}",
        f"- Version: {catalog.version}",
        f"- Saved Views: {audit.view_count}",
        f"- Alert Subscriptions: {audit.subscription_count}",
        f"- Readiness Score: {audit.readiness_score:.1f}",
        "",
        "## Saved Views",
        "",
    ]

    if catalog.views:
        for view in catalog.views:
            filters = ", ".join(
                f"{view_filter.field}{view_filter.operator}{view_filter.value}"
                for view_filter in view.filters
            ) or "none"
            lines.append(
                f"- {view.name}: route={view.route} owner={view.owner} visibility={view.visibility} "
                f"filters={filters} sort={view.sort_by or 'none'} pinned={view.pinned} default={view.is_default}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Alert Digests", ""])
    if catalog.subscriptions:
        for subscription in catalog.subscriptions:
            recipients = ", ".join(subscription.recipients) or "none"
            lines.append(
                f"- {subscription.subscription_id}: view={subscription.saved_view_id} channel={subscription.channel} "
                f"cadence={subscription.cadence} recipients={recipients} "
                f"include_empty={subscription.include_empty_results} muted={subscription.muted}"
            )
    else:
        lines.append("- None")

    lines.extend(["", "## Gaps", ""])
    duplicate_names = (
        "; ".join(f"{scope}={', '.join(names)}" for scope, names in audit.duplicate_view_names.items())
        if audit.duplicate_view_names
        else "none"
    )
    duplicate_defaults = (
        "; ".join(f"{scope}={', '.join(names)}" for scope, names in audit.duplicate_default_views.items())
        if audit.duplicate_default_views
        else "none"
    )
    lines.append(f"- Duplicate view names: {duplicate_names}")
    lines.append(
        f"- Invalid view visibility: {', '.join(audit.invalid_visibility_views) if audit.invalid_visibility_views else 'none'}"
    )
    lines.append(
        f"- Views missing filters: {', '.join(audit.views_missing_filters) if audit.views_missing_filters else 'none'}"
    )
    lines.append(f"- Duplicate default views: {duplicate_defaults}")
    lines.append(
        f"- Orphan subscriptions: {', '.join(audit.orphan_subscriptions) if audit.orphan_subscriptions else 'none'}"
    )
    lines.append(
        "- Subscriptions missing recipients: "
        f"{', '.join(audit.subscriptions_missing_recipients) if audit.subscriptions_missing_recipients else 'none'}"
    )
    lines.append(
        "- Subscriptions with invalid channel: "
        f"{', '.join(audit.subscriptions_with_invalid_channel) if audit.subscriptions_with_invalid_channel else 'none'}"
    )
    lines.append(
        "- Subscriptions with invalid cadence: "
        f"{', '.join(audit.subscriptions_with_invalid_cadence) if audit.subscriptions_with_invalid_cadence else 'none'}"
    )
    return "\n".join(lines) + "\n"


@dataclass
class PilotKPI:
    name: str
    target: float
    actual: float
    higher_is_better: bool = True

    @property
    def met(self) -> bool:
        return self.actual >= self.target if self.higher_is_better else self.actual <= self.target


@dataclass
class PilotImplementationResult:
    customer: str
    environment: str
    kpis: List[PilotKPI] = field(default_factory=list)
    production_runs: int = 0
    incidents: int = 0

    @property
    def kpi_pass_rate(self) -> float:
        if not self.kpis:
            return 0.0
        passed = len([k for k in self.kpis if k.met])
        return round((passed / len(self.kpis)) * 100, 1)

    @property
    def ready(self) -> bool:
        return self.production_runs > 0 and self.incidents == 0 and self.kpi_pass_rate >= 80.0


def render_pilot_implementation_report(result: PilotImplementationResult) -> str:
    lines = [
        "# Pilot Implementation Report",
        "",
        f"- Customer: {result.customer}",
        f"- Environment: {result.environment}",
        f"- Production Runs: {result.production_runs}",
        f"- Incidents: {result.incidents}",
        f"- KPI Pass Rate: {result.kpi_pass_rate}%",
        f"- Ready: {result.ready}",
        "",
        "## KPI Details",
        "",
    ]
    if result.kpis:
        for kpi in result.kpis:
            lines.append(f"- {kpi.name}: target={kpi.target} actual={kpi.actual} met={kpi.met}")
    else:
        lines.append("- none")
    return "\n".join(lines) + "\n"


class GitSyncError(RuntimeError):
    """Raised when repository sync automation cannot complete safely."""


@dataclass
class RepoSyncStatus:
    branch: str
    local_sha: str
    remote_sha: str
    dirty: bool
    remote_exists: bool
    synced: bool
    pushed: bool = False

    def to_dict(self) -> dict[str, Any]:
        return asdict(self)


@dataclass
class CommandResult:
    stdout: str
    stderr: str
    returncode: int


EXECUTABLE_BITS = stat.S_IXUSR | stat.S_IXGRP | stat.S_IXOTH


def _run(command: Sequence[str], repo: Path) -> CommandResult:
    completed = subprocess.run(
        list(command),
        cwd=repo,
        text=True,
        capture_output=True,
        check=False,
    )
    return CommandResult(
        stdout=completed.stdout.strip(),
        stderr=completed.stderr.strip(),
        returncode=completed.returncode,
    )


def _git(repo: Path, *args: str) -> CommandResult:
    return _run(["git", *args], repo)


def _require_git(repo: Path, *args: str) -> str:
    result = _git(repo, *args)
    if result.returncode != 0:
        detail = result.stderr or result.stdout or f"git {' '.join(args)} failed"
        raise GitSyncError(detail)
    return result.stdout


def _dirty(repo: Path) -> bool:
    return bool(_require_git(repo, "status", "--porcelain"))


def _remote_default_branch(repo: Path, remote: str) -> str:
    symbolic_ref = _git(repo, "symbolic-ref", "--quiet", f"refs/remotes/{remote}/HEAD")
    if symbolic_ref.returncode == 0 and symbolic_ref.stdout:
        prefix = f"refs/remotes/{remote}/"
        if symbolic_ref.stdout.startswith(prefix):
            return symbolic_ref.stdout[len(prefix) :]

    symref_result = _git(repo, "ls-remote", "--symref", remote, "HEAD")
    if symref_result.returncode != 0:
        detail = symref_result.stderr or symref_result.stdout or f"git ls-remote --symref failed for {remote}/HEAD"
        raise GitSyncError(detail)

    for line in symref_result.stdout.splitlines():
        if line.startswith("ref: ") and line.endswith("\tHEAD"):
            ref = line.split()[1]
            prefix = "refs/heads/"
            if ref.startswith(prefix):
                return ref[len(prefix) :]

    raise GitSyncError(f"Could not determine default branch for remote {remote}")


def _remote_branch_sha(repo: Path, remote: str, branch: str) -> str:
    local_ref = _git(repo, "rev-parse", f"refs/remotes/{remote}/{branch}")
    if local_ref.returncode == 0 and local_ref.stdout:
        return local_ref.stdout

    remote_result = _git(repo, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    return remote_result.stdout.split()[0] if remote_result.stdout else ""


def _matches_remote_default_branch(repo: Path, remote: str, local_sha: str) -> bool:
    try:
        default_branch = _remote_default_branch(repo, remote)
        default_sha = _remote_branch_sha(repo, remote, default_branch)
    except GitSyncError:
        return False

    return bool(default_sha) and local_sha == default_sha


def inspect_repo_sync(repo: Path | str, remote: str = "origin") -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    branch = _require_git(repo_path, "branch", "--show-current")
    if not branch:
        raise GitSyncError("Detached HEAD does not support issue branch sync automation")

    local_sha = _require_git(repo_path, "rev-parse", "HEAD")
    remote_result = _git(repo_path, "ls-remote", "--heads", remote, branch)
    if remote_result.returncode != 0:
        detail = remote_result.stderr or remote_result.stdout or f"git ls-remote failed for {remote}/{branch}"
        raise GitSyncError(detail)

    remote_sha = remote_result.stdout.split()[0] if remote_result.stdout else ""
    dirty = _dirty(repo_path)
    remote_exists = bool(remote_sha)
    synced = remote_exists and local_sha == remote_sha

    if not remote_exists and _matches_remote_default_branch(repo_path, remote, local_sha):
        synced = True

    return RepoSyncStatus(
        branch=branch,
        local_sha=local_sha,
        remote_sha=remote_sha,
        dirty=dirty,
        remote_exists=remote_exists,
        synced=synced,
    )


def install_git_hooks(repo: Path | str, hooks_path: str = ".githooks") -> Path:
    repo_path = Path(repo).resolve()
    hooks_dir = repo_path / hooks_path
    if not hooks_dir.is_dir():
        raise GitSyncError(f"Missing hooks directory: {hooks_dir}")

    _require_git(repo_path, "config", "core.hooksPath", hooks_path)
    for hook in hooks_dir.iterdir():
        if hook.is_file():
            hook.chmod(hook.stat().st_mode | EXECUTABLE_BITS)
    return hooks_dir


def ensure_repo_sync(
    repo: Path | str,
    remote: str = "origin",
    auto_push: bool = True,
    allow_dirty: bool = False,
) -> RepoSyncStatus:
    repo_path = Path(repo).resolve()
    status = inspect_repo_sync(repo_path, remote=remote)

    if status.dirty:
        if allow_dirty:
            return status
        raise GitSyncError("Working tree is dirty; commit or stash changes before syncing")

    if status.remote_exists and not status.synced:
        fetch_result = _git(repo_path, "fetch", remote, status.branch)
        if fetch_result.returncode != 0:
            detail = fetch_result.stderr or fetch_result.stdout or f"git fetch {remote} {status.branch} failed"
            raise GitSyncError(detail)

        ff_result = _git(repo_path, "pull", "--ff-only", remote, status.branch)
        if ff_result.returncode != 0:
            detail = ff_result.stderr or ff_result.stdout or f"git pull --ff-only {remote} {status.branch} failed"
            raise GitSyncError(detail)
        status = inspect_repo_sync(repo_path, remote=remote)

    if not auto_push or status.synced:
        return status

    push_args = ["push", remote, "HEAD"]
    if not status.remote_exists:
        push_args = ["push", "-u", remote, "HEAD"]

    push_result = _git(repo_path, *push_args)
    if push_result.returncode != 0:
        detail = push_result.stderr or push_result.stdout or f"git {' '.join(push_args)} failed"
        raise GitSyncError(detail)

    refreshed = inspect_repo_sync(repo_path, remote=remote)
    refreshed.pushed = True
    if not refreshed.synced:
        raise GitSyncError(
            f"Remote SHA mismatch after push: local={refreshed.local_sha} remote={refreshed.remote_sha or 'missing'}"
        )
    return refreshed


PULL_REQUEST_COMMENT_EVENT = "pull_request.comment"
CI_COMPLETED_EVENT = "ci.completed"
TASK_FAILED_EVENT = "task.failed"

EventSubscriber = Callable[["BusEvent", TaskRun], None]


@dataclass(frozen=True)
class BusEvent:
    event_type: str
    run_id: str
    actor: str
    details: Dict[str, Any] = field(default_factory=dict)
    timestamp: str = field(default_factory=utc_now)


class EventBus:
    def __init__(self, ledger: Optional[ObservabilityLedger] = None):
        self.ledger = ledger
        self._runs: Dict[str, TaskRun] = {}
        self._subscribers: DefaultDict[str, List[EventSubscriber]] = defaultdict(list)

    def register_run(self, run: TaskRun) -> None:
        self._runs[run.run_id] = run

    def subscribe(self, event_type: str, handler: EventSubscriber) -> None:
        self._subscribers[event_type].append(handler)

    def publish(self, event: BusEvent) -> TaskRun:
        run = self._resolve_run(event.run_id)
        previous_status = run.status
        self._record_event(run, event)

        next_status, summary = self._resolve_transition(run, event)
        if next_status:
            run.finalize(next_status, summary)
            run.audit(
                "event_bus.transition",
                "event-bus",
                next_status,
                event_type=event.event_type,
                previous_status=previous_status,
                status=next_status,
                summary=summary,
                event_timestamp=event.timestamp,
            )

        for handler in self._subscribers.get(event.event_type, []):
            handler(event, run)

        if self.ledger is not None:
            self.ledger.upsert(run)
        return run

    def _resolve_run(self, run_id: str) -> TaskRun:
        registered = self._runs.get(run_id)
        if registered is not None:
            return registered
        if self.ledger is not None:
            for run in self.ledger.load_runs():
                if run.run_id == run_id:
                    self._runs[run_id] = run
                    return run
        raise KeyError(f"run {run_id!r} is not registered with the event bus")

    def _record_event(self, run: TaskRun, event: BusEvent) -> None:
        run.audit(
            "event_bus.event",
            event.actor,
            "received",
            event_type=event.event_type,
            event_timestamp=event.timestamp,
            **event.details,
        )
        if event.event_type != PULL_REQUEST_COMMENT_EVENT:
            return
        body = str(event.details.get("body", "")).strip()
        if not body:
            return
        mentions = [str(item) for item in event.details.get("mentions", [])]
        run.add_comment(
            author=event.actor,
            body=body,
            mentions=mentions,
            anchor="pull-request",
            surface="pull-request",
        )

    def _resolve_transition(self, run: TaskRun, event: BusEvent) -> tuple[str, str]:
        explicit_status = str(event.details.get("target_status", "")).strip()
        if explicit_status:
            return explicit_status, self._build_summary(event, explicit_status)

        if event.event_type == PULL_REQUEST_COMMENT_EVENT:
            decision = str(event.details.get("decision", "")).strip().lower()
            if decision in {"approved", "accept", "accepted", "lgtm"}:
                return "approved", self._build_summary(event, "approved")
            if decision in {"blocked", "changes-requested", "rejected"}:
                return "needs-approval", self._build_summary(event, "needs-approval")
        elif event.event_type == CI_COMPLETED_EVENT:
            conclusion = str(event.details.get("conclusion", "")).strip().lower()
            if conclusion in {"success", "passed", "green"}:
                return "completed", self._build_summary(event, "completed")
            if conclusion in {"cancelled", "canceled", "error", "failed", "failure", "timed_out"}:
                return "failed", self._build_summary(event, "failed")
        elif event.event_type == TASK_FAILED_EVENT:
            return "failed", self._build_summary(event, "failed")

        return "", run.summary

    def _build_summary(self, event: BusEvent, status: str) -> str:
        summary = str(event.details.get("summary", "")).strip()
        if summary:
            return summary
        if event.event_type == PULL_REQUEST_COMMENT_EVENT:
            body = str(event.details.get("body", "")).strip()
            if body:
                return body
            return f"pull request comment set run to {status}"
        if event.event_type == CI_COMPLETED_EVENT:
            workflow = str(event.details.get("workflow", "")).strip()
            conclusion = str(event.details.get("conclusion", "")).strip() or status
            if workflow:
                return f"CI workflow {workflow} completed with {conclusion}"
            return f"CI completed with {conclusion}"
        if event.event_type == TASK_FAILED_EVENT:
            reason = str(event.details.get("error", "")).strip() or str(event.details.get("reason", "")).strip()
            if reason:
                return reason
            return "task failed"
        return status


@dataclass(frozen=True)
class SchemaField:
    name: str
    field_type: str
    required: bool = True
    description: str = ""

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "field_type": self.field_type,
            "required": self.required,
            "description": self.description,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SchemaField":
        return cls(
            name=str(data["name"]),
            field_type=str(data["field_type"]),
            required=bool(data.get("required", True)),
            description=str(data.get("description", "")),
        )


@dataclass
class SurfaceSchema:
    name: str
    owner: str
    description: str = ""
    fields: List[SchemaField] = field(default_factory=list)
    sample: Dict[str, object] = field(default_factory=dict)

    @property
    def required_fields(self) -> List[str]:
        return [field.name for field in self.fields if field.required]

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "owner": self.owner,
            "description": self.description,
            "fields": [field.to_dict() for field in self.fields],
            "sample": self.sample,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "SurfaceSchema":
        return cls(
            name=str(data["name"]),
            owner=str(data.get("owner", "")),
            description=str(data.get("description", "")),
            fields=[SchemaField.from_dict(field) for field in data.get("fields", [])],
            sample=dict(data.get("sample", {})),
        )


@dataclass
class DashboardRunContract:
    contract_id: str
    version: str
    dashboard_schema: SurfaceSchema
    run_detail_schema: SurfaceSchema

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "dashboard_schema": self.dashboard_schema.to_dict(),
            "run_detail_schema": self.run_detail_schema.to_dict(),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardRunContract":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            dashboard_schema=SurfaceSchema.from_dict(dict(data["dashboard_schema"])),
            run_detail_schema=SurfaceSchema.from_dict(dict(data["run_detail_schema"])),
        )


@dataclass
class DashboardRunContractAudit:
    contract_id: str
    version: str
    dashboard_missing_fields: List[str] = field(default_factory=list)
    dashboard_sample_gaps: List[str] = field(default_factory=list)
    run_detail_missing_fields: List[str] = field(default_factory=list)
    run_detail_sample_gaps: List[str] = field(default_factory=list)

    @property
    def release_ready(self) -> bool:
        return not (
            self.dashboard_missing_fields
            or self.dashboard_sample_gaps
            or self.run_detail_missing_fields
            or self.run_detail_sample_gaps
        )

    def to_dict(self) -> Dict[str, object]:
        return {
            "contract_id": self.contract_id,
            "version": self.version,
            "dashboard_missing_fields": list(self.dashboard_missing_fields),
            "dashboard_sample_gaps": list(self.dashboard_sample_gaps),
            "run_detail_missing_fields": list(self.run_detail_missing_fields),
            "run_detail_sample_gaps": list(self.run_detail_sample_gaps),
            "release_ready": self.release_ready,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, object]) -> "DashboardRunContractAudit":
        return cls(
            contract_id=str(data["contract_id"]),
            version=str(data["version"]),
            dashboard_missing_fields=[str(item) for item in data.get("dashboard_missing_fields", [])],
            dashboard_sample_gaps=[str(item) for item in data.get("dashboard_sample_gaps", [])],
            run_detail_missing_fields=[str(item) for item in data.get("run_detail_missing_fields", [])],
            run_detail_sample_gaps=[str(item) for item in data.get("run_detail_sample_gaps", [])],
        )


class DashboardRunContractLibrary:
    DASHBOARD_REQUIRED_FIELDS = [
        "dashboard_id",
        "generated_at",
        "period.label",
        "period.start",
        "period.end",
        "filters.team",
        "summary.total_runs",
        "summary.success_rate",
        "summary.approval_queue_depth",
        "summary.sla_breach_count",
        "kpis",
        "kpis[].name",
        "kpis[].value",
        "kpis[].target",
        "funnel",
        "blockers",
        "activity",
    ]
    RUN_DETAIL_REQUIRED_FIELDS = [
        "run_id",
        "task_id",
        "status",
        "started_at",
        "ended_at",
        "summary",
        "timeline",
        "timeline[].event_id",
        "timeline[].status",
        "artifacts",
        "closeout.validation_evidence",
        "closeout.git_push_succeeded",
        "closeout.git_log_stat_output",
    ]

    def audit(self, contract: DashboardRunContract) -> DashboardRunContractAudit:
        return DashboardRunContractAudit(
            contract_id=contract.contract_id,
            version=contract.version,
            dashboard_missing_fields=self._missing_field_defs(
                self.DASHBOARD_REQUIRED_FIELDS,
                contract.dashboard_schema.fields,
            ),
            dashboard_sample_gaps=self._missing_sample_paths(
                self.DASHBOARD_REQUIRED_FIELDS,
                contract.dashboard_schema.sample,
            ),
            run_detail_missing_fields=self._missing_field_defs(
                self.RUN_DETAIL_REQUIRED_FIELDS,
                contract.run_detail_schema.fields,
            ),
            run_detail_sample_gaps=self._missing_sample_paths(
                self.RUN_DETAIL_REQUIRED_FIELDS,
                contract.run_detail_schema.sample,
            ),
        )

    def build_default_contract(self) -> DashboardRunContract:
        return DashboardRunContract(
            contract_id="BIG-4301",
            version="v1",
            dashboard_schema=SurfaceSchema(
                name="DashboardKpiAggregate",
                owner="operations",
                description="Team dashboard response for KPI cards, funnel, blockers, and recent activity.",
                fields=[
                    SchemaField("dashboard_id", "string", description="Stable dashboard identifier."),
                    SchemaField("generated_at", "datetime", description="UTC generation timestamp."),
                    SchemaField("period.label", "string", description="Human-readable period label."),
                    SchemaField("period.start", "date", description="Inclusive reporting window start."),
                    SchemaField("period.end", "date", description="Inclusive reporting window end."),
                    SchemaField("filters.team", "string", description="Team or org filter applied."),
                    SchemaField("filters.viewer_role", "string", required=False, description="Persona used for visibility rules."),
                    SchemaField("summary.total_runs", "integer", description="Total runs in the period."),
                    SchemaField("summary.success_rate", "number", description="Completed-success ratio in percent."),
                    SchemaField("summary.approval_queue_depth", "integer", description="Runs waiting for approval."),
                    SchemaField("summary.sla_breach_count", "integer", description="Runs over SLA."),
                    SchemaField("kpis", "DashboardKpi[]", description="Ordered KPI cards shown in the hero grid."),
                    SchemaField("kpis[].name", "string", description="Metric identifier."),
                    SchemaField("kpis[].value", "number", description="Observed metric value."),
                    SchemaField("kpis[].target", "number", description="Target threshold."),
                    SchemaField("kpis[].unit", "string", required=False, description="Display suffix or unit."),
                    SchemaField("kpis[].direction", "string", required=False, description="Healthy trend direction."),
                    SchemaField("kpis[].healthy", "boolean", required=False, description="Precomputed health state."),
                    SchemaField("funnel", "DashboardFunnelStage[]", description="Pipeline distribution."),
                    SchemaField("blockers", "DashboardBlocker[]", description="Highest impact blockers."),
                    SchemaField("activity", "DashboardActivity[]", description="Most recent run activity."),
                ],
                sample={
                    "dashboard_id": "eng-overview-core-product",
                    "generated_at": "2026-03-11T09:30:00Z",
                    "period": {
                        "label": "2026-W11",
                        "start": "2026-03-09",
                        "end": "2026-03-15",
                    },
                    "filters": {
                        "team": "core-product",
                        "viewer_role": "engineering-manager",
                    },
                    "summary": {
                        "total_runs": 42,
                        "success_rate": 88.1,
                        "approval_queue_depth": 3,
                        "sla_breach_count": 2,
                    },
                    "kpis": [
                        {
                            "name": "success-rate",
                            "value": 88.1,
                            "target": 90.0,
                            "unit": "%",
                            "direction": "up",
                            "healthy": False,
                        },
                        {
                            "name": "average-cycle-minutes",
                            "value": 47.3,
                            "target": 60.0,
                            "unit": "m",
                            "direction": "down",
                            "healthy": True,
                        },
                    ],
                    "funnel": [
                        {"name": "queued", "count": 5, "share": 11.9},
                        {"name": "in-progress", "count": 7, "share": 16.7},
                        {"name": "awaiting-approval", "count": 3, "share": 7.1},
                        {"name": "completed", "count": 27, "share": 64.3},
                    ],
                    "blockers": [
                        {
                            "summary": "Security scan failures on release branch",
                            "affected_runs": 2,
                            "affected_tasks": ["OPE-121", "OPE-127"],
                            "owner": "security",
                            "severity": "high",
                        }
                    ],
                    "activity": [
                        {
                            "timestamp": "2026-03-11T09:20:00Z",
                            "run_id": "run-204",
                            "task_id": "OPE-127",
                            "status": "failed",
                            "summary": "Security scan failed after dependency bump",
                        }
                    ],
                },
            ),
            run_detail_schema=SurfaceSchema(
                name="RunDetail",
                owner="runtime",
                description="Canonical run detail payload for replay, timeline inspection, artifacts, and closeout evidence.",
                fields=[
                    SchemaField("run_id", "string", description="Stable run identifier."),
                    SchemaField("task_id", "string", description="Parent task identifier."),
                    SchemaField("status", "string", description="Current run status."),
                    SchemaField("started_at", "datetime", description="Run start timestamp."),
                    SchemaField("ended_at", "datetime", description="Run end timestamp."),
                    SchemaField("summary", "string", description="Operator-readable summary."),
                    SchemaField("medium", "string", required=False, description="Execution medium."),
                    SchemaField("timeline", "RunTimelineEvent[]", description="Chronological execution events."),
                    SchemaField("timeline[].event_id", "string", description="Stable event identifier."),
                    SchemaField("timeline[].lane", "string", required=False, description="UI lane or track."),
                    SchemaField("timeline[].timestamp", "datetime", required=False, description="Event timestamp."),
                    SchemaField("timeline[].status", "string", description="Event outcome."),
                    SchemaField("artifacts", "RunArtifact[]", description="Artifacts emitted by the run."),
                    SchemaField("closeout.validation_evidence", "string[]", description="Validation proof captured before finish."),
                    SchemaField("closeout.git_push_succeeded", "boolean", description="Whether push completed successfully."),
                    SchemaField("closeout.git_push_output", "string", required=False, description="Push command output."),
                    SchemaField("closeout.git_log_stat_output", "string", description="Captured git log --stat output."),
                ],
                sample={
                    "run_id": "run-204",
                    "task_id": "OPE-127",
                    "status": "completed",
                    "started_at": "2026-03-11T08:58:00Z",
                    "ended_at": "2026-03-11T09:24:00Z",
                    "medium": "codex-cli",
                    "summary": "Shipped release-branch scan hardening and captured closeout evidence.",
                    "timeline": [
                        {
                            "event_id": "evt-1",
                            "lane": "analysis",
                            "timestamp": "2026-03-11T08:59:00Z",
                            "status": "completed",
                            "title": "Inspected release branch failures",
                            "summary": "Reviewed the failing dependency diff and scan output.",
                            "details": ["Focused on lockfile drift and transitive CVE triage."],
                        },
                        {
                            "event_id": "evt-2",
                            "lane": "validation",
                            "timestamp": "2026-03-11T09:22:00Z",
                            "status": "completed",
                            "title": "Validated fix",
                            "summary": "Executed targeted tests and recorded the validation report.",
                            "details": ["pytest passed", "report stored under reports/OPE-127-validation.md"],
                        },
                    ],
                    "artifacts": [
                        {
                            "name": "validation-report",
                            "kind": "report",
                            "path": "reports/OPE-127-validation.md",
                            "timestamp": "2026-03-11T09:23:00Z",
                            "sha256": "abc123",
                            "metadata": {"ticket": "OPE-127"},
                        }
                    ],
                    "closeout": {
                        "validation_evidence": [
                            "python3 -m pytest tests/test_security.py -> 7 passed",
                            "python3 -m pytest -> 126 passed",
                        ],
                        "git_push_succeeded": True,
                        "git_push_output": "To github.com:OpenAGIs/BigClaw.git\\n   abc123..def456  main -> main",
                        "git_log_stat_output": "commit def456\\n src/bigclaw/security.py | 12 ++++++++++--",
                    },
                },
            ),
        )

    def _missing_field_defs(self, required_fields: Sequence[str], fields: Sequence[SchemaField]) -> List[str]:
        defined = {field.name for field in fields}
        return [field_name for field_name in required_fields if field_name not in defined]

    def _missing_sample_paths(self, required_fields: Sequence[str], payload: Dict[str, object]) -> List[str]:
        return [field_name for field_name in required_fields if not self._path_exists(payload, field_name)]

    def _path_exists(self, payload: object, path: str) -> bool:
        current_items = [payload]
        for part in path.split("."):
            next_items: List[object] = []
            is_list = part.endswith("[]")
            key = part[:-2] if is_list else part
            for item in current_items:
                if not isinstance(item, dict) or key not in item:
                    continue
                value = item[key]
                if is_list:
                    if isinstance(value, list) and value:
                        next_items.extend(value)
                    else:
                        continue
                else:
                    next_items.append(value)
            if not next_items:
                return False
            current_items = next_items
        return True


def render_dashboard_run_contract_report(
    contract: DashboardRunContract,
    audit: DashboardRunContractAudit,
) -> str:
    sections = [
        "# Dashboard and Run Contract",
        "",
        f"- Contract ID: {contract.contract_id}",
        f"- Version: {contract.version}",
        f"- Release Ready: {audit.release_ready}",
        "",
        "## Dashboard KPI Aggregate",
        f"- Name: {contract.dashboard_schema.name}",
        f"- Owner: {contract.dashboard_schema.owner}",
        f"- Required Fields: {', '.join(contract.dashboard_schema.required_fields)}",
        f"- Missing Required Fields: {', '.join(audit.dashboard_missing_fields) or 'none'}",
        f"- Sample Gaps: {', '.join(audit.dashboard_sample_gaps) or 'none'}",
        "",
        "```json",
        json.dumps(contract.dashboard_schema.sample, indent=2, sort_keys=True),
        "```",
        "",
        "## Run Detail",
        f"- Name: {contract.run_detail_schema.name}",
        f"- Owner: {contract.run_detail_schema.owner}",
        f"- Required Fields: {', '.join(contract.run_detail_schema.required_fields)}",
        f"- Missing Required Fields: {', '.join(audit.run_detail_missing_fields) or 'none'}",
        f"- Sample Gaps: {', '.join(audit.run_detail_sample_gaps) or 'none'}",
        "",
        "```json",
        json.dumps(contract.run_detail_schema.sample, indent=2, sort_keys=True),
        "```",
    ]
    return "\n".join(sections)
