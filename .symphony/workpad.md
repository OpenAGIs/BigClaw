# BIG-GO-1013 Workpad

## Scope

Target a narrow residual-module consolidation batch under `src/bigclaw/**` to
reduce Python module count without expanding into unrelated runtime surfaces.

Batch file list:

- `src/bigclaw/__init__.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/runtime.py`
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/legacy_shim.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/observability.py`
- `src/bigclaw/pilot.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/deprecation.py`
- `src/bigclaw/cost_control.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/workspace_bootstrap_validation.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/event_bus.py`
- `src/bigclaw/dsl.py`
- `docs/go-mainline-cutover-issue-pack.md`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/validation_policy.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_registry.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_validation_policy.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_triage.py`
- `tests/test_workspace_bootstrap.py`
- `bigclaw-go/internal/legacyshim/compilecheck.go`
- `bigclaw-go/internal/legacyshim/compilecheck_test.go`
- `bigclaw-go/cmd/bigclawctl/main_test.go`
- `reports/BIG-GO-902-validation.md`
- `reports/BIG-GO-902-closeout.md`
- `reports/BIG-GO-902-pr.md`
- `reports/BIG-GO-902-status.json`

Repository inventory at start of lane:

- `src/bigclaw/**/*.py`: `45`
- `src/**/*.go`: `0`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

Selected tranche rationale:

- `repo_commits.py`, `repo_links.py`, and `repo_registry.py` are small,
  repo-specific modules with low fan-out.
- Their definitions fit naturally into existing repo-domain modules:
  `repo_gateway.py` and `repo_plane.py`.
- Compatibility for legacy import paths can be preserved from `bigclaw.__init__`
  using synthetic submodules, matching the package's existing migration pattern.
- `mapping.py` is a thin source-issue normalization layer whose natural owner is
  `connectors.py`.
- `validation_policy.py` is a tiny report-artifact policy layer whose natural
  owner is `reports.py`.
- `repo_governance.py` is an execution-permission specialization that fits
  naturally inside `execution_contract.py`.
- `repo_triage.py` is a repo-run decision helper that fits naturally inside
  `repo_plane.py`.
- `pilot.py` is a reporting-oriented artifact generator that fits naturally
  inside `reports.py`.
- `roadmap.py` is a planning-oriented structure that fits naturally inside
  `planning.py`.
- `deprecation.py` is a legacy-runtime helper that fits directly inside
  `runtime.py`.
- `cost_control.py` is a runtime budget helper with no external fan-out and can
  be folded into `runtime.py`.
- `legacy_shim.py` still fronts multiple Python operator wrappers, but its
  logic is self-contained and can be folded into `runtime.py` if the frozen Go
  compile-check list and repository docs are updated in the same batch.
- `workspace_bootstrap_cli.py` and `workspace_bootstrap_validation.py` are thin
  wrappers around `workspace_bootstrap.py` and can be folded into that owning
  module without widening scope.
- `parallel_refill.py` appears to be dead Python residue with no remaining code
  imports, so it can be deleted once the remaining repo docs stop pointing at
  the removed file path.
- `risk.py` is already runtime-owned logic in practice and can be folded into
  `runtime.py`.
- `audit_events.py` is now only a shared observability schema surface and can
  be folded into `observability.py`.
- `repo_board.py` is a collaboration surface and can be folded into
  `collaboration.py`.
- `event_bus.py` already centers on `ObservabilityLedger` and `TaskRun`, so it
  can be folded into `observability.py`.
- `dsl.py` is legacy workflow-definition scaffolding and can be folded into the
  frozen runtime/workflow surface.

## Plan

1. Move commit DTOs from `repo_commits.py` into `repo_gateway.py`.
2. Move run-commit binding helpers and registry models into `repo_plane.py`.
3. Update in-package imports to use the new owning modules.
4. Install compatibility submodules for `bigclaw.repo_commits`,
   `bigclaw.repo_links`, and `bigclaw.repo_registry` from `__init__.py`.
5. Delete the three residual Python modules after all references are updated.
6. Run targeted tests for repo gateway, repo links, repo registry, and any
   package paths affected by the import relocation.
7. Record exact validation commands and repository file-count impact.
8. Commit and push the scoped batch for `BIG-GO-1013`.
9. Fold `mapping.py` into `connectors.py` and preserve `bigclaw.mapping`
   compatibility via `__init__.py`.
10. Fold `validation_policy.py` into `reports.py` and preserve
   `bigclaw.validation_policy` compatibility via `__init__.py`.
11. Run targeted validation for the second consolidation batch and push a
   follow-up commit.
