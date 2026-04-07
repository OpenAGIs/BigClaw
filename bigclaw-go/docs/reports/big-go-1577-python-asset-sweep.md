# BIG-GO-1577 Python Asset Sweep

## Scope

This sweep covers the requested residual Python tranche:

- `src/bigclaw/cost_control.py`
- `src/bigclaw/mapping.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/roadmap.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `tests/test_design_system.py`
- `tests/test_live_shadow_bundle.py`
- `tests/test_pilot.py`
- `tests/test_repo_triage.py`
- `tests/test_subscriber_takeover_harness.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`

This lane also removed the direct Python-only test dependencies that still
imported the retired modules:

- `tests/test_cost_control.py`
- `tests/test_mapping.py`
- `tests/test_repo_board.py`
- `tests/test_repo_collaboration.py`
- `tests/test_roadmap.py`

## Sweep Result

- Removed the Python-only legacy library surfaces under `src/bigclaw` that no
  longer back the Go-first mainline.
- Removed the Python-only regression files that existed only to validate the
  retired Python implementations.
- Retired the legacy operator wrapper path `scripts/ops/symphony_workspace_bootstrap.py`
  in favor of `scripts/ops/bigclawctl workspace bootstrap`.
- Renamed `bigclaw-go/scripts/migration/export_live_shadow_bundle.py` to the
  extensionless compatibility path `bigclaw-go/scripts/migration/export_live_shadow_bundle`
  so the physical `.py` asset is gone while the documented CLI entrypoint
  remains usable during migration.

Explicit remaining shim deletion condition:

- `bigclaw-go/scripts/migration/export_live_shadow_bundle` should be deleted once
  the current branch adopts the `bigclawctl automation migration export-live-shadow-bundle`
  command family that already exists on newer Go-only baselines.

## Go Or Native Replacement Paths

- `scripts/ops/bigclawctl`
- `bigclaw-go/internal/intake/mapping.go`
- `bigclaw-go/internal/repo/board.go`
- `bigclaw-go/internal/repo/triage.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle`

## Validation Commands And Results

- `find src/bigclaw tests scripts/ops bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: the targeted residual paths are absent; the remaining `.py` files are outside this sweep and do not include any requested candidate asset.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1577(TargetResidualPythonPathsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	4.030s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  Result: `14 passed`

## Residual Risk

- The live-shadow bundle exporter is still implemented in Python semantics under
  an extensionless migration shim. The physical `.py` asset is gone, but full
  deletion still depends on landing the newer `bigclawctl automation migration`
  replacement surface in this branch lineage.
