# BIG-GO-1027 Workpad

## Plan
- Inspect remaining Python tests and identify a narrow tranche with existing Go coverage.
- Remove only the selected residual Python test files from `tests/`.
- Remove `tests/conftest.py` by inlining its bootstrap into the remaining Python test modules.
- Validate the migrated coverage with targeted `go test` runs in the corresponding Go packages.
- Validate the remaining Python test suite directly after deleting the shared bootstrap helper.
- Record repo impact, including `.py`/`.go` file counts and whether any `pyproject`/`setup` files changed.
- Commit and push the scoped change set to `origin/big-go-1027`.

## Acceptance
- Changes are limited to the remaining repository-level Python test assets for this tranche.
- `.py` file count decreases.
- Go coverage remains in place for the removed Python test behaviors.
- Final report includes `.py`/`.go` count impact and `pyproject`/`setup` impact.
- Tracker state is not used as a substitute for repository changes.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1027/bigclaw-go && go test ./internal/workflow ./internal/observability`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1027/bigclaw-go && go test ./internal/scheduler ./internal/product ./internal/regression`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1027 && python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_evaluation.py tests/test_operations.py tests/test_planning.py tests/test_reports.py tests/test_ui_review.py -q`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1027 && find . -type f \( -name '*.py' -o -name '*.go' \) | sed 's#^\./##' | awk 'BEGIN{py=0;go=0} /\.py$/{py++} /\.go$/{go++} END{printf("py=%d\ngo=%d\n",py,go)}'`
