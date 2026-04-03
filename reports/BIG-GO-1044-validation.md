# BIG-GO-1044 Validation

Date: 2026-04-03

## Scope

Issue: `BIG-GO-1044`

Title: `Go-replacement N: tests top-level purge tranche 2`

This lane removes the residual UI migration-only Python files
`src/bigclaw/console_ia.py`, `src/bigclaw/design_system.py`, and
`src/bigclaw/ui_review.py`, repoints the release-control planning evidence to
Go-native files only, and adds regression coverage so the deleted Python tranche
stays absent.

## Delivered

- deleted:
  - `src/bigclaw/console_ia.py`
  - `src/bigclaw/design_system.py`
  - `src/bigclaw/ui_review.py`
- updated planning manifests so the active release-control candidate now points
  at Go-native validation and evidence only:
  - `bigclaw-go/internal/planning/planning.go`
  - `bigclaw-go/internal/planning/planning_test.go`
  - `src/bigclaw/planning.py`
- added regression coverage:
  - `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`
- removed stale live inventory references from:
  - `docs/go-mainline-cutover-issue-pack.md`

## Validation

### Python and Go file counts

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044 -name '*.py' | wc -l
```

Result:

```text
14
```

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044 -name '*.go' | wc -l
```

Result:

```text
323
```

Note: repo `.py` count dropped from `17` to `14` on this branch.

### Targeted Go tests

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go && go test ./internal/planning -run 'Test(CandidateBacklogRoundTripPreservesManifestShape|CandidateBacklogRanksReadyItemsAheadOfBlockedWork|EntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence|EntryGateHoldsWhenV2BaselineIsMissingOrNotReady|RenderCandidateBacklogReportSummarizesBacklogAndGateFindings|BuildV3CandidateBacklogMatchesIssuePlanTraceability)' -count=1
```

Result:

```text
ok  	bigclaw-go/internal/planning	0.902s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go && go test ./internal/regression -run 'TestTopLevelModulePurgeTranche15' -count=1
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.105s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go && go test ./internal/planning ./internal/regression -count=1
```

Result:

```text
ok  	bigclaw-go/internal/planning	0.489s
ok  	bigclaw-go/internal/regression	0.913s
```

### Formatting

Command:

```bash
gofmt -w /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go/internal/planning/planning.go /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go/internal/planning/planning_test.go /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go
```

Result: exit code `0`

### Live reference sweep

Command:

```bash
rg -n "src/bigclaw/(console_ia|design_system|ui_review)\.py|tests/test_(console_ia|design_system|ui_review)\.py" /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/README.md /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/docs /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/bigclaw-go /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/src /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044/scripts /Users/openagi/code/bigclaw-workspaces/BIG-GO-1044 -g '*.md' -g '*.go' -g '*.sh' -g '*.json' -g '*.yml'
```

Result:

- remaining matches are only:
  - historical validation reports under `reports/`
  - the intentional regression assertions in `bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`
  - the intentional deleted-target checks in `bigclaw-go/internal/planning/planning_test.go`

## Commit And Push

- Commit: `b51e3a94`
- Message: `BIG-GO-1044: purge UI python tranche`
- Follow-up commit: `e39d108f`
- Follow-up message: `BIG-GO-1044: align cutover inventory`
- Push: `git push` succeeded for branch `symphony/BIG-GO-1044`

## Residual Risk

- `BIG-GO-1044` is not present in `local-issues.json`, so there is no local
  tracker record in this checkout to mark `Done`.
- Historical reports intentionally retain old Python-path references as archived
  evidence; the live repo surfaces for this tranche are clean.
