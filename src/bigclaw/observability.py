import hashlib
import json
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

from .audit_events import missing_required_fields
from .collaboration import CollaborationComment, DecisionNote
from .models import Task
from .repo_links import bind_run_commits
from .repo_plane import RunCommitLink


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
class RunCloseout:
    validation_evidence: List[str] = field(default_factory=list)
    git_push_succeeded: bool = False
    git_push_output: str = ""
    git_log_stat_output: str = ""
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
        self.artifacts.append(
            ArtifactRecord(
                name=name,
                kind=kind,
                path=path,
                sha256=sha256_file(path),
                metadata=metadata,
            )
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
        run_commit_links: Optional[List[RunCommitLink]] = None,
    ) -> None:
        links = list(run_commit_links or [])
        binding = bind_run_commits(links) if links else None
        self.closeout = RunCloseout(
            validation_evidence=list(validation_evidence),
            git_push_succeeded=git_push_succeeded,
            git_push_output=git_push_output,
            git_log_stat_output=git_log_stat_output,
            run_commit_links=links,
            accepted_commit_hash=binding.accepted_commit_hash if binding else "",
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
