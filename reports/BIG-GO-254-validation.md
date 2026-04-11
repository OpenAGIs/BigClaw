# BIG-GO-254 Validation

Date: 2026-04-12

## Scope

Issue: `BIG-GO-254`

Title: `Residual scripts Python sweep U`

This lane replaced the remaining repo-root `bigclawctl` shell launcher path
that still behaved as a migration-era wrapper and recorded the repository's
already-zero Python baseline.

The checked-out workspace started with a repository-wide Python file count of
`0`, so there was no physical `.py` asset left to delete or replace in-branch.
The delivered work hardens that zero-Python baseline by moving
`scripts/ops/bigclawctl` off the `go run` wrapper path and onto a cached
compiled-binary launcher.

## Remaining Python Asset Inventory

- Repository-wide physical `.py` files: `none`

## Replacement Paths

- Cached launcher: `scripts/ops/bigclawctl`
- Root operator guide: `README.md`
- Migration plan: `docs/go-cli-script-migration-plan.md`
- Issue lane report: `bigclaw-go/docs/reports/big-go-254-python-asset-sweep.md`
- Issue regression guard: `bigclaw-go/internal/regression/big_go_254_residual_wrapper_sweep_test.go`

## Validation Commands

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `bash scripts/ops/bigclawctl --help`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'Test(BIGGO254|RootScriptResidualSweep|RunGitHubSyncInstallJSONOutputDoesNotEscapeArrowTokens|RunGitHubSyncHelpPrintsUsageAndExitsZero)'`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-254 --json url,title,state,number`
- Public compare page: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-254`

## Validation Results

### Repository Python inventory

Command:

```bash
find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort
```

Result:

```text
none
```

### Cached launcher help

Command:

```bash
bash scripts/ops/bigclawctl --help
```

Result:

```text
usage: bigclawctl <github-sync|workspace|automation|refill|local-issues|create-issues|dev-smoke|symphony|issue|panel> ...

commands:
  github-sync     install/sync/status hooks and branch sync state
  workspace       bootstrap/cleanup/validate workspaces using the shared mirror
  automation      run migrated e2e/benchmark/migration automation entrypoints
  refill          promote issues to maintain target in-progress count
  local-issues    manage the repo-native issue store in local-issues.json
  create-issues   seed the GitHub repo with the canned issue plans
  dev-smoke       run the Go control-plane smoke decision check
  symphony        launch Symphony against this repo workflow
  issue           open local tracker flows or proxy symphony issue
  panel           proxy symphony panel against this repo workflow
```

### Targeted regression and CLI tests

Command:

```bash
cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'Test(BIGGO254|RootScriptResidualSweep|RunGitHubSyncInstallJSONOutputDoesNotEscapeArrowTokens|RunGitHubSyncHelpPrintsUsageAndExitsZero)'
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	0.518s
ok  	bigclaw-go/internal/regression	0.195s
```

### Branch sync status

Command:

```bash
bash scripts/ops/bigclawctl github-sync status --json
```

Result:

```json
{
  "ahead": 0,
  "behind": 0,
  "branch": "BIG-GO-254",
  "detached": false,
  "dirty": false,
  "diverged": false,
  "local_sha": "ad6b67758bb03808724b8df3f0f4bfdeb4a7b137",
  "pushed": true,
  "relation_known": true,
  "remote_exists": true,
  "remote_sha": "ad6b67758bb03808724b8df3f0f4bfdeb4a7b137",
  "status": "ok",
  "synced": true
}
```

### PR query attempt

Command:

```bash
gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-254 --json url,title,state,number
```

Result:

```text
To get started with GitHub CLI, please run:  gh auth login
Alternatively, populate the GH_TOKEN environment variable with a GitHub API authentication token.
```

Known branch PR creation URL from push output:

```text
https://github.com/OpenAGIs/BigClaw/pull/new/BIG-GO-254
```

### Public compare page

Source:

```text
https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-254
```

Observed result:

```text
GitHub renders the public compare page anonymously, showing base `main`, head `BIG-GO-254`, `7 commits`, and `8 files changed`.
```

## Git

- Branch: `BIG-GO-254`
- Baseline HEAD before lane commit: `1858cdb1`
- Push target: `origin/BIG-GO-254`

## Residual Risk

- The live branch baseline was already Python-free, so BIG-GO-254 could only
  harden and document the compiled launcher path rather than numerically lower
  the repository `.py` count.
- GitHub PR creation/querying is blocked in this workspace because `gh` is not
  authenticated and no `GH_TOKEN`/`GITHUB_TOKEN` is present.
- The branch is still publicly reviewable through the compare page even without
  GitHub CLI authentication.
