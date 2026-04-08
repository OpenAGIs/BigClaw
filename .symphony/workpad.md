# BIG-GO-174 Workpad

## Plan

1. Inspect the remaining repo-level shell wrappers and Go CLI helper coverage to find the residual non-Python script surface still presented to operators.
2. Update the wrapper layer and operator docs so `scripts/ops/bigclawctl` is the single supported root entrypoint and compatibility shims stay explicitly secondary.
3. Add or tighten regression tests that lock the remaining script/helper expectations in place.
4. Run targeted Go tests and command-level validation for the affected wrapper and CLI paths.
5. Commit the scoped changes and push the branch to `origin`.

## Acceptance

- Root operator documentation identifies `scripts/ops/bigclawctl` as the supported entrypoint and does not present compatibility wrappers as peer primary commands.
- Residual wrapper/helper behavior stays functional for compatibility callers without reintroducing Python-specific implementation paths.
- Regression coverage protects the intended wrapper/helper posture for this issue.
- Targeted validation commands run successfully and their exact commands and results are recorded in the final report.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `bash scripts/ops/bigclawctl --help`
- `bash scripts/ops/bigclawctl issue --help`
- `bash scripts/ops/bigclawctl symphony --help`
- `bash scripts/ops/bigclawctl panel --help`
