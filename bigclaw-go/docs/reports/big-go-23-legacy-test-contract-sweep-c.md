# BIG-GO-23 Legacy Test Contract Sweep C

## Scope

`BIG-GO-23` closes the remaining completed legacy Python test-contract slice
from `reports/BIG-GO-948-validation.md`: `tests/test_issue_archive.py` and
`tests/test_pilot.py`.

Repository-wide Python file count: `0`.

The checked-out workspace is already physically Python-free, so this lane does
not delete a live `.py` file. It records the Go-native replacement ownership
for the retired test contracts and adds a focused regression guard that keeps
that mapping explicit.

## Replacement Registry

- Registry source: `bigclaw-go/internal/migration/legacy_test_contract_sweep_c.go`
- Regression guard: `bigclaw-go/internal/regression/big_go_23_legacy_test_contract_sweep_c_test.go`

## Retired Test Contracts

- `tests/test_issue_archive.py`
  - Go replacement kind: `go-issue-priority-archive-surface`
  - Go replacement paths:
    - `bigclaw-go/internal/issuearchive/archive.go`
    - `bigclaw-go/internal/issuearchive/archive_test.go`
  - Supporting evidence:
    - `bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md`
    - `reports/BIG-GO-948-validation.md`
  - Status: retired Python issue archive coverage is replaced by the Go
    issue-priority archive manifest, audit, and reporting surface.

- `tests/test_pilot.py`
  - Go replacement kind: `go-pilot-readiness-surface`
  - Go replacement paths:
    - `bigclaw-go/internal/pilot/report.go`
    - `bigclaw-go/internal/pilot/report_test.go`
  - Supporting evidence:
    - `bigclaw-go/docs/reports/big-go-1597-python-asset-sweep.md`
    - `reports/BIG-GO-948-validation.md`
  - Status: retired Python pilot coverage is replaced by the Go pilot
    readiness report and implementation-result surface.

## Validation Commands

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `for path in tests/test_issue_archive.py tests/test_pilot.py bigclaw-go/internal/issuearchive/archive.go bigclaw-go/internal/issuearchive/archive_test.go bigclaw-go/internal/pilot/report.go bigclaw-go/internal/pilot/report_test.go bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md bigclaw-go/docs/reports/big-go-1597-python-asset-sweep.md reports/BIG-GO-948-validation.md; do if test -e "$path"; then printf 'present %s\n' "$path"; else printf 'absent %s\n' "$path"; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO23LegacyTestContractSweepC(ManifestMatchesDeferredLegacyTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`

## Expected Results

- Repository-wide Python inventory remains empty.
- Retired Python tests stay absent.
- Replacement and evidence paths stay present.
- The focused regression guard passes and preserves the batch C mapping.
