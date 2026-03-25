# BIGCLAW-194 Workpad

## Plan

1. Inspect the active Go mainline surfaces for task lifecycle orchestration, control-plane reporting, and batch operation handling to identify the narrowest implementation slice that fits BIG-vNext-024.
2. Implement the issue-scoped change in the relevant module(s), keeping the change set limited to lifecycle orchestration overview and batch start/stop strategy output.
3. Add or update targeted tests that lock the new behavior.
4. Run targeted validation commands, capture exact commands and outcomes, then commit and push the branch.

## Acceptance

- The codebase exposes a concrete task lifecycle orchestration overview that includes batch start/stop strategy details in the relevant reporting or control-plane surface.
- The implementation is scoped to this issue and does not broaden into unrelated scheduler or runtime refactors.
- Targeted automated tests cover the new behavior and pass locally.
- The final branch contains a commit for BIGCLAW-194 and is pushed to the configured remote.

## Validation

- Inspect the changed surface with focused unit tests in `bigclaw-go/internal/...` or `tests/...`, depending on the implementation target.
- Record the exact test commands run and whether they passed or failed.
- Verify git state before closeout with `git status --short`, `git log -1 --stat`, and `git push`.

## Execution Notes

- Added a new `task_lifecycle_orchestration` control-center payload surface in `bigclaw-go/internal/api/task_lifecycle_orchestration_surface.go`.
- Wired the surface into the `/v2/control-center` response in `bigclaw-go/internal/api/v2.go`.
- Added a focused API contract test in `bigclaw-go/internal/api/server_test.go`.

## Validation Results

- `go test ./bigclaw-go/internal/api -run 'TestV2ControlCenterIncludesTaskLifecycleOrchestrationOverview|TestV2ControlCenterIncludesAdmissionPolicySummary|TestV2ControlCenterActionsAndRunDetail'`
  - Failed from repo root: `go: cannot find main module, but found .git/config in /Users/openagi/code/bigclaw-workspaces/BIGCLAW-194`
- `cd bigclaw-go && go test ./internal/api -run 'TestV2ControlCenterIncludesTaskLifecycleOrchestrationOverview|TestV2ControlCenterIncludesAdmissionPolicySummary|TestV2ControlCenterActionsAndRunDetail'`
  - Passed: `ok  	bigclaw-go/internal/api	2.917s`
