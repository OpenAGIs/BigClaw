## BIG-GO-1026

### Plan
- Inspect the remaining Python test tranche and map the live `tests/test_reports.py` references to repo-native Go coverage.
- Update active validation metadata, docs, and bootstrap scripts so they no longer reference `tests/test_reports.py`.
- Delete `tests/test_reports.py`, run targeted validation, recount `*.py`/`*.go`, then commit and push the branch.

### Acceptance
- Work stays scoped to the remaining Python test tranche for this issue.
- Repository `.py` file count decreases.
- Final report includes `.py`/`.go` count impact and confirms whether `pyproject.toml` or `setup*` files changed.
- Validation records exact commands and results.

### Validation
- Run `cd bigclaw-go && go test ./internal/reporting ./internal/pilot ./internal/runtimecompat ./internal/scheduler`.
- Run `cd bigclaw-go && go test ./internal/queue ./internal/product ./internal/worker ./internal/workflow`.
- Recount `*.py` and `*.go` files after edits.
- Verify no `pyproject.toml`, `setup.py`, or `setup.cfg` files were added or modified.
- Verify active repo surfaces no longer reference `tests/test_reports.py`.
