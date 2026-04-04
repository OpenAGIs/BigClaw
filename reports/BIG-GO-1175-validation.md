# BIG-GO-1175 Validation

Date: 2026-04-04

## Scope

Issue: `BIG-GO-1175`

Title: `Go-only sweep lane 1175: remove remaining Python assets batch 5/10`

This lane covers the retained repo-root helper path after the repository had
already reached a zero-`.py` baseline. The work keeps the lane scoped to
concrete replacement evidence by asserting that `scripts/dev_bootstrap.sh`
stays a shell/Go validation helper and by documenting the supported Go-native
replacements for the retired root Python entrypoints.

## Delivered

- Added regression coverage in
  `bigclaw-go/internal/regression/root_script_residual_sweep_test.go` for the
  retained `scripts/dev_bootstrap.sh` helper so it must keep using `go test`,
  `bash scripts/ops/bigclawctl dev-smoke`, and the optional
  `bash scripts/ops/bigclawctl legacy-python compile-check --json` path.
- Updated `docs/go-cli-script-migration-plan.md` and
  `bigclaw-go/docs/go-cli-script-migration.md` to record `BIG-GO-1175` as
  follow-on replacement evidence for the root helper sweep area.
- Refreshed `.symphony/workpad.md` with the issue plan, acceptance criteria,
  validation commands, and exact command results.

## Validation

### Python count baseline

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1175 -name '*.py' | wc -l
```

Result:

```text
0
```

### Targeted regression coverage

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1175/bigclaw-go && go test ./internal/regression -run 'Test(RootScriptResidualSweep|RootScriptResidualSweepDocs|BIGGO1175DevBootstrapStaysGoOnly|BIGGO1175DocsRecordDevBootstrapReplacementEvidence)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.470s
```

### Root helper smoke path

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1175 && bash scripts/ops/bigclawctl dev-smoke
```

Result:

```text
smoke_ok local
```

## Git

- Commit: `5eb5a487` (`BIG-GO-1175 harden root helper go-only evidence`)
- Push: `git push -u origin BIG-GO-1175` -> success

## Residual Risk

- This workspace already started at `find . -name '*.py' | wc -l = 0`, so the
  lane could only commit concrete replacement evidence and regression hardening
  rather than numerically reduce the Python count further.
