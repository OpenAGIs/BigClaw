# BIG-GO-264 Validation

## Summary

Issue: `BIG-GO-264`

Title: `Residual scripts Python sweep V`

This lane removes residual Python command defaults from the mixed-workload CLI
helper and adds regression coverage plus sweep evidence for that helper
surface.

## Validation Commands

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-264 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go/cmd /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationMixedWorkloadMatrixBuildsReport|TestDefaultMixedWorkloadTasksUseNoPythonEntrypoints'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO264(MixedWorkloadHelperAvoidsPythonEntrypoints|LaneReportCapturesMixedWorkloadHelperSweep)$'`

## Results

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-264 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output.
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go/cmd /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
  Result: no output.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationMixedWorkloadMatrixBuildsReport|TestDefaultMixedWorkloadTasksUseNoPythonEntrypoints'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	0.351s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-264/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO264(MixedWorkloadHelperAvoidsPythonEntrypoints|LaneReportCapturesMixedWorkloadHelperSweep)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.177s`
