goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/queue
BenchmarkMemoryQueueEnqueueLease-8   	   21180	     66075 ns/op
BenchmarkFileQueueEnqueueLease-8     	      32	  31627767 ns/op
BenchmarkSQLiteQueueEnqueueLease-8   	      66	  18057898 ns/op
PASS
ok  	bigclaw-go/internal/queue	5.622s
goos: darwin
goarch: arm64
pkg: bigclaw-go/internal/scheduler
BenchmarkSchedulerDecide-8   	16466796	        73.98 ns/op
PASS
ok  	bigclaw-go/internal/scheduler	1.310s
