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

type RolloutScorecard struct {
	Status          string                     `json:"status"`
	Ready           bool                       `json:"ready"`
	TargetPhase     string                     `json:"target_phase"`
	Summary         string                     `json:"summary"`
	Blockers        []RolloutBlocker           `json:"blockers,omitempty"`
	MissingEvidence []RolloutMissingEvidence   `json:"missing_evidence,omitempty"`
	Checks          []RolloutCheckStatus       `json:"checks"`
	Verification    []RolloutEvidenceStatus    `json:"verification"`
	BrokerBootstrap *RolloutBootstrapReadiness `json:"broker_bootstrap,omitempty"`
}

type RolloutBlocker struct {
	Code    string `json:"code"`
	Source  string `json:"source"`
	Message string `json:"message"`
}

type RolloutMissingEvidence struct {
	Code     string `json:"code"`
	Evidence string `json:"evidence"`
	Artifact string `json:"artifact"`
	Reason   string `json:"reason"`
}

type RolloutCheckStatus struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	Blocking    bool   `json:"blocking"`
	Requirement string `json:"requirement"`
	FailureMode string `json:"failure_mode"`
	Reason      string `json:"reason"`
}

type RolloutEvidenceStatus struct {
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	Reason             string   `json:"reason"`
	Artifacts          []string `json:"artifacts,omitempty"`
	AvailableArtifacts []string `json:"available_artifacts,omitempty"`
	MissingArtifacts   []string `json:"missing_artifacts,omitempty"`
}

type RolloutBootstrapReadiness struct {
	Ready            bool     `json:"ready"`
	Status           string   `json:"status"`
	Driver           string   `json:"driver,omitempty"`
	URLs             []string `json:"urls,omitempty"`
	Topic            string   `json:"topic,omitempty"`
	ConsumerGroup    string   `json:"consumer_group,omitempty"`
	ValidationErrors []string `json:"validation_errors,omitempty"`
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
	RolloutScorecard     RolloutScorecard       `json:"rollout_scorecard"`
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
					"docs/reports/broker-durability-rollout-scorecard.json",
				},
			},
		},
	}
	if current.Replicated || target.Replicated {
		plan.BrokerBootstrap = BrokerBootstrapStatusFromConfig(broker)
	}
	plan.RolloutScorecard = BuildRolloutScorecard(plan)
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

func BuildRolloutScorecard(plan DurabilityPlan) RolloutScorecard {
	scorecard := RolloutScorecard{
		Status:       "ready",
		Ready:        true,
		TargetPhase:  rolloutTargetPhase(plan),
		Checks:       make([]RolloutCheckStatus, 0, len(plan.RolloutChecks)),
		Verification: make([]RolloutEvidenceStatus, 0, len(plan.VerificationEvidence)),
	}

	for _, check := range plan.RolloutChecks {
		scorecard.Checks = append(scorecard.Checks, RolloutCheckStatus{
			Name:        check.Name,
			Status:      "pending_validation",
			Blocking:    true,
			Requirement: check.Requirement,
			FailureMode: check.FailureMode,
			Reason:      "replicated durability contract is defined, but repo-native validation is still incomplete",
		})
	}

	for _, evidence := range plan.VerificationEvidence {
		status := classifyVerificationEvidence(evidence)
		scorecard.Verification = append(scorecard.Verification, status)
		for _, artifact := range status.MissingArtifacts {
			scorecard.MissingEvidence = append(scorecard.MissingEvidence, RolloutMissingEvidence{
				Code:     "missing_verification_artifact",
				Evidence: evidence.Name,
				Artifact: artifact,
				Reason:   status.Reason,
			})
		}
		if status.Status != "available" {
			scorecard.Blockers = append(scorecard.Blockers, RolloutBlocker{
				Code:    "verification_evidence_incomplete",
				Source:  evidence.Name,
				Message: status.Reason,
			})
		}
	}

	if plan.BrokerBootstrap != nil {
		scorecard.BrokerBootstrap = &RolloutBootstrapReadiness{
			Ready:            plan.BrokerBootstrap.Ready,
			Status:           "ready",
			Driver:           plan.BrokerBootstrap.Driver,
			URLs:             append([]string(nil), plan.BrokerBootstrap.URLs...),
			Topic:            plan.BrokerBootstrap.Topic,
			ConsumerGroup:    plan.BrokerBootstrap.ConsumerGroup,
			ValidationErrors: append([]string(nil), plan.BrokerBootstrap.ValidationErrors...),
		}
		if !plan.BrokerBootstrap.Ready {
			scorecard.BrokerBootstrap.Status = "blocked"
			scorecard.Blockers = append(scorecard.Blockers, RolloutBlocker{
				Code:    "broker_bootstrap_not_ready",
				Source:  "broker_bootstrap",
				Message: firstNonEmpty(plan.BrokerBootstrap.ValidationErrors...),
			})
		}
	}

	if len(scorecard.Blockers) > 0 {
		scorecard.Status = "blocked"
		scorecard.Ready = false
	}
	scorecard.Summary = rolloutSummary(plan, scorecard)
	return scorecard
}

func rolloutTargetPhase(plan DurabilityPlan) string {
	if !plan.Target.Replicated {
		return "single_backend_trial"
	}
	return "rollout_ready"
}

func classifyVerificationEvidence(evidence VerificationEvidence) RolloutEvidenceStatus {
	status := RolloutEvidenceStatus{
		Name:      evidence.Name,
		Artifacts: append([]string(nil), evidence.Artifacts...),
	}
	for _, artifact := range evidence.Artifacts {
		if artifactAvailableForRollout(artifact) {
			status.AvailableArtifacts = append(status.AvailableArtifacts, artifact)
			continue
		}
		status.MissingArtifacts = append(status.MissingArtifacts, artifact)
	}
	switch {
	case len(status.MissingArtifacts) == 0:
		status.Status = "available"
		status.Reason = "all declared artifacts are available in the current contract"
	case len(status.AvailableArtifacts) == 0:
		status.Status = "missing"
		status.Reason = "all evidence for this gate is still pending implementation or validation"
	default:
		status.Status = "partial"
		status.Reason = "some evidence is declared, but rollout still depends on pending artifacts"
	}
	return status
}

func artifactAvailableForRollout(artifact string) bool {
	normalized := strings.ToLower(strings.TrimSpace(artifact))
	if strings.HasPrefix(normalized, "future ") {
		return false
	}
	return normalized != ""
}

func rolloutSummary(plan DurabilityPlan, scorecard RolloutScorecard) string {
	if scorecard.Ready {
		return "replicated durability rollout contract is fully satisfied"
	}
	if plan.BrokerBootstrap != nil && !plan.BrokerBootstrap.Ready {
		return "replicated durability is blocked by broker bootstrap readiness and incomplete validation evidence"
	}
	return "replicated durability is blocked until the remaining verification evidence is available"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return "missing readiness detail"
}
