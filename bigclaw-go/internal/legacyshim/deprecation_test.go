package legacyshim

import (
	"strings"
	"testing"
)

func TestWarnLegacyRuntimeSurfaceMessage(t *testing.T) {
	message := WarnLegacyRuntimeSurface("python -m bigclaw", "bash scripts/ops/bigclawctl")

	if !strings.Contains(message, "frozen for migration-only use") {
		t.Fatalf("expected migration-only guidance, got %q", message)
	}
	if !strings.Contains(message, "bash scripts/ops/bigclawctl") {
		t.Fatalf("expected replacement guidance in message, got %q", message)
	}
}

func TestLegacyRuntimeModulesExposeGoMainlineReplacements(t *testing.T) {
	if RuntimeGoMainlineReplacement != "bigclaw-go/internal/worker/runtime.go" {
		t.Fatalf("unexpected runtime replacement: %q", RuntimeGoMainlineReplacement)
	}
	if SchedulerGoMainlineReplacement != "bigclaw-go/internal/scheduler/scheduler.go" {
		t.Fatalf("unexpected scheduler replacement: %q", SchedulerGoMainlineReplacement)
	}
	if WorkflowGoMainlineReplacement != "bigclaw-go/internal/workflow/engine.go" {
		t.Fatalf("unexpected workflow replacement: %q", WorkflowGoMainlineReplacement)
	}
	if OrchestrationGoMainlineReplacement != "bigclaw-go/internal/workflow/orchestration.go" {
		t.Fatalf("unexpected orchestration replacement: %q", OrchestrationGoMainlineReplacement)
	}
	if QueueGoMainlineReplacement != "bigclaw-go/internal/queue/queue.go" {
		t.Fatalf("unexpected queue replacement: %q", QueueGoMainlineReplacement)
	}
	if !strings.Contains(LegacyRuntimeGuidance, "sole implementation mainline") {
		t.Fatalf("expected legacy runtime guidance to mention go mainline, got %q", LegacyRuntimeGuidance)
	}
}

func TestLegacyServiceSurfaceMessage(t *testing.T) {
	message := WarnLegacyServiceSurface()
	if !strings.Contains(message, "go run ./bigclaw-go/cmd/bigclawd") {
		t.Fatalf("expected bigclawd replacement in service warning, got %q", message)
	}
}

func TestServiceModuleExposesGoMainlineReplacement(t *testing.T) {
	if ServiceGoMainlineReplacement != "bigclaw-go/cmd/bigclawd/main.go" {
		t.Fatalf("unexpected service replacement: %q", ServiceGoMainlineReplacement)
	}
	if !strings.Contains(ServiceLegacyMainlineStatus, "sole implementation mainline") {
		t.Fatalf("expected service legacy status to mention go mainline, got %q", ServiceLegacyMainlineStatus)
	}
}
