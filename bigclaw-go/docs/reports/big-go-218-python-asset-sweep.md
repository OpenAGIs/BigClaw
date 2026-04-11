# BIG-GO-218 Python Asset Sweep

## Scope

Broad repo Python reduction sweep `BIG-GO-218` hardens the remaining active
documentation surface after the repository reached a physical zero-Python-file
baseline.

This lane focuses on removing stale Python-era bootstrap and cutover guidance
from:

- `docs/symphony-repo-bootstrap-template.md`
- `docs/go-mainline-cutover-handoff.md`

## Remaining Python Inventory

Repository-wide Python file count: `0`.

This lane therefore lands as a documentation and regression-prevention sweep
rather than a direct Python-file deletion batch in this checkout.

## Go Or Native Replacement Paths

The active Go/native replacement surface covering this sweep remains:

- `scripts/ops/bigclawctl`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`
- `docs/symphony-repo-bootstrap-template.md`
- `docs/go-mainline-cutover-handoff.md`

## Documentation Reductions

- `docs/symphony-repo-bootstrap-template.md` now requires repo-native
  `bigclawctl workspace bootstrap`, `bigclawctl workspace cleanup`, and
  `bigclawctl workspace validate` support instead of copying Python bootstrap
  compatibility files.
- `docs/go-mainline-cutover-handoff.md` now records Go-native workspace
  validation evidence via `bash scripts/ops/bigclawctl workspace validate --help >/dev/null`
  instead of a Python shim assertion snippet.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `rg -n "workspace_bootstrap\\.py|workspace_bootstrap_cli\\.py|PYTHONPATH=src python3" docs/symphony-repo-bootstrap-template.md docs/go-mainline-cutover-handoff.md`
  Result: no output; the active docs no longer carry the retired Python
  bootstrap or cutover strings.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO218(RepositoryHasNoPythonFiles|ActiveBootstrapDocsStayGoOnly|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.187s`
