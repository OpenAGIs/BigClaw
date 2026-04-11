# BIG-GO-199 Python Asset Sweep

## Scope

`BIG-GO-199` records a residual auxiliary Python sweep focused on hidden,
nested, or otherwise easy-to-overlook directories that can hide Python-like
leftovers outside the primary removal surfaces.

## Remaining Python Inventory

Repository-wide Python-like file count: `0`.

Overlooked Python-like suffixes audited: `.py`, `.pyw`, `.pyi`, `.pyx`, `.pxd`, `.pxi`, `.ipynb`.

- `.githooks`: `0` Python-like files
- `.github`: `0` Python-like files
- `.symphony`: `0` Python-like files
- `bigclaw-go/examples`: `0` Python-like files
- `bigclaw-go/docs/reports/live-validation-runs`: `0` Python-like files
- `reports`: `0` Python-like files

This checkout therefore lands as a regression-prevention sweep for hidden and
nested auxiliary surfaces rather than a direct Python-file deletion batch.

## Retained Native Assets

The audited directories still contain non-Python assets that should remain in
place:

- `.githooks/post-commit`
- `.githooks/post-rewrite`
- `.github/workflows/ci.yml`
- `.symphony/workpad.md`
- `bigclaw-go/examples/shadow-task.json`
- `bigclaw-go/examples/shadow-task-validation.json`
- `bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`
- `reports/BIG-GO-192-validation.md`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.pyx' -o -name '*.pxd' -o -name '*.pxi' -o -name '*.ipynb' \) -print | sort`
  Result: no output; repository-wide Python-like file count remained `0`.
- `find .githooks .github .symphony bigclaw-go/examples bigclaw-go/docs/reports/live-validation-runs reports -type f \( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.pyx' -o -name '*.pxd' -o -name '*.pxi' -o -name '*.ipynb' \) 2>/dev/null | sort`
  Result: no output; the hidden and nested residual directories remained Python-free across all audited suffixes.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO199(RepositoryHasNoPythonFiles|HiddenAndNestedResidualDirectoriesStayPythonFree|HiddenAndNestedNativeAssetsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.237s`
