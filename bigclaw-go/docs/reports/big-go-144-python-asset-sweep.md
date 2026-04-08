# BIG-GO-144 Python Asset Sweep

## Scope

This sweep covers residual Python scripts, wrappers, and CLI helpers that have
already been retired from the current branch baseline:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/benchmark/run_matrix.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

## Sweep Result

- The residual Python wrapper/helper paths in this issue remain absent.
- The supported operator surface is the retained shell-wrapper layer plus the
  Go-owned `bigclawctl` commands behind it.
- This lane hardens the replacement contract for repo-root wrappers and
  `bigclaw-go/scripts/*` entrypoints without widening scope into unrelated
  library or fixture cleanup.

## Go Or Native Replacement Paths

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-symphony`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
- `bigclaw-go/scripts/e2e/ray_smoke.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/automation_commands.go`

## Validation Commands And Results

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-144/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-144/bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the scoped script-wrapper directories contain no Python files.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-144/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO144(ResidualPythonWrappersAndHelpersStayAbsent|GoWrapperAndCLIReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	3.203s`

## Residual Risk

- The retained shell wrappers still depend on `go run`, so local toolchain
  availability and first-run startup cost remain part of the operator path.
- Follow-up cleanup can still collapse more convenience wrappers into direct
  `bigclawctl` binary usage, but that is outside this issue scope.
