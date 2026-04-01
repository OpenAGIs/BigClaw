## BIG-GO-1076

### Plan

1. Inspect the residual root `scripts/ops/*.py` wrappers and their current call sites.
2. Delete the four root `scripts/ops/*.py` operator wrappers and route the supported operator path through `scripts/ops/bigclawctl` only.
3. Update the minimal docs/tests that still describe these wrappers as Python entrypoints.
4. Run targeted validation covering wrapper behavior, wrapper help/invocation, and repository `.py` count.
5. Commit and push the scoped branch changes.

### Acceptance

- Root `scripts/ops` no longer contains the residual wrapper Python files:
  - `bigclaw_refill_queue.py`
  - `bigclaw_workspace_bootstrap.py`
  - `symphony_workspace_bootstrap.py`
  - `symphony_workspace_validate.py`
- The supported refill and workspace operator paths are `bash scripts/ops/bigclawctl refill ...`, `bash scripts/ops/bigclawctl workspace bootstrap ...`, and `bash scripts/ops/bigclawctl workspace validate ...`.
- Repository Python file count decreases from the current baseline.

### Validation

- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `rg --files -g '*.py' | wc -l`
- `git status --short`
