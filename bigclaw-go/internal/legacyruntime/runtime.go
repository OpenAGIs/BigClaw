package legacyruntime

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/workflow"
)

type Queue struct {
	storagePath string
}

type queueSnapshot struct {
	Queue       []queueItem        `json:"queue"`
	DeadLetters []deadLetterRecord `json:"dead_letters"`
}

type queueItem struct {
	Priority int         `json:"priority"`
	TaskID   string      `json:"task_id"`
	Task     domain.Task `json:"task"`
}

type deadLetterRecord struct {
	TaskID string      `json:"task_id"`
	Task   domain.Task `json:"task"`
	Reason string      `json:"reason"`
}

func NewQueue(path string) *Queue {
	return &Queue{storagePath: strings.TrimSpace(path)}
}

func (q *Queue) Enqueue(task domain.Task) error {
	snapshot, err := q.loadSnapshot()
	if err != nil {
		return err
	}
	filtered := snapshot.Queue[:0]
	for _, item := range snapshot.Queue {
		if item.TaskID != task.ID {
			filtered = append(filtered, item)
		}
	}
	snapshot.Queue = append(filtered, queueItem{
		Priority: task.Priority,
		TaskID:   task.ID,
		Task:     task,
	})
	sortQueue(snapshot.Queue)
	return q.saveSnapshot(snapshot)
}

func (q *Queue) DequeueTask() (*domain.Task, error) {
	snapshot, err := q.loadSnapshot()
	if err != nil {
		return nil, err
	}
	if len(snapshot.Queue) == 0 {
		return nil, nil
	}
	item := snapshot.Queue[0]
	snapshot.Queue = snapshot.Queue[1:]
	if err := q.saveSnapshot(snapshot); err != nil {
		return nil, err
	}
	task := item.Task
	return &task, nil
}

func (q *Queue) loadSnapshot() (queueSnapshot, error) {
	if strings.TrimSpace(q.storagePath) == "" {
		return queueSnapshot{}, nil
	}
	body, err := os.ReadFile(q.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return queueSnapshot{}, nil
		}
		return queueSnapshot{}, err
	}
	if len(body) == 0 {
		return queueSnapshot{}, nil
	}
	var snapshot queueSnapshot
	if err := json.Unmarshal(body, &snapshot); err != nil {
		var legacyItems []queueItem
		if errLegacy := json.Unmarshal(body, &legacyItems); errLegacy != nil {
			return queueSnapshot{}, err
		}
		snapshot.Queue = legacyItems
	}
	sortQueue(snapshot.Queue)
	return snapshot, nil
}

func (q *Queue) saveSnapshot(snapshot queueSnapshot) error {
	if strings.TrimSpace(q.storagePath) == "" {
		return nil
	}
	body, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(q.storagePath), 0o755); err != nil {
		return err
	}
	tmpPath := q.storagePath + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, q.storagePath)
}

func sortQueue(items []queueItem) {
	slices.SortStableFunc(items, func(left, right queueItem) int {
		if left.Priority != right.Priority {
			return left.Priority - right.Priority
		}
		return strings.Compare(left.TaskID, right.TaskID)
	})
}

type Ledger struct {
	path string
}

func NewLedger(path string) *Ledger {
	return &Ledger{path: strings.TrimSpace(path)}
}

func (l *Ledger) Load() ([]map[string]any, error) {
	if strings.TrimSpace(l.path) == "" {
		return nil, nil
	}
	body, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if len(body) == 0 {
		return nil, nil
	}
	var entries []map[string]any
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l *Ledger) Append(run TaskRun) error {
	entries, err := l.Load()
	if err != nil {
		return err
	}
	body, err := json.Marshal(run)
	if err != nil {
		return err
	}
	var serialized map[string]any
	if err := json.Unmarshal(body, &serialized); err != nil {
		return err
	}
	entries = append(entries, serialized)
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	output, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, output, 0o644)
}

