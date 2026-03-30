# BIG-GO-1007 Workpad

## Scope

Target the remaining Python tests in the repo/governance/reporting batch:

- `tests/test_governance.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_repo_gateway.py`
- `tests/test_repo_governance.py`
- `tests/test_repo_links.py`
- `tests/test_repo_registry.py`
- `tests/test_repo_rollout.py`
- `tests/test_repo_triage.py`
- `tests/test_reports.py`

Current repository Python file count before changes in this workspace: `108`
Current targeted batch Python file count before this lane: `10`

## Plan

1. Inspect the targeted Python tests and map each one to an existing Go replacement, a deletable obsolete case, or a required keep.
2. Remove redundant Python tests where equivalent Go coverage already exists, keeping scope limited to this batch.
3. Run targeted validation for the touched area and capture exact commands and results.
4. Record delete/replace/keep rationale for every file in the batch and report Python file count impact.
5. Commit and push the scoped changes for `BIG-GO-1007`.

## Acceptance

- Produce the exact `BIG-GO-1007` Python file list for the repo/governance/reporting batch.
- Reduce Python file count in this batch as far as practical without widening scope.
- Document delete/replace/keep rationale for every targeted file.
- Report repository-wide Python file count impact.

## Validation

- `find . -name '*.py' | wc -l`
- `go test ./internal/governance ./internal/repo ./internal/reporting`
- `python3 -m pytest tests/test_repo_rollout.py tests/test_reports.py -q`
- `git status --short`
- `git log -1 --stat`

## Delete Replace Keep Rationale

- Delete `tests/test_governance.py`: replaced by `bigclaw-go/internal/governance/freeze_test.go`.
- Delete `tests/test_repo_board.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- Delete `tests/test_repo_collaboration.py`: replaced in practice by repo-board and repo-surface Go coverage under `bigclaw-go/internal/repo/repo_surfaces_test.go`; Python-only merge wiring no longer needs a dedicated batch file.
- Delete `tests/test_repo_gateway.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- Delete `tests/test_repo_governance.py`: replaced by `bigclaw-go/internal/repo/governance_test.go`.
- Delete `tests/test_repo_links.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- Delete `tests/test_repo_registry.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- Delete `tests/test_repo_triage.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- Keep `tests/test_repo_rollout.py`: no direct Go equivalent found for `build_pilot_rollout_scorecard`, `evaluate_candidate_gate`, `render_pilot_rollout_gate_report`, or repo narrative exports.
- Keep `tests/test_reports.py`: this file still covers broad Python reporting surfaces without a single equivalent Go replacement in this batch.

## Results

- Repository Python file count after changes: `100`
- Targeted batch Python file count after changes: `2`
- Net repository Python file reduction in this workspace: `8`
- Net targeted batch reduction: `8`

## Validation Results

- `go test ./internal/governance ./internal/repo ./internal/reporting`
  - `ok   bigclaw-go/internal/governance 1.530s`
  - `ok   bigclaw-go/internal/repo 1.116s`
  - `ok   bigclaw-go/internal/reporting 1.910s`
- `python3 -m pytest tests/test_repo_rollout.py tests/test_reports.py -q`
  - `36 passed in 0.29s`
- `find . -name '*.py' | wc -l`
  - `100`
