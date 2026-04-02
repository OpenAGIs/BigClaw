package worker

import (
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

func TestClawWorkerRuntimeExecutesMultipleToolsAndRecordsCompletedLifecycle(t *testing.T) {
	task := domain.Task{
		ID:            "BIG-301-matrix",
		Source:        "github",
		Title:         "worker lifecycle matrix",
		RequiredTools: []string{"github", "browser"},
	}
	run := NewToolRun(task, "run-big301-matrix", "docker")
	runtime := ToolRuntime{
		Handlers: map[string]func(action string, payload map[string]any) string{
			"github":  func(action string, payload map[string]any) string { return action + ":" + payload["repo"].(string) },
			"browser": func(action string, payload map[string]any) string { return action + ":" + payload["url"].(string) },
		},
	}
	worker := ClawWorkerRuntime{ToolRuntime: runtime}

	result := worker.Execute(
		task,
		ToolDecision{Medium: "docker", Approved: true, Reason: "ok"},
		run,
		map[string]map[string]any{
			"github":  {"repo": "OpenAGIs/BigClaw"},
			"browser": {"url": "https://example.com"},
		},
	)

	if len(result.ToolResults) != 2 {
		t.Fatalf("expected 2 tool results, got %+v", result.ToolResults)
	}
	for _, item := range result.ToolResults {
		if !item.Success {
			t.Fatalf("expected all tool results to succeed, got %+v", result.ToolResults)
		}
	}
	if got := run.Audits[len(run.Audits)-1]; got.Action != "worker.lifecycle" || got.Outcome != "completed" {
		t.Fatalf("unexpected lifecycle audit: %+v", got)
	}
}

func TestToolRuntimePolicyAndAuditChain(t *testing.T) {
	task := domain.Task{
		ID:            "BIG-303-matrix",
		Source:        "local",
		Title:         "tool policy",
		RequiredTools: []string{"github", "browser"},
	}
	run := NewToolRun(task, "run-big303-matrix", "docker")
	runtime := ToolRuntime{
		Policy:   ToolPolicy{AllowedTools: []string{"github"}, BlockedTools: []string{"browser"}},
		Handlers: map[string]func(action string, payload map[string]any) string{"github": func(action string, payload map[string]any) string { return "ok" }},
	}

	allow := runtime.Invoke("github", "execute", map[string]any{"repo": "OpenAGIs/BigClaw"}, run)
	block := runtime.Invoke("browser", "execute", map[string]any{"url": "https://example.com"}, run)

	if !allow.Success || block.Success {
		t.Fatalf("unexpected tool invocation results: allow=%+v block=%+v", allow, block)
	}
	outcomes := make([]string, 0, len(run.Audits))
	for _, audit := range run.Audits {
		if audit.Action == "tool.invoke" {
			outcomes = append(outcomes, audit.Outcome)
		}
	}
	if len(outcomes) != 2 || outcomes[0] != "success" || outcomes[1] != "blocked" {
		t.Fatalf("unexpected tool.invoke outcomes: %+v", outcomes)
	}
}

func TestSchedulerRiskRoutesToExpectedGoExecutors(t *testing.T) {
	s := scheduler.New()

	low := domain.Task{ID: "low", Source: "local", Title: "low", RiskLevel: domain.RiskLow}
	high := domain.Task{ID: "high", Source: "local", Title: "high", RiskLevel: domain.RiskHigh}
	browser := domain.Task{ID: "browser", Source: "local", Title: "browser", RequiredTools: []string{"browser"}, RiskLevel: domain.RiskMedium}

	if got := s.Decide(low, scheduler.QuotaSnapshot{}).Assignment.Executor; got != domain.ExecutorLocal {
		t.Fatalf("expected low risk task to default to local executor, got %s", got)
	}
	if got := s.Decide(high, scheduler.QuotaSnapshot{}).Assignment.Executor; got != domain.ExecutorKubernetes {
		t.Fatalf("expected high risk task to route to kubernetes, got %s", got)
	}
	if got := s.Decide(browser, scheduler.QuotaSnapshot{}).Assignment.Executor; got != domain.ExecutorKubernetes {
		t.Fatalf("expected browser task to route to kubernetes, got %s", got)
	}
}
