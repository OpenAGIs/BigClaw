# BIG-GO-1562 Workpad

## Plan

1. Confirm the repository-wide physical Python baseline and the focused
   `src/bigclaw` workflow-orchestration tranche-B surface.
2. Record the exact Go/native replacement paths that now own the retired
   `src/bigclaw` tranche-B modules.
3. Add a lane-specific regression guard that keeps the repository Python-free,
   keeps the tranche-B source paths absent, and asserts the replacement paths
   remain available.
4. Run targeted validation, record the exact commands and results, then commit
   and push `BIG-GO-1562`.

## Acceptance

- Physical Python files on disk decrease, or exact Go/native replacement
  evidence lands in git for the targeted `src/bigclaw` tranche-B surface.
- The focused tranche stays scoped to the workflow/scheduler orchestration
  modules formerly under `src/bigclaw`.
- Exact validation commands and outcomes are recorded in repo-native artifacts.
- The change is committed and pushed on `BIG-GO-1562`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1562(RepositoryHasNoPythonFiles|WorkflowOrchestrationTrancheBStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`

## Outcome

- Replacement-evidence and regression-hardening landed because the repository
  was already Python-free for the targeted tranche when the lane started.
- Latest branch state pushed on `origin/BIG-GO-1562`; see the compare URL in
  `reports/BIG-GO-1562-status.json`.
- PR opened: `https://github.com/OpenAGIs/BigClaw/pull/225`
