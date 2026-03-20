package product

import (
	"encoding/json"
	"sort"
	"strings"
)

type ContractField struct {
	Name        string `json:"name"`
	FieldType   string `json:"field_type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type ContractSurface struct {
	Name        string          `json:"name"`
	Owner       string          `json:"owner"`
	Description string          `json:"description,omitempty"`
	Fields      []ContractField `json:"fields,omitempty"`
	Sample      map[string]any  `json:"sample,omitempty"`
}

type DashboardRunContract struct {
	ContractID      string          `json:"contract_id"`
	Version         string          `json:"version"`
	DashboardSchema ContractSurface `json:"dashboard_schema"`
	RunDetailSchema ContractSurface `json:"run_detail_schema"`
}

type DashboardRunContractAudit struct {
	ContractID             string   `json:"contract_id"`
	Version                string   `json:"version"`
	DashboardMissingFields []string `json:"dashboard_missing_fields,omitempty"`
	DashboardSampleGaps    []string `json:"dashboard_sample_gaps,omitempty"`
	RunDetailMissingFields []string `json:"run_detail_missing_fields,omitempty"`
	RunDetailSampleGaps    []string `json:"run_detail_sample_gaps,omitempty"`
	ReleaseReady           bool     `json:"release_ready"`
}

var dashboardContractRequiredFields = []string{
	"dashboard_id",
	"generated_at",
	"filters.team",
	"filters.project",
	"filters.viewer_team",
	"summary.total_tasks",
	"summary.active_runs",
	"summary.blockers",
	"summary.premium_runs",
	"summary.sla_risk_runs",
	"summary.budget_cents_total",
	"ticket_to_merge_funnel.tickets",
	"ticket_to_merge_funnel.prs_opened",
	"ticket_to_merge_funnel.merged_prs",
	"project_breakdown",
	"team_breakdown",
	"trend",
	"blocked_tasks",
	"blocked_tasks[].task.id",
	"high_risk_tasks",
	"high_risk_tasks[].task.id",
	"tasks",
	"tasks[].task.id",
	"tasks[].policy.plan",
	"tasks[].risk_score.total",
	"tasks[].drilldown.run",
}

var runDetailContractRequiredFields = []string{
	"task.id",
	"task.trace_id",
	"state",
	"policy.plan",
	"risk_score.total",
	"timeline",
	"timeline[].id",
	"timeline[].type",
	"artifacts.report",
	"artifact_refs",
	"artifact_refs[].kind",
	"tool_traces",
	"tool_traces[].status",
	"audit_summary.total",
	"reports",
	"reports[].url",
	"closeout.validation_evidence",
	"closeout.git_push_succeeded",
	"closeout.git_log_stat_output",
	"closeout.remote_synced",
	"closeout.complete",
}

func BuildDefaultDashboardRunContract() DashboardRunContract {
	return DashboardRunContract{
		ContractID: "BIG-GOM-305",
		Version:    "go-v1",
		DashboardSchema: ContractSurface{
			Name:        "EngineeringDashboard",
			Owner:       "api",
			Description: "Canonical Go-owned operator dashboard payload for engineering and operations status views.",
			Fields: []ContractField{
				{Name: "dashboard_id", FieldType: "string", Required: true, Description: "Stable dashboard identifier."},
				{Name: "generated_at", FieldType: "datetime", Required: true, Description: "UTC generation timestamp."},
				{Name: "filters.team", FieldType: "string", Required: true, Description: "Team filter applied to the dashboard."},
				{Name: "filters.project", FieldType: "string", Required: true, Description: "Project filter applied to the dashboard."},
				{Name: "filters.viewer_team", FieldType: "string", Required: true, Description: "Viewer team used for scoping."},
				{Name: "summary.total_tasks", FieldType: "integer", Required: true, Description: "Total tasks included in the dashboard window."},
				{Name: "summary.active_runs", FieldType: "integer", Required: true, Description: "Queued, leased, running, or retrying tasks."},
				{Name: "summary.blockers", FieldType: "integer", Required: true, Description: "Blocked tasks needing operator attention."},
				{Name: "summary.premium_runs", FieldType: "integer", Required: true, Description: "Premium plan tasks in scope."},
				{Name: "summary.sla_risk_runs", FieldType: "integer", Required: true, Description: "Tasks at elevated operational risk."},
				{Name: "summary.budget_cents_total", FieldType: "integer", Required: true, Description: "Budget sum across in-scope tasks."},
				{Name: "summary.state_distribution", FieldType: "map[string]int", Required: false, Description: "State counts for charting."},
				{Name: "ticket_to_merge_funnel.tickets", FieldType: "integer", Required: true, Description: "Total tasks considered for the delivery funnel."},
				{Name: "ticket_to_merge_funnel.prs_opened", FieldType: "integer", Required: true, Description: "Tasks with PR activity."},
				{Name: "ticket_to_merge_funnel.merged_prs", FieldType: "integer", Required: true, Description: "Tasks with merged PRs."},
				{Name: "project_breakdown", FieldType: "DashboardBreakdown[]", Required: true, Description: "Project-level dashboard breakdown."},
				{Name: "team_breakdown", FieldType: "DashboardBreakdown[]", Required: true, Description: "Team-level dashboard breakdown."},
				{Name: "trend", FieldType: "DashboardTrendPoint[]", Required: true, Description: "Time-bucket trend points for the selected range."},
				{Name: "blocked_tasks", FieldType: "DashboardTaskOverview[]", Required: true, Description: "Blocked task drilldowns."},
				{Name: "blocked_tasks[].task.id", FieldType: "string", Required: true, Description: "Blocked task identifier."},
				{Name: "high_risk_tasks", FieldType: "DashboardTaskOverview[]", Required: true, Description: "High-risk task drilldowns."},
				{Name: "high_risk_tasks[].task.id", FieldType: "string", Required: true, Description: "High-risk task identifier."},
				{Name: "tasks", FieldType: "DashboardTaskOverview[]", Required: true, Description: "Ordered dashboard task overviews."},
				{Name: "tasks[].task.id", FieldType: "string", Required: true, Description: "Dashboard task identifier."},
				{Name: "tasks[].policy.plan", FieldType: "string", Required: true, Description: "Resolved policy lane."},
				{Name: "tasks[].risk_score.total", FieldType: "integer", Required: true, Description: "Resolved risk score."},
				{Name: "tasks[].drilldown.run", FieldType: "string", Required: true, Description: "Run detail URL."},
			},
			Sample: map[string]any{
				"dashboard_id": "engineering-dashboard-platform-alpha",
				"generated_at": "2026-03-19T07:00:00Z",
				"filters": map[string]any{
					"team":        "platform",
					"project":     "alpha",
					"viewer_team": "platform",
				},
				"summary": map[string]any{
					"total_tasks":        2,
					"active_runs":        1,
					"blockers":           1,
					"premium_runs":       1,
					"sla_risk_runs":      1,
					"budget_cents_total": 1200,
					"state_distribution": map[string]any{"blocked": 1, "running": 1},
				},
				"ticket_to_merge_funnel": map[string]any{
					"tickets":    2,
					"prs_opened": 1,
					"merged_prs": 0,
				},
				"project_breakdown": []map[string]any{
					{"key": "alpha", "total_tasks": 2, "active_runs": 1, "blockers": 1, "premium_runs": 1, "sla_risk_runs": 1, "budget_cents_total": 1200, "merged_prs": 0},
				},
				"team_breakdown": []map[string]any{
					{"key": "platform", "total_tasks": 2, "active_runs": 1, "blockers": 1, "premium_runs": 1, "sla_risk_runs": 1, "budget_cents_total": 1200, "merged_prs": 0},
				},
				"trend": []map[string]any{
					{"label": "2026-03-19", "total_tasks": 2, "active_runs": 1, "blockers": 1, "premium_runs": 1, "sla_risk_runs": 1, "budget_cents_total": 1200},
				},
				"blocked_tasks": []map[string]any{
					{
						"task":       map[string]any{"id": "task-platform-1", "title": "Review release blocker", "trace_id": "trace-platform-1"},
						"policy":     map[string]any{"plan": "premium"},
						"risk_score": map[string]any{"total": 65},
						"drilldown":  map[string]any{"run": "/v2/runs/task-platform-1"},
					},
				},
				"high_risk_tasks": []map[string]any{
					{
						"task":       map[string]any{"id": "task-platform-1", "title": "Review release blocker", "trace_id": "trace-platform-1"},
						"policy":     map[string]any{"plan": "premium"},
						"risk_score": map[string]any{"total": 65},
						"drilldown":  map[string]any{"run": "/v2/runs/task-platform-1"},
					},
				},
				"tasks": []map[string]any{
					{
						"task":       map[string]any{"id": "task-platform-1", "title": "Review release blocker", "trace_id": "trace-platform-1"},
						"policy":     map[string]any{"plan": "premium"},
						"risk_score": map[string]any{"total": 65},
						"drilldown":  map[string]any{"run": "/v2/runs/task-platform-1"},
					},
					{
						"task":       map[string]any{"id": "task-platform-2", "title": "Validate rollout", "trace_id": "trace-platform-2"},
						"policy":     map[string]any{"plan": "standard"},
						"risk_score": map[string]any{"total": 20},
						"drilldown":  map[string]any{"run": "/v2/runs/task-platform-2"},
					},
				},
			},
		},
		RunDetailSchema: ContractSurface{
			Name:        "RunDetail",
			Owner:       "api",
			Description: "Canonical Go-owned run detail payload for replay, artifacts, audit history, and closeout evidence.",
			Fields: []ContractField{
				{Name: "task.id", FieldType: "string", Required: true, Description: "Task identifier."},
				{Name: "task.trace_id", FieldType: "string", Required: true, Description: "Trace identifier."},
				{Name: "state", FieldType: "string", Required: true, Description: "Current task state."},
				{Name: "policy.plan", FieldType: "string", Required: true, Description: "Resolved policy lane."},
				{Name: "risk_score.total", FieldType: "integer", Required: true, Description: "Resolved risk score total."},
				{Name: "timeline", FieldType: "Event[]", Required: true, Description: "Chronological execution events."},
				{Name: "timeline[].id", FieldType: "string", Required: true, Description: "Stable event identifier."},
				{Name: "timeline[].type", FieldType: "string", Required: true, Description: "Event type."},
				{Name: "artifacts.report", FieldType: "string", Required: true, Description: "Run report endpoint URL."},
				{Name: "artifact_refs", FieldType: "RunArtifactRef[]", Required: true, Description: "Collected run artifacts and linked resources."},
				{Name: "artifact_refs[].kind", FieldType: "string", Required: true, Description: "Artifact kind."},
				{Name: "tool_traces", FieldType: "RunToolTrace[]", Required: true, Description: "Observed tool traces from declared tools and events."},
				{Name: "tool_traces[].status", FieldType: "string", Required: true, Description: "Tool trace lifecycle status."},
				{Name: "audit_summary.total", FieldType: "integer", Required: true, Description: "Recent action count."},
				{Name: "reports", FieldType: "RunReportLink[]", Required: true, Description: "Downloadable run report links."},
				{Name: "reports[].url", FieldType: "string", Required: true, Description: "Report URL."},
				{Name: "closeout.validation_evidence", FieldType: "string[]", Required: true, Description: "Validation evidence captured before completion."},
				{Name: "closeout.git_push_succeeded", FieldType: "boolean", Required: true, Description: "Whether git push succeeded."},
				{Name: "closeout.git_push_output", FieldType: "string", Required: false, Description: "Push command output if recorded."},
				{Name: "closeout.git_log_stat_output", FieldType: "string", Required: true, Description: "Captured git log -1 --stat output."},
				{Name: "closeout.remote_synced", FieldType: "boolean", Required: true, Description: "Whether local and remote branch SHAs matched."},
				{Name: "closeout.complete", FieldType: "boolean", Required: true, Description: "Whether minimum closeout evidence is complete."},
			},
			Sample: map[string]any{
				"task": map[string]any{
					"id":       "task-platform-1",
					"trace_id": "trace-platform-1",
					"title":    "Review release blocker",
				},
				"state": "blocked",
				"policy": map[string]any{
					"plan": "premium",
				},
				"risk_score": map[string]any{
					"total": 65,
				},
				"timeline": []map[string]any{
					{"id": "evt-1", "type": "task.started", "timestamp": "2026-03-19T06:50:00Z"},
					{"id": "evt-2", "type": "run.takeover", "timestamp": "2026-03-19T06:55:00Z"},
				},
				"artifacts": map[string]any{
					"report": "/v2/runs/task-platform-1/report?limit=20",
					"audit":  "/v2/runs/task-platform-1/audit?limit=20",
				},
				"artifact_refs": []map[string]any{
					{"kind": "workpad", "uri": "https://docs.example.com/workpads/task-platform-1"},
				},
				"tool_traces": []map[string]any{
					{"name": "browser", "status": "required"},
					{"name": "scheduler", "status": "routed"},
				},
				"audit_summary": map[string]any{
					"total": 1,
				},
				"reports": []map[string]any{
					{"url": "/v2/runs/task-platform-1/report?limit=20", "format": "markdown", "download": true},
				},
				"closeout": map[string]any{
					"validation_evidence": []string{"go test ./internal/api", "git log -1 --stat"},
					"git_push_succeeded":  true,
					"git_push_output":     "To github.com:OpenAGIs/BigClaw.git",
					"git_log_stat_output": "commit abc123\n bigclaw-go/internal/api/v2.go | 8 ++++++--",
					"remote_synced":       true,
					"complete":            true,
				},
			},
		},
	}
}

func AuditDashboardRunContract(contract DashboardRunContract) DashboardRunContractAudit {
	audit := DashboardRunContractAudit{
		ContractID:             contract.ContractID,
		Version:                contract.Version,
		DashboardMissingFields: missingFieldDefs(dashboardContractRequiredFields, contract.DashboardSchema.Fields),
		DashboardSampleGaps:    missingSamplePaths(dashboardContractRequiredFields, contract.DashboardSchema.Sample),
		RunDetailMissingFields: missingFieldDefs(runDetailContractRequiredFields, contract.RunDetailSchema.Fields),
		RunDetailSampleGaps:    missingSamplePaths(runDetailContractRequiredFields, contract.RunDetailSchema.Sample),
	}
	audit.ReleaseReady = len(audit.DashboardMissingFields) == 0 && len(audit.DashboardSampleGaps) == 0 && len(audit.RunDetailMissingFields) == 0 && len(audit.RunDetailSampleGaps) == 0
	return audit
}

func RenderDashboardRunContractReport(contract DashboardRunContract, audit DashboardRunContractAudit) string {
	dashboardJSON, _ := json.MarshalIndent(contract.DashboardSchema.Sample, "", "  ")
	runDetailJSON, _ := json.MarshalIndent(contract.RunDetailSchema.Sample, "", "  ")
	lines := []string{
		"# Dashboard and Run Contract",
		"",
		"- Contract ID: " + contract.ContractID,
		"- Version: " + contract.Version,
		"- Release Ready: " + boolText(audit.ReleaseReady),
		"",
		"## Dashboard",
		"- Name: " + contract.DashboardSchema.Name,
		"- Owner: " + contract.DashboardSchema.Owner,
		"- Missing Required Fields: " + fallbackJoin(audit.DashboardMissingFields),
		"- Sample Gaps: " + fallbackJoin(audit.DashboardSampleGaps),
		"",
		"```json",
		string(dashboardJSON),
		"```",
		"",
		"## Run Detail",
		"- Name: " + contract.RunDetailSchema.Name,
		"- Owner: " + contract.RunDetailSchema.Owner,
		"- Missing Required Fields: " + fallbackJoin(audit.RunDetailMissingFields),
		"- Sample Gaps: " + fallbackJoin(audit.RunDetailSampleGaps),
		"",
		"```json",
		string(runDetailJSON),
		"```",
	}
	return strings.Join(lines, "\n") + "\n"
}

