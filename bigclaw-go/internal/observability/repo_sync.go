package observability

import "strings"

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

func (telemetry GitSyncTelemetry) OK() bool {
	return strings.EqualFold(strings.TrimSpace(telemetry.Status), "synced")
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

func (freshness PullRequestFreshness) Fresh() bool {
	return strings.EqualFold(strings.TrimSpace(freshness.BranchState), "in-sync") &&
		strings.EqualFold(strings.TrimSpace(freshness.BodyState), "fresh")
}

type RepoSyncAudit struct {
	Sync        GitSyncTelemetry     `json:"sync"`
	PullRequest PullRequestFreshness `json:"pull_request"`
}

func (audit RepoSyncAudit) Verified() bool {
	return audit.Sync.OK() && audit.PullRequest.Fresh()
}

func (audit RepoSyncAudit) Summary() string {
	parts := []string{"sync=" + firstNonEmpty(audit.Sync.Status, "unknown")}
	if failure := strings.TrimSpace(audit.Sync.FailureCategory); failure != "" {
		parts = append(parts, "failure="+failure)
	}
	parts = append(parts,
		"pr-branch="+firstNonEmpty(audit.PullRequest.BranchState, "unknown"),
		"pr-body="+firstNonEmpty(audit.PullRequest.BodyState, "unknown"),
	)
	return strings.Join(parts, ", ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
