# BIG-GO-1468 Python Reference Sweep

## Summary

`BIG-GO-1468` cleared the remaining active docs/examples/manifests that still
advertised deleted Python assets as current repo requirements or live
validation commands.

Repository-wide Python file count: `0`.

This lane did not delete new physical Python files because the checked-out
baseline was already Go-only. Instead, it removed the residual references that
could mislead future cleanup work or encourage reintroduction of deleted paths.

## Exact Files Updated

- `docs/symphony-repo-bootstrap-template.md`
  - removed the template requirement for `src/<your_package>/workspace_bootstrap.py`
  - removed the template requirement for `src/<your_package>/workspace_bootstrap_cli.py`
  - replaced those examples with `scripts/ops/bigclawctl`, `workflow.md`, and a Go/native bootstrap note
- `docs/go-cli-script-migration-plan.md`
  - removed the long manifest-style list of deleted `bigclaw-go/scripts/**/*.py` helper paths
  - kept the Go replacement surface explicit via `bigclawctl automation ...`
- `docs/go-mainline-cutover-handoff.md`
  - removed the stale `PYTHONPATH=src python3 ...` validation step
  - replaced it with zero-Python inventory evidence plus the Go regression guard

## Retired Path To Replacement Map

- `src/<your_package>/workspace_bootstrap.py`
  - explicit delete condition: no repo-local Python compatibility module is required to satisfy the bootstrap template
  - replacement: `scripts/ops/bigclawctl` plus the repo's Go/native `bigclawctl workspace ...` implementation
- `src/<your_package>/workspace_bootstrap_cli.py`
  - explicit delete condition: bootstrap CLI ownership stays behind `bigclawctl workspace ...`
  - replacement: `scripts/ops/bigclawctl` and `workflow.md`
- deleted `bigclaw-go/scripts/benchmark/*.py`, `bigclaw-go/scripts/e2e/*.py`, and `bigclaw-go/scripts/migration/*.py` manifest references
  - explicit delete condition: docs should summarize the retired sweep areas instead of listing deleted file-by-file manifests
  - replacement: `bigclawctl automation ...`
- `PYTHONPATH=src python3 - <<"... legacy shim assertions ..."`
  - explicit delete condition: validation must reflect the Go-only checkout that no longer carries Python runtime shims
  - replacement: `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name 'requirements*.txt' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'Pipfile' -o -name 'Pipfile.lock' \) -print | sort`
  - replacement: `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1454RepositoryHasNoPythonFiles$'`

## Validation

- Inventory:
  - `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name 'requirements*.txt' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'Pipfile' -o -name 'Pipfile.lock' \) -print | sort`
- Residual reference scan:
  - `rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!*.json' "src/bigclaw/[A-Za-z0-9_/-]+\\.py|scripts/[A-Za-z0-9_./-]+\\.py|bigclaw-go/scripts/[A-Za-z0-9_./-]+\\.py|python3? -|PYTHONPATH=src python3|pytest\\b|requirements\\.txt|pyproject\\.toml" docs README.md workflow.md .github scripts bigclaw-go`
- Targeted regression:
  - `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1160ScriptMigrationDocsListGoReplacements|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1468ActiveDocsStayGoFirst|BIGGO1468MigrationPlanAvoidsDeletedPythonManifests|BIGGO1468LaneReportCapturesDocReferenceSweep)$'`
