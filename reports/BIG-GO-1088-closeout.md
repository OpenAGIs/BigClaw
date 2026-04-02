# BIG-GO-1088 Closeout

Issue: `BIG-GO-1088`

Title: `Go-replacement AV: delete bigclaw-go benchmark Python helpers tranche`

Date: `2026-04-02`

## Outcome

`BIG-GO-1088` is already satisfied in the current base branch. The benchmark helper tranche under
`bigclaw-go/scripts/benchmark/` no longer contains any Python files, and the retained operator
entrypoint is `bigclaw-go/scripts/benchmark/run_suite.sh`, which dispatches into Go automation.

The physical `.py` deletions were already merged before this lane:

- `da168148` deleted:
  - `bigclaw-go/scripts/benchmark/capacity_certification.py`
  - `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  - `bigclaw-go/scripts/benchmark/run_matrix.py`
  - `bigclaw-go/scripts/benchmark/soak_local.py`
- `9746a50c` followed up by enforcing the Go-only benchmark surface and adding regression coverage
  around `TestBenchmarkScriptsStayGoOnly`.

## Validation Commands

```bash
find bigclaw-go/scripts/benchmark -maxdepth 1 -name '*.py' | wc -l
find . -name '*.py' | wc -l
git ls-tree -r --name-only HEAD bigclaw-go/scripts/benchmark
cd bigclaw-go && go test ./cmd/bigclawctl -run TestBenchmarkScriptsStayGoOnly -count=1
git show --stat --summary da168148 | sed -n '1,220p'
git show --stat --summary 9746a50c | sed -n '1,220p'
```

## Validation Results

- `find bigclaw-go/scripts/benchmark -maxdepth 1 -name '*.py' | wc -l` -> `0`
- `find . -name '*.py' | wc -l` -> `23`
- `git ls-tree -r --name-only HEAD bigclaw-go/scripts/benchmark` -> `bigclaw-go/scripts/benchmark/run_suite.sh`
- `cd bigclaw-go && go test ./cmd/bigclawctl -run TestBenchmarkScriptsStayGoOnly -count=1` -> `ok   bigclaw-go/cmd/bigclawctl 0.640s`
- `git show --stat --summary da168148 | sed -n '1,220p'` -> confirms the benchmark Python helper deletions listed above
- `git show --stat --summary 9746a50c | sed -n '1,220p'` -> confirms the later Go-only enforcement pass

## Blocker

The issue's hard acceptance criterion, "repository `.py` count decreases after this lane," cannot be
re-satisfied on top of the current base because the benchmark Python helper tranche has already been
physically deleted upstream. This lane therefore records the blocker evidence rather than fabricating
an unrelated `.py` deletion outside the issue scope.
