# BIG-GO-1519

## Plan
- Use the last `main` ancestor with residual Python files as the refill baseline.
- Delete the largest remaining Python-bearing directory slice first.
- Update the Go legacy compile-check so zero frozen shim files is treated as the expected post-migration state.
- Run targeted validation, then commit and push the issue branch.

## Acceptance
- Repository `.py` count drops below the baseline for the chosen refill base.
- Deleted-file evidence shows the Python residuals removed from the largest remaining directory slice.
- Validation commands and exact results are captured.
- The issue branch is committed and pushed.

## Validation
- Baseline count:
  - Command: `git ls-tree -r --name-only 410602dc46b18acf1c482c3283cb8914200c005a | rg '\.py$' | wc -l`
  - Result: `2`
- Largest-directory-first selection:
  - Command: `git ls-tree -r --name-only 410602dc46b18acf1c482c3283cb8914200c005a | rg '\.py$' | awk -F/ '{dir=(NF>1?$1:"."); counts[dir]++} END {for (d in counts) print counts[d], d}' | sort -nr`
  - Result: `2 src`
- Deleted-file evidence:
  - Command: `git ls-tree -r --name-only 410602dc46b18acf1c482c3283cb8914200c005a | rg '\.py$'`
  - Result:
    - `src/bigclaw/legacy_shim.py`
    - `src/bigclaw/models.py`
- Post-change count:
  - Command: `find . -type f -name '*.py' | wc -l`
  - Result: `0`
- Targeted test:
  - Command: `go test ./internal/legacyshim`
  - Workdir: `bigclaw-go`
  - Result: `ok  	bigclaw-go/internal/legacyshim	0.771s`
- Incorrect module-root invocation recorded:
  - Command: `go test ./bigclaw-go/internal/legacyshim`
  - Result: `go: cannot find main module, but found .git/config in /Users/openagi/code/bigclaw-workspaces/BIG-GO-1519`
