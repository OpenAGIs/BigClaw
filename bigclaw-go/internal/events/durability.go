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

type BrokerFailureModeSummary struct {
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	Mitigation string `json:"mitigation"`
}

type BrokerAdapterDryRun struct {
	Mode                  string                     `json:"mode"`
	ImplementationState   string                     `json:"implementation_state"`
	Driver                string                     `json:"driver,omitempty"`
	URLs                  []string                   `json:"urls,omitempty"`
	Topic                 string                     `json:"topic,omitempty"`
	ConsumerGroup         string                     `json:"consumer_group,omitempty"`
	PublishTimeout        string                     `json:"publish_timeout,omitempty"`
	ReplayLimit           int                        `json:"replay_limit,omitempty"`
	CheckpointInterval    string                     `json:"checkpoint_interval,omitempty"`
	Ready                 bool                       `json:"ready"`
	ValidationErrors      []string                   `json:"validation_errors,omitempty"`
	Capabilities          BackendCapabilities        `json:"capabilities"`
	OrderingScope         string                     `json:"ordering_scope"`
	PortablePositionKeys  []string                   `json:"portable_position_keys"`
	FailureModes          []BrokerFailureModeSummary `json:"failure_modes"`
	VerificationArtifacts []string                   `json:"verification_artifacts"`
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
	BrokerProbe          *BrokerAdapterDryRun   `json:"broker_probe,omitempty"`
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
		plan.BrokerProbe = BrokerAdapterDryRunFromConfig(broker)
	}
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
	if issues := cfg.ValidationErrors(); len(issues) > 0 {
		status.ValidationErrors = issues
		return status
	}
	status.Ready = true
	return status
}

func BrokerAdapterDryRunFromConfig(cfg BrokerRuntimeConfig) *BrokerAdapterDryRun {
	probe := &BrokerAdapterDryRun{
		Mode:                "dry_run",
		ImplementationState: "planned_only",
		Driver:              strings.TrimSpace(cfg.Driver),
		URLs:                append([]string(nil), cfg.URLs...),
		Topic:               strings.TrimSpace(cfg.Topic),
		ConsumerGroup:       strings.TrimSpace(cfg.ConsumerGroup),
		PublishTimeout:      cfg.PublishTimeout.String(),
		ReplayLimit:         cfg.ReplayLimit,
		CheckpointInterval:  cfg.CheckpointInterval.String(),
		Capabilities: BackendCapabilities{
			Backend: "broker_adapter",
			Scope:   "provider_contract",
			Publish: FeatureSupport{
				Supported: true,
				Mode:      "replicated_ack_required",
				Detail:    "Success is expected to mean replicated commit acknowledgement rather than leader-local enqueue.",
			},
			Replay: FeatureSupport{
				Supported: true,
				Mode:      "portable_sequence_cursor",
				Detail:    "Replay is expected to map provider offsets back to Position.Sequence for resume safety.",
			},
			Checkpoint: FeatureSupport{
				Supported: true,
				Mode:      "durable_consumer_progress",
				Detail:    "Checkpoint writes must stay monotonic in the durable sequence space across failover.",
			},
			Dedup: FeatureSupport{
				Supported: false,
				Detail:    "Consumer dedup persistence remains a separate contract from the broker event-log adapter.",
			},
			Filtering: FeatureSupport{
				Supported: true,
				Mode:      "derived_task_trace_filters",
				Detail:    "Task and trace filtering must remain stable across replay even if the provider exposes different native selectors.",
			},
			Retention: FeatureSupport{
				Supported: true,
				Mode:      "retention_boundary_visible",
				Detail:    "The adapter must expose oldest/newest retained replay boundaries before resumable recovery is claimed.",
			},
		},
		OrderingScope:        "partition or quorum log order",
		PortablePositionKeys: []string{"sequence", "partition", "offset"},
		FailureModes: []BrokerFailureModeSummary{
			{
				Name:       "ambiguous_publish_outcome",
				Summary:    "A leader crash or timeout can leave publish outcome unknown without replay-visible reconciliation.",
				Mitigation: "Require replicated acknowledgements and surface unknown-commit outcomes explicitly.",
			},
			{
				Name:       "checkpoint_fencing_gap",
				Summary:    "Consumer takeover can regress durable progress if stale writers are not fenced.",
				Mitigation: "Persist checkpoint sequence with lease or epoch metadata and reject stale writes.",
			},
			{
				Name:       "retention_boundary_drift",
				Summary:    "Expired checkpoints can appear structurally valid after compaction even when the log window moved forward.",
				Mitigation: "Expose retention boundaries and fail closed until an operator chooses a reset action.",
			},
		},
		VerificationArtifacts: []string{
			"docs/reports/broker-event-log-adapter-contract.md",
			"docs/reports/replicated-event-log-durability-rollout-contract.md",
			"docs/reports/broker-failover-fault-injection-validation-pack.md",
		},
	}
	if issues := cfg.ValidationErrors(); len(issues) > 0 {
		probe.ValidationErrors = issues
		return probe
	}
	probe.Ready = true
	return probe
}
