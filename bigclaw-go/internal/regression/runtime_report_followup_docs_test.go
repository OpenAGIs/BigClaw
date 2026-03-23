package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeReportFollowUpDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/live-validation-index.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-digest.md",
				"validation-bundle-continuation-scorecard.json",
				"validation-bundle-continuation-policy-gate.json",
			},
		},
		{
			path: "docs/reports/live-validation-runs/20260316T140138Z/README.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-digest.md",
				"validation-bundle-continuation-scorecard.json",
				"validation-bundle-continuation-policy-gate.json",
				"live-validation-index.md",
			},
		},
		{
			path: "docs/reports/multi-node-coordination-report.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-digest.md",
				"cross-process-coordination-boundary-digest.md",
			},
		},
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"OPE-269` / `BIG-PAR-080",
				"subscriber-takeover-executability-follow-up-digest.md",
				"live-multi-node-subscriber-takeover-report.json",
			},
		},
		{
			path: "docs/reports/subscriber-takeover-executability-follow-up-digest.md",
			substrings: []string{
				"OPE-269` / `BIG-PAR-080",
				"live-multi-node-subscriber-takeover-report.json",
			},
		},
		{
			path: "docs/reports/validation-bundle-continuation-digest.md",
			substrings: []string{
				"OPE-271` / `BIG-PAR-082",
				"validation-bundle-continuation-policy-gate.json",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}

func TestLiveValidationIndexContinuationMetadata(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	contents, err := os.ReadFile(filepath.Join(repoRoot, "docs", "reports", "live-validation-index.json"))
	if err != nil {
		t.Fatalf("read live validation index json: %v", err)
	}

	var payload struct {
		ContinuationGate struct {
			Path         string `json:"path"`
			ReviewerPath struct {
				IndexPath   string `json:"index_path"`
				DigestPath  string `json:"digest_path"`
				DigestIssue struct {
					ID   string `json:"id"`
					Slug string `json:"slug"`
				} `json:"digest_issue"`
			} `json:"reviewer_path"`
		} `json:"continuation_gate"`
	}
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatalf("decode live validation index json: %v", err)
	}

	if payload.ContinuationGate.Path != "docs/reports/validation-bundle-continuation-policy-gate.json" {
		t.Fatalf("unexpected continuation gate path: %s", payload.ContinuationGate.Path)
	}
	if payload.ContinuationGate.ReviewerPath.IndexPath != "docs/reports/live-validation-index.md" {
		t.Fatalf("unexpected reviewer index path: %s", payload.ContinuationGate.ReviewerPath.IndexPath)
	}
	if payload.ContinuationGate.ReviewerPath.DigestPath != "docs/reports/validation-bundle-continuation-digest.md" {
		t.Fatalf("unexpected reviewer digest path: %s", payload.ContinuationGate.ReviewerPath.DigestPath)
	}
	if payload.ContinuationGate.ReviewerPath.DigestIssue.ID != "OPE-271" || payload.ContinuationGate.ReviewerPath.DigestIssue.Slug != "BIG-PAR-082" {
		t.Fatalf("unexpected reviewer digest issue: %+v", payload.ContinuationGate.ReviewerPath.DigestIssue)
	}
}

func TestContinuationPolicyGateReviewerMetadata(t *testing.T) {
	repoRoot := filepath.Join("..", "..")
	contents, err := os.ReadFile(filepath.Join(repoRoot, "docs", "reports", "validation-bundle-continuation-policy-gate.json"))
	if err != nil {
		t.Fatalf("read continuation policy gate json: %v", err)
	}

	var payload struct {
		ReviewerPath struct {
			IndexPath   string `json:"index_path"`
			DigestPath  string `json:"digest_path"`
			DigestIssue struct {
				ID   string `json:"id"`
				Slug string `json:"slug"`
			} `json:"digest_issue"`
		} `json:"reviewer_path"`
	}
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatalf("decode continuation policy gate json: %v", err)
	}

	if payload.ReviewerPath.IndexPath != "docs/reports/live-validation-index.md" {
		t.Fatalf("unexpected policy gate reviewer index path: %s", payload.ReviewerPath.IndexPath)
	}
	if payload.ReviewerPath.DigestPath != "docs/reports/validation-bundle-continuation-digest.md" {
		t.Fatalf("unexpected policy gate reviewer digest path: %s", payload.ReviewerPath.DigestPath)
	}
	if payload.ReviewerPath.DigestIssue.ID != "OPE-271" || payload.ReviewerPath.DigestIssue.Slug != "BIG-PAR-082" {
		t.Fatalf("unexpected policy gate reviewer digest issue: %+v", payload.ReviewerPath.DigestIssue)
	}
}
