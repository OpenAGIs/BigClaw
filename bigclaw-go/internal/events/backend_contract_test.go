package events

import (
	"strings"
	"testing"
	"time"
)

func TestValidateBackendConfigMemoryDefaults(t *testing.T) {
	report := ValidateBackendConfig(BackendConfig{
		Backend:          BackendMemory,
		RequireReplay:    true,
		RequireFiltering: true,
	})
	if report.HasErrors() {
		t.Fatalf("expected memory backend validation to pass, got %+v", report.Issues)
	}
}

func TestValidateBackendConfigRejectsCheckpointOnMemory(t *testing.T) {
	report := ValidateBackendConfig(BackendConfig{
		Backend:           BackendMemory,
		RequireReplay:     true,
		RequireFiltering:  true,
		RequireCheckpoint: true,
	})
	if !report.HasErrors() {
		t.Fatal("expected checkpoint requirement on memory backend to fail")
	}
	if err := report.Error(); err == nil || !strings.Contains(err.Error(), "does not support checkpoints") {
		t.Fatalf("expected checkpoint validation error, got %v", err)
	}
}

func TestValidateBackendConfigAcceptsSQLiteDurableBackend(t *testing.T) {
	report := ValidateBackendConfig(BackendConfig{
		Backend:           BackendSQLite,
		LogDSN:            "file:events.db",
		CheckpointDSN:     "file:checkpoints.db",
		Retention:         time.Hour,
		RequireReplay:     true,
		RequireFiltering:  true,
		RequireCheckpoint: true,
	})
	if report.HasErrors() {
		t.Fatalf("expected sqlite backend validation to pass, got %+v", report.Issues)
	}
}

func TestValidateBackendConfigAcceptsHTTPDurableBackend(t *testing.T) {
	report := ValidateBackendConfig(BackendConfig{
		Backend:           BackendHTTP,
		LogDSN:            "http://127.0.0.1:8080/internal/events/log",
		CheckpointDSN:     "http://127.0.0.1:8080/internal/events/log",
		Retention:         time.Hour,
		RequireReplay:     true,
		RequireFiltering:  true,
		RequireCheckpoint: true,
	})
	if report.HasErrors() {
		t.Fatalf("expected http backend validation to pass, got %+v", report.Issues)
	}
}

func TestValidateBackendConfigRequiresDurableFields(t *testing.T) {
	report := ValidateBackendConfig(BackendConfig{
		Backend:           BackendBroker,
		RequireReplay:     true,
		RequireFiltering:  true,
		RequireCheckpoint: true,
	})
	if !report.HasErrors() {
		t.Fatal("expected broker backend to require durable config fields")
	}
	errText := report.Error().Error()
	for _, fragment := range []string{"BIGCLAW_EVENT_LOG_DSN", "BIGCLAW_EVENT_CHECKPOINT_DSN", "BIGCLAW_EVENT_RETENTION"} {
		if !strings.Contains(errText, fragment) {
			t.Fatalf("expected %q in validation error, got %s", fragment, errText)
		}
	}
}
