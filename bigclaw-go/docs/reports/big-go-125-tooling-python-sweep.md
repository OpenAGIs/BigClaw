# BIG-GO-125 Tooling Python Sweep

## Scope

`BIG-GO-125` removes residual Python tooling guidance from active developer
surfaces without touching historical reports or archival evidence.

The sweep is limited to:

- root repository hygiene guidance in `README.md`
- historical cutover handoff guidance in `docs/go-mainline-cutover-handoff.md`
- active migration-shadow operator guidance in `docs/migration-shadow.md`
- checked-in live shadow reviewer indexes in `docs/reports/live-shadow-index.md`
  and `docs/reports/live-shadow-index.json`
- checked-in live shadow summary artifacts in `docs/reports/live-shadow-summary.json`
  and `docs/reports/live-shadow-runs/20260313T085655Z/{README.md,summary.json}`
- checked-in live validation reviewer artifacts in `docs/reports/live-validation-index.json`,
  `docs/reports/live-validation-summary.json`, `docs/reports/ray-live-smoke-report.json`,
  and `docs/reports/live-validation-runs/20260316T140138Z/{ray-live-smoke-report.json,summary.json}`
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
- historical handoff docs now record retired Python validation as archival context
  rather than an active command
- checked-in reviewer indexes now use the same Go migration closeout commands as
  the canonical shadow workflow doc
- checked-in live shadow summaries and bundled reviewer README now use the same
  Go migration closeout commands as the canonical shadow workflow doc
- checked-in live validation reviewer artifacts now reflect the shell-native Ray
  smoke default already documented in `docs/e2e-validation.md`

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
- Compare URL: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1`

## Validation Results

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	0.196s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
Result: matches only the active Go CLI migration commands in `bigclaw-go/docs/migration-shadow.md`, `bigclaw-go/docs/reports/live-shadow-index.md`, `bigclaw-go/docs/reports/live-shadow-index.json`, `bigclaw-go/docs/reports/live-shadow-summary.json`, and `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/{README.md,summary.json}`; no retired Python command or root hygiene match remains

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	3.231s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`
Result: no matches, exit code `1`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`
Result: the live-validation index, summary, canonical Ray smoke report, and bundled Ray smoke report/summary all retain the shell-native `sh -c 'echo hello from ray'` entrypoint

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
Result: not logged into any GitHub hosts, exit code `1`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
Result: `[]`

Command: compare URL
Result: reachable without auth and shows the `BIG-GO-125` compare stack against `main`
