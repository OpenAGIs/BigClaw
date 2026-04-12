## Codex Workpad

```text
/Users/openagi/code/bigclaw-workspaces/BIG-GO-26
```

### Plan

- [x] Audit remaining non-core Python under active tooling and support paths (`scripts/ops`, `bigclaw-go/scripts`, packaging metadata, related docs/tests).
- [x] Replace or retire active Python support-tool entrypoints with Go or shell-owned paths where the repo already has Go-first ownership.
- [x] Remove physically residual Python files and stale Python packaging/config only when no active Go-mainline dependency remains.
- [x] Update tests/docs to reflect the surviving support surface and validate the scoped cleanup.

### Acceptance Criteria

- [x] Python support/tooling residuals covered by this slice are removed from active repo paths or reduced to intentional archival references only.
- [x] The repo’s default operator/tooling flow remains Go-first and does not depend on removed Python entrypoints.
- [x] Targeted validation covering the touched support surfaces passes.

### Validation

- [x] `rg -n "scripts/ops/.*\\.py|scripts/e2e/.*\\.py|scripts/migration/.*\\.py|scripts/benchmark/.*\\.py|scripts/dev_smoke.py|scripts/create_issues.py|python3 scripts/e2e|python3 scripts/migration|python3 scripts/benchmark" README.md docs/go-mainline-cutover-issue-pack.md bigclaw-go bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go | head -400`
  Result: only intentional retired-reference lines remain in `docs/go-mainline-cutover-issue-pack.md` plus one future-looking note in `bigclaw-go/docs/reports/broker-failover-fault-injection-validation-pack.md`.
- [x] `cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestAutomation|TestRunDevSmoke|TestRunCreateIssues|TestRunIssue|TestRunPanel|TestRun(GitHubSyncHelp|WorkspaceHelp|CreateIssuesHelp|DevSmokeHelp)'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	0.467s`
- [x] `cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadow|TestLiveValidation(Index|Summary)|TestParallelValidationMatrixDocsStayAligned|TestCrossProcessCoordinationDocsStayAligned|TestTakeoverProofSurfaceStaysAligned|TestSharedQueueReportStaysAligned'`
  Result: `ok  	bigclaw-go/internal/regression	(cached)`
- [x] `cd bigclaw-go && bash -n ./scripts/e2e/run_all.sh && bash -n ./scripts/e2e/kubernetes_smoke.sh && bash -n ./scripts/e2e/ray_smoke.sh && bash -n ./scripts/benchmark/run_suite.sh`
  Result: success
- [x] `cd bigclaw-go && go run ./cmd/bigclawctl automation --help >/dev/null && go run ./cmd/bigclawctl automation e2e run-task-smoke --help >/dev/null && go run ./cmd/bigclawctl automation migration shadow-compare --help >/dev/null && go run ./cmd/bigclawctl automation benchmark run-matrix --help >/dev/null && go run ./cmd/bigclawctl dev-smoke --json`
  Result: success; `dev-smoke --json` returned `{"accepted":true,"executor":"local","reason":"default local executor for low/medium risk","status":"ok","task_id":"SMOKE-1"}`

### Notes

- Scope for `BIG-GO-26`: support tooling, fixtures, and non-core residual Python still physically present outside the already-frozen legacy core modules in `src/bigclaw`.
- The workspace started with an invalid `HEAD` and was recovered locally by rebasing the worktree onto the cached mirror `../.symphony/bigclaw-mirror.git` and creating branch `BIG-GO-26` from `origin/main`.
- The repository had broad unrelated dirty-worktree drift outside this issue slice. Only the staged support-tooling retirement files should be included in the final commit.
