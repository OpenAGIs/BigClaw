package runtimecompat

import (
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/scheduler"
)

func TestRuntimeExecutesMultipleToolsAndCompletesLifecycle(t *testing.T) {
	task := domain.Task{
		ID:            "BIG-301-matrix",
		Source:        "github",
		Title:         "worker lifecycle matrix",
		RequiredTools: []string{"github", "browser"},
	}
	run := NewRunRecord("run-big301-matrix", "docker")
	runtime := Runtime{
		ToolRuntime: ToolRuntime{
			Handlers: map[string]Handler{
				"github":  func(action string, payload map[string]any) string { return action + ":" + payload["repo"].(string) },
				"browser": func(action string, payload map[string]any) string { return action + ":" + payload["url"].(string) },
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
			t.Fatalf("expected successful tool results, got %+v", result.ToolResults)
		}
	}
	if run.Audits[len(run.Audits)-1].Action != "worker.lifecycle" || run.Audits[len(run.Audits)-1].Outcome != "completed" {
		t.Fatalf("expected completed worker lifecycle audit, got %+v", run.Audits)
	}
}

func TestToolRuntimePolicyAndAuditChain(t *testing.T) {
	task := domain.Task{ID: "BIG-303-matrix", RequiredTools: []string{"github", "browser"}}
	run := NewRunRecord("run-big303-matrix", "docker")
	runtime := ToolRuntime{
		Policy: ToolPolicy{AllowedTools: []string{"github"}, BlockedTools: []string{"browser"}},
		Handlers: map[string]Handler{
			"github": func(action string, payload map[string]any) string { return "ok" },
		},
	}

	allow := runtime.Invoke("github", "execute", map[string]any{"repo": "OpenAGIs/BigClaw"}, run)
	block := runtime.Invoke("browser", "execute", map[string]any{"url": "https://example.com"}, run)

	if !allow.Success || block.Success {
		t.Fatalf("expected allow success and block failure, got allow=%+v block=%+v", allow, block)
	}
	outcomes := make([]string, 0, len(run.Audits))
	for _, audit := range run.Audits {
		if audit.Action == "tool.invoke" {
			outcomes = append(outcomes, audit.Outcome)
		}
	}
	if len(outcomes) != 2 || outcomes[0] != "success" || outcomes[1] != "blocked" {
		t.Fatalf("expected success/blocked audit chain, got %+v", outcomes)
	}

	low := scheduler.DecideLegacyMedium(domain.Task{ID: "low", Source: "local", Title: "low"})
	high := scheduler.DecideLegacyMedium(domain.Task{ID: "high", Source: "local", Title: "high", RiskLevel: domain.RiskHigh})
	browser := scheduler.DecideLegacyMedium(domain.Task{ID: "browser", Source: "local", Title: "browser", RequiredTools: []string{"browser"}, RiskLevel: domain.RiskMedium})
	if low.Medium != "docker" || high.Medium != "vm" || (browser.Medium != "browser" && browser.Medium != "docker") {
		t.Fatalf("unexpected legacy medium routing: low=%+v high=%+v browser=%+v", low, high, browser)
	}

	_ = task
}