12. Fold `repo_governance.py` into `execution_contract.py` with compatibility
    preserved via `__init__.py`.
13. Fold `repo_triage.py` into `repo_plane.py` with compatibility preserved via
    `__init__.py`.
14. Run targeted repo-domain tests for the third consolidation batch and push a
    follow-up commit.
15. Fold `pilot.py` into `reports.py` and preserve `bigclaw.pilot`
    compatibility via `__init__.py`.
16. Fold `roadmap.py` into `planning.py` and preserve `bigclaw.roadmap`
    compatibility via `__init__.py`.
17. Run targeted smoke validation for the fourth consolidation batch and push a
    follow-up commit.
18. Fold `deprecation.py` into `runtime.py` and preserve `bigclaw.deprecation`
    compatibility via `__init__.py`.
19. Fold `cost_control.py` into `runtime.py` and preserve
    `bigclaw.cost_control` compatibility via `__init__.py`.
20. Run targeted runtime smoke validation for the fifth consolidation batch and
    push a follow-up commit.
21. Fold `legacy_shim.py` into `runtime.py` and preserve `bigclaw.legacy_shim`
    compatibility via `__init__.py`.
22. Update Go-side frozen compile-check coverage to remove the deleted Python
    file from the shim file list.
23. Refresh the remaining repository reports that hard-code
    `src/bigclaw/legacy_shim.py`.
24. Run targeted Python and Go validation for the sixth consolidation batch and
    push a follow-up commit.
25. Fold `workspace_bootstrap_cli.py` into `workspace_bootstrap.py` and
    preserve `bigclaw.workspace_bootstrap_cli` compatibility via `__init__.py`.
26. Fold `workspace_bootstrap_validation.py` into `workspace_bootstrap.py` and
    preserve `bigclaw.workspace_bootstrap_validation` compatibility via
    `__init__.py`.
27. Run targeted workspace bootstrap validation for the seventh consolidation
    batch and push a follow-up commit.
28. Delete `parallel_refill.py` after confirming it has no remaining code-path
    imports.
29. Refresh the remaining in-repo docs that still point at
    `src/bigclaw/parallel_refill.py`.
30. Run targeted refill validation for the eighth consolidation batch and push a
    follow-up commit.
31. Fold `risk.py` into `runtime.py` and preserve `bigclaw.risk` compatibility
    via `__init__.py`.
32. Fold `audit_events.py` into `observability.py` and preserve
    `bigclaw.audit_events` compatibility via `__init__.py`.
33. Run targeted risk/audit validation for the ninth consolidation batch and
    push a follow-up commit.
34. Fold `repo_board.py` into `collaboration.py` and preserve
    `bigclaw.repo_board` compatibility via `__init__.py`.
35. Fold `event_bus.py` into `observability.py` and preserve
    `bigclaw.event_bus` compatibility via `__init__.py`.
36. Run targeted repo-board/event-bus validation for the tenth consolidation
    batch and push a follow-up commit.
37. Fold `dsl.py` into `runtime.py` and preserve `bigclaw.dsl` compatibility
    via `__init__.py`.
38. Run targeted DSL validation for the eleventh consolidation batch and push a
    follow-up commit.

## Acceptance

- Directly reduce `src/bigclaw/**` residual Python module count.
- Keep behavior stable for existing import paths used by tests.
- Keep changes scoped to the selected repo-domain tranche.
- Report impact on `py files` / `go files` / `pyproject.toml` / `setup.py`.
- Validate with exact commands and results, not tracker state.

## Validation

- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
- `find src -type f -name '*.go' | sort | wc -l`
- `test -f pyproject.toml; echo $?`
- `test -f setup.py; echo $?`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_gateway.py tests/test_repo_links.py tests/test_repo_registry.py`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py`
- `PYTHONPATH=src python3 -m pytest tests/test_validation_policy.py`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py tests/test_repo_triage.py`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.pilot import PilotImplementationResult, PilotKPI, render_pilot_implementation_report`
  `from bigclaw.roadmap import build_execution_pack_roadmap`
  `print("ok")`
  `PY`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.deprecation import warn_legacy_runtime_surface`
  `from bigclaw.cost_control import CostController`
  `print("ok")`
  `PY`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.legacy_shim import run_bigclawctl_shim`
  `print("ok")`
  `PY`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.workspace_bootstrap_cli import build_parser`
  `from bigclaw.workspace_bootstrap_validation import build_validation_report`
  `print("ok")`
  `PY`
- `python3 -m py_compile scripts/ops/bigclaw_refill_queue.py`
- `cd bigclaw-go && go test ./internal/refill -run TestParallelIssueQueueRepoFixtureSelectionStaysAligned`
- `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_audit_events.py tests/test_runtime_matrix.py tests/test_observability.py`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_event_bus.py tests/test_observability.py`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_runtime_matrix.py`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.risk import RiskScorer`
  `from bigclaw.audit_events import SCHEDULER_DECISION_EVENT`
  `print("ok")`
  `PY`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.repo_board import RepoDiscussionBoard`
  `from bigclaw.event_bus import EventBus`
  `print("ok")`
  `PY`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.dsl import WorkflowDefinition`
  `print("ok")`
  `PY`
