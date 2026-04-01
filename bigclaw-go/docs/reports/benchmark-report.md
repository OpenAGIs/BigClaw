goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/queue
cpu: Apple M4
BenchmarkMemoryQueueEnqueueLease-10    	   40660	     29227 ns/op
BenchmarkFileQueueEnqueueLease-10      	      24	  48228771 ns/op
BenchmarkSQLiteQueueEnqueueLease-10    	     128	   9340130 ns/op
PASS
ok  	bigclaw-go/internal/queue	31.514s
goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/scheduler
cpu: Apple M4
BenchmarkSchedulerDecide-10    	 2735659	       436.0 ns/op
PASS
ok  	bigclaw-go/internal/scheduler	2.330s
