## BIGCLAW-186 Workpad

### Plan

- [x] Inspect the Go control-center action and audit flow to identify the smallest change set that adds batch governance audit closure.
- [x] Extend the control-center action surface with batch governance execution metadata, audit linkage, and rollback entry generation.
- [x] Add targeted API tests covering batch audit logging, rollback entry visibility, and closeout behavior.
- [x] Run targeted Go tests, record exact commands and results, then commit and push the branch.

### Acceptance

- [x] Control-center batch governance actions can be executed through the existing action surface without widening scope beyond this issue.
- [x] Batch action audit entries retain enough metadata to reconstruct the batch, inspect per-task results, and discover a rollback entry.
- [x] A rollback entry is exposed for reversible batch collaboration actions and is itself auditable.
- [x] Targeted Go tests pass for the new batch action + audit flow.

### Validation

- [x] `cd bigclaw-go && go test ./internal/api -run 'TestV2ControlCenter(BatchTakeoverAuditAndRollback|AuditFiltersOwnerReviewerAndScope|AuditIncludesCheckpointResetSummary)'` -> `ok  	bigclaw-go/internal/api	1.401s`
- [x] `cd bigclaw-go && go test ./internal/api -run 'TestV2ControlCenter(ActionsAndRunDetail|AuthorizationEnforcedByRole|AuditFiltersOwnerReviewerAndScope|AuditIncludesCheckpointResetSummary|BatchTakeoverAuditAndRollback)'` -> `ok  	bigclaw-go/internal/api	0.601s`

### Notes

- Keep the implementation scoped to the Go control-center API and its tests.
- Record exact validation commands and outcomes in the final closeout.
- Batch support is intentionally constrained to collaboration actions so queue/system control semantics stay unchanged in this issue slice.
