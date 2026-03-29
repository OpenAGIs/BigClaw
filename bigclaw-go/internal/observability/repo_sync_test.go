package observability

import (
	"strings"
	"testing"
)

func TestRenderRepoSyncAuditReport(t *testing.T) {
	prNumber := 219
	audit := RepoSyncAudit{
		Sync: GitSyncTelemetry{
			Status:          "failed",
			FailureCategory: "auth",
			Summary:         "github token expired",
			Branch:          "dcjcloud/ope-219",
			RemoteRef:       "origin/dcjcloud/ope-219",
			AuthTarget:      "github.com/OpenAGIs/BigClaw.git",
		},
		PullRequest: PullRequestFreshness{
			PRNumber:           &prNumber,
			PRURL:              "https://github.com/OpenAGIs/BigClaw/pull/219",
			BranchState:        "in-sync",
			BodyState:          "drifted",
			BranchHeadSHA:      "abc123",
			PRHeadSHA:          "abc123",
			ExpectedBodyDigest: "expected",
			ActualBodyDigest:   "actual",
		},
	}
	report := RenderRepoSyncAuditReport(audit)
	for _, want := range []string{
		"# Repo Sync Audit",
		"Failure Category: auth",
		"Branch State: in-sync",
		"Body State: drifted",
		"sync=failed, failure=auth, pr-branch=in-sync, pr-body=drifted",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}
