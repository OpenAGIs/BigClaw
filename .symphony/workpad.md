# BIG-GO-1015 Workpad

## Scope

Target tranche 3 of the remaining `src/bigclaw/**` repository-surface Python
helpers that already have repo-native Go replacements and no longer need to
exist as active Python modules.

Batch file list:

- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_commits.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_governance.py`
- `src/bigclaw/repo_registry.py`
- `src/bigclaw/repo_triage.py`
- `tests/test_repo_board.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_triage.py`
- `tests/test_repo_collaboration.py`

Context at start of lane:

- `src/bigclaw` Python files: `45`
- `bigclaw-go` Go files: `267`
- Root `pyproject.toml`: absent
- Root `setup.py`: absent

Keep-out files for this lane:

- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/observability.py`

Reason:

- `observability.py` still imports `repo_links.bind_run_commits` and
  `repo_plane.RunCommitLink`, so deleting those modules in this lane would
  broaden scope beyond the safe tranche.

## Plan

1. Remove the six repo-surface Python modules that already map to checked-in Go
   implementations under `bigclaw-go/internal/repo` and
   `bigclaw-go/internal/triage`.
2. Remove Python tests that only exercised those deleted modules.
3. Update any remaining Python tests that referenced the removed board helper so
   the suite remains coherent after deletion.
4. Run targeted validation for remaining observability and repo-link surfaces,
   plus inventory counts and diff hygiene.
5. Commit and push the scoped lane branch for `BIG-GO-1015`.

## Acceptance

- Directly reduce repository-resident Python assets under `src/bigclaw/**`.
- Keep changes scoped to the tranche-3 repo helper slice only.
- Report exact impact on `py files`, `go files`, `pyproject.toml`, and
  `setup.py`.
- Record exact validation commands and results.
- End with committed and pushed repository changes; do not substitute tracker
  state for repo results.

## Validation

- `find src/bigclaw -type f -name '*.py' | wc -l`
- `find bigclaw-go -type f -name '*.go' | wc -l`
- `test -f pyproject.toml && echo present || echo absent`
- `test -f setup.py && echo present || echo absent`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py`
- `git diff --check`
- `git status --short`
- `git log -1 --stat`

## Results

### File Disposition

- `src/bigclaw/repo_board.py`
  - Deleted.
  - Reason: repo discussion-board helper already exists in
    `bigclaw-go/internal/repo/board.go` and had no remaining production Python
    callers.
- `src/bigclaw/repo_commits.py`
  - Deleted.
  - Reason: repo commit and lineage structs already exist in
    `bigclaw-go/internal/repo/commits.go`; remaining Python usage was only the
    deleted gateway test.
- `src/bigclaw/repo_gateway.py`
  - Deleted.
  - Reason: gateway client contract and normalization logic already exist in
    `bigclaw-go/internal/repo/gateway.go` and had no remaining Python imports.
- `src/bigclaw/repo_governance.py`
  - Deleted.
  - Reason: repo permission contract already exists in
    `bigclaw-go/internal/repo/governance.go` and had no remaining production
    Python imports.
- `src/bigclaw/repo_registry.py`
  - Deleted.
  - Reason: repo registry logic already exists in
    `bigclaw-go/internal/repo/registry.go` and had no remaining Python imports.
- `src/bigclaw/repo_triage.py`
  - Deleted.
  - Reason: repo triage recommendation logic already exists in
    `bigclaw-go/internal/repo/triage.go` and had no remaining Python imports.
- `tests/test_repo_board.py`
  - Deleted.
  - Reason: exercised deleted Python-only board helper.
- `tests/test_repo_gateway.py`
  - Deleted.
  - Reason: exercised deleted Python-only gateway helper.
- `tests/test_repo_governance.py`
  - Deleted.
  - Reason: exercised deleted Python-only governance helper.
- `tests/test_repo_registry.py`
  - Deleted.
  - Reason: exercised deleted Python-only registry helper.
- `tests/test_repo_triage.py`
  - Deleted.
  - Reason: exercised deleted Python-only triage helper.
- `tests/test_repo_collaboration.py`
  - Replaced.
  - Reason: preserved the collaboration merge assertion while removing the last
    dependency on the deleted `RepoDiscussionBoard` helper.

### Inventory Impact

- `src/bigclaw` Python files before: `45`
- `src/bigclaw` Python files after: `39`
- Net `src/bigclaw` reduction: `6`
- Repository-wide Python files before: `108`
- Repository-wide Python files after: `97`
- Net repository-wide Python reduction: `11`
- `bigclaw-go` Go files before: `267`
- `bigclaw-go` Go files after: `267`
- Net Go file reduction: `0`
- Root `pyproject.toml`: absent before and after
- Root `setup.py`: absent before and after

### Validation Record

- `rg -n "repo_board|repo_commits|repo_gateway|repo_governance|repo_registry|repo_triage" src tests || true`
  - Result: no matches
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_links.py tests/test_repo_rollout.py tests/test_repo_collaboration.py`
  - Result: `11 passed in 0.17s`
