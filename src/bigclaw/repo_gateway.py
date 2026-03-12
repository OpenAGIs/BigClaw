from __future__ import annotations

from dataclasses import dataclass
from typing import Any, Dict, List, Protocol

from .repo_commits import CommitDiff, CommitLineage, RepoCommit


class RepoGatewayClient(Protocol):
    def push_bundle(self, repo_space_id: str, bundle_ref: str) -> Dict[str, Any]: ...

    def fetch_bundle(self, repo_space_id: str, bundle_ref: str) -> Dict[str, Any]: ...

    def list_commits(self, repo_space_id: str) -> List[Dict[str, Any]]: ...

    def get_commit(self, repo_space_id: str, commit_hash: str) -> Dict[str, Any]: ...

    def get_children(self, repo_space_id: str, commit_hash: str) -> List[str]: ...

    def get_lineage(self, repo_space_id: str, commit_hash: str) -> Dict[str, Any]: ...

    def get_leaves(self, repo_space_id: str, commit_hash: str) -> List[str]: ...

    def diff(self, repo_space_id: str, left_hash: str, right_hash: str) -> Dict[str, Any]: ...


@dataclass(frozen=True)
class RepoGatewayError:
    code: str
    message: str
    retryable: bool = False

    def to_dict(self) -> Dict[str, Any]:
        return {"code": self.code, "message": self.message, "retryable": self.retryable}


def normalize_gateway_error(error: Exception) -> RepoGatewayError:
    message = str(error).lower()
    if "timeout" in message:
        return RepoGatewayError(code="timeout", message=str(error), retryable=True)
    if "not found" in message:
        return RepoGatewayError(code="not_found", message=str(error), retryable=False)
    return RepoGatewayError(code="gateway_error", message=str(error), retryable=False)


def normalize_commit(payload: Dict[str, Any]) -> RepoCommit:
    return RepoCommit.from_dict(payload)


def normalize_lineage(payload: Dict[str, Any]) -> CommitLineage:
    return CommitLineage.from_dict(payload)


def normalize_diff(payload: Dict[str, Any]) -> CommitDiff:
    return CommitDiff.from_dict(payload)


def repo_audit_payload(*, actor: str, action: str, outcome: str, commit_hash: str, repo_space_id: str) -> Dict[str, Any]:
    return {
        "actor": actor,
        "action": action,
        "outcome": outcome,
        "commit_hash": commit_hash,
        "repo_space_id": repo_space_id,
    }
