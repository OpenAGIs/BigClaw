Issue: `BIG-GO-1483`

Summary:
- Current `origin/main` is already Go-only for `bigclaw-go/scripts` and for the full repository Python-file inventory.
- There are no remaining checked-in `.py` files to migrate or delete from this branch baseline.
- The live `bigclaw-go/scripts` surface is limited to Go and shell entrypoints that already dispatch through `bigclawctl`.

Exact evidence captured on branch `BIG-GO-1483` at `a63c8ec0f999d976a1af890c920a54ac2d6c693a`:

```bash
find . -type f -name '*.py' | sort | wc -l
0

find bigclaw-go/scripts -type f -name '*.py' | sort | wc -l
0

find bigclaw-go/scripts -type f | sort
bigclaw-go/scripts/benchmark/run_suite.sh
bigclaw-go/scripts/e2e/broker_bootstrap_summary.go
bigclaw-go/scripts/e2e/kubernetes_smoke.sh
bigclaw-go/scripts/e2e/ray_smoke.sh
bigclaw-go/scripts/e2e/run_all.sh
```

Checked-in caller state:
- `bigclaw-go/scripts/benchmark/run_suite.sh` invokes `go test` and `go run ./cmd/bigclawctl automation benchmark run-matrix`.
- `bigclaw-go/scripts/e2e/kubernetes_smoke.sh` invokes `go run ./cmd/bigclawctl automation e2e run-task-smoke`.
- `bigclaw-go/scripts/e2e/ray_smoke.sh` invokes `go run ./cmd/bigclawctl automation e2e run-task-smoke`.
- `bigclaw-go/scripts/e2e/run_all.sh` orchestrates `bigclawctl automation e2e ...` subcommands and `broker_bootstrap_summary.go`.

Blocker:
- The hard requirement to reduce the repository Python file count cannot be satisfied on the current branch baseline because the repository already contains zero checked-in Python files before any change.
- This issue appears stale relative to `origin/main`; the migration described in the issue was completed by earlier work before `BIG-GO-1483` was created.
