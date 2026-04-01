package runtimematrix

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
	run := NewTaskRun(task, "run-big301-matrix", "docker")
	runtime := ToolRuntime{
		Handlers: map[string]ToolHandler{
			"github":  func(action string, payload map[string]any) string { return action + ":" + payload["repo"].(string) },
			"browser": func(action string, payload map[string]any) string { return action + ":" + payload["url"].(string) },
		},
	}
	worker := ClawWorkerRuntime{ToolRuntime: &runtime}

	result := worker.Execute(task, Decision{Medium: "docker", Approved: true, Reason: "ok"}, run, map[string]map[string]any{
		"github":  {"repo": "OpenAGIs/BigClaw"},
		"browser": {"url": "https://example.com"},
	})

	if len(result.ToolResults) != 2 {
		t.Fatalf("expected two tool results, got %d", len(result.ToolResults))
	}
	for _, item := range result.ToolResults {
		if !item.Success {
			t.Fatalf("expected successful tool result, got %+v", item)
		}
	}
	if got := run.Audits[len(run.Audits)-1]; got.Action != "worker.lifecycle" || got.Outcome != "completed" {
		t.Fatalf("expected final lifecycle audit to be completed, got %+v", got)
	}
}

func TestRiskRoutesToExpectedSandboxMediums(t *testing.T) {
	scheduler := Scheduler{}

	low := domain.Task{ID: "low", Source: "local", Title: "low", RiskLevel: domain.RiskLow}
	high := domain.Task{ID: "high", Source: "local", Title: "high", RiskLevel: domain.RiskHigh}
	browser := domain.Task{ID: "browser", Source: "local", Title: "browser", RequiredTools: []string{"browser"}, RiskLevel: domain.RiskMedium}

	if got := scheduler.Decide(low).Medium; got != "docker" {
		t.Fatalf("expected low-risk task to route to docker, got %s", got)
	}
	if got := scheduler.Decide(high).Medium; got != "vm" {
		t.Fatalf("expected high-risk task to route to vm, got %s", got)
	}
	if got := scheduler.Decide(browser).Medium; got != "browser" {
		t.Fatalf("expected browser task to route to browser, got %s", got)
	}
}

func TestToolRuntimePolicyAndAuditChain(t *testing.T) {
	task := domain.Task{ID: "BIG-303-matrix", Source: "local", Title: "tool policy", RequiredTools: []string{"github", "browser"}}
	run := NewTaskRun(task, "run-big303-matrix", "docker")

	runtime := ToolRuntime{
		Policy: ToolPolicy{AllowedTools: []string{"github"}, BlockedTools: []string{"browser"}},
		Handlers: map[string]ToolHandler{
			"github": func(action string, payload map[string]any) string { return "ok" },
		},
	}

	allow := runtime.Invoke("github", "execute", map[string]any{"repo": "OpenAGIs/BigClaw"}, run)
	block := runtime.Invoke("browser", "execute", map[string]any{"url": "https://example.com"}, run)

	if !allow.Success {
		t.Fatalf("expected github invocation to succeed, got %+v", allow)
	}
	if block.Success {
		t.Fatalf("expected browser invocation to be blocked, got %+v", block)
	}

	outcomes := make([]string, 0)
	for _, audit := range run.Audits {
		if audit.Action == "tool.invoke" {
			outcomes = append(outcomes, audit.Outcome)
		}
	}
	if !contains(outcomes, "success") {
		t.Fatalf("expected tool audit success outcome, got %v", outcomes)
	}
	if !contains(outcomes, "blocked") {
		t.Fatalf("expected tool audit blocked outcome, got %v", outcomes)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
