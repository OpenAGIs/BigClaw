# BIG-GO-125 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-125`

Title: `Residual tooling Python sweep H`

This lane removes residual Python tooling guidance from active developer-facing
docs and locks the replacement surface with targeted regression coverage.

The delivered change refreshes the checked-in live shadow reviewer indexes,
summary artifacts, and bundled reviewer README so their workflow closeout
commands point at the live Go CLI entrypoints instead of retired Python
helpers, and it expands targeted regression coverage to guard those
reviewer-facing files alongside the canonical migration-shadow doc.
The lane also refreshes the checked-in live-validation reviewer artifacts so
their Ray smoke evidence now matches the shell-native default already
documented in `docs/e2e-validation.md` instead of advertising the retired
inline-Python smoke entrypoint. The final follow-up also normalizes a skipped
Ray bundle so reviewer-facing docs do not point at Ray smoke report artifacts
that were never produced. A second follow-up then swept adjacent checked-in Ray
reviewer evidence that still embedded inline-Python entrypoints outside the
main live-validation bundle. A third follow-up then removed Python-specific
bootstrap-template wording from the active Symphony workspace bootstrap guide.

## Active Replacement Paths

- Repository hygiene: `git diff --check`
- Repository hygiene: `make test`
- Repository hygiene: `make build`
- Historical cutover validation note: archival-only wording in `docs/go-mainline-cutover-handoff.md`
- Bootstrap template guidance: `docs/symphony-repo-bootstrap-template.md`
- Migration shadow compare: `go run ./cmd/bigclawctl automation migration shadow-compare`
- Migration shadow matrix: `go run ./cmd/bigclawctl automation migration shadow-matrix`
- Migration shadow scorecard: `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- Migration shadow bundle export: `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- Checked-in reviewer closeout index: `bigclaw-go/docs/reports/live-shadow-index.md`
- Checked-in reviewer closeout index JSON: `bigclaw-go/docs/reports/live-shadow-index.json`
- Checked-in reviewer summary JSON: `bigclaw-go/docs/reports/live-shadow-summary.json`
- Bundled reviewer run README: `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- Bundled reviewer run summary JSON: `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- Live validation index JSON: `bigclaw-go/docs/reports/live-validation-index.json`
- Live validation summary JSON: `bigclaw-go/docs/reports/live-validation-summary.json`
- Canonical Ray smoke report: `bigclaw-go/docs/reports/ray-live-smoke-report.json`
- Ray jobs snapshot: `bigclaw-go/docs/reports/ray-live-jobs.json`
- Mixed workload matrix report: `bigclaw-go/docs/reports/mixed-workload-matrix-report.json`
- Previous bundled Ray smoke report: `bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json`
- Previous bundled live validation summary: `bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- Skipped bundled live validation README: `bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md`
- Skipped bundled live validation summary: `bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json`
- Bundled Ray smoke report: `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json`
- Bundled live validation summary: `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-125 --json url,title,state,headRefName,baseRefName`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && git push origin BIG-GO-125`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'ray-live-smoke-report.json|executor disabled; no Ray smoke report was produced for this bundle' bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c|sh -c '\''echo (gpu via ray|required ray|ray driver snapshot)'\''' bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/mixed-workload-matrix-report.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i '\bpython\b|workspace_bootstrap\.py|workspace_bootstrap_cli\.py' docs/symphony-repo-bootstrap-template.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- Public compare page: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1`

## Validation Results

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.196s
```

### Residual active-doc reference search

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json
```

Result:

```text
bigclaw-go/docs/reports/live-shadow-index.json:93:      "cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --shadow-compare-report ./docs/reports/shadow-compare-report.json --shadow-matrix-report ./docs/reports/shadow-matrix-report.json --output ./docs/reports/live-shadow-mirror-scorecard.json",
bigclaw-go/docs/reports/live-shadow-index.json:94:      "cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
bigclaw-go/docs/reports/live-shadow-summary.json:92:    "cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --shadow-compare-report ./docs/reports/shadow-compare-report.json --shadow-matrix-report ./docs/reports/shadow-matrix-report.json --output ./docs/reports/live-shadow-mirror-scorecard.json",
bigclaw-go/docs/reports/live-shadow-summary.json:93:    "cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md:42:- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --shadow-compare-report ./docs/reports/shadow-compare-report.json --shadow-matrix-report ./docs/reports/shadow-matrix-report.json --output ./docs/reports/live-shadow-mirror-scorecard.json`
bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md:43:- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json:92:    "cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --shadow-compare-report ./docs/reports/shadow-compare-report.json --shadow-matrix-report ./docs/reports/shadow-matrix-report.json --output ./docs/reports/live-shadow-mirror-scorecard.json",
bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json:93:    "cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle",
bigclaw-go/docs/reports/live-shadow-index.md:42:- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --shadow-compare-report ./docs/reports/shadow-compare-report.json --shadow-matrix-report ./docs/reports/shadow-matrix-report.json --output ./docs/reports/live-shadow-mirror-scorecard.json`
bigclaw-go/docs/reports/live-shadow-index.md:43:- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
bigclaw-go/docs/migration-shadow.md:44:go run ./cmd/bigclawctl automation migration live-shadow-scorecard \
bigclaw-go/docs/migration-shadow.md:59:go run ./cmd/bigclawctl automation migration export-live-shadow-bundle
```

Observed result:

