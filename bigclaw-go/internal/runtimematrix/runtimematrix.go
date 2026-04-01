package runtimematrix

import "bigclaw-go/internal/domain"

type AuditEntry struct {
	Action  string         `json:"action"`
	Actor   string         `json:"actor"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	TaskID string       `json:"task_id"`
	RunID  string       `json:"run_id"`
	Medium string       `json:"medium"`
	Audits []AuditEntry `json:"audits"`
}

type Decision struct {
	Medium   string
	Approved bool
	Reason   string
}

type ToolResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

type ExecuteResult struct {
	ToolResults []ToolResult `json:"tool_results"`
}

type ToolHandler func(action string, payload map[string]any) string

type ToolPolicy struct {
	AllowedTools []string
	BlockedTools []string
}

type ToolRuntime struct {
	Policy   ToolPolicy
	Handlers map[string]ToolHandler
}

type ClawWorkerRuntime struct {
	ToolRuntime *ToolRuntime
}

func NewTaskRun(task domain.Task, runID string, medium string) *TaskRun {
	return &TaskRun{
		TaskID: task.ID,
		RunID:  runID,
		Medium: medium,
		Audits: []AuditEntry{},
	}
}

func (r *TaskRun) Audit(action string, actor string, outcome string, details map[string]any) {
	if r == nil {
		return
	}
	r.Audits = append(r.Audits, AuditEntry{
		Action:  action,
		Actor:   actor,
		Outcome: outcome,
		Details: details,
	})
}

func (w ClawWorkerRuntime) Execute(task domain.Task, decision Decision, run *TaskRun, toolPayloads map[string]map[string]any) ExecuteResult {
	result := ExecuteResult{ToolResults: make([]ToolResult, 0, len(task.RequiredTools))}
	if run != nil {
		run.Audit("worker.lifecycle", "worker", "started", map[string]any{
			"task_id":  task.ID,
			"medium":   decision.Medium,
			"approved": decision.Approved,
			"reason":   decision.Reason,
		})
	}
	for _, tool := range task.RequiredTools {
		payload := toolPayloads[tool]
		toolResult := ToolResult{Tool: tool}
		if w.ToolRuntime == nil {
			toolResult.Error = "tool runtime unavailable"
		} else {
			toolResult = w.ToolRuntime.Invoke(tool, "execute", payload, run)
		}
		result.ToolResults = append(result.ToolResults, toolResult)
	}
	if run != nil {
		run.Audit("worker.lifecycle", "worker", "completed", map[string]any{
			"task_id":      task.ID,
			"result_count": len(result.ToolResults),
		})
	}
	return result
}

func (r ToolRuntime) Invoke(tool string, action string, payload map[string]any, run *TaskRun) ToolResult {
	result := ToolResult{Tool: tool}
	if r.Policy.isBlocked(tool) || !r.Policy.isAllowed(tool) {
		result.Error = "tool blocked by policy"
		if run != nil {
			run.Audit("tool.invoke", "worker", "blocked", map[string]any{"tool": tool, "action": action})
		}
		return result
	}
	handler, ok := r.Handlers[tool]
	if !ok {
		result.Error = "no handler registered"
		if run != nil {
			run.Audit("tool.invoke", "worker", "missing_handler", map[string]any{"tool": tool, "action": action})
		}
		return result
	}
	result.Success = true
	result.Output = handler(action, payload)
	if run != nil {
		run.Audit("tool.invoke", "worker", "success", map[string]any{"tool": tool, "action": action})
	}
	return result
}

type Scheduler struct{}

func (Scheduler) Decide(task domain.Task) Decision {
	for _, tool := range task.RequiredTools {
		if tool == "browser" {
			return Decision{Medium: "browser", Approved: true, Reason: "browser workload requires browser medium"}
		}
	}
	if task.RiskLevel == domain.RiskHigh {
		return Decision{Medium: "vm", Approved: true, Reason: "high risk workload requires vm"}
	}
	return Decision{Medium: "docker", Approved: true, Reason: "default docker medium"}
}

func (p ToolPolicy) isAllowed(tool string) bool {
	if len(p.AllowedTools) == 0 {
		return true
	}
	for _, candidate := range p.AllowedTools {
		if candidate == tool {
			return true
		}
	}
	return false
}

func (p ToolPolicy) isBlocked(tool string) bool {
	for _, candidate := range p.BlockedTools {
		if candidate == tool {
			return true
		}
	}
	return false
}
