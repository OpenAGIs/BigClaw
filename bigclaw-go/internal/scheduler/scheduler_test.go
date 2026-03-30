package scheduler

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestSchedulerBudgetGuardrail(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "t1", BudgetCents: 200}, QuotaSnapshot{BudgetRemaining: 100})
	if decision.Accepted {
		t.Fatalf("expected budget rejection")
	}
	if decision.Reason != "budget exceeded" {
		t.Fatalf("expected budget exceeded reason, got %+v", decision)
	}
}

func TestSchedulerRoutesHighRiskToKubernetes(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "t1", RiskLevel: domain.RiskHigh}, QuotaSnapshot{})
	if !decision.Accepted {
		t.Fatalf("expected accepted decision")
	}
	if decision.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected kubernetes executor, got %s", decision.Assignment.Executor)
	}
}

func TestSchedulerRoutesComputedHighRiskToKubernetes(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "risk-1", Priority: 1, Labels: []string{"security", "prod"}, RequiredTools: []string{"deploy"}}, QuotaSnapshot{})
	if !decision.Accepted {
		t.Fatalf("expected accepted decision")
	}
	if decision.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected computed high-risk task to route to kubernetes, got %s", decision.Assignment.Executor)
	}
	if decision.Reason == "" {
		t.Fatalf("expected populated risk-aware routing reason")
	}
}

func TestSchedulerRoutesGPUToRay(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "gpu-1", RequiredTools: []string{"gpu"}}, QuotaSnapshot{})
	if !decision.Accepted {
		t.Fatalf("expected accepted decision")
	}
	if decision.Assignment.Executor != domain.ExecutorRay {
		t.Fatalf("expected ray executor, got %s", decision.Assignment.Executor)
	}
}

func TestSchedulerRoutesBrowserToKubernetes(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "browser-1", RequiredTools: []string{"browser"}}, QuotaSnapshot{})
	if !decision.Accepted {
		t.Fatalf("expected accepted decision")
	}
	if decision.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected kubernetes executor, got %s", decision.Assignment.Executor)
	}
	if decision.Reason != "browser workloads default to kubernetes executor" {
		t.Fatalf("expected browser routing reason, got %+v", decision)
	}
}

func TestSchedulerRoutesLowRiskToLocalWithDefaultReason(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "low-1"}, QuotaSnapshot{})
	if !decision.Accepted {
		t.Fatalf("expected accepted decision")
	}
	if decision.Assignment.Executor != domain.ExecutorLocal {
		t.Fatalf("expected local executor, got %s", decision.Assignment.Executor)
	}
	if decision.Reason != "default local executor for low/medium risk" {
		t.Fatalf("expected default low-risk reason, got %+v", decision)
	}
}

func TestSchedulerAssessmentBuildsSecurityHandoffForRejectedDecision(t *testing.T) {
	s := New()
	assessment := s.Assess(domain.Task{
		ID:            "risk-handoff-1",
		RiskLevel:     domain.RiskHigh,
		Labels:        []string{"security"},
		RequiredTools: []string{"deploy"},
		BudgetCents:   500,
	}, QuotaSnapshot{BudgetRemaining: 100})
	if assessment.Decision.Accepted {
		t.Fatalf("expected rejected decision for budget-blocked assessment, got %+v", assessment.Decision)
	}
	if assessment.Risk.Level != domain.RiskHigh {
		t.Fatalf("expected high risk score, got %+v", assessment.Risk)
	}
	if assessment.HandoffRequest == nil || assessment.HandoffRequest.TargetTeam != "security" || assessment.HandoffRequest.Reason == "" {
		t.Fatalf("expected security handoff request, got %+v", assessment.HandoffRequest)
	}
	if len(assessment.HandoffRequest.RequiredApprovals) == 0 || assessment.HandoffRequest.RequiredApprovals[0] != "security-review" {
		t.Fatalf("expected security-review approval requirement, got %+v", assessment.HandoffRequest)
	}
}

func TestSchedulerAssessmentBuildsUpgradeHandoffForStandardTier(t *testing.T) {
	s := New()
	assessment := s.Assess(domain.Task{
		ID:            "upgrade-handoff-1",
		Source:        "linear",
		Title:         "Customer analytics rollout",
		Description:   "Need customer stakeholder rollout and analytics validation.",
		Labels:        []string{"customer", "analytics"},
		RequiredTools: []string{"browser", "sql"},
	}, QuotaSnapshot{})
	if !assessment.Decision.Accepted {
		t.Fatalf("expected accepted decision, got %+v", assessment.Decision)
	}
	if assessment.OrchestrationPolicy.UpgradeRequired != true {
		t.Fatalf("expected upgrade-required policy, got %+v", assessment.OrchestrationPolicy)
	}
	if assessment.HandoffRequest == nil || assessment.HandoffRequest.TargetTeam != "operations" || assessment.HandoffRequest.Status != "blocked" {
		t.Fatalf("expected blocked operations handoff, got %+v", assessment.HandoffRequest)
	}
	if assessment.OrchestrationPlan.CollaborationMode != "tier-limited" || len(assessment.OrchestrationPlan.Handoffs) != 2 || len(assessment.OrchestrationPolicy.BlockedDepartments) == 0 {
		t.Fatalf("expected constrained orchestration plan with blocked departments, got plan=%+v policy=%+v", assessment.OrchestrationPlan, assessment.OrchestrationPolicy)
	}
}

