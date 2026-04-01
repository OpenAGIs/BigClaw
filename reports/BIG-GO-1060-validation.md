# BIG-GO-1060 Validation Report

Date: 2026-04-01

## Scope

Issue: `BIG-GO-1060`

Title: `Go-replacement AD: remove residual Python entrypoints tracked in README/workflow`

This lane removes the last operator-facing Python wrappers still shipped under `scripts/ops/`
for refill and workspace flows, updates the tracked repo surfaces to point at
`bash scripts/ops/bigclawctl ...`, and adds regression coverage so those deleted entrypoints do
not return.

## Delivered

- Deleted these residual Python operator entrypoints:
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- Removed the now-dead Go wrapper helper surface:
  - `bigclaw-go/internal/legacyshim/wrappers.go`
  - `bigclaw-go/internal/legacyshim/wrappers_test.go`
- Updated tracked operator references in:
  - `README.md`
  - `.github/workflows/ci.yml`
  - `docs/go-cli-script-migration-plan.md`
- Added regression coverage in:
  - `bigclaw-go/internal/regression/operator_entrypoint_cutover_test.go`
- Kept workflow and hook entrypoints Go-first through:
  - `workflow.md`
  - `.githooks/post-commit`
  - `.githooks/post-rewrite`

## Validation

### Python file count drop

Command:

```bash
git ls-tree -r --name-only HEAD^ | rg '\.py$' | wc -l
```

Result:

```text
45
```

Command:

```bash
git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l
```

Result:

```text
41
```

### Targeted regression tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1060/bigclaw-go && go test ./internal/regression -run 'TestResidualPythonOperatorEntrypointsStayDeleted|TestTrackedOperatorSurfacesStayGoOnly|TestE2EMigrationDocListsOnlyActiveEntrypoints'
```

Result:

```text
ok  	bigclaw-go/internal/regression	(cached)
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1060/bigclaw-go && go test ./internal/legacyshim
```

Result:

```text
ok  	bigclaw-go/internal/legacyshim	(cached)
```

### Go entrypoint checks

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1060/scripts/ops/bigclawctl refill --help | sed -n '1,20p'
```

Result:

```text
usage: bigclawctl refill [flags]
       bigclawctl refill seed [flags]
  -apply
    	apply
  -interval int
    	interval (default 20)
  -local-issues string
    	local issue store path
  -markdown string
    	human-readable queue markdown path (default "docs/parallel-refill-queue.md")
  -queue string
    	queue path (default "docs/parallel-refill-queue.json")
  -refresh-url string
    	refresh url
  -repo string
    	repo root (default "..")
  -sync-queue-status
    	sync queue issue statuses and recent batches from local tracker metadata (local backend only; requires --apply to write)
  -target-in-progress int
    	override target (default -1)
```

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1060/scripts/ops/bigclawctl workspace validate --help | sed -n '1,20p'
```

Result:

```text
usage: bigclawctl workspace validate [flags]
  -cache-base string
    	cache base (default "~/.cache/symphony/repos")
  -cache-key string
    	cache key
  -cache-root string
    	cache root
  -cleanup
    	cleanup (default true)
  -default-branch string
    	default branch (default "main")
  -issue string
    	issue identifier
  -issues string
    	comma-separated issues
  -json
    	json
  -repo string
    	repo root (default "..")
  -repo-url string
```

### Stale reference scan

Command:

```bash
rg -n "python3 scripts/ops/bigclaw_refill_queue\.py|scripts/ops/\*workspace\*\.py|python3 scripts/ops/symphony_workspace_validate\.py|python3 scripts/ops/bigclaw_workspace_bootstrap\.py|python3 scripts/ops/symphony_workspace_bootstrap\.py" README.md .github/workflows/ci.yml workflow.md .githooks docs/go-cli-script-migration-plan.md
```

Result: exit code `1` with no matches.

### Branch sync check

Command:

```bash
git status --short --branch && git rev-parse HEAD && git rev-parse origin/symphony/BIG-GO-1060 && git log -1 --stat --oneline
```

Result:

```text
## symphony/BIG-GO-1060...origin/symphony/BIG-GO-1060
c8f270b1fa5e58ae413561d29aae983a8d7ab55e
c8f270b1fa5e58ae413561d29aae983a8d7ab55e
c8f270b1 BIG-GO-1060 remove residual Python operator entrypoints
 .github/workflows/ci.yml                           |  10 ++
 .symphony/workpad.md                               |  29 +++--
 README.md                                          |  12 ++-
 bigclaw-go/internal/legacyshim/wrappers.go         |  77 -------------
 bigclaw-go/internal/legacyshim/wrappers_test.go    | 107 ------------------
 .../regression/operator_entrypoint_cutover_test.go | 119 +++++++++++++++++++++
 docs/go-cli-script-migration-plan.md               |  30 ++----
 scripts/ops/bigclaw_refill_queue.py                |  20 ----
 scripts/ops/bigclaw_workspace_bootstrap.py         |  26 -----
 scripts/ops/symphony_workspace_bootstrap.py        |  20 ----
 scripts/ops/symphony_workspace_validate.py         |  24 -----
 11 files changed, 161 insertions(+), 313 deletions(-)
```

## Notes

- `BIG-GO-1060` is not currently mirrored in `local-issues.json`, so this closeout records the
  in-repo implementation and validation state without a local tracker transition.
- The follow-up report-only commit for these artifacts is intentionally separate from the code
  migration commit so the file-count delta remains attributable to `c8f270b1fa5e58ae413561d29aae983a8d7ab55e`.
