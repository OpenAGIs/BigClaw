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
- Legacy Python implementation assets originally present for the same governance surfaces:
  - `src/bigclaw/repo_governance.py`
  - `src/bigclaw/governance.py`
  - `src/bigclaw/repo_board.py`

## Landed in this issue

- Added Go parity coverage for the repo-board surface in `bigclaw-go/internal/repo/board_test.go`.
- Added a minimal Go collaboration-thread replacement in `bigclaw-go/internal/collaboration/thread.go` and `thread_test.go` to cover repo-board/native thread merge behavior.
- Deleted the migrated Python tests:
  - `tests/test_repo_governance.py`
  - `tests/test_governance.py`
  - `tests/test_repo_board.py`
  - `tests/test_repo_collaboration.py`
- Deleted the legacy Python governance implementation modules:
  - `src/bigclaw/repo_governance.py`
  - `src/bigclaw/governance.py`
  - `src/bigclaw/repo_board.py`
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

## Deletion gate outcome

- `tests/test_planning.py` no longer imports `ScopeFreezeAudit`; it now uses a local baseline-audit stub because planning only needs the structural contract (`version`, `release_ready`, `readiness_score`) rather than the old governance module.
- `src/bigclaw/planning.py` no longer imports the governance module and instead relies on a structural `BaselineAudit` protocol.
- The targeted governance Python tests and implementation modules have been removed from the repo.

## Regression commands

- `cd bigclaw-go && go test ./internal/repo ./internal/governance ./internal/collaboration`
- Optional focused reruns while deleting Python assets later:
  - `cd bigclaw-go && go test ./internal/repo -run 'TestRepo(Board|Permission|Audit)'`
  - `cd bigclaw-go && go test ./internal/governance -run 'TestScopeFreeze'`
