# BIG-GO-1141 Validation

Date: 2026-04-04

## Scope

Issue: `BIG-GO-1141`

Title: `src/bigclaw residual sweep`

This lane hardens the already-materialized Go-only state for the remaining
`src/bigclaw` candidate paths named by `BIG-GO-1141`, updates stale repo
guidance that still implied a live `src/bigclaw` tree existed, and adds
regression coverage so those deleted Python modules do not silently return.

## Delivered

- added `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go`
  to enforce continued absence of:
  - `src/bigclaw/__init__.py`
  - `src/bigclaw/__main__.py`
  - `src/bigclaw/audit_events.py`
  - `src/bigclaw/collaboration.py`
  - `src/bigclaw/console_ia.py`
  - `src/bigclaw/design_system.py`
  - `src/bigclaw/evaluation.py`
  - `src/bigclaw/run_detail.py`
  - `src/bigclaw/runtime.py`
- aligned `README.md` with the current repo state so it no longer claims
  `src/bigclaw` is still an included migration tree
- aligned `workflow.md` with the current repo state so it describes
  `bigclaw-go` as the sole implementation mainline and treats any future Python
  artifact as an explicitly justified exception
- recorded the exact acceptance and validation evidence in `.symphony/workpad.md`

## Validation

### Python file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141 -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
git -C /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141 ls-tree -r --name-only HEAD | rg '\.py$'
```

Result:

```text
exit code 1 with no tracked Python files
```

Note: the repo-wide Python baseline was already `0` before this lane’s edits, so
`BIG-GO-1141` hardens the deletion boundary and retires stale narrative residue
rather than lowering the count further from the current workspace baseline.

### Targeted regression checks

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141/bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche17
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.682s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141/bigclaw-go && go test ./internal/regression
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.508s
```

### Stale guidance checks

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141 && rg -n --fixed-strings 'pending staged migration to Go' README.md workflow.md
```

Result:

```text
exit code 1
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141 && rg -n --fixed-strings 'this repo currently carries no live `src/bigclaw` tree' workflow.md
```

Result:

```text
73:- Treat `bigclaw-go` as the sole implementation mainline for new development; this repo currently carries no live `src/bigclaw` tree, so any future Python artifact must be migration-only and explicitly justified.
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1141 && rg -n --fixed-strings 'retired `src/bigclaw` Python foundations' README.md
```

Result:

```text
18:- retired `src/bigclaw` Python foundations: the historical runtime, reporting, and operator-console surfaces have been materialized into `bigclaw-go/internal/*`, and regression tests now keep the deleted Python lane absent
```

## Commit And Push

- Commit: `c985bcd85531766fd105edbdf0e7ffe8443bf968`
- Message: `BIG-GO-1141 harden residual src/bigclaw sweep`
- Branch: `symphony/BIG-GO-1141`
- Push: `git push -u origin symphony/BIG-GO-1141` succeeded

## Residual Risk

- This lane cannot reduce the repo-wide `.py` count numerically because the
  checkout already started at `0`.
- No additional in-repo Python asset remains under the `BIG-GO-1141` candidate
  surface in this workspace; follow-up issue state, if any, is external to the
  worktree contents.
