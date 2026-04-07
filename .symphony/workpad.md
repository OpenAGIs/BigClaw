# BIG-GO-1578

## Plan
1. Reconfirm the current repository baseline for the issue's candidate Python files.
2. Record the exact Go or native replacement owners for each candidate path.
3. Add a lane-specific regression guard and repo-native reports that keep this candidate set absent and auditable.
4. Run targeted validation, record exact commands and results, then commit and push `BIG-GO-1578`.

## Acceptance
- Enumerate the candidate Python files covered by this sweep.
- Prefer deletion or Go replacement for each covered file.
- Any retained Python file would need to be reduced to a thin compatibility layer with a documented deletion condition.
- Record exact validation commands, results, and residual risks.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src tests scripts/ops bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1578(RepositoryHasNoPythonFiles|CandidatePathsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepLedger)$'`

## Notes
- The issue workspace was initially provisioned with an invalid git HEAD and no checked-out files, so restoring a usable checkout from `origin/main` was part of execution.
- Current `main` baseline is already physically Python-free, so this lane is focused on regression-hardening and exact replacement evidence for the listed residual candidates.
