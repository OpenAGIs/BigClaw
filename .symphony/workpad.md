## Codex Workpad

### Plan

- [x] Inventory the Python `runtime` / `scheduler` / `orchestration` / `workflow` tests and scripts that still act as migration references.
- [x] Add a repo-native migration plan for `BIG-GO-906` with phased implementation steps, first-batch conversion targets, validation commands, regression surface, branch strategy, PR slicing, and risks.
- [x] Link the new migration plan from the existing migration/readiness docs so reviewers can find it from the canonical migration path.
- [x] Add regression coverage that pins the new migration-plan doc and its key commands/risk statements.
- [x] Run targeted validation and record the exact commands and results in this workpad.
- [ ] Commit the scoped changes and push the issue branch to the remote.

### Acceptance Criteria

- [x] The repo contains an executable migration plan covering Python runtime/scheduler/orchestration/workflow tests and script migration steps into Go ownership.
- [x] The plan names the first implementation/retrofit batch and maps Python sources/tests to Go packages or scripts.
- [x] Validation commands and the regression blast radius are explicit and reviewer-usable.
- [x] Branch/PR suggestions and migration risks are documented.
- [x] Targeted regression tests for the new documentation pass.

### Validation

- [x] `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && go test ./internal/regression -run 'Test(RuntimeSchedulerOrchestrationMigrationPlanDocs|MigrationFollowUpIndexDocsStayAligned|PlanningFollowUpIndexDocsStayAligned)' -count=1`

### Results

- `2026-03-27`: `go test ./internal/regression -run 'Test(RuntimeSchedulerOrchestrationMigrationPlanDocs|MigrationFollowUpIndexDocsStayAligned|PlanningFollowUpIndexDocsStayAligned)' -count=1` -> `ok  	bigclaw-go/internal/regression	1.164s`

### Notes

- Scope is intentionally limited to `BIG-GO-906`: migration planning, doc integration, and regression coverage for runtime/scheduler/orchestration migration.
- Python runtime surfaces in `src/bigclaw/*.py` are already marked legacy/frozen; this slice documents how remaining tests and scripts retire cleanly behind the Go mainline.
