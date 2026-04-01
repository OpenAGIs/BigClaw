# BIG-GO-1039 Workpad

## Plan

1. Verify the current `bigclaw-go/scripts/**` inventory and confirm the Python helper surface is already gone on the branch baseline.
2. Add a Go-native regression test that fails if `bigclaw-go/scripts/**` gains `.py` files again or if Python packaging files (`pyproject.toml`, `setup.py`) reappear in the migrated scope.
3. Refresh the script migration document so it reflects the fully migrated state instead of describing remaining Python helper follow-ups.
4. Run targeted validation for the touched regression package and exact filesystem checks, then record commands and results.
5. Commit the scoped ticket changes and push the issue branch to origin.

## Acceptance

- `bigclaw-go/scripts/**` contains no Python helper files.
- Go coverage increases via a regression test that enforces the no-Python state for the migrated script surface.
- `pyproject.toml` and `setup.py` remain absent in the repository scope enforced by the migration guard.
- The final change can explicitly name the Go files added or updated for this ticket.

## Validation

- `find bigclaw-go/scripts -type f | sort`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results

- `find bigclaw-go/scripts -type f | sort`
  - `bigclaw-go/scripts/benchmark/run_suite.sh`
  - `bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`
  - `bigclaw-go/scripts/e2e/kubernetes_smoke.sh`
  - `bigclaw-go/scripts/e2e/ray_smoke.sh`
  - `bigclaw-go/scripts/e2e/run_all.sh`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
- `cd bigclaw-go && go test ./internal/regression`
  - `ok  	bigclaw-go/internal/regression	0.516s`
- `git status --short`
  - ` M .symphony/workpad.md`
  - ` M bigclaw-go/docs/go-cli-script-migration.md`
  - `?? bigclaw-go/internal/regression/script_migration_guard_test.go`
