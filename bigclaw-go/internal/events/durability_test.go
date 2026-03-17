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
	if plan.VerificationEvidence[1].Artifacts[1] != "docs/reports/broker-failover-stub-report.json" {
		t.Fatalf("expected stub failover report artifact, got %+v", plan.VerificationEvidence[1])
	}
	if plan.VerificationEvidence[2].Artifacts[1] != "docs/reports/replicated-event-log-durability-rollout-contract.md" {
		t.Fatalf("expected rollout contract artifact, got %+v", plan.VerificationEvidence[2])
	}
	if plan.VerificationEvidence[2].Artifacts[2] != "docs/reports/replicated-broker-durability-rollout-spike.md" {
		t.Fatalf("expected rollout spike artifact, got %+v", plan.VerificationEvidence[2])
	}
	if plan.VerificationEvidence[2].Artifacts[3] != "docs/reports/broker-durability-rollout-scorecard.json" {
		t.Fatalf("expected broker rollout scorecard artifact, got %+v", plan.VerificationEvidence[2])
	}
	if plan.VerificationEvidence[2].Artifacts[4] != "docs/reports/durability-rollout-scorecard.json" {
		t.Fatalf("expected generic rollout scorecard artifact, got %+v", plan.VerificationEvidence[2])
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

func TestNewDurabilityPlanIncludesRolloutScorecard(t *testing.T) {
	plan := NewDurabilityPlan("http", "broker_replicated", 3)

	if plan.RolloutScorecard.Status != "blocked" || plan.RolloutScorecard.RolloutReady {
		t.Fatalf("expected blocked rollout scorecard, got %+v", plan.RolloutScorecard)
	}
	if plan.RolloutScorecard.CurrentBackend != "http" || plan.RolloutScorecard.TargetBackend != "broker_replicated" {
		t.Fatalf("unexpected rollout scorecard backends: %+v", plan.RolloutScorecard)
	}
	if plan.RolloutScorecard.BlockedChecks != len(plan.RolloutChecks) {
		t.Fatalf("expected all rollout checks blocked, got %+v", plan.RolloutScorecard)
	}
}

func TestDurabilityPlanRolloutScorecardFlagsRuntimeGaps(t *testing.T) {
	plan := NewDurabilityPlan("http", "broker_replicated", 5)
	scorecard := plan.RolloutScorecard

	if scorecard.Status != "blocked" || scorecard.RolloutReady {
		t.Fatalf("expected blocked rollout scorecard, got %+v", scorecard)
	}
	if scorecard.ReadyEvidence != 3 || scorecard.PartialEvidence != 0 || scorecard.BlockedEvidence != 1 {
		t.Fatalf("unexpected evidence counts: %+v", scorecard)
	}
	if len(scorecard.Blockers) != 2 {
		t.Fatalf("expected two rollout blockers, got %+v", scorecard.Blockers)
	}
	if scorecard.Blockers[0] != "current backend http does not yet match the replicated target broker_replicated" {
		t.Fatalf("unexpected backend blocker: %+v", scorecard.Blockers)
	}
	if scorecard.Blockers[1] != "broker bootstrap configuration is not ready: broker event log config missing driver, urls, topic" {
		t.Fatalf("unexpected bootstrap blocker: %+v", scorecard.Blockers)
	}
	for _, check := range scorecard.Checks {
		if check.Status != "blocked" {
			t.Fatalf("expected blocked rollout check, got %+v", check)
		}
	}
}

func TestDurabilityPlanRolloutScorecardKeepsFailoverGateVisibleWhenBootstrapReady(t *testing.T) {
	plan := NewDurabilityPlanWithBrokerConfig("broker_replicated", "broker_replicated", 3, BrokerRuntimeConfig{
		Driver:             "kafka",
		URLs:               []string{"kafka-1:9092", "kafka-2:9092"},
		Topic:              "bigclaw.events",
		ConsumerGroup:      "bigclaw-consumers",
		PublishTimeout:     7 * time.Second,
		ReplayLimit:        2048,
		CheckpointInterval: 15 * time.Second,
	})
	scorecard := plan.RolloutScorecard

	if scorecard.Status != "ready" || !scorecard.RolloutReady {
		t.Fatalf("expected repo-native failover evidence to unblock rollout, got %+v", scorecard)
	}
	if scorecard.ReadyEvidence != 4 || scorecard.PartialEvidence != 0 || scorecard.BlockedEvidence != 0 {
		t.Fatalf("unexpected evidence counts with ready bootstrap: %+v", scorecard)
	}
	if len(scorecard.Blockers) != 0 {
		t.Fatalf("expected no blockers with ready bootstrap and repo-native failover evidence, got %+v", scorecard.Blockers)
	}
}
