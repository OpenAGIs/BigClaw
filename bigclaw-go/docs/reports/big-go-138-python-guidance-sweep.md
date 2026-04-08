# BIG-GO-138 Migration Guidance Python Reduction Sweep

`BIG-GO-138` keeps pressure on repo-wide Python retirement by removing stale
Python migration invocations from the active live-shadow documentation surface.

## Scope

- `bigclaw-go/docs/migration-shadow.md`
- `bigclaw-go/docs/reports/migration-readiness-report.md`
- `bigclaw-go/docs/reports/live-shadow-index.md`
- `bigclaw-go/docs/reports/live-shadow-summary.json`
- `bigclaw-go/docs/reports/live-shadow-index.json`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/README.md`
- `bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`

## Result

- Active migration-shadow guidance now uses:
  - `go run ./cmd/bigclawctl automation migration shadow-compare`
  - `go run ./cmd/bigclawctl automation migration shadow-matrix`
  - `go run ./cmd/bigclawctl automation migration live-shadow-scorecard`
  - `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle`
- Retired active guidance removed:
  - `python3 scripts/migration/shadow_compare.py`
  - `python3 scripts/migration/shadow_matrix.py`
  - `python3 scripts/migration/live_shadow_scorecard.py`
  - `python3 scripts/migration/export_live_shadow_bundle`
- Canonical and bundled live-shadow summary artifacts now advertise the Go
  closeout commands instead of Python-era invocations.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO138MigrationGuidancePrefersGoAutomation|BIGGO138LaneReportCapturesSweepState|LiveShadowBundleSummaryAndIndexStayAligned)$'`
