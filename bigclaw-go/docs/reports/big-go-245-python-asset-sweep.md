# BIG-GO-245 Python Asset Sweep

## Scope

`BIG-GO-245` (`Residual tooling Python sweep T`) tightens the active
tooling/build-helper/dev-utility docs that still carried Python-first helper
names after the repo itself had already gone physically Python-free.

This checkout already has a repository-wide Python file count of `0`, so the
lane lands as documentation cleanup, regression prevention, and evidence capture
rather than an in-branch Python-file deletion batch.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `README.md`: active helper guidance refreshed to stay Go-and-shell only
- `docs/go-cli-script-migration-plan.md`: root helper migration guidance now uses Go-first helper labels
- `bigclaw-go/docs/go-cli-script-migration.md`: automation migration guidance now uses Go-only helper labels

Explicit remaining Python asset list: none.

## Go Or Native Replacement Paths

The active Go/native replacement surface covering this sweep remains:

- `README.md`
- `docs/go-cli-script-migration-plan.md`
- `bigclaw-go/docs/go-cli-script-migration.md`
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `bigclaw-go/cmd/bigclawctl/main.go`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `rg -n "scripts/create_issues\\.py|scripts/dev_smoke\\.py|scripts/ops/bigclaw_github_sync\\.py|scripts/ops/bigclaw_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_validate\\.py|Python-free operator surface|Python-side tests|## Python asset status" README.md docs/go-cli-script-migration-plan.md bigclaw-go/docs/go-cli-script-migration.md`
  Result: no output; active tooling docs no longer carry the retired Python-first helper names checked by this lane.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO245(RepositoryHasNoPythonFiles|ToolingDocsStayGoOnly|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: see `reports/BIG-GO-245-validation.md` for the exact test output recorded for this lane.
