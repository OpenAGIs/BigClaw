package scheduler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/policy"
	"bigclaw-go/internal/risk"
	"bigclaw-go/internal/workflow"
)

type QuotaSnapshot struct {
	TenantID              string
	OwnerID               string
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
	Isolation  IsolationDecision
}

type IsolationDecision struct {
	TenantMode         string   `json:"tenant_mode,omitempty"`
	TenantSource       string   `json:"tenant_source,omitempty"`
	TenantMetadataKeys []string `json:"tenant_metadata_keys,omitempty"`
	RequireOwnerMatch  bool     `json:"require_owner_match,omitempty"`
	OwnerMetadataKeys  []string `json:"owner_metadata_keys,omitempty"`
	TaskTenantID       string   `json:"task_tenant_id,omitempty"`
	QuotaTenantID      string   `json:"quota_tenant_id,omitempty"`
	TaskOwner          string   `json:"task_owner,omitempty"`
	OwnerSource        string   `json:"owner_source,omitempty"`
	QuotaOwnerID       string   `json:"quota_owner_id,omitempty"`
	Boundary           string   `json:"boundary,omitempty"`
	Violation          bool     `json:"violation,omitempty"`
	Reason             string   `json:"reason,omitempty"`
}

type Assessment struct {
	Decision            Decision
	Risk                risk.Score
	OrchestrationPlan   workflow.OrchestrationPlan
	OrchestrationPolicy workflow.OrchestrationPolicyDecision
	HandoffRequest      *workflow.HandoffRequest
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

func (s *Scheduler) FairnessServiceHandler() http.Handler {
	if s == nil {
		return NewFairnessServiceHandler(newFairnessTracker())
	}
	return NewFairnessServiceHandler(s.fairness)
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
	isolation := evaluateIsolation(task, quota, rules)
	if isolation.Violation {
		return Decision{Accepted: false, Reason: isolation.Reason, Isolation: isolation}
	}
	if quota.BudgetRemaining > 0 && task.BudgetCents > quota.BudgetRemaining {
		return Decision{Accepted: false, Reason: "budget exceeded", Isolation: isolation}
	}
	if quota.MaxQueueDepth > 0 && quota.QueueDepth >= quota.MaxQueueDepth && !isPriorityExempt(task, rules) {
		return Decision{Accepted: false, Reason: "backpressure activated: queue depth limit exceeded", Isolation: isolation}
	}
	if shouldThrottleForFairness(task, rules) && s.fairness.ShouldThrottle(now, strings.TrimSpace(task.TenantID), rules) {
		return Decision{Accepted: false, Reason: fairnessThrottleReason(strings.TrimSpace(task.TenantID), rules), Isolation: isolation}
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
				Isolation:  isolation,
			}
		}
		return Decision{Accepted: false, Reason: "tenant concurrency quota exceeded", Isolation: isolation}
	}

	s.fairness.RecordAccepted(now, strings.TrimSpace(task.TenantID), rules)
	return Decision{Accepted: true, Assignment: assignment, Reason: assignment.Reason, Isolation: isolation}
}

