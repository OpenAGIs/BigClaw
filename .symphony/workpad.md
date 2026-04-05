Issue: BIG-GO-1369

Plan
- Inspect the Go-native `scripts/ops` wrappers and existing `bigclawctl` tests to find an operational gap that still lacks replacement evidence.
- Add scoped regression coverage proving the `scripts/ops` entrypoints stay Go-native and preserve repo path forwarding from operator invocations.
- Record lane evidence in issue-specific status and validation reports only.
- Run targeted validation commands, then commit and push the branch.

Acceptance
- Keep the issue scoped to the `scripts/ops` python replacement sweep.
- Land concrete Go/native replacement evidence in git even if repository `.py` count is already zero.
- Preserve repository reality that `find . -name '*.py' | wc -l` remains at zero or lower after the change.
- Commit and push the completed lane.

Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1369(RepositoryHasNoPythonFiles|OpsReplacementPathsRemainAvailable|OpsWrappersStayGoNative|BigclawctlWrapperResolvesRelativeRepoFromInvocationDir|LaneReportCapturesSweepState)$'`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `bash scripts/ops/bigclawctl --help`
- `bash scripts/ops/bigclaw-issue --help`
- `bash scripts/ops/bigclaw-panel --help`
- `bash scripts/ops/bigclaw-symphony --help`
