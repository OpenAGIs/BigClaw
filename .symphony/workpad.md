# BIG-GO-1077 Workpad

## Plan
- Confirm the live `scripts/ops/*.py` tranche and every non-historical repo reference that still points to those Python operator shims.
- Delete tranche-1 residual Python wrappers in `scripts/ops` and replace any needed operator compatibility surface with shell or direct `bigclawctl` entrypoints.
- Update targeted Go regression coverage and operator-facing docs so the default paths are Go/shell only.
- Run focused validation for purge coverage and the replacement shell/Go entrypoints, then commit and push the branch.

## Acceptance
- `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py` are removed.
- The repo exposes Go or shell replacements for any remaining operator workflows covered by those shims.
- Non-historical default documentation no longer directs operators through those Python files.
- Targeted regression coverage asserts the deleted tranche stays absent and the replacement path remains available.
- Repo-wide `.py` file count decreases from the pre-change baseline.

## Validation
- `rg --files . | rg '\.py$' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `bash scripts/ops/bigclaw_refill_queue --help`
- `bash scripts/ops/bigclaw_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_validate --help`
