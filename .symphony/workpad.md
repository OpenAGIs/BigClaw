## Plan

1. Verify repository materialization and current Python file count for the candidate sweep area.
2. Inspect the replacement Go script layout that covers the listed benchmark, e2e, migration, and root script lanes.
3. Add a scoped regression guard and issue evidence so this lane preserves the zero-Python state.
4. Run targeted validation commands and record exact commands plus results.
5. Commit the scoped change and push branch `BIG-GO-1167`.

## Acceptance

- Candidate sweep areas contain no real Python files.
- Repository-level `find . -name '*.py' | wc -l` remains `0`.
- Go replacement and compatibility paths for the affected script areas are verifiable from the current tree.
- Added issue-scoped guard/evidence without broad unrelated edits.

## Validation

- `find . -name '*.py' | wc -l`
- `rg --files bigclaw-go/scripts scripts | rg '\\.py$'`
- Targeted inspection of Go replacement files for benchmark, e2e, migration, and root script paths.
- Targeted test command for the added regression guard.

## Results

- `find . -name '*.py' | wc -l` -> `0`
- `rg --files bigclaw-go/scripts scripts | rg '\\.py$'` -> exit `1` (no matches)
- `cd bigclaw-go && go test ./internal/regression -run BIGGO1167` -> `ok  	bigclaw-go/internal/regression	0.481s`
