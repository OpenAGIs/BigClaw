# BIG-GO-1569 Workpad

## Plan

1. Verify the checked-out `origin/main` baseline for residual Python files and
   confirm whether any physical validation-helper `.py` files still exist.
2. Because the validation-helper Python surface is already absent in the branch
   baseline, add lane-scoped evidence that records the exact zero-file state and
   the Go/native replacement entrypoint for `workspace validate`.
3. Add a focused regression test that keeps the retired validation helper path
   deleted and the replacement surface available.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1569`.

## Acceptance

- Repository reality is recorded for the validation-helper tranche.
- Either `.py` count drops or exact Go/native replacement evidence lands in git.
- Targeted validation commands and outcomes are recorded exactly.
- Changes stay scoped to `BIG-GO-1569`.
- The branch is committed and pushed to `origin/BIG-GO-1569`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find scripts scripts/ops bigclaw-go/cmd/bigclawctl -type f -name '*.py' 2>/dev/null | sort`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1569(RepositoryHasNoPythonFiles|ValidationHelperPathsStayDeleted|ValidationHelperGoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`
