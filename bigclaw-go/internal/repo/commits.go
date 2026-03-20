package repo

import (
	"fmt"
	"strings"
)

type RepoCommit struct {
	CommitHash   string         `json:"commit_hash"`
	Title        string         `json:"title"`
	Author       string         `json:"author,omitempty"`
	ParentHashes []string       `json:"parent_hashes,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type CommitLineage struct {
	RootHash string              `json:"root_hash"`
	Lineage  []RepoCommit        `json:"lineage,omitempty"`
	Children map[string][]string `json:"children,omitempty"`
	Leaves   []string            `json:"leaves,omitempty"`
}

type CommitDiff struct {
	LeftHash     string `json:"left_hash"`
	RightHash    string `json:"right_hash"`
	FilesChanged int    `json:"files_changed"`
	Insertions   int    `json:"insertions"`
	Deletions    int    `json:"deletions"`
	Summary      string `json:"summary,omitempty"`
}

func NormalizeCommit(payload map[string]any) RepoCommit {
	return RepoCommit{
		CommitHash:   stringValue(payload["commit_hash"]),
		Title:        stringValue(payload["title"]),
		Author:       stringValue(payload["author"]),
		ParentHashes: stringSliceValue(payload["parent_hashes"]),
		Metadata:     mapValue(payload["metadata"]),
	}
}

func NormalizeLineage(payload map[string]any) CommitLineage {
	lineageItems := sliceValue(payload["lineage"])
	lineage := make([]RepoCommit, 0, len(lineageItems))
	for _, item := range lineageItems {
		lineage = append(lineage, NormalizeCommit(mapValue(item)))
	}
	children := make(map[string][]string)
	for key, value := range mapValue(payload["children"]) {
		children[key] = stringSliceValue(value)
	}
	return CommitLineage{
		RootHash: stringValue(payload["root_hash"]),
		Lineage:  lineage,
		Children: children,
		Leaves:   stringSliceValue(payload["leaves"]),
	}
}

func NormalizeDiff(payload map[string]any) CommitDiff {
	return CommitDiff{
		LeftHash:     stringValue(payload["left_hash"]),
		RightHash:    stringValue(payload["right_hash"]),
		FilesChanged: intValue(payload["files_changed"]),
		Insertions:   intValue(payload["insertions"]),
		Deletions:    intValue(payload["deletions"]),
		Summary:      stringValue(payload["summary"]),
	}
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func defaultString(value any, fallback string) string {
	if text := stringValue(value); text != "" {
		return text
	}
	return fallback
}

func stringSliceValue(value any) []string {
	items := sliceValue(value)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := stringValue(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func sliceValue(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			out[key] = item
		}
		return out
	default:
		return map[string]any{}
	}
}

func defaultBool(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	default:
		return fallback
	}
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func slugify(value string) string {
	parts := make([]rune, 0, len(value))
	lastDash := false
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			parts = append(parts, r)
			lastDash = false
			continue
		}
		if !lastDash {
			parts = append(parts, '-')
			lastDash = true
		}
	}
	out := strings.Trim(string(parts), "-")
	if out == "" {
		return "agent"
	}
	return out
}

func lowerASCII(value string) string {
	return strings.ToLower(value)
}

func joinStrings(items []string, sep string) string {
	return strings.Join(items, sep)
}
