package repo

import (
	"strings"
	"testing"
)

func TestBindRunCommitsSupportsAcceptedHash(t *testing.T) {
	links := []RunCommitLink{
		{RunID: "run-143", CommitHash: "aaa111", Role: "source", RepoSpaceID: "space-1"},
		{RunID: "run-143", CommitHash: "bbb222", Role: "candidate", RepoSpaceID: "space-1"},
		{RunID: "run-143", CommitHash: "ccc333", Role: "accepted", RepoSpaceID: "space-1"},
	}

	binding, err := BindRunCommits(links)
	if err != nil {
		t.Fatalf("expected commit binding to succeed: %v", err)
	}
	if binding.AcceptedCommitHash != "ccc333" {
		t.Fatalf("unexpected accepted commit hash: %+v", binding)
	}
	if len(binding.Links) != 3 {
		t.Fatalf("unexpected bound links: %+v", binding.Links)
	}
}

func TestValidateRunCommitRolesRejectsUnsupportedRoles(t *testing.T) {
	err := ValidateRunCommitRoles([]RunCommitLink{
		{RunID: "run-143", CommitHash: "aaa111", Role: "candidate", RepoSpaceID: "space-1"},
		{RunID: "run-143", CommitHash: "bbb222", Role: "mystery", RepoSpaceID: "space-1"},
		{RunID: "run-143", CommitHash: "ccc333", Role: "unknown", RepoSpaceID: "space-1"},
	})
	if err == nil {
		t.Fatalf("expected unsupported roles to fail validation")
	}
	if got := err.Error(); !strings.Contains(got, "mystery") || !strings.Contains(got, "unknown") {
		t.Fatalf("unexpected validation error: %s", got)
	}
}
