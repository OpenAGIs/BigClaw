import hashlib
import json
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

from .audit_events import missing_required_fields
from .collaboration import CollaborationComment, DecisionNote
from .models import Task

VALID_RUN_COMMIT_ROLES = {"source", "candidate", "closeout", "accepted"}


@dataclass
class RunCommitLink:
    run_id: str
    commit_hash: str
    role: str
    repo_space_id: str
    actor: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "run_id": self.run_id,
            "commit_hash": self.commit_hash,
            "role": self.role,
            "repo_space_id": self.repo_space_id,
            "actor": self.actor,
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RunCommitLink":
        return cls(
            run_id=str(data["run_id"]),
            commit_hash=str(data["commit_hash"]),
            role=str(data["role"]),
            repo_space_id=str(data["repo_space_id"]),
            actor=str(data.get("actor", "")),
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class RunCommitBinding:
    links: List[RunCommitLink]

    @property
    def accepted_commit_hash(self) -> str:
        for link in self.links:
            if link.role == "accepted":
                return link.commit_hash
        return ""


def validate_run_commit_roles(links: List[RunCommitLink]) -> None:
    invalid = [link.role for link in links if link.role not in VALID_RUN_COMMIT_ROLES]
    if invalid:
        invalid_text = ", ".join(sorted(set(invalid)))
        raise ValueError(f"unsupported run commit roles: {invalid_text}")


def bind_run_commits(links: List[RunCommitLink]) -> RunCommitBinding:
    validate_run_commit_roles(links)
    return RunCommitBinding(links=list(links))


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def sha256_file(path: str) -> str:
    file_path = Path(path)
    if not file_path.exists() or not file_path.is_file():
        return ""

    digest = hashlib.sha256()
    with file_path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(8192), b""):
            digest.update(chunk)
    return digest.hexdigest()


@dataclass
class LogEntry:
    level: str
    message: str
    timestamp: str = field(default_factory=utc_now)
    context: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "level": self.level,
            "message": self.message,
            "timestamp": self.timestamp,
            "context": self.context,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "LogEntry":
        return cls(
            level=data["level"],
            message=data["message"],
            timestamp=data.get("timestamp", utc_now()),
            context=data.get("context", {}),
        )


@dataclass
class TraceEntry:
    span: str
    status: str
    timestamp: str = field(default_factory=utc_now)
    attributes: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "span": self.span,
            "status": self.status,
            "timestamp": self.timestamp,
            "attributes": self.attributes,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TraceEntry":
        return cls(
            span=data["span"],
            status=data["status"],
            timestamp=data.get("timestamp", utc_now()),
            attributes=data.get("attributes", {}),
        )


@dataclass
class ArtifactRecord:
    name: str
    kind: str
    path: str
    timestamp: str = field(default_factory=utc_now)
    sha256: str = ""
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "name": self.name,
            "kind": self.kind,
            "path": self.path,
            "timestamp": self.timestamp,
            "sha256": self.sha256,
            "metadata": self.metadata,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "ArtifactRecord":
        return cls(
            name=data["name"],
            kind=data["kind"],
            path=data["path"],
            timestamp=data.get("timestamp", utc_now()),
            sha256=data.get("sha256", ""),
            metadata=data.get("metadata", {}),
        )


@dataclass
class AuditEntry:
    action: str
    actor: str
    outcome: str
    timestamp: str = field(default_factory=utc_now)
    details: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "action": self.action,
            "actor": self.actor,
            "outcome": self.outcome,
            "timestamp": self.timestamp,
            "details": self.details,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "AuditEntry":
        return cls(
            action=data["action"],
            actor=data["actor"],
            outcome=data["outcome"],
            timestamp=data.get("timestamp", utc_now()),
            details=data.get("details", {}),
        )


