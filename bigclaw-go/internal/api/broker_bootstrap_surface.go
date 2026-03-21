package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"bigclaw-go/internal/events"
)

const brokerBootstrapSurfacePath = "docs/reports/broker-validation-summary.json"
const brokerBootstrapOperatorGuidePath = "docs/reports/broker-event-log-adapter-contract.md"

type brokerBootstrapConfigDiagnostics struct {
	MissingFields      []string                        `json:"missing_fields,omitempty"`
	RequiredEnv        []string                        `json:"required_env,omitempty"`
	MissingRequiredEnv []string                        `json:"missing_required_env,omitempty"`
	AdvisoryEnv        []string                        `json:"advisory_env,omitempty"`
	MissingAdvisoryEnv []string                        `json:"missing_advisory_env,omitempty"`
	RuntimeKnobs       brokerBootstrapRuntimeKnobs     `json:"runtime_knobs"`
	EnvGuidance        []brokerBootstrapEnvExpectation `json:"env_guidance,omitempty"`
	NextActions        []string                        `json:"next_actions,omitempty"`
	ReferenceDocs      []string                        `json:"reference_docs,omitempty"`
}

type brokerBootstrapRuntimeGate struct {
	Status                 string                                  `json:"status,omitempty"`
	Requested              bool                                    `json:"requested"`
	FailClosed             bool                                    `json:"fail_closed"`
	ContractOnly           bool                                    `json:"contract_only"`
	StubDriverOnly         bool                                    `json:"stub_driver_only"`
	LiveAdapterImplemented bool                                    `json:"live_adapter_implemented"`
	BootstrapReady         bool                                    `json:"bootstrap_ready"`
	SafeForLiveTraffic     bool                                    `json:"safe_for_live_traffic"`
	ProofBoundary          string                                  `json:"proof_boundary,omitempty"`
	OperatorMessage        string                                  `json:"operator_message,omitempty"`
	TransitionGuide        []brokerBootstrapPostureTransitionGuide `json:"transition_guide,omitempty"`
}

type brokerBootstrapPostureTransitionGuide struct {
	Posture string `json:"posture"`
	Trigger string `json:"trigger"`
	Meaning string `json:"meaning"`
}

type brokerBootstrapRuntimeKnobs struct {
	PublishTimeout     string `json:"publish_timeout,omitempty"`
	ReplayLimit        int    `json:"replay_limit,omitempty"`
	CheckpointInterval string `json:"checkpoint_interval,omitempty"`
	ConsumerGroup      string `json:"consumer_group,omitempty"`
}

type brokerBootstrapEnvExpectation struct {
	Name      string `json:"name"`
	Field     string `json:"field"`
	Required  bool   `json:"required"`
	Satisfied bool   `json:"satisfied"`
	Purpose   string `json:"purpose"`
}

type brokerBootstrapSurface struct {
	ReportPath                    string                              `json:"report_path"`
	Enabled                       bool                                `json:"enabled"`
	Backend                       string                              `json:"backend,omitempty"`
	BundleSummaryPath             string                              `json:"bundle_summary_path,omitempty"`
	CanonicalSummaryPath          string                              `json:"canonical_summary_path,omitempty"`
	BundleBootstrapSummaryPath    string                              `json:"bundle_bootstrap_summary_path,omitempty"`
	CanonicalBootstrapSummaryPath string                              `json:"canonical_bootstrap_summary_path,omitempty"`
	ValidationPackPath            string                              `json:"validation_pack_path,omitempty"`
	ConfigurationState            string                              `json:"configuration_state,omitempty"`
	BootstrapSummary              events.BrokerBootstrapReviewSummary `json:"bootstrap_summary"`
	BootstrapReady                bool                                `json:"bootstrap_ready"`
	RuntimePosture                string                              `json:"runtime_posture,omitempty"`
	LiveAdapterImplemented        bool                                `json:"live_adapter_implemented"`
	ProofBoundary                 string                              `json:"proof_boundary,omitempty"`
	ValidationErrors              []string                            `json:"validation_errors,omitempty"`
	ConfigCompleteness            events.BrokerBootstrapCompleteness  `json:"config_completeness"`
	ConfigDiagnostics             brokerBootstrapConfigDiagnostics    `json:"config_diagnostics"`
	RuntimeGate                   brokerBootstrapRuntimeGate          `json:"runtime_gate"`
	Status                        string                              `json:"status,omitempty"`
	Reason                        string                              `json:"reason,omitempty"`
	Error                         string                              `json:"error,omitempty"`
}

func brokerBootstrapSurfacePayload() brokerBootstrapSurface {
	surface := brokerBootstrapSurface{ReportPath: brokerBootstrapSurfacePath}
	reportPath := resolveRepoRelativePath(brokerBootstrapSurfacePath)
	if reportPath == "" {
		surface.Status = "unavailable"
		surface.Error = "report path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	if err := json.Unmarshal(contents, &surface); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", brokerBootstrapSurfacePath, err)
		return surface
	}
	surface.ReportPath = brokerBootstrapSurfacePath
	surface.ConfigDiagnostics = buildBrokerBootstrapConfigDiagnostics(surface)
	surface.RuntimeGate = buildBrokerBootstrapRuntimeGate(surface)
	return surface
}

