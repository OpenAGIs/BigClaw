package repo

import (
	"encoding/json"
	"strings"
)

type RepoGatewayClient interface {
	PushBundle(repoSpaceID, bundleRef string) (map[string]any, error)
	FetchBundle(repoSpaceID, bundleRef string) (map[string]any, error)
	ListCommits(repoSpaceID string) ([]map[string]any, error)
	GetCommit(repoSpaceID, commitHash string) (map[string]any, error)
	GetChildren(repoSpaceID, commitHash string) ([]string, error)
	GetLineage(repoSpaceID, commitHash string) (map[string]any, error)
	GetLeaves(repoSpaceID, commitHash string) ([]string, error)
	Diff(repoSpaceID, leftHash, rightHash string) (map[string]any, error)
}

type RepoGatewayError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func NormalizeGatewayError(err error) RepoGatewayError {
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(message, "timeout"):
		return RepoGatewayError{Code: "timeout", Message: err.Error(), Retryable: true}
	case strings.Contains(message, "not found"):
		return RepoGatewayError{Code: "not_found", Message: err.Error()}
	default:
		return RepoGatewayError{Code: "gateway_error", Message: err.Error()}
	}
}

func NormalizeCommit(payload map[string]any) (RepoCommit, error) {
	var commit RepoCommit
	if err := decodeMap(payload, &commit); err != nil {
		return RepoCommit{}, err
	}
	return commit, nil
}

func NormalizeLineage(payload map[string]any) (CommitLineage, error) {
	var lineage CommitLineage
	if err := decodeMap(payload, &lineage); err != nil {
		return CommitLineage{}, err
	}
	return lineage, nil
}

func NormalizeDiff(payload map[string]any) (CommitDiff, error) {
	var diff CommitDiff
	if err := decodeMap(payload, &diff); err != nil {
		return CommitDiff{}, err
	}
	return diff, nil
}

func RepoAuditPayload(actor, action, outcome, commitHash, repoSpaceID string) map[string]any {
	return map[string]any{
		"actor":         actor,
		"action":        action,
		"outcome":       outcome,
		"commit_hash":   commitHash,
		"repo_space_id": repoSpaceID,
	}
}

func decodeMap(payload map[string]any, target any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, target)
}
