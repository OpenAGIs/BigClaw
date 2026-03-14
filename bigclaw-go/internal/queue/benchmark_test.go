package queue

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func BenchmarkMemoryQueueEnqueueLease(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		q := NewMemoryQueue()
		for n := 0; n < 100; n++ {
			_ = q.Enqueue(ctx, domain.Task{ID: strconv.Itoa(n), Priority: n % 3, CreatedAt: time.Now()})
		}
		_, _, _ = q.LeaseNext(ctx, "worker-a", time.Second)
	}
}

func BenchmarkFileQueueEnqueueLease(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()
	for i := 0; i < b.N; i++ {
		q, _ := NewFileQueue(filepath.Join(tempDir, strconv.Itoa(i)+".json"))
		for n := 0; n < 100; n++ {
			_ = q.Enqueue(ctx, domain.Task{ID: strconv.Itoa(n), Priority: n % 3, CreatedAt: time.Now()})
		}
		_, _, _ = q.LeaseNext(ctx, "worker-a", time.Second)
	}
}

func BenchmarkSQLiteQueueEnqueueLease(b *testing.B) {
	ctx := context.Background()
	tempDir := b.TempDir()
	for i := 0; i < b.N; i++ {
		q, _ := NewSQLiteQueue(filepath.Join(tempDir, strconv.Itoa(i)+".db"))
		for n := 0; n < 100; n++ {
			_ = q.Enqueue(ctx, domain.Task{ID: strconv.Itoa(n), Priority: n % 3, CreatedAt: time.Now()})
		}
		_, _, _ = q.LeaseNext(ctx, "worker-a", time.Second)
		_ = q.Close()
	}
}
