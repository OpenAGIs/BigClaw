# BIG-GO-239 Workpad

## Plan

1. Confirm the repository has no residual physical Python assets, including hidden, nested, and auxiliary paths relevant to this sweep.
2. Add an issue-specific regression test file under `bigclaw-go/internal/regression` covering the repository-wide sweep plus targeted hidden/nested auxiliary directories and retained native evidence paths.
3. Add an issue-specific lane report under `bigclaw-go/docs/reports` documenting audited directories, retained assets, validation commands, and expected zero-Python results.
4. Run targeted regression tests and direct filesystem sweep commands, capture exact commands and results, then commit and push the issue branch.

## Acceptance

- `.symphony/workpad.md` exists and records plan, acceptance, and validation before code changes.
- Issue-scoped regression coverage exists for `BIG-GO-239`.
- Issue-scoped report exists for `BIG-GO-239`.
- Targeted hidden, nested, or overlooked auxiliary directories are asserted Python-free.
- Validation commands complete successfully and their exact commands/results are recorded.
- Changes stay scoped to the new `BIG-GO-239` regression/report artifacts plus this workpad.

## Validation

- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' -o -name '*.pyc' -o -name '.python-version' \) -print | sort`
- `find .github .githooks .symphony docs/reports bigclaw-go/docs/adr bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs reports -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' -o -name '*.pyc' -o -name '.python-version' \) 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO239(RepositoryHasNoPythonFiles|HiddenNestedAndOverlookedAuxiliaryDirectoriesStayPythonFree|RetainedNativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`
