# BIG-GO-1002 Root Script Python Removal

## Batch Inventory

Removed from `scripts/*.py` and `scripts/ops/*.py`:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

Remaining Python files in those target globs after this change:

- none

## Disposition

- `scripts/create_issues.py`
  - Deleted.
  - Basis: the only behavior was `bigclawctl create-issues`; the supported operator entrypoint is now `bash scripts/ops/bigclawctl create-issues`.
- `scripts/dev_smoke.py`
  - Deleted.
  - Basis: the only runtime behavior was deprecation plus `bigclawctl dev-smoke`; the Go/Bash path is already the documented mainline.
- `scripts/ops/bigclaw_github_sync.py`
  - Deleted.
  - Basis: wrapper-only shim over `bigclawctl github-sync`.
- `scripts/ops/bigclaw_refill_queue.py`
  - Deleted.
  - Basis: wrapper-only shim over `bigclawctl refill`.
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  - Deleted.
  - Basis: wrapper-only shim that added default flags before dispatching to `bigclawctl workspace`; operators should call the Go/Bash command directly.
- `scripts/ops/symphony_workspace_bootstrap.py`
  - Deleted.
  - Basis: wrapper-only shim over `bigclawctl workspace`.
- `scripts/ops/symphony_workspace_validate.py`
  - Deleted.
  - Basis: wrapper-only shim that translated legacy flags for `bigclawctl workspace validate`; callers should use the current Go/Bash flags directly.
- `src/bigclaw/legacy_shim.py`
  - Deleted.
  - Basis: no remaining importers after removing the root/operator Python wrappers.
- `bigclaw-go/internal/legacyshim/wrappers.go`
  - Deleted.
  - Basis: Go helper layer existed only to mirror the removed Python wrapper behavior.
- `bigclaw-go/internal/legacyshim/wrappers_test.go`
  - Deleted.
  - Basis: test coverage was only for the removed wrapper helper layer.

## Expected Command Replacements

- `python3 scripts/create_issues.py ...` -> `bash scripts/ops/bigclawctl create-issues ...`
- `python3 scripts/dev_smoke.py` -> `bash scripts/ops/bigclawctl dev-smoke`
- `python3 scripts/ops/bigclaw_github_sync.py ...` -> `bash scripts/ops/bigclawctl github-sync ...`
- `python3 scripts/ops/bigclaw_refill_queue.py ...` -> `bash scripts/ops/bigclawctl refill ...`
- `python3 scripts/ops/bigclaw_workspace_bootstrap.py ...` -> `bash scripts/ops/bigclawctl workspace ...`
- `python3 scripts/ops/symphony_workspace_bootstrap.py ...` -> `bash scripts/ops/bigclawctl workspace ...`
- `python3 scripts/ops/symphony_workspace_validate.py ...` -> `bash scripts/ops/bigclawctl workspace validate ...`

## Python Count Impact

- Repository Python files before: `108`
- Repository Python files after: `100`
- Targeted batch Python files before: `7`
- Targeted batch Python files after: `0`
- Net reduction: `8` repository-wide and `7` in the targeted batch

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim ./internal/regression`
  - Result: `ok   bigclaw-go/cmd/bigclawctl 2.492s`, `ok   bigclaw-go/internal/legacyshim (cached)`, `ok   bigclaw-go/internal/regression (cached)`
- `bash scripts/ops/bigclawctl create-issues --help`
  - Result: usage printed for `bigclawctl create-issues`
- `bash scripts/ops/bigclawctl dev-smoke`
  - Result: `smoke_ok local`
- `bash scripts/ops/bigclawctl refill --help`
  - Result: usage printed for `bigclawctl refill` and `refill seed`
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: usage printed for `bigclawctl workspace validate`
- `bash scripts/ops/bigclawctl github-sync status --json`
  - Result: `status: ok`, `dirty: true`, `behind: 1`, `synced: false` before commit/push
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
  - Result: `status: ok` for `src/bigclaw/runtime.py` and `src/bigclaw/__main__.py`
- `find . -name '*.py' | wc -l`
  - Result: `100`

## Python Count Impact

- Repository Python files at `HEAD`: `108`
- Repository Python files after change: `100`
- Targeted batch Python files at `HEAD`: `7`
- Targeted batch Python files after change: `0`
- Net repository reduction: `8`

## Validation

- `rg --files -g 'scripts/*.py' -g 'scripts/ops/*.py'`
  - Result: no output; exit `1`.
- `bash scripts/ops/bigclawctl create-issues --help`
  - Result: exit `0`; printed `usage: bigclawctl create-issues [flags]`.
- `bash scripts/ops/bigclawctl dev-smoke`
  - Result: exit `0`; printed `smoke_ok local`.
- `bash scripts/ops/bigclawctl github-sync status --json`
  - Result: exit `0`; JSON reported `status: ok` and `synced: true`.
- `bash scripts/ops/bigclawctl refill --help`
  - Result: exit `0`; printed `usage: bigclawctl refill [flags]`.
- `bash scripts/ops/bigclawctl workspace --help`
  - Result: exit `0`; printed `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`.
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: exit `0`; printed `usage: bigclawctl workspace validate [flags]`.
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim`
  - Result: exit `0`; both packages passed.
