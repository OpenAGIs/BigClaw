package repo

import (
	"errors"
	"reflect"
	"testing"
)

type fakeGatewayClient struct{}

func (fakeGatewayClient) PushBundle(repoSpaceID string, bundleRef string) map[string]any {
	return map[string]any{"repo_space_id": repoSpaceID, "bundle_ref": bundleRef}
}

func (fakeGatewayClient) FetchBundle(repoSpaceID string, bundleRef string) map[string]any {
	return map[string]any{"repo_space_id": repoSpaceID, "bundle_ref": bundleRef}
}

func (fakeGatewayClient) ListCommits(repoSpaceID string) []map[string]any {
	return []map[string]any{{"commit_hash": "abc123", "title": "ship"}}
}

func (fakeGatewayClient) GetCommit(repoSpaceID string, commitHash string) map[string]any {
	return map[string]any{"commit_hash": commitHash, "title": "ship"}
}

func (fakeGatewayClient) GetChildren(repoSpaceID string, commitHash string) []string {
	return []string{"def456"}
}

func (fakeGatewayClient) GetLineage(repoSpaceID string, commitHash string) map[string]any {
	return map[string]any{"root_hash": commitHash}
}

func (fakeGatewayClient) GetLeaves(repoSpaceID string, commitHash string) []string {
	return []string{commitHash}
}

func (fakeGatewayClient) Diff(repoSpaceID string, leftHash string, rightHash string) map[string]any {
	return map[string]any{"left_hash": leftHash, "right_hash": rightHash}
}

func TestGatewayContractNormalization(t *testing.T) {
	var client GatewayClient = fakeGatewayClient{}
	bundle := NormalizeBundle(client.PushBundle("space-1", "bundle-1"))
	if bundle.RepoSpaceID != "space-1" || bundle.BundleRef != "bundle-1" {
		t.Fatalf("unexpected normalized bundle: %+v", bundle)
	}

	commits := NormalizeCommitList(client.ListCommits("space-1"))
	if len(commits) != 1 || commits[0].CommitHash != "abc123" {
		t.Fatalf("unexpected normalized commit list: %+v", commits)
	}

	if children := client.GetChildren("space-1", "abc123"); !reflect.DeepEqual(children, []string{"def456"}) {
		t.Fatalf("unexpected children: %+v", children)
	}
	if leaves := client.GetLeaves("space-1", "abc123"); !reflect.DeepEqual(leaves, []string{"abc123"}) {
		t.Fatalf("unexpected leaves: %+v", leaves)
	}
}

func TestGatewayErrorAndAuditPayloadRemainDeterministic(t *testing.T) {
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
