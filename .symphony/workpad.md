# BIG-GO-1080 Workpad

## Plan
- Confirm the residual Python test tranche under `tests/` and map each file to the current Go-native replacement surface.
- Add focused Go coverage where the current replacement is still implicit, especially the `BIG-4204` UI review builder/report path and the release-planning entry that still references `tests/test_ui_review.py`.
- Remove the residual Python test files for this tranche: `tests/test_ui_review.py`, `tests/reports_legacy.py`, and `tests/conftest.py`.
- Update any default execution path or metadata that still points at the deleted Python files so the repo no longer advertises them as the primary validation route.
- Run targeted Go tests plus repo-level file-count checks, then commit and push the scoped branch.

## Acceptance
- `tests/test_ui_review.py`, `tests/reports_legacy.py`, and `tests/conftest.py` are deleted.
- Go-native tests cover the removed UI review and report-studio/reporting behaviors strongly enough that this slice is not a cosmetic deletion.
- No default validation command or candidate metadata continues to reference the deleted Python test files.
- Repository `.py` count decreases after the deletion.
- The change set stays scoped to this issue.

## Validation
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l`
- `find tests -maxdepth 1 -name '*.py' | sort`
- `cd bigclaw-go && go test ./internal/uireview ./internal/reportstudio ./internal/planning`
- `git status --short`

## Validation Results
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l` -> `23`
- `find tests -maxdepth 1 -name '*.py' | sort` -> no output
- `cd bigclaw-go && go test ./internal/uireview ./internal/reportstudio ./internal/planning ./internal/regression` -> first run failed in `internal/uireview` and `internal/regression` due to a sorted unresolved-question assertion mismatch and a bad regression root helper; reran after fixes and got `ok   bigclaw-go/internal/uireview 0.639s`, `ok   bigclaw-go/internal/reportstudio (cached)`, `ok   bigclaw-go/internal/planning (cached)`, `ok   bigclaw-go/internal/regression 0.974s`
- `git status --short` -> modified workpad, planning/uireview Go files, deleted `tests/conftest.py`, `tests/reports_legacy.py`, `tests/test_ui_review.py`, added `bigclaw-go/internal/regression/python_test_tranche14_removal_test.go`, plus the in-scope Go replacement file `bigclaw-go/internal/uireview/builder.go`
