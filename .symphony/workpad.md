# BIG-GO-1483 Workpad

## Plan

1. Confirm the repository baseline for `bigclaw-go/scripts` and capture exact before counts for physical `.py` files plus any checked-in caller references that still mention those retired Python entrypoints.
2. Update the checked-in migration documentation so the supported paths point only at the Go CLI and retained shell wrappers, without listing `bigclaw-go/scripts/*.py` as active caller targets.
3. Add lane-scoped regression coverage and evidence artifacts that pin the zero-Python baseline and the checked-in caller cutover state for `bigclaw-go/scripts`.
4. Run targeted validation, record exact commands and results, then commit and push the scoped change set.

## Acceptance

- `bigclaw-go/scripts` remains free of physical `.py` files.
- The checked-in caller documentation for the residual `bigclaw-go/scripts` migration surface points to Go CLI or retained shell wrappers instead of retired Python entrypoints.
- Lane artifacts capture exact before/after evidence for Python file counts and caller-reference counts.
- Targeted regression tests pass and their exact commands/results are recorded.
- The branch is committed and pushed to the remote issue branch.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find bigclaw-go/scripts -type f -name '*.py' | sort`
- `rg -n --glob '!reports/**' --glob '!local-issues.json' --glob '!.symphony/**' 'bigclaw-go/scripts/[^[:space:]]+\\.py|python3? [^\\n]*bigclaw-go/scripts/[^[:space:]]+\\.py' README.md docs scripts .github bigclaw-go | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'`

## Notes

- Baseline on 2026-04-06 already has zero physical `.py` files anywhere in the repository, including `bigclaw-go/scripts`.
- The active implementation task is therefore to remove stale checked-in caller references to retired `bigclaw-go/scripts/*.py` paths and lock that state with evidence and regression tests.
