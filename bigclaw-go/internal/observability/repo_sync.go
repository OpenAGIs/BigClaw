package observability

import (
	"fmt"
	"strings"
)

type GitSyncTelemetry struct {
	Status          string   `json:"status,omitempty"`
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
	BranchState        string `json:"branch_state,omitempty"`
	BodyState          string `json:"body_state,omitempty"`
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

func (a RepoSyncAudit) Summary() string {
	parts := []string{fmt.Sprintf("sync=%s", a.Sync.Status)}
	if strings.TrimSpace(a.Sync.FailureCategory) != "" {
		parts = append(parts, fmt.Sprintf("failure=%s", a.Sync.FailureCategory))
	}
	parts = append(parts,
		fmt.Sprintf("pr-branch=%s", a.PullRequest.BranchState),
		fmt.Sprintf("pr-body=%s", a.PullRequest.BodyState),
	)
	return strings.Join(parts, ", ")
}

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	remote := audit.Sync.Remote
	if strings.TrimSpace(remote) == "" {
		remote = "origin"
	}
	prNumber := "unknown"
	if audit.PullRequest.PRNumber != nil {
		prNumber = fmt.Sprintf("%d", *audit.PullRequest.PRNumber)
	}
	lines := []string{
		"# Repo Sync Audit",
		"",
		"## Sync Status",
		"",
		fmt.Sprintf("- Status: %s", firstNonEmptyAudit(audit.Sync.Status, "unknown")),
		fmt.Sprintf("- Failure Category: %s", firstNonEmptyAudit(audit.Sync.FailureCategory, "none")),
		fmt.Sprintf("- Summary: %s", firstNonEmptyAudit(audit.Sync.Summary, "none")),
		fmt.Sprintf("- Branch: %s", firstNonEmptyAudit(audit.Sync.Branch, "unknown")),
		fmt.Sprintf("- Remote: %s", remote),
		fmt.Sprintf("- Remote Ref: %s", firstNonEmptyAudit(audit.Sync.RemoteRef, "unknown")),
		fmt.Sprintf("- Ahead By: %d", audit.Sync.AheadBy),
		fmt.Sprintf("- Behind By: %d", audit.Sync.BehindBy),
		fmt.Sprintf("- Dirty Paths: %s", joinAuditOrNone(audit.Sync.DirtyPaths)),
		fmt.Sprintf("- Auth Target: %s", firstNonEmptyAudit(audit.Sync.AuthTarget, "none")),
		fmt.Sprintf("- Checked At: %s", firstNonEmptyAudit(audit.Sync.Timestamp, "unknown")),
		"",
		"## Pull Request Freshness",
		"",
		fmt.Sprintf("- PR Number: %s", prNumber),
		fmt.Sprintf("- PR URL: %s", firstNonEmptyAudit(audit.PullRequest.PRURL, "none")),
		fmt.Sprintf("- Branch State: %s", firstNonEmptyAudit(audit.PullRequest.BranchState, "unknown")),
		fmt.Sprintf("- Body State: %s", firstNonEmptyAudit(audit.PullRequest.BodyState, "unknown")),
		fmt.Sprintf("- Branch Head SHA: %s", firstNonEmptyAudit(audit.PullRequest.BranchHeadSHA, "unknown")),
		fmt.Sprintf("- PR Head SHA: %s", firstNonEmptyAudit(audit.PullRequest.PRHeadSHA, "unknown")),
		fmt.Sprintf("- Expected Body Digest: %s", firstNonEmptyAudit(audit.PullRequest.ExpectedBodyDigest, "unknown")),
		fmt.Sprintf("- Actual Body Digest: %s", firstNonEmptyAudit(audit.PullRequest.ActualBodyDigest, "unknown")),
		fmt.Sprintf("- Checked At: %s", firstNonEmptyAudit(audit.PullRequest.CheckedAt, "unknown")),
		"",
		"## Summary",
		"",
		fmt.Sprintf("- %s", audit.Summary()),
	}
	return strings.Join(lines, "\n") + "\n"
}

func firstNonEmptyAudit(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func joinAuditOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
