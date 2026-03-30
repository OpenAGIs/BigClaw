package runtimecompat

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestWorkerLifecycleIsStableWithMultipleTools(t *testing.T) {
	task := domain.Task{
		ID:            "BIG-301-matrix",
		Source:        "github",
		Title:         "worker lifecycle matrix",
		Description:   "validate stable lifecycle",
		RequiredTools: []string{"github", "browser"},
	}
	run := NewRun(task, "run-big301-matrix", "docker")
	runtime := ClawWorkerRuntime{
		ToolRuntime: ToolRuntime{
			Handlers: map[string]Handler{
				"github": func(action string, payload map[string]any) string { return action + ":" + payload["repo"].(string) },
				"browser": func(action string, payload map[string]any) string {
					return action + ":" + payload["url"].(string)
				},
			},
		},
	}

	result := runtime.Execute(task, Decision{Medium: "docker", Approved: true, Reason: "ok"}, run, map[string]map[string]any{
		"github":  {"repo": "OpenAGIs/BigClaw"},
		"browser": {"url": "https://example.com"},
	})

	if len(result.ToolResults) != 2 {
		t.Fatalf("expected two tool results, got %+v", result.ToolResults)
	}
	for _, item := range result.ToolResults {
		if !item.Success {
			t.Fatalf("expected successful tool result, got %+v", item)
		}
	}
	last := run.Audits[len(run.Audits)-1]
	if last.Action != "worker.lifecycle" || last.Outcome != "completed" {
		t.Fatalf("unexpected final audit: %+v", last)
	}
}

func TestRiskRoutesToExpectedSandboxMediums(t *testing.T) {
	low := domain.Task{ID: "low", Source: "local", Title: "low", RiskLevel: domain.RiskLow}
	high := domain.Task{ID: "high", Source: "local", Title: "high", RiskLevel: domain.RiskHigh}
	browser := domain.Task{ID: "browser", Source: "local", Title: "browser", RequiredTools: []string{"browser"}, RiskLevel: domain.RiskMedium}

	if got := RouteMedium(low); got != "docker" {
		t.Fatalf("expected docker for low risk task, got %q", got)
	}
	if got := RouteMedium(high); got != "vm" {
		t.Fatalf("expected vm for high risk task, got %q", got)
	}
	if got := RouteMedium(browser); got != "browser" && got != "docker" {
		t.Fatalf("expected browser or docker for browser task, got %q", got)
	}
}

func TestToolRuntimePolicyAndAuditChain(t *testing.T) {
	task := domain.Task{ID: "BIG-303-matrix", Source: "local", Title: "tool policy", RequiredTools: []string{"github", "browser"}}
	run := NewRun(task, "run-big303-matrix", "docker")

	runtime := ToolRuntime{
		Policy: ToolPolicy{AllowedTools: []string{"github"}, BlockedTools: []string{"browser"}},
		Handlers: map[string]Handler{
			"github": func(action string, payload map[string]any) string { return "ok" },
		},
	}

	allow := runtime.Invoke("github", "execute", map[string]any{"repo": "OpenAGIs/BigClaw"}, run)
	block := runtime.Invoke("browser", "execute", map[string]any{"url": "https://example.com"}, run)

	if !allow.Success {
		t.Fatalf("expected allowed tool invocation to succeed, got %+v", allow)
	}
	if block.Success {
		t.Fatalf("expected blocked tool invocation to fail, got %+v", block)
	}

	outcomes := make([]string, 0, len(run.Audits))
	for _, audit := range run.Audits {
		if audit.Action == "tool.invoke" {
			outcomes = append(outcomes, audit.Outcome)
		}
	}
	if !containsOutcome(outcomes, "success") || !containsOutcome(outcomes, "blocked") {
		t.Fatalf("expected success and blocked outcomes, got %+v", outcomes)
	}
}

func containsOutcome(outcomes []string, want string) bool {
	for _, outcome := range outcomes {
		if outcome == want {
			return true
		}
	}
	return false
}
