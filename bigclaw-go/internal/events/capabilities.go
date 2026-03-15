package events

import "context"

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
