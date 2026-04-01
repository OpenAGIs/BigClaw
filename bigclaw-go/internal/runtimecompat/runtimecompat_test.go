package runtimecompat

import (
	"strings"
	"testing"
)

func TestBig301WorkerLifecycleIsStableWithMultipleTools(t *testing.T) {
	task := Task{
		TaskID:        "BIG-301-matrix",
		Source:        "github",
		Title:         "worker lifecycle matrix",
		Description:   "validate stable lifecycle",
		RequiredTools: []string{"github", "browser"},
	}
	run := NewTaskRun(task, "run-big301-matrix", "docker")
	runtime := ClawWorkerRuntime{
		ToolRuntime: ToolRuntime{
			Handlers: map[string]ToolHandler{
				"github":  func(action string, payload map[string]string) string { return action + ":" + payload["repo"] },
				"browser": func(action string, payload map[string]string) string { return action + ":" + payload["url"] },
			},
		},
	}

	result := runtime.Execute(task, SchedulerDecision{Medium: "docker", Approved: true, Reason: "ok"}, run, "", map[string]map[string]string{
		"github":  {"repo": "OpenAGIs/BigClaw"},
		"browser": {"url": "https://example.com"},
	})

	if len(result.ToolResults) != 2 {
		t.Fatalf("expected 2 tool results, got %+v", result.ToolResults)
	}
	for _, item := range result.ToolResults {
		if !item.Success {
			t.Fatalf("expected successful tool result, got %+v", item)
		}
	}
	last := run.Audits[len(run.Audits)-1]
	if last.Action != "worker.lifecycle" || last.Outcome != "completed" {
		t.Fatalf("expected completed worker lifecycle audit, got %+v", last)
	}
}

func TestBig302RiskRoutesToExpectedSandboxMediums(t *testing.T) {
	scheduler := Scheduler{}

	low := Task{TaskID: "low", Source: "local", Title: "low", RiskLevel: RiskLow}
	high := Task{TaskID: "high", Source: "local", Title: "high", RiskLevel: RiskHigh}
	browser := Task{TaskID: "browser", Source: "local", Title: "browser", RequiredTools: []string{"browser"}, RiskLevel: RiskMedium}

	if got := scheduler.Decide(low).Medium; got != "docker" {
		t.Fatalf("expected docker route for low risk, got %q", got)
	}
	if got := scheduler.Decide(high).Medium; got != "vm" {
		t.Fatalf("expected vm route for high risk, got %q", got)
	}
	if got := scheduler.Decide(browser).Medium; got != "browser" && got != "docker" {
		t.Fatalf("expected browser-compatible route, got %q", got)
	}
}

func TestSchedulerHighRiskRequiresApproval(t *testing.T) {
	decision := Scheduler{}.Decide(Task{TaskID: "x", Source: "jira", Title: "prod op", RiskLevel: RiskHigh})
	if decision.Medium != "vm" || decision.Approved {
		t.Fatalf("expected unapproved vm decision, got %+v", decision)
	}
}

func TestSchedulerBrowserTaskRoutesBrowser(t *testing.T) {
	decision := Scheduler{}.Decide(Task{TaskID: "y", Source: "github", Title: "ui test", RequiredTools: []string{"browser"}})
	if decision.Medium != "browser" || !decision.Approved {
		t.Fatalf("expected approved browser route, got %+v", decision)
	}
}

func TestSchedulerOverBudgetDegradesBrowserTaskToDocker(t *testing.T) {
	decision := Scheduler{}.Decide(Task{
		TaskID:        "z",
		Source:        "github",
		Title:         "budgeted ui test",
		RequiredTools: []string{"browser"},
		Budget:        15.0,
	})
	if decision.Medium != "docker" || !decision.Approved {
		t.Fatalf("expected approved docker degradation, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "budget degraded browser route to docker") {
		t.Fatalf("expected budget degradation reason, got %+v", decision)
	}
}

func TestSchedulerOverBudgetPausesTask(t *testing.T) {
	decision := Scheduler{}.Decide(Task{TaskID: "b", Source: "linear", Title: "tiny budget", Budget: 5.0})
	if decision.Medium != "none" || decision.Approved {
		t.Fatalf("expected paused decision, got %+v", decision)
	}
	if decision.Reason != "paused: budget 5.0 below required docker budget 10.0" {
		t.Fatalf("unexpected pause reason: %+v", decision)
	}
}

func TestBig303ToolRuntimePolicyAndAuditChain(t *testing.T) {
	task := Task{TaskID: "BIG-303-matrix", Source: "local", Title: "tool policy", RequiredTools: []string{"github", "browser"}}
	run := NewTaskRun(task, "run-big303-matrix", "docker")
	runtime := ToolRuntime{
		Policy:   ToolPolicy{AllowedTools: []string{"github"}, BlockedTools: []string{"browser"}},
		Handlers: map[string]ToolHandler{"github": func(action string, payload map[string]string) string { return "ok" }},
	}

	allow := runtime.Invoke("github", "execute", map[string]string{"repo": "OpenAGIs/BigClaw"}, run, "")
	block := runtime.Invoke("browser", "execute", map[string]string{"url": "https://example.com"}, run, "")

	if !allow.Success {
		t.Fatalf("expected github invocation success, got %+v", allow)
	}
	if block.Success {
		t.Fatalf("expected browser invocation to be blocked, got %+v", block)
	}
	outcomes := make([]string, 0, len(run.Audits))
	for _, audit := range run.Audits {
		if audit.Action == "tool.invoke" {
			outcomes = append(outcomes, audit.Outcome)
		}
	}
	if len(outcomes) != 2 || outcomes[0] != "success" || outcomes[1] != "blocked" {
		t.Fatalf("unexpected tool invoke outcomes: %+v", outcomes)
	}
}
