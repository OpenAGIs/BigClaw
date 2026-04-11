# BIG-GO-254 Residual Scripts Python Sweep U

## Scope

`BIG-GO-254` audits the last repo-root CLI helper wrapper that still behaved
like a migration-era shim instead of a compiled launcher.

The checked-out workspace already reports a physical Python file inventory of
`0`, so this lane lands as a regression-prevention and wrapper-replacement
sweep rather than an in-branch Python deletion batch.

## Python Inventory

Repository-wide Python file count before lane changes: `0`.

Repository-wide Python file count after lane changes: `0`.

Explicit remaining Python asset list: none.

## Residual Wrapper Replacement

- `scripts/ops/bigclawctl` now builds and reuses a cached `bigclawctl` binary
  under the user cache directory instead of shelling into `go run`.
- The launcher still preserves the repo-root `--repo` injection behavior for
  callers that omit it.
- The cache location remains configurable via `BIGCLAWCTL_BIN_DIR` and falls
  back to `${XDG_CACHE_HOME:-$HOME/.cache}/bigclaw`.

## Go Or Native Replacement Paths

- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `docs/go-cli-script-migration-plan.md`
- `README.md`
- `bigclaw-go/internal/regression/big_go_254_residual_wrapper_sweep_test.go`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `bash scripts/ops/bigclawctl --help`
  Result: exit `0`; the cached compiled launcher printed the root `bigclawctl` usage.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'Test(BIGGO254|RootScriptResidualSweep|RunGitHubSyncInstallJSONOutputDoesNotEscapeArrowTokens|RunGitHubSyncHelpPrintsUsageAndExitsZero)'`
  Result: targeted command passed.

Residual risk: the repository already started this lane at zero physical Python
files, so BIG-GO-254 hardens the compiled-wrapper path instead of lowering the
file count further.
