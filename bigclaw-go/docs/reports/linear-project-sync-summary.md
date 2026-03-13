# Linear Project Sync Summary

## BigClaw v4.0 Execution Pack

- Project status: `Completed`
- Completion date: `2026-03-13`
- Linear project id: `a8ea6b90-7918-4b50-8cc0-359e556cdf9f`
- Scope summary: execution-pack work and the Go rewrite batch are fully closed in Linear.

### Milestones

- `Architecture`: `2/2` done, target `2026-03-16`
- `Core Runtime`: `5/5` done, target `2026-03-20`
- `Executors`: `2/2` done, target `2026-03-22`
- `Migration & Benchmark`: `2/2` done, target `2026-03-25`

### Notes

- `OPE-175` through `OPE-186` are now closed for the current rewrite scope.
- A project update has been posted in Linear to capture the closure summary and evidence package.
- Follow-up hardening remains outside the closed execution-pack baseline.

## BigClaw v2.0

- Project status: `Planned`
- Project target date: `2026-05-08`
- Linear project id: `46d206d6-a329-493f-b83d-435f39e7506f`
- Scope summary: v2.0 remains frozen for execution, but now has dated phase placeholders plus an explicit priority-first activation order.

### Milestones

- `Phase 1`: `6` issues, target `2026-04-03`
- `Phase 2`: `6` issues, target `2026-04-17`
- `Phase 3`: `6` issues, target `2026-05-01`
- `Shared`: `7` issues, target `2026-05-08`

### Notes

- Current v2.0 state distribution has moved from pure planning into active execution: `Backlog` plus `In Progress` for the first Phase 1 slice.
- Activation order remains explicit: `Phase 1 -> Phase 2 -> Phase 3 -> Shared`.
- Phase 1 backend foundation is now in progress for `OPE-69`, `OPE-70`, `OPE-71`, `OPE-72`, and `OPE-73`.
- The control-center slice now also includes queue-backed live task inspection plus `cancel` and `transfer_to_human` operational actions.
- The same slice now adds filtered queue views, budget/risk/priority summaries, worker-pool packaging, and a dedicated control-center audit endpoint.
- The control-center slice also now enforces explicit role-based mutating actions and returns allowed actions to callers for UI gating.
- Evidence package added in `docs/reports/v2-phase1-operations-foundation-report.md`.
- Linear comments were posted on `2026-03-13` to capture backend evidence and validation via `go test ./...`.

## BigClaw AgentHub Integration

- Project status: `Completed`
- Completion date: `2026-03-13`
- Linear project id: `42ad2ff4-4d89-44d4-8e9f-e87e4e4af531`
- Scope summary: AgentHub integration is historically grouped and fully closed in Linear.

### Milestones

- `Repo-Native Collaboration Plane`: `6` issues, target `2026-03-07`
- `Collaboration Console & Reporting`: `5` issues, target `2026-03-10`
- `Governance, Security & Rollout`: `4` issues, target `2026-03-13`

### Notes

- All AgentHub issues remain `Done` after milestone backfill.
- A project update has been posted in Linear to capture the completed historical structure.

## BigClaw v1.0

- Project status: `Completed`
- Completion date: `2026-03-13`
- Linear project id: `aac4caa5-f584-4b0d-88c5-d184d798d353`
- Scope summary: v1.0 is historically grouped and fully closed in Linear.

### Milestones

- `Task Intake & Connectors`: `3` issues, target `2026-02-10`
- `Control Plane`: `4` issues, target `2026-02-17`
- `Execution & Workflow`: `8` issues, target `2026-02-24`
- `Memory, Audit & Evaluation`: `6` issues, target `2026-03-03`
- `Pilot & Commercial Validation`: `2` issues, target `2026-03-13`

### Notes

- All v1.0 issues remain `Done` after milestone backfill.
- A project update has been posted in Linear to capture the completed historical structure.