func missingFieldDefs(required []string, fields []ContractField) []string {
	defined := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		defined[field.Name] = struct{}{}
	}
	missing := make([]string, 0)
	for _, item := range required {
		if _, ok := defined[item]; !ok {
			missing = append(missing, item)
		}
	}
	return missing
}

func missingSamplePaths(required []string, sample map[string]any) []string {
	missing := make([]string, 0)
	for _, item := range required {
		if !contractPathExists(sample, item) {
			missing = append(missing, item)
		}
	}
	return missing
}

func contractPathExists(payload any, path string) bool {
	current := []any{payload}
	for _, part := range strings.Split(path, ".") {
		next := make([]any, 0)
		isList := strings.HasSuffix(part, "[]")
		key := strings.TrimSuffix(part, "[]")
		for _, item := range current {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			value, ok := object[key]
			if !ok {
				continue
			}
			if isList {
				items, ok := value.([]map[string]any)
				if ok && len(items) > 0 {
					for _, entry := range items {
						next = append(next, entry)
					}
					continue
				}
				generic, ok := value.([]any)
				if ok && len(generic) > 0 {
					next = append(next, generic...)
				}
				continue
			}
			next = append(next, value)
		}
		if len(next) == 0 {
			return false
		}
		current = next
	}
	return true
}

func fallbackJoin(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	copyValues := append([]string(nil), values...)
	sort.Strings(copyValues)
	return strings.Join(copyValues, ", ")
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