@dataclass
class GitSyncTelemetry:
    status: str = "unknown"
    failure_category: str = ""
    summary: str = ""
    branch: str = ""
    remote: str = "origin"
    remote_ref: str = ""
    ahead_by: int = 0
    behind_by: int = 0
    dirty_paths: List[str] = field(default_factory=list)
    auth_target: str = ""
    timestamp: str = field(default_factory=utc_now)

    @property
    def ok(self) -> bool:
        return self.status == "synced"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "status": self.status,
            "failure_category": self.failure_category,
            "summary": self.summary,
            "branch": self.branch,
            "remote": self.remote,
            "remote_ref": self.remote_ref,
            "ahead_by": self.ahead_by,
            "behind_by": self.behind_by,
            "dirty_paths": list(self.dirty_paths),
            "auth_target": self.auth_target,
            "timestamp": self.timestamp,
            "ok": self.ok,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "GitSyncTelemetry":
        return cls(
            status=str(data.get("status", "unknown")),
            failure_category=str(data.get("failure_category", "")),
            summary=str(data.get("summary", "")),
            branch=str(data.get("branch", "")),
            remote=str(data.get("remote", "origin")),
            remote_ref=str(data.get("remote_ref", "")),
            ahead_by=int(data.get("ahead_by", 0)),
            behind_by=int(data.get("behind_by", 0)),
            dirty_paths=[str(item) for item in data.get("dirty_paths", [])],
            auth_target=str(data.get("auth_target", "")),
            timestamp=data.get("timestamp", utc_now()),
        )


@dataclass
class PullRequestFreshness:
    pr_number: Optional[int] = None
    pr_url: str = ""
    branch_state: str = "unknown"
    body_state: str = "unknown"
    branch_head_sha: str = ""
    pr_head_sha: str = ""
    expected_body_digest: str = ""
    actual_body_digest: str = ""
    checked_at: str = field(default_factory=utc_now)

    @property
    def fresh(self) -> bool:
        return self.branch_state == "in-sync" and self.body_state == "fresh"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "pr_number": self.pr_number,
            "pr_url": self.pr_url,
            "branch_state": self.branch_state,
            "body_state": self.body_state,
            "branch_head_sha": self.branch_head_sha,
            "pr_head_sha": self.pr_head_sha,
            "expected_body_digest": self.expected_body_digest,
            "actual_body_digest": self.actual_body_digest,
            "checked_at": self.checked_at,
            "fresh": self.fresh,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "PullRequestFreshness":
        pr_number = data.get("pr_number")
        return cls(
            pr_number=int(pr_number) if pr_number is not None else None,
            pr_url=str(data.get("pr_url", "")),
            branch_state=str(data.get("branch_state", "unknown")),
            body_state=str(data.get("body_state", "unknown")),
            branch_head_sha=str(data.get("branch_head_sha", "")),
            pr_head_sha=str(data.get("pr_head_sha", "")),
            expected_body_digest=str(data.get("expected_body_digest", "")),
            actual_body_digest=str(data.get("actual_body_digest", "")),
            checked_at=data.get("checked_at", utc_now()),
        )


@dataclass
class RepoSyncAudit:
    sync: GitSyncTelemetry = field(default_factory=GitSyncTelemetry)
    pull_request: PullRequestFreshness = field(default_factory=PullRequestFreshness)

    @property
    def summary(self) -> str:
        parts = [f"sync={self.sync.status}"]
        if self.sync.failure_category:
            parts.append(f"failure={self.sync.failure_category}")
        parts.append(f"pr-branch={self.pull_request.branch_state}")
        parts.append(f"pr-body={self.pull_request.body_state}")
        return ", ".join(parts)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "sync": self.sync.to_dict(),
            "pull_request": self.pull_request.to_dict(),
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoSyncAudit":
        return cls(
            sync=GitSyncTelemetry.from_dict(data.get("sync", {})),
            pull_request=PullRequestFreshness.from_dict(data.get("pull_request", {})),
        )


