package runtimecompat

import "bigclaw-go/internal/domain"

type AuditRecord struct {
	Action  string `json:"action"`
	Outcome string `json:"outcome"`
}

type RunRecord struct {
	RunID  string        `json:"run_id"`
	Medium string        `json:"medium"`
	Audits []AuditRecord `json:"audits,omitempty"`
}

type ToolPolicy struct {
	AllowedTools []string `json:"allowed_tools,omitempty"`
	BlockedTools []string `json:"blocked_tools,omitempty"`
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
	ToolName string `json:"tool_name"`
	Action   string `json:"action"`
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

type Handler func(action string, payload map[string]any) string

type ToolRuntime struct {
	Policy   ToolPolicy
	Handlers map[string]Handler
}

func (r ToolRuntime) Invoke(toolName, action string, payload map[string]any, run *RunRecord) ToolCallResult {
	if !r.Policy.Allows(toolName) {
		if run != nil {
			run.Audits = append(run.Audits, AuditRecord{Action: "tool.invoke", Outcome: "blocked"})
		}
		return ToolCallResult{ToolName: toolName, Action: action, Success: false, Error: "tool blocked by policy"}
	}
	handler := r.Handlers[toolName]
	if handler == nil {
		handler = defaultHandler
	}
	output := handler(action, payloadOrEmpty(payload))
	if run != nil {
		run.Audits = append(run.Audits, AuditRecord{Action: "tool.invoke", Outcome: "success"})
	}
	return ToolCallResult{ToolName: toolName, Action: action, Success: true, Output: output}
}

type Decision struct {
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

type WorkerExecutionResult struct {
	ToolResults []ToolCallResult `json:"tool_results,omitempty"`
}

type Runtime struct {
	ToolRuntime ToolRuntime
}

func (r Runtime) Execute(task domain.Task, decision Decision, run *RunRecord, toolPayloads map[string]map[string]any) WorkerExecutionResult {
	if !decision.Approved {
		outcome := "waiting-approval"
		if decision.Medium == "none" {
			outcome = "paused"
		}
		if run != nil {
			run.Audits = append(run.Audits, AuditRecord{Action: "worker.lifecycle", Outcome: outcome})
		}
		return WorkerExecutionResult{}
	}
	results := make([]ToolCallResult, 0, len(task.RequiredTools))
	for _, toolName := range task.RequiredTools {
		results = append(results, r.ToolRuntime.Invoke(toolName, "execute", payloadOrEmpty(toolPayloads[toolName]), run))
	}
	if run != nil {
		run.Audits = append(run.Audits, AuditRecord{Action: "worker.lifecycle", Outcome: "completed"})
	}
	return WorkerExecutionResult{ToolResults: results}
}

func NewRunRecord(runID, medium string) *RunRecord {
	return &RunRecord{RunID: runID, Medium: medium}
}

func payloadOrEmpty(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	return payload
}

func defaultHandler(action string, payload map[string]any) string {
	return action
}
