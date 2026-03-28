package repo

import (
	"encoding/json"
	"strconv"
	"strings"

	"bigclaw-go/internal/domain"
)

type RunCloseout struct {
	ValidationEvidence []string        `json:"validation_evidence,omitempty"`
	GitPushSucceeded   bool            `json:"git_push_succeeded"`
	GitPushOutput      string          `json:"git_push_output,omitempty"`
	GitLogStatOutput   string          `json:"git_log_stat_output,omitempty"`
	RemoteSynced       bool            `json:"remote_synced"`
	LocalSHA           string          `json:"local_sha,omitempty"`
	RemoteSHA          string          `json:"remote_sha,omitempty"`
	AcceptedCommitHash string          `json:"accepted_commit_hash,omitempty"`
	RunCommitLinks     []RunCommitLink `json:"run_commit_links,omitempty"`
}

func BuildRunCloseout(task domain.Task) RunCloseout {
	links := runCommitLinksFromTask(task)
	accepted := strings.TrimSpace(task.Metadata["accepted_commit_hash"])
	if accepted == "" {
		accepted = commitHashForRole(links, "accepted")
	}
	if accepted == "" {
		accepted = strings.TrimSpace(task.Metadata["accepted_ancestor"])
	}
	return RunCloseout{
		ValidationEvidence: parseMetadataStringSlice(task.Metadata["validation_evidence"]),
		GitPushSucceeded:   metadataBoolValue(task.Metadata, "git_push_succeeded"),
		GitPushOutput:      strings.TrimSpace(task.Metadata["git_push_output"]),
		GitLogStatOutput:   strings.TrimSpace(task.Metadata["git_log_stat_output"]),
		RemoteSynced:       metadataBoolValue(task.Metadata, "remote_synced"),
		LocalSHA:           strings.TrimSpace(task.Metadata["local_sha"]),
		RemoteSHA:          strings.TrimSpace(task.Metadata["remote_sha"]),
		AcceptedCommitHash: accepted,
		RunCommitLinks:     links,
	}
}

func runCommitLinksFromTask(task domain.Task) []RunCommitLink {
	raw := strings.TrimSpace(task.Metadata["run_commit_links"])
	if raw != "" {
		var links []RunCommitLink
		if err := json.Unmarshal([]byte(raw), &links); err == nil {
			for i := range links {
				if strings.TrimSpace(links[i].RunID) == "" {
					links[i].RunID = task.ID
				}
			}
			return links
		}
	}
	links := make([]RunCommitLink, 0, 2)
	appendLink := func(role string, hashes ...string) {
		for _, hash := range hashes {
			hash = strings.TrimSpace(hash)
			if hash == "" {
				continue
			}
			links = append(links, RunCommitLink{
				RunID:       task.ID,
				CommitHash:  hash,
				Role:        role,
				RepoSpaceID: strings.TrimSpace(task.Metadata["repo_space_id"]),
				Actor:       strings.TrimSpace(task.Metadata["repo_actor"]),
			})
			return
		}
	}
	appendLink("candidate", task.Metadata["candidate_commit_hash"])
	appendLink("accepted", task.Metadata["accepted_commit_hash"], task.Metadata["accepted_ancestor"])
	return links
}

func commitHashForRole(links []RunCommitLink, role string) string {
	for _, link := range links {
		if strings.EqualFold(strings.TrimSpace(link.Role), role) {
			return strings.TrimSpace(link.CommitHash)
		}
	}
	return ""
}

func metadataBoolValue(metadata map[string]string, key string) bool {
	value := strings.TrimSpace(metadata[key])
	if value == "" {
		return false
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func parseMetadataStringSlice(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "[") {
		var values []string
		if err := json.Unmarshal([]byte(raw), &values); err == nil {
			return values
		}
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ';'
	})
	if len(parts) == 1 && strings.Contains(parts[0], ",") {
		parts = strings.Split(parts[0], ",")
	}
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
