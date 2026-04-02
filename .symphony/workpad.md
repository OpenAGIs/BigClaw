# BIG-GO-982 Workpad

## Scope

Final sweep for the remaining Python entrypoints under `scripts/*.py` and
`scripts/ops/*.py`.

In-scope files:

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`

Current repository Python file count before this lane: `116`
Current in-scope Python file count before this lane: `7`

## Plan

1. Confirm each in-scope Python file is only a legacy shim and identify the Go
   replacement command.
2. Delete the redundant Python wrapper files from `scripts/` and `scripts/ops/`.
3. Update repo docs that still present these Python paths as supported
   compatibility entrypoints so they point at `bash scripts/ops/bigclawctl ...`
   instead.
4. Sweep for any remaining non-historical docs that still imply the retired
   Python shims remain active and correct them.
5. Run targeted Go CLI validation that covers the replaced entrypoints.
6. Record exact file disposition, replacement basis, and repository Python file
   count impact.
7. Commit and push the scoped lane changes.
8. Publish a repo-native closeout artifact for reviewers.

## Acceptance

- Produce the exact `BIG-GO-982` batch file list under `scripts/*.py` and
  `scripts/ops/*.py`.
- Reduce the Python file count in those directories as far as possible for this
  batch.
- Document whether each file was deleted or replaced, with the corresponding Go
  command as justification.
- Report repository-wide Python file count before and after the sweep.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `rg --files scripts -g '*.py'`
- `rg --files . -g '*.py' | wc -l`

## Results

### File Disposition

- `scripts/create_issues.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl create-issues`.
  - Basis: file only proxied CLI arguments into the Go command.
- `scripts/dev_smoke.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl dev-smoke`.
  - Basis: file only emitted a deprecation warning and proxied into the Go command.
- `scripts/ops/bigclaw_github_sync.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl github-sync`.
  - Basis: file only routed into the Go command through legacy shim helpers.
- `scripts/ops/bigclaw_refill_queue.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl refill`.
  - Basis: file only routed into the Go command through legacy shim helpers.
- `scripts/ops/bigclaw_workspace_bootstrap.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl workspace`.
  - Basis: file only filled legacy default flags before dispatching to the Go command.
- `scripts/ops/symphony_workspace_bootstrap.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl workspace`.
  - Basis: file only routed into the Go command through legacy shim helpers.
- `scripts/ops/symphony_workspace_validate.py`
  - Deleted.
  - Replaced by `bash scripts/ops/bigclawctl workspace validate`.
  - Basis: file only translated legacy validate flags before dispatching to the Go command.
- `docs/go-mainline-cutover-issue-pack.md`
  - Updated.
  - Basis: the slice history still said the Python wrappers remained as compatibility
    shims, which was no longer true after this sweep removed them.

### Python File Count Impact

- Repository Python files before: `116`
- Repository Python files after: `109`
- In-scope root/ops Python files before: `7`
- In-scope root/ops Python files after: `0`
- Net reduction: `7`

### Validation Record

- `cd bigclaw-go && go test ./cmd/bigclawctl`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	3.329s`
- `bash scripts/ops/bigclawctl create-issues --help`
  - Result: exit `0`; printed `usage: bigclawctl create-issues [flags]`.
- `bash scripts/ops/bigclawctl dev-smoke`
  - Result: exit `0`; printed `smoke_ok local`.
- `bash scripts/ops/bigclawctl github-sync status --json`
  - Result: exit `0`; returned `status: ok`, `synced: true`, local and remote SHA `d295f07d50e979a3cb62785e62f1d84b674df32a`.
- `bash scripts/ops/bigclawctl refill --help`
  - Result: exit `0`; printed `usage: bigclawctl refill [flags]`.
- `bash scripts/ops/bigclawctl workspace --help`
  - Result: exit `0`; printed `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`.
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: exit `0`; printed `usage: bigclawctl workspace validate [flags]`.
- `rg --files scripts -g '*.py' || true`
  - Result: no output; there are no remaining Python files under `scripts/`.
- `rg --files . -g '*.py' | wc -l`
  - Result: `109`.
- `rg -n "scripts/(create_issues|dev_smoke)\\.py|scripts/ops/(bigclaw_github_sync|bigclaw_refill_queue|bigclaw_workspace_bootstrap|symphony_workspace_bootstrap|symphony_workspace_validate)\\.py" docs README.md .symphony src tests bigclaw-go`
  - Result: only expected migration-plan/workpad references plus a repo-root helper-path
    fixture in `bigclaw-go/internal/legacyshim/wrappers_test.go`; no active operator docs
    still point to the removed Python wrappers.

### Artifacts

- `reports/BIG-GO-982-closeout.md`
  - Repo-native closeout note summarizing the final sweep, replacement map, validation commands,
    and resulting Python file count reduction.
