# BIG-GO-163 Legacy Test Contract Sweep X

## Scope

`BIG-GO-163` closes the next retired legacy Python test-contract slice that did
not yet have its own Go/native replacement manifest:
`tests/test_audit_events.py`, `tests/test_connectors.py`,
`tests/test_console_ia.py`, and `tests/test_dashboard_run_contract.py`.

## Python Baseline

Repository-wide Python file count: `0`.

This checkout therefore lands as a native-replacement sweep rather than a direct
Python-file deletion batch because there are no physical `.py` assets left to
remove in-branch.

## Deferred Legacy Test Replacements

The sweep-X Go/native replacement registry lives in
`bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go`.

- `tests/test_audit_events.py`
  - Go replacements:
    - `bigclaw-go/internal/observability/audit_spec.go`
    - `bigclaw-go/internal/observability/audit.go`
    - `bigclaw-go/internal/observability/audit_spec_test.go`
  - Evidence:
    - `bigclaw-go/internal/observability/audit_test.go`
    - `docs/go-mainline-cutover-issue-pack.md`
    - `reports/OPE-134-validation.md`
- `tests/test_connectors.py`
  - Go replacements:
    - `bigclaw-go/internal/intake/connector.go`
    - `bigclaw-go/internal/intake/types.go`
    - `bigclaw-go/internal/api/v2.go`
  - Evidence:
    - `bigclaw-go/internal/intake/connector_test.go`
    - `docs/go-domain-intake-parity-matrix.md`
    - `docs/go-mainline-cutover-issue-pack.md`
- `tests/test_console_ia.py`
  - Go replacements:
    - `bigclaw-go/internal/consoleia/consoleia.go`
    - `bigclaw-go/internal/product/console.go`
    - `bigclaw-go/internal/designsystem/designsystem.go`
  - Evidence:
    - `bigclaw-go/internal/consoleia/consoleia_test.go`
    - `bigclaw-go/internal/product/console_test.go`
    - `reports/OPE-127-validation.md`
- `tests/test_dashboard_run_contract.py`
  - Go replacements:
    - `bigclaw-go/internal/product/dashboard_run_contract.go`
    - `bigclaw-go/internal/contract/execution.go`
    - `bigclaw-go/internal/api/server.go`
  - Evidence:
    - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
    - `bigclaw-go/internal/contract/execution_test.go`
    - `reports/OPE-129-validation.md`

## Why This Sweep Is Now Safe

These retired Python tests already have stable Go-native owners in the active
repo tree:

- audit-event validation is owned by the Go audit-event specification and
  observability validation surface.
- connector behavior is owned by the Go intake connector package and the v2
  intake API surface.
- console IA behavior is owned by the Go console IA contract together with the
  product console and design-system surfaces.
- dashboard/run contract behavior is owned by the Go dashboard/run contract and
  execution contract surfaces.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO163LegacyTestContractSweepX(ManifestMatchesDeferredLegacyTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.193s`
