Issue: BIG-GO-1034

Scope
- Remove migrated Python governance/reporting/observability/memory/cost-control modules from `src/bigclaw`.
- Remove repo governance Python surface once the Go `internal/repo` replacement is confirmed equivalent and no Python consumers remain.
- Remove the dependent legacy Python runtime/reporting cluster required to complete that deletion cleanly:
  `__main__.py`, `event_bus.py`, `evaluation.py`, `operations.py`, `runtime.py`.
- Remove direct Python tests that only validate the deleted cluster.
- Collapse `src/bigclaw/__init__.py` to a minimal safe package surface so surviving submodule imports do not depend on deleted modules.
- Use existing Go implementations in `bigclaw-go/internal/governance`, `bigclaw-go/internal/reporting`, `bigclaw-go/internal/observability`, and `bigclaw-go/internal/costcontrol` as the replacement surface.
- Add or adjust Go validation only if needed to cover deleted Python behavior gaps inside this slice.

Acceptance
- Python file count in the targeted migration slice decreases.
- Python file count in the adjacent legacy runtime/reporting dependency cluster also decreases when required to complete the removal cleanly.
- Go implementation remains present for governance/reporting/observability/cost-control, with Go tests covering the retained replacement surface.
- No `pyproject.toml` or `setup.py` remains at repo root.
- Commit explains which Python files were deleted and which Go files validate the replacement surface.

Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034/bigclaw-go && go test ./internal/repo ./internal/governance ./internal/reporting ./internal/observability ./internal/costcontrol ./internal/memory`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034 && PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_design_system.py tests/test_saved_views.py tests/test_validation_policy.py tests/test_models.py tests/test_execution_contract.py tests/test_repo_board.py tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_triage.py tests/test_repo_collaboration.py tests/test_workspace_bootstrap.py tests/test_dashboard_run_contract.py tests/test_console_ia.py tests/test_ui_review.py tests/test_github_sync.py -q`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034 && rg -n "from \\.repo_governance|import bigclaw\\.repo_governance|from bigclaw\\.repo_governance|RepoPermissionContract|missing_repo_audit_fields" src tests`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034 && rg -n "from \\.observability|from \\.reports|from \\.event_bus|from \\.evaluation|from \\.operations|from \\.runtime|import bigclaw\\.(observability|reports|event_bus|evaluation|operations|runtime)|from bigclaw\\.(observability|reports|event_bus|evaluation|operations|runtime)" src tests`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1034 && git diff --stat`

Notes
- Keep changes scoped to the migrated governance/reporting/observability/memory/cost-control slice and its direct legacy Python dependency chain.
