# BIG-GO-173 Workpad

## Plan
- Inspect existing Python-removal regression patterns and identify a small residual set of retired `tests/*.py` contracts that still lack dedicated replacement-manifest coverage.
- Add a new issue-scoped migration manifest, regression test, and lane report for that residual set.
- Run targeted regression tests plus a repository Python-file sweep, then capture exact commands and results.
- Commit the scoped changes and push the `BIG-GO-173` branch to `origin`.

## Acceptance
- `.symphony/workpad.md` exists before implementation edits.
- The change is limited to issue-scoped regression/migration/report artifacts for `BIG-GO-173`.
- New coverage documents a residual retired Python test set and verifies the mapped Go replacement surface exists.
- Targeted validation commands pass and are recorded exactly.
- Changes are committed on `BIG-GO-173` and pushed to `origin/BIG-GO-173`.

## Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO173LegacyTestContractSweepZ(ManifestMatchesDeferredLegacyTests|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
