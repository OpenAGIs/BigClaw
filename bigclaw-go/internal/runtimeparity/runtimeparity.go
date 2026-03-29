package runtimeparity

import "fmt"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Task struct {
	ID            string
	Source        string
	Title         string
	Description   string
	RequiredTools []string
	RiskLevel     RiskLevel
}

type Decision struct {
	Medium   string
	Approved bool
	Reason   string
}

type Audit struct {
	Action  string
	Actor   string
	Outcome string
	Details map[string]any
}

type Run struct {
	Audits []Audit
}

func (r *Run) Audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, Audit{
		Action:  action,
		Actor:   actor,
		Outcome: outcome,
		Details: cloneMap(details),
	})
}

type SandboxProfile struct {
	Medium           string
	Isolation        string
	NetworkAccess    string
	FilesystemAccess string
}

type SandboxRouter struct{}

func (SandboxRouter) ProfileFor(medium string) SandboxProfile {
	switch medium {
	case "docker":
		return SandboxProfile{
			Medium:           "docker",
			Isolation:        "container",
			NetworkAccess:    "restricted",
			FilesystemAccess: "workspace-write",
		}
	case "browser":
		return SandboxProfile{
			Medium:           "browser",
			Isolation:        "browser-automation",
			NetworkAccess:    "enabled",
			FilesystemAccess: "downloads-only",
		}
	case "vm":
		return SandboxProfile{
			Medium:           "vm",
			Isolation:        "virtual-machine",
			NetworkAccess:    "restricted",
			FilesystemAccess: "workspace-write",
		}
	default:
		return SandboxProfile{
			Medium:           "none",
			Isolation:        "disabled",
			NetworkAccess:    "none",
			FilesystemAccess: "none",
		}
	}
}

func Decide(task Task) Decision {
	if task.RiskLevel == RiskHigh {
		return Decision{
			Medium:   "vm",
			Approved: false,
			Reason:   "requires approval for high-risk task",
		}
	}
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return Decision{
				Medium:   "browser",
				Approved: true,
				Reason:   "browser automation task",
			}
		}
	}
	return Decision{
		Medium:   "docker",
		Approved: true,
		Reason:   "default low/medium risk path",
	}
}

type ToolPolicy struct {
	AllowedTools []string
	BlockedTools []string
}

func (p ToolPolicy) Allows(toolName string) bool {
	for _, blocked := range p.BlockedTools {
		if blocked == toolName {
			return false
		}
	}
	if len(p.AllowedTools) == 0 {
		return true
	}
	for _, allowed := range p.AllowedTools {
		if allowed == toolName {
			return true
		}
	}
	return false
}

type ToolCallResult struct {
	ToolName string
	Action   string
	Success  bool
	Output   string
	Error    string
}

type Handler func(action string, payload map[string]string) string

type ToolRuntime struct {
	Policy   ToolPolicy
	Handlers map[string]Handler
}

func (r ToolRuntime) Invoke(toolName, action string, payload map[string]string, run *Run, actor string) ToolCallResult {
	if !r.Policy.Allows(toolName) {
		if run != nil {
			run.Audit("tool.invoke", actor, "blocked", map[string]any{
				"tool":      toolName,
				"operation": action,
			})
		}
		return ToolCallResult{
			ToolName: toolName,
			Action:   action,
			Success:  false,
			Error:    "tool blocked by policy",
		}
	}

	handler := r.Handlers[toolName]
	if handler == nil {
		handler = defaultHandler
	}
	output := handler(action, payload)
	if run != nil {
		run.Audit("tool.invoke", actor, "success", map[string]any{
			"tool":      toolName,
			"operation": action,
		})
	}
	return ToolCallResult{
		ToolName: toolName,
		Action:   action,
		Success:  true,
		Output:   output,
	}
}

func defaultHandler(action string, payload map[string]string) string {
	if len(payload) == 0 {
		return action
	}
	return fmt.Sprintf("%s:%v", action, payload)
}

type WorkerExecutionResult struct {
	ToolResults    []ToolCallResult
	SandboxProfile SandboxProfile
}

type WorkerRuntime struct {
	ToolRuntime   ToolRuntime
	SandboxRouter SandboxRouter
}

func (w WorkerRuntime) Execute(task Task, decision Decision, run *Run, toolPayloads map[string]map[string]string, actor string) WorkerExecutionResult {
	profile := w.SandboxRouter.ProfileFor(decision.Medium)
	if !decision.Approved {
		if run != nil {
			run.Audit("worker.lifecycle", actor, "waiting-approval", map[string]any{
				"medium":         decision.Medium,
				"required_tools": append([]string(nil), task.RequiredTools...),
			})
		}
		return WorkerExecutionResult{SandboxProfile: profile}
	}

	results := make([]ToolCallResult, 0, len(task.RequiredTools))
	for _, toolName := range task.RequiredTools {
		results = append(results, w.ToolRuntime.Invoke(toolName, "execute", toolPayloads[toolName], run, actor))
	}
	if run != nil {
		run.Audit("worker.lifecycle", actor, "completed", map[string]any{
			"medium":           decision.Medium,
			"successful_tools": successfulTools(results),
			"blocked_tools":    blockedTools(results),
		})
	}
	return WorkerExecutionResult{
		ToolResults:    results,
		SandboxProfile: profile,
	}
}

func successfulTools(results []ToolCallResult) []string {
	names := make([]string, 0, len(results))
	for _, result := range results {
		if result.Success {
			names = append(names, result.ToolName)
		}
	}
	return names
}

func blockedTools(results []ToolCallResult) []string {
	names := make([]string, 0, len(results))
	for _, result := range results {
		if !result.Success {
			names = append(names, result.ToolName)
		}
	}
	return names
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}
