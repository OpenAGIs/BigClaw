package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/billing"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/flow"
	"bigclaw-go/internal/prd"
	"bigclaw-go/internal/product"
	"bigclaw-go/internal/reporting"
)

type flowTemplateRequest struct {
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name"`
	Summary string      `json:"summary,omitempty"`
	Nodes   []flow.Node `json:"nodes"`
}

type flowLaunchRequest struct {
	Actor   string `json:"actor,omitempty"`
	Team    string `json:"team,omitempty"`
	Project string `json:"project,omitempty"`
	Title   string `json:"title,omitempty"`
}

type prdIntakeRequest struct {
	Title              string   `json:"title"`
	Body               string   `json:"body"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
}

func (s *Server) handleV2WeeklyReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, start, end, err := parseWeeklyFilters(r, s.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	report := reporting.Build(s.filteredTasks(team, project, "", start, end), s.Recorder.EventsByTask("", 0), start, end)
	writeJSON(w, http.StatusOK, map[string]any{
		"filters":        map[string]any{"team": team, "project": project, "week_start": start, "week_end": end},
		"summary":        report.Summary,
		"team_breakdown": report.TeamBreakdown,
		"highlights":     report.Highlights,
		"actions":        report.Actions,
		"report":         map[string]any{"markdown": report.Markdown, "export_url": weeklyExportURL(team, project, start, end)},
	})
}

func (s *Server) handleV2WeeklyReportExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, start, end, err := parseWeeklyFilters(r, s.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	report := reporting.Build(s.filteredTasks(team, project, "", start, end), s.Recorder.EventsByTask("", 0), start, end)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fmt.Sprintf("bigclaw-weekly-report-%s.md", start.Format("2006-01-02"))))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(report.Markdown))
}

func (s *Server) handleV2FlowTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"templates": s.FlowStore.List()})
	case http.MethodPost:
		var request flowTemplateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, fmt.Sprintf("decode template: %v", err), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(request.Name) == "" || len(request.Nodes) == 0 {
			http.Error(w, "missing name or nodes", http.StatusBadRequest)
			return
		}
		template := s.FlowStore.Save(flow.Template{ID: request.ID, Name: request.Name, Summary: request.Summary, Nodes: request.Nodes}, s.Now())
		writeJSON(w, http.StatusOK, map[string]any{"template": template})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleV2FlowTemplateAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/v2/flows/templates/"), "/")
	if !strings.HasSuffix(path, "/launch") {
		http.Error(w, "unsupported action", http.StatusNotFound)
		return
	}
	templateID := strings.TrimSuffix(path, "/launch")
	templateID = strings.TrimSuffix(templateID, "/")
	template, ok := s.FlowStore.Get(templateID)
	if !ok {
		http.Error(w, "template not found", http.StatusNotFound)
		return
	}
	var request flowLaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode launch request: %v", err), http.StatusBadRequest)
		return
	}
	actor := strings.TrimSpace(request.Actor)
	if actor == "" {
		actor = "system"
	}
	flowID := fmt.Sprintf("flow-%d", s.Now().UnixNano())
	tasks := flow.LaunchTasks(template, flowID, request.Title, request.Team, request.Project, actor, s.Now())
	for _, task := range tasks {
		if err := s.Queue.Enqueue(r.Context(), task); err != nil {
			http.Error(w, fmt.Sprintf("launch flow task: %v", err), http.StatusInternalServerError)
			return
		}
		s.Recorder.StoreTask(task)
		s.publish(domain.Event{ID: task.ID + "-queued", Type: domain.EventTaskQueued, TaskID: task.ID, TraceID: task.TraceID, Timestamp: s.Now(), Payload: map[string]any{"actor": actor, "flow_id": flowID, "template_id": template.ID, "department": task.Metadata["department"]}})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"flow_id": flowID, "template": template, "launched_tasks": tasks})
}

func (s *Server) handleV2FlowOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	flowID := strings.TrimSpace(r.URL.Query().Get("flow_id"))
	tasks := s.filteredTasks(team, project, "", time.Time{}, time.Time{})
	if flowID != "" {
		filtered := make([]domain.Task, 0)
		for _, task := range tasks {
			if strings.EqualFold(strings.TrimSpace(task.Metadata["flow_id"]), flowID) {
				filtered = append(filtered, task)
			}
		}
		tasks = filtered
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": flow.BuildOverview(tasks)})
}

func (s *Server) handleV2PRDIntake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var request prdIntakeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("decode prd intake: %v", err), http.StatusBadRequest)
		return
	}
	result := prd.Build(request.Title, request.Body, request.AcceptanceCriteria, s.Now())
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleV2LaunchChecklist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flowID := strings.TrimSpace(r.URL.Query().Get("flow_id"))
	if flowID == "" {
		http.Error(w, "missing flow_id", http.StatusBadRequest)
		return
	}
	checklist := flow.BuildLaunchChecklist(s.Recorder.Tasks(0), flowID)
	writeJSON(w, http.StatusOK, checklist)
}

func (s *Server) handleV2SupportHandoff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flowID := strings.TrimSpace(r.URL.Query().Get("flow_id"))
	if flowID == "" {
		http.Error(w, "missing flow_id", http.StatusBadRequest)
		return
	}
	handoff := flow.BuildSupportHandoff(s.Recorder.Tasks(0), flowID)
	writeJSON(w, http.StatusOK, handoff)
}

func (s *Server) handleV2Navigation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := normalizeHomeRole(r)
	writeJSON(w, http.StatusOK, map[string]any{"role": role, "sections": product.Navigation()})
}

func (s *Server) handleV2Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	role := normalizeHomeRole(r)
	home := product.HomeForRole(role, s.Recorder.Tasks(0))
	writeJSON(w, http.StatusOK, map[string]any{"home": home, "summary": product.SummaryText(home)})
}

func (s *Server) handleV2DesignSystem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, product.DefaultDesignSystem())
}

func (s *Server) handleV2BillingUsage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	organization := strings.TrimSpace(r.URL.Query().Get("organization"))
	tier := strings.TrimSpace(r.URL.Query().Get("tier"))
	tasks := s.filteredTasks(team, project, "", time.Time{}, time.Time{})
	writeJSON(w, http.StatusOK, billing.BuildUsage(tasks, organization, tier))
}

func (s *Server) handleV2BillingEntitlements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, billing.EntitlementsForTier(r.URL.Query().Get("tier")))
}

func parseWeeklyFilters(r *http.Request, now time.Time) (string, string, time.Time, time.Time, error) {
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	start, err := parseOptionalTime(r.URL.Query().Get("week_start"))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid week_start value, expected RFC3339")
	}
	end, err := parseOptionalTime(r.URL.Query().Get("week_end"))
	if err != nil {
		return "", "", time.Time{}, time.Time{}, fmt.Errorf("invalid week_end value, expected RFC3339")
	}
	if start.IsZero() {
		endOfWindow := now.UTC()
		start = endOfWindow.Add(-7 * 24 * time.Hour)
		if end.IsZero() {
			end = endOfWindow
		}
	}
	if end.IsZero() {
		end = start.Add(7*24*time.Hour - time.Second)
	}
	return team, project, start, end, nil
}

func weeklyExportURL(team string, project string, start time.Time, end time.Time) string {
	parts := make([]string, 0)
	if team != "" {
		parts = append(parts, "team="+team)
	}
	if project != "" {
		parts = append(parts, "project="+project)
	}
	parts = append(parts, "week_start="+start.Format(time.RFC3339))
	parts = append(parts, "week_end="+end.Format(time.RFC3339))
	return "/v2/reports/weekly/export?" + strings.Join(parts, "&")
}

func normalizeHomeRole(r *http.Request) string {
	if role := strings.TrimSpace(r.URL.Query().Get("role")); role != "" {
		return strings.ToLower(role)
	}
	authorization := parseControlAuthorization(r, "", "", "")
	return strings.ToLower(string(authorization.Role))
}

func renderJSONBody(body any) *bytes.Reader {
	payload, _ := json.Marshal(body)
	return bytes.NewReader(payload)
}

func collectFlowTasks(ctx context.Context, qTasks []domain.Task) []domain.Task {
	out := append([]domain.Task(nil), qTasks...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	_ = ctx
	return out
}