- `python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/runtime.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/create_issues.py scripts/dev_smoke.py`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.mapping import map_source_issue_to_task`
  `from bigclaw.validation_policy import enforce_validation_report_policy`
  `print("ok")`
  `PY`
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.repo_governance import RepoPermissionContract`
  `from bigclaw.repo_triage import recommend_triage_action`
  `print("ok")`
  `PY`
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `src/bigclaw/repo_gateway.py`
  - Replaced.
  - Reason: absorbed `RepoCommit`, `CommitLineage`, and `CommitDiff` so commit
    DTOs now live with the gateway normalization helpers that use them.
- `src/bigclaw/repo_plane.py`
  - Replaced.
  - Reason: absorbed run-commit binding helpers and `RepoRegistry` so repo
    topology, repo-agent identity, and run-commit link state now live in one
    repo-domain module.
- `src/bigclaw/observability.py`
  - Replaced.
  - Reason: switched internal import ownership from deleted `repo_links.py` to
    `repo_plane.py`.
- `src/bigclaw/__init__.py`
  - Replaced.
  - Reason: installs compatibility submodules for `bigclaw.repo_commits`,
    `bigclaw.repo_links`, and `bigclaw.repo_registry` so old import paths still
    resolve after consolidation.
- `src/bigclaw/repo_commits.py`
  - Deleted.
  - Reason: its contents moved into `repo_gateway.py`.
- `src/bigclaw/repo_links.py`
  - Deleted.
  - Reason: its contents moved into `repo_plane.py`.
- `src/bigclaw/repo_registry.py`
  - Deleted.
  - Reason: its contents moved into `repo_plane.py`.
- `src/bigclaw/connectors.py`
  - Replaced.
  - Reason: absorbed `map_priority`, `map_state`, and
    `map_source_issue_to_task` so source issue fetch and normalization logic now
    live together.
- `src/bigclaw/reports.py`
  - Replaced.
  - Reason: absorbed `ValidationReportDecision`,
    `REQUIRED_REPORT_ARTIFACTS`, and `enforce_validation_report_policy` so
    report-artifact policy lives with other report utilities.
- `src/bigclaw/mapping.py`
  - Deleted.
  - Reason: its contents moved into `connectors.py`.
- `src/bigclaw/validation_policy.py`
  - Deleted.
  - Reason: its contents moved into `reports.py`.
- `src/bigclaw/execution_contract.py`
  - Replaced.
  - Reason: absorbed repo-specific permission policy and audit-field helpers so
    execution permissions and repo permission specializations live together.
- `src/bigclaw/repo_plane.py`
  - Replaced again.
  - Reason: absorbed repo triage evidence and recommendation helpers so repo
    topology, run linkage, registry state, and triage decisions now share one
    repo-plane module.
- `src/bigclaw/repo_governance.py`
  - Deleted.
  - Reason: its contents moved into `execution_contract.py`.
- `src/bigclaw/repo_triage.py`
  - Deleted.
  - Reason: its contents moved into `repo_plane.py`.
- `src/bigclaw/reports.py`
  - Replaced again.
  - Reason: absorbed pilot implementation KPI/report helpers so pilot-oriented
    report generation now lives with the rest of the reporting surface.
- `src/bigclaw/planning.py`
  - Replaced.
  - Reason: absorbed execution-pack roadmap structures so roadmap planning data
    lives with the existing planning surface.
- `src/bigclaw/pilot.py`
  - Deleted.
  - Reason: its contents moved into `reports.py`.
- `src/bigclaw/roadmap.py`
  - Deleted.
  - Reason: its contents moved into `planning.py`.
- `src/bigclaw/runtime.py`
  - Replaced again.
  - Reason: absorbed legacy deprecation helpers and runtime budget controller
    helpers so migration-only runtime behavior now lives in one module.
- `src/bigclaw/deprecation.py`
  - Deleted.
  - Reason: its contents moved into `runtime.py`.
