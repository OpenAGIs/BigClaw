# BIG-GO-1143

## Plan
- confirm the repo baseline for live Python files and the actual state of each lane-owned candidate path
- document the pre-change constraint explicitly: this workspace is already materialized to zero `*.py` files, so the only scoped implementation left is regression hardening plus Go-path validation
- add a new regression tranche only for the uncovered lane guarantees around the retired root scripts and their active Go or shell entrypoints
- add a cutover-doc regression guard so the broader Go-mainline handoff artifacts cannot drift back to Python execution guidance for the retired root scripts
- add a wrapper-surface regression guard so the retained shell entrypoints stay wired to Go-first execution rather than drifting back to retired Python scripts
- add a workflow-and-hook regression guard so repo automation surfaces stay on `bigclawctl` instead of reintroducing removed Python root-script paths
- clean the checked-in BIG-GO-902 report artifacts so they stop advertising removed Python root-script commands, and add a regression guard for that report surface
- add a broad repo-surface regression guard for non-tracker docs/config/report files so direct Python root-script execution guidance cannot reappear outside historical tracker comments
- run targeted validation for the Python-file baseline, the new tranche, the active CLI replacements, and the retained legacy compile-check
- commit and push the scoped change set

## Acceptance
- the candidate retired Python paths remain absent:
- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- Go or compatibility entrypoints are asserted for the lane:
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-symphony`
- `scripts/dev_bootstrap.sh`
- `find . -name '*.py' | wc -l` stays at `0`
- exact validation commands and outcomes are recorded below
- residual risk is explicit that this workspace cannot numerically decrease the Python count further because the pre-change baseline is already zero

## Validation
- `find . -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$'`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst|TestRootScriptWorkflowAndHooksStayGoOnly'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst|TestRootScriptWorkflowAndHooksStayGoOnly|TestRootScriptReportsStayGoOnly'`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst|TestRootScriptWorkflowAndHooksStayGoOnly|TestRootScriptReportsStayGoOnly|TestRootScriptRepoSurfacesStayGoOnly'`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke --help`
- `bash scripts/ops/bigclawctl github-sync --help`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `bash scripts/ops/bigclaw-symphony --help`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- `find . -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$'` -> exit `1` with no tracked Python files
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17` -> `ok  	bigclaw-go/internal/regression	0.805s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly'` -> `ok  	bigclaw-go/internal/regression	0.767s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst'` -> `ok  	bigclaw-go/internal/regression	0.790s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst|TestRootScriptWorkflowAndHooksStayGoOnly'` -> `ok  	bigclaw-go/internal/regression	1.188s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst|TestRootScriptWorkflowAndHooksStayGoOnly|TestRootScriptReportsStayGoOnly'` -> `ok  	bigclaw-go/internal/regression	0.529s`
- `cd bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche17|TestRootScriptCutoverDocsStayGoOnly|TestRootScriptWrappersStayGoFirst|TestRootScriptWorkflowAndHooksStayGoOnly|TestRootScriptReportsStayGoOnly|TestRootScriptRepoSurfacesStayGoOnly'` -> `ok  	bigclaw-go/internal/regression	1.004s`
- `bash scripts/ops/bigclawctl create-issues --help` -> exit `0`; printed `usage: bigclawctl create-issues [flags]`
- `bash scripts/ops/bigclawctl dev-smoke --help` -> exit `0`; printed `usage: bigclawctl dev-smoke [flags]`
- `bash scripts/ops/bigclawctl github-sync --help` -> exit `0`; printed `usage: bigclawctl github-sync <install|status|sync> [flags]`
- `bash scripts/ops/bigclawctl refill --help` -> exit `0`; printed `usage: bigclawctl refill [flags]` and `bigclawctl refill seed [flags]`
- `bash scripts/ops/bigclawctl workspace bootstrap --help` -> exit `0`; printed `usage: bigclawctl workspace bootstrap [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]`
- `bash scripts/ops/bigclaw-symphony --help` -> exit `0`; printed `usage: bigclawctl symphony [flags] [args...]`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and `files: []`
- `git status --short` -> modified `.symphony/workpad.md`; added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`
- final `git status --short` after follow-up commits -> clean working tree

## Git
- branch: `feat/BIG-GO-1143-root-scripts-residual-sweep`
- commit: `b3ce907c` (`BIG-GO-1143: lock root script migration sweep`)
- follow-up commit: `2d8f34b1` (`BIG-GO-1143: record branch closeout evidence`)
- follow-up commit: `faad602f` (`BIG-GO-1143: guard root script cutover docs`)
- follow-up commit: `082b1565` (`BIG-GO-1143: guard root script wrappers`)
- follow-up commit: `24dafceb` (`BIG-GO-1143: sync workpad branch history`)
- follow-up commit: `c7dfa0c4` (`BIG-GO-1143: guard workflow root script surfaces`)
- follow-up commit: `939c8976` (`BIG-GO-1143: clean root script report surfaces`)
- follow-up commit: `d590f2df` (`BIG-GO-1143: finalize workpad history`)
- follow-up commit: `df2901b8` (`BIG-GO-1143: guard repo root script surfaces`)
- first `git push -u origin feat/BIG-GO-1143-root-scripts-residual-sweep` attempt -> exit `128` with `LibreSSL SSL_connect: SSL_ERROR_SYSCALL`
- second `git push -u origin feat/BIG-GO-1143-root-scripts-residual-sweep` attempt -> success; remote published the branch and returned the PR helper URL `https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-1143-root-scripts-residual-sweep`
- `git push` after `2d8f34b1` -> success
- `git push` after `faad602f` -> success
- `git push` after `082b1565` -> success
- `git push` after `24dafceb` -> success
- `git push` after `c7dfa0c4` -> success
- `git push` after `939c8976` -> success
- `git push` after `d590f2df` -> success
- `git push` after `df2901b8` -> success

## Residual Risk
- the repo already starts from a zero-`.py` baseline in this worktree, so this issue can only harden deletion enforcement for the lane and confirm the Go replacements; it cannot make the Python file count numerically lower from the current baseline

## External Blocker
- `BIG-GO-1143` does not exist in the reachable local tracker artifacts for this workspace (`local-issues.json`, `docs/parallel-refill-queue.json`, `docs/parallel-refill-queue.md`, or `docs/go-mainline-cutover-issue-pack.md`), so the implementation branch is complete but the issue state cannot be transitioned from here
- the only remaining stale root-script references are historical tracker comments in `local-issues.json`; rewriting prior issue history is out of scope for this lane