func buildBrokerBootstrapConfigDiagnostics(surface brokerBootstrapSurface) brokerBootstrapConfigDiagnostics {
	diagnostics := brokerBootstrapConfigDiagnostics{
		RequiredEnv: []string{
			"BIGCLAW_EVENT_LOG_BROKER_DRIVER",
			"BIGCLAW_EVENT_LOG_BROKER_URLS",
			"BIGCLAW_EVENT_LOG_BROKER_TOPIC",
		},
		AdvisoryEnv: []string{
			"BIGCLAW_EVENT_LOG_CONSUMER_GROUP",
			"BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT",
			"BIGCLAW_EVENT_LOG_REPLAY_LIMIT",
			"BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL",
		},
		RuntimeKnobs: brokerBootstrapRuntimeKnobs{},
		ReferenceDocs: []string{
			brokerBootstrapOperatorGuidePath,
			brokerBootstrapSurfacePath,
		},
	}
	if surface.ValidationPackPath != "" {
		diagnostics.ReferenceDocs = append(diagnostics.ReferenceDocs, surface.ValidationPackPath)
	}
	bootstrap := surface.BootstrapSummary.BrokerBootstrap
	if bootstrap != nil {
		diagnostics.RuntimeKnobs.PublishTimeout = bootstrap.PublishTimeout
		diagnostics.RuntimeKnobs.ReplayLimit = bootstrap.ReplayLimit
		diagnostics.RuntimeKnobs.CheckpointInterval = bootstrap.CheckpointInterval
		diagnostics.RuntimeKnobs.ConsumerGroup = strings.TrimSpace(bootstrap.ConsumerGroup)
	}
	if !surface.ConfigCompleteness.Driver {
		diagnostics.MissingFields = append(diagnostics.MissingFields, "driver")
		diagnostics.MissingRequiredEnv = append(diagnostics.MissingRequiredEnv, "BIGCLAW_EVENT_LOG_BROKER_DRIVER")
	}
	if !surface.ConfigCompleteness.URLs {
		diagnostics.MissingFields = append(diagnostics.MissingFields, "urls")
		diagnostics.MissingRequiredEnv = append(diagnostics.MissingRequiredEnv, "BIGCLAW_EVENT_LOG_BROKER_URLS")
	}
	if !surface.ConfigCompleteness.Topic {
		diagnostics.MissingFields = append(diagnostics.MissingFields, "topic")
		diagnostics.MissingRequiredEnv = append(diagnostics.MissingRequiredEnv, "BIGCLAW_EVENT_LOG_BROKER_TOPIC")
	}
	if !surface.ConfigCompleteness.ConsumerGroup {
		diagnostics.MissingAdvisoryEnv = append(diagnostics.MissingAdvisoryEnv, "BIGCLAW_EVENT_LOG_CONSUMER_GROUP")
	}
	diagnostics.EnvGuidance = []brokerBootstrapEnvExpectation{
		{
			Name:      "BIGCLAW_EVENT_LOG_BROKER_DRIVER",
			Field:     "driver",
			Required:  true,
			Satisfied: surface.ConfigCompleteness.Driver,
			Purpose:   "select the broker adapter implementation such as kafka or nats-jetstream",
		},
		{
			Name:      "BIGCLAW_EVENT_LOG_BROKER_URLS",
			Field:     "urls",
			Required:  true,
			Satisfied: surface.ConfigCompleteness.URLs,
			Purpose:   "declare the broker bootstrap endpoints as a comma-separated list",
		},
		{
			Name:      "BIGCLAW_EVENT_LOG_BROKER_TOPIC",
			Field:     "topic",
			Required:  true,
			Satisfied: surface.ConfigCompleteness.Topic,
			Purpose:   "pin the shared event stream, topic, or subject family",
		},
		{
			Name:      "BIGCLAW_EVENT_LOG_CONSUMER_GROUP",
			Field:     "consumer_group",
			Required:  false,
			Satisfied: surface.ConfigCompleteness.ConsumerGroup,
			Purpose:   "stabilize replay and checkpoint ownership for reviewer or rollout lanes",
		},
		{
			Name:      "BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT",
			Field:     "publish_timeout",
			Required:  false,
			Satisfied: diagnostics.RuntimeKnobs.PublishTimeout != "",
			Purpose:   "bound broker publish latency before fail-closed fallback engages",
		},
		{
			Name:      "BIGCLAW_EVENT_LOG_REPLAY_LIMIT",
			Field:     "replay_limit",
			Required:  false,
			Satisfied: diagnostics.RuntimeKnobs.ReplayLimit > 0,
			Purpose:   "cap replay catch-up batch size during bootstrap and recovery",
		},
		{
			Name:      "BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL",
			Field:     "checkpoint_interval",
			Required:  false,
			Satisfied: diagnostics.RuntimeKnobs.CheckpointInterval != "",
			Purpose:   "control checkpoint cadence for replicated consumers",
		},
	}
	if len(diagnostics.MissingRequiredEnv) > 0 {
		diagnostics.NextActions = append(diagnostics.NextActions,
			"Set BIGCLAW_EVENT_LOG_BROKER_DRIVER to the broker adapter key (for example kafka or nats-jetstream).",
			"Set BIGCLAW_EVENT_LOG_BROKER_URLS to the broker bootstrap endpoints as a comma-separated list.",
			"Set BIGCLAW_EVENT_LOG_BROKER_TOPIC to the shared event stream, topic, or subject family.",
		)
	}
	if len(diagnostics.MissingAdvisoryEnv) > 0 {
		diagnostics.NextActions = append(diagnostics.NextActions,
			"Optionally set BIGCLAW_EVENT_LOG_CONSUMER_GROUP to pin replay/checkpoint identity for reviewer runs.",
		)
	}
	if diagnostics.RuntimeKnobs.PublishTimeout != "" || diagnostics.RuntimeKnobs.ReplayLimit > 0 || diagnostics.RuntimeKnobs.CheckpointInterval != "" {
		diagnostics.NextActions = append(diagnostics.NextActions,
			fmt.Sprintf("Review BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT, BIGCLAW_EVENT_LOG_REPLAY_LIMIT, and BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL; resolved runtime knobs are publish_timeout=%s replay_limit=%d checkpoint_interval=%s.", firstNonEmpty(diagnostics.RuntimeKnobs.PublishTimeout, "unknown"), diagnostics.RuntimeKnobs.ReplayLimit, firstNonEmpty(diagnostics.RuntimeKnobs.CheckpointInterval, "unknown")),
		)
	}
	return diagnostics
}

