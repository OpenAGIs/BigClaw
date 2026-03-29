package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

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

type RunCloseout struct {
	ValidationEvidence []string             `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool                 `json:"git_push_succeeded"`
	GitPushOutput      string               `json:"git_push_output,omitempty"`
	GitLogStatOutput   string               `json:"git_log_stat_output,omitempty"`
	RepoSyncAudit      *RepoSyncAudit       `json:"repo_sync_audit,omitempty"`
	RunCommitLinks     []repo.RunCommitLink `json:"run_commit_links,omitempty"`
	AcceptedCommitHash string               `json:"accepted_commit_hash,omitempty"`
	Timestamp          string               `json:"timestamp"`
}

func (c RunCloseout) Complete() bool {
	return len(c.ValidationEvidence) > 0 && c.GitPushSucceeded && c.GitLogStatOutput != ""
}

func (c RunCloseout) MarshalJSON() ([]byte, error) {
	type alias RunCloseout
	payload := struct {
		alias
		Complete bool `json:"complete"`
	}{
		alias:    alias(c),
		Complete: c.Complete(),
	}
	return json.Marshal(payload)
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

func NewTaskRunFromTask(task domain.Task, runID string, medium string) TaskRun {
	return TaskRun{
		RunID:     runID,
		TaskID:    task.ID,
		Source:    task.Source,
		Title:     task.Title,
		Medium:    medium,
		StartedAt: nowUTC(),
		Status:    "running",
		Closeout:  RunCloseout{Timestamp: nowUTC()},
	}
}

func (r *TaskRun) Log(level string, message string, context map[string]any) {
	r.Logs = append(r.Logs, LogEntry{
		Level:     level,
		Message:   message,
		Timestamp: nowUTC(),
		Context:   cloneMap(context),
	})
}

func (r *TaskRun) Trace(span string, status string, attributes map[string]any) {
	r.Traces = append(r.Traces, TraceEntry{
		Span:       span,
		Status:     status,
		Timestamp:  nowUTC(),
		Attributes: cloneMap(attributes),
	})
}

func (r *TaskRun) RegisterArtifact(name string, kind string, path string, metadata map[string]any) {
	digest := sha256File(path)
	r.Artifacts = append(r.Artifacts, ArtifactRecord{
		Name:      name,
		Kind:      kind,
		Path:      path,
		Timestamp: nowUTC(),
		SHA256:    digest,
		Metadata:  cloneMap(metadata),
	})
	r.Audit("artifact.registered", "task-run", "recorded", map[string]any{
		"artifact_name": name,
		"artifact_kind": kind,
		"path":          path,
		"sha256":        digest,
	})
}

func (r *TaskRun) Audit(action string, actor string, outcome string, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:    action,
		Actor:     actor,
		Outcome:   outcome,
		Timestamp: nowUTC(),
		Details:   cloneMap(details),
	})
}

func (r *TaskRun) AddComment(author string, body string, mentions []string, anchor string) {
	commentID := r.RunID + "-comment-" + itoa(countAudits(r.Audits, "collaboration.comment")+1)
	r.Audit("collaboration.comment", author, "recorded", map[string]any{
		"surface":    "run",
		"comment_id": commentID,
		"body":       body,
		"mentions":   cloneStrings(mentions),
		"anchor":     anchor,
		"status":     "open",
	})
}

func (r *TaskRun) AddDecisionNote(author string, summary string, outcome string, mentions []string) {
	decisionID := r.RunID + "-decision-" + itoa(countAudits(r.Audits, "collaboration.decision")+1)
	r.Audit("collaboration.decision", author, outcome, map[string]any{
		"surface":     "run",
		"decision_id": decisionID,
		"summary":     summary,
		"mentions":    cloneStrings(mentions),
	})
}

func (r *TaskRun) RecordCloseout(validationEvidence []string, gitPushSucceeded bool, gitPushOutput string, gitLogStatOutput string, repoSyncAudit *RepoSyncAudit, runCommitLinks []repo.RunCommitLink) {
	acceptedCommitHash := ""
	if len(runCommitLinks) > 0 {
		if binding, err := repo.BindRunCommits(runCommitLinks); err == nil {
			acceptedCommitHash = binding.AcceptedCommitHash()
		}
	}
	r.Closeout = RunCloseout{
		ValidationEvidence: cloneStrings(validationEvidence),
		GitPushSucceeded:   gitPushSucceeded,
		GitPushOutput:      gitPushOutput,
		GitLogStatOutput:   gitLogStatOutput,
		RepoSyncAudit:      repoSyncAudit,
		RunCommitLinks:     append([]repo.RunCommitLink(nil), runCommitLinks...),
		AcceptedCommitHash: acceptedCommitHash,
		Timestamp:          nowUTC(),
	}
	r.Audit("closeout.recorded", "task-run", "recorded", map[string]any{
		"validation_evidence_count": len(validationEvidence),
		"git_push_succeeded":        gitPushSucceeded,
		"git_log_stat_captured":     gitLogStatOutput != "",
		"has_repo_sync_audit":       repoSyncAudit != nil,
	})
}

func (r *TaskRun) Finalize(status string, summary string) {
	r.Status = status
	r.Summary = summary
	r.EndedAt = nowUTC()
}

type ObservabilityLedger struct {
	StoragePath string
}

func (l ObservabilityLedger) Load() ([]map[string]any, error) {
	content, err := os.ReadFile(l.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(content, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func (l ObservabilityLedger) LoadRuns() ([]TaskRun, error) {
	entries, err := l.Load()
	if err != nil {
		return nil, err
	}
	runs := make([]TaskRun, 0, len(entries))
	for _, entry := range entries {
		raw, err := json.Marshal(entry)
		if err != nil {
			return nil, err
		}
		var run TaskRun
		if err := json.Unmarshal(raw, &run); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func (l ObservabilityLedger) Append(run TaskRun) error {
	return l.Upsert(run)
}

func (l ObservabilityLedger) Upsert(run TaskRun) error {
	entries, err := l.Load()
	if err != nil {
		return err
	}
	serialized, err := serializeRun(run)
	if err != nil {
		return err
	}
	for i, entry := range entries {
		if runID, _ := entry["run_id"].(string); runID == run.RunID {
			entries[i] = serialized
			return l.writeEntries(entries)
		}
	}
	entries = append(entries, serialized)
	return l.writeEntries(entries)
}

func (l ObservabilityLedger) writeEntries(entries []map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(l.StoragePath), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.StoragePath, content, 0o644)
}

func serializeRun(run TaskRun) (map[string]any, error) {
	content, err := json.Marshal(run)
	if err != nil {
		return nil, err
	}
	var entry map[string]any
	if err := json.Unmarshal(content, &entry); err != nil {
		return nil, err
	}
	return entry, nil
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func sha256File(path string) string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func cloneMap(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	out := make(map[string]any, len(value))
	for key, item := range value {
		out[key] = item
	}
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func countAudits(audits []AuditEntry, action string) int {
	count := 0
	for _, audit := range audits {
		if audit.Action == action {
			count++
		}
	}
	return count
}

func itoa(value int) string {
	return strconv.Itoa(value)
}
