# BIG-GO-1529

## Plan
- Record the current repository `.py` file baseline and identify the largest residual Python-heavy directory.
- Inspect references to that directory to confirm whether it is already outside the active Go-only path and can be removed without expanding scope.
- Delete the selected residual Python files and any now-stale references that must change for tests or documentation evidence.
- Run targeted validation for the affected migration/refill surface and record exact commands and results.
- Commit the change on `BIG-GO-1529` and push the branch to `origin`.

## Acceptance
- The repository contains fewer physical `.py` files after the change than before it.
- The change removes files from the largest residual Python directory that is safe to delete in this migration lane.
- Final notes include before/after `.py` counts and exact removed-file evidence.
- Targeted tests covering the affected refill/migration surface pass.
- The branch is committed and pushed to `origin/BIG-GO-1529`.

## Validation
- `rg --files -g '*.py' | wc -l`
- Reference search for the removed directory/files with `rg`
- Targeted test command(s) chosen after identifying the deleted surface
- `git status --short`
- `git show --stat --name-status --oneline HEAD`

## Results
- Baseline Python file count before edits: `138`
- Python file count after edits: `135`
- Removed file evidence:
  - `src/bigclaw/parallel_refill.py`
  - `tests/test_parallel_refill.py`
  - `scripts/ops/bigclaw_refill_queue.py`
- Reference sweep:
  - `rg -n "src/bigclaw/parallel_refill.py|scripts/ops/bigclaw_refill_queue.py|test_parallel_refill" README.md docs scripts tests src workflow.md bigclaw-go .github`
  - Result: no matches
- Targeted tests:
  - `cd bigclaw-go && go test ./internal/refill ./cmd/bigclawctl`
  - Result:
    - `ok  	bigclaw-go/internal/refill	0.233s`
    - `ok  	bigclaw-go/cmd/bigclawctl	0.165s`
