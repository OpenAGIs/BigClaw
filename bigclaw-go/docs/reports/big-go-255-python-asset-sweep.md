# BIG-GO-255 Python Asset Sweep

## Scope

`BIG-GO-255` (`Residual tooling Python sweep U`) removes a residual
Python-based tooling helper that was still checked in inside
`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

This lane therefore hardens the Go-only baseline by converting the checked-in
test helper itself from Python to shell rather than deleting a physical `.py`
file in this checkout.

## Active Tooling Replacement

- `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` no longer embeds a Python fake `go` helper
- The fake `go` helper now starts with `#!/bin/sh`
- The fake `go` helper now uses `find_arg()` shell parsing instead of Python argument decoding

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `rg -n '#!/bin/sh|find_arg\\(\\)|unexpected go args: \\$\\*' bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
  Result: the checked-in stub matched the expected shell-only helper markers.
- `! rg -n '#!/usr/bin/env python3|import json, pathlib, sys|raise SystemExit' bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
  Result: success; the retired Python helper markers were absent.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`
  Result: targeted command test passed with the shell stub.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO255(RepositoryHasNoPythonFiles|AutomationBundleTestStubStaysShellOnly|LaneReportCapturesSweepState)$'`
  Result: targeted regression guard passed.

## Residual Risk

- The repository-wide physical `.py` count was already `0`, so `BIG-GO-255`
  removes residual Python tooling from a checked-in test helper rather than
  lowering a nonzero Python file count.
