Issue: BIG-GO-1024

Plan
- Fold `connectors.py`, `deprecation.py`, `issue_archive.py`, `pilot.py`, `roadmap.py`, and `parallel_refill.py` into the existing `bigclaw` package initialization layer via dynamic submodules so the package keeps working while the physical `.py` count drops.
- Delete those standalone Python source files from `src/bigclaw`.
- Continue the same pattern for the next low-coupling tranche: `cost_control.py`, `legacy_shim.py`, `mapping.py`, `workspace_bootstrap_cli.py`, `github_sync.py`, `workspace_bootstrap_validation.py`, `validation_policy.py`, `event_bus.py`, `memory.py`, `dsl.py`, `risk.py`, `run_detail.py`, `dashboard_run_contract.py`, `governance.py`, `repo_commits.py`, `repo_gateway.py`, `repo_triage.py`, `repo_board.py`, and `repo_registry.py`.
- Add targeted regression coverage for legacy submodule imports plus the preserved helper/report behavior.
- Run scoped validation, capture exact commands and results, then commit and push the issue branch.

Acceptance
- `src/bigclaw` physical `.py` file count is lower after the tranche.
- `bigclaw.connectors`, `bigclaw.deprecation`, `bigclaw.issue_archive`, `bigclaw.pilot`, `bigclaw.roadmap`, and `bigclaw.parallel_refill` remain importable and preserve their core behavior.
- `bigclaw.cost_control`, `bigclaw.legacy_shim`, `bigclaw.mapping`, and `bigclaw.workspace_bootstrap_cli` remain importable and preserve their core behavior.
- `bigclaw.github_sync`, `bigclaw.workspace_bootstrap_validation`, and `bigclaw.validation_policy` remain importable and preserve their tested behavior.
- `bigclaw.event_bus`, `bigclaw.memory`, and `bigclaw.dsl` remain importable and preserve their existing tested behavior.
- `bigclaw.risk`, `bigclaw.run_detail`, and `bigclaw.dashboard_run_contract` remain importable and preserve their existing tested behavior.
- `bigclaw.governance` remains importable and preserves its existing tested behavior.
- `bigclaw.repo_commits`, `bigclaw.repo_gateway`, `bigclaw.repo_triage`, and `bigclaw.repo_board` remain importable and preserve their existing tested behavior.
- `bigclaw.repo_registry` remains importable and preserves its existing tested behavior.
- No `pyproject.toml`, `setup.py`, or `setup.cfg` changes are required for this slice unless such files exist and are directly impacted.
- Changes remain scoped to the selected residual Python modules and direct validation assets.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_legacy_surface_modules.py -q`
- `PYTHONPATH=src python3 - <<'PY'`
- `import bigclaw.connectors, bigclaw.cost_control, bigclaw.deprecation, bigclaw.issue_archive, bigclaw.legacy_shim, bigclaw.mapping, bigclaw.pilot, bigclaw.roadmap, bigclaw.parallel_refill, bigclaw.workspace_bootstrap_cli`
- `print("legacy module smoke checks passed")`
- `PY`
- `git status --short && git diff --stat`
