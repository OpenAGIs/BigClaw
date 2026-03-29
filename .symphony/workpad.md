## BIG-GO-950

### Plan
1. Inventory `bigclaw-go/scripts/e2e/**` and `bigclaw-go/scripts/migration/**`, using the current worktree as source of truth because `reports/go-migration-lanes-2026-03-29.md` is not present in this checkout.
2. Replace Python entrypoints that are directly invoked by shell workflows with Go/native-command equivalents, keeping behavior and output paths stable where practical.
3. Convert Python-only script tests that cover migrated entrypoints into Go tests.
4. Add a lane-scoped migration inventory and deletion plan for remaining large Python assets that are not safely portable within this change.
5. Run targeted verification, then commit and push the issue branch.

### Acceptance
- Lane file list is documented from the checked-out repository contents.
- Migrated entrypoints have Go/native-command replacements, and remaining files have an explicit delete/follow-up plan.
- Validation commands and exact results are recorded.
- Residual risks are documented.

### Validation
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `cd bigclaw-go && go test ./internal/...`
- `cd bigclaw-go && ./scripts/e2e/run_all.sh` with local lanes constrained as needed for targeted verification
- `git status --short`
