package scheduler

import (
	"fmt"
	"strings"
	"time"

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

type PreemptionPlan struct {
	Required bool   `json:"required,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type Decision struct {
	Assignment executor.Assignment
	Accepted   bool
	Reason     string
	Preemption PreemptionPlan
}

type Scheduler struct {
	policyStore *PolicyStore
	fairness    FairnessStore
	now         func() time.Time
}

func New() *Scheduler {
	return NewWithStores(nil, nil)
}

func NewWithPolicyStore(store *PolicyStore) *Scheduler {
	return NewWithStores(store, nil)
}

func NewWithStores(store *PolicyStore, fairness FairnessStore) *Scheduler {
	if store == nil {
		store = NewDefaultPolicyStore()
	}
	if fairness == nil {
		fairness = newFairnessTracker()
	}
	return &Scheduler{policyStore: store, fairness: fairness, now: time.Now}
}

func (s *Scheduler) Rules() RoutingRules {
	if s == nil || s.policyStore == nil {
		return DefaultRoutingRules()
	}
	return s.policyStore.Snapshot()
}

func (s *Scheduler) FairnessSnapshot() FairnessSnapshot {
	if s == nil {
		return FairnessSnapshot{}
	}
	return s.fairness.Snapshot(s.currentTime(), s.Rules())
}

func (s *Scheduler) currentTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}
	return s.now()
}

func (s *Scheduler) Decide(task domain.Task, quota QuotaSnapshot) Decision {
	rules := s.Rules()
	now := s.currentTime()
	if quota.BudgetRemaining > 0 && task.BudgetCents > quota.BudgetRemaining {
		return Decision{Accepted: false, Reason: "budget exceeded"}
	}
	if quota.MaxQueueDepth > 0 && quota.QueueDepth >= quota.MaxQueueDepth && !isPriorityExempt(task, rules) {
		return Decision{Accepted: false, Reason: "backpressure activated: queue depth limit exceeded"}
	}
	if shouldThrottleForFairness(task, rules) && s.fairness.ShouldThrottle(now, strings.TrimSpace(task.TenantID), rules) {
		return Decision{Accepted: false, Reason: fairnessThrottleReason(strings.TrimSpace(task.TenantID), rules)}
	}
	assignment := assignmentForTask(task, rules)
	if quota.ConcurrentLimit > 0 && quota.CurrentRunning >= quota.ConcurrentLimit {
		if isPreemptible(task, rules) && quota.PreemptibleExecutions > 0 {
			assignment.Reason = assignment.Reason + "; using preemptible capacity"
			s.fairness.RecordAccepted(now, strings.TrimSpace(task.TenantID), rules)
			return Decision{
				Accepted:   true,
				Assignment: assignment,
				Reason:     assignment.Reason,
				Preemption: PreemptionPlan{Required: true, Reason: "urgent task may reclaim lower-priority active capacity"},
			}
		}
		return Decision{Accepted: false, Reason: "tenant concurrency quota exceeded"}
	}

	s.fairness.RecordAccepted(now, strings.TrimSpace(task.TenantID), rules)
	return Decision{Accepted: true, Assignment: assignment, Reason: assignment.Reason}
}

func assignmentForTask(task domain.Task, rules RoutingRules) executor.Assignment {
	executorKind, reason := routeExecutorAndReason(task, rules)
	return executor.Assignment{Executor: executorKind, Reason: reason}
}

func routeExecutorAndReason(task domain.Task, rules RoutingRules) (domain.ExecutorKind, string) {
	if task.RequiredExecutor != "" {
		return task.RequiredExecutor, fmt.Sprintf("required executor=%s", task.RequiredExecutor)
	}
	if tool, executorKind, ok := routeToolExecutor(task, rules); ok {
		return executorKind, toolRouteReason(tool, executorKind)
	}
	score := risk.ScoreTask(task, nil)
	if score.Level == domain.RiskHigh {
		return rules.HighRiskExecutor, highRiskRouteReason(score.Total, rules.HighRiskExecutor)
	}
	return rules.DefaultExecutor, defaultRouteReason(rules.DefaultExecutor)
}

func routeToolExecutor(task domain.Task, rules RoutingRules) (string, domain.ExecutorKind, bool) {
	for _, tool := range []string{"gpu", "browser"} {
		if !requiresTool(task, tool) {
			continue
		}
		if executorKind, ok := rules.ToolExecutors[tool]; ok {
			return tool, executorKind, true
		}
	}
	for _, tool := range task.RequiredTools {
		normalizedTool := normalizeToolName(tool)
		if normalizedTool == "" || normalizedTool == "gpu" || normalizedTool == "browser" {
			continue
		}
		if executorKind, ok := rules.ToolExecutors[normalizedTool]; ok {
			return normalizedTool, executorKind, true
		}
	}
	return "", "", false
}

func toolRouteReason(tool string, executorKind domain.ExecutorKind) string {
	switch {
	case tool == "gpu" && executorKind == domain.ExecutorRay:
		return "gpu workloads default to ray executor"
	case tool == "browser" && executorKind == domain.ExecutorKubernetes:
		return "browser workloads default to kubernetes executor"
	default:
		return fmt.Sprintf("%s workloads default to %s executor", tool, executorKind)
	}
}

func highRiskRouteReason(score int, executorKind domain.ExecutorKind) string {
	if executorKind == domain.ExecutorKubernetes {
		return fmt.Sprintf("risk score %d defaults task to isolated executor", score)
	}
	return fmt.Sprintf("risk score %d defaults task to %s executor", score, executorKind)
}

func defaultRouteReason(executorKind domain.ExecutorKind) string {
	if executorKind == domain.ExecutorLocal {
		return "default local executor for low/medium risk"
	}
	return fmt.Sprintf("default executor=%s for low/medium risk", executorKind)
}

func requiresTool(task domain.Task, tool string) bool {
	for _, item := range task.RequiredTools {
		if item == tool {
			return true
		}
	}
	return false
}

func isPreemptible(task domain.Task, rules RoutingRules) bool {
	return task.Priority > 0 && task.Priority <= rules.UrgentPriorityThreshold
}

func isPriorityExempt(task domain.Task, rules RoutingRules) bool {
	return isPreemptible(task, rules) || risk.ScoreTask(task, nil).Level == domain.RiskHigh
}

func shouldThrottleForFairness(task domain.Task, rules RoutingRules) bool {
	if !fairnessEnabled(rules) {
		return false
	}
	if strings.TrimSpace(task.TenantID) == "" {
		return false
	}
	return !isPriorityExempt(task, rules)
}
