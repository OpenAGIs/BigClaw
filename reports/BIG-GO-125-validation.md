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

## Active Replacement Paths

- Repository hygiene: `git diff --check`
- Repository hygiene: `make test`
- Repository hygiene: `make build`
- Historical cutover validation note: archival-only wording in `docs/go-mainline-cutover-handoff.md`
- Migration shadow compare: `go run ./cmd/bigclawctl automation migration shadow-compare`
- Migration shadow matrix: `go run ./cmd/bigclawctl automation migration shadow-matrix`
- Migration shadow scorecard: `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- Migration shadow bundle export: `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- Checked-in reviewer closeout index: `bigclaw-go/docs/reports/live-shadow-index.md`
- Checked-in reviewer closeout index JSON: `bigclaw-go/docs/reports/live-shadow-index.json`
- Checked-in reviewer summary JSON: `bigclaw-go/docs/reports/live-shadow-summary.json`
- Bundled reviewer run README: `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- Bundled reviewer run summary JSON: `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned|TestLiveShadowBundleSurface'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<|go run ./cmd/bigclawctl automation migration (live-shadow-scorecard|export-live-shadow-bundle)" README.md docs/go-mainline-cutover-handoff.md bigclaw-go/docs/migration-shadow.md bigclaw-go/docs/reports/live-shadow-index.md bigclaw-go/docs/reports/live-shadow-index.json bigclaw-go/docs/reports/live-shadow-summary.json bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-125 --json url,title,state,headRefName,baseRefName`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && git push origin BIG-GO-125`
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
- GitHub CLI authentication is unavailable in this workspace, so PR inspection
  or creation cannot be completed here even though the branch is already
  pushed to `origin/BIG-GO-125`.
- The public GitHub API currently reports no PR for head branch `BIG-GO-125`,
  so manual authenticated PR creation remains the only external remaining step.
- The public compare page is reachable and can be used as the manual PR
  creation handoff URL once authenticated access is available.
