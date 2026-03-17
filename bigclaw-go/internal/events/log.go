package events

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type EventLogBackend string

const (
	EventLogBackendMemory EventLogBackend = "memory"
	EventLogBackendBroker EventLogBackend = "broker"
)

type Position struct {
	Sequence  int64  `json:"sequence,omitempty"`
	Partition string `json:"partition,omitempty"`
	Offset    string `json:"offset,omitempty"`
}

type Record struct {
	Event       domain.Event `json:"event"`
	Position    Position     `json:"position"`
	PublishedAt time.Time    `json:"published_at"`
}

type Capabilities struct {
	Durable             bool `json:"durable"`
	OrderedReplay       bool `json:"ordered_replay"`
	LiveSubscriptions   bool `json:"live_subscriptions"`
	ConsumerCheckpoints bool `json:"consumer_checkpoints"`
	BrokerBacked        bool `json:"broker_backed"`
}

type PartitionKeyKind string

const (
	PartitionKeyTraceID   PartitionKeyKind = "trace_id"
	PartitionKeyTaskID    PartitionKeyKind = "task_id"
	PartitionKeyEventType PartitionKeyKind = "event_type"
)

type PartitionRoute struct {
	Topic         string           `json:"topic"`
	PartitionKey  PartitionKeyKind `json:"partition_key"`
	OrderingScope string           `json:"ordering_scope,omitempty"`
	Description   string           `json:"description,omitempty"`
}

type ReplayRequest struct {
	After   Position `json:"after,omitempty"`
	Limit   int      `json:"limit,omitempty"`
	TaskID  string   `json:"task_id,omitempty"`
	TraceID string   `json:"trace_id,omitempty"`
}

type OwnershipMode string

const (
	OwnershipModeExclusive OwnershipMode = "exclusive"
	OwnershipModeShared    OwnershipMode = "shared"
)

type SubscriberOwnershipContract struct {
	SubscriberGroup string            `json:"subscriber_group"`
	Consumer        string            `json:"consumer"`
	Mode            OwnershipMode     `json:"mode"`
	Epoch           int64             `json:"epoch,omitempty"`
	LeaseToken      string            `json:"lease_token,omitempty"`
	PartitionHints  []string          `json:"partition_hints,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

type SubscriptionRequest struct {
	Replay            ReplayRequest                `json:"replay"`
	Buffer            int                          `json:"buffer,omitempty"`
	PartitionRoute    *PartitionRoute              `json:"partition_route,omitempty"`
	OwnershipContract *SubscriberOwnershipContract `json:"ownership_contract,omitempty"`
}

type Checkpoint struct {
	Consumer  string            `json:"consumer"`
	Position  Position          `json:"position"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type DurableEventLog interface {
	Backend() EventLogBackend
	Capabilities() Capabilities
	Publish(context.Context, domain.Event) (Record, error)
	Replay(context.Context, ReplayRequest) ([]Record, error)
	Subscribe(context.Context, SubscriptionRequest) (<-chan Record, func(), error)
}

type DurableCheckpointStore interface {
	GetCheckpoint(context.Context, string) (Checkpoint, bool, error)
	SaveCheckpoint(context.Context, Checkpoint) error
}

type BrokerRuntimeConfig struct {
	Driver             string
	URLs               []string
	Topic              string
	ConsumerGroup      string
	PublishTimeout     time.Duration
	ReplayLimit        int
	CheckpointInterval time.Duration
}

func (cfg BrokerRuntimeConfig) Validate() error {
	missing := make([]string, 0, 3)
	if strings.TrimSpace(cfg.Driver) == "" {
		missing = append(missing, "driver")
	}
	if len(cfg.URLs) == 0 {
		missing = append(missing, "urls")
	}
	if strings.TrimSpace(cfg.Topic) == "" {
		missing = append(missing, "topic")
	}
	if len(missing) > 0 {
		return fmt.Errorf("broker event log config missing %s", strings.Join(missing, ", "))
	}
	if cfg.PublishTimeout <= 0 {
		return fmt.Errorf("broker event log publish timeout must be positive")
	}
	if cfg.ReplayLimit <= 0 {
		return fmt.Errorf("broker event log replay limit must be positive")
	}
	if cfg.CheckpointInterval <= 0 {
		return fmt.Errorf("broker event log checkpoint interval must be positive")
	}
	return nil
}
