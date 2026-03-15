package main

import (
	"path/filepath"
	"testing"
	"time"

	"bigclaw-go/internal/config"
)

func TestBuildConfiguredEventLogUsesSQLiteDurableBackendContract(t *testing.T) {
	cfg := config.Default()
	cfg.EventBackend = "sqlite"
	cfg.EventLogDSN = filepath.Join(t.TempDir(), "event-log.db")
	cfg.EventCheckpointDSN = cfg.EventLogDSN
	cfg.EventRetention = time.Hour

	eventLog, err := buildConfiguredEventLog(cfg)
	if err != nil {
		t.Fatalf("build sqlite event log: %v", err)
	}
	if eventLog == nil {
		t.Fatal("expected sqlite event log")
	}
	defer closeEventLog(eventLog)
	if eventLog.Backend() != "sqlite" {
		t.Fatalf("expected sqlite backend, got %q", eventLog.Backend())
	}
}

func TestBuildConfiguredEventLogUsesHTTPDurableBackendContract(t *testing.T) {
	cfg := config.Default()
	cfg.EventBackend = "http"
	cfg.EventLogDSN = "http://127.0.0.1:8080/internal/events/log"
	cfg.EventCheckpointDSN = cfg.EventLogDSN
	cfg.EventRetention = time.Hour

	eventLog, err := buildConfiguredEventLog(cfg)
	if err != nil {
		t.Fatalf("build http event log: %v", err)
	}
	if eventLog == nil {
		t.Fatal("expected http event log")
	}
	if eventLog.Backend() != "http" {
		t.Fatalf("expected http backend, got %q", eventLog.Backend())
	}
}