@dataclass
class RunCloseout:
    validation_evidence: List[str] = field(default_factory=list)
    git_push_succeeded: bool = False
    git_push_output: str = ""
    git_log_stat_output: str = ""
    repo_sync_audit: Optional[RepoSyncAudit] = None
    run_commit_links: List[RunCommitLink] = field(default_factory=list)
    accepted_commit_hash: str = ""
    timestamp: str = field(default_factory=utc_now)

    @property
    def complete(self) -> bool:
        return bool(self.validation_evidence) and self.git_push_succeeded and bool(self.git_log_stat_output.strip())

    def to_dict(self) -> Dict[str, Any]:
        return {
            "validation_evidence": self.validation_evidence,
            "git_push_succeeded": self.git_push_succeeded,
            "git_push_output": self.git_push_output,
            "git_log_stat_output": self.git_log_stat_output,
            "repo_sync_audit": self.repo_sync_audit.to_dict() if self.repo_sync_audit else None,
            "run_commit_links": [link.to_dict() for link in self.run_commit_links],
            "accepted_commit_hash": self.accepted_commit_hash,
            "timestamp": self.timestamp,
            "complete": self.complete,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RunCloseout":
        return cls(
            validation_evidence=data.get("validation_evidence", []),
            git_push_succeeded=data.get("git_push_succeeded", False),
            git_push_output=data.get("git_push_output", ""),
            git_log_stat_output=data.get("git_log_stat_output", ""),
            repo_sync_audit=RepoSyncAudit.from_dict(data["repo_sync_audit"]) if data.get("repo_sync_audit") else None,
            run_commit_links=[RunCommitLink.from_dict(item) for item in data.get("run_commit_links", [])],
            accepted_commit_hash=str(data.get("accepted_commit_hash", "")),
            timestamp=data.get("timestamp", utc_now()),
        )


@dataclass
class TaskRun:
    run_id: str
    task_id: str
    source: str
    title: str
    medium: str
    started_at: str = field(default_factory=utc_now)
    ended_at: str = ""
    status: str = "running"
    summary: str = ""
    logs: List[LogEntry] = field(default_factory=list)
    traces: List[TraceEntry] = field(default_factory=list)
    artifacts: List[ArtifactRecord] = field(default_factory=list)
    audits: List[AuditEntry] = field(default_factory=list)
    closeout: RunCloseout = field(default_factory=RunCloseout)

    @classmethod
    def from_task(cls, task: Task, run_id: str, medium: str) -> "TaskRun":
        return cls(
            run_id=run_id,
            task_id=task.task_id,
            source=task.source,
            title=task.title,
            medium=medium,
        )

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "TaskRun":
        return cls(
            run_id=data["run_id"],
            task_id=data["task_id"],
            source=data["source"],
            title=data["title"],
            medium=data["medium"],
            started_at=data.get("started_at", utc_now()),
            ended_at=data.get("ended_at", ""),
            status=data.get("status", "running"),
            summary=data.get("summary", ""),
            logs=[LogEntry.from_dict(entry) for entry in data.get("logs", [])],
            traces=[TraceEntry.from_dict(entry) for entry in data.get("traces", [])],
            artifacts=[ArtifactRecord.from_dict(entry) for entry in data.get("artifacts", [])],
            audits=[AuditEntry.from_dict(entry) for entry in data.get("audits", [])],
            closeout=RunCloseout.from_dict(data.get("closeout", {})),
        )

    def log(self, level: str, message: str, **context: Any) -> None:
        self.logs.append(LogEntry(level=level, message=message, context=context))

    def trace(self, span: str, status: str, **attributes: Any) -> None:
        self.traces.append(TraceEntry(span=span, status=status, attributes=attributes))

    def register_artifact(self, name: str, kind: str, path: str, **metadata: Any) -> None:
        digest = sha256_file(path)
        self.artifacts.append(
            ArtifactRecord(
                name=name,
                kind=kind,
                path=path,
                sha256=digest,
                metadata=metadata,
            )
        )
        self.audit(
            "artifact.registered",
            "task-run",
            "recorded",
            artifact_name=name,
            artifact_kind=kind,
            path=path,
            sha256=digest,
        )

    def audit(self, action: str, actor: str, outcome: str, **details: Any) -> None:
        self.audits.append(AuditEntry(action=action, actor=actor, outcome=outcome, details=details))

    def audit_spec_event(self, action: str, actor: str, outcome: str, **details: Any) -> None:
        missing = missing_required_fields(action, details)
        if missing:
            missing_text = ", ".join(missing)
            raise ValueError(f"audit event {action} missing required fields: {missing_text}")
        self.audit(action, actor, outcome, **details)

    def add_comment(
        self,
        *,
        author: str,
        body: str,
        mentions: Optional[List[str]] = None,
        anchor: str = "",
        status: str = "open",
        comment_id: str = "",
        surface: str = "run",
    ) -> CollaborationComment:
        resolved_id = comment_id or f"{self.run_id}-comment-{len([audit for audit in self.audits if audit.action == 'collaboration.comment']) + 1}"
        comment = CollaborationComment(
            comment_id=resolved_id,
            author=author,
            body=body,
            mentions=list(mentions or []),
            anchor=anchor,
            status=status,
        )
        self.audit(
            "collaboration.comment",
            author,
            "recorded",
            surface=surface,
            comment_id=comment.comment_id,
            body=comment.body,
            mentions=comment.mentions,
            anchor=comment.anchor,
            status=comment.status,
        )
        return comment

    def add_decision_note(
        self,
        *,
        author: str,
        summary: str,
        outcome: str,
        mentions: Optional[List[str]] = None,
        related_comment_ids: Optional[List[str]] = None,
        follow_up: str = "",
        decision_id: str = "",
        surface: str = "run",
    ) -> DecisionNote:
        resolved_id = decision_id or f"{self.run_id}-decision-{len([audit for audit in self.audits if audit.action == 'collaboration.decision']) + 1}"
        decision = DecisionNote(
            decision_id=resolved_id,
            author=author,
            outcome=outcome,
            summary=summary,
            mentions=list(mentions or []),
            related_comment_ids=list(related_comment_ids or []),
            follow_up=follow_up,
        )
        self.audit(
            "collaboration.decision",
            author,
            outcome,
            surface=surface,
            decision_id=decision.decision_id,
            summary=decision.summary,
            mentions=decision.mentions,
            related_comment_ids=decision.related_comment_ids,
            follow_up=decision.follow_up,
        )
        return decision

    def record_closeout(
        self,
        *,
        validation_evidence: List[str],
        git_push_succeeded: bool,
        git_push_output: str = "",
        git_log_stat_output: str = "",
        repo_sync_audit: Optional[RepoSyncAudit] = None,
        run_commit_links: Optional[List[RunCommitLink]] = None,
    ) -> None:
        links = list(run_commit_links or [])
        binding = bind_run_commits(links) if links else None
        self.closeout = RunCloseout(
            validation_evidence=list(validation_evidence),
            git_push_succeeded=git_push_succeeded,
            git_push_output=git_push_output,
            git_log_stat_output=git_log_stat_output,
            repo_sync_audit=repo_sync_audit,
            run_commit_links=links,
            accepted_commit_hash=binding.accepted_commit_hash if binding else "",
        )
        self.audit(
            "closeout.recorded",
            "task-run",
            "recorded",
            validation_evidence_count=len(validation_evidence),
            git_push_succeeded=git_push_succeeded,
            git_log_stat_captured=bool(git_log_stat_output.strip()),
            has_repo_sync_audit=repo_sync_audit is not None,
        )

    def finalize(self, status: str, summary: str) -> None:
        self.status = status
        self.summary = summary
        self.ended_at = utc_now()

    def to_dict(self) -> Dict[str, Any]:
        return {
            "run_id": self.run_id,
            "task_id": self.task_id,
            "source": self.source,
            "title": self.title,
            "medium": self.medium,
            "started_at": self.started_at,
            "ended_at": self.ended_at,
            "status": self.status,
            "summary": self.summary,
            "logs": [entry.to_dict() for entry in self.logs],
            "traces": [entry.to_dict() for entry in self.traces],
            "artifacts": [entry.to_dict() for entry in self.artifacts],
            "audits": [entry.to_dict() for entry in self.audits],
            "closeout": self.closeout.to_dict(),
        }


class ObservabilityLedger:
    def __init__(self, storage_path: str):
        self.storage_path = Path(storage_path)

    def load(self) -> List[Dict[str, Any]]:
        if not self.storage_path.exists():
            return []
        return json.loads(self.storage_path.read_text())

    def _write_entries(self, entries: List[Dict[str, Any]]) -> None:
        self.storage_path.parent.mkdir(parents=True, exist_ok=True)
        self.storage_path.write_text(json.dumps(entries, ensure_ascii=False, indent=2))

    def append(self, run: TaskRun) -> None:
        self.upsert(run)

    def upsert(self, run: TaskRun) -> None:
        entries = self.load()
        serialized = run.to_dict()
        for index, entry in enumerate(entries):
            if entry.get("run_id") == run.run_id:
                entries[index] = serialized
                self._write_entries(entries)
                return
        entries.append(serialized)
        self._write_entries(entries)

    def load_runs(self) -> List[TaskRun]:
        return [TaskRun.from_dict(entry) for entry in self.load()]
