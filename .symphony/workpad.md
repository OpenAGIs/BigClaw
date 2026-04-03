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

### Validation Results
- `cd bigclaw-go && go test ./internal/reporting ./internal/pilot ./internal/runtimecompat ./internal/scheduler`
  `ok   bigclaw-go/internal/reporting (cached)`
  `ok   bigclaw-go/internal/pilot 0.613s`
  `ok   bigclaw-go/internal/runtimecompat (cached)`
  `ok   bigclaw-go/internal/scheduler (cached)`
- `cd bigclaw-go && go test ./internal/queue ./internal/product ./internal/worker ./internal/workflow`
  `ok   bigclaw-go/internal/queue (cached)`
  `ok   bigclaw-go/internal/product (cached)`
  `ok   bigclaw-go/internal/worker 1.265s`
  `ok   bigclaw-go/internal/workflow (cached)`
- `find . -type f | sed 's#^./##' | awk 'BEGIN{py=0;go=0} /\\.py$/{py++} /\\.go$/{go++} END{print "py="py" go="go}'`
  `py=50 go=288`
- `rg -n "tests/test_reports.py" README.md docs src scripts tests bigclaw-go --glob '!reports/**'`
  no matches
- `git diff --name-only -- pyproject.toml setup.py setup.cfg '**/pyproject.toml' '**/setup.py' '**/setup.cfg'`
  no output; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were added or modified
- `git push origin BIG-GO-1026`
  pushed commit `aa8a088d8000b1dc64b43b8ba1eed59e8363a31d` to `origin/BIG-GO-1026`
