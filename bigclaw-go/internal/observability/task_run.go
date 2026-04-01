package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

type RunCommitLink = repo.RunCommitLink

type LogEntry struct {
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Timestamp string         `json:"timestamp"`
	Context   map[string]any `json:"context,omitempty"`
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
	SHA256    string         `json:"sha256,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AuditEntry struct {
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Outcome   string         `json:"outcome"`
	Timestamp string         `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

type GitSyncTelemetry struct {
	Status          string   `json:"status"`
	FailureCategory string   `json:"failure_category,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	Remote          string   `json:"remote,omitempty"`
	RemoteRef       string   `json:"remote_ref,omitempty"`
	AheadBy         int      `json:"ahead_by,omitempty"`
	BehindBy        int      `json:"behind_by,omitempty"`
	DirtyPaths      []string `json:"dirty_paths,omitempty"`
	AuthTarget      string   `json:"auth_target,omitempty"`
	Timestamp       string   `json:"timestamp,omitempty"`
}

type PullRequestFreshness struct {
	PRNumber           *int   `json:"pr_number,omitempty"`
	PRURL              string `json:"pr_url,omitempty"`
	BranchState        string `json:"branch_state"`
	BodyState          string `json:"body_state"`
	BranchHeadSHA      string `json:"branch_head_sha,omitempty"`
	PRHeadSHA          string `json:"pr_head_sha,omitempty"`
	ExpectedBodyDigest string `json:"expected_body_digest,omitempty"`
	ActualBodyDigest   string `json:"actual_body_digest,omitempty"`
	CheckedAt          string `json:"checked_at,omitempty"`
}

type RepoSyncAudit struct {
	Sync        GitSyncTelemetry     `json:"sync"`
	PullRequest PullRequestFreshness `json:"pull_request"`
}

func (audit RepoSyncAudit) Summary() string {
	parts := []string{fmt.Sprintf("sync=%s", audit.Sync.Status)}
	if strings.TrimSpace(audit.Sync.FailureCategory) != "" {
		parts = append(parts, fmt.Sprintf("failure=%s", audit.Sync.FailureCategory))
	}
	parts = append(parts, fmt.Sprintf("pr-branch=%s", audit.PullRequest.BranchState))
	parts = append(parts, fmt.Sprintf("pr-body=%s", audit.PullRequest.BodyState))
	return strings.Join(parts, ", ")
}

type RunCloseout struct {
	ValidationEvidence []string        `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool            `json:"git_push_succeeded"`
	GitPushOutput      string          `json:"git_push_output,omitempty"`
	GitLogStatOutput   string          `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *RepoSyncAudit  `json:"repo_sync_audit,omitempty"`
	RunCommitLinks     []RunCommitLink `json:"run_commit_links,omitempty"`
	AcceptedCommitHash string          `json:"accepted_commit_hash,omitempty"`
	Timestamp          string          `json:"timestamp"`
	Complete           bool            `json:"complete"`
}

type TaskRun struct {
	RunID     string           `json:"run_id"`
	TaskID    string           `json:"task_id"`
	Source    string           `json:"source"`
	Title     string           `json:"title"`
	Medium    string           `json:"medium"`
	StartedAt string           `json:"started_at"`
	EndedAt   string           `json:"ended_at,omitempty"`
	Status    string           `json:"status"`
	Summary   string           `json:"summary,omitempty"`
	Logs      []LogEntry       `json:"logs,omitempty"`
	Traces    []TraceEntry     `json:"traces,omitempty"`
	Artifacts []ArtifactRecord `json:"artifacts,omitempty"`
	Audits    []AuditEntry     `json:"audits,omitempty"`
	Closeout  RunCloseout      `json:"closeout"`
}

func NewTaskRun(task domain.Task, runID, medium string) TaskRun {
	return TaskRun{
		RunID:     runID,
		TaskID:    task.ID,
		Source:    task.Source,
		Title:     task.Title,
		Medium:    medium,
		StartedAt: nowRFC3339(),
		Status:    "running",
		Closeout:  RunCloseout{Timestamp: nowRFC3339()},
	}
}

func (run *TaskRun) Log(level, message string, context map[string]any) {
	run.Logs = append(run.Logs, LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: nowRFC3339(),
		Context:   cloneAnyMap(context),
	})
}

func (run *TaskRun) Trace(span, status string, attributes map[string]any) {
	run.Traces = append(run.Traces, TraceEntry{
		Span:       span,
		Status:     status,
		Timestamp:  nowRFC3339(),
		Attributes: cloneAnyMap(attributes),
	})
}

func (run *TaskRun) RegisterArtifact(name, kind, path string, metadata map[string]any) {
	digest := sha256File(path)
	run.Artifacts = append(run.Artifacts, ArtifactRecord{
		Name:      name,
		Kind:      kind,
		Path:      path,
		Timestamp: nowRFC3339(),
		SHA256:    digest,
		Metadata:  cloneAnyMap(metadata),
	})
	run.Audit("artifact.registered", "task-run", "recorded", map[string]any{
		"artifact_name": name,
		"artifact_kind": kind,
		"path":          path,
		"sha256":        digest,
	})
}

func (run *TaskRun) Audit(action, actor, outcome string, details map[string]any) {
	run.Audits = append(run.Audits, AuditEntry{
		Action:    action,
		Actor:     actor,
		Outcome:   outcome,
		Timestamp: nowRFC3339(),
		Details:   cloneAnyMap(details),
	})
}

func (run *TaskRun) AddComment(author, body string, mentions []string, anchor string) string {
	count := 0
	for _, audit := range run.Audits {
		if audit.Action == "collaboration.comment" {
			count++
		}
	}
	commentID := fmt.Sprintf("%s-comment-%d", run.RunID, count+1)
	run.Audit("collaboration.comment", author, "recorded", map[string]any{
		"surface":    "run",
		"comment_id": commentID,
		"body":       body,
		"mentions":   append([]string(nil), mentions...),
		"anchor":     anchor,
		"status":     "open",
	})
	return commentID
}

func (run *TaskRun) AddDecisionNote(author, summary, outcome string, mentions, relatedCommentIDs []string, followUp string) string {
	count := 0
	for _, audit := range run.Audits {
		if audit.Action == "collaboration.decision" {
			count++
		}
	}
	decisionID := fmt.Sprintf("%s-decision-%d", run.RunID, count+1)
	run.Audit("collaboration.decision", author, outcome, map[string]any{
		"surface":             "run",
		"decision_id":         decisionID,
		"summary":             summary,
		"mentions":            append([]string(nil), mentions...),
		"related_comment_ids": append([]string(nil), relatedCommentIDs...),
		"follow_up":           followUp,
	})
	return decisionID
}

func (run *TaskRun) RecordCloseout(validationEvidence []string, gitPushSucceeded bool, gitPushOutput, gitLogStatOutput string, repoSyncAudit *RepoSyncAudit, runCommitLinks []RunCommitLink) error {
	acceptedCommitHash := ""
	if len(runCommitLinks) > 0 {
		binding, err := repo.BindRunCommits(runCommitLinks)
		if err != nil {
			return err
		}
		acceptedCommitHash = binding.AcceptedCommitHash()
	}
	closeout := RunCloseout{
		ValidationEvidence: append([]string(nil), validationEvidence...),
		GitPushSucceeded:   gitPushSucceeded,
		GitPushOutput:      gitPushOutput,
		GitLogStatOutput:   gitLogStatOutput,
		RepoSyncAudit:      repoSyncAudit,
		RunCommitLinks:     append([]RunCommitLink(nil), runCommitLinks...),
		AcceptedCommitHash: acceptedCommitHash,
		Timestamp:          nowRFC3339(),
	}
	closeout.Complete = len(closeout.ValidationEvidence) > 0 && closeout.GitPushSucceeded && strings.TrimSpace(closeout.GitLogStatOutput) != ""
	run.Closeout = closeout
	run.Audit("closeout.recorded", "task-run", "recorded", map[string]any{
		"validation_evidence_count": len(validationEvidence),
		"git_push_succeeded":        gitPushSucceeded,
		"git_log_stat_captured":     strings.TrimSpace(gitLogStatOutput) != "",
		"has_repo_sync_audit":       repoSyncAudit != nil,
	})
	return nil
}

func (run *TaskRun) Finalize(status, summary string) {
	run.Status = status
	run.Summary = summary
	run.EndedAt = nowRFC3339()
}

type ObservabilityLedger struct {
	StoragePath string
}

func (ledger ObservabilityLedger) Load() ([]map[string]any, error) {
	body, err := os.ReadFile(ledger.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (ledger ObservabilityLedger) Append(run TaskRun) error {
	return ledger.Upsert(run)
}

func (ledger ObservabilityLedger) Upsert(run TaskRun) error {
	entries, err := ledger.Load()
	if err != nil {
		return err
	}
	payload, err := marshalToMap(run)
	if err != nil {
		return err
	}
	for index, entry := range entries {
		if stringValue(entry["run_id"]) == run.RunID {
			entries[index] = payload
			return ledger.write(entries)
		}
	}
	entries = append(entries, payload)
	return ledger.write(entries)
}

func (ledger ObservabilityLedger) LoadRuns() ([]TaskRun, error) {
	entries, err := ledger.Load()
	if err != nil {
		return nil, err
	}
	runs := make([]TaskRun, 0, len(entries))
	for _, entry := range entries {
		body, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		var run TaskRun
		if err := json.Unmarshal(body, &run); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (ledger ObservabilityLedger) write(entries []map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(ledger.StoragePath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ledger.StoragePath, body, 0o644)
}

type CollaborationComment struct {
	CommentID string
	Author    string
	Body      string
	CreatedAt string
	Mentions  []string
	Anchor    string
	Status    string
}

type DecisionNote struct {
	DecisionID        string
	Author            string
	Outcome           string
	Summary           string
	RecordedAt        string
	Mentions          []string
	RelatedCommentIDs []string
	FollowUp          string
}

type CollaborationThread struct {
	Surface   string
	TargetID  string
	Comments  []CollaborationComment
	Decisions []DecisionNote
}

func (thread CollaborationThread) ParticipantCount() int {
	participants := map[string]struct{}{}
	for _, comment := range thread.Comments {
		if strings.TrimSpace(comment.Author) != "" {
			participants[comment.Author] = struct{}{}
		}
	}
	for _, decision := range thread.Decisions {
		if strings.TrimSpace(decision.Author) != "" {
			participants[decision.Author] = struct{}{}
		}
	}
	return len(participants)
}

func (thread CollaborationThread) MentionCount() int {
	total := 0
	for _, comment := range thread.Comments {
		total += len(comment.Mentions)
	}
	for _, decision := range thread.Decisions {
		total += len(decision.Mentions)
	}
	return total
}

func (thread CollaborationThread) OpenCommentCount() int {
	total := 0
	for _, comment := range thread.Comments {
		if comment.Status != "resolved" {
			total++
		}
	}
	return total
}

func (thread CollaborationThread) Recommendation() string {
	switch {
	case len(thread.Decisions) > 0:
		return "share-latest-decision"
	case thread.OpenCommentCount() > 0:
		return "resolve-open-comments"
	case len(thread.Comments) > 0:
		return "monitor-collaboration"
	default:
		return "no-collaboration-recorded"
	}
}

func BuildCollaborationThreadFromAudits(audits []AuditEntry, surface, targetID string) *CollaborationThread {
	comments := make([]CollaborationComment, 0)
	decisions := make([]DecisionNote, 0)
	for _, audit := range audits {
		if stringValue(audit.Details["surface"]) != surface {
			continue
		}
		switch audit.Action {
		case "collaboration.comment":
			comments = append(comments, CollaborationComment{
				CommentID: stringValue(audit.Details["comment_id"]),
				Author:    audit.Actor,
				Body:      stringValue(audit.Details["body"]),
				CreatedAt: audit.Timestamp,
				Mentions:  stringSliceValue(audit.Details["mentions"]),
				Anchor:    stringValue(audit.Details["anchor"]),
				Status:    firstNonEmpty(stringValue(audit.Details["status"]), "open"),
			})
		case "collaboration.decision":
			decisions = append(decisions, DecisionNote{
				DecisionID:        stringValue(audit.Details["decision_id"]),
				Author:            audit.Actor,
				Outcome:           audit.Outcome,
				Summary:           stringValue(audit.Details["summary"]),
				RecordedAt:        audit.Timestamp,
				Mentions:          stringSliceValue(audit.Details["mentions"]),
				RelatedCommentIDs: stringSliceValue(audit.Details["related_comment_ids"]),
				FollowUp:          stringValue(audit.Details["follow_up"]),
			})
		}
	}
	if len(comments) == 0 && len(decisions) == 0 {
		return nil
	}
	sort.SliceStable(comments, func(i, j int) bool { return comments[i].CreatedAt < comments[j].CreatedAt })
	sort.SliceStable(decisions, func(i, j int) bool { return decisions[i].RecordedAt < decisions[j].RecordedAt })
	return &CollaborationThread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  comments,
		Decisions: decisions,
	}
}

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	prNumber := "unknown"
	if audit.PullRequest.PRNumber != nil {
		prNumber = fmt.Sprintf("%d", *audit.PullRequest.PRNumber)
	}
	lines := []string{
		"# Repo Sync Audit",
		"",
		"## Sync Status",
		"",
		fmt.Sprintf("- Status: %s", audit.Sync.Status),
		fmt.Sprintf("- Failure Category: %s", firstNonEmpty(audit.Sync.FailureCategory, "none")),
		fmt.Sprintf("- Summary: %s", firstNonEmpty(audit.Sync.Summary, "none")),
		fmt.Sprintf("- Branch: %s", firstNonEmpty(audit.Sync.Branch, "unknown")),
		fmt.Sprintf("- Remote: %s", firstNonEmpty(audit.Sync.Remote, "origin")),
		fmt.Sprintf("- Remote Ref: %s", firstNonEmpty(audit.Sync.RemoteRef, "unknown")),
		fmt.Sprintf("- Ahead By: %d", audit.Sync.AheadBy),
		fmt.Sprintf("- Behind By: %d", audit.Sync.BehindBy),
		fmt.Sprintf("- Dirty Paths: %s", joinOrNone(audit.Sync.DirtyPaths)),
		fmt.Sprintf("- Auth Target: %s", firstNonEmpty(audit.Sync.AuthTarget, "none")),
		fmt.Sprintf("- Checked At: %s", audit.Sync.Timestamp),
		"",
		"## Pull Request Freshness",
		"",
		fmt.Sprintf("- PR Number: %s", prNumber),
		fmt.Sprintf("- PR URL: %s", firstNonEmpty(audit.PullRequest.PRURL, "none")),
		fmt.Sprintf("- Branch State: %s", audit.PullRequest.BranchState),
		fmt.Sprintf("- Body State: %s", audit.PullRequest.BodyState),
		fmt.Sprintf("- Branch Head SHA: %s", firstNonEmpty(audit.PullRequest.BranchHeadSHA, "unknown")),
		fmt.Sprintf("- PR Head SHA: %s", firstNonEmpty(audit.PullRequest.PRHeadSHA, "unknown")),
		fmt.Sprintf("- Expected Body Digest: %s", firstNonEmpty(audit.PullRequest.ExpectedBodyDigest, "unknown")),
		fmt.Sprintf("- Actual Body Digest: %s", firstNonEmpty(audit.PullRequest.ActualBodyDigest, "unknown")),
		fmt.Sprintf("- Checked At: %s", audit.PullRequest.CheckedAt),
		"",
		"## Summary",
		"",
		fmt.Sprintf("- %s", audit.Summary()),
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderTaskRunReport(run TaskRun) string {
	thread := BuildCollaborationThreadFromAudits(run.Audits, "run", run.RunID)
	lines := []string{
		"# Task Run Report",
		"",
		fmt.Sprintf("- Run ID: %s", run.RunID),
		fmt.Sprintf("- Task ID: %s", run.TaskID),
		fmt.Sprintf("- Source: %s", run.Source),
		fmt.Sprintf("- Medium: %s", run.Medium),
		fmt.Sprintf("- Status: %s", run.Status),
		fmt.Sprintf("- Started At: %s", run.StartedAt),
		fmt.Sprintf("- Ended At: %s", firstNonEmpty(run.EndedAt, "n/a")),
		"",
		"## Summary",
		"",
		firstNonEmpty(run.Summary, "No summary recorded."),
		"",
		"## Logs",
		"",
	}
	if len(run.Logs) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range run.Logs {
			lines = append(lines, fmt.Sprintf("- [%s] %s %s", entry.Level, entry.Timestamp, entry.Message))
		}
	}
	lines = append(lines, "", "## Trace", "")
	if len(run.Traces) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range run.Traces {
			lines = append(lines, fmt.Sprintf("- %s: %s @ %s", entry.Span, entry.Status, entry.Timestamp))
		}
	}
	lines = append(lines, "", "## Artifacts", "")
	if len(run.Artifacts) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range run.Artifacts {
			lines = append(lines, fmt.Sprintf("- %s (%s): %s", entry.Name, entry.Kind, entry.Path))
		}
	}
	lines = append(lines, "", "## Audit", "")
	if len(run.Audits) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, entry := range run.Audits {
			lines = append(lines, fmt.Sprintf("- %s by %s: %s", entry.Action, entry.Actor, entry.Outcome))
		}
	}
	lines = append(lines,
		"",
		"## Closeout",
		"",
		fmt.Sprintf("- Complete: %t", run.Closeout.Complete),
		fmt.Sprintf("- Validation Evidence: %s", firstNonEmpty(strings.Join(run.Closeout.ValidationEvidence, ", "), "None")),
		fmt.Sprintf("- Git Push Succeeded: %s", titleBool(run.Closeout.GitPushSucceeded)),
		fmt.Sprintf("- Git Push Output: %s", firstNonEmpty(run.Closeout.GitPushOutput, "None")),
		fmt.Sprintf("- Git Log -1 --stat Output: %s", firstNonEmpty(run.Closeout.GitLogStatOutput, "None")),
	)
	if run.Closeout.RepoSyncAudit != nil {
		lines = append(lines,
			fmt.Sprintf("- Repo Sync Status: %s", run.Closeout.RepoSyncAudit.Sync.Status),
			fmt.Sprintf("- Repo Sync Failure Category: %s", firstNonEmpty(run.Closeout.RepoSyncAudit.Sync.FailureCategory, "none")),
			fmt.Sprintf("- PR Branch State: %s", run.Closeout.RepoSyncAudit.PullRequest.BranchState),
			fmt.Sprintf("- PR Body State: %s", run.Closeout.RepoSyncAudit.PullRequest.BodyState),
		)
	}
	lines = append(lines, "", "## Actions", "", fmt.Sprintf("- %s", renderConsoleActions(consoleActions(run)...)))
	lines = append(lines, renderCollaborationLines(thread)...)
	return strings.Join(lines, "\n") + "\n"
}

func RenderTaskRunDetailPage(run TaskRun) string {
	thread := BuildCollaborationThreadFromAudits(run.Audits, "run", run.RunID)
	type timelineEvent struct {
		Title string `json:"title"`
	}
	timeline := make([]timelineEvent, 0, len(run.Logs)+len(run.Traces))
	for _, entry := range run.Logs {
		timeline = append(timeline, timelineEvent{Title: entry.Message})
	}
	for _, entry := range run.Traces {
		timeline = append(timeline, timelineEvent{Title: entry.Span})
	}
	timelineJSON, _ := json.Marshal(timeline)
	escapedTimelineJSON := string(timelineJSON)
	escapedTimelineJSON = strings.ReplaceAll(escapedTimelineJSON, `\u003c`, "<")
	escapedTimelineJSON = strings.ReplaceAll(escapedTimelineJSON, `\u003e`, ">")
	escapedTimelineJSON = strings.ReplaceAll(escapedTimelineJSON, "</", "<\\/")
	var artifactsBuilder strings.Builder
	for _, artifact := range run.Artifacts {
		artifactsBuilder.WriteString(fmt.Sprintf("<li>%s (%s) %s</li>", html.EscapeString(artifact.Name), html.EscapeString(artifact.Kind), html.EscapeString(artifact.Path)))
	}
	var repoLinksBuilder strings.Builder
	for _, link := range run.Closeout.RunCommitLinks {
		repoLinksBuilder.WriteString(fmt.Sprintf("<li>%s (%s)</li>", html.EscapeString(link.CommitHash), html.EscapeString(link.Role)))
	}
	if repoLinksBuilder.Len() == 0 {
		repoLinksBuilder.WriteString("<li>none</li>")
	}
	if artifactsBuilder.Len() == 0 {
		artifactsBuilder.WriteString("<li>none</li>")
	}
	var collabBuilder strings.Builder
	if thread != nil {
		for _, comment := range thread.Comments {
			collabBuilder.WriteString(fmt.Sprintf("<li>%s</li>", html.EscapeString(comment.Body)))
		}
		for _, decision := range thread.Decisions {
			collabBuilder.WriteString(fmt.Sprintf("<li>%s</li>", html.EscapeString(decision.Summary)))
		}
	} else {
		collabBuilder.WriteString("<li>none</li>")
	}
	page := fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Task Run Detail · %s</title>
</head>
<body>
  <main>
    <section>
      <h1 data-detail="title">%s</h1>
      <p>%s</p>
    </section>
    <section>
      <h2>Timeline / Log Sync</h2>
      <ul>`, html.EscapeString(run.RunID), html.EscapeString(run.Title), html.EscapeString(firstNonEmpty(run.Summary, "Operational detail page with synced logs, traces, audits, and artifacts.")))
	for _, entry := range run.Logs {
		page += fmt.Sprintf("<li>%s</li>", html.EscapeString(entry.Message))
	}
	for _, entry := range run.Traces {
		page += fmt.Sprintf("<li>%s</li>", html.EscapeString(entry.Span))
	}
	page += `</ul></section>
    <section><h2>Reports</h2><ul>` + artifactsBuilder.String() + `</ul></section>
    <section><h2>Closeout</h2><p>complete=` + html.EscapeString(fmt.Sprintf("%t", run.Closeout.Complete)) + `</p><p>` + html.EscapeString(run.Summary) + `</p></section>
    <section><h2>Repo Evidence</h2><p>` + html.EscapeString(firstNonEmpty(run.Closeout.AcceptedCommitHash, "none")) + `</p><ul>` + repoLinksBuilder.String() + `</ul></section>
    <section><h2>Actions</h2><p>` + html.EscapeString(renderConsoleActions(consoleActions(run)...)) + `</p></section>
    <section><h2>Collaboration</h2><ul>` + collabBuilder.String() + `</ul></section>
    <script>const timeline=` + escapedTimelineJSON + `;</script>
  </main>
</body>
</html>`
	return page
}

