# BIG-GO-986

## Plan
- Audit `tests/conftest.py` and the core Python runtime-chain tests in scope for this batch.
- Map each in-scope Python test to existing Go coverage under `bigclaw-go/internal/...`, then add any missing Go assertions needed to preserve behavior.
- Delete the redundant Python test files once equivalent Go coverage exists.
- Run targeted Go tests plus repository-level Python file counts, then record exact commands and results.
- Commit the scoped change set and push the branch to `origin`.

## Acceptance
- Enumerate the Python files handled by this batch, centered on `tests/conftest.py` and core runtime-chain tests.
- Reduce Python file count in the targeted area by deleting files that are now replaced by Go tests.
- Document keep/replace/delete rationale in the final report.
- Report the repository-wide Python file count delta caused by this batch.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/worker ./internal/scheduler ./internal/workflow`
- `git status --short`
- `git log -1 --stat`
