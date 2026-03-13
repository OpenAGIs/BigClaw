package prd

import (
	"fmt"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/flow"
)

type Intake struct {
	Title             string        `json:"title"`
	Summary           string        `json:"summary"`
	SuggestedTasks    []domain.Task `json:"suggested_tasks"`
	SuggestedTemplate flow.Template `json:"suggested_template"`
	Signals           []string      `json:"signals,omitempty"`
}

func Build(title string, body string, acceptanceCriteria []string, now time.Time) Intake {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "PRD Intake"
	}
	summary := summarize(body)
	signals := detectSignals(body)
	tasks := []domain.Task{
		makeTask(title, "engineering", "Implement product scope and core workflows", acceptanceCriteria, now),
		makeTask(title, "docs", "Draft operator docs and rollout notes", acceptanceCriteria, now),
		makeTask(title, "release", "Prepare launch checklist and release gate", acceptanceCriteria, now),
		makeTask(title, "support", "Prepare support handoff package and FAQ draft", acceptanceCriteria, now),
	}
	template := flow.Template{
		ID:      strings.ToLower(strings.ReplaceAll(title, " ", "-")),
		Name:    title,
		Summary: summary,
		Nodes: []flow.Node{
			{ID: "engineering", Name: "Engineering implementation", Department: "engineering", Validation: "ship core implementation", Approval: "eng_lead", Kind: "engineering"},
			{ID: "docs", Name: "Documentation handoff", Department: "docs", Validation: "publish docs package", Approval: "docs_owner", Kind: "docs", DependsOn: []string{"engineering"}},
			{ID: "release", Name: "Launch checklist", Department: "release", Validation: "complete launch checklist", Approval: "release_manager", Kind: "release", DependsOn: []string{"engineering", "docs"}},
			{ID: "support", Name: "Support handoff", Department: "support", Validation: "prepare support packet", Approval: "support_lead", Kind: "support", DependsOn: []string{"release"}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return Intake{Title: title, Summary: summary, SuggestedTasks: tasks, SuggestedTemplate: template, Signals: signals}
}

func makeTask(title string, department string, detail string, acceptanceCriteria []string, now time.Time) domain.Task {
	identifier := strings.ToLower(strings.ReplaceAll(department, " ", "-"))
	return domain.Task{
		ID:                 fmt.Sprintf("prd-%s-%d", identifier, now.UnixNano()),
		TraceID:            fmt.Sprintf("prd-%s-%d", identifier, now.UnixNano()),
		Title:              fmt.Sprintf("%s / %s", title, strings.Title(department)),
		AcceptanceCriteria: append([]string(nil), acceptanceCriteria...),
		ValidationPlan:     []string{detail},
		Metadata: map[string]string{
			"department": department,
			"workflow":   department,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func summarize(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return "PRD intake generated a default cross-functional delivery template."
	}
	lines := strings.Fields(body)
	if len(lines) > 24 {
		lines = lines[:24]
	}
	return strings.Join(lines, " ")
}

func detectSignals(body string) []string {
	body = strings.ToLower(body)
	out := make([]string, 0)
	for _, signal := range []string{"launch", "support", "billing", "approval", "documentation", "release"} {
		if strings.Contains(body, signal) {
			out = append(out, signal)
		}
	}
	if len(out) == 0 {
		out = append(out, "default")
	}
	return out
}
