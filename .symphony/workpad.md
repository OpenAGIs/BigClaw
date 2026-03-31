Issue: BIG-GO-1028

Plan
- Identify a small tranche of remaining Python test files that already have clear Go-native ownership or need only minimal Go-side test coverage.
- Port the selected Python assertions into Go tests under existing `bigclaw-go/internal/**` packages, adding only the smallest supporting Go code required to preserve behavior.
- Delete the migrated Python test files so this issue measurably reduces repository `.py` test inventory without expanding scope into unrelated Python modules.
- Run targeted Go validation for the migrated tranche, capture exact commands and outcomes, then commit and push the scoped branch changes.

Acceptance
- Changes stay scoped to the remaining Python tests tranche for this issue plus directly coupled Go test/support files.
- Repository `.py` file count decreases by removing migrated Python test files.
- Repository `.go` file count only changes where needed to host migrated Go-native coverage.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged unless the selected tranche proves they are directly affected.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `gofmt -w bigclaw-go/internal/repo/repo_surfaces_test.go`
- `go test ./internal/repo -run 'TestPermissionMatrixResolvesRoles|TestAuditFieldContractIsDeterministic|TestRepoRegistryResolvesSpaceChannelAndAgent|TestRepoRegistryJSONRoundTripPreservesSpacesAndAgents|TestNormalizeGatewayPayloadsAndErrors|TestNormalizeGatewayPayloadsReturnDecodeErrors'`
- `go test ./internal/triage -run 'TestRecommendRepoActionFollowsLineageAndDiscussionEvidence|TestApprovalEvidencePacketCapturesAcceptedAndCandidateLinks'`
- `git diff --stat`
- `git status --short`
