# BIG-GO-1203 Workpad

## Plan

1. Confirm the repository-wide `.py` inventory, with explicit sweeps for `src/bigclaw`, `tests`, `scripts`, and `bigclaw-go/scripts`.
2. Add lane-specific Go regression coverage that preserves the zero-Python baseline for `BIG-GO-1203`.
3. Record lane evidence in `reports/BIG-GO-1203-validation.md` and `reports/BIG-GO-1203-status.json`, including Go replacement paths and exact validation results.
4. Run targeted validation, then commit and push the scoped lane changes.

## Acceptance

- Produce an explicit remaining Python asset inventory for this lane.
- Keep the repository Go-only by replacing the lane deliverable with Go regression and documentation assets rather than restoring any Python behavior.
- Document the Go replacement paths and validation commands.
- Preserve or improve the repository Python file count, with the current expected baseline at `0`.

## Validation

- `find . -name '*.py' | wc -l`
- `find src tests scripts bigclaw-go/scripts -name '*.py' 2>/dev/null || true`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO1203(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree)$'`
