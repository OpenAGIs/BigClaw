# BIG-GO-1512 Validation

Date: 2026-04-06

## Scope

Issue: `BIG-GO-1512`

Title: `Refill: tests deletion-first sweep targeting actual Python file removal including conftest blockers`

This refill lane starts from the residual Python branch state carried by
`BIG-GO-1511`, where `tests/conftest.py` and the tranche-14 Python test files
were already absent but four repo-root ops Python wrappers still remained on
disk. The implemented sweep removes those physical `.py` files and adds
regression coverage so the Go entrypoints remain the only supported root ops
path.

## Delivered

- Deleted `scripts/ops/bigclaw_refill_queue.py`.
- Deleted `scripts/ops/bigclaw_workspace_bootstrap.py`.
- Deleted `scripts/ops/symphony_workspace_bootstrap.py`.
- Deleted `scripts/ops/symphony_workspace_validate.py`.
- Added `bigclaw-go/internal/regression/big_go_1512_root_ops_wrapper_removal_test.go`
  to keep the wrappers absent and keep the active docs on the Go replacements.
- Updated `README.md` and `docs/go-cli-script-migration-plan.md` so the root ops
  guidance points only at `bash scripts/ops/bigclawctl ...` entrypoints.

## Python File Count

Before:

```bash
find . -type f -name '*.py' | sort | wc -l
```

```text
23
```

After:

```bash
find . -type f -name '*.py' | sort | wc -l
```

```text
19
```

Deleted-file evidence:

```bash
git diff --name-status -- scripts/ops
```

```text
D	scripts/ops/bigclaw_refill_queue.py
D	scripts/ops/bigclaw_workspace_bootstrap.py
D	scripts/ops/symphony_workspace_bootstrap.py
D	scripts/ops/symphony_workspace_validate.py
```

## Validation

### Targeted regression coverage

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1512-clean/bigclaw-go && go test ./internal/regression -run 'Test(PythonTestTranche14Removed|BIGGO1512RootOpsPythonWrappersRemoved|BIGGO1512RootOpsDocsUseGoEntrypoints)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.708s
```

### Replacement entrypoint smoke

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1512-clean && bash scripts/ops/bigclawctl refill --help >/dev/null && bash scripts/ops/bigclawctl workspace bootstrap --help >/dev/null && bash scripts/ops/bigclawctl workspace validate --help >/dev/null
```

Result:

```text
success
```
