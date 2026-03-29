package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"

	"bigclaw-go/internal/observability"
)

type runDetailEvent struct {
	ID        string   `json:"id"`
	Lane      string   `json:"lane"`
	Title     string   `json:"title"`
	Timestamp string   `json:"timestamp"`
	Status    string   `json:"status"`
	Summary   string   `json:"summary"`
	Details   []string `json:"details"`
}

func RenderTaskRunDetailPage(run observability.TaskRun) string {
	statusTone := "warning"
	if run.Status == "approved" || run.Status == "completed" || run.Status == "succeeded" {
		statusTone = "accent"
	}
	if run.Status == "failed" || run.Status == "rejected" {
		statusTone = "danger"
	}
	actions := workflowConsoleActions(
		run.RunID,
		run.Status == "failed" || run.Status == "needs-approval",
		"retry is available for failed or approval-blocked runs",
		run.Status != "failed" && run.Status != "completed" && run.Status != "approved",
		"completed or failed runs cannot be paused",
		true,
		"",
		true,
		"",
	)
	events := buildRunDetailEvents(run)
	escapedTimelineJSON := marshalTimelineJSON(events)
	artifactsHTML, reportCount := runDetailResources("Artifacts", run.Artifacts, false)
	reportsHTML, _ := runDetailResources("Reports", run.Artifacts, true)
	repoEvidenceHTML := repoEvidenceResources(run)
	collaborationHTML := collaborationPanel(run)
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Task Run Detail · ` + html.EscapeString(run.RunID) + `</title>
</head>
<body>
  <main>
    <section class="shell">
      <header class="hero">
        <span class="eyebrow">Run Detail</span>
        <h1>` + html.EscapeString(run.Title) + `</h1>
        <p>` + html.EscapeString(firstNonEmptyString(run.Summary, "Operational detail page with synced logs, traces, audits, and artifacts.")) + `</p>
        <div class="stats">
          <article class="stat-card"><span>Run ID</span><strong>` + html.EscapeString(run.RunID) + `</strong></article>
          <article class="stat-card"><span>Task ID</span><strong>` + html.EscapeString(run.TaskID) + `</strong></article>
          <article class="stat-card" data-tone="accent"><span>Medium</span><strong>` + html.EscapeString(run.Medium) + `</strong></article>
          <article class="stat-card" data-tone="` + statusTone + `"><span>Status</span><strong>` + html.EscapeString(run.Status) + `</strong></article>
          <article class="stat-card"><span>Artifacts</span><strong>` + fmt.Sprintf("%d", len(run.Artifacts)) + `</strong></article>
          <article class="stat-card"><span>Reports</span><strong>` + fmt.Sprintf("%d", reportCount) + `</strong></article>
          <article class="stat-card"><span>Closeout</span><strong>` + closeoutStatus(run.Closeout) + `</strong></article>
          <article class="stat-card"><span>Repo Links</span><strong>` + fmt.Sprintf("%d", len(run.Closeout.RunCommitLinks)) + `</strong></article>
        </div>
      </header>
      <section class="surface">
        <h2>Overview</h2>
        <p>` + html.EscapeString(firstNonEmptyString(run.Summary, "No summary recorded.")) + `</p>
        <p class="meta">Task ` + html.EscapeString(run.TaskID) + ` from ` + html.EscapeString(run.Source) + ` started at ` + html.EscapeString(run.StartedAt) + ` and ended at ` + html.EscapeString(firstNonEmptyString(run.EndedAt, "n/a")) + `.</p>
      </section>
      <section class="surface">
        <h2>Closeout</h2>
        <p>Validation evidence: ` + html.EscapeString(validationEvidenceText(run.Closeout.ValidationEvidence)) + `</p>
        <p class="meta">git push succeeded=` + html.EscapeString(pyBoolLower(run.Closeout.GitPushSucceeded)) + ` | git log captured=` + html.EscapeString(pyBoolLower(strings.TrimSpace(run.Closeout.GitLogStatOutput) != "")) + ` | complete=` + html.EscapeString(pyBoolLower(run.Closeout.Complete())) + `</p>
        <p class="meta">accepted_commit_hash=` + html.EscapeString(firstNonEmptyString(run.Closeout.AcceptedCommitHash, "none")) + ` | commit_links=` + fmt.Sprintf("%d", len(run.Closeout.RunCommitLinks)) + `</p>
      </section>
      <section class="surface">
        <h2>Actions</h2>
        <p>` + html.EscapeString(RenderConsoleActions(actions)) + `</p>
      </section>
      <section class="surface">
        <h2>Timeline / Log Sync</h2>
        <div class="timeline-shell"><div class="timeline-list">` + timelineButtons(events) + `</div><aside class="surface detail-pane"><h3 data-detail="title">No event selected</h3><span class="meta" data-detail="meta">timeline / idle / n/a</span><p data-detail="summary">Select a timeline item to inspect the synced log, trace, audit, or artifact details.</p><ul class="detail-list" data-detail="list"><li>No additional details.</li></ul></aside></div>
      </section>
      ` + artifactsHTML + `
      ` + reportsHTML + `
      ` + repoEvidenceHTML + `
      ` + collaborationHTML + `
    </section>
  </main>
  <script id="timeline-data" type="application/json">` + escapedTimelineJSON + `</script>
</body>
</html>
`
}

func buildRunDetailEvents(run observability.TaskRun) []runDetailEvent {
	events := make([]runDetailEvent, 0, len(run.Logs)+len(run.Traces)+len(run.Audits)+len(run.Artifacts))
	for i, entry := range run.Logs {
		details := mapDetails(entry.Context, "No structured context recorded.")
		events = append(events, runDetailEvent{ID: fmt.Sprintf("log-%d", i), Lane: "log", Title: entry.Message, Timestamp: entry.Timestamp, Status: entry.Level, Summary: "log entry at " + entry.Timestamp, Details: details})
	}
	for i, entry := range run.Traces {
		details := mapDetails(entry.Attributes, "No trace attributes recorded.")
		events = append(events, runDetailEvent{ID: fmt.Sprintf("trace-%d", i), Lane: "trace", Title: entry.Span, Timestamp: entry.Timestamp, Status: entry.Status, Summary: "trace span " + entry.Span, Details: details})
	}
	for i, entry := range run.Audits {
		details := []string{"actor=" + entry.Actor}
		details = append(details, mapDetails(entry.Details, "")...)
		if len(details) == 0 {
			details = []string{"No audit details recorded."}
		}
		events = append(events, runDetailEvent{ID: fmt.Sprintf("audit-%d", i), Lane: "audit", Title: entry.Action, Timestamp: entry.Timestamp, Status: entry.Outcome, Summary: "audit by " + entry.Actor, Details: details})
	}
	for i, entry := range run.Artifacts {
		details := []string{"path=" + entry.Path, "sha256=" + firstNonEmptyString(entry.SHA256, "n/a")}
		details = append(details, mapDetails(entry.Metadata, "")...)
		events = append(events, runDetailEvent{ID: fmt.Sprintf("artifact-%d", i), Lane: "artifact", Title: entry.Name, Timestamp: entry.Timestamp, Status: entry.Kind, Summary: "artifact emitted at " + entry.Path, Details: details})
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Timestamp < events[j].Timestamp })
	return events
}

func timelineButtons(events []runDetailEvent) string {
	if len(events) == 0 {
		return `<div class="empty">No timeline events recorded.</div>`
	}
	var builder strings.Builder
	for _, event := range events {
		builder.WriteString(`<button class="timeline-item" type="button" data-event-id="` + html.EscapeString(event.ID) + `"><span class="kicker">` + html.EscapeString(event.Lane) + `</span><strong>` + html.EscapeString(event.Title) + `</strong><span class="timeline-meta">` + html.EscapeString(event.Timestamp) + ` | ` + html.EscapeString(event.Status) + `</span><p>` + html.EscapeString(event.Summary) + `</p></button>`)
	}
	return builder.String()
}

func runDetailResources(title string, artifacts []observability.ArtifactRecord, reportsOnly bool) (string, int) {
	var builder strings.Builder
	count := 0
	builder.WriteString(`<section class="surface"><h2>` + html.EscapeString(title) + `</h2>`)
	builder.WriteString(`<div class="resource-grid">`)
	for _, artifact := range artifacts {
		if reportsOnly && artifact.Kind != "report" {
			continue
		}
		if !reportsOnly || artifact.Kind == "report" {
			count++
			builder.WriteString(`<article class="resource-card"><span class="kicker">` + html.EscapeString(artifact.Kind) + `</span><h3>` + html.EscapeString(artifact.Name) + `</h3><p><code>` + html.EscapeString(artifact.Path) + `</code></p></article>`)
		}
	}
	builder.WriteString(`</div></section>`)
	return builder.String(), count
}

func repoEvidenceResources(run observability.TaskRun) string {
	var builder strings.Builder
	builder.WriteString(`<section class="surface"><h2>Repo Evidence</h2><div class="resource-grid">`)
	for _, link := range run.Closeout.RunCommitLinks {
		builder.WriteString(`<article class="resource-card"><span class="kicker">` + html.EscapeString(link.Role) + `</span><h3>` + html.EscapeString(link.CommitHash) + `</h3><p><code>` + html.EscapeString("repo:"+link.RepoSpaceID) + `</code></p></article>`)
	}
	builder.WriteString(`</div></section>`)
	return builder.String()
}

func collaborationPanel(run observability.TaskRun) string {
	audits := make([]map[string]any, 0, len(run.Audits))
	for _, audit := range run.Audits {
		audits = append(audits, map[string]any{"action": audit.Action, "actor": audit.Actor, "outcome": audit.Outcome, "timestamp": audit.Timestamp, "details": audit.Details})
	}
	thread := BuildCollaborationThreadFromAudits(audits, "run", run.RunID)
	lines := RenderCollaborationLines(thread)
	return `<section class="surface"><h2>Collaboration</h2><p>` + html.EscapeString(strings.Join(lines, " ")) + `</p></section>`
}

func mapDetails(values map[string]any, fallback string) []string {
	if len(values) == 0 {
		if fallback == "" {
			return nil
		}
		return []string{fallback}
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, fmt.Sprintf("%s=%v", key, values[key]))
	}
	return out
}

func closeoutStatus(closeout observability.RunCloseout) string {
	if closeout.Complete() {
		return "complete"
	}
	return "pending"
}

func validationEvidenceText(values []string) string {
	if len(values) == 0 {
		return "None recorded."
	}
	return strings.Join(values, ", ")
}

func pyBoolLower(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func marshalTimelineJSON(events []runDetailEvent) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(events)
	return strings.TrimSpace(strings.ReplaceAll(buffer.String(), "</", "<\\/"))
}
