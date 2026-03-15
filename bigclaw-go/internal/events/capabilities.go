package events

import (
	"context"
	"fmt"
)

type FeatureSupport struct {
	Supported bool   `json:"supported"`
	Mode      string `json:"mode,omitempty"`
	Detail    string `json:"detail,omitempty"`
}

type BackendCapabilities struct {
	Backend    string         `json:"backend"`
	Scope      string         `json:"scope,omitempty"`
	Publish    FeatureSupport `json:"publish"`
	Replay     FeatureSupport `json:"replay"`
	Checkpoint FeatureSupport `json:"checkpoint"`
	Dedup      FeatureSupport `json:"dedup"`
	Filtering  FeatureSupport `json:"filtering"`
	Retention  FeatureSupport `json:"retention"`
}

type CapabilityProvider interface {
	Capabilities(context.Context) BackendCapabilities
}

func defaultBusCapabilities() BackendCapabilities {
	return BackendCapabilities{
		Backend: "in_memory_history",
		Scope:   "process_local",
		Publish: FeatureSupport{
			Supported: true,
			Mode:      "live_fanout",
			Detail:    "Publishes to in-process subscribers and configured sinks.",
		},
		Replay: FeatureSupport{
			Supported: true,
			Mode:      "memory_history",
			Detail:    "Replay is served from process memory and resets on restart.",
		},
		Checkpoint: FeatureSupport{
			Supported: false,
			Detail:    "No durable subscriber checkpoint backend is configured.",
		},
		Dedup: FeatureSupport{
			Supported: false,
			Detail:    "Consumer dedup state is process-local and resets on restart.",
		},
		Filtering: FeatureSupport{
			Supported: true,
			Mode:      "task_id,trace_id",
			Detail:    "Runtime filtering is available on replay and live SSE requests.",
		},
		Retention: FeatureSupport{
			Supported: true,
			Mode:      "process_memory",
			Detail:    "Retention lasts only for the current process lifetime.",
		},
	}
}

func UnavailableCapabilities() BackendCapabilities {
	return BackendCapabilities{
		Backend: "unavailable",
		Scope:   "disabled",
		Publish: FeatureSupport{
			Supported: false,
			Detail:    "No event bus backend is configured.",
		},
		Replay: FeatureSupport{
			Supported: false,
			Detail:    "Replay requires an active event bus backend.",
		},
		Checkpoint: FeatureSupport{
			Supported: false,
			Detail:    "No event bus backend is configured.",
		},
		Dedup: FeatureSupport{
			Supported: false,
			Detail:    "No event bus backend is configured.",
		},
		Filtering: FeatureSupport{
			Supported: false,
			Detail:    "No event bus backend is configured.",
		},
		Retention: FeatureSupport{
			Supported: false,
			Detail:    "No event bus backend is configured.",
		},
	}
}

func BrokerBootstrapCapabilities(cfg BrokerRuntimeConfig) BackendCapabilities {
	detail := "Broker adapter bootstrap config validated; live provider wiring is not implemented yet."
	if cfg.Driver != "" || cfg.Topic != "" {
		detail = fmt.Sprintf("Broker adapter bootstrap config validated for driver=%s topic=%s; live provider wiring is not implemented yet.", cfg.Driver, cfg.Topic)
	}
	return BackendCapabilities{
		Backend: "broker",
		Scope:   "broker_bootstrap",
		Publish: FeatureSupport{
			Supported: false,
			Mode:      "contract_validated",
			Detail:    detail,
		},
		Replay: FeatureSupport{
			Supported: false,
			Mode:      "contract_validated",
			Detail:    "Replay contract and cursor expectations are reserved for the future broker adapter.",
		},
		Checkpoint: FeatureSupport{
			Supported: false,
			Mode:      "contract_validated",
			Detail:    "Checkpoint semantics are defined, but no broker-backed checkpoint store is wired yet.",
		},
		Dedup: FeatureSupport{
			Supported: false,
			Detail:    "Durable consumer dedup remains a separate backend concern until a broker adapter lands.",
		},
		Filtering: FeatureSupport{
			Supported: true,
			Mode:      "provider_defined",
			Detail:    "Broker adapters are expected to preserve task and trace filtering semantics.",
		},
		Retention: FeatureSupport{
			Supported: true,
			Mode:      "broker_managed",
			Detail:    "Retention boundaries will be enforced by the chosen broker backend once implemented.",
		},
	}
}
