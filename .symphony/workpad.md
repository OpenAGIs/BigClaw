# BIG-GO-1056 Workpad

## Plan
- Confirm the root `dev_smoke` Python entrypoint is absent from the tracked tree and identify any remaining live references outside archived reports.
- Update repo-facing entry surfaces that still describe or validate the removed Python path so the Go `bigclawctl dev-smoke` route is the only supported root smoke entrypoint.
- Add or tighten regression coverage to pin the removal of `scripts/dev_smoke.py` and the continued availability of the Go smoke command.
- Run targeted validation including `.py` file counts, focused searches for stale references, and Go tests for the affected command/regression surface.
- Commit the scoped changes and push the branch tip to origin.

## Acceptance
- `scripts/dev_smoke.py` is absent from the tracked repository after the change.
- Non-archived README / workflow / hooks / CI-facing references do not call or recommend `scripts/dev_smoke.py`.
- The supported root dev smoke path is Go-only via `bash scripts/ops/bigclawctl dev-smoke` or the equivalent Go command.
- Regression coverage fails if the deleted root Python smoke path returns or if the documented Go path regresses.
- Repository `.py` file count is not increased by this slice, and the root dev smoke Python entrypoint remains removed.

## Validation
- `find . -type f -name '*.py' | wc -l`
- `rg -n "dev_smoke\\.py|scripts/dev_smoke\\.py|python3? .*dev_smoke\\.py|PYTHONPATH=.*dev_smoke\\.py" . -g '!reports/**' -g '!.git/**'`
- `rg -n "dev-smoke|bigclawctl dev-smoke" README.md workflow.md scripts .github bigclaw-go docs -g '!reports/**'`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `bash scripts/ops/bigclawctl dev-smoke`
- Capture exact command results in the closeout summary and local tracker comment.
