package runtimeparity

import "testing"

func TestWorkerLifecycleIsStableWithMultipleTools(t *testing.T) {
	t.Parallel()

	task := Task{
		ID:            "BIG-301-matrix",
		Source:        "github",
		Title:         "worker lifecycle matrix",
		Description:   "validate stable lifecycle",
		RequiredTools: []string{"github", "browser"},
	}
	run := &Run{}
	runtime := WorkerRuntime{
		ToolRuntime: ToolRuntime{
			Handlers: map[string]Handler{
				"github": func(action string, payload map[string]string) string {
					return action + ":" + payload["repo"]
				},
				"browser": func(action string, payload map[string]string) string {
					return action + ":" + payload["url"]
				},
			},
		},
	}

	result := runtime.Execute(task, Decision{Medium: "docker", Approved: true, Reason: "ok"}, run, map[string]map[string]string{
		"github":  {"repo": "OpenAGIs/BigClaw"},
		"browser": {"url": "https://example.com"},
	}, "worker-runtime")

	if len(result.ToolResults) != 2 {
		t.Fatalf("tool results = %d, want 2", len(result.ToolResults))
	}
	for _, item := range result.ToolResults {
		if !item.Success {
			t.Fatalf("expected successful tool result, got %+v", item)
		}
	}
	if result.SandboxProfile.Medium != "docker" {
		t.Fatalf("sandbox profile = %+v", result.SandboxProfile)
	}
	last := run.Audits[len(run.Audits)-1]
	if last.Action != "worker.lifecycle" || last.Outcome != "completed" {
		t.Fatalf("last audit = %+v", last)
	}
}

func TestRiskRoutesToExpectedSandboxMediums(t *testing.T) {
	t.Parallel()

	low := Task{ID: "low", Source: "local", Title: "low", RiskLevel: RiskLow}
	high := Task{ID: "high", Source: "local", Title: "high", RiskLevel: RiskHigh}
	browser := Task{
		ID:            "browser",
		Source:        "local",
		Title:         "browser",
		RequiredTools: []string{"browser"},
		RiskLevel:     RiskMedium,
	}

	if decision := Decide(low); decision.Medium != "docker" {
		t.Fatalf("low medium = %q, want docker", decision.Medium)
	}
	if decision := Decide(high); decision.Medium != "vm" {
		t.Fatalf("high medium = %q, want vm", decision.Medium)
	}
	if decision := Decide(browser); decision.Medium != "browser" && decision.Medium != "docker" {
		t.Fatalf("browser medium = %q, want browser or docker", decision.Medium)
	}
}

func TestToolRuntimePolicyAndAuditChain(t *testing.T) {
	t.Parallel()

	run := &Run{}
	runtime := ToolRuntime{
		Policy: ToolPolicy{
			AllowedTools: []string{"github"},
			BlockedTools: []string{"browser"},
		},
		Handlers: map[string]Handler{
			"github": func(_ string, _ map[string]string) string { return "ok" },
		},
	}

	allow := runtime.Invoke("github", "execute", map[string]string{"repo": "OpenAGIs/BigClaw"}, run, "tool-runtime")
	block := runtime.Invoke("browser", "execute", map[string]string{"url": "https://example.com"}, run, "tool-runtime")

	if !allow.Success {
		t.Fatalf("allow result = %+v, want success", allow)
	}
	if block.Success {
		t.Fatalf("block result = %+v, want blocked failure", block)
	}

	outcomes := make([]string, 0, len(run.Audits))
	for _, audit := range run.Audits {
		if audit.Action == "tool.invoke" {
			outcomes = append(outcomes, audit.Outcome)
		}
	}
	if len(outcomes) != 2 || outcomes[0] != "success" || outcomes[1] != "blocked" {
		t.Fatalf("tool.invoke outcomes = %v", outcomes)
	}
}
