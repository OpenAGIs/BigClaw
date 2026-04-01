goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/queue
cpu: Apple M4
BenchmarkMemoryQueueEnqueueLease-10    	   38758	     29831 ns/op
BenchmarkFileQueueEnqueueLease-10      	      22	  46735343 ns/op
BenchmarkSQLiteQueueEnqueueLease-10    	     100	  10345601 ns/op
PASS
ok  	bigclaw-go/internal/queue	28.804s
goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/scheduler
cpu: Apple M4
BenchmarkSchedulerDecide-10    	 2678006	       425.5 ns/op
PASS
ok  	bigclaw-go/internal/scheduler	4.971s