func buildBrokerBootstrapRuntimeGate(surface brokerBootstrapSurface) brokerBootstrapRuntimeGate {
	currentBackend := string(surface.BootstrapSummary.EventLogBackend)
	targetBackend := string(surface.BootstrapSummary.TargetBackend)
	gate := brokerBootstrapRuntimeGate{
		Status:                 surface.RuntimePosture,
		Requested:              currentBackend == string(events.DurabilityBackendBrokerReplicated) || targetBackend == string(events.DurabilityBackendBrokerReplicated),
		FailClosed:             surface.RuntimePosture == "fail_closed_until_adapter_exists",
		ContractOnly:           surface.RuntimePosture == "contract_only",
		StubDriverOnly:         surface.RuntimePosture == "stub_driver_only",
		LiveAdapterImplemented: surface.LiveAdapterImplemented,
		BootstrapReady:         surface.BootstrapReady,
		SafeForLiveTraffic:     surface.LiveAdapterImplemented && surface.BootstrapReady && surface.RuntimePosture != "contract_only" && surface.RuntimePosture != "stub_driver_only" && surface.RuntimePosture != "fail_closed_until_adapter_exists",
		ProofBoundary:          surface.ProofBoundary,
		TransitionGuide: []brokerBootstrapPostureTransitionGuide{
			{
				Posture: "contract_only",
				Trigger: "target backend is broker_replicated while the current runtime stays on a non-broker backend",
				Meaning: "runtime exposes only the pre-adapter contract surface and must not be treated as live broker durability proof",
			},
			{
				Posture: "stub_driver_only",
				Trigger: "current backend is broker_replicated and the configured driver is the deterministic stub",
				Meaning: "runtime can exercise local broker scaffolding but still does not prove a native broker adapter",
			},
			{
				Posture: "fail_closed_until_adapter_exists",
				Trigger: "current backend is broker_replicated with non-stub config before a native adapter ships",
				Meaning: "runtime validates bootstrap config and then fails closed instead of claiming live broker support",
			},
		},
	}
	switch surface.RuntimePosture {
	case "contract_only":
		gate.OperatorMessage = "Broker durability remains a contract-only target: bootstrap evidence is reviewer-visible, but the runtime is not executing a native broker adapter."
	case "stub_driver_only":
		gate.OperatorMessage = "Broker runtime is using the deterministic stub driver only: this path is scaffolding and cannot be treated as live broker durability proof."
	case "fail_closed_until_adapter_exists":
		gate.OperatorMessage = "Broker runtime validates the configured bootstrap settings and then fails closed until a native broker adapter exists."
	case "not_requested":
		gate.OperatorMessage = "Broker replicated durability is not requested by the current runtime plan."
	default:
		gate.OperatorMessage = "Broker runtime posture is explicit, but still must be checked against proof boundary and live-adapter state before rollout claims."
	}
	return gate
}
