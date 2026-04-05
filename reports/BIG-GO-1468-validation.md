# BIG-GO-1468 Validation

## Scope outcome

The checkout already had zero physical Python assets. This lane therefore
removed active docs/template/manifests references that still advertised deleted
Python files or Python validation commands as live guidance.

## Commands and results

1. Repository-wide Python asset inventory

```bash
find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name 'requirements*.txt' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'Pipfile' -o -name 'Pipfile.lock' \) -print | sort
```

Result: no output.

2. Residual reference scan

```bash
rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!*.json' "src/bigclaw/[A-Za-z0-9_/-]+\.py|scripts/[A-Za-z0-9_./-]+\.py|bigclaw-go/scripts/[A-Za-z0-9_./-]+\.py|python3? -|PYTHONPATH=src python3|pytest\b|requirements\.txt|pyproject\.toml" docs README.md workflow.md .github scripts bigclaw-go
```

Result: edited active docs no longer emit the removed bootstrap-template Python
requirements, the deleted `bigclaw-go/scripts/**/*.py` manifest list, or the
stale `PYTHONPATH=src python3` handoff command. Remaining matches come from
historical cutover docs and regression fixtures that intentionally preserve
retired-path lineage outside this lane's scope.

3. Targeted regression coverage

```bash
cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1160ScriptMigrationDocsListGoReplacements|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1468ActiveDocsStayGoFirst|BIGGO1468MigrationPlanAvoidsDeletedPythonManifests|BIGGO1468LaneReportCapturesDocReferenceSweep)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.903s
```
