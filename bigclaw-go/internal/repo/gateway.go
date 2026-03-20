package repo

import "strings"

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

func RepoAuditPayload(actor string, action string, outcome string, commitHash string, repoSpaceID string) map[string]any {
	return map[string]any{
		"actor":         actor,
		"action":        action,
		"outcome":       outcome,
		"commit_hash":   commitHash,
		"repo_space_id": repoSpaceID,
	}
}
