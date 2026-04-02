# BIG-GO-1036 Workpad

## Plan
- Replace the next scoped Python console-IA test file with Go coverage.
- Keep the scope limited to the existing `bigclaw-go/internal/ui` package.
- Add Go console information-architecture models, audits, interaction-contract audits, draft builder, and report renderers needed to remove the Python file.
- Delete the matched Python test file once Go parity is explicit.
- Run targeted Go tests, record exact commands and exact results here.
- Commit and push the branch.

## Scoped Tranche
- `tests/test_console_ia.py`

## Acceptance
- Python test file count decreases by deleting the scoped file above.
- Go test coverage increases under `bigclaw-go/internal/ui`.
- Replacement coverage explicitly includes:
  - console IA manifests, audits, and report rendering
  - console interaction draft manifests, audits, required-role coverage, and report rendering
  - `build_big_4203_console_interaction_draft` equivalent release-ready draft coverage
- Changes remain scoped to this tranche only.

## Validation
- `gofmt -w bigclaw-go/internal/ui/console_ia.go bigclaw-go/internal/ui/console_ia_test.go`
  - Result: exit 0
- `cd bigclaw-go && go test ./internal/ui`
  - Result: `ok  	bigclaw-go/internal/ui	0.451s`

## Completed
- Added `bigclaw-go/internal/ui/console_ia.go` with console IA manifests, audits, interaction-contract audits, release-ready draft builder, and report renderers.
- Added `bigclaw-go/internal/ui/console_ia_test.go` covering JSON round trips, audit findings, required-role coverage, frame-contract validation, and report rendering parity for the replaced Python tests.
- Deleted `tests/test_console_ia.py`.
