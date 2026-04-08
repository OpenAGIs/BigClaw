# BIG-GO-1582

## Plan
1. Treat bucket B as the second alphabetical slice of the remaining `src/bigclaw/*.py` surface, covering `collaboration.py`, `connectors.py`, `console_ia.py`, `cost_control.py`, `dashboard_run_contract.py`, `deprecation.py`, and `design_system.py`.
2. Delete the isolated Go-owned modules directly and replace the still-referenced Python modules by folding the minimal surviving definitions into existing non-bucket files.
3. Update package exports and imports so the remaining Python surface still compiles after the bucket-B files are removed.
4. Capture before/after Python file counts and run targeted validation against the touched Python package and the Go mainline test surface.
5. Commit the scoped change set and push `BIG-GO-1582` to `origin`.

## Acceptance
- The bucket-B `src/bigclaw/*.py` files are physically removed from the tree.
- The repository-wide `.py` count drops from the current baseline.
- Remaining in-repo imports and package exports resolve without referencing the deleted bucket-B modules.
- Validation commands and exact results are recorded here.
- The change lands as a commit pushed to `origin/BIG-GO-1582`.

## Validation
- `find . -path './.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort | wc -l | tr -d ' '`
- `find src/bigclaw -type f -name '*.py' | sed 's#^src/bigclaw/##' | sort`
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/mapping.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/repo_board.py src/bigclaw/service.py src/bigclaw/runtime.py src/bigclaw/queue.py src/bigclaw/scheduler.py src/bigclaw/workflow.py src/bigclaw/orchestration.py src/bigclaw/__main__.py scripts/dev_smoke.py`
- `cd bigclaw-go && go test ./...`
- `git status --short`

## Results
- Repository `.py` file count before deletion: `81`
- Repository `.py` file count after deletion: `74`
- Removed bucket-B file count: `7`
- Removed files:
  - `src/bigclaw/collaboration.py`
  - `src/bigclaw/connectors.py`
  - `src/bigclaw/console_ia.py`
  - `src/bigclaw/cost_control.py`
  - `src/bigclaw/dashboard_run_contract.py`
  - `src/bigclaw/deprecation.py`
  - `src/bigclaw/design_system.py`

## Command Log
- `find . -path './.git' -prune -o -type f -name '*.py' -print | sed 's#^./##' | sort | wc -l | tr -d ' '` -> `81` before changes, `74` after changes
- `find src/bigclaw -type f -name '*.py' | sed 's#^src/bigclaw/##' | sort` -> remaining `src/bigclaw` Python surface excludes the seven bucket-B files and now lists `43` files
- `rg -n 'from \\.collaboration|from \\.connectors|from \\.console_ia|from \\.cost_control|from \\.dashboard_run_contract|from \\.deprecation|from \\.design_system|bigclaw\\.deprecation|bigclaw\\.connectors|bigclaw\\.design_system|bigclaw\\.console_ia' src scripts` -> exit `1` with no matches
- `python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/mapping.py src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/repo_board.py src/bigclaw/service.py src/bigclaw/runtime.py src/bigclaw/queue.py src/bigclaw/scheduler.py src/bigclaw/workflow.py src/bigclaw/orchestration.py src/bigclaw/__main__.py scripts/dev_smoke.py` -> success, no output
- `cd bigclaw-go && go test ./...` -> success; all packages passed, with `internal/flow`, `internal/prd`, and `scripts/e2e` reporting `[no test files]`
- `git diff --name-only --diff-filter=D | sort` -> exactly the seven bucket-B Python files above
- `git status --short` -> only the scoped workpad, import rewrites, and the seven file deletions are present
