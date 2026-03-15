package events

import "testing"

func TestNewDurabilityPlanForReplicatedTargetIncludesRolloutContract(t *testing.T) {
	plan := NewDurabilityPlan("http", "broker_replicated", 5)

	if !plan.RequiresPublisherAck {
		t.Fatal("expected replicated target to require publisher acknowledgements")
	}
	if len(plan.RolloutChecks) < 4 {
		t.Fatalf("expected rollout checks for replicated target, got %+v", plan.RolloutChecks)
	}
	if plan.RolloutChecks[0].Name != "durable_publish_ack" {
		t.Fatalf("unexpected first rollout check: %+v", plan.RolloutChecks[0])
	}
	if len(plan.FailureDomains) < 3 {
		t.Fatalf("expected failure domains for replicated target, got %+v", plan.FailureDomains)
	}
	if plan.FailureDomains[1].Name != "checkpoint_store_failover" {
		t.Fatalf("unexpected checkpoint failure domain: %+v", plan.FailureDomains[1])
	}
	if len(plan.VerificationEvidence) < 3 {
		t.Fatalf("expected verification evidence entries, got %+v", plan.VerificationEvidence)
	}
	if plan.VerificationEvidence[2].Artifacts[1] != "docs/reports/replicated-event-log-durability-rollout-contract.md" {
		t.Fatalf("expected rollout contract artifact, got %+v", plan.VerificationEvidence[2])
	}
}