- `src/bigclaw/cost_control.py`
  - Deleted.
  - Reason: its contents moved into `runtime.py`.
- `src/bigclaw/runtime.py`
  - Replaced again.
  - Reason: absorbed legacy operator shim helpers so the remaining Python
    compatibility helpers now share the frozen migration-only runtime surface.
- `src/bigclaw/legacy_shim.py`
  - Deleted.
  - Reason: its contents moved into `runtime.py`, with `bigclaw.legacy_shim`
    preserved via a compatibility submodule in `__init__.py`.
- `bigclaw-go/internal/legacyshim/compilecheck.go`
  - Replaced.
  - Reason: removed deleted `src/bigclaw/legacy_shim.py` from the frozen
    compile-check file list.
- `bigclaw-go/internal/legacyshim/compilecheck_test.go`
  - Replaced.
  - Reason: aligned the frozen compile-check expectations with the updated shim
    file inventory.
- `bigclaw-go/cmd/bigclawctl/main_test.go`
  - Replaced.
  - Reason: aligned JSON compile-check fixture setup with the updated shim file
    inventory.
- `reports/BIG-GO-902-validation.md`
  - Replaced.
  - Reason: removed the deleted file path from the historical py_compile command
    and described the shim behavior as living behind the runtime compatibility
    surface.
- `reports/BIG-GO-902-closeout.md`
  - Replaced.
  - Reason: removed the deleted file path from the historical validation
    command list.
- `reports/BIG-GO-902-pr.md`
  - Replaced.
  - Reason: removed the deleted file path and replaced the stale module-path
    statement with the surviving compatibility surface wording.
- `reports/BIG-GO-902-status.json`
  - Replaced.
  - Reason: removed the deleted file path from the frozen validation command
    list.
- `src/bigclaw/workspace_bootstrap.py`
  - Replaced again.
  - Reason: absorbed the workspace bootstrap CLI and validation helper surface
    so all shared bootstrap behavior now lives in the owning bootstrap module.
- `src/bigclaw/workspace_bootstrap_cli.py`
  - Deleted.
  - Reason: its contents moved into `workspace_bootstrap.py`.
- `src/bigclaw/workspace_bootstrap_validation.py`
  - Deleted.
  - Reason: its contents moved into `workspace_bootstrap.py`.
- `src/bigclaw/parallel_refill.py`
  - Deleted.
  - Reason: no remaining code-path imports referenced it; the surviving refill
    implementation mainline is already the Go `internal/refill` surface.
- `docs/go-mainline-cutover-issue-pack.md`
  - Replaced.
  - Reason: removed stale references to the deleted Python refill module from
    the cutover issue pack.
- `src/bigclaw/runtime.py`
  - Replaced again.
  - Reason: absorbed risk scoring models and logic so runtime scheduling and
    risk scoring now share one owning module.
- `src/bigclaw/observability.py`
  - Replaced again.
  - Reason: absorbed canonical audit event specifications so operational audit
    schema and audit consumers now share one module.
- `src/bigclaw/risk.py`
  - Deleted.
  - Reason: its contents moved into `runtime.py`.
- `src/bigclaw/audit_events.py`
  - Deleted.
  - Reason: its contents moved into `observability.py`.
- `src/bigclaw/reports.py`
  - Replaced again.
  - Reason: switched audit-event imports to the new owning observability
    module.
- `src/bigclaw/collaboration.py`
  - Replaced again.
  - Reason: absorbed repo discussion board models and behavior so collaboration
    records and repo-native discussion state now live together.
- `src/bigclaw/observability.py`
  - Replaced again.
  - Reason: absorbed event bus definitions so event ingestion and ledger state
    now live in the same observability module.
- `src/bigclaw/repo_board.py`
  - Deleted.
  - Reason: its contents moved into `collaboration.py`.
- `src/bigclaw/event_bus.py`
  - Deleted.
  - Reason: its contents moved into `observability.py`.
- `src/bigclaw/runtime.py`
  - Replaced again.
  - Reason: absorbed workflow DSL dataclasses and JSON loading helpers so the
    frozen runtime/workflow surface owns the remaining workflow-definition
    compatibility layer.
- `src/bigclaw/__init__.py`
  - Replaced again.
  - Reason: installs a `bigclaw.dsl` compatibility submodule that re-exports
    the moved workflow DSL types from `runtime.py`.
- `src/bigclaw/dsl.py`
  - Deleted.
  - Reason: its contents moved into `runtime.py`.

### Inventory Impact

