# BIG-GO-254 Workpad

## Plan
1. Confirm the current repository baseline for residual Python files and identify the remaining wrapper/helper surface assigned to this sweep.
2. Replace the repo-root `scripts/ops/bigclawctl` launcher so it no longer shells into `go run`, while preserving the existing operator interface and `--repo` path handling.
3. Update the migration docs to reflect the compiled launcher path and capture issue-scoped sweep evidence.
4. Add an issue-scoped regression test for the residual wrapper/helper behavior.
5. Run targeted validation, then commit and push `BIG-GO-254`.

## Acceptance
- The repository remains free of tracked `.py` files.
- `scripts/ops/bigclawctl` no longer embeds `go run ./cmd/bigclawctl`; it launches a compiled `bigclawctl` binary path instead.
- Operator docs no longer describe `scripts/ops/bigclawctl` as a `go run` wrapper backlog item.
- Issue-scoped regression coverage exists for the compiled launcher behavior.
- Targeted validation commands are recorded with exact results.

## Validation
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `bash scripts/ops/bigclawctl --help`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/regression -run 'Test(BIGGO254|RootScriptResidualSweep|RunGitHubSyncInstallJSONOutputDoesNotEscapeArrowTokens|RunGitHubSyncHelpPrintsUsageAndExitsZero)'`

## Execution Notes
- 2026-04-12: `origin/BIG-GO-254` is pushed and synced at the current branch tip.
- 2026-04-12: `gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-254 --json url,title,state,number` failed with exit code `4` because `gh` is unauthenticated in this workspace.
- 2026-04-12: The push output exposed the deterministic PR creation URL `https://github.com/OpenAGIs/BigClaw/pull/new/BIG-GO-254`; this was recorded in the issue artifacts because PR API access is blocked locally.
