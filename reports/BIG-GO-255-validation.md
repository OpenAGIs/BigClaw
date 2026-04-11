# BIG-GO-255 Validation

Date: 2026-04-12

## Scope

Issue: `BIG-GO-255`

Title: `Residual tooling Python sweep U`

This lane removes residual Python tooling from the checked-in fake `go` helper
used by `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`.

The branch baseline was already at a repository-wide physical Python file count
of `0`, so the delivered change converts a checked-in test helper from Python
to shell and adds regression evidence to hold that line.

## Remaining Python Asset Inventory

- Repository-wide physical `.py` files: `none`
- Residual tooling helper in `automation_e2e_bundle_commands_test.go`: migrated to shell

## Replacement Surface

- Issue regression guard: `bigclaw-go/internal/regression/big_go_255_zero_python_guard_test.go`
- Updated tooling helper: `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
- Lane report: `bigclaw-go/docs/reports/big-go-255-python-asset-sweep.md`

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-255 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `rg -n '#!/bin/sh|find_arg\\(\\)|unexpected go args: \\$\\*' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
- `! rg -n '#!/usr/bin/env python3|import json, pathlib, sys|raise SystemExit' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO255(RepositoryHasNoPythonFiles|AutomationBundleTestStubStaysShellOnly|LaneReportCapturesSweepState)$'`

## Validation Results

### Repository Python inventory

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-255 -path '*/.git' -prune -o -name '*.py' -type f -print | sort
```

Result:

```text
none
```

### Shell-only tooling helper markers

Command:

```bash
rg -n -e '#!/bin/sh' -e 'find_arg\(\)' -e 'unexpected go args: %s\\n' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go
```

Result:

```text
278:	stubGo := `#!/bin/sh
288:find_arg() {
358:printf 'unexpected go args: %s\n' "$*" >&2
```

### Retired Python helper markers

Command:

```bash
! rg -n '#!/usr/bin/env python3|import json, pathlib, sys|raise SystemExit' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go
```

Result:

```text
success
```

### Targeted command test

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	0.596s
```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO255(RepositoryHasNoPythonFiles|AutomationBundleTestStubStaysShellOnly|LaneReportCapturesSweepState)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.187s
```

## Residual Risk

- The live checkout already had a repository-wide physical Python file count of
  `0`, so this issue removes residual Python tooling from a checked-in test
  helper instead of deleting a `.py` file.
