# BIG-GO-1111

## Plan
- baseline the materialized repo-wide Python file count and confirm whether the lane candidate root scripts still exist
- verify live docs and regression coverage for the candidate root script set:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- add a Go regression contract that asserts the full candidate set stays absent from the repository root/script layer
- keep the change scoped to root-script Python retirement enforcement and the issue-local workpad/tracker evidence
- run targeted validation and record exact commands and outcomes
- commit and push the scoped change set

## Acceptance
- lane file coverage is explicit through the seven candidate root-script paths above
- the repository root/script layer is guarded against reintroducing those retired Python files
- validation records the current Python-file baseline and confirms the candidate set is absent
- exact validation commands and outcomes are recorded below

## Validation
- `find . -type f -name '*.py' | wc -l`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `rg -n "scripts/create_issues\\.py|scripts/dev_smoke\\.py|scripts/ops/bigclaw_github_sync\\.py|scripts/ops/bigclaw_refill_queue\\.py|scripts/ops/bigclaw_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_validate\\.py" README.md docs bigclaw-go scripts .github workflow.md -g '!reports/**' -g '!local-issues.json'`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Validation Results
- `find . -type f -name '*.py' | wc -l` -> `0`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l` -> `0`
- `rg -n "scripts/create_issues\\.py|scripts/dev_smoke\\.py|scripts/ops/bigclaw_github_sync\\.py|scripts/ops/bigclaw_refill_queue\\.py|scripts/ops/bigclaw_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_bootstrap\\.py|scripts/ops/symphony_workspace_validate\\.py" README.md docs bigclaw-go scripts .github workflow.md -g '!reports/**' -g '!local-issues.json'` -> active matches are limited to retirement guidance in `docs/go-cli-script-migration-plan.md`, a Go-only warning in `workflow.md`, the archived cutover pack, and the new regression assertions
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.504s`
- `git status --short` -> modified `.symphony/workpad.md`; modified `docs/go-cli-script-migration-plan.md`; added `bigclaw-go/internal/regression/root_python_sweep_issue1111_test.go`
