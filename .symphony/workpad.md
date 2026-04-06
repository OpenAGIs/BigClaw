# BIG-GO-1532 Workpad

## Plan

1. Reconfirm the repository-wide physical Python test-file baseline and the
   focused `workspace` / `bootstrap` / `planning` blocker surface.
2. Add lane-scoped reporting artifacts that record exact before/after counts,
   the exact deleted-file ledger, and the active Go/native replacement paths.
3. Add focused regression coverage so the repository and the
   `workspace/bootstrap/planning` surface remain Python-free.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1532`.

## Acceptance

- The lane records repository-wide `.py` counts before and after the change.
- The lane records the focused `workspace/bootstrap/planning` residual scan.
- The lane includes an exact deleted-file ledger, even if it is empty because
  the baseline is already Python-free.
- The lane names the active Go/native replacement paths for the retired
  bootstrap/planning surface.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1532`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find workspace bootstrap planning bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1532(RepositoryHasNoPythonFiles|BootstrapPlanningResidualAreaStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`

## GitHub

- Branch to push: `origin/BIG-GO-1532`
- Compare view after push: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-1532?expand=1`
