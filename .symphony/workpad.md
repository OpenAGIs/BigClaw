# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python design-system test file with Go coverage.
- Keep the scope limited to a new Go UI-governance package under `bigclaw-go/internal/ui`.
- Add Go models, audits, route helpers, and report renderers needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_design_system.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/ui`.
- Replacement coverage explicitly includes:
  - design-system component/token manifests, audits, and report rendering
  - console top-bar manifest, audit checks, and report rendering
  - information-architecture route resolution, audits, and report rendering
  - UI acceptance suite audits and report rendering
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/ui/governance.go bigclaw-go/internal/ui/governance_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/ui`
  - Result: `ok  	bigclaw-go/internal/ui	0.425s`

## Completed
- Added `bigclaw-go/internal/ui/governance.go` with UI-governance manifests, design-system audits, console top-bar audits, information-architecture helpers, UI acceptance audits, and markdown report renderers.
- Added `bigclaw-go/internal/ui/governance_test.go` covering JSON round trips, audit findings, route resolution, and report rendering parity for the replaced Python tests.
- Deleted `tests/test_design_system.py`.