type consoleAction struct {
	ActionID string
	Label    string
	Target   string
	Enabled  bool
	Reason   string
}

func consoleActions(run TaskRun) []consoleAction {
	return []consoleAction{
		{
			ActionID: "retry",
			Label:    "Retry",
			Target:   run.RunID,
			Enabled:  run.Status == "failed" || run.Status == "needs-approval",
			Reason:   ternaryString(run.Status == "failed" || run.Status == "needs-approval", "", "retry is available for failed or approval-blocked runs"),
		},
		{
			ActionID: "pause",
			Label:    "Pause",
			Target:   run.RunID,
			Enabled:  run.Status != "failed" && run.Status != "completed" && run.Status != "approved",
			Reason:   ternaryString(run.Status != "failed" && run.Status != "completed" && run.Status != "approved", "", "completed or failed runs cannot be paused"),
		},
	}
}

func renderConsoleActions(actions ...consoleAction) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		state := "disabled"
		if action.Enabled {
			state = "enabled"
		}
		part := fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ActionID, state, action.Target)
		if strings.TrimSpace(action.Reason) != "" {
			part += " reason=" + action.Reason
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | ")
}

func renderCollaborationLines(thread *CollaborationThread) []string {
	if thread == nil {
		return nil
	}
	lines := []string{
		"",
		"## Collaboration",
		"",
		fmt.Sprintf("- Surface: %s", thread.Surface),
		fmt.Sprintf("- Target: %s", thread.TargetID),
		fmt.Sprintf("- Participants: %d", thread.ParticipantCount()),
		fmt.Sprintf("- Comments: %d", len(thread.Comments)),
		fmt.Sprintf("- Open Comments: %d", thread.OpenCommentCount()),
		fmt.Sprintf("- Mentions: %d", thread.MentionCount()),
		fmt.Sprintf("- Decision Notes: %d", len(thread.Decisions)),
		fmt.Sprintf("- Recommendation: %s", thread.Recommendation()),
		"",
		"## Comments",
		"",
	}
	if len(thread.Comments) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, comment := range thread.Comments {
			lines = append(lines, fmt.Sprintf("- %s: author=%s status=%s anchor=%s mentions=%s body=%s", comment.CommentID, comment.Author, comment.Status, firstNonEmpty(comment.Anchor, "none"), joinOrNone(comment.Mentions), comment.Body))
		}
	}
	lines = append(lines, "", "## Decision Notes", "")
	if len(thread.Decisions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, decision := range thread.Decisions {
			lines = append(lines, fmt.Sprintf("- %s: author=%s outcome=%s mentions=%s related=%s summary=%s", decision.DecisionID, decision.Author, decision.Outcome, joinOrNone(decision.Mentions), joinOrNone(decision.RelatedCommentIDs), decision.Summary))
		}
	}
	return lines
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func sha256File(path string) string {
	body, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func marshalToMap(value any) (map[string]any, error) {
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, stringValue(item))
		}
		return out
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func titleBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func ternaryString(condition bool, yes, no string) string {
	if condition {
		return yes
	}
	return no
}
