package events

import (
	"fmt"
	"strings"
	"time"
)

type BackendKind string

const (
	BackendMemory BackendKind = "memory"
	BackendSQLite BackendKind = "sqlite"
	BackendHTTP   BackendKind = "http"
	BackendBroker BackendKind = "broker"
)

type SupportLevel string

const (
	SupportUnsupported SupportLevel = "unsupported"
	SupportDerived     SupportLevel = "derived"
	SupportNative      SupportLevel = "native"
)

type CapabilityMatrix struct {
	Publish     SupportLevel
	Replay      SupportLevel
	Checkpoints SupportLevel
	Filtering   SupportLevel
}

type BackendProfile struct {
	Backend               BackendKind
	Implemented           bool
	Durable               bool
	RequiresLogDSN        bool
	RequiresCheckpointDSN bool
	Capabilities          CapabilityMatrix
}

type BackendConfig struct {
	Backend           BackendKind
	LogDSN            string
	CheckpointDSN     string
	Retention         time.Duration
	RequireReplay     bool
	RequireCheckpoint bool
	RequireFiltering  bool
}

type ValidationIssue struct {
	Level   string
	Field   string
	Message string
}

type ValidationReport struct {
	Profile BackendProfile
	Issues  []ValidationIssue
}

func Catalog() map[BackendKind]BackendProfile {
	return map[BackendKind]BackendProfile{
		BackendMemory: {
			Backend:     BackendMemory,
			Implemented: true,
			Durable:     false,
			Capabilities: CapabilityMatrix{
				Publish:     SupportNative,
				Replay:      SupportNative,
				Checkpoints: SupportUnsupported,
				Filtering:   SupportNative,
			},
		},
		BackendSQLite: {
			Backend:               BackendSQLite,
			Implemented:           false,
			Durable:               true,
			RequiresLogDSN:        true,
			RequiresCheckpointDSN: true,
			Capabilities: CapabilityMatrix{
				Publish:     SupportNative,
				Replay:      SupportNative,
				Checkpoints: SupportNative,
				Filtering:   SupportDerived,
			},
		},
		BackendHTTP: {
			Backend:               BackendHTTP,
			Implemented:           false,
			Durable:               true,
			RequiresLogDSN:        true,
			RequiresCheckpointDSN: true,
			Capabilities: CapabilityMatrix{
				Publish:     SupportNative,
				Replay:      SupportNative,
				Checkpoints: SupportNative,
				Filtering:   SupportDerived,
			},
		},
		BackendBroker: {
			Backend:               BackendBroker,
			Implemented:           false,
			Durable:               true,
			RequiresLogDSN:        true,
			RequiresCheckpointDSN: true,
			Capabilities: CapabilityMatrix{
				Publish:     SupportNative,
				Replay:      SupportNative,
				Checkpoints: SupportNative,
				Filtering:   SupportDerived,
			},
		},
	}
}

func ValidateBackendConfig(cfg BackendConfig) ValidationReport {
	profile, ok := Catalog()[cfg.Backend]
	report := ValidationReport{}
	if !ok {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "backend",
			Message: fmt.Sprintf("unsupported event backend %q", cfg.Backend),
		})
		return report
	}
	report.Profile = profile

	if !profile.Implemented {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "backend",
			Message: fmt.Sprintf("event backend %q is defined in the durability matrix but not wired into the bootstrap runtime", cfg.Backend),
		})
	}
	if profile.RequiresLogDSN && strings.TrimSpace(cfg.LogDSN) == "" {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "log_dsn",
			Message: "durable event backend requires BIGCLAW_EVENT_LOG_DSN",
		})
	}
	if !profile.RequiresLogDSN && strings.TrimSpace(cfg.LogDSN) != "" {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "log_dsn",
			Message: fmt.Sprintf("event backend %q does not accept BIGCLAW_EVENT_LOG_DSN", cfg.Backend),
		})
	}
	if profile.Durable && cfg.Retention <= 0 {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "retention",
			Message: "durable event backend requires BIGCLAW_EVENT_RETENTION to be greater than zero",
		})
	}
	if !profile.Durable && cfg.Retention < 0 {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "retention",
			Message: "event retention cannot be negative",
		})
	}
	if cfg.RequireReplay && report.Profile.Capabilities.Replay == SupportUnsupported {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "require_replay",
			Message: fmt.Sprintf("event backend %q does not support replay", cfg.Backend),
		})
	}
	if cfg.RequireCheckpoint {
		if report.Profile.Capabilities.Checkpoints == SupportUnsupported {
			report.Issues = append(report.Issues, ValidationIssue{
				Level:   "error",
				Field:   "require_checkpoint",
				Message: fmt.Sprintf("event backend %q does not support checkpoints", cfg.Backend),
			})
		}
		if profile.RequiresCheckpointDSN && strings.TrimSpace(cfg.CheckpointDSN) == "" {
			report.Issues = append(report.Issues, ValidationIssue{
				Level:   "error",
				Field:   "checkpoint_dsn",
				Message: "checkpoint-capable backends require BIGCLAW_EVENT_CHECKPOINT_DSN when checkpoint support is required",
			})
		}
	}
	if cfg.RequireFiltering && report.Profile.Capabilities.Filtering == SupportUnsupported {
		report.Issues = append(report.Issues, ValidationIssue{
			Level:   "error",
			Field:   "require_filtering",
			Message: fmt.Sprintf("event backend %q does not support server-side filtering", cfg.Backend),
		})
	}
	return report
}

func (r ValidationReport) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Level == "error" {
			return true
		}
	}
	return false
}

func (r ValidationReport) Error() error {
	if !r.HasErrors() {
		return nil
	}
	parts := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		if issue.Level != "error" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
	}
	return fmt.Errorf("event backend validation failed: %s", strings.Join(parts, "; "))
}
