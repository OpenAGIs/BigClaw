# BIG-GO-1483 Python Asset Sweep

## Scope

Go-only refill lane `BIG-GO-1483` removes the remaining checked-in caller
references to retired Python entrypoints under `bigclaw-go/scripts` and records
the matching zero-Python baseline for the directory itself.

## Before And After Evidence

- Before update physical Python files under `bigclaw-go/scripts`: `0`.
- After update physical Python files under `bigclaw-go/scripts`: `0`.
- Before update checked-in caller references to retired `bigclaw-go/scripts` Python paths: `23`.
- After update checked-in caller references to retired `bigclaw-go/scripts` Python paths: `0`.

The before-reference count comes from baseline commit
`a63c8ec0f999d976a1af890c920a54ac2d6c693a`, where
`docs/go-cli-script-migration-plan.md` still enumerated the deleted
`bigclaw-go/scripts/*.py` sweep candidates. This lane replaces that stale list
with the active Go CLI and retained wrapper surface.

## Active Replacement Surface

- `bigclawctl automation e2e ...`
- `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`
- `bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`
- `benchmark/run_suite.sh`
- `e2e/run_all.sh`
- `e2e/kubernetes_smoke.sh`
- `e2e/ray_smoke.sh`
- `e2e/broker_bootstrap_summary.go`

## Validation Commands And Results

- `git show a63c8ec0f999d976a1af890c920a54ac2d6c693a:docs/go-cli-script-migration-plan.md | rg -n 'bigclaw-go/scripts/.*\.py' | wc -l | tr -d ' '`
  Result: `23`
- `find bigclaw-go/scripts -type f -name '*.py' | wc -l | tr -d ' '`
  Result: `0`
- `rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\\.py' README.md docs scripts .github bigclaw-go | sort`
  Result: no output; active checked-in caller references remained `0`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'`
  Result: `ok  	bigclaw-go/internal/regression	3.449s`