func TestSchedulerAssessmentOmitsHandoffWhenStandardPlanFits(t *testing.T) {
	s := New()
	assessment := s.Assess(domain.Task{
		ID:            "no-handoff-1",
		Source:        "linear",
		Title:         "Basic browser task",
		RequiredTools: []string{"browser"},
	}, QuotaSnapshot{})
	if !assessment.Decision.Accepted {
		t.Fatalf("expected accepted decision, got %+v", assessment.Decision)
	}
	if assessment.OrchestrationPolicy.UpgradeRequired {
		t.Fatalf("expected no upgrade requirement, got %+v", assessment.OrchestrationPolicy)
	}
	if assessment.HandoffRequest != nil {
		t.Fatalf("expected no handoff request, got %+v", assessment.HandoffRequest)
	}
}

func TestSchedulerRejectsLowPriorityOnBackpressure(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "bp-1", Priority: 3}, QuotaSnapshot{QueueDepth: 50, MaxQueueDepth: 50})
	if decision.Accepted {
		t.Fatalf("expected backpressure rejection")
	}
}

func TestSchedulerAllowsUrgentTaskThroughBackpressure(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "bp-2", Priority: 1}, QuotaSnapshot{QueueDepth: 50, MaxQueueDepth: 50})
	if !decision.Accepted {
		t.Fatalf("expected urgent task to bypass backpressure")
	}
}

func TestSchedulerUsesPreemptibleCapacityForUrgentTask(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "preempt-1", Priority: 1}, QuotaSnapshot{ConcurrentLimit: 1, CurrentRunning: 1, PreemptibleExecutions: 1})
	if !decision.Accepted {
		t.Fatalf("expected urgent task to use preemptible capacity")
	}
	if decision.Reason == "" || decision.Assignment.Executor == "" || !decision.Preemption.Required {
		t.Fatalf("expected populated live-preemptive routing decision: %+v", decision)
	}
}

func TestSchedulerUsesFileBackedPolicyOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scheduler-policy.json")
	content := []byte(`{"default_executor":"ray","high_risk_executor":"local","tool_executors":{"browser":"ray","deploy":"kubernetes"},"urgent_priority_threshold":2}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
	store, err := NewPolicyStore(path)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	s := NewWithPolicyStore(store)
	decision := s.Decide(domain.Task{ID: "browser-1", RequiredTools: []string{"browser"}}, QuotaSnapshot{})
	if !decision.Accepted || decision.Assignment.Executor != domain.ExecutorRay {
		t.Fatalf("expected browser override to route to ray, got %+v", decision)
	}
	decision = s.Decide(domain.Task{ID: "default-1"}, QuotaSnapshot{})
	if !decision.Accepted || decision.Assignment.Executor != domain.ExecutorRay {
		t.Fatalf("expected default executor override to route to ray, got %+v", decision)
	}
	decision = s.Decide(domain.Task{ID: "preempt-2", Priority: 2}, QuotaSnapshot{ConcurrentLimit: 1, CurrentRunning: 1, PreemptibleExecutions: 1})
	if !decision.Accepted || !decision.Preemption.Required {
		t.Fatalf("expected priority 2 task to use configured urgent threshold with live preemption, got %+v", decision)
	}
	if err := os.WriteFile(path, []byte(`{"default_executor":"kubernetes"}`), 0o644); err != nil {
		t.Fatalf("rewrite policy file: %v", err)
	}
	if err := store.Reload(); err != nil {
		t.Fatalf("reload policy store: %v", err)
	}
	reloaded := s.Decide(domain.Task{ID: "default-2"}, QuotaSnapshot{})
	if !reloaded.Accepted || reloaded.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected reloaded default executor override to route to kubernetes, got %+v", reloaded)
	}
}

func TestSchedulerPolicyStoreRejectsInvalidExecutor(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scheduler-policy.json")
	if err := os.WriteFile(path, []byte(`{"default_executor":"invalid"}`), 0o644); err != nil {
		t.Fatalf("write invalid policy file: %v", err)
	}
	if _, err := NewPolicyStore(path); err == nil {
		t.Fatal("expected invalid executor error")
	}
}

func TestSchedulerPolicyStoreCanShareSQLiteState(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "scheduler-policy.json")
	sharedPath := filepath.Join(dir, "scheduler-policy.db")
	if err := os.WriteFile(policyPath, []byte(`{"default_executor":"ray","urgent_priority_threshold":2}`), 0o644); err != nil {
		t.Fatalf("write initial policy file: %v", err)
	}
	updater, err := NewPolicyStoreWithSQLite(policyPath, sharedPath)
	if err != nil {
		t.Fatalf("new updater policy store: %v", err)
	}
	t.Cleanup(func() { _ = updater.Close() })
	follower, err := NewPolicyStoreWithSQLite("", sharedPath)
	if err != nil {
		t.Fatalf("new follower policy store: %v", err)
	}
	t.Cleanup(func() { _ = follower.Close() })
	if updater.Backend() != "sqlite" || !updater.Shared() || follower.SharedPath() != sharedPath {
		t.Fatalf("expected shared sqlite policy metadata, got updater=%s follower=%s", updater.Backend(), follower.SharedPath())
	}
	s1 := NewWithPolicyStore(updater)
	s2 := NewWithPolicyStore(follower)
	if decision := s1.Decide(domain.Task{ID: "shared-policy-1"}, QuotaSnapshot{}); !decision.Accepted || decision.Assignment.Executor != domain.ExecutorRay {
		t.Fatalf("expected updater to use initial shared policy, got %+v", decision)
	}
	if decision := s2.Decide(domain.Task{ID: "shared-policy-2"}, QuotaSnapshot{}); !decision.Accepted || decision.Assignment.Executor != domain.ExecutorRay {
		t.Fatalf("expected follower to read initial shared policy, got %+v", decision)
	}
	if err := os.WriteFile(policyPath, []byte(`{"default_executor":"kubernetes","urgent_priority_threshold":3}`), 0o644); err != nil {
		t.Fatalf("rewrite shared policy file: %v", err)
	}
	if err := updater.Reload(); err != nil {
		t.Fatalf("reload updater policy store: %v", err)
	}
	if err := follower.Reload(); err != nil {
		t.Fatalf("reload follower policy store: %v", err)
	}
	if decision := s2.Decide(domain.Task{ID: "shared-policy-3"}, QuotaSnapshot{}); !decision.Accepted || decision.Assignment.Executor != domain.ExecutorKubernetes {
		t.Fatalf("expected follower to observe shared sqlite policy update, got %+v", decision)
	}
	decision := s2.Decide(domain.Task{ID: "shared-preempt-3", Priority: 3}, QuotaSnapshot{ConcurrentLimit: 1, CurrentRunning: 1, PreemptibleExecutions: 1})
	if !decision.Accepted || !decision.Preemption.Required {
		t.Fatalf("expected shared policy urgent threshold update to affect follower decisions, got %+v", decision)
	}
}

func TestSchedulerFairnessWindowThrottlesDominantTenant(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scheduler-policy.json")
	content := []byte(`{"fairness":{"window_seconds":30,"max_recent_decisions_per_tenant":1}}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fairness policy file: %v", err)
	}
	store, err := NewPolicyStore(path)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	s := NewWithPolicyStore(store)
	now := time.Unix(1700000000, 0)
	s.now = func() time.Time { return now }
	if decision := s.Decide(domain.Task{ID: "tenant-a-1", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected first tenant-a task accepted, got %+v", decision)
	}
	now = now.Add(time.Second)
	if decision := s.Decide(domain.Task{ID: "tenant-b-1", TenantID: "tenant-b", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected tenant-b task accepted, got %+v", decision)
	}
	now = now.Add(time.Second)
	throttled := s.Decide(domain.Task{ID: "tenant-a-2", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{})
	if throttled.Accepted || !strings.Contains(throttled.Reason, "fairness window throttled tenant tenant-a") {
		t.Fatalf("expected tenant-a throttled by fairness window, got %+v", throttled)
	}
	urgent := s.Decide(domain.Task{ID: "tenant-a-urgent", TenantID: "tenant-a", Priority: 1}, QuotaSnapshot{})
	if !urgent.Accepted {
		t.Fatalf("expected urgent tenant-a task to bypass fairness throttle, got %+v", urgent)
	}
	snapshot := s.FairnessSnapshot()
	if !snapshot.Enabled || snapshot.ActiveTenants != 2 || len(snapshot.Tenants) != 2 || snapshot.Tenants[0].TenantID != "tenant-a" || snapshot.Tenants[0].RecentAcceptedCount != 2 {
		t.Fatalf("unexpected fairness snapshot: %+v", snapshot)
	}
	now = now.Add(31 * time.Second)
	if decision := s.Decide(domain.Task{ID: "tenant-a-3", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected tenant-a accepted after fairness window expiry, got %+v", decision)
	}
}

func TestSchedulerFairnessWindowCanUseSharedSQLiteState(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "scheduler-policy.json")
	fairnessPath := filepath.Join(dir, "fairness.db")
	content := []byte(`{"fairness":{"window_seconds":30,"max_recent_decisions_per_tenant":1}}`)
	if err := os.WriteFile(policyPath, content, 0o644); err != nil {
		t.Fatalf("write fairness policy file: %v", err)
	}
	store, err := NewPolicyStore(policyPath)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	fairnessA, err := NewFairnessStore(fairnessPath)
	if err != nil {
		t.Fatalf("new shared fairness store A: %v", err)
	}
	fairnessB, err := NewFairnessStore(fairnessPath)
	if err != nil {
		t.Fatalf("new shared fairness store B: %v", err)
	}
	if closable, ok := fairnessA.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closable.Close() })
	}
	if closable, ok := fairnessB.(interface{ Close() error }); ok {
		t.Cleanup(func() { _ = closable.Close() })
	}
	s1 := NewWithStores(store, fairnessA)
	s2 := NewWithStores(store, fairnessB)
	now := time.Unix(1700001000, 0)
	s1.now = func() time.Time { return now }
	s2.now = func() time.Time { return now }
	if decision := s1.Decide(domain.Task{ID: "shared-a-1", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected shared tenant-a task accepted, got %+v", decision)
	}
	now = now.Add(time.Second)
	if decision := s2.Decide(domain.Task{ID: "shared-b-1", TenantID: "tenant-b", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected shared tenant-b task accepted, got %+v", decision)
	}
	now = now.Add(time.Second)
	throttled := s1.Decide(domain.Task{ID: "shared-a-2", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{})
	if throttled.Accepted || !strings.Contains(throttled.Reason, "fairness window throttled tenant tenant-a") {
		t.Fatalf("expected tenant-a throttled via shared fairness state, got %+v", throttled)
	}
	snapshot := s2.FairnessSnapshot()
	if !snapshot.Shared || snapshot.Backend != "sqlite" || snapshot.ActiveTenants != 2 {
		t.Fatalf("unexpected shared fairness snapshot: %+v", snapshot)
	}
}

func TestSchedulerFairnessWindowCanUseRemoteHTTPState(t *testing.T) {
	dir := t.TempDir()
	policyPath := filepath.Join(dir, "scheduler-policy.json")
	content := []byte(`{"fairness":{"window_seconds":30,"max_recent_decisions_per_tenant":1}}`)
	if err := os.WriteFile(policyPath, content, 0o644); err != nil {
		t.Fatalf("write fairness policy file: %v", err)
	}
	store, err := NewPolicyStore(policyPath)
	if err != nil {
		t.Fatalf("new policy store: %v", err)
	}
	serviceStore, err := NewFairnessStore("")
	if err != nil {
		t.Fatalf("new service fairness store: %v", err)
	}
	server := httptest.NewServer(NewFairnessServiceHandler(serviceStore))
	defer server.Close()
	fairnessA, err := NewFairnessStoreWithRemote("", server.URL, "")
	if err != nil {
		t.Fatalf("new remote fairness store A: %v", err)
	}
	fairnessB, err := NewFairnessStoreWithRemote("", server.URL, "")
	if err != nil {
		t.Fatalf("new remote fairness store B: %v", err)
	}
	s1 := NewWithStores(store, fairnessA)
	s2 := NewWithStores(store, fairnessB)
	now := time.Unix(1700002000, 0)
	s1.now = func() time.Time { return now }
	s2.now = func() time.Time { return now }
	if decision := s1.Decide(domain.Task{ID: "remote-a-1", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected remote tenant-a task accepted, got %+v", decision)
	}
	now = now.Add(time.Second)
	if decision := s2.Decide(domain.Task{ID: "remote-b-1", TenantID: "tenant-b", Priority: 3}, QuotaSnapshot{}); !decision.Accepted {
		t.Fatalf("expected remote tenant-b task accepted, got %+v", decision)
	}
	now = now.Add(time.Second)
	throttled := s1.Decide(domain.Task{ID: "remote-a-2", TenantID: "tenant-a", Priority: 3}, QuotaSnapshot{})
	if throttled.Accepted || !strings.Contains(throttled.Reason, "fairness window throttled tenant tenant-a") {
		t.Fatalf("expected tenant-a throttled via remote fairness state, got %+v", throttled)
	}
	snapshot := s2.FairnessSnapshot()
	if !snapshot.Shared || snapshot.Backend != "http" || !snapshot.Healthy || snapshot.Endpoint != server.URL || snapshot.ActiveTenants != 2 {
		t.Fatalf("unexpected remote fairness snapshot: %+v", snapshot)
	}
}
