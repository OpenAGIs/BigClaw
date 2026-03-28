# BIG-GO-923 pytest harness migration

## Scope

This issue migrates the current pytest bootstrap baseline toward a Go-native test harness for the `bigclaw-go` tree.

The Python-side harness surface in scope today is intentionally small:

- `tests/conftest.py`
- `tests/test_*.py` under the repository root

## Current Python and non-Go asset inventory

`tests/conftest.py` currently performs a single harness function:

- resolve the repository root from `tests/`
- prepend `<repo>/src` to `sys.path`
- make `from bigclaw...` imports work for pytest

Observed inventory at the time of migration:

- `56` Python test modules under `tests/`
- `47` modules directly importing `bigclaw...`
- `3` modules importing `pytest`: `test_audit_events.py`, `test_planning.py`, `test_roadmap.py`
- no shared pytest fixtures in `tests/` and no fixture definitions in `tests/conftest.py`

This means the legacy pytest harness is an import bootstrap, not a fixture/runtime orchestration layer.

## Go replacement landed in this issue

The new Go-native baseline lives in `bigclaw-go/internal/testharness`.

It provides:

- `RepoRoot(tb)` to locate the `bigclaw-go` module root without relying on package cwd
- `ProjectRoot(tb)` to reach the parent repository root that still contains legacy `src/` and `tests/`
- `JoinRepoRoot(tb, elems...)` and `JoinProjectRoot(tb, elems...)` for stable fixture/report path resolution
- `ResolveProjectPath(tb, candidate)` for paths that may still be prefixed with `bigclaw-go/`
- `PrependPathEnv(tb, dir)` for path-based CLI bootstrapping
- `Chdir(tb, dir)` for temporary cwd changes with automatic cleanup

First-batch adoption landed here:

- `internal/regression/*_test.go` now uses the shared repo-root baseline instead of ad hoc `../..` resolution and `runtime.Caller` plumbing
- `cmd/bigclawctl/migration_commands_test.go` now uses the shared cwd and `PATH` bootstrap helpers

First migrated Python test slice now covered explicitly in Go:

- `tests/test_dashboard_run_contract.py`
  - `test_dashboard_run_contract_default_bundle_is_release_ready`
  - `test_dashboard_run_contract_audit_detects_missing_field_definitions_and_samples`
  - `test_dashboard_run_contract_round_trip_preserves_samples_and_audit`
  - covered by `bigclaw-go/internal/product/dashboard_run_contract_test.go`
- `tests/test_saved_views.py`
  - `test_saved_view_catalog_round_trip_preserves_manifest_shape`
  - `test_saved_view_catalog_audit_surfaces_configuration_gaps`
  - `test_saved_view_catalog_audit_round_trip_preserves_findings`
  - `test_render_saved_view_report_summarizes_views_and_digest_coverage`
  - covered by `bigclaw-go/internal/product/saved_views_test.go`
- `tests/test_legacy_shim.py`
  - `test_dev_smoke_shim_runs_without_pythonpath`
  - `test_create_issues_shim_help_runs_without_pythonpath`
  - `test_github_sync_shim_help_runs_without_pythonpath`
  - `test_workspace_bootstrap_shim_help_runs_without_pythonpath`
  - `test_symphony_workspace_bootstrap_shim_help_runs_without_pythonpath`
  - `test_symphony_workspace_validate_shim_help_runs_without_pythonpath`
  - `test_refill_shim_help_runs_without_pythonpath`
  - covered by `bigclaw-go/cmd/bigclawctl/migration_commands_test.go` and `bigclaw-go/cmd/bigclawctl/main_test.go`
- `tests/test_governance.py`
  - `test_scope_freeze_board_round_trip_preserves_manifest_shape`
  - `test_scope_freeze_audit_flags_backlog_governance_and_closeout_gaps`
  - `test_scope_freeze_audit_round_trip_and_ready_state`
  - `test_render_scope_freeze_report_summarizes_board_and_run_closeout_requirements`
  - covered by `bigclaw-go/internal/governance/freeze_test.go`
- `tests/test_workspace_bootstrap.py`
  - `test_repo_cache_key_derives_from_repo_locator`
  - `test_cache_root_for_repo_uses_repo_specific_directory`
  - `test_bootstrap_workspace_creates_shared_worktree_from_local_seed`
  - `test_second_workspace_reuses_warm_cache_without_full_clone`
  - `test_bootstrap_workspace_reuses_existing_issue_worktree`
  - `test_cleanup_workspace_preserves_shared_cache_for_future_reuse`
  - `test_bootstrap_recovers_from_stale_seed_directory_without_remote_reclone`
  - `test_cleanup_workspace_prunes_worktree_and_bootstrap_branch`
  - covered by `bigclaw-go/internal/bootstrap/bootstrap_test.go`

Still legacy-only within `tests/test_workspace_bootstrap.py`:

- Python validation report aggregation via `build_validation_report`
- JSON/reporting parity for the validation bundle still needs a direct Go equivalent before the full Python file can leave the active validation path

Still legacy-only within `tests/test_legacy_shim.py`:

- Python wrapper argument translation helpers in `src/bigclaw/legacy_shim.py`
- These are compatibility shims rather than target Go mainline behavior and can be retired only when the Python wrappers themselves are removed from supported validation paths

## Migration plan

1. Treat `internal/testharness` as the only shared bootstrap layer for Go tests that need repository-relative assets or CLI environment setup.
2. Continue porting Python contract/report tests into `bigclaw-go/internal/...` packages on top of that harness instead of extending pytest infrastructure.
3. Keep Python tests runnable only as long as there are remaining `src/bigclaw` behaviors without Go coverage.

Recommended next migration slices:

- `tests/test_dashboard_run_contract.py` into `bigclaw-go/internal/product`
- `tests/test_saved_views.py` into `bigclaw-go/internal/product`
- `tests/test_repo_governance.py` into `bigclaw-go/internal/repo`
- `tests/test_legacy_shim.py` into `bigclaw-go/internal/legacyshim` and `cmd/bigclawctl`
- `tests/test_workspace_bootstrap.py` into `bigclaw-go/internal/bootstrap`

## Deletion gate for legacy Python harness

`tests/conftest.py` is safe to delete only when all of the following are true:

- no remaining validation lane depends on `python3 -m pytest`
- no remaining test module imports `bigclaw...` from `src/`
- Go replacements cover the active regression surface for the remaining Python tests
- a repo-wide validation run succeeds without Python path injection

Until then, `tests/conftest.py` remains a compatibility shim and should not grow new behavior.

## Regression commands

Primary validation for this issue:

```bash
cd bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl
cd bigclaw-go && go test ./...
```

Deletion-readiness validation for the legacy Python harness, once migration is further along:

```bash
python3 -m pytest tests
cd bigclaw-go && go test ./...
```
