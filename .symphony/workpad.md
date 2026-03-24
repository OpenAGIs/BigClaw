## BIGCLAW-185 Workpad

### Plan

- [x] Inspect the current worker retry scheduling path and isolate the fixed backoff delays that prevent strategy comparison.
- [x] Add a scoped worker backoff policy surface with a few deterministic strategies suitable for benchmark comparison.
- [x] Wire the runtime retry/requeue paths to the backoff policy without changing unrelated scheduler or queue behavior.
- [x] Add targeted unit coverage and benchmarks that compare worker scheduling backoff strategies under the same retry attempts.
- [x] Run targeted validation, then commit and push the issue branch.

### Acceptance

- [x] Worker retry scheduling no longer relies on scattered hard-coded requeue delays in the runtime path.
- [x] The codebase exposes benchmark coverage that compares multiple worker backoff strategies for scheduling retries.
- [x] Existing worker runtime behavior remains covered by targeted tests.

### Validation

- [x] `cd bigclaw-go && go test ./internal/worker ./internal/scheduler -run 'TestRuntime|TestBackoff' -bench BenchmarkWorkerBackoffStrategy -benchmem`
- [x] `cd bigclaw-go && go test ./internal/worker ./internal/scheduler`

### Results

- [x] `cd bigclaw-go && go test ./internal/worker ./internal/scheduler -run 'TestRuntime|TestBackoff' -bench BenchmarkWorkerBackoffStrategy -benchmem`
- [x] Result: `ok  	bigclaw-go/internal/worker	6.769s`, `ok  	bigclaw-go/internal/scheduler	0.912s`
- [x] Benchmark: `fixed 14.39 ns/op`, `linear 14.16 ns/op`, `exponential 13.94 ns/op`, `0 B/op`, `0 allocs/op`
- [x] `cd bigclaw-go && go test ./internal/worker ./internal/scheduler`
- [x] Result: `ok  	bigclaw-go/internal/worker	3.766s`, `ok  	bigclaw-go/internal/scheduler	(cached)`
