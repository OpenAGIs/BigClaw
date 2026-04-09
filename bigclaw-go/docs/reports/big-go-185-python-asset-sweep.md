# BIG-GO-185 Python Asset Sweep

## Scope

`BIG-GO-185` records the retired root tooling/build-helper/dev-utility Python
surface that used to live under the repository root `scripts/` and `scripts/ops/`
helpers plus the root Python build metadata.

Representative retired paths pinned by this lane:

- `setup.py`
- `pyproject.toml`
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

## Python Baseline

Repository-wide Python file count: `0`.

Focused tooling directory state:

- `scripts`: `0` Python files
- `scripts/ops`: `0` Python files
- `bigclaw-go/cmd/bigclawctl`: `0` Python files and active Go CLI command coverage
- `bigclaw-go/internal/{githubsync,refill,bootstrap}`: `0` Python files

This checkout therefore lands as a regression-hardening sweep rather than a
fresh deletion batch because the targeted Python tooling assets are already
absent from the branch baseline.

## Go Or Native Replacement Paths

The active repo-native replacement surface for this retired tooling bucket
remains:

- `bigclaw-go/go.mod`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`
- `bigclaw-go/internal/githubsync/sync.go`
- `bigclaw-go/internal/refill/queue.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`
- `scripts/ops/bigclawctl`
- `scripts/dev_bootstrap.sh`

These paths keep the migrated workflows in Go or shell:
`bigclawctl create-issues`, `bigclawctl dev-smoke`,
`bigclawctl github-sync`, `bigclawctl refill`,
`bash scripts/ops/bigclawctl workspace bootstrap`, and
`bash scripts/ops/bigclawctl workspace validate`.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find scripts bigclaw-go -type f \( -name '*.py' -o -path 'bigclaw-go/go.mod' -o -path 'bigclaw-go/cmd/bigclawctl/main.go' -o -path 'bigclaw-go/cmd/bigclawctl/migration_commands.go' -o -path 'bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go' -o -path 'bigclaw-go/internal/githubsync/sync.go' -o -path 'bigclaw-go/internal/refill/queue.go' -o -path 'bigclaw-go/internal/bootstrap/bootstrap.go' -o -path 'scripts/ops/bigclawctl' -o -path 'scripts/dev_bootstrap.sh' \) 2>/dev/null | sort`
  Result: only the active replacement paths were listed:
  `bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`,
  `bigclaw-go/cmd/bigclawctl/main.go`,
  `bigclaw-go/cmd/bigclawctl/migration_commands.go`,
  `bigclaw-go/go.mod`,
  `bigclaw-go/internal/bootstrap/bootstrap.go`,
  `bigclaw-go/internal/githubsync/sync.go`,
  `bigclaw-go/internal/refill/queue.go`,
  `scripts/dev_bootstrap.sh`, and
  `scripts/ops/bigclawctl`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO185(ResidualPythonToolingPathsStayAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.187s`
