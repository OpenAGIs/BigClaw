# BIG-GO-1528 Workpad

## Scope

Remove support/example Python assets that still count toward actual repository inventory while keeping the change scoped to already-migrated Go-owned entrypoints.

Initial repository Python inventory on branch `BIG-GO-1528`:

- Repository-wide `.py` files before: `108`
- Candidate support/example assets:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
  - `bigclaw-go/scripts/benchmark/soak_local.py`

## Plan

1. Confirm the candidate Python files are compatibility/support assets with Go replacements already present.
2. Remove only the support/example Python files that are already superseded.
3. Update the directly affected docs and any helper that still shells through a removed Python file.
4. Recount repository `.py` files and record exact removed-file evidence.
5. Run targeted validation commands for the touched Go entrypoints and the benchmark helper.
6. Commit and push the scoped branch.

## Acceptance

- Actual repository `.py` inventory decreases from the branch baseline.
- Exact removed-file evidence is captured with before/after counts.
- Only support/example Python assets are removed in this lane.
- Direct references to removed files are updated.
- Targeted validation commands are run and recorded with exact results.
- The branch change is committed and pushed.

## Validation

- `find . -type f -name '*.py' | wc -l`
- `git diff --name-status --diff-filter=D`
- `bash scripts/ops/bigclawctl create-issues --help`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/ops/bigclawctl github-sync status --json`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `python3 bigclaw-go/scripts/benchmark/run_matrix.py --help`
- `git status --short`
- `git log -1 --stat`

## Results

### Removed Files

- `scripts/create_issues.py`
- `scripts/dev_smoke.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `scripts/ops/bigclaw_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/symphony_workspace_validate.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`

### Count Impact

- Repository `.py` files before: `108`
- Repository `.py` files after: `100`
- Net reduction: `8`

### Validation Record

- `find . -type f -name '*.py' | wc -l`
  - Result: `100`
- `git diff --name-status --diff-filter=D`
  - Result: exactly the eight files listed in `Removed Files`
- `bash scripts/ops/bigclawctl create-issues --help`
  - Result: passed, printed `bigclawctl create-issues` usage
- `bash scripts/ops/bigclawctl dev-smoke`
  - Result: passed, `smoke_ok local`
- `bash scripts/ops/bigclawctl github-sync status --json`
  - Result: passed, branch `BIG-GO-1528`, local SHA `d0170b0c0df469fe4ed6062a5b67b2bd10016fd3`, `status: ok`
- `bash scripts/ops/bigclawctl refill --help`
  - Result: passed, printed `bigclawctl refill` usage
- `bash scripts/ops/bigclawctl workspace validate --help`
  - Result: passed, printed `bigclawctl workspace validate` usage
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
  - Result: passed, printed `bigclawctl automation benchmark soak-local` usage
- `python3 bigclaw-go/scripts/benchmark/run_matrix.py --help`
  - Result: passed, printed `run_matrix.py` usage
