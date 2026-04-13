# BIG-GO-1614 Workpad

## Plan

1. Inspect the remaining repo-root and `scripts/ops` operator wrappers and confirm which ones are still thin shell shims.
2. Remove the obsolete alias wrappers so `scripts/ops/bigclawctl` is the only supported ops entrypoint.
3. Update documentation and regression coverage to require the direct `bigclawctl` subcommand paths instead of alias wrapper files.
4. Run targeted validation for the wrapper inventory, docs guidance, and direct help/dev-smoke commands.
5. Commit the scoped change set and push the branch.

## Acceptance

- The obsolete shell alias wrappers under `scripts/ops` are absent.
- Repo docs no longer present the deleted alias wrappers as supported entrypoints.
- Regression tests assert the reduced wrapper inventory and the direct `bash scripts/ops/bigclawctl ...` operator paths.
- Targeted validation commands pass and their exact commands/results are recorded.

## Validation

- `cd bigclaw-go && go test ./internal/regression -run TestRootScriptResidualSweep -count=1`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements)' -count=1`
- `bash scripts/ops/bigclawctl issue --help`
- `bash scripts/ops/bigclawctl panel --help`
- `bash scripts/ops/bigclawctl symphony --help`
- `bash scripts/ops/bigclawctl dev-smoke`

## Execution Notes

- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \) -print | sort`
  Result: no output; the workspace contains no tracked or untracked Python files.
- `cd bigclaw-go && go test ./internal/regression -run TestRootScriptResidualSweep -count=1`
  Result: `ok  	bigclaw-go/internal/regression	3.206s`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run 'Test(BenchmarkScriptsStayGoOnly|AutomationUsageListsBIGGO1160GoReplacements)' -count=1`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	1.747s`
- `bash scripts/ops/bigclawctl issue --help`
  Result: exited `0`; printed `usage: bigclawctl issue [flags] [args...]`.
- `bash scripts/ops/bigclawctl panel --help`
  Result: exited `0`; printed `usage: bigclawctl panel [flags] [args...]`.
- `bash scripts/ops/bigclawctl symphony --help`
  Result: exited `0`; printed `usage: bigclawctl symphony [flags] [args...]`.
- `bash scripts/ops/bigclawctl dev-smoke`
  Result: exited `0`; printed `smoke_ok local`.
