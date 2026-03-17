package main

import (
	"testing"

	"bigclaw-go/internal/config"
	"bigclaw-go/internal/events"
)

func TestBuildEventLogUsesBrokerStubDriver(t *testing.T) {
	defaults := config.Default()
	eventLog, err := buildEventLog(config.Config{
		EventLogBackend:            string(events.EventLogBackendBroker),
		EventLogBrokerDriver:       events.BrokerDriverStub,
		EventLogBrokerURLs:         []string{"stub://broker-a"},
		EventLogBrokerTopic:        "bigclaw.events",
		EventLogPublishTimeout:     defaults.EventLogPublishTimeout,
		EventLogReplayLimit:        defaults.EventLogReplayLimit,
		EventLogCheckpointInterval: defaults.EventLogCheckpointInterval,
	})
	if err != nil {
		t.Fatalf("build event log: %v", err)
	}
	if eventLog == nil || eventLog.Backend() != "broker_stub" {
		t.Fatalf("expected broker stub backend, got %#v", eventLog)
	}
	if capability := eventLog.Capabilities(); capability.Backend != "broker_stub" || capability.Retention.Mode != "process_memory_stub" {
		t.Fatalf("unexpected broker stub capability payload: %+v", capability)
	}
}

func TestBuildEventLogRejectsUnimplementedBrokerDriver(t *testing.T) {
	cfg := config.Default()
	cfg.EventLogBackend = string(events.EventLogBackendBroker)
	cfg.EventLogBrokerDriver = "kafka"
	cfg.EventLogBrokerURLs = []string{"kafka-1:9092"}
	cfg.EventLogBrokerTopic = "bigclaw.events"

	eventLog, err := buildEventLog(cfg)
	if err == nil {
		t.Fatalf("expected unimplemented broker driver error, got event log %#v", eventLog)
	}
}
