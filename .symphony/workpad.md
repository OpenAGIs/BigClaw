## BIGCLAW-191 Workpad

### Plan

- [x] Audit the existing control center payload, task metadata, and action surface for approval and degradation concepts already present in `bigclaw-go/internal/api`.
- [x] Add a scoped control-center panel for batch approval visibility and exception downgrade visibility without changing unrelated routes or persistence.
- [x] Extend targeted API tests to cover the new panel summaries and task membership.
- [x] Run targeted Go tests for the modified API package and record the exact command/results.
- [ ] Commit the issue-scoped changes and push branch `BIGCLAW-191` to `origin`.

### Acceptance Criteria

- [x] `GET /v2/control-center` returns a dedicated panel describing approval-required queue work suitable for batch approval review.
- [x] The same response returns a dedicated exception downgrade panel summarizing tasks degraded by policy/exception signals.
- [x] Panel contents honor existing control-center filters and reuse existing task metadata where possible.
- [x] Targeted regression tests cover both panels and pass.

### Validation

- [x] `cd bigclaw-go && go test ./internal/api` -> `ok  	bigclaw-go/internal/api	1.730s`

### Notes

- 2026-03-24: `BIGCLAW-191` arrived without a checked-out branch in this workspace. Recreated the workspace from local `main`, created local branch `BIGCLAW-191`, and kept the implementation scoped to the Go control-center API/tests.
- 2026-03-24: Issue description was empty, so implementation scope is inferred from the title: expose control-center batch approval and exception downgrade visibility as additive API payload panels.
- 2026-03-24: Added `batch_approval_panel` for approval-needed queue items and `exception_downgrade_panel` for policy-blocked tasks plus degraded worker/executor signals in `GET /v2/control-center`.
