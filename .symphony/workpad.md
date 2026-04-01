# BIG-GO-1078 Workpad

## Plan
- Confirm the remaining `scripts/ops/*.py` wrappers are thin compatibility shims with Go replacements already available in `scripts/ops/bigclawctl`.
- Remove the residual operator-facing Python wrapper files from `scripts/ops` and switch repo-default references onto the Go or shell entrypoints.
- Update targeted docs and regression tests so the deleted Python entrypoints are no longer advertised and cannot silently return.
- Run targeted validation, capture the exact commands and results, then commit and push the issue branch.

## Acceptance
- `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py` are deleted from the repository.
- Repo-default operator guidance points at `bash scripts/ops/bigclawctl ...` instead of the deleted Python entrypoints.
- Regression coverage asserts the tranche-2 Python wrapper files stay absent while the Go replacement surfaces remain present.
- Regression coverage also asserts `scripts/ops` stays Python-free.
- Validation records show the `.py` file count drops and the replacement Go entrypoints still execute successfully.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `find scripts/ops -maxdepth 1 -type f -name '*.py'`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
