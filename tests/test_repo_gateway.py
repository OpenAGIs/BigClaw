from bigclaw.repo_commits import (
    CommitDiff,
    CommitLineage,
    RepoCommit,
    normalize_commit,
    normalize_diff,
    normalize_gateway_error,
    normalize_lineage,
    repo_audit_payload,
)


def test_repo_gateway_normalization_and_audit_payload():
    commit = normalize_commit({"commit_hash": "abc123", "title": "feat: add repo plane", "author": "bot"})
    assert isinstance(commit, RepoCommit)
    assert commit.commit_hash == "abc123"

    lineage = normalize_lineage(
        {
            "root_hash": "abc123",
            "lineage": [commit.to_dict()],
            "children": {"abc123": ["def456"]},
            "leaves": ["def456"],
        }
    )
    assert isinstance(lineage, CommitLineage)
    assert lineage.leaves == ["def456"]

    diff = normalize_diff(
        {
            "left_hash": "abc123",
            "right_hash": "def456",
            "files_changed": 3,
            "insertions": 20,
            "deletions": 4,
            "summary": "3 files changed",
        }
    )
    assert isinstance(diff, CommitDiff)
    assert diff.files_changed == 3

    payload = repo_audit_payload(
        actor="native cloud",
        action="repo.diff",
        outcome="success",
        commit_hash="def456",
        repo_space_id="space-1",
    )
    assert payload["actor"] == "native cloud"
    assert payload["commit_hash"] == "def456"


def test_repo_gateway_error_normalization_is_deterministic():
    timeout = normalize_gateway_error(RuntimeError("gateway timeout while fetching lineage"))
    assert timeout.code == "timeout"
    assert timeout.retryable is True

    missing = normalize_gateway_error(RuntimeError("commit not found"))
    assert missing.code == "not_found"
    assert missing.retryable is False
