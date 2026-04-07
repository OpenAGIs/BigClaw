# BIG-GO-1567

## Plan
- Bootstrap this workspace into a valid checkout from `origin`.
- Inspect the remaining Python footprint, with emphasis on any `scripts/ops`-adjacent deletion tranche still present on disk.
- Remove a narrow unblocked Python tranche only when an equivalent Go/native path exists or the deletion is otherwise repository-safe.
- Run targeted validation and record exact commands and results.
- Commit and push the scoped change to the remote branch.

## Acceptance
- `find . -name '*.py' | wc -l` decreases from the checked-out baseline, or
- the change lands exact Go/native replacement evidence in git for the removed Python tranche.

## Validation
- Capture baseline and final Python file counts.
- Run targeted repository checks for the touched area.
- Record exact commands and exit results in this file after implementation.

## Notes
- Checked out `origin/main` into local branch `BIG-GO-1567`; repository baseline already has `0` Python files.
- This issue therefore lands exact `scripts/ops` Go/native replacement evidence with regression coverage instead of a fresh `.py` deletion.

## Validation Results
- `find . -name '*.py' | wc -l`
  Result: exit `0`, output `0`
- `find scripts/ops -maxdepth 1 -type f | sort`
  Result: exit `0`, output `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, `scripts/ops/bigclaw-symphony`, `scripts/ops/bigclawctl`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOps'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	4.903s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOps|TestRootScriptResidualSweep'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	5.372s`
- `for f in docs/go-mainline-cutover-issue-pack.md bigclaw-go/docs/reports/big-go-1567-scripts-ops-deletion-tranche.md; do echo "$f"; grep -F 'retired \`scripts/ops/bigclaw_github_sync.py\`; use \`bigclawctl github-sync\`' "$f"; grep -F 'retired \`scripts/ops/bigclaw_refill_queue.py\`; use \`bigclawctl refill\`' "$f"; grep -F 'retired \`scripts/ops/bigclaw_workspace_bootstrap.py\`; use \`bash scripts/ops/bigclawctl workspace bootstrap\`' "$f"; grep -F 'retired \`scripts/ops/symphony_workspace_bootstrap.py\`; use \`bash scripts/ops/bigclawctl workspace bootstrap\`' "$f"; grep -F 'retired \`scripts/ops/symphony_workspace_validate.py\`; use \`bash scripts/ops/bigclawctl workspace validate\`' "$f"; done`
  Result: exit `0`, output confirms all five retired `scripts/ops` Python replacement lines exist in both docs files
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOps|TestRootScriptResidualSweep'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	3.372s` after resolving the stash-pop merge state
- `rg -n 'scripts/ops/bigclaw_refill_queue.py|bigclawctl refill' docs/go-cli-script-migration-plan.md bigclaw-go/internal/regression/root_script_residual_sweep_test.go bigclaw-go/docs/reports/big-go-1567-scripts-ops-deletion-tranche.md`
  Result: exit `0`, output confirms the exact `scripts/ops/bigclaw_refill_queue.py` replacement mapping in the migration plan, residual sweep regression, and 1567 evidence report
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOps|TestRootScriptResidualSweep'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	4.237s` after tightening the refill-path migration wording
- `rg -n 'bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclawctl github-sync status --json|bigclawctl refill --help|Refill cutover' docs/go-cli-script-migration-plan.md bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go`
  Result: exit `0`, output confirms the migration plan and root-ops regression now carry the exact GitHub sync and refill cutover guidance
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestRootScriptResidualSweep'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	2.534s`
- `rg -n 'bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|bigclaw_github_sync.py|bigclawctl workspace \\.\\.\\.|bigclawctl github-sync \\.\\.\\.' README.md bigclaw-go/internal/regression/root_script_residual_sweep_test.go`
  Result: exit `0`, output confirms the README now names the retired refill, workspace, and GitHub sync Python shims together with their Go/native replacements
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootScriptResidualSweep(Docs)?|TestRootOpsMigrationDocsListOnlyGoEntrypoints'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	1.738s`
- `rg -n 'bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|scripts/ops/bigclawctl.; retired root|replaced by \`scripts/ops/bigclawctl\`' README.md bigclaw-go/internal/regression/root_script_residual_sweep_test.go`
  Result: exit `0`, output confirms the README footer now names all retired root ops shims and ties them back to `scripts/ops/bigclawctl`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootScriptResidualSweep(Docs)?|TestRootOpsMigrationDocsListOnlyGoEntrypoints'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	1.281s`
- `rg -n 'bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|bigclaw_github_sync.py|bigclawctl refill --apply --watch|bigclawctl github-sync' workflow.md bigclaw-go/internal/regression/big_go_1567_scripts_ops_deletion_tranche_test.go`
  Result: exit `0`, output confirms `workflow.md` now names the retired refill, workspace, and GitHub sync shims alongside the active `bigclawctl` replacements
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootScriptResidualSweep(Docs)?|TestRootOpsMigrationDocsListOnlyGoEntrypoints'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	1.142s`
- `rg -n 'bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|bash scripts/ops/bigclawctl \\.\\.\\.' docs/go-mainline-cutover-handoff.md bigclaw-go/internal/regression/big_go_1567_scripts_ops_deletion_tranche_test.go`
  Result: exit `0`, output confirms the cutover handoff doc now names the retired root `scripts/ops` shims and keeps operators on `bash scripts/ops/bigclawctl ...`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootScriptResidualSweep(Docs)?|TestRootOpsMigrationDocsListOnlyGoEntrypoints'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	1.247s`
- `rg -n 'bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|bash scripts/ops/bigclawctl \\.\\.\\.' docs/go-cli-script-migration-plan.md bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go`
  Result: exit `0`, output confirms the migration plan now names the full retired root shim set before the direct `bigclawctl` cutover note
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestRootScriptResidualSweep(Docs)?'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	3.217s`
- `rg -n 'bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|bigclawctl github-sync\\|refill\\|workspace \\.\\.\\.|remains removed' docs/go-mainline-cutover-issue-pack.md bigclaw-go/internal/regression/big_go_1567_scripts_ops_deletion_tranche_test.go`
  Result: exit `0`, output confirms the cutover issue pack now names the retired root `scripts/ops` shim set in both acceptance and current-progress sections
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestRootScriptResidualSweep(Docs)?'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	0.188s`
- `rg -n 'bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py|bash scripts/ops/bigclawctl \\.\\.\\.|and switch to' docs/go-cli-script-migration-plan.md bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go`
  Result: exit `0`, output confirms the migration plan now names the retired root shim set both in the slice summary and in the root compatibility retirement note
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestRootScriptResidualSweep(Docs)?'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	1.227s`
- `rg -n 'bash scripts/ops/bigclawctl github-sync\\|refill\\|workspace \\.\\.\\.|bigclaw_github_sync.py|bigclaw_refill_queue.py|bigclaw_workspace_bootstrap.py|symphony_workspace_bootstrap.py|symphony_workspace_validate.py' docs/go-mainline-cutover-handoff.md docs/go-cli-script-migration-plan.md bigclaw-go/internal/regression/root_ops_entrypoint_migration_test.go bigclaw-go/internal/regression/big_go_1567_scripts_ops_deletion_tranche_test.go`
  Result: exit `0`, output confirms the active docs and regressions now use the explicit `github-sync|refill|workspace` `bigclawctl` surface instead of the generic `...` shorthand
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1567|TestRootOpsMigrationDocsListOnlyGoEntrypoints|TestRootScriptResidualSweep(Docs)?'`
  Result: exit `0`, output `ok  	bigclaw-go/internal/regression	1.154s`
