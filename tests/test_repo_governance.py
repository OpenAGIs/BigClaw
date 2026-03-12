from bigclaw.repo_governance import RepoPermissionContract, missing_repo_audit_fields


def test_repo_permission_matrix_resolves_roles():
    contract = RepoPermissionContract()

    assert contract.check(action_permission="repo.push", actor_roles=["eng-lead"]) is True
    assert contract.check(action_permission="repo.accept", actor_roles=["reviewer"]) is True
    assert contract.check(action_permission="repo.push", actor_roles=["execution-agent"]) is False


def test_repo_audit_field_contract_is_deterministic():
    missing = missing_repo_audit_fields(
        "repo.accept",
        {
            "task_id": "OPE-172",
            "run_id": "run-172",
            "repo_space_id": "space-1",
            "actor": "reviewer",
        },
    )
    assert missing == ["accepted_commit_hash", "reviewer"]
