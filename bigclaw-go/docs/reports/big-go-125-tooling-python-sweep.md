# BIG-GO-125 Tooling Python Sweep

## Scope

`BIG-GO-125` removes residual Python tooling guidance from active developer
surfaces without touching historical reports or archival evidence.

The sweep is limited to:

- root repository hygiene guidance in `README.md`
- historical cutover handoff guidance in `docs/go-mainline-cutover-handoff.md`
- bootstrap operator template guidance in `docs/symphony-repo-bootstrap-template.md`
- refill planning guidance in `docs/parallel-refill-queue.md`
- active migration-shadow operator guidance in `docs/migration-shadow.md`
- checked-in live shadow reviewer indexes in `docs/reports/live-shadow-index.md`
  and `docs/reports/live-shadow-index.json`
- checked-in live shadow summary artifacts in `docs/reports/live-shadow-summary.json`
  and `docs/reports/live-shadow-runs/20260313T085655Z/{README.md,summary.json}`
- checked-in live validation reviewer artifacts in `docs/reports/live-validation-index.json`,
  `docs/reports/live-validation-summary.json`, `docs/reports/ray-live-smoke-report.json`,
  `docs/reports/ray-live-jobs.json`, `docs/reports/mixed-workload-matrix-report.json`, and
  `docs/reports/live-validation-runs/{20260314T163430Z,20260314T164647Z,20260316T140138Z}/...`
- regression coverage that keeps those active docs from drifting back to retired
  Python helper commands or misleading skipped-lane Ray report links

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
- skipped live-validation bundles now explain that disabled Ray lanes did not
  produce Ray smoke report artifacts instead of linking to nonexistent evidence
- adjacent checked-in Ray reviewer evidence now uses shell-native placeholders
  instead of lingering inline-Python entrypoints outside the main smoke bundle
- bootstrap workspace template guidance now describes repo-specific implementation
  paths generically instead of framing the active path around Python shims
- refill planning guidance now describes legacy migration-only paths generically
  instead of framing the default repo workflow around Python-specific paths

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'ray-live-smoke-report.json|executor disabled; no Ray smoke report was produced for this bundle' bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c|sh -c '\''echo (gpu via ray|required ray|ray driver snapshot)'\''' bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/mixed-workload-matrix-report.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i '\bpython\b|workspace_bootstrap\.py|workspace_bootstrap_cli\.py' docs/symphony-repo-bootstrap-template.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'Python paths are migration-only unless explicitly marked otherwise|legacy migration-only paths stay out of the default developer workflow unless explicitly marked otherwise' docs/parallel-refill-queue.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
- Compare URL: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1`

## Validation Results

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	0.196s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
Result: matches only the active Go CLI migration commands in `bigclaw-go/docs/migration-shadow.md`, `bigclaw-go/docs/reports/live-shadow-index.md`, `bigclaw-go/docs/reports/live-shadow-index.json`, `bigclaw-go/docs/reports/live-shadow-summary.json`, and `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/{README.md,summary.json}`; no retired Python command or root hygiene match remains

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	0.255s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
Result: no matches, exit code `1`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
Result: the live-validation index, summary, canonical Ray smoke report, Ray jobs snapshot, and both recent bundled Ray smoke report/summary surfaces all retain the shell-native `sh -c 'echo hello from ray'` entrypoint

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'ray-live-smoke-report.json|executor disabled; no Ray smoke report was produced for this bundle' bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json`
Result: only the new disabled-lane reason matched in `README.md:36` and `summary.json:421`; no skipped-bundle `ray-live-smoke-report.json` reference remains

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	0.211s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c|sh -c '\''echo (gpu via ray|required ray|ray driver snapshot)'\''' bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/mixed-workload-matrix-report.json`
Result: no `python -c` match remains; only the shell-native `ray driver snapshot`, `gpu via ray`, and `required ray` strings matched in the checked-in reviewer artifacts

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i '\bpython\b|workspace_bootstrap\.py|workspace_bootstrap_cli\.py' docs/symphony-repo-bootstrap-template.md`
Result: no matches, exit code `1`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	0.183s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'Python paths are migration-only unless explicitly marked otherwise|legacy migration-only paths stay out of the default developer workflow unless explicitly marked otherwise' docs/parallel-refill-queue.md`
Result: only the new Go-first legacy wording matched at `docs/parallel-refill-queue.md:45`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
Result: `ok  	bigclaw-go/internal/regression	3.237s`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
Result: not logged into any GitHub hosts, exit code `1`

Command: `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
Result: `[]`

Command: compare URL
Result: reachable without auth and shows the `BIG-GO-125` compare stack against `main`
