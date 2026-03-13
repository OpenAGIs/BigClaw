package scheduler

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestSchedulerBudgetGuardrail(t *testing.T) {
	s := New()
	decision := s.Decide(domain.Task{ID: "t1", BudgetCents: 200}, QuotaSnapshot{BudgetRemaining: 100})
	if decision.Accepted {
		t.Fatalf("expected budget rejection")
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
	if decision.Reason == "" || decision.Assignment.Executor == "" {
		t.Fatalf("expected populated preemptive routing decision: %+v", decision)
	}
}
