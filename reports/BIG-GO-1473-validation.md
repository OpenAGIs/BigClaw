# BIG-GO-1473 Validation

## Outcome

`BIG-GO-1473` audited the remaining repo-level Python script migration surface and
confirmed this checkout is already at a zero-Python baseline. There were no
physical `scripts/**/*.py` or `bigclaw-go/scripts/**/*.py` assets left to
delete, so the blocker for a literal file-count reduction is repository state,
not missing implementation work in this branch.

## Migrated Or Deleted Python Paths And Replacements

The deleted Python entrypoints covered by this lane's migration audit remain
absent, and their active ownership stays on Go or shell wrappers around Go:

- `scripts/create_issues.py` -> `bigclaw-go/cmd/bigclawctl` `create-issues`
- `scripts/dev_smoke.py` -> `bigclaw-go/cmd/bigclawctl` `dev-smoke`
- `scripts/ops/bigclaw_github_sync.py` -> `bigclaw-go/cmd/bigclawctl` `github-sync`
- `scripts/ops/bigclaw_refill_queue.py` -> `bigclaw-go/internal/refill/queue.go` via `bigclawctl refill`
- `scripts/ops/bigclaw_workspace_bootstrap.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go` via `bash scripts/ops/bigclawctl workspace bootstrap`
- `scripts/ops/symphony_workspace_bootstrap.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go` via `bash scripts/ops/bigclawctl workspace bootstrap`
- `scripts/ops/symphony_workspace_validate.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go` via `bash scripts/ops/bigclawctl workspace validate`
- `bigclaw-go/scripts/benchmark/capacity_certification.py` -> `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go` `automation benchmark capacity-certification`
- `bigclaw-go/scripts/benchmark/run_matrix.py` -> `bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go` `automation benchmark run-matrix`
- `bigclaw-go/scripts/benchmark/soak_local.py` -> `bigclaw-go/cmd/bigclawctl/automation_commands.go` `automation benchmark soak-local`
- `bigclaw-go/scripts/e2e/run_task_smoke.py` -> `bigclaw-go/cmd/bigclawctl/automation_commands.go` `automation e2e run-task-smoke`
- `bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclaw-go/cmd/bigclawctl/automation_commands.go` `automation migration shadow-compare`

Delete condition for all listed Python paths: keep them absent. Reintroducing a
tracked `.py` file under the repository root, `scripts`, or `bigclaw-go/scripts`
would regress the Go-only migration state verified here.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide physical `.py` count is `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the priority migration directories remain Python-free.
- `rg -n "scripts/create_issues\\.py|scripts/dev_smoke\\.py|scripts/ops/bigclaw_github_sync\\.py|scripts/ops/bigclaw_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_validate\\.py|bigclaw-go/scripts/.+\\.py" docs/go-cli-script-migration-plan.md workflow.md bigclaw-go/internal/regression`
  Result: only migration documentation and regression guards reference the retired Python paths; active workflow callers point at `bash scripts/ops/bigclawctl ...` or Go commands.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1473ZeroPythonBaselineAndReplacementOwnership|BIGGO1473ValidationReportCapturesBlockedPhysicalDeletionState)$'`
  Result: `ok  	bigclaw-go/internal/regression`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationBenchmarkDirectoryContainsNoPythonHelpers|TestRunHelpMentionsCoreCommands'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl`

## Blocker

The issue title assumes there are remaining physical Python entrypoints to
delete, but this branch baseline already reached zero tracked `.py` files
before `BIG-GO-1473` started. This lane therefore records the exact migrated
ownership and locks the zero-Python baseline with regression coverage, but it
cannot honestly claim a fresh file-count reduction in this checkout.
