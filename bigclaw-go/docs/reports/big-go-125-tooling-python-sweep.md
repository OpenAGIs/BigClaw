# BIG-GO-125 Tooling Python Sweep

## Scope

`BIG-GO-125` removes residual Python tooling guidance from active developer
surfaces without touching historical reports or archival evidence.

The sweep is limited to:

- root repository hygiene guidance in `README.md`
- active migration-shadow operator guidance in `docs/migration-shadow.md`
- regression coverage that keeps those active docs from drifting back to retired
  Python helper commands

## Active Replacement Paths

The active non-Python tooling surface for this lane is:

- `git diff --check`
- `make test`
- `make build`
- `go run ./cmd/bigclawctl automation migration shadow-compare`
- `go run ./cmd/bigclawctl automation migration shadow-matrix`
- `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)" README.md bigclaw-go/docs/migration-shadow.md`

## Validation Results

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned'`
Result: `ok  	bigclaw-go/internal/regression	(cached)`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)" README.md bigclaw-go/docs/migration-shadow.md`
Result: no matches, exit code `1`
