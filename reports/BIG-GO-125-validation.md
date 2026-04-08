# BIG-GO-125 Validation

Date: 2026-04-09

## Scope

Issue: `BIG-GO-125`

Title: `Residual tooling Python sweep H`

This lane removes residual Python tooling guidance from active developer-facing
docs and locks the replacement surface with targeted regression coverage.

The delivered change updates the root repository hygiene instructions and the
active migration-shadow helper commands so they point at the live Go and
shell-native entrypoints instead of retired Python helpers. It also converts a
remaining cutover handoff Python validation command into archival wording so it
no longer reads as an active workflow.

## Active Replacement Paths

- Repository hygiene: `git diff --check`
- Repository hygiene: `make test`
- Repository hygiene: `make build`
- Historical cutover validation note: archival-only wording in `docs/go-mainline-cutover-handoff.md`
- Migration shadow compare: `go run ./cmd/bigclawctl automation migration shadow-compare`
- Migration shadow matrix: `go run ./cmd/bigclawctl automation migration shadow-matrix`
- Migration shadow scorecard: `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
- Migration shadow bundle export: `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`

## Validation Commands

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned'`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<" README.md bigclaw-go/docs/migration-shadow.md docs/go-mainline-cutover-handoff.md`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh auth status`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && gh pr list --repo OpenAGIs/BigClaw --head BIG-GO-125 --json url,title,state,headRefName,baseRefName`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && curl -s 'https://api.github.com/repos/OpenAGIs/BigClaw/pulls?head=OpenAGIs:BIG-GO-125&state=all'`
- Public compare page: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1`

## Validation Results

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125/bigclaw-go && go test ./internal/regression -run 'TestRootScriptResidualSweepDocs|TestLiveShadowRuntimeDocsStayAligned'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.187s
```

### Residual active-doc reference search

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-125 && rg -n "pre-commit run --all-files|python3 scripts/migration/(shadow_compare|shadow_matrix|live_shadow_scorecard|export_live_shadow_bundle)|PYTHONPATH=src python3 - <<" README.md bigclaw-go/docs/migration-shadow.md docs/go-mainline-cutover-handoff.md
```

Result:

```text
no matches
exit code 1
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
- Published commits:
  - `ddfc4eb9` (`BIG-GO-125 sync latest published head`)
  - `f90388b0` (`BIG-GO-125 refresh published commit metadata`)
  - `6d2cc025` (`BIG-GO-125 sync workpad blocker`)
  - `2b7adcae` (`BIG-GO-125 add compare handoff link`)
  - `c1cb7bcf` (`BIG-GO-125 record public pr status`)
  - `68c44536` (`BIG-GO-125 sync blocker metadata`)
  - `feefe211` (`BIG-GO-125 retire handoff python validation guidance`)
  - `7827166b` (`BIG-GO-125 add validation artifacts`)
  - `4b6a6183` (`BIG-GO-125 refresh workpad`)
  - `aedbb76a` (`BIG-GO-125 remove residual python tooling guidance`)
- Push target: `origin/BIG-GO-125`
- Final tip: `tracked in git history after BIG-GO-125 final metadata sync`
- Compare URL: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-125?expand=1`

## Residual Risk

- Historical reports and archived regression fixtures still mention retired
  Python tooling as evidence, but the active developer-facing docs covered by
  this lane no longer present those commands as current workflow guidance.
- GitHub CLI authentication is unavailable in this workspace, so PR inspection
  or creation cannot be completed here even though the branch is already
  pushed to `origin/BIG-GO-125`.
- The public GitHub API currently reports no PR for head branch `BIG-GO-125`,
  so manual authenticated PR creation remains the only external remaining step.
- The public compare page is reachable and can be used as the manual PR
  creation handoff URL once authenticated access is available.
