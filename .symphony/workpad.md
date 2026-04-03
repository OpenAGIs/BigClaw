# BIG-GO-1069 Workpad

## Plan
- Capture the current residual Python baseline with a focused inventory of `scripts/ops/*.py` wrappers and the repo-wide `.py` count.
- Delete the remaining operator-facing Python wrapper assets in `scripts/ops/` that already have Go-first replacements:
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- Update live docs and regression checks so they point directly at `bash scripts/ops/bigclawctl ...` instead of the deleted Python paths.
- Trim Go-side migration-only helper coverage that existed solely for the deleted wrappers, while preserving compile-check coverage for the remaining Python source compatibility layer.
- Run targeted validation, record exact commands and outcomes, then commit and push the branch.

## Acceptance
- This batch's Python asset list is explicit and limited to the four `scripts/ops/*.py` compatibility wrappers above.
- Those four Python files are removed from the repository, reducing the physical `.py` file count.
- Active operator docs and regression coverage no longer present those Python scripts as supported entrypoints.
- Go-first replacements remain directly verifiable through `scripts/ops/bigclawctl`.
- Final closeout includes exact validation commands, results, Python file count impact, and residual risks.

## Validation
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `rg --files -g '*.py' -g '*.pyi' -g '*.pyw' | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression -run 'TestTopLevelModulePurgeTranche14|TestRunLegacyPythonCompileCheckJSONOutputDoesNotEscapeArrowTokens'`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
