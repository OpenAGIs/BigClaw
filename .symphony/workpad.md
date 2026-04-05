# BIG-GO-1468 Workpad

## Plan

1. Reconfirm the repository-wide physical Python asset inventory so this lane only edits real residual blockers and not tracker-only bookkeeping.
2. Patch active docs/examples/manifests that still advertise deleted Python files as current repo assets, replacing them with Go-first ownership or explicit retired-path wording.
3. Record the migrated/deleted-path evidence for this lane, run targeted validation, then commit and push `BIG-GO-1468`.

## Acceptance

- The active checkout still reports zero physical Python assets, with exact inventory commands captured for this lane.
- Repo-native docs/examples/manifests no longer present deleted Python files as live required files or active validation commands when Go replacements already exist.
- The lane documents the exact retired Python paths touched and the Go/native replacements or explicit deletion conditions for each.
- Exact validation commands and observed results are recorded.
- The branch is committed and pushed to the remote `BIG-GO-1468` branch.

## Validation

- `find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name 'requirements*.txt' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'Pipfile' -o -name 'Pipfile.lock' \\) -print | sort`
- `rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!*.json' \"src/bigclaw/[A-Za-z0-9_/-]+\\.py|scripts/[A-Za-z0-9_./-]+\\.py|bigclaw-go/scripts/[A-Za-z0-9_./-]+\\.py|python3? -|PYTHONPATH=src python3|pytest\\b|requirements\\.txt|pyproject\\.toml\" docs README.md workflow.md .github scripts bigclaw-go`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1160ScriptMigrationDocsListGoReplacements|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1468ActiveDocsStayGoFirst|BIGGO1468MigrationPlanAvoidsDeletedPythonManifests|BIGGO1468LaneReportCapturesDocReferenceSweep)$'`

## Execution Notes

- 2026-04-06: Baseline inventory in `BIG-GO-1468` confirmed no physical `.py`, `.pyi`, `.pyw`, `requirements*.txt`, `pyproject.toml`, `setup.py`, `setup.cfg`, or `Pipfile*` assets remain in the checkout.
- 2026-04-06: This lane is therefore scoped to active docs/examples/manifests that still keep deleted Python paths visible enough to block final cleanup or mislead future operators.
- 2026-04-06: Updated `docs/symphony-repo-bootstrap-template.md`, `docs/go-cli-script-migration-plan.md`, and `docs/go-mainline-cutover-handoff.md` to remove stale Python-required examples and Python validation commands in favor of Go-first guidance.
- 2026-04-06: Added `bigclaw-go/docs/reports/big-go-1468-python-asset-sweep.md` and `bigclaw-go/internal/regression/big_go_1468_doc_reference_sweep_test.go` to document and guard the residual-reference cleanup.
- 2026-04-06: Ran `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name 'requirements*.txt' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'Pipfile' -o -name 'Pipfile.lock' \) -print | sort` and observed no output.
- 2026-04-06: Ran the residual reference scan and confirmed the edited active docs were cleaned up; the remaining hits are historical cutover docs and regression fixtures outside this lane's scope.
- 2026-04-06: Ran `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1160ScriptMigrationDocsListGoReplacements|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1468ActiveDocsStayGoFirst|BIGGO1468MigrationPlanAvoidsDeletedPythonManifests|BIGGO1468LaneReportCapturesDocReferenceSweep)$'` and observed `ok  	bigclaw-go/internal/regression	1.903s`.
