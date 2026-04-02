# BIG-GO-1077 Workpad

## Plan
- Confirm the current `scripts/ops` tranche-1 state, the shell replacements, and any remaining non-historical references that still imply Python operator entrypoints.
- Keep the issue scoped to the residual ops tranche by hardening the Go-only replacement path and removing any remaining default execution or documentation paths that point at the deleted `.py` wrappers.
- Extend targeted regression coverage so the deleted Python tranche stays absent and the replacement shell entrypoints remain executable and wired to `bigclawctl`.
- Run focused validation for the wrapper behavior, regression coverage, and `.py` inventory delta, then commit and push the branch.

## Acceptance
- `scripts/ops/bigclaw_refill_queue.py`, `scripts/ops/bigclaw_workspace_bootstrap.py`, `scripts/ops/symphony_workspace_bootstrap.py`, and `scripts/ops/symphony_workspace_validate.py` remain absent.
- Operator-facing default paths for the tranche use `scripts/ops/*` shell wrappers or `bigclawctl`, with no Python fallback in the active path.
- Regression coverage fails if the deleted tranche reappears or if the replacement wrappers stop dispatching to the Go CLI.
- The branch shows a lower repo-wide `.py` count than the pre-change baseline when measured in this workspace.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `bash scripts/ops/bigclaw_refill_queue --help`
- `bash scripts/ops/bigclaw_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_bootstrap --help`
- `bash scripts/ops/symphony_workspace_validate --help`