```text
Only the active Go CLI migration commands matched. No retired Python migration helper, retired root hygiene, or heredoc-based Python validation command matched in the searched files, including the bundled run summary surfaces.
```

### Live validation reviewer artifact regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.255s
```

### Ray smoke inline-Python scan

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c \"print\\(' bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json
```

Result:

```text
no matches
exit code 1
```

### Ray smoke shell-native scan

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "sh -c 'echo hello from ray'" bigclaw-go/docs/reports/live-validation-index.json bigclaw-go/docs/reports/live-validation-summary.json bigclaw-go/docs/reports/ray-live-smoke-report.json bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json
```

Result:

```text
The live-validation index, summary, canonical Ray smoke report, Ray jobs snapshot, and both recent bundled Ray smoke report/summary surfaces all matched the shell-native entrypoint.
```

### Skipped Ray bundle normalization scan

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n 'ray-live-smoke-report.json|executor disabled; no Ray smoke report was produced for this bundle' bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json
```

Result:

```text
bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/README.md:36:- Reason: `executor disabled; no Ray smoke report was produced for this bundle`
bigclaw-go/docs/reports/live-validation-runs/20260314T163430Z/summary.json:421:    "reason": "executor disabled; no Ray smoke report was produced for this bundle"
```

Observed result:

```text
Only the disabled-lane reason matched. No skipped-bundle ray-live-smoke-report.json reference remains in the checked-in reviewer bundle.
```

### Adjacent Ray evidence regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.211s
```

### Adjacent Ray evidence scan

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i 'python -c|sh -c '\''echo (gpu via ray|required ray|ray driver snapshot)'\''' bigclaw-go/docs/reports/ray-live-jobs.json bigclaw-go/docs/reports/mixed-workload-matrix-report.json
```

Result:

```text
bigclaw-go/docs/reports/ray-live-jobs.json:78:    "entrypoint": "sh -c 'echo ray driver snapshot'",
bigclaw-go/docs/reports/mixed-workload-matrix-report.json:215:            "message": "ray job bigclaw-mixed-gpu-1773395066 succeeded: 2026-03-13 02:44:31,429\tINFO job_manager.py:579 -- Runtime env is setting up.\nRunning entrypoint for job bigclaw-mixed-gpu-1773395066: sh -c 'echo gpu via ray'\ngpu via ray"
bigclaw-go/docs/reports/mixed-workload-matrix-report.json:357:            "message": "ray job bigclaw-mixed-required-ray-1773395066 succeeded: 2026-03-13 02:44:38,415\tINFO job_manager.py:579 -- Runtime env is setting up.\nRunning entrypoint for job bigclaw-mixed-required-ray-1773395066: sh -c 'echo required ray'\nrequired ray"
```

Observed result:

```text
No python -c match remains. Only the shell-native ray driver snapshot, gpu via ray, and required ray strings matched in the checked-in reviewer artifacts.
```

### Bootstrap template scan

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n -i '\bpython\b|workspace_bootstrap\.py|workspace_bootstrap_cli\.py' docs/symphony-repo-bootstrap-template.md
```

Result:

```text
no matches
exit code 1
```

### Bootstrap template regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveValidationSummaryStaysAligned|TestParallelValidationMatrixDocsStayAligned|TestLiveValidationIndexStaysAligned|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.183s
```

### GitHub publication visibility

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status
```

Result:

```text
You are not logged into any GitHub hosts. To log in, run: gh auth login
exit code 1
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-125 --json url,title,state,headRefName,baseRefName
```

Result:

```text
To get started with GitHub CLI, please run: gh auth login
Alternatively, populate the GH_TOKEN environment variable with a GitHub API authentication token.
exit code 4
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'
```

Result:

```json
[

]
```

### Branch publication

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && git push origin BIG-GO-125
```

Result:

```text
The branch was pushed successfully to `origin/BIG-GO-125` after the final metadata sync.
See `git log --oneline --grep 'BIG-GO-125'` for the exact published tip sequence.
```

Public compare page:

```text
https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1
```

Observed result:

```text
Reachable without auth; shows the BIG-GO-125 compare stack against main and can be used as the manual PR handoff link.
```

## Git

- Branch: `BIG-GO-125`
- Published commits: see `git log --oneline --grep 'BIG-GO-125'`
- Push target: `origin/BIG-GO-125`
- Final tip: tracked in git history after the final BIG-GO-125 metadata sync
- Compare URL: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1`

## Residual Risk

- Historical reports and archived regression fixtures still mention retired
  Python tooling as evidence, but the active developer-facing docs covered by
  this lane no longer present those commands as current workflow guidance.
- Other archived or generated reports outside the touched live-shadow summary
  and reviewer-index surfaces may still contain historical Python command
  strings and remain
  intentionally out of scope for this lane.
- Other archived raw runtime evidence such as historical stdout logs, audit
  streams, and older validation bundles may still include pre-migration Python
  command strings and remain intentionally out of scope for this lane.
- GitHub CLI authentication is unavailable in this workspace, so PR inspection
  or creation cannot be completed here even though the branch is already
  pushed to `origin/BIG-GO-125`.
- The public GitHub API currently reports no PR for head branch `BIG-GO-125`,
  so manual authenticated PR creation remains the only external remaining step.
- The public compare page is reachable and can be used as the manual PR
  creation handoff URL once authenticated access is available.