- `src/bigclaw/**/*.py` before: `45`
- `src/bigclaw/**/*.py` after batch 1: `42`
- `src/bigclaw/**/*.py` after batch 2: `40`
- `src/bigclaw/**/*.py` after batch 3: `38`
- `src/bigclaw/**/*.py` after batch 4: `36`
- `src/bigclaw/**/*.py` after batch 5: `34`
- `src/bigclaw/**/*.py` after batch 6: `33`
- `src/bigclaw/**/*.py` after batch 7: `31`
- `src/bigclaw/**/*.py` after batch 8: `30`
- `src/bigclaw/**/*.py` after batch 9: `28`
- `src/bigclaw/**/*.py` after batch 10: `26`
- `src/bigclaw/**/*.py` after batch 11: `25`
- Net Python module reduction in tranche so far: `20`
- `src/**/*.go` before: `0`
- `src/**/*.go` after: `0`
- Root `pyproject.toml`: absent before and after
- Root `setup.py`: absent before and after

### Validation Record

- `PYTHONPATH=src python3 -m pytest tests/test_repo_gateway.py tests/test_repo_links.py tests/test_repo_registry.py`
  - Result: `5 passed in 0.15s`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py`
  - Result: `7 passed in 0.15s`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result: `42`
- `find src -type f -name '*.go' | sort | wc -l`
  - Result: `0`
- `if [ -f pyproject.toml ]; then echo present; else echo absent; fi`
  - Result: `absent`
- `if [ -f setup.py ]; then echo present; else echo absent; fi`
  - Result: `absent`
- `git diff --check`
  - Result: clean
- `PYTHONPATH=src python3 -m pytest tests/test_validation_policy.py`
  - Result: `2 passed in 0.07s`
- `PYTHONPATH=src python3 - <<'PY' ... PY`
  - Result: `ok`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py`
  - Result: `34 passed in 0.08s`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 2: `40`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_governance.py tests/test_repo_triage.py tests/test_repo_links.py tests/test_repo_registry.py`
  - Result: `7 passed in 0.08s`
- `PYTHONPATH=src python3 - <<'PY' ... PY`
  - Result: `ok`
- `PYTHONPATH=src python3 -m pytest tests/test_execution_contract.py`
  - Result: `7 passed in 0.08s`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 3: `38`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py tests/test_planning.py`
  - Result: `48 passed in 0.10s`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `bigclaw.pilot` / `bigclaw.roadmap`
  - Result: `ok`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on package-root exports
  - Result: `ok`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 4: `36`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py`
  - Result: `3 passed in 0.07s`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `bigclaw.deprecation` / `bigclaw.cost_control`
  - Result: `ok` with expected `DeprecationWarning`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `bigclaw.__main__`
  - Result: `ok`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 5: `34`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `bigclaw.legacy_shim`
  - Result: `ok`
- `python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/runtime.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/create_issues.py scripts/dev_smoke.py`
  - Result: exit `0`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
  - Result: `ok bigclaw-go/internal/legacyshim 3.126s`; `ok bigclaw-go/cmd/bigclawctl 3.964s`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py`
  - Result: `3 passed in 0.07s`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 6: `33`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py`
  - Result: `9 passed in 3.20s`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on workspace bootstrap compatibility modules
  - Result: `ok`
- `python3 -m py_compile src/bigclaw/workspace_bootstrap.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py`
  - Result: exit `0`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 7: `31`
- `python3 -m py_compile scripts/ops/bigclaw_refill_queue.py`
  - Result: exit `0`
- `cd bigclaw-go && go test ./internal/refill -run TestParallelIssueQueueRepoFixtureSelectionStaysAligned`
  - Result: `ok bigclaw-go/internal/refill 2.238s`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 8: `30`
- `PYTHONPATH=src python3 -m pytest tests/test_risk.py tests/test_audit_events.py tests/test_runtime_matrix.py tests/test_observability.py`
  - Result: `18 passed in 0.09s`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `bigclaw.risk` / `bigclaw.audit_events`
  - Result: `ok`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 9: `28`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_event_bus.py tests/test_observability.py`
  - Result: `12 passed in 0.08s`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `bigclaw.repo_board` / `bigclaw.event_bus`
  - Result: `ok`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 10: `26`
- `PYTHONPATH=src python3 -m pytest tests/test_dsl.py tests/test_runtime_matrix.py`
  - Result: `7 passed in 0.08s`
- `PYTHONPATH=src python3 - <<'PY' ... PY` on `WorkflowDefinition.from_json`
  - Result: `ok`
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after batch 11: `25`
