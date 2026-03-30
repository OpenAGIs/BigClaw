# BIG-GO-987

## Plan
- Inventory Python files related to repo, governance, and reporting tests plus corresponding Go implementations/tests.
- Determine which Python tests are fully superseded by Go coverage and can be removed without expanding scope.
- Apply scoped deletions or minimal follow-up edits needed to keep the repository consistent.
- Run targeted validation for affected Go packages and repository Python-file counts.
- Commit the change set and push the branch to the remote.

## Acceptance
- Produce an explicit list of Python files in this batch.
- Reduce Python file count in the targeted repo/governance/reporting test area where replacement coverage already exists.
- Record the rationale for each retained, replaced, or deleted file.
- Report the net effect on total repository Python file count.

## Validation
- Use `rg --files -g '*.py'` before and after changes to measure total Python file count.
- Run targeted `go test` commands for replacement coverage in repo, governance, and reporting packages.
- Confirm git status only includes issue-scoped edits before committing.

## Batch Inventory
- Deleted: `tests/test_governance.py`
- Deleted: `tests/test_repo_board.py`
- Deleted: `tests/test_repo_gateway.py`
- Deleted: `tests/test_repo_governance.py`
- Deleted: `tests/test_repo_links.py`
- Deleted: `tests/test_repo_registry.py`
- Deleted: `tests/test_repo_triage.py`
- Consolidated and retained: `tests/test_reports.py`
- Deleted after consolidation: `tests/test_repo_rollout.py`
- Deleted after consolidation: `tests/test_repo_collaboration.py`

## Rationale
- `tests/test_governance.py`: replaced by `bigclaw-go/internal/governance/freeze_test.go`, which covers scope-freeze manifest round-trip, audit gaps, readiness scoring, and report rendering.
- `tests/test_repo_board.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`, which covers repo discussion board post/reply/filter behavior.
- `tests/test_repo_gateway.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`, which covers commit/lineage/diff normalization plus gateway error decoding and audit payload behavior.
- `tests/test_repo_governance.py`: replaced by `bigclaw-go/internal/repo/governance_test.go`, which covers role permission checks and required audit field rules.
- `tests/test_repo_links.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`, which covers run-commit binding, accepted commit hash resolution, and invalid role handling.
- `tests/test_repo_registry.py`: replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`, which covers repo-space resolution, default channel generation, and agent resolution.
- `tests/test_repo_triage.py`: replaced by `bigclaw-go/internal/triage/repo_test.go`, which covers lineage-driven triage recommendations and approval evidence packets.
- `tests/test_reports.py`: retained as the consolidated Python reporting surface. It still covers report studio, launch/final-delivery closeout, pilot portfolio, shared view rendering, repo rollout exports, and collaboration-thread merging that are not yet fully mirrored by Go tests.
- `tests/test_repo_rollout.py`: deleted by consolidation into `tests/test_reports.py`; kept coverage without needing a separate Python file because direct Go replacements for these rollout/narrative helpers do not yet exist.
- `tests/test_repo_collaboration.py`: deleted by consolidation into `tests/test_reports.py`; kept coverage without needing a separate Python file because merged collaboration-thread behavior does not yet have a direct Go replacement.
- Retained Python source modules under `src/bigclaw/` for this sweep because they are still referenced by active Python code paths:
  - `src/bigclaw/governance.py` is imported by `src/bigclaw/planning.py`, `src/bigclaw/__init__.py`, and exercised via `tests/test_planning.py`.
  - `src/bigclaw/repo_board.py` is still exercised by `tests/test_reports.py` through collaboration-thread merging coverage.
  - `src/bigclaw/repo_links.py` is imported by `src/bigclaw/observability.py`.
  - The remaining `repo_*` Python modules were not removed because this ticket is scoped to final sweep tests, and their deletion would require a broader source-level migration pass.

## Validation Results
- Command: `rg --files -g '*.py' | wc -l`
  - Before changes: `116`
  - After deleting replaced tests: `109`
  - After consolidating retained repo/reporting tests: `107`
- Command: `cd bigclaw-go && go test ./internal/governance ./internal/repo ./internal/triage ./internal/reporting`
  - Result: `ok   bigclaw-go/internal/governance`
  - Result: `ok   bigclaw-go/internal/repo`
  - Result: `ok   bigclaw-go/internal/triage`
  - Result: `ok   bigclaw-go/internal/reporting`
- Command: `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
  - Result: `37 passed in 0.20s`
- Command: `PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_reports.py -q`
  - Result: `51 passed in 0.10s`
- Command: `git push -u origin BIG-GO-987`
  - Result: pushed branch `BIG-GO-987` to `origin`
