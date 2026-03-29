package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/repo"
)

type GitSyncTelemetry struct {
	Status          string    `json:"status"`
	FailureCategory string    `json:"failure_category,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	Branch          string    `json:"branch,omitempty"`
	Remote          string    `json:"remote,omitempty"`
	RemoteRef       string    `json:"remote_ref,omitempty"`
	AheadBy         int       `json:"ahead_by,omitempty"`
	BehindBy        int       `json:"behind_by,omitempty"`
	DirtyPaths      []string  `json:"dirty_paths,omitempty"`
	AuthTarget      string    `json:"auth_target,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

func (g GitSyncTelemetry) OK() bool {
	return g.Status == "synced"
}

type PullRequestFreshness struct {
	PRNumber           *int      `json:"pr_number,omitempty"`
	PRURL              string    `json:"pr_url,omitempty"`
	BranchState        string    `json:"branch_state,omitempty"`
	BodyState          string    `json:"body_state,omitempty"`
	BranchHeadSHA      string    `json:"branch_head_sha,omitempty"`
	PRHeadSHA          string    `json:"pr_head_sha,omitempty"`
	ExpectedBodyDigest string    `json:"expected_body_digest,omitempty"`
	ActualBodyDigest   string    `json:"actual_body_digest,omitempty"`
	CheckedAt          time.Time `json:"checked_at"`
}

func (p PullRequestFreshness) Fresh() bool {
	return p.BranchState == "in-sync" && p.BodyState == "fresh"
}

type RepoSyncAudit struct {
	Sync        GitSyncTelemetry     `json:"sync"`
	PullRequest PullRequestFreshness `json:"pull_request"`
}

func (a RepoSyncAudit) Summary() string {
	parts := []string{fmt.Sprintf("sync=%s", a.Sync.Status)}
	if a.Sync.FailureCategory != "" {
		parts = append(parts, fmt.Sprintf("failure=%s", a.Sync.FailureCategory))
	}
	parts = append(parts, fmt.Sprintf("pr-branch=%s", a.PullRequest.BranchState))
	parts = append(parts, fmt.Sprintf("pr-body=%s", a.PullRequest.BodyState))
	return joinFields(parts)
}

type ArtifactRecord struct {
	Name      string         `json:"name"`
	Kind      string         `json:"kind"`
	Path      string         `json:"path"`
	Timestamp time.Time      `json:"timestamp"`
	SHA256    string         `json:"sha256,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type AuditEntry struct {
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Outcome   string         `json:"outcome"`
	Timestamp time.Time      `json:"timestamp"`
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
	Timestamp          time.Time            `json:"timestamp"`
}

func (c RunCloseout) Complete() bool {
	return len(c.ValidationEvidence) > 0 && c.GitPushSucceeded && c.GitLogStatOutput != ""
}

type TaskRun struct {
	RunID     string           `json:"run_id"`
	TaskID    string           `json:"task_id"`
	Source    string           `json:"source,omitempty"`
	Title     string           `json:"title"`
	Medium    string           `json:"medium"`
	StartedAt time.Time        `json:"started_at"`
	EndedAt   time.Time        `json:"ended_at,omitempty"`
	Status    string           `json:"status"`
	Summary   string           `json:"summary,omitempty"`
	Artifacts []ArtifactRecord `json:"artifacts,omitempty"`
	Audits    []AuditEntry     `json:"audits,omitempty"`
	Closeout  RunCloseout      `json:"closeout"`
}

func NewTaskRunFromTask(task domain.Task, runID, medium string, now time.Time) TaskRun {
	return TaskRun{
		RunID:     runID,
		TaskID:    task.ID,
		Source:    task.Source,
		Title:     task.Title,
		Medium:    medium,
		StartedAt: now,
		Status:    "running",
		Closeout:  RunCloseout{Timestamp: now},
	}
}

func (r *TaskRun) RegisterArtifact(name, kind, path string, timestamp time.Time, metadata map[string]any) error {
	record := ArtifactRecord{
		Name:      name,
		Kind:      kind,
		Path:      path,
		Timestamp: timestamp,
		Metadata:  metadata,
	}
	digest, err := sha256File(path)
	if err != nil {
		return err
	}
	record.SHA256 = digest
	r.Artifacts = append(r.Artifacts, record)
	r.Audit("artifact.registered", "task-run", "recorded", timestamp, map[string]any{
		"artifact_name": name,
		"artifact_kind": kind,
		"path":          path,
		"sha256":        digest,
	})
	return nil
}

func (r *TaskRun) Audit(action, actor, outcome string, timestamp time.Time, details map[string]any) {
	r.Audits = append(r.Audits, AuditEntry{
		Action:    action,
		Actor:     actor,
		Outcome:   outcome,
		Timestamp: timestamp,
		Details:   details,
	})
}

func (r *TaskRun) RecordCloseout(validationEvidence []string, gitPushSucceeded bool, gitPushOutput, gitLogStatOutput string, repoSyncAudit *RepoSyncAudit, runCommitLinks []repo.RunCommitLink, timestamp time.Time) error {
	closeout := RunCloseout{
		ValidationEvidence: append([]string(nil), validationEvidence...),
		GitPushSucceeded:   gitPushSucceeded,
		GitPushOutput:      gitPushOutput,
		GitLogStatOutput:   gitLogStatOutput,
		RepoSyncAudit:      repoSyncAudit,
		RunCommitLinks:     append([]repo.RunCommitLink(nil), runCommitLinks...),
		Timestamp:          timestamp,
	}
	if len(runCommitLinks) > 0 {
		binding, err := repo.BindRunCommits(runCommitLinks)
		if err != nil {
			return err
		}
		closeout.AcceptedCommitHash = binding.AcceptedCommitHash()
	}
	r.Closeout = closeout
	r.Audit("closeout.recorded", "task-run", "recorded", timestamp, map[string]any{
		"validation_evidence_count": len(validationEvidence),
		"git_push_succeeded":        gitPushSucceeded,
		"git_log_stat_captured":     gitLogStatOutput != "",
		"has_repo_sync_audit":       repoSyncAudit != nil,
	})
	return nil
}

func (r *TaskRun) Finalize(status, summary string, endedAt time.Time) {
	r.Status = status
	r.Summary = summary
	r.EndedAt = endedAt
}

type Ledger struct {
	path string
}

func NewLedger(path string) Ledger {
	return Ledger{path: path}
}

func (l Ledger) LoadRuns() ([]TaskRun, error) {
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
	var runs []TaskRun
	if err := json.Unmarshal(body, &runs); err != nil {
		return nil, err
	}
	return runs, nil
}

func (l Ledger) Append(run TaskRun) error {
	runs, err := l.LoadRuns()
	if err != nil {
		return err
	}
	runs = append(runs, cloneRuns([]TaskRun{run})...)
	return l.writeRuns(runs)
}

func (l Ledger) Upsert(run TaskRun) error {
	runs, err := l.LoadRuns()
	if err != nil {
		return err
	}
	replaced := false
	for i := range runs {
		if runs[i].RunID == run.RunID {
			runs[i] = run
			replaced = true
			break
		}
	}
	if !replaced {
		runs = append(runs, run)
	}
	return l.writeRuns(runs)
}

func (l Ledger) writeRuns(runs []TaskRun) error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(runs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(l.path, body, 0o644)
}

func cloneRuns(runs []TaskRun) []TaskRun {
	out := make([]TaskRun, 0, len(runs))
	for _, run := range runs {
		copyRun := run
		if len(run.Artifacts) > 0 {
			copyRun.Artifacts = append([]ArtifactRecord(nil), run.Artifacts...)
		}
		if len(run.Audits) > 0 {
			copyRun.Audits = append([]AuditEntry(nil), run.Audits...)
		}
		if len(run.Closeout.ValidationEvidence) > 0 {
			copyRun.Closeout.ValidationEvidence = append([]string(nil), run.Closeout.ValidationEvidence...)
		}
		if len(run.Closeout.RunCommitLinks) > 0 {
			copyRun.Closeout.RunCommitLinks = append([]repo.RunCommitLink(nil), run.Closeout.RunCommitLinks...)
		}
		out = append(out, copyRun)
	}
	return out
}

func sha256File(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
