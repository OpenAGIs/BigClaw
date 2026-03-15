package events

import "strings"

type DurabilityBackend string

const (
	DurabilityBackendMemory           DurabilityBackend = "memory"
	DurabilityBackendSQLite           DurabilityBackend = "sqlite"
	DurabilityBackendHTTP             DurabilityBackend = "http"
	DurabilityBackendBrokerReplicated DurabilityBackend = "broker_replicated"
)

type DurabilityProfile struct {
	Backend             DurabilityBackend `json:"backend"`
	Shared              bool              `json:"shared"`
	Replicated          bool              `json:"replicated"`
	Replay              bool              `json:"replay"`
	SubscriberState     bool              `json:"subscriber_state"`
	MonotonicCheckpoint bool              `json:"monotonic_checkpoint"`
	OrderingScope       string            `json:"ordering_scope"`
}

type RolloutCheck struct {
	Name        string `json:"name"`
	Requirement string `json:"requirement"`
	FailureMode string `json:"failure_mode"`
}

type FailureDomain struct {
	Name      string   `json:"name"`
	Impact    string   `json:"impact"`
	Mitigates []string `json:"mitigates"`
}

type VerificationEvidence struct {
	Name      string   `json:"name"`
	Artifacts []string `json:"artifacts"`
}

type BrokerBootstrapStatus struct {
	Driver             string   `json:"driver,omitempty"`
	URLs               []string `json:"urls,omitempty"`
	Topic              string   `json:"topic,omitempty"`
	ConsumerGroup      string   `json:"consumer_group,omitempty"`
	PublishTimeout     string   `json:"publish_timeout,omitempty"`
	ReplayLimit        int      `json:"replay_limit,omitempty"`
	CheckpointInterval string   `json:"checkpoint_interval,omitempty"`
	Ready              bool     `json:"ready"`
	ValidationErrors   []string `json:"validation_errors,omitempty"`
}

type CapabilityProbeSummary struct {
	Publish    string `json:"publish"`
	Replay     string `json:"replay"`
	Checkpoint string `json:"checkpoint"`
	Filtering  string `json:"filtering"`
	Retention  string `json:"retention"`
}

type RolloutReadiness struct {
	Phase             string                 `json:"phase"`
	Status            string                 `json:"status"`
	Summary           string                 `json:"summary"`
	ReadinessNotes    []string               `json:"readiness_notes"`
	RemainingChecks   []string               `json:"remaining_checks,omitempty"`
	CurrentProbe      CapabilityProbeSummary `json:"current_probe"`
	TargetProbe       CapabilityProbeSummary `json:"target_probe"`
	BrokerRuntime     *BrokerBootstrapStatus `json:"broker_runtime,omitempty"`
	EvidenceArtifacts []string               `json:"evidence_artifacts,omitempty"`
}

type DurabilityPlan struct {
	Current              DurabilityProfile      `json:"current"`
	Target               DurabilityProfile      `json:"target"`
	ReplicationFactor    int                    `json:"replication_factor"`
	RequiresPublisherAck bool                   `json:"requires_publisher_ack"`
	MigrationConstraints []string               `json:"migration_constraints"`
	IntegrationPoints    []string               `json:"integration_points"`
	RolloutChecks        []RolloutCheck         `json:"rollout_checks"`
	FailureDomains       []FailureDomain        `json:"failure_domains"`
	VerificationEvidence []VerificationEvidence `json:"verification_evidence"`
	BrokerBootstrap      *BrokerBootstrapStatus `json:"broker_bootstrap,omitempty"`
	RolloutReadiness     RolloutReadiness       `json:"rollout_readiness"`
}

func NormalizeDurabilityBackend(value string) DurabilityBackend {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "memory", "in_memory", "in-memory":
		return DurabilityBackendMemory
	case "sqlite":
		return DurabilityBackendSQLite
	case "http", "remote_http", "remote-http":
		return DurabilityBackendHTTP
	case "broker", "broker_replicated", "broker-replicated", "replicated":
		return DurabilityBackendBrokerReplicated
	default:
		return DurabilityBackend(strings.ToLower(strings.TrimSpace(value)))
	}
}

func DurabilityProfileForBackend(backend DurabilityBackend) DurabilityProfile {
	switch backend {
	case DurabilityBackendSQLite:
		return DurabilityProfile{
			Backend:             backend,
			Replay:              true,
			SubscriberState:     true,
			MonotonicCheckpoint: true,
			OrderingScope:       "single-node append order",
		}
	case DurabilityBackendHTTP:
		return DurabilityProfile{
			Backend:             backend,
			Shared:              true,
			Replay:              true,
			SubscriberState:     true,
			MonotonicCheckpoint: true,
			OrderingScope:       "single-writer service order",
		}
	case DurabilityBackendBrokerReplicated:
		return DurabilityProfile{
			Backend:             backend,
			Shared:              true,
			Replicated:          true,
			Replay:              true,
			SubscriberState:     true,
			MonotonicCheckpoint: true,
			OrderingScope:       "partition or quorum log order",
		}
	default:
		return DurabilityProfile{
			Backend:       DurabilityBackendMemory,
			Replay:        true,
			OrderingScope: "process-local publish order",
		}
	}
}

