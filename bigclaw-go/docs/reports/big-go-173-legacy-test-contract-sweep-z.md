# BIG-GO-173 Legacy Test Contract Sweep Z

## Scope

`BIG-GO-173` closes the next residual retired Python test-contract slice that
did not yet have its own Go/native replacement manifest:
`tests/test_repo_collaboration.py`, `tests/test_repo_gateway.py`,
`tests/test_repo_governance.py`, and `tests/test_repo_registry.py`.

## Python Baseline

Repository-wide Python file count: `0`.

This checkout therefore lands as a native-replacement sweep rather than a direct
Python-file deletion batch because there are no physical `.py` assets left to
remove in-branch.

## Deferred Legacy Test Replacements

The sweep-Z Go/native replacement registry lives in
`bigclaw-go/internal/migration/legacy_test_contract_sweep_z.go`.

- `tests/test_repo_collaboration.py`
  - Go replacements:
    - `bigclaw-go/internal/collaboration/thread.go`
    - `bigclaw-go/internal/collaboration/thread_test.go`
    - `bigclaw-go/internal/repo/board.go`
  - Evidence:
    - `bigclaw-go/internal/repo/repo_surfaces_test.go`
    - `docs/go-mainline-cutover-issue-pack.md`
    - `docs/go-mainline-cutover-handoff.md`
- `tests/test_repo_gateway.py`
  - Go replacements:
    - `bigclaw-go/internal/repo/gateway.go`
    - `bigclaw-go/internal/repo/repo_surfaces_test.go`
    - `bigclaw-go/internal/repo/commits.go`
  - Evidence:
    - `docs/go-mainline-cutover-issue-pack.md`
    - `docs/go-mainline-cutover-handoff.md`
- `tests/test_repo_governance.py`
  - Go replacements:
    - `bigclaw-go/internal/repo/governance.go`
    - `bigclaw-go/internal/repo/governance_test.go`
    - `bigclaw-go/internal/repo/plane.go`
  - Evidence:
    - `docs/go-mainline-cutover-issue-pack.md`
    - `docs/go-mainline-cutover-handoff.md`
- `tests/test_repo_registry.py`
  - Go replacements:
    - `bigclaw-go/internal/repo/registry.go`
    - `bigclaw-go/internal/repo/repo_surfaces_test.go`
    - `bigclaw-go/internal/repo/links.go`
  - Evidence:
    - `docs/go-mainline-cutover-issue-pack.md`
    - `docs/go-mainline-cutover-handoff.md`

## Why This Sweep Is Now Safe

These retired Python tests already have stable Go-native owners in the active
repo tree:

- repo collaboration behavior is owned by the Go-native collaboration thread
  merge surface and repo discussion board contract.
- repo gateway behavior is owned by the Go repo gateway normalization and commit
  surface.
- repo governance behavior is owned by the Go permission contract and repo
  control-plane surface.
- repo registry behavior is owned by the Go repo registry, routing, and
  run-commit link surface.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO173LegacyTestContractSweepZ(ManifestMatchesDeferredLegacyTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.191s`
