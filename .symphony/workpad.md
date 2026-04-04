# BIG-GO-1179 Workpad

## Plan
- Verify the current repository baseline for Python files and confirm which retired `src/bigclaw/*.py` modules still lack dedicated regression coverage.
- Add a scoped regression tranche for the uncovered retired modules `src/bigclaw/service.py`, `src/bigclaw/scheduler.py`, `src/bigclaw/workflow.py`, and `src/bigclaw/ui_review.py`, asserting both deletion and concrete Go replacement files.
- Tighten the cutover issue-pack documentation so those retired Python surfaces name the active Go-owned replacements directly.
- Run targeted validation commands, capture exact command lines and results, then commit and push the lane branch.

## Acceptance
- The repository still contains no physical `.py` files, validated with `find . -name '*.py' | wc -l`.
- The uncovered retired Python module batch for this lane is pinned by regression coverage to concrete Go/native replacements.
- The migration/cutover docs explicitly point reviewers at the Go-owned replacements for the retired Python surfaces covered by this lane.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche18$'`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche18$'` -> `ok  	bigclaw-go/internal/regression	0.466s`
- `git status --short` -> `M .symphony/workpad.md`, `M docs/go-mainline-cutover-issue-pack.md`, `?? bigclaw-go/internal/regression/top_level_module_purge_tranche18_test.go`
