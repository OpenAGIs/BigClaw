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
