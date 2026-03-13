goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/queue
BenchmarkMemoryQueueEnqueueLease-8   	   19989	     58938 ns/op
BenchmarkFileQueueEnqueueLease-8     	      37	  68003796 ns/op
BenchmarkSQLiteQueueEnqueueLease-8   	      57	  30264842 ns/op
PASS
ok  	bigclaw-go/internal/queue	7.843s
goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/scheduler
BenchmarkSchedulerDecide-8   	30962115	        51.08 ns/op
PASS
ok  	bigclaw-go/internal/scheduler	1.647s