func (s *Scheduler) Assess(task domain.Task, quota QuotaSnapshot) Assessment {
	decision := s.Decide(task, quota)
	score := risk.ScoreTask(task, nil)
	rawPlan := workflow.CrossDepartmentOrchestrator{}.Plan(task)
	orchestrationPlan, policyDecision := workflow.PremiumOrchestrationPolicy{}.Apply(task, rawPlan)
	return Assessment{
		Decision:            decision,
		Risk:                score,
		OrchestrationPlan:   orchestrationPlan,
		OrchestrationPolicy: policyDecision,
		HandoffRequest:      buildHandoffRequest(decision, orchestrationPlan, policyDecision),
	}
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

func buildHandoffRequest(decision Decision, plan workflow.OrchestrationPlan, policyDecision workflow.OrchestrationPolicyDecision) *workflow.HandoffRequest {
	if !decision.Accepted {
		requiredApprovals := plan.RequiredApprovals()
		if len(requiredApprovals) == 0 {
			requiredApprovals = []string{"security-review"}
		}
		return &workflow.HandoffRequest{
			TargetTeam:        "security",
			Reason:            decision.Reason,
			Status:            "pending",
			RequiredApprovals: requiredApprovals,
		}
	}
	if policyDecision.UpgradeRequired {
		return &workflow.HandoffRequest{
			TargetTeam:        "operations",
			Reason:            policyDecision.Reason,
			Status:            "blocked",
			RequiredApprovals: []string{"ops-manager"},
		}
	}
	return nil
}

func evaluateIsolation(task domain.Task, quota QuotaSnapshot, rules RoutingRules) IsolationDecision {
	effective := effectiveIsolationRules(task, rules)
	taskTenantID := strings.TrimSpace(task.TenantID)
	quotaTenantID := strings.TrimSpace(quota.TenantID)
	taskOwner := taskOwner(task, effective.OwnerMetadataKeys)
	quotaOwnerID := strings.TrimSpace(quota.OwnerID)
	isolation := IsolationDecision{
		TenantMode:        effective.TenantMode,
		RequireOwnerMatch: effective.RequireOwnerMatch,
		OwnerMetadataKeys: append([]string(nil), effective.OwnerMetadataKeys...),
		TaskTenantID:      taskTenantID,
		QuotaTenantID:     quotaTenantID,
		TaskOwner:         taskOwner,
		QuotaOwnerID:      quotaOwnerID,
	}
	if effective.TenantMode == "tenant" {
		isolation.Boundary = "tenant"
		switch {
		case quotaTenantID != "" && taskTenantID == "":
			isolation.Violation = true
			isolation.Reason = fmt.Sprintf("tenant isolation boundary: unscoped task cannot use tenant %s capacity", quotaTenantID)
			return isolation
		case quotaTenantID != "" && taskTenantID != quotaTenantID:
			isolation.Violation = true
			isolation.Reason = fmt.Sprintf("tenant isolation boundary: task tenant %s cannot use tenant %s capacity", taskTenantID, quotaTenantID)
			return isolation
		}
	}
	if effective.RequireOwnerMatch && quotaOwnerID != "" {
		isolation.Boundary = "owner"
		switch {
		case taskOwner == "":
			isolation.Violation = true
			isolation.Reason = fmt.Sprintf("ownership boundary: task missing owner cannot use owner %s capacity", quotaOwnerID)
			return isolation
		case taskOwner != quotaOwnerID:
			isolation.Violation = true
			isolation.Reason = fmt.Sprintf("ownership boundary: task owner %s cannot use owner %s capacity", taskOwner, quotaOwnerID)
			return isolation
		}
	}
	return isolation
}

func effectiveIsolationRules(task domain.Task, rules RoutingRules) IsolationRules {
	effective := cloneIsolationRules(rules.Isolation)
	taskPolicy := policy.Resolve(task)
	if taskPolicy.TenantIsolationMode == "tenant" {
		effective.TenantMode = "tenant"
	}
	if taskPolicy.OwnerMatchingRequired {
		effective.RequireOwnerMatch = true
	}
	effective.OwnerMetadataKeys = mergeMetadataKeys(effective.OwnerMetadataKeys, taskPolicy.OwnerMetadataKeys)
	return effective
}

func cloneIsolationRules(rules IsolationRules) IsolationRules {
	out := rules
	out.OwnerMetadataKeys = append([]string(nil), rules.OwnerMetadataKeys...)
	return out
}

func mergeMetadataKeys(base []string, extra []string) []string {
	seen := make(map[string]struct{}, len(base)+len(extra))
	merged := make([]string, 0, len(base)+len(extra))
	for _, key := range append(append([]string(nil), base...), extra...) {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, key)
	}
	return merged
}

func taskOwner(task domain.Task, metadataKeys []string) string {
	for _, key := range metadataKeys {
		if owner := strings.TrimSpace(task.Metadata[key]); owner != "" {
			return owner
		}
	}
	return ""
}
