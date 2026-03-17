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

type RolloutEvidenceStatus struct {
	Name      string   `json:"name"`
	Status    string   `json:"status"`
	Artifacts []string `json:"artifacts"`
	Detail    string   `json:"detail,omitempty"`
}

type RolloutScorecardCheck struct {
	Name               string   `json:"name"`
	Status             string   `json:"status"`
	Requirement        string   `json:"requirement"`
	FailureMode        string   `json:"failure_mode"`
	SupportingEvidence []string `json:"supporting_evidence,omitempty"`
	Blockers           []string `json:"blockers,omitempty"`
}

type RolloutScorecard struct {
	Status            string                  `json:"status"`
	RolloutReady      bool                    `json:"rollout_ready"`
	CurrentBackend    DurabilityBackend       `json:"current_backend"`
	TargetBackend     DurabilityBackend       `json:"target_backend"`
	ReplicationFactor int                     `json:"replication_factor"`
	ReadyChecks       int                     `json:"ready_checks"`
	BlockedChecks     int                     `json:"blocked_checks"`
	ReadyEvidence     int                     `json:"ready_evidence"`
	PartialEvidence   int                     `json:"partial_evidence"`
	BlockedEvidence   int                     `json:"blocked_evidence"`
	Evidence          []RolloutEvidenceStatus `json:"evidence"`
	Checks            []RolloutScorecardCheck `json:"checks"`
	Blockers          []string                `json:"blockers,omitempty"`
	NextActions       []string                `json:"next_actions,omitempty"`
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
					"docs/reports/durability-rollout-scorecard.json",
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

func BuildRolloutScorecard(plan DurabilityPlan) RolloutScorecard {
	scorecard := RolloutScorecard{
		CurrentBackend:    plan.Current.Backend,
		TargetBackend:     plan.Target.Backend,
		ReplicationFactor: plan.ReplicationFactor,
	}
	if !plan.Target.Replicated {
		scorecard.Status = "not_applicable"
		scorecard.NextActions = []string{"configure a replicated target backend before using the durability rollout scorecard"}
		return scorecard
	}

	evidence := buildRolloutEvidenceStatuses(plan)
	scorecard.Evidence = evidence
	for _, item := range evidence {
		switch item.Status {
		case "ready":
			scorecard.ReadyEvidence++
		case "partial":
			scorecard.PartialEvidence++
		default:
			scorecard.BlockedEvidence++
		}
	}

	blockers := buildRolloutBlockers(plan, evidence)
	scorecard.Blockers = blockers
	scorecard.NextActions = buildRolloutNextActions(plan, evidence)
	ready := len(blockers) == 0
	checks := make([]RolloutScorecardCheck, 0, len(plan.RolloutChecks))
	for _, check := range plan.RolloutChecks {
		entry := RolloutScorecardCheck{
			Name:               check.Name,
			Status:             "blocked",
			Requirement:        check.Requirement,
			FailureMode:        check.FailureMode,
			SupportingEvidence: rolloutCheckEvidence(check.Name),
			Blockers:           append([]string(nil), blockers...),
		}
		if ready {
			entry.Status = "ready"
			entry.Blockers = nil
			scorecard.ReadyChecks++
		} else {
			scorecard.BlockedChecks++
		}
		checks = append(checks, entry)
	}
	scorecard.Checks = checks
	if ready {
		scorecard.Status = "ready"
		scorecard.RolloutReady = true
	} else {
		scorecard.Status = "blocked"
	}
	return scorecard
}

func buildRolloutEvidenceStatuses(plan DurabilityPlan) []RolloutEvidenceStatus {
	evidence := make([]RolloutEvidenceStatus, 0, len(plan.VerificationEvidence)+1)
	for _, item := range plan.VerificationEvidence {
		status := "ready"
		detail := "repo advertises concrete supporting artifacts for reviewer inspection"
		switch {
		case len(item.Artifacts) == 0:
			status = "blocked"
			detail = "no supporting artifacts are declared for this rollout signal"
		case hasFutureArtifact(item.Artifacts):
			status = "partial"
			detail = "repo includes the rollout contract, but at least one required scenario output is still a future placeholder"
		}
		evidence = append(evidence, RolloutEvidenceStatus{
			Name:      item.Name,
			Status:    status,
			Artifacts: append([]string(nil), item.Artifacts...),
			Detail:    detail,
		})
	}
	if plan.Current.Replicated || plan.Target.Replicated {
		status := RolloutEvidenceStatus{
			Name:   "broker_bootstrap_config",
			Status: "blocked",
			Detail: "broker runtime configuration is not yet valid for a replicated backend",
		}
		if plan.BrokerBootstrap != nil {
			status.Artifacts = []string{brokerBootstrapArtifactLabel(plan.BrokerBootstrap)}
			if plan.BrokerBootstrap.Ready {
				status.Status = "ready"
				status.Detail = "broker runtime configuration validates for replicated publish and replay setup"
			} else if len(plan.BrokerBootstrap.ValidationErrors) > 0 {
				status.Detail = strings.Join(plan.BrokerBootstrap.ValidationErrors, "; ")
			}
		}
		evidence = append(evidence, status)
	}
	return evidence
}

func buildRolloutBlockers(plan DurabilityPlan, evidence []RolloutEvidenceStatus) []string {
	blockers := make([]string, 0, 3)
	if plan.Current.Backend != plan.Target.Backend || !plan.Current.Replicated {
		blockers = append(blockers, "current backend "+string(plan.Current.Backend)+" does not yet match the replicated target "+string(plan.Target.Backend))
	}
	if plan.BrokerBootstrap == nil {
		blockers = append(blockers, "broker bootstrap status is missing for the replicated target")
	} else if !plan.BrokerBootstrap.Ready {
		message := "broker bootstrap configuration is not ready"
		if len(plan.BrokerBootstrap.ValidationErrors) > 0 {
			message += ": " + strings.Join(plan.BrokerBootstrap.ValidationErrors, "; ")
		}
		blockers = append(blockers, message)
	}
	for _, item := range evidence {
		if item.Name == "replay_and_failover_validation" && item.Status != "ready" {
			blockers = append(blockers, "failover validation evidence is incomplete because scenario outputs are still placeholders")
			break
		}
	}
	return blockers
}

func buildRolloutNextActions(plan DurabilityPlan, evidence []RolloutEvidenceStatus) []string {
	actions := make([]string, 0, 3)
	if plan.Current.Backend != plan.Target.Backend || !plan.Current.Replicated {
		actions = append(actions, "switch BIGCLAW_EVENT_LOG_BACKEND to "+string(plan.Target.Backend)+" after the replicated adapter is wired into runtime paths")
	}
	if plan.BrokerBootstrap == nil || !plan.BrokerBootstrap.Ready {
		actions = append(actions, "set BIGCLAW_EVENT_LOG_BROKER_DRIVER, BIGCLAW_EVENT_LOG_BROKER_URLS, and BIGCLAW_EVENT_LOG_BROKER_TOPIC so broker bootstrap becomes ready")
	}
	for _, item := range evidence {
		if item.Name == "replay_and_failover_validation" && item.Status != "ready" {
			actions = append(actions, "replace the future broker failover placeholder with checked-in scenario outputs under docs/reports/")
			break
		}
	}
	return actions
}

func rolloutCheckEvidence(name string) []string {
	switch name {
	case "durable_publish_ack":
		return []string{"operator_rollout_contract", "broker_bootstrap_config", "debug_and_control_plane_surface"}
	case "replay_checkpoint_alignment":
		return []string{"operator_rollout_contract", "replay_and_failover_validation", "broker_bootstrap_config"}
	case "retention_boundary_visibility":
		return []string{"debug_and_control_plane_surface", "replay_and_failover_validation"}
	case "live_fanout_isolation":
		return []string{"debug_and_control_plane_surface", "replay_and_failover_validation"}
	default:
		return nil
	}
}

func brokerBootstrapArtifactLabel(status *BrokerBootstrapStatus) string {
	if status == nil {
		return "broker runtime configuration"
	}
	parts := make([]string, 0, 3)
	if strings.TrimSpace(status.Driver) != "" {
		parts = append(parts, "driver="+status.Driver)
	}
	if strings.TrimSpace(status.Topic) != "" {
		parts = append(parts, "topic="+status.Topic)
	}
	if len(status.URLs) > 0 {
		parts = append(parts, "urls="+strings.Join(status.URLs, ","))
	}
	if len(parts) == 0 {
		return "broker runtime configuration"
	}
	return "broker runtime configuration (" + strings.Join(parts, "; ") + ")"
}

func hasFutureArtifact(artifacts []string) bool {
	for _, artifact := range artifacts {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(artifact)), "future ") {
			return true
		}
	}
	return false
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
