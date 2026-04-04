# BIG-GO-1256 Workpad

## Plan

1. Audit repository-wide physical Python assets, with priority on `src/bigclaw`,
   `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Record the lane inventory, Go replacement paths, and validation evidence in
   lane-specific report artifacts.
3. Add a regression guard that keeps the repository and priority directories
   Python-free and verifies the replacement paths remain available.
4. Run targeted validation commands, capture exact results, then commit and push
   the lane changes.

## Acceptance

- Remaining Python asset inventory for the lane is explicitly recorded.
- The lane either removes residual `.py` files or, if the baseline is already
  zero, hardens and documents that Go-only state.
- Go replacement paths and verification commands are recorded.
- Targeted tests run successfully and exact commands plus results are preserved.

## Validation

- `find . -type f -name '*.py' | wc -l`
- `for dir in src/bigclaw tests scripts bigclaw-go/scripts; do if [ -d "$dir" ]; then find "$dir" -type f -name '*.py' | sort; fi; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1256(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable)$'`

---

# BIG-GO-1252 Workpad Archive

## Plan

- Reconfirm the repository-wide physical Python asset inventory, with explicit checks for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- Add a lane-specific Go regression guard for `BIG-GO-1252` that keeps the repository and the priority residual directories Python-free while asserting the replacement entrypoints still exist.
- Add lane documentation for `BIG-GO-1252` that records the remaining Python asset inventory, the Go replacement paths, and the exact validation commands/results for this sweep.
- Run targeted validation, capture exact command outcomes, then commit and push the lane changes to the remote branch.

## Acceptance

- The `BIG-GO-1252` lane has an explicit, auditable remaining Python asset inventory.
- The repository remains free of physical `.py` files, including the priority directories `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
- The retained Go or shell replacement paths for the removed Python assets are documented and enforced by regression coverage.
- Exact validation commands and outcomes are recorded for this lane.

## Validation

- `find . -type f -name '*.py' | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1252(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
- `git status --short`
