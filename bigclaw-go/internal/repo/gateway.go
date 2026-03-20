package repo

import "strings"

type GatewayClient interface {
	PushBundle(repoSpaceID string, bundleRef string) map[string]any
	FetchBundle(repoSpaceID string, bundleRef string) map[string]any
	ListCommits(repoSpaceID string) []map[string]any
	GetCommit(repoSpaceID string, commitHash string) map[string]any
	GetChildren(repoSpaceID string, commitHash string) []string
	GetLineage(repoSpaceID string, commitHash string) map[string]any
	GetLeaves(repoSpaceID string, commitHash string) []string
	Diff(repoSpaceID string, leftHash string, rightHash string) map[string]any
}

type RepoBundle struct {
	RepoSpaceID string         `json:"repo_space_id,omitempty"`
	BundleRef   string         `json:"bundle_ref,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type RepoGatewayError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func NormalizeGatewayError(err error) RepoGatewayError {
	if err == nil {
		return RepoGatewayError{}
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "timeout"):
		return RepoGatewayError{Code: "timeout", Message: err.Error(), Retryable: true}
	case strings.Contains(message, "not found"):
		return RepoGatewayError{Code: "not_found", Message: err.Error()}
	default:
		return RepoGatewayError{Code: "gateway_error", Message: err.Error()}
	}
}

func NormalizeBundle(payload map[string]any) RepoBundle {
	return RepoBundle{
		RepoSpaceID: stringValue(payload["repo_space_id"]),
		BundleRef:   stringValue(payload["bundle_ref"]),
		Metadata:    mapValue(payload["metadata"]),
	}
}

func NormalizeCommitList(payload []map[string]any) []RepoCommit {
	commits := make([]RepoCommit, 0, len(payload))
	for _, item := range payload {
		commits = append(commits, NormalizeCommit(item))
	}
	return commits
}

func RepoAuditPayload(actor string, action string, outcome string, commitHash string, repoSpaceID string) map[string]any {
	return map[string]any{
		"actor":         actor,
		"action":        action,
		"outcome":       outcome,
		"commit_hash":   commitHash,
		"repo_space_id": repoSpaceID,
	}
}
