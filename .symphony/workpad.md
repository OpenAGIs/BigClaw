# BIG-GO-1522 Workpad

## Plan

1. Use `origin/big-go-948-lane8-remaining-python-tests` as the working base because `origin/main` is already Python-free and cannot satisfy the required before/after reduction.
2. Record the repository-wide `.py` baseline and focus the change on physically deleting only residual Python test files under `bigclaw-go/scripts`.
3. Run targeted validation for the affected Go-owned automation surface plus exact filesystem evidence commands.
4. Record before/after counts and exact removed-file evidence, then commit and push `BIG-GO-1522`.

## Acceptance

- The repository-wide `.py` file count decreases from the measured baseline.
- Only residual Python test files are removed from disk for this issue.
- Exact deleted-file evidence is available from `git diff --name-status` and `git status --short`.
- Exact validation commands and results are captured from this run.
- The branch is committed and pushed to `origin/BIG-GO-1522`.

## Validation

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | wc -l`
- `find bigclaw-go/scripts -type f -name '*_test.py' -print | sort`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./cmd/bigclawd`
- `git diff --name-status`
- `git status --short`
