# BIG-GO-902 Workpad

## Plan

1. Audit the remaining repo-root `scripts/*.py` and `scripts/ops/*` automation entrypoints against the existing `bigclaw-go/cmd/bigclawctl` command surface.
2. Implement the smallest missing migration slice so common automation entrypoints resolve through Go-owned CLI behavior while keeping file-name compatibility shims intact.
3. Update the migration plan and operator docs so the first implemented batch, deferred backlog, validation commands, regression surface, branch/PR suggestion, and risks are explicit.
4. Run targeted Go/Python/script validation, record exact commands/results, then commit and push the branch.

## Acceptance

- Executable migration plan exists for repo-level script entrypoints and compatibility shims.
- First-batch implementation / retrofit list is explicit and consistent with the actual CLI surface.
- Validation commands and regression surface are documented.
- Branch / PR suggestion and risk notes are included.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/refill`
- `PYTHONPATH=src python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py`
- `bash scripts/ops/bigclawctl dev-smoke`
- `python3 scripts/create_issues.py --help`
- `bash scripts/ops/bigclawctl issue --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
