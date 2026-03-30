# BIG-GO-1014 Workpad

## Scope

Target the second refill batch of residual Python modules under `src/bigclaw/**`
that already have clear Go ownership in `bigclaw-go` and can be retired without
expanding into unrelated runtime surfaces.

Candidate tranche identified from the repository state:

- `src/bigclaw/governance.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/issue_archive.py`

Matching Go ownership already exists in:

- `bigclaw-go/internal/governance`
- `bigclaw-go/internal/repo`
- `bigclaw-go/internal/githubsync`
- `bigclaw-go/internal/issuearchive`

Repository inventory at start of lane:

- `src/bigclaw/*.py` files: `45`
- `src/bigclaw/*.go` files: `0`
- root `pyproject.toml`: absent
- root `setup.py`: absent

## Plan

1. Inspect the candidate tranche modules and their Python test coverage to
   confirm they are residual-only surfaces that can be removed safely.
2. Delete Python modules that are superseded by existing Go implementations and
   remove any package exports or tests that only exercised those retired Python
   surfaces.
3. Keep the change scoped to `src/bigclaw/**`, impacted tests, and package
   surface files only where required by imports.
4. Run targeted validation that proves the retired modules are gone, package
   exports stay coherent, and Go ownership remains test-covered.
5. Record exact file-count impact for `py files`, `go files`,
   `pyproject.toml`, and `setup.py`.
6. Commit and push the scoped branch for `BIG-GO-1014`.

## Acceptance

- Directly reduce residual Python assets under `src/bigclaw/**`.
- Minimize `.py` file count for the selected tranche without broad unrelated
  refactors.
- Report impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.
- Validation record contains exact commands and outcomes for this lane.

## Validation

- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
- `find src/bigclaw -type f -name '*.go' | sort | wc -l`
- `test -f pyproject.toml; echo $?`
- `test -f setup.py; echo $?`
- `python3 -m pytest` on targeted Python tests still expected to remain after
  the tranche is removed
- `cd bigclaw-go && go test` on packages that already own the retired surfaces
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `src/bigclaw/repository.py`
  - Added.
  - Reason: consolidated the residual repository-support Python surfaces that
    were already covered by Go ownership areas, so the package keeps import
    compatibility while reducing file count.
- `src/bigclaw/reports.py`
  - Replaced.
  - Reason: absorbed `issue_archive.py` so the issue-archive residual no longer
    needs its own module file.
- `src/bigclaw/__init__.py`
  - Replaced.
  - Reason: package init now installs compatibility submodules for the retired
    `repo_*`, `github_sync`, and `issue_archive` import paths.
- `src/bigclaw/observability.py`
  - Replaced.
  - Reason: imports the consolidated repository surface directly instead of the
    retired split modules.
- Deleted residual tranche files:
  - `src/bigclaw/github_sync.py`
  - `src/bigclaw/issue_archive.py`
  - `src/bigclaw/repo_board.py`
  - `src/bigclaw/repo_gateway.py`
  - `src/bigclaw/repo_governance.py`
  - `src/bigclaw/repo_links.py`
  - `src/bigclaw/repo_plane.py`
  - `src/bigclaw/repo_registry.py`
  - `src/bigclaw/repo_triage.py`
  - `src/bigclaw/repo_commits.py`
  - `src/bigclaw/governance.py`
- `src/bigclaw/planning.py`
  - Replaced.
  - Reason: absorbed `governance.py` and now also serves the compatibility
    `bigclaw.governance` module surface.

### Inventory Impact

- `src/bigclaw` Python files before: `45`
- `src/bigclaw` Python files after first pass: `37`
- `src/bigclaw` Python files after continuation pass: `35`
- Net Python file reduction: `10`
- `src/bigclaw` Go files before: `0`
- `src/bigclaw` Go files after: `0`
- Root `pyproject.toml` before/after: absent
- Root `setup.py` before/after: absent

### Validation Record

- `python3 -m compileall src/bigclaw`
  - Result: success
- `find src/bigclaw -type f -name '*.py' | sort | wc -l`
  - Result after first pass: `37`
  - Result after continuation pass: `35`
- `find src/bigclaw -type f -name '*.go' | sort | wc -l`
  - Result after: `0`
- `printf 'pyproject='; test -f pyproject.toml; echo $?; printf 'setup='; test -f setup.py; echo $?`
  - Result: `pyproject=1`, `setup=1`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_gateway.py tests/test_repo_governance.py tests/test_repo_registry.py tests/test_repo_links.py tests/test_repo_triage.py tests/test_github_sync.py tests/test_observability.py tests/test_reports.py`
  - Result: `57 passed in 1.20s`
- `cd bigclaw-go && go test ./internal/repo ./internal/governance ./internal/githubsync ./internal/issuearchive`
  - Result: `ok   bigclaw-go/internal/repo 0.824s`, `ok   bigclaw-go/internal/governance 2.104s`, `ok   bigclaw-go/internal/githubsync 3.887s`, `ok   bigclaw-go/internal/issuearchive 1.688s`
- `PYTHONPATH=src python3 -m pytest tests/test_governance.py tests/test_planning.py tests/test_repo_gateway.py tests/test_repo_board.py tests/test_repo_collaboration.py tests/test_repo_governance.py tests/test_repo_registry.py tests/test_repo_links.py tests/test_repo_triage.py tests/test_github_sync.py tests/test_observability.py tests/test_reports.py`
  - Result: `75 passed in 1.15s`
- `cd bigclaw-go && go test ./internal/repo ./internal/governance`
  - Result: `ok   bigclaw-go/internal/repo (cached)`, `ok   bigclaw-go/internal/governance (cached)`
- `git diff --check`
  - Result: clean
