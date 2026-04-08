# BIG-GO-138 Validation

## Summary

`BIG-GO-138` removed stale Python migration command guidance from the active
live-shadow documentation surface and replaced it with the supported Go
automation entrypoints.

## Changed Scope

- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/docs/reports/live-shadow-index.md`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`
- `bigclaw-go/docs/reports/big-go-138-python-guidance-sweep.md`
- `bigclaw-go/internal/regression/big_go_138_python_guidance_sweep_test.go`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`

## Validation Commands

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO138MigrationGuidancePrefersGoAutomation|BIGGO138LaneReportCapturesSweepState|LiveShadowBundleSummaryAndIndexStayAligned)$'`

## Validation Results

### `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`

```text
[no output]
```

### `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`

```text
[no output]
```

### `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO138MigrationGuidancePrefersGoAutomation|BIGGO138LaneReportCapturesSweepState|LiveShadowBundleSummaryAndIndexStayAligned)$'`

```text
ok  	bigclaw-go/internal/regression	0.190s
```

## Outcome

- Active migration-shadow docs no longer instruct operators to run retired
  `python3 scripts/migration/...` commands.
- Canonical and bundled live-shadow summary/index artifacts now expose the Go
  closeout commands already supported by `bigclawctl automation migration`.
- Regression coverage now pins this guidance so Python-era commands do not
  silently reappear in the active surface.

## Git

- Branch: `feat/BIG-GO-138-python-guidance-sweep`
- Commit: `ebd03799af31643fdaf21fb81ace048e273fce6e`
- Push:

```text
git push -u origin feat/BIG-GO-138-python-guidance-sweep
remote:
remote: Create a pull request for 'feat/BIG-GO-138-python-guidance-sweep' on GitHub by visiting:
remote:      https://github.com/OpenAGIs/BigClaw/pull/new/feat/BIG-GO-138-python-guidance-sweep
remote:
To https://github.com/OpenAGIs/BigClaw.git
 * [new branch]        feat/BIG-GO-138-python-guidance-sweep -> feat/BIG-GO-138-python-guidance-sweep
branch 'feat/BIG-GO-138-python-guidance-sweep' set up to track 'origin/feat/BIG-GO-138-python-guidance-sweep'.
```
