# BIG-GO-1065 Workpad

## Plan
- Confirm the residual Python assets in this batch and capture the pre-change `.py` file baseline.
- Delete the remaining batch-owned Python test files under `tests/`.
- Remove active references that still point operators or source metadata at the deleted tests.
- Add Go regression coverage that locks the deleted file set and verifies representative Go-native replacement paths remain present.
- Run targeted validation, capture exact commands and outcomes, then commit and push the issue branch.

## Acceptance
- The batch-owned Python test asset list is explicit and removed from the repo.
- Active bootstrap/docs/source references no longer direct users toward the deleted Python tests.
- A Go regression test asserts the deleted Python assets stay absent and the expected Go replacement coverage files exist.
- Validation commands and outcomes are recorded for the closeout.
- The overall repository `.py` file count decreases from the pre-change baseline.

## Validation
- `rg --files . | rg '\.py$' | wc -l`
- `go test ./internal/regression -run 'TestTopLevelModulePurgeTranche14|TestLane8|TestE2E'`
- `go test ./internal/workflow ./internal/queue ./internal/api`
- `bash scripts/dev_bootstrap.sh`
- `git status --short`
