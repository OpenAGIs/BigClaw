package scheduler

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func BenchmarkSchedulerDecide(b *testing.B) {
	s := New()
	task := domain.Task{ID: "task-1", RiskLevel: domain.RiskHigh, BudgetCents: 100}
	quota := QuotaSnapshot{ConcurrentLimit: 100, CurrentRunning: 10, BudgetRemaining: 1000}
	for i := 0; i < b.N; i++ {
		_ = s.Decide(task, quota)
	}
}
