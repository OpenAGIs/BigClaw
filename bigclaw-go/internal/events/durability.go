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

type DurabilityPlan struct {
	Current              DurabilityProfile `json:"current"`
	Target               DurabilityProfile `json:"target"`
	ReplicationFactor    int               `json:"replication_factor"`
	RequiresPublisherAck bool              `json:"requires_publisher_ack"`
	MigrationConstraints []string          `json:"migration_constraints"`
	IntegrationPoints    []string          `json:"integration_points"`
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
	if replicationFactor <= 0 {
		replicationFactor = 3
	}
	current := DurabilityProfileForBackend(NormalizeDurabilityBackend(currentBackend))
	target := DurabilityProfileForBackend(NormalizeDurabilityBackend(targetBackend))
	return DurabilityPlan{
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
	}
}
