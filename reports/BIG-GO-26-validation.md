# BIG-GO-26 Validation

- Identifier: `BIG-GO-26`
- Title: `Sweep support tooling and fixtures Python residuals`
- Branch: `origin/BIG-GO-26`
- Landed commits:
  - `3e87a8a0714f9fdc1174cdc7ad961edf6463ddf5` `BIG-GO-26: sweep support tooling python residuals`

## Completed scope

- Repointed repo-root support wrappers to shell and Symphony-owned entrypoints:
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
  - `scripts/ops/bigclaw-symphony`
- Made `scripts/dev_bootstrap.sh` Go-first by default, with legacy Python bootstrap opt-in via `BIGCLAW_ENABLE_LEGACY_PYTHON=1`.
- Updated `README.md` and `docs/go-mainline-cutover-issue-pack.md` so support tooling references no longer depend on retired Python wrappers.

## Validation

1. Residual support-tooling references

```bash
rg -n "scripts/ops/.*\\.py|scripts/e2e/.*\\.py|scripts/migration/.*\\.py|scripts/benchmark/.*\\.py|scripts/dev_smoke.py|scripts/create_issues.py|python3 scripts/e2e|python3 scripts/migration|python3 scripts/benchmark" README.md docs/go-mainline-cutover-issue-pack.md bigclaw-go bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go | head -400
```

Result:

- Only intentional retired-reference lines remain in `docs/go-mainline-cutover-issue-pack.md`.
- One future-looking note remains in `bigclaw-go/docs/reports/broker-failover-fault-injection-validation-pack.md`.

2. CLI coverage for support wrappers

```bash
cd bigclaw-go && go test ./cmd/bigclawctl -run 'TestAutomation|TestRunDevSmoke|TestRunCreateIssues|TestRunIssue|TestRunPanel|TestRun(GitHubSyncHelp|WorkspaceHelp|CreateIssuesHelp|DevSmokeHelp)'
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	0.467s
```

3. Regression coverage for aligned docs and validation surfaces

```bash
cd bigclaw-go && go test ./internal/regression -run 'TestLiveShadow|TestLiveValidation(Index|Summary)|TestParallelValidationMatrixDocsStayAligned|TestCrossProcessCoordinationDocsStayAligned|TestTakeoverProofSurfaceStaysAligned|TestSharedQueueReportStaysAligned'
```

Result:

```text
ok  	bigclaw-go/internal/regression	(cached)
```

4. Shell syntax checks for retained wrappers

```bash
cd bigclaw-go && bash -n ./scripts/e2e/run_all.sh && bash -n ./scripts/e2e/kubernetes_smoke.sh && bash -n ./scripts/e2e/ray_smoke.sh && bash -n ./scripts/benchmark/run_suite.sh
```

Result:

```text
success
```

5. Go-owned automation and smoke entrypoints

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation --help >/dev/null && go run ./cmd/bigclawctl automation e2e run-task-smoke --help >/dev/null && go run ./cmd/bigclawctl automation migration shadow-compare --help >/dev/null && go run ./cmd/bigclawctl automation benchmark run-matrix --help >/dev/null && go run ./cmd/bigclawctl dev-smoke --json
```

Result:

```json
{
  "accepted": true,
  "executor": "local",
  "reason": "default local executor for low/medium risk",
  "status": "ok",
  "task_id": "SMOKE-1"
}
```
