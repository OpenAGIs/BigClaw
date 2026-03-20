package repo

import (
	"errors"
	"reflect"
	"testing"
)

func TestNormalizeCommitLineageAndDiff(t *testing.T) {
	commit := NormalizeCommit(map[string]any{
		"commit_hash":   "abc123",
		"title":         "Ship repo slice",
		"author":        "alice",
		"parent_hashes": []any{"base1", "base2"},
		"metadata":      map[string]any{"ticket": "BIG-1"},
	})
	if commit.CommitHash != "abc123" || commit.Title != "Ship repo slice" || commit.Author != "alice" {
		t.Fatalf("unexpected commit normalization: %+v", commit)
	}
	if !reflect.DeepEqual(commit.ParentHashes, []string{"base1", "base2"}) {
		t.Fatalf("unexpected parent hashes: %+v", commit.ParentHashes)
	}
	if commit.Metadata["ticket"] != "BIG-1" {
		t.Fatalf("unexpected commit metadata: %+v", commit.Metadata)
	}

	lineage := NormalizeLineage(map[string]any{
		"root_hash": "root1",
		"lineage": []any{
			map[string]any{"commit_hash": "abc123", "title": "Ship repo slice"},
			map[string]any{"commit_hash": "def456", "title": "Review repo slice"},
		},
		"children": map[string]any{
			"abc123": []any{"def456"},
		},
		"leaves": []any{"def456"},
	})
	if lineage.RootHash != "root1" || len(lineage.Lineage) != 2 {
		t.Fatalf("unexpected lineage normalization: %+v", lineage)
	}
	if !reflect.DeepEqual(lineage.Children["abc123"], []string{"def456"}) || !reflect.DeepEqual(lineage.Leaves, []string{"def456"}) {
		t.Fatalf("unexpected lineage graph normalization: %+v", lineage)
	}

	diff := NormalizeDiff(map[string]any{
		"left_hash":     "abc123",
		"right_hash":    "def456",
		"files_changed": float64(3),
		"insertions":    int64(8),
		"deletions":     2,
		"summary":       "3 files changed",
	})
	if diff.LeftHash != "abc123" || diff.RightHash != "def456" || diff.FilesChanged != 3 || diff.Insertions != 8 || diff.Deletions != 2 || diff.Summary != "3 files changed" {
		t.Fatalf("unexpected diff normalization: %+v", diff)
	}
}

func TestNormalizeGatewayErrorAndAuditPayload(t *testing.T) {
	timeout := NormalizeGatewayError(errors.New("gateway timeout while fetching lineage"))
	if timeout.Code != "timeout" || !timeout.Retryable {
		t.Fatalf("unexpected timeout normalization: %+v", timeout)
	}
	notFound := NormalizeGatewayError(errors.New("commit not found"))
	if notFound.Code != "not_found" || notFound.Retryable {
		t.Fatalf("unexpected not found normalization: %+v", notFound)
	}
	fallback := NormalizeGatewayError(errors.New("permission denied"))
	if fallback.Code != "gateway_error" || fallback.Retryable {
		t.Fatalf("unexpected fallback normalization: %+v", fallback)
	}

	payload := RepoAuditPayload("alice", "repo.fetch", "ok", "abc123", "repo-space-1")
	if payload["actor"] != "alice" || payload["action"] != "repo.fetch" || payload["outcome"] != "ok" || payload["commit_hash"] != "abc123" || payload["repo_space_id"] != "repo-space-1" {
		t.Fatalf("unexpected repo audit payload: %+v", payload)
	}
}
