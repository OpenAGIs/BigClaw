# BIG-GO-1052 PR Draft

## Title

`BIG-GO-1052: lock e2e tranche 1 to Go-only entrypoints`

## Summary

- add executable regression coverage that keeps `bigclaw-go/scripts/e2e/` free of tranche-1 Python helpers
- add executable regression coverage that keeps active README/workflow/e2e-guide surfaces on Go-only entrypoints
- align Go-facing e2e operator docs with the active `bigclawctl automation e2e ...` commands
- add repo-native validation, closeout, and machine-readable status artifacts for this lane

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go/scripts/e2e -name '*.py' | wc -l`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go test ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052/bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1052 && git diff --check`

## Artifacts

- validation: `reports/BIG-GO-1052-validation.md`
- closeout: `reports/BIG-GO-1052-closeout.md`
- status: `reports/BIG-GO-1052-status.json`

## Reviewer Note

The target tranche-1 `bigclaw-go/scripts/e2e/*.py` files were already absent in the
starting checkout, so this lane hardens the Go-only contract and removes residual
entrypoint drift rather than performing a fresh in-branch Python-file deletion.
