# BIG-GO-1466

## Plan

1. Inventory the remaining Python-linked reporting and observability surfaces in the repository.
2. Remove or replace the in-scope Ray reporting helper remnants while keeping the change scoped to checked-in Go-owned artifacts.
3. Add issue-specific regression coverage and a lane report documenting the exact files updated or deleted.
4. Run targeted validation proving the repo remains at zero physical `.py` files and no longer ships the stale Python helper surfaces.
5. Commit and push `BIG-GO-1466`.

## Acceptance

- Repository remains at zero physical `.py` files.
- Remaining Python-linked reporting or observability helper surfaces in scope are removed or replaced with shell/Go-native equivalents.
- Exact files updated or deleted are documented.
- Targeted tests and sweep commands are recorded with exact commands and outcomes.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `rg -n "python -c|import ray; ray.init" bigclaw-go --glob '!**/.git/**' --glob '!bigclaw-go/docs/reports/big-go-1466-python-reporting-observability-sweep.md' --glob '!bigclaw-go/internal/regression/big_go_1466_reporting_surface_guard_test.go' --glob '!bigclaw-go/internal/regression/big_go_1359_zero_python_guard_test.go'`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression -run 'TestBIGGO1466|TestLiveValidation|TestParallelValidationMatrixDocs'`
