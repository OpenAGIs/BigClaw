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

type DurabilityComparisonRow struct {
	Backend                DurabilityBackend `json:"backend"`
	Readiness              string            `json:"readiness"`
	SelectedCurrent        bool              `json:"selected_current"`
	SelectedTarget         bool              `json:"selected_target"`
	ImplementedInBootstrap bool              `json:"implemented_in_bootstrap"`
	CapabilityProbeBackend string            `json:"capability_probe_backend"`
	CapabilityProbeStatus  string            `json:"capability_probe_status"`
	RuntimeSelectors       []string          `json:"runtime_selectors"`
	RequiredConfig         []string          `json:"required_config"`
	OrderingScope          string            `json:"ordering_scope"`
	Shared                 bool              `json:"shared"`
	Replicated             bool              `json:"replicated"`
	Replay                 bool              `json:"replay"`
	SubscriberState        bool              `json:"subscriber_state"`
	MonotonicCheckpoint    bool              `json:"monotonic_checkpoint"`
	Evidence               []string          `json:"evidence"`
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

type DurabilityPlan struct {
	Current              DurabilityProfile         `json:"current"`
	Target               DurabilityProfile         `json:"target"`
	Comparison           []DurabilityComparisonRow `json:"comparison"`
	ReplicationFactor    int                       `json:"replication_factor"`
	RequiresPublisherAck bool                      `json:"requires_publisher_ack"`
	MigrationConstraints []string                  `json:"migration_constraints"`
	IntegrationPoints    []string                  `json:"integration_points"`
	RolloutChecks        []RolloutCheck            `json:"rollout_checks"`
	FailureDomains       []FailureDomain           `json:"failure_domains"`
	VerificationEvidence []VerificationEvidence    `json:"verification_evidence"`
	BrokerBootstrap      *BrokerBootstrapStatus    `json:"broker_bootstrap,omitempty"`
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
		Comparison:           durabilityComparison(current.Backend, target.Backend),
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
	return plan
}

func durabilityComparison(current, target DurabilityBackend) []DurabilityComparisonRow {
	backends := []DurabilityBackend{
		DurabilityBackendMemory,
		DurabilityBackendSQLite,
		DurabilityBackendHTTP,
		DurabilityBackendBrokerReplicated,
	}
	rows := make([]DurabilityComparisonRow, 0, len(backends))
	for _, backend := range backends {
		rows = append(rows, durabilityComparisonRow(backend, current, target))
	}
	return rows
}

func durabilityComparisonRow(backend, current, target DurabilityBackend) DurabilityComparisonRow {
	profile := DurabilityProfileForBackend(backend)
	row := DurabilityComparisonRow{
		Backend:                backend,
		SelectedCurrent:        backend == current,
		SelectedTarget:         backend == target,
		OrderingScope:          profile.OrderingScope,
		Shared:                 profile.Shared,
		Replicated:             profile.Replicated,
		Replay:                 profile.Replay,
		SubscriberState:        profile.SubscriberState,
		MonotonicCheckpoint:    profile.MonotonicCheckpoint,
		CapabilityProbeBackend: durabilityProbeBackend(backend),
		CapabilityProbeStatus:  durabilityProbeStatus(backend),
		RuntimeSelectors:       durabilityRuntimeSelectors(backend),
		RequiredConfig:         durabilityRequiredConfig(backend),
		Evidence:               durabilityEvidence(backend),
	}

	if backend == current {
		row.Readiness = "current_runtime"
	} else if backend == target {
		row.Readiness = "target_contract"
	} else {
		row.Readiness = "comparison_candidate"
	}

	if catalog, ok := Catalog()[BackendKind(row.CapabilityProbeBackend)]; ok {
		row.ImplementedInBootstrap = catalog.Implemented
	}
	return row
}

func durabilityProbeBackend(backend DurabilityBackend) string {
	switch backend {
	case DurabilityBackendBrokerReplicated:
		return string(BackendBroker)
	case DurabilityBackendSQLite:
		return string(BackendSQLite)
	case DurabilityBackendHTTP:
		return string(BackendHTTP)
	default:
		return string(BackendMemory)
	}
}

func durabilityProbeStatus(backend DurabilityBackend) string {
	switch backend {
	case DurabilityBackendBrokerReplicated:
		return "contract_validated"
	case DurabilityBackendSQLite, DurabilityBackendHTTP:
		return "catalog_defined"
	default:
		return "implemented"
	}
}

func durabilityRuntimeSelectors(backend DurabilityBackend) []string {
	switch backend {
	case DurabilityBackendSQLite:
		return []string{
			"BIGCLAW_EVENT_LOG_SQLITE_PATH",
			"BIGCLAW_EVENT_LOG_BACKEND=sqlite",
		}
	case DurabilityBackendHTTP:
		return []string{
			"BIGCLAW_EVENT_LOG_REMOTE_URL",
			"BIGCLAW_EVENT_LOG_BACKEND=http",
		}
	case DurabilityBackendBrokerReplicated:
		return []string{
			"BIGCLAW_EVENT_LOG_TARGET_BACKEND=broker_replicated",
			"BIGCLAW_EVENT_LOG_BACKEND=broker",
		}
	default:
		return []string{"BIGCLAW_EVENT_LOG_BACKEND=memory"}
	}
}

func durabilityRequiredConfig(backend DurabilityBackend) []string {
	switch backend {
	case DurabilityBackendSQLite:
		return []string{
			"BIGCLAW_EVENT_LOG_SQLITE_PATH",
			"BIGCLAW_EVENT_RETENTION",
			"BIGCLAW_EVENT_BACKEND=sqlite",
			"BIGCLAW_EVENT_LOG_DSN",
			"BIGCLAW_EVENT_CHECKPOINT_DSN",
		}
	case DurabilityBackendHTTP:
		return []string{
			"BIGCLAW_EVENT_LOG_REMOTE_URL",
			"BIGCLAW_EVENT_RETENTION",
			"BIGCLAW_EVENT_BACKEND=http",
			"BIGCLAW_EVENT_LOG_DSN",
			"BIGCLAW_EVENT_CHECKPOINT_DSN",
		}
	case DurabilityBackendBrokerReplicated:
		return []string{
			"BIGCLAW_EVENT_LOG_TARGET_BACKEND=broker_replicated",
			"BIGCLAW_EVENT_LOG_REPLICATION_FACTOR",
			"BIGCLAW_EVENT_LOG_BROKER_DRIVER",
			"BIGCLAW_EVENT_LOG_BROKER_URLS",
			"BIGCLAW_EVENT_LOG_BROKER_TOPIC",
			"BIGCLAW_EVENT_LOG_CONSUMER_GROUP",
			"BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT",
			"BIGCLAW_EVENT_LOG_REPLAY_LIMIT",
			"BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL",
			"BIGCLAW_EVENT_BACKEND=broker",
			"BIGCLAW_EVENT_LOG_DSN",
			"BIGCLAW_EVENT_CHECKPOINT_DSN",
			"BIGCLAW_EVENT_RETENTION",
		}
	default:
		return []string{"BIGCLAW_EVENT_BACKEND=memory"}
	}
}

func durabilityEvidence(backend DurabilityBackend) []string {
	switch backend {
	case DurabilityBackendBrokerReplicated:
		return []string{
			"docs/reports/replicated-event-log-durability-rollout-contract.md",
			"docs/reports/broker-failover-fault-injection-validation-pack.md",
		}
	case DurabilityBackendSQLite, DurabilityBackendHTTP:
		return []string{
			"internal/events/backend_contract.go",
			"docs/reports/event-bus-reliability-report.md",
		}
	default:
		return []string{
			"internal/events/memory_log.go",
			"docs/reports/event-bus-reliability-report.md",
		}
	}
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
