# BIG-GO-1098 Validation

## Scope

Replaced residual deleted Python test references with Go-native validation paths across the remaining active planning and bootstrap surfaces:

- `bigclaw-go/internal/planning/planning.go`
- `src/bigclaw/planning.py`
- `scripts/dev_bootstrap.sh`
- `README.md`
- `docs/BigClaw-AgentHub-Integration-Alignment.md`
- `docs/go-cli-script-migration-plan.md`
- `bigclaw-go/internal/regression/planning_python_test_replacement_test.go`

## Go Replacement Evidence

- v3 candidate backlog validation now points at:
  - `go test ./internal/designsystem ./internal/consoleia ./internal/uireview`
  - `go test ./internal/contract ./internal/evaluation ./internal/product ./internal/worker ./internal/workflow ./internal/scheduler`
  - `go test ./internal/collaboration ./internal/pilot ./internal/reportstudio`
- `BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh` now runs:
  - `go test ./internal/bootstrap ./internal/planning ./internal/regression`
- regression coverage now scans repo-managed active surfaces so removed Python test commands cannot reappear outside archival fixtures.

## Validation Commands

1. `cd bigclaw-go && go test ./internal/planning ./internal/regression`
   - Result:
     - `ok  	bigclaw-go/internal/planning	0.941s`
     - `ok  	bigclaw-go/internal/regression	2.392s`
2. `cd bigclaw-go && go test ./internal/bootstrap ./internal/planning ./internal/regression`
   - Result:
     - `ok  	bigclaw-go/internal/bootstrap	3.196s`
     - `ok  	bigclaw-go/internal/planning	(cached)`
     - `ok  	bigclaw-go/internal/regression	(cached)`
3. `bash scripts/dev_bootstrap.sh`
   - Result:
     - `ok  	bigclaw-go/cmd/bigclawctl	3.453s`
     - `BigClaw Go development environment is ready.`
4. `BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh`
   - Result:
     - `ok  	bigclaw-go/cmd/bigclawctl	3.857s`
     - `smoke_ok local`
     - `ok  	bigclaw-go/internal/bootstrap	(cached)`
     - `ok  	bigclaw-go/internal/planning	(cached)`
     - `ok  	bigclaw-go/internal/regression	(cached)`
5. `cd bigclaw-go && go test ./internal/repo ./internal/collaboration ./internal/observability ./internal/reportstudio ./internal/governance ./internal/triage ./internal/service ./internal/product ./internal/legacyshim ./internal/regression`
   - Result:
     - `ok  	bigclaw-go/internal/repo	0.874s`
     - `ok  	bigclaw-go/internal/collaboration	1.323s`
     - `ok  	bigclaw-go/internal/observability	1.743s`
     - `ok  	bigclaw-go/internal/reportstudio	2.227s`
     - `ok  	bigclaw-go/internal/governance	2.687s`
     - `ok  	bigclaw-go/internal/triage	3.128s`
     - `ok  	bigclaw-go/internal/service	3.581s`
     - `ok  	bigclaw-go/internal/product	3.960s`
     - `ok  	bigclaw-go/internal/legacyshim	4.378s`
     - `ok  	bigclaw-go/internal/regression	(cached)`
6. `cd bigclaw-go && go test ./internal/regression`
   - Result:
     - `ok  	bigclaw-go/internal/regression	1.300s`
7. `rg -n "pytest|tests/test_.*\\.py" README.md docs scripts src bigclaw-go --glob '!bigclaw-go/internal/regression/**' --glob '!bigclaw-go/internal/planning/planning_test.go' --glob '!bigclaw-go/internal/workflow/**' --glob '!bigclaw-go/internal/policy/**' --glob '!bigclaw-go/internal/observability/**' --glob '!bigclaw-go/internal/events/**'`
   - Result: exit `1` with no matches

## Python Count Impact

- Baseline tree count before this slice (`261a43fe14a0f801f71d49ebe7be4a6d6f26d5ce`): `19`
- Tree count after this slice: `19`
- Net `.py` delta for this issue: `0`

The residual physical Python test files had already been removed before this branch. This slice replaced the remaining active commands, evidence links, bootstrap paths, and repo-managed documentation so the surviving Python compatibility files no longer point at deleted Python tests.
