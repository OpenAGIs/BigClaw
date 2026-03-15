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
	if plan.RolloutReadiness.Status != "blocked" || plan.RolloutReadiness.Phase != "contract" {
		t.Fatalf("expected blocked contract readiness without broker config, got %+v", plan.RolloutReadiness)
	}
	if len(plan.RolloutReadiness.RemainingChecks) != len(plan.RolloutChecks) {
		t.Fatalf("expected remaining checks to mirror rollout checks, got %+v", plan.RolloutReadiness.RemainingChecks)
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
	if plan.RolloutReadiness.Status != "contract_ready" || plan.RolloutReadiness.BrokerRuntime == nil || !plan.RolloutReadiness.BrokerRuntime.Ready {
		t.Fatalf("expected contract-ready rollout readiness, got %+v", plan.RolloutReadiness)
	}
}

func TestNewDurabilityPlanRolloutReadinessByBackend(t *testing.T) {
	cases := []struct {
		name                string
		current             string
		target              string
		wantStatus          string
		wantCurrentProbe    string
		wantCheckpointProbe string
	}{
		{
			name:                "memory current backend",
			current:             "memory",
			target:              "memory",
			wantStatus:          "current_backend_active",
			wantCurrentProbe:    "process_memory",
			wantCheckpointProbe: "unsupported",
		},
		{
			name:                "sqlite current backend",
			current:             "sqlite",
			target:              "sqlite",
			wantStatus:          "current_backend_active",
			wantCurrentProbe:    "durable_single_node",
			wantCheckpointProbe: "native",
		},
		{
			name:                "http current backend",
			current:             "http",
			target:              "http",
			wantStatus:          "current_backend_active",
			wantCurrentProbe:    "durable_shared_service",
			wantCheckpointProbe: "native",
		},
		{
			name:                "broker replicated target",
			current:             "http",
			target:              "broker_replicated",
			wantStatus:          "blocked",
			wantCurrentProbe:    "durable_shared_service",
			wantCheckpointProbe: "native",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := NewDurabilityPlan(tc.current, tc.target, 3)
			if plan.RolloutReadiness.Status != tc.wantStatus {
				t.Fatalf("expected status %q, got %+v", tc.wantStatus, plan.RolloutReadiness)
			}
			if plan.RolloutReadiness.CurrentProbe.Retention != tc.wantCurrentProbe {
				t.Fatalf("expected current retention probe %q, got %+v", tc.wantCurrentProbe, plan.RolloutReadiness.CurrentProbe)
			}
			if plan.RolloutReadiness.CurrentProbe.Checkpoint != tc.wantCheckpointProbe {
				t.Fatalf("expected current checkpoint probe %q, got %+v", tc.wantCheckpointProbe, plan.RolloutReadiness.CurrentProbe)
			}
			if tc.target == "broker_replicated" && plan.RolloutReadiness.TargetProbe.Retention != "replicated_log" {
				t.Fatalf("expected replicated target probe, got %+v", plan.RolloutReadiness.TargetProbe)
			}
		})
	}
}
