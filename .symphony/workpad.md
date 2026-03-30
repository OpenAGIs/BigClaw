# BIG-GO-1013 Workpad

## Scope

Target a narrow residual-module consolidation batch under `src/bigclaw/**` to
reduce Python module count without expanding into unrelated runtime surfaces.

Batch file list:

- `src/bigclaw/__init__.py`
- `src/bigclaw/connectors.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/observability.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/validation_policy.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_registry.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_validation_policy.py`

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
- `PYTHONPATH=src python3 - <<'PY'`
  `from bigclaw.mapping import map_source_issue_to_task`
  `from bigclaw.validation_policy import enforce_validation_report_policy`
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

### Inventory Impact

- `src/bigclaw/**/*.py` before: `45`
- `src/bigclaw/**/*.py` after batch 1: `42`
- `src/bigclaw/**/*.py` after batch 2: `40`
- Net Python module reduction in tranche so far: `5`
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
