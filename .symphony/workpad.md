# BIG-GO-949

## Plan
- Inventory `bigclaw-go/scripts/benchmark/**` and map each script to an existing Go/native replacement or a new Go implementation.
- Replace Python benchmark orchestration/certification scripts with Go commands under the existing BigClaw Go tree, keeping invocation paths repo-native and scoped to benchmark workflows.
- Remove Python benchmark assets that become obsolete, update any shell wrappers if they remain useful, and keep outputs/paths stable where practical.
- Run targeted validation for the new Go implementations and benchmark-related tests, then commit and push the issue branch.

## Acceptance
- Lane file list is explicit and covered by the change.
- `bigclaw-go/scripts/benchmark/**` no longer requires Python for active benchmark workflows, or includes a clear delete/replacement plan embodied in code changes.
- Verification commands and exact results are recorded in the final response.
- Residual risks are stated.

## Validation
- `go test ./...` for any newly added packages if scope is small enough, otherwise targeted package tests.
- Direct command execution for the benchmark matrix generator and certification generator against checked-in reports.
- Basic shell wrapper verification if `run_suite.sh` remains.
