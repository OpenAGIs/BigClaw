# BIG-GO-255 Workpad

## Plan

1. Remove the residual Python-based fake `go` helper embedded in
   `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` by
   replacing it with a shell-only stub.
2. Add issue-scoped regression evidence so this tooling surface does not drift
   back to Python:
   - `bigclaw-go/internal/regression/big_go_255_zero_python_guard_test.go`
   - `bigclaw-go/docs/reports/big-go-255-python-asset-sweep.md`
   - `reports/BIG-GO-255-validation.md`
3. Run targeted validation for the stub migration and regression guard, then
   commit and push branch `BIG-GO-255`.

## Acceptance

- The checked-in fake `go` helper used by the E2E bundle command test is
  shell-only rather than Python-based.
- Issue-scoped regression evidence captures the zero-Python baseline and the
  shell-only test helper expectation.
- Validation commands and exact results are recorded.
- Changes are committed and pushed to `origin/BIG-GO-255`.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-255 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `rg -n '#!/bin/sh|find_arg\\(\\)|unexpected go args: \\$\\*' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
- `! rg -n '#!/usr/bin/env python3|import json, pathlib, sys|raise SystemExit' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO255(RepositoryHasNoPythonFiles|AutomationBundleTestStubStaysShellOnly|LaneReportCapturesSweepState)$'`

## Results

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-255 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `rg -n -e '#!/bin/sh' -e 'find_arg\\(\\)' -e 'unexpected go args: %s\\\\n' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
  Result:
  - `278:	stubGo := \`#!/bin/sh`
  - `288:find_arg() {`
  - `358:printf 'unexpected go args: %s\n' \"$*\" >&2`
- `! rg -n '#!/usr/bin/env python3|import json, pathlib, sys|raise SystemExit' /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`
  Result: success.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	0.596s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-255/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO255(RepositoryHasNoPythonFiles|AutomationBundleTestStubStaysShellOnly|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.187s`

## Notes

- Branch baseline before the lane change: `62626d62`.
- The repository already had no physical `.py` files, so this issue removed
  residual Python tooling from a checked-in test helper rather than deleting a
  tracked `.py` asset.
