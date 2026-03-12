from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any, Dict, List


@dataclass
class RepoCommit:
    commit_hash: str
    title: str
    author: str = ""
    parent_hashes: List[str] = field(default_factory=list)
    metadata: Dict[str, Any] = field(default_factory=dict)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "commit_hash": self.commit_hash,
            "title": self.title,
            "author": self.author,
            "parent_hashes": list(self.parent_hashes),
            "metadata": dict(self.metadata),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "RepoCommit":
        return cls(
            commit_hash=str(data["commit_hash"]),
            title=str(data.get("title", "")),
            author=str(data.get("author", "")),
            parent_hashes=[str(item) for item in data.get("parent_hashes", [])],
            metadata=dict(data.get("metadata", {})),
        )


@dataclass
class CommitLineage:
    root_hash: str
    lineage: List[RepoCommit] = field(default_factory=list)
    children: Dict[str, List[str]] = field(default_factory=dict)
    leaves: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "root_hash": self.root_hash,
            "lineage": [item.to_dict() for item in self.lineage],
            "children": {key: list(value) for key, value in self.children.items()},
            "leaves": list(self.leaves),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CommitLineage":
        return cls(
            root_hash=str(data.get("root_hash", "")),
            lineage=[RepoCommit.from_dict(item) for item in data.get("lineage", [])],
            children={str(key): [str(v) for v in value] for key, value in dict(data.get("children", {})).items()},
            leaves=[str(item) for item in data.get("leaves", [])],
        )


@dataclass
class CommitDiff:
    left_hash: str
    right_hash: str
    files_changed: int
    insertions: int
    deletions: int
    summary: str = ""

    def to_dict(self) -> Dict[str, Any]:
        return {
            "left_hash": self.left_hash,
            "right_hash": self.right_hash,
            "files_changed": self.files_changed,
            "insertions": self.insertions,
            "deletions": self.deletions,
            "summary": self.summary,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> "CommitDiff":
        return cls(
            left_hash=str(data.get("left_hash", "")),
            right_hash=str(data.get("right_hash", "")),
            files_changed=int(data.get("files_changed", 0)),
            insertions=int(data.get("insertions", 0)),
            deletions=int(data.get("deletions", 0)),
            summary=str(data.get("summary", "")),
        )
