# BIG-GO-1058

## Plan
- Inspect the refill queue Python shim, current `bigclawctl refill` replacement path, and all repo references that still point at the Python entrypoint.
- Update README and workflow-facing guidance to use the Go-first `bash scripts/ops/bigclawctl refill ...` path and remove stale mentions of `scripts/ops/bigclaw_refill_queue.py`.
- Delete `scripts/ops/bigclaw_refill_queue.py` and add a targeted regression check so the removed shim path stays absent while the Go replacement remains present.
- Run focused validation for reference cleanup, `.py` count reduction, and the relevant Go tests, then commit and push the issue branch.

## Acceptance
- `scripts/ops/bigclaw_refill_queue.py` is deleted from the repository.
- README / workflow / hooks / CI no longer direct operators to the removed Python refill queue entrypoint.
- The supported refill queue operator path is `bash scripts/ops/bigclawctl refill ...`.
- Repository evidence shows the tracked `.py` file count decreased by one from the pre-change baseline.
- Targeted regression coverage passes for the deleted shim path and its Go replacement.

## Validation
- `find . -type f -name '*.py' | sort | wc -l`
- `rg -n "bigclaw_refill_queue\\.py|python3 scripts/ops/bigclaw_refill_queue\\.py" README.md workflow.md .github .githooks docs bigclaw-go --hidden`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression`

## Execution Result
- Removed `scripts/ops/bigclaw_refill_queue.py` and kept `bash scripts/ops/bigclawctl refill ...` as the only supported refill queue entrypoint.
- Updated `README.md` so local orchestration guidance no longer points at the deleted Python refill shim.
- Updated migration tracking docs to reflect that the refill shim is retired while other migration-only wrappers remain.
- Added `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go` so the deleted refill shim path stays absent and the Go replacement files stay present.
- Updated `bigclaw-go/internal/legacyshim/wrappers_test.go` to stop referencing the removed refill shim path.

## Validation Result
- `find . -type f -name '*.py' | sort | wc -l`
  - passed; count dropped from `46` before the change to `45` after deleting `scripts/ops/bigclaw_refill_queue.py`
- `rg -n "bigclaw_refill_queue\\.py|python3 scripts/ops/bigclaw_refill_queue\\.py" README.md workflow.md .github .githooks --hidden`
  - passed; no matches in README / workflow / hooks / CI surfaces
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression`
  - passed
- `bash scripts/ops/bigclawctl refill --help`
  - passed; help output confirms the supported Go refill entrypoint and flags

## Remaining Blocker
- PR creation is blocked in this workspace because `gh auth status` reports no logged-in GitHub host and no `GITHUB_TOKEN`/`GH_TOKEN` environment variable is present.
- Branch `symphony/BIG-GO-1058` is pushed to origin and local/remote SHAs match at `1c4d9e531faaf7cf0e0a3641c46f3b941179c5d3`.
