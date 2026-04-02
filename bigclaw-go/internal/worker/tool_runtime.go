package worker

import (
	"fmt"

	"bigclaw-go/internal/domain"
)

type ToolPolicy struct {
	AllowedTools []string
	BlockedTools []string
}

type ToolInvocationAudit struct {
	Action  string
	Outcome string
	Tool    string
}

type ToolRun struct {
	TaskID  string
	RunID   string
	Medium  string
	Audits  []ToolInvocationAudit
	Summary string
}

type ToolResult struct {
	Tool    string
	Success bool
	Output  string
}

type ToolRuntime struct {
	Policy   ToolPolicy
	Handlers map[string]func(action string, payload map[string]any) string
}

type ToolDecision struct {
	Medium   string
	Approved bool
	Reason   string
}

type WorkerExecutionResult struct {
	ToolResults []ToolResult
}

type ClawWorkerRuntime struct {
	ToolRuntime ToolRuntime
}

func NewToolRun(task domain.Task, runID string, medium string) *ToolRun {
	return &ToolRun{
		TaskID: task.ID,
		RunID:  runID,
		Medium: medium,
		Audits: []ToolInvocationAudit{},
	}
}

func (r *ToolRun) audit(action string, outcome string, tool string) {
	r.Audits = append(r.Audits, ToolInvocationAudit{
		Action:  action,
		Outcome: outcome,
		Tool:    tool,
	})
}

func (t ToolRuntime) Invoke(tool string, action string, payload map[string]any, run *ToolRun) ToolResult {
	if containsTool(t.Policy.BlockedTools, tool) || (len(t.Policy.AllowedTools) > 0 && !containsTool(t.Policy.AllowedTools, tool)) {
		if run != nil {
			run.audit("tool.invoke", "blocked", tool)
		}
		return ToolResult{Tool: tool, Success: false, Output: "blocked"}
	}
	handler, ok := t.Handlers[tool]
	if !ok {
		if run != nil {
			run.audit("tool.invoke", "blocked", tool)
		}
		return ToolResult{Tool: tool, Success: false, Output: "missing-handler"}
	}
	output := handler(action, payload)
	if run != nil {
		run.audit("tool.invoke", "success", tool)
	}
	return ToolResult{Tool: tool, Success: true, Output: output}
}

func (w ClawWorkerRuntime) Execute(task domain.Task, decision ToolDecision, run *ToolRun, toolPayloads map[string]map[string]any) WorkerExecutionResult {
	results := make([]ToolResult, 0, len(task.RequiredTools))
	for _, tool := range task.RequiredTools {
		payload := map[string]any{}
		if provided, ok := toolPayloads[tool]; ok && provided != nil {
			payload = provided
		}
		results = append(results, w.ToolRuntime.Invoke(tool, "execute", payload, run))
	}
	if run != nil {
		outcome := "completed"
		for _, result := range results {
			if !result.Success {
				outcome = "blocked"
				break
			}
		}
		run.audit("worker.lifecycle", outcome, fmt.Sprintf("%s:%t", decision.Medium, decision.Approved))
		run.Summary = decision.Reason
	}
	return WorkerExecutionResult{ToolResults: results}
}

func containsTool(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
