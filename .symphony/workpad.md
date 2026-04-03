# BIG-GO-1116 Workpad

## Plan
- Confirm the real lane scope by checking whether the candidate `src/bigclaw/*.py` files still exist and which already have Go-side purge regression coverage.
- Add or tighten repo-native regression coverage for any candidate modules that are already deleted but not yet guarded by a dedicated purge test.
- Update the Go-mainline cutover issue pack so this lane no longer advertises already-removed Python modules as pending migration work.
- Run targeted regression tests, capture exact commands/results, then commit and push the branch.

## Acceptance
- Lane file list is explicit for this slice: `src/bigclaw/run_detail.py`, `src/bigclaw/runtime.py`, `src/bigclaw/scheduler.py`, `src/bigclaw/service.py`, `src/bigclaw/ui_review.py`, `src/bigclaw/workflow.py`.
- Repo-native tests assert those Python paths remain absent and that their Go replacements remain present.
- Documentation no longer treats those already-removed Python files as unresolved backlog in the cutover issue pack.
- Validation section records exact commands and outcomes.

## Validation
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14 -count=1`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche(3|4|7|8|9|11|14)' -count=1`
- `rg --files . | rg '\.py$' | wc -l`
