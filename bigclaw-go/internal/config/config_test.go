package config

import (
	"testing"
	"time"
)

func TestLoadFromEnvIncludesEventLogBrokerSettings(t *testing.T) {
	t.Setenv("BIGCLAW_EVENT_LOG_BACKEND", "broker")
	t.Setenv("BIGCLAW_EVENT_LOG_BROKER_DRIVER", "kafka")
	t.Setenv("BIGCLAW_EVENT_LOG_BROKER_URLS", "kafka-1:9092,kafka-2:9092")
	t.Setenv("BIGCLAW_EVENT_LOG_BROKER_TOPIC", "bigclaw.events")
	t.Setenv("BIGCLAW_EVENT_LOG_CONSUMER_GROUP", "bigclaw-consumers")
	t.Setenv("BIGCLAW_EVENT_LOG_PUBLISH_TIMEOUT", "7s")
	t.Setenv("BIGCLAW_EVENT_LOG_REPLAY_LIMIT", "2048")
	t.Setenv("BIGCLAW_EVENT_LOG_CHECKPOINT_INTERVAL", "15s")

	cfg := LoadFromEnv()
	if cfg.EventLogBackend != "broker" {
		t.Fatalf("expected broker backend, got %s", cfg.EventLogBackend)
	}
	if cfg.EventLogBrokerDriver != "kafka" {
		t.Fatalf("expected kafka driver, got %s", cfg.EventLogBrokerDriver)
	}
	if len(cfg.EventLogBrokerURLs) != 2 || cfg.EventLogBrokerURLs[1] != "kafka-2:9092" {
		t.Fatalf("unexpected broker urls: %#v", cfg.EventLogBrokerURLs)
	}
	if cfg.EventLogBrokerTopic != "bigclaw.events" {
		t.Fatalf("expected topic bigclaw.events, got %s", cfg.EventLogBrokerTopic)
	}
	if cfg.EventLogConsumerGroup != "bigclaw-consumers" {
		t.Fatalf("expected consumer group, got %s", cfg.EventLogConsumerGroup)
	}
	if cfg.EventLogReplayLimit != 2048 {
		t.Fatalf("expected replay limit 2048, got %d", cfg.EventLogReplayLimit)
	}
	if cfg.EventLogPublishTimeout.String() != "7s" {
		t.Fatalf("expected publish timeout 7s, got %s", cfg.EventLogPublishTimeout)
	}
	if cfg.EventLogCheckpointInterval.String() != "15s" {
		t.Fatalf("expected checkpoint interval 15s, got %s", cfg.EventLogCheckpointInterval)
	}
}

func TestLoadFromEnvIncludesSubscriberLeaseSQLitePath(t *testing.T) {
	t.Setenv("BIGCLAW_SUBSCRIBER_LEASE_SQLITE_PATH", "/tmp/shared-subscriber-leases.db")
	t.Setenv("BIGCLAW_COORDINATOR_LEASE_SQLITE_PATH", "/tmp/shared-coordinator-leases.db")
	t.Setenv("BIGCLAW_COORDINATOR_SCOPE", "scheduler")
	t.Setenv("BIGCLAW_COORDINATOR_ID", "node-a")
	t.Setenv("BIGCLAW_COORDINATOR_LEASE_TTL", "9s")

	cfg := LoadFromEnv()
	if cfg.SubscriberLeaseSQLitePath != "/tmp/shared-subscriber-leases.db" {
		t.Fatalf("expected subscriber lease sqlite path, got %q", cfg.SubscriberLeaseSQLitePath)
	}
	if cfg.CoordinatorLeaseSQLitePath != "/tmp/shared-coordinator-leases.db" {
		t.Fatalf("expected coordinator lease sqlite path, got %q", cfg.CoordinatorLeaseSQLitePath)
	}
	if cfg.CoordinatorScope != "scheduler" || cfg.CoordinatorID != "node-a" || cfg.CoordinatorLeaseTTL != 9*time.Second {
		t.Fatalf("unexpected coordinator lease config: %+v", cfg)
	}
}
