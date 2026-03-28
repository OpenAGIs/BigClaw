# BIG-GO-924 Governance Test Migration

## Original targeted Python and non-Go assets

- `tests/test_repo_governance.py`
  - Python coverage for repo permission checks and deterministic audit field requirements.
  - Go replacement already lives in `bigclaw-go/internal/repo/governance.go` and `bigclaw-go/internal/repo/governance_test.go`.
- `tests/test_governance.py`
  - Python coverage for scope-freeze board serialization, audit gaps, readiness scoring, and rendered governance report output.
  - Go replacement already lives in `bigclaw-go/internal/governance/freeze.go` and `bigclaw-go/internal/governance/freeze_test.go`.
- `tests/test_repo_board.py`
  - Python coverage for repo discussion post creation, reply threading, target filtering, and collaboration comment anchoring.
  - Go replacement for the board model lives in `bigclaw-go/internal/repo/board.go`.
  - This issue adds Go parity coverage in `bigclaw-go/internal/repo/board_test.go`, including the collaboration-comment conversion that was previously only exercised on the Python side.
- Legacy Python implementation assets still present for the same governance surfaces:
  - `src/bigclaw/repo_governance.py`
  - `src/bigclaw/governance.py`
  - `src/bigclaw/repo_board.py`

## Landed in this issue

- Added Go parity coverage for the repo-board surface in `bigclaw-go/internal/repo/board_test.go`.
- Deleted the migrated Python tests:
  - `tests/test_repo_governance.py`
  - `tests/test_governance.py`
  - `tests/test_repo_board.py`
- Updated validation guidance in `docs/BigClaw-AgentHub-Integration-Alignment.md` to use the Go governance test command for this migrated slice.

## Go migration mapping

- `RepoPermissionContract.check` -> `repo.PermissionContract.Check`
- `missing_repo_audit_fields` -> `repo.MissingAuditFields`
- `ScopeFreezeBoard` / `ScopeFreezeAudit` / `ScopeFreezeGovernance.audit` / `render_scope_freeze_report`
  -> `governance.ScopeFreezeBoard`, `governance.ScopeFreezeAudit`, `governance.ScopeFreezeGovernance.Audit`, `governance.RenderScopeFreezeReport`
- `RepoDiscussionBoard.create_post` / `reply` / `list_posts` / `RepoPost.to_collaboration_comment`
  -> `repo.RepoDiscussionBoard.CreatePost`, `Reply`, `ListPosts`, `RepoPost.ToCollaborationComment`

## Python asset deletion conditions

- Do not delete the legacy Python files until all three conditions hold:
  - no remaining Python runtime or test entrypoint imports `src/bigclaw/governance.py`, `src/bigclaw/repo_governance.py`, or `src/bigclaw/repo_board.py`
  - the Go tests below remain green and are wired into the repo's normal regression lane
  - any consumer-facing documentation or compatibility shim references are updated to point at the Go surfaces instead of the Python modules

## Current blockers to deleting the Python assets

- `tests/test_planning.py` still imports `ScopeFreezeAudit` from `src/bigclaw/governance.py`
- `tests/test_repo_collaboration.py` still imports `RepoDiscussionBoard` from `src/bigclaw/repo_board.py`

These remaining references make the deletion gate concrete: the targeted governance tests are now Go-owned, but the legacy Python implementation modules should stay until the remaining Python-side consumers migrate or are deleted in their own scoped tickets.

## Regression commands

- `cd bigclaw-go && go test ./internal/repo ./internal/governance`
- Optional focused reruns while deleting Python assets later:
  - `cd bigclaw-go && go test ./internal/repo -run 'TestRepo(Board|Permission|Audit)'`
  - `cd bigclaw-go && go test ./internal/governance -run 'TestScopeFreeze'`
