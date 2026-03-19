package observability

import "testing"

func TestRepoSyncAuditSummaryIncludesFailureAndPRState(t *testing.T) {
	audit := RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "blocked",
			FailureCategory: "diverged",
		},
		PullRequest: PullRequestFreshness{
			BranchState: "stale",
			BodyState:   "outdated",
		},
	}
	if got := audit.Summary(); got != "sync=blocked, failure=diverged, pr-branch=stale, pr-body=outdated" {
		t.Fatalf("unexpected summary: %q", got)
	}
}

func TestPullRequestFreshnessFreshRequiresBranchAndBodyAlignment(t *testing.T) {
	fresh := PullRequestFreshness{BranchState: "in-sync", BodyState: "fresh"}
	if !fresh.Fresh() {
		t.Fatal("expected fresh PR state")
	}
	stale := PullRequestFreshness{BranchState: "stale", BodyState: "fresh"}
	if stale.Fresh() {
		t.Fatal("expected stale PR state to be false")
	}
}

func TestRepoSyncAuditVerifiedRequiresSyncAndPRFreshness(t *testing.T) {
	verified := RepoSyncAudit{
		Sync:        GitSyncTelemetry{Status: "synced"},
		PullRequest: PullRequestFreshness{BranchState: "in-sync", BodyState: "fresh"},
	}
	if !verified.Verified() {
		t.Fatal("expected verified repo sync audit")
	}
	incomplete := RepoSyncAudit{
		Sync:        GitSyncTelemetry{Status: "synced"},
		PullRequest: PullRequestFreshness{BranchState: "stale", BodyState: "fresh"},
	}
	if incomplete.Verified() {
		t.Fatal("expected stale PR branch to fail verification")
	}
}
