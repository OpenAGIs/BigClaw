package scheduler

import (
	"fmt"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/risk"
)

type QuotaSnapshot struct {
	TenantID              string
	ConcurrentLimit       int
	CurrentRunning        int
	BudgetRemaining       int64
	QueueDepth            int
	MaxQueueDepth         int
	PreemptibleExecutions int
}

type Decision struct {
	Assignment executor.Assignment
	Accepted   bool
	Reason     string
}

type Scheduler struct{}

func New() *Scheduler {
	return &Scheduler{}
}

func (s *Scheduler) Decide(task domain.Task, quota QuotaSnapshot) Decision {
	if quota.BudgetRemaining > 0 && task.BudgetCents > quota.BudgetRemaining {
		return Decision{Accepted: false, Reason: "budget exceeded"}
	}
	if quota.MaxQueueDepth > 0 && quota.QueueDepth >= quota.MaxQueueDepth && !isPriorityExempt(task) {
		return Decision{Accepted: false, Reason: "backpressure activated: queue depth limit exceeded"}
	}
	assignment := executor.Assignment{
		Executor: routeExecutor(task),
		Reason:   routeReason(task),
	}
	if quota.ConcurrentLimit > 0 && quota.CurrentRunning >= quota.ConcurrentLimit {
		if isPreemptible(task) && quota.PreemptibleExecutions > 0 {
			assignment.Reason = assignment.Reason + "; using preemptible capacity"
			return Decision{Accepted: true, Assignment: assignment, Reason: assignment.Reason}
		}
		return Decision{Accepted: false, Reason: "tenant concurrency quota exceeded"}
	}

	return Decision{Accepted: true, Assignment: assignment, Reason: assignment.Reason}
}

func routeExecutor(task domain.Task) domain.ExecutorKind {
	if task.RequiredExecutor != "" {
		return task.RequiredExecutor
	}
	score := risk.ScoreTask(task, nil)
	if requiresTool(task, "gpu") {
		return domain.ExecutorRay
	}
	if requiresTool(task, "browser") {
		return domain.ExecutorKubernetes
	}
	if score.Level == domain.RiskHigh {
		return domain.ExecutorKubernetes
	}
	return domain.ExecutorLocal
}

func routeReason(task domain.Task) string {
	if task.RequiredExecutor != "" {
		return fmt.Sprintf("required executor=%s", task.RequiredExecutor)
	}
	score := risk.ScoreTask(task, nil)
	if requiresTool(task, "gpu") {
		return "gpu workloads default to ray executor"
	}
	if requiresTool(task, "browser") {
		return "browser workloads default to kubernetes executor"
	}
	if score.Level == domain.RiskHigh {
		return fmt.Sprintf("risk score %d defaults task to isolated executor", score.Total)
	}
	return "default local executor for low/medium risk"
}

func requiresTool(task domain.Task, tool string) bool {
	for _, item := range task.RequiredTools {
		if item == tool {
			return true
		}
	}
	return false
}

func isPreemptible(task domain.Task) bool {
	return task.Priority > 0 && task.Priority <= 1
}

func isPriorityExempt(task domain.Task) bool {
	return isPreemptible(task) || risk.ScoreTask(task, nil).Level == domain.RiskHigh
}