func NewDurabilityPlan(currentBackend, targetBackend string, replicationFactor int) DurabilityPlan {
	return NewDurabilityPlanWithBrokerConfig(currentBackend, targetBackend, replicationFactor, BrokerRuntimeConfig{})
}

func NewDurabilityPlanWithBrokerConfig(currentBackend, targetBackend string, replicationFactor int, broker BrokerRuntimeConfig) DurabilityPlan {
	if replicationFactor <= 0 {
		replicationFactor = 3
	}
	current := DurabilityProfileForBackend(NormalizeDurabilityBackend(currentBackend))
	target := DurabilityProfileForBackend(NormalizeDurabilityBackend(targetBackend))
	plan := DurabilityPlan{
		Current:              current,
		Target:               target,
		ReplicationFactor:    replicationFactor,
		RequiresPublisherAck: target.Replicated,
		MigrationConstraints: []string{
			"preserve append-only replay semantics across backend cutover",
			"keep subscriber checkpoints monotonic during dual-write or backfill",
			"carry task_id, trace_id, and event_type as stable partitioning keys",
			"avoid coupling live SSE fanout directly to broker consumer lag",
		},
		IntegrationPoints: []string{
			"cmd/bigclawd/main.go bootstrap/backend selection",
			"internal/api/server.go debug and control-plane reporting",
			"internal/events bus publish/subscribe path",
			"subscriber checkpoint persistence and replay endpoints",
		},
		RolloutChecks: []RolloutCheck{
			{
				Name:        "durable_publish_ack",
				Requirement: "publish success must represent replicated commit acknowledgement rather than leader-local enqueue",
				FailureMode: "ambiguous client timeout or leader crash can leave operators unable to classify whether an event committed",
			},
			{
				Name:        "replay_checkpoint_alignment",
				Requirement: "replay cursors and subscriber checkpoints must resolve to the same durable sequence domain across failover",
				FailureMode: "consumer recovery can skip or duplicate committed events after broker election or reconnect",
			},
			{
				Name:        "retention_boundary_visibility",
				Requirement: "the backend must expose oldest/newest retained replay boundaries before rollout claims resumable recovery",
				FailureMode: "aged-out checkpoints can silently start from an unsafe later point instead of surfacing truncation",
			},
			{
				Name:        "live_fanout_isolation",
				Requirement: "SSE and in-process live subscribers must stay decoupled from broker catch-up lag and replay backfill",
				FailureMode: "replay recovery or broker lag can stall live delivery and blur the source of operator-visible latency",
			},
		},
		FailureDomains: []FailureDomain{
			{
				Name:   "broker_leader_or_quorum_loss",
				Impact: "publish acknowledgements and replay visibility may diverge until leadership or quorum is re-established",
				Mitigates: []string{
					"require replicated publish acknowledgements",
					"record ambiguous publish outcomes for replay reconciliation",
				},
			},
			{
				Name:   "checkpoint_store_failover",
				Impact: "stale writers can regress subscriber progress if checkpoint ownership is not fenced by durable sequence and lease epoch",
				Mitigates: []string{
					"persist monotonic checkpoint sequence with ownership metadata",
					"reject stale checkpoint writes after takeover",
				},
			},
			{
				Name:   "retention_or_compaction_drift",
				Impact: "resume requests can point outside the retained log window even when the cursor shape is valid",
				Mitigates: []string{
					"surface retention watermarks in operator diagnostics",
					"fail closed on expired checkpoints until an explicit reset policy is chosen",
				},
			},
		},
		VerificationEvidence: []VerificationEvidence{
			{
				Name: "debug_and_control_plane_surface",
				Artifacts: []string{
					"GET /debug/status event_durability payload",
					"control-plane snapshots carrying backend and rollout metadata",
				},
			},
			{
				Name: "replay_and_failover_validation",
				Artifacts: []string{
					"docs/reports/broker-failover-fault-injection-validation-pack.md",
					"future broker-failover-<backend>-report.json scenario outputs",
				},
			},
			{
				Name: "operator_rollout_contract",
				Artifacts: []string{
					"docs/reports/event-bus-reliability-report.md",
					"docs/reports/replicated-event-log-durability-rollout-contract.md",
				},
			},
		},
	}
	if current.Replicated || target.Replicated {
		plan.BrokerBootstrap = BrokerBootstrapStatusFromConfig(broker)
	}
	plan.RolloutReadiness = buildRolloutReadiness(plan)
	return plan
}

