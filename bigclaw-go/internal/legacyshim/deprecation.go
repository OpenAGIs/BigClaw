package legacyshim

import "fmt"

const (
	LegacyRuntimeGuidance = "bigclaw-go is the sole implementation mainline for active development; the legacy Python runtime surface remains migration-only."

	RuntimeGoMainlineReplacement       = "bigclaw-go/internal/worker/runtime.go"
	SchedulerGoMainlineReplacement     = "bigclaw-go/internal/scheduler/scheduler.go"
	WorkflowGoMainlineReplacement      = "bigclaw-go/internal/workflow/engine.go"
	OrchestrationGoMainlineReplacement = "bigclaw-go/internal/workflow/orchestration.go"
	QueueGoMainlineReplacement         = "bigclaw-go/internal/queue/queue.go"

	ServiceGoMainlineReplacement = "bigclaw-go/cmd/bigclawd/main.go"
	ServiceLegacyMainlineStatus  = "bigclaw-go is the sole implementation mainline for active development; service.py remains migration-only compatibility scaffolding."
)

func LegacyRuntimeMessage(surface, replacement string) string {
	return fmt.Sprintf("%s is frozen for migration-only use. %s Use %s instead.", surface, LegacyRuntimeGuidance, replacement)
}

func WarnLegacyRuntimeSurface(surface, replacement string) string {
	return LegacyRuntimeMessage(surface, replacement)
}

func WarnLegacyServiceSurface() string {
	return WarnLegacyRuntimeSurface("python -m bigclaw serve", "go run ./bigclaw-go/cmd/bigclawd")
}
