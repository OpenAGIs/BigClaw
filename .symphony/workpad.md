# BIG-GO-1018

## Plan
- Remove the residual Python tranche under `tests/` for console IA, reports, operations, UI review, design system, and the Python-only `conftest.py`.
- Add a repo-native Go regression test that asserts this tranche stays deleted and documents the corresponding Go coverage packages already present in the repository.
- Keep scope limited to this cleanup tranche; do not modify unrelated packages or tracker state.
- Run targeted Go regression validation plus repository file-count checks, then commit and push the branch.

## Acceptance
- Changes stay scoped to the residual `tests/**` tranche for this issue.
- Repository `.py` asset count decreases by deleting the selected test tranche directly from the working tree.
- A Go regression test prevents the deleted Python tranche from reappearing unnoticed.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `cd bigclaw-go && go test ./internal/regression -run TestResidualPythonTestsTranche3StaysRemoved`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`

## Results
- `cd bigclaw-go && go test ./internal/regression -run TestResidualPythonTestsTranche3StaysRemoved` -> `ok  	bigclaw-go/internal/regression	1.936s`
- `cd bigclaw-go && go test ./internal/planningsurface` -> `ok  	bigclaw-go/internal/planningsurface	1.305s`
- `cd bigclaw-go && go test ./internal/product ./internal/reporting` -> `ok  	bigclaw-go/internal/product	(cached)`; `ok  	bigclaw-go/internal/reporting	(cached)`
- `find . -name '*.py' | wc -l` -> `68`
- `find . -name '*.go' | wc -l` -> `284`
- `rg --files | rg '(^|/)pyproject\.toml$|(^|/)setup\.py$'` -> no matches; `pyproject.toml` absent and unchanged; `setup.py` absent and unchanged
- `git status --short` -> `.symphony/workpad.md` modified; `src/bigclaw/planning.py` modified; `bigclaw-go/internal/planningsurface/planningsurface.go` and `bigclaw-go/internal/planningsurface/planningsurface_test.go` modified; `bigclaw-go/internal/regression/residual_python_tests_tranche3_test.go` added; `tests/conftest.py`, `tests/test_console_ia.py`, `tests/test_design_system.py`, `tests/test_operations.py`, `tests/test_reports.py`, and `tests/test_ui_review.py` deleted
- Impact: `py files` decreased from `74` to `68`; `go files` increased from `283` to `284`; `pyproject.toml` absent and unchanged; `setup.py` absent and unchanged
