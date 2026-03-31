# BIG-GO-1023

## Plan
- Inspect remaining `src/bigclaw` Python modules and existing Go migration patterns in the repo.
- Pick a small, coherent tranche of residual modules that can be migrated together without widening scope.
- Replace the selected Python modules with Go implementations and remove the corresponding `.py` files.
- Run targeted validation for the migrated package area and record exact commands and results.
- Commit the scoped changes and push the branch to the remote.

## Acceptance
- Changes apply to residual Python assets under `src/bigclaw`.
- The repository ends with fewer `.py` files under `src/bigclaw`.
- Report the impact on `.py` and `.go` file counts and on any `pyproject` or `setup` files.
- Repository results, not tracker updates, are the closure mechanism.

## Validation
- Capture pre/post counts for `src/bigclaw` `.py` and `.go` files.
- Run targeted Go tests covering the migrated package or module entrypoints.
- Verify removed Python files are replaced by buildable Go sources in the same functional area.

## Results
- Pre-change counts: `src/bigclaw` had `45` `.py` files and `0` `.go` files.
- Mid-change counts after tranche A: `src/bigclaw` had `36` `.py` files and `0` `.go` files.
- Mid-change counts after tranche B: `src/bigclaw` had `30` `.py` files and `0` `.go` files.
- Post-change counts: `src/bigclaw` has `25` `.py` files and `0` `.go` files.
- `pyproject.toml`, `setup.py`, and `setup.cfg` at repo root: not present before or after this issue.

## Validation Runs
- `python3 -m compileall src/bigclaw`
  - Result: succeeded.
- `PYTHONPATH=src python3 -m pytest tests/test_repo_gateway.py tests/test_repo_registry.py tests/test_repo_links.py tests/test_repo_triage.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_governance.py tests/test_governance.py tests/test_risk.py tests/test_observability.py`
  - Result: `25 passed in 0.14s`.
- `cd bigclaw-go && go test ./internal/repo ./internal/governance ./internal/risk ./internal/issuearchive`
  - Result: all four packages passed.
- `python3 -m compileall src/bigclaw tests`
  - Result: succeeded.
- `PYTHONPATH=src python3 -m pytest tests/test_saved_views.py tests/test_dashboard_run_contract.py tests/test_event_bus.py tests/test_github_sync.py tests/test_mapping_connectors.py tests/test_pilot.py`
  - Result: `20 passed in 1.28s`.
- `cd bigclaw-go && go test ./internal/intake ./internal/product ./internal/pilot ./internal/githubsync ./internal/events`
  - Result: all five packages passed.
- `python3 -m compileall src/bigclaw tests`
  - Result: succeeded.
- `PYTHONPATH=src python3 -m pytest tests/test_cost_control.py tests/test_roadmap.py tests/test_memory.py tests/test_validation_policy.py tests/test_parallel_refill.py tests/test_legacy_shim.py`
  - Result: `11 passed in 0.08s`.
- `cd bigclaw-go && go test ./internal/costcontrol ./internal/refill ./internal/legacyshim ./internal/regression`
  - Result: all four packages passed.
