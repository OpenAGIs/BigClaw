package events

import (
	"testing"
	"time"
)

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

func TestNewDurabilityPlanWithBrokerConfigIncludesBootstrapStatus(t *testing.T) {
	plan := NewDurabilityPlanWithBrokerConfig("sqlite", "broker_replicated", 3, BrokerRuntimeConfig{
		Driver:             "kafka",
		URLs:               []string{"kafka-1:9092", "kafka-2:9092"},
		Topic:              "bigclaw.events",
		ConsumerGroup:      "bigclaw-consumers",
		PublishTimeout:     7 * time.Second,
		ReplayLimit:        2048,
		CheckpointInterval: 15 * time.Second,
	})

	if plan.BrokerBootstrap == nil {
		t.Fatal("expected broker bootstrap status")
	}
	if !plan.BrokerBootstrap.Ready {
		t.Fatalf("expected broker bootstrap to be ready, got %+v", plan.BrokerBootstrap)
	}
	if plan.BrokerBootstrap.Driver != "kafka" || plan.BrokerBootstrap.Topic != "bigclaw.events" {
		t.Fatalf("unexpected broker bootstrap payload: %+v", plan.BrokerBootstrap)
	}
	if plan.BrokerBootstrap.ReplayLimit != 2048 || plan.BrokerBootstrap.CheckpointInterval != "15s" {
		t.Fatalf("unexpected broker bootstrap timings: %+v", plan.BrokerBootstrap)
	}
}
