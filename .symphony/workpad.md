# BIG-GO-1037

## Plan
- Verify the governance/reporting-adjacent Python test files that already have equivalent Go coverage.
- Remove only the Python test files whose behavior is already covered by Go tests in `bigclaw-go`.
- Add any missing Go assertions only if required to preserve parity for the removed Python coverage.
- Run targeted Go tests for the affected packages and capture exact commands and results.
- Commit the scoped migration changes and push the branch to the remote.

## Acceptance
- Python file count decreases within the scoped governance/reporting tranche.
- Go test coverage remains present for the removed Python test behavior.
- No unrelated Python files are added or expanded.
- The final change can state exactly which Python files were removed and which Go test files provide the replacement coverage.

## Validation
- `go test ./internal/governance ./internal/repo`
- `git diff --stat`
- `git status --short`
- `git push <remote> <branch>`
