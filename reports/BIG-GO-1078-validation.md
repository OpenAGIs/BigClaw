# BIG-GO-1078 Validation

## Scope

Remove the tranche-2 residual Python operator wrappers in `scripts/ops` and keep the Go-only
replacement path on `bash scripts/ops/bigclawctl`.

Deleted files:

- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

Replacement surfaces confirmed:

- `scripts/ops/bigclawctl`
- `bigclaw-go/internal/bootstrap/bootstrap.go`
- `bigclaw-go/internal/refill/queue.go`
- `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`

## Validation Commands

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `find scripts/ops -maxdepth 1 -type f -name '*.py'`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `git rev-parse HEAD`
- `git branch --show-current`

## Results

- `find . -name '*.py' | wc -l`
  - Result: `39`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
  - Result:
    - `ok  	bigclaw-go/internal/legacyshim	1.219s`
    - `ok  	bigclaw-go/internal/regression	1.256s`
    - `ok  	bigclaw-go/cmd/bigclawctl	5.340s`
- `find scripts/ops -maxdepth 1 -type f -name '*.py'`
  - Result: no output
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14`
  - Result: `ok  	bigclaw-go/internal/regression	0.523s`
- `bash scripts/ops/bigclawctl refill --help`
  - Result: exited `0`; printed `usage: bigclawctl refill [flags]`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
  - Result: exited `0`; printed `usage: bigclawctl workspace bootstrap [flags]`
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: exited `0`; printed `usage: bigclawctl workspace validate [flags]`
- `git rev-parse HEAD`
  - Result: `cf3860ae8f129da1d6d070c09995c617fd773f71`
- `git branch --show-current`
  - Result: `symphony/BIG-GO-1078`

## Notes

- The branch is pushed to `origin/symphony/BIG-GO-1078`.
- `gh pr list --repo OpenAGIs/BigClaw --head symphony/BIG-GO-1078 --json url,title,state,number`
  could not run in this workspace because GitHub CLI is not authenticated.
