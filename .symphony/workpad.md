# BIG-GO-1038 Workpad

## Plan

1. Keep this tranche scoped to porting the simpler remaining `tests/test_reports.py` surfaces into
   `bigclaw-go/internal/reporting` instead of widening into the UI-review module.
2. Add Go-native report studio, issue validation, launch/final-delivery checklist, pilot
   scorecard/portfolio, and shared-view rendering helpers, plus any small collaboration support
   needed to render those surfaces cleanly.
3. Add targeted Go tests for the newly ported reporting behavior, reusing the existing reporting
   package rather than introducing a new package layer.
4. Run targeted Go validation for `./internal/reporting` and any touched supporting package plus
   repo-level file-count and Python packaging checks, then record exact commands and results here.
4. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The Go tree gains native coverage for the report-studio/checklist/pilot/shared-view portion of
  `tests/test_reports.py`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/reporting ./internal/collaboration`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/collaboration ./internal/reporting`
  - `ok  	bigclaw-go/internal/collaboration	0.433s`
  - `ok  	bigclaw-go/internal/reporting	0.805s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `2`
- `find . \( -name pyproject.toml -o -name setup.py \) -print | sort`
  - no output
- `git status --short`
  - ` M .symphony/workpad.md`
  - ` M bigclaw-go/internal/collaboration/thread.go`
  - ` M bigclaw-go/internal/collaboration/thread_test.go`
  - ` M bigclaw-go/internal/reporting/reporting.go`
  - `?? bigclaw-go/internal/reporting/reporting_surface.go`
  - `?? bigclaw-go/internal/reporting/reporting_surface_test.go`
  - ` M docs/BigClaw-AgentHub-Integration-Alignment.md`