type SchedulerDecision struct {
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

type RiskScore struct {
	Level            string   `json:"level"`
	Total            int      `json:"total"`
	RequiresApproval bool     `json:"requires_approval"`
	Summary          string   `json:"summary"`
	Factors          []string `json:"factors,omitempty"`
}

type TraceEntry struct {
	Span       string         `json:"span"`
	Status     string         `json:"status"`
	Timestamp  string         `json:"timestamp"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type ArtifactRecord struct {
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	Path      string         `json:"path"`
	Timestamp string         `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AuditEntry struct {
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Outcome   string         `json:"outcome"`
	Timestamp string         `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

type TaskRun struct {
	RunID     string           `json:"run_id"`
	TaskID    string           `json:"task_id"`
	Source    string           `json:"source"`
	Title     string           `json:"title"`
	Medium    string           `json:"medium"`
	Status    string           `json:"status"`
	Summary   string           `json:"summary"`
	Logs      []map[string]any `json:"logs,omitempty"`
	Traces    []TraceEntry     `json:"traces,omitempty"`
	Artifacts []ArtifactRecord `json:"artifacts,omitempty"`
	Audits    []AuditEntry     `json:"audits,omitempty"`
}

type ExecutionRecord struct {
	Decision            SchedulerDecision                     `json:"decision"`
	Run                 TaskRun                               `json:"run"`
	ReportPath          string                                `json:"report_path,omitempty"`
	OrchestrationPlan   *workflow.OrchestrationPlan           `json:"orchestration_plan,omitempty"`
	OrchestrationPolicy *workflow.OrchestrationPolicyDecision `json:"orchestration_policy,omitempty"`
	HandoffRequest      *workflow.HandoffRequest              `json:"handoff_request,omitempty"`
	RiskScore           RiskScore                             `json:"risk_score"`
}

type Scheduler struct{}

func (Scheduler) Decide(task domain.Task) SchedulerDecision {
	switch {
	case task.BudgetCents < 0:
		return SchedulerDecision{Medium: "none", Approved: false, Reason: "invalid budget"}
	case task.RiskLevel == domain.RiskHigh:
		return SchedulerDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	case hasTool(task.RequiredTools, "browser"):
		return SchedulerDecision{Medium: "browser", Approved: true, Reason: "browser automation task"}
	case task.RiskLevel == domain.RiskMedium:
		return SchedulerDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"}
	default:
		return SchedulerDecision{Medium: "docker", Approved: true, Reason: "default low risk path"}
	}
}

func (s Scheduler) Execute(task domain.Task, runID string, ledger *Ledger, reportPath string) (ExecutionRecord, error) {
	decision := s.Decide(task)
	riskScore := scoreTask(task)
	rawPlan := workflow.CrossDepartmentOrchestrator{}.Plan(task)
	orchestrationPlan, policyDecision := workflow.PremiumOrchestrationPolicy{}.Apply(task, rawPlan)
	handoffRequest := buildHandoffRequest(decision, orchestrationPlan, policyDecision)
	run := TaskRun{
		RunID:   strings.TrimSpace(runID),
		TaskID:  task.ID,
		Source:  task.Source,
		Title:   task.Title,
		Medium:  decision.Medium,
		Status:  "running",
		Summary: "",
	}
	run.Logs = append(run.Logs, map[string]any{
		"level":     "info",
		"message":   "task received",
		"timestamp": utcNow(),
		"context": map[string]any{
			"source":   task.Source,
			"priority": task.Priority,
		},
	})
	run.Logs = append(run.Logs, map[string]any{
		"level":     "info",
		"message":   "orchestration planned",
		"timestamp": utcNow(),
		"context": map[string]any{
			"collaboration_mode": orchestrationPlan.CollaborationMode,
			"departments":        orchestrationPlan.Departments(),
		},
	})
	run.trace("scheduler.decide", map[bool]string{true: "ok", false: "pending"}[decision.Approved], map[string]any{
		"approved": decision.Approved,
		"medium":   decision.Medium,
	})
	run.trace("risk.score", riskScore.Level, map[string]any{
		"total":             riskScore.Total,
		"requires_approval": riskScore.RequiresApproval,
		"factors":           riskScore.Factors,
	})
	run.trace("orchestration.plan", "ready", map[string]any{
		"collaboration_mode": orchestrationPlan.CollaborationMode,
		"departments":        orchestrationPlan.Departments(),
		"handoffs":           orchestrationPlan.DepartmentCount(),
	})
	run.trace("orchestration.policy", map[bool]string{true: "upgrade-required", false: "ok"}[policyDecision.UpgradeRequired], map[string]any{
		"tier":                 policyDecision.Tier,
		"entitlement_status":   policyDecision.EntitlementStatus,
		"billing_model":        policyDecision.BillingModel,
		"estimated_cost_usd":   policyDecision.EstimatedCostUSD,
		"included_usage_units": policyDecision.IncludedUsageUnits,
		"overage_usage_units":  policyDecision.OverageUsageUnits,
		"overage_cost_usd":     policyDecision.OverageCostUSD,
		"blocked_departments":  policyDecision.BlockedDepartments,
	})
	run.audit("scheduler.decision", "scheduler", map[bool]string{true: "approved", false: "pending"}[decision.Approved], map[string]any{
		"reason": decision.Reason,
	})
	run.audit("risk.score", "scheduler", riskScore.Level, map[string]any{
		"total":             riskScore.Total,
		"requires_approval": riskScore.RequiresApproval,
		"summary":           riskScore.Summary,
	})
	run.audit("orchestration.plan", "scheduler", "ready", map[string]any{
		"collaboration_mode": orchestrationPlan.CollaborationMode,
		"departments":        orchestrationPlan.Departments(),
		"approvals":          orchestrationPlan.RequiredApprovals(),
	})
	run.audit("orchestration.policy", "scheduler", map[bool]string{true: "upgrade-required", false: "enabled"}[policyDecision.UpgradeRequired], map[string]any{
		"tier":                 policyDecision.Tier,
		"reason":               policyDecision.Reason,
		"entitlement_status":   policyDecision.EntitlementStatus,
		"billing_model":        policyDecision.BillingModel,
		"estimated_cost_usd":   policyDecision.EstimatedCostUSD,
		"included_usage_units": policyDecision.IncludedUsageUnits,
		"overage_usage_units":  policyDecision.OverageUsageUnits,
		"overage_cost_usd":     policyDecision.OverageCostUSD,
		"blocked_departments":  policyDecision.BlockedDepartments,
	})
	if handoffRequest != nil {
		run.trace("orchestration.handoff", handoffRequest.Status, map[string]any{
			"target_team":        handoffRequest.TargetTeam,
			"required_approvals": handoffRequest.RequiredApprovals,
		})
		run.audit("orchestration.handoff", "scheduler", handoffRequest.Status, map[string]any{
			"target_team":        handoffRequest.TargetTeam,
			"reason":             handoffRequest.Reason,
			"required_approvals": handoffRequest.RequiredApprovals,
		})
	}

	finalStatus := "approved"
	if !decision.Approved {
		if decision.Medium == "none" {
			finalStatus = "paused"
		} else {
			finalStatus = "needs-approval"
		}
	}
	run.Status = finalStatus
	run.Summary = decision.Reason

	resolvedReportPath := strings.TrimSpace(reportPath)
	if resolvedReportPath != "" {
		if err := writeReports(resolvedReportPath, run); err != nil {
			return ExecutionRecord{}, err
		}
		run.registerArtifact("task-run-detail", "page", strings.TrimSpace(strings.TrimSuffix(resolvedReportPath, filepath.Ext(resolvedReportPath))+".html"), map[string]any{"format": "html"})
		run.registerArtifact("task-run-report", "report", resolvedReportPath, map[string]any{"format": "markdown"})
	}

	if ledger != nil {
		if err := ledger.Append(run); err != nil {
			return ExecutionRecord{}, err
		}
	}
	return ExecutionRecord{
		Decision:            decision,
		Run:                 run,
		ReportPath:          resolvedReportPath,
		OrchestrationPlan:   &orchestrationPlan,
		OrchestrationPolicy: &policyDecision,
		HandoffRequest:      handoffRequest,
		RiskScore:           riskScore,
	}, nil
}

func RenderOrchestrationPlan(
	plan workflow.OrchestrationPlan,
	policyDecision *workflow.OrchestrationPolicyDecision,
	handoffRequest *workflow.HandoffRequest,
) string {
	lines := []string{
		"# Cross-Department Orchestration Plan",
		"",
		fmt.Sprintf("- Task ID: %s", plan.TaskID),
		fmt.Sprintf("- Collaboration Mode: %s", plan.CollaborationMode),
		fmt.Sprintf("- Departments: %s", joinedOrNone(plan.Departments())),
		fmt.Sprintf("- Required Approvals: %s", joinedOrNone(plan.RequiredApprovals())),
	}
	if policyDecision != nil {
		lines = append(lines,
			fmt.Sprintf("- Tier: %s", policyDecision.Tier),
			fmt.Sprintf("- Upgrade Required: %t", policyDecision.UpgradeRequired),
			fmt.Sprintf("- Entitlement Status: %s", policyDecision.EntitlementStatus),
			fmt.Sprintf("- Billing Model: %s", policyDecision.BillingModel),
			fmt.Sprintf("- Estimated Cost (USD): %.2f", policyDecision.EstimatedCostUSD),
			fmt.Sprintf("- Included Usage Units: %d", policyDecision.IncludedUsageUnits),
			fmt.Sprintf("- Overage Usage Units: %d", policyDecision.OverageUsageUnits),
			fmt.Sprintf("- Overage Cost (USD): %.2f", policyDecision.OverageCostUSD),
			fmt.Sprintf("- Policy Reason: %s", policyDecision.Reason),
			fmt.Sprintf("- Blocked Departments: %s", joinedOrNone(policyDecision.BlockedDepartments)),
		)
	}
	if handoffRequest != nil {
		lines = append(lines,
			fmt.Sprintf("- Human Handoff Team: %s", handoffRequest.TargetTeam),
			fmt.Sprintf("- Human Handoff Status: %s", handoffRequest.Status),
			fmt.Sprintf("- Human Handoff Reason: %s", handoffRequest.Reason),
			fmt.Sprintf("- Human Handoff Approvals: %s", joinedOrNone(handoffRequest.RequiredApprovals)),
		)
	}
	lines = append(lines, "", "## Handoffs", "")
	if len(plan.Handoffs) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, handoff := range plan.Handoffs {
			lines = append(lines, fmt.Sprintf(
				"- %s: reason=%s tools=%s approvals=%s",
				handoff.Department,
				handoff.Reason,
				joinedOrNone(handoff.RequiredTools),
				joinedOrNone(handoff.Approvals),
			))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func buildHandoffRequest(
	decision SchedulerDecision,
	plan workflow.OrchestrationPlan,
	policyDecision workflow.OrchestrationPolicyDecision,
) *workflow.HandoffRequest {
	if !decision.Approved {
		requiredApprovals := plan.RequiredApprovals()
		if len(requiredApprovals) == 0 {
			requiredApprovals = []string{"security-review"}
		}
		return &workflow.HandoffRequest{
			TargetTeam:        "security",
			Reason:            decision.Reason,
			Status:            "pending",
			RequiredApprovals: requiredApprovals,
		}
	}
	if policyDecision.UpgradeRequired {
		return &workflow.HandoffRequest{
			TargetTeam:        "operations",
			Reason:            policyDecision.Reason,
			Status:            "pending",
			RequiredApprovals: []string{"ops-manager"},
		}
	}
	return nil
}

func scoreTask(task domain.Task) RiskScore {
	if task.RiskLevel == domain.RiskHigh {
		return RiskScore{
			Level:            "high",
			Total:            90,
			RequiresApproval: true,
			Summary:          "high-risk task requires manual approval",
			Factors:          []string{"risk_level"},
		}
	}
	if task.RiskLevel == domain.RiskMedium || hasTool(task.RequiredTools, "browser") {
		return RiskScore{
			Level:            "medium",
			Total:            50,
			RequiresApproval: false,
			Summary:          "medium-risk task can proceed automatically",
			Factors:          []string{"tooling"},
		}
	}
	return RiskScore{
		Level:            "low",
		Total:            10,
		RequiresApproval: false,
		Summary:          "low-risk task can proceed automatically",
		Factors:          []string{"default"},
	}
}

func hasTool(tools []string, want string) bool {
	for _, tool := range tools {
		if strings.EqualFold(strings.TrimSpace(tool), want) {
			return true
		}
	}
	return false
}

func joinedOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (r *TaskRun) trace(span, status string, attributes map[string]any) {
	r.Traces = append(r.Traces, TraceEntry{
		Span:       strings.TrimSpace(span),
		Status:     strings.TrimSpace(status),
		Timestamp:  utcNow(),
		Attributes: cloneMap(attributes),
	})
}

func (r *TaskRun) audit(action, actor, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:    strings.TrimSpace(action),
		Actor:     strings.TrimSpace(actor),
		Outcome:   strings.TrimSpace(outcome),
		Timestamp: utcNow(),
		Details:   cloneMap(details),
	})
}

func (r *TaskRun) registerArtifact(name, kind, path string, metadata map[string]any) {
	r.Artifacts = append(r.Artifacts, ArtifactRecord{
		Name:      strings.TrimSpace(name),
		Kind:      strings.TrimSpace(kind),
		Path:      strings.TrimSpace(path),
		Timestamp: utcNow(),
		Metadata:  cloneMap(metadata),
	})
}

func writeReports(reportPath string, run TaskRun) error {
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	markdown := fmt.Sprintf(
		"# Task Run Report\n\n- Run ID: %s\n- Task ID: %s\n- Medium: %s\n- Status: %s\n- Summary: %s\n",
		run.RunID,
		run.TaskID,
		run.Medium,
		run.Status,
		run.Summary,
	)
	if err := os.WriteFile(reportPath, []byte(markdown), 0o644); err != nil {
		return err
	}
	htmlPath := strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".html"
	body := fmt.Sprintf("<html><body><h1>Task Run Report</h1><p>Status: %s</p><pre>%s</pre></body></html>", html.EscapeString(run.Status), html.EscapeString(markdown))
	return os.WriteFile(htmlPath, []byte(body), 0o644)
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