func BrokerBootstrapStatusFromConfig(cfg BrokerRuntimeConfig) *BrokerBootstrapStatus {
	status := &BrokerBootstrapStatus{
		Driver:             strings.TrimSpace(cfg.Driver),
		URLs:               append([]string(nil), cfg.URLs...),
		Topic:              strings.TrimSpace(cfg.Topic),
		ConsumerGroup:      strings.TrimSpace(cfg.ConsumerGroup),
		PublishTimeout:     cfg.PublishTimeout.String(),
		ReplayLimit:        cfg.ReplayLimit,
		CheckpointInterval: cfg.CheckpointInterval.String(),
	}
	if err := cfg.Validate(); err != nil {
		status.ValidationErrors = []string{err.Error()}
		return status
	}
	status.Ready = true
	return status
}

func buildRolloutReadiness(plan DurabilityPlan) RolloutReadiness {
	readiness := RolloutReadiness{
		CurrentProbe:      capabilityProbeSummary(plan.Current.Backend),
		TargetProbe:       capabilityProbeSummary(plan.Target.Backend),
		BrokerRuntime:     plan.BrokerBootstrap,
		EvidenceArtifacts: readinessArtifacts(plan.VerificationEvidence),
	}
	if !plan.Target.Replicated {
		readiness.Phase = "current_backend"
		readiness.Status = "current_backend_active"
		readiness.Summary = "Current event-log backend is active; replicated durability rollout is not configured."
		readiness.ReadinessNotes = []string{
			"Current runtime exposes its active backend profile directly through event_durability for operator review.",
			"Replicated publish acknowledgements and broker failover checks are not required until a broker_replicated target is selected.",
		}
		return readiness
	}

	readiness.Phase = "contract"
	readiness.RemainingChecks = remainingCheckRequirements(plan.RolloutChecks)
	readiness.ReadinessNotes = []string{
		"Replicated durability remains in the contract-validation phase until a live broker adapter satisfies publish, replay, checkpoint, and retention checks.",
		"Operator payloads should review broker bootstrap readiness together with rollout checks before claiming replicated durability.",
	}
	if plan.BrokerBootstrap == nil || !plan.BrokerBootstrap.Ready {
		readiness.Status = "blocked"
		readiness.Summary = "Replicated target is configured, but broker bootstrap validation is still blocking rollout readiness."
		if plan.BrokerBootstrap != nil && len(plan.BrokerBootstrap.ValidationErrors) > 0 {
			readiness.ReadinessNotes = append(readiness.ReadinessNotes, plan.BrokerBootstrap.ValidationErrors...)
		}
		return readiness
	}

	readiness.Status = "contract_ready"
	readiness.Summary = "Replicated target metadata is configured and validated, but rollout remains gated on the documented failover and checkpoint evidence."
	readiness.ReadinessNotes = append(readiness.ReadinessNotes,
		"Broker runtime metadata is present so release/export payloads can attach target driver, topic, and consumer-group expectations.",
	)
	return readiness
}

func capabilityProbeSummary(backend DurabilityBackend) CapabilityProbeSummary {
	switch backend {
	case DurabilityBackendSQLite:
		return CapabilityProbeSummary{
			Publish:    "native",
			Replay:     "native",
			Checkpoint: "native",
			Filtering:  "derived",
			Retention:  "durable_single_node",
		}
	case DurabilityBackendHTTP:
		return CapabilityProbeSummary{
			Publish:    "native",
			Replay:     "native",
			Checkpoint: "native",
			Filtering:  "derived",
			Retention:  "durable_shared_service",
		}
	case DurabilityBackendBrokerReplicated:
		return CapabilityProbeSummary{
			Publish:    "native",
			Replay:     "native",
			Checkpoint: "native",
			Filtering:  "derived",
			Retention:  "replicated_log",
		}
	default:
		return CapabilityProbeSummary{
			Publish:    "native",
			Replay:     "native",
			Checkpoint: "unsupported",
			Filtering:  "native",
			Retention:  "process_memory",
		}
	}
}

func remainingCheckRequirements(checks []RolloutCheck) []string {
	out := make([]string, 0, len(checks))
	for _, check := range checks {
		out = append(out, check.Requirement)
	}
	return out
}

func readinessArtifacts(entries []VerificationEvidence) []string {
	out := make([]string, 0)
	for _, entry := range entries {
		out = append(out, entry.Artifacts...)
	}
	return out
}
