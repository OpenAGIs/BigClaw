package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLeaderElectionCapabilitySurfaceReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "leader-election-capability-surface.json")

	var report struct {
		Ticket  string `json:"ticket"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Summary struct {
			BackendCount          int    `json:"backend_count"`
			LiveProvenBackends    int    `json:"live_proven_backends"`
			HarnessProvenBackends int    `json:"harness_proven_backends"`
			ContractOnlyBackends  int    `json:"contract_only_backends"`
			CurrentProofBackend   string `json:"current_proof_backend"`
		} `json:"summary"`
		Backends []struct {
			Backend          string   `json:"backend"`
			RuntimeReadiness string   `json:"runtime_readiness"`
			EndpointSurfaces []string `json:"endpoint_surfaces"`
		} `json:"backends"`
	}
	readJSONFile(t, reportPath, &report)
	if report.Ticket != "BIG-PAR-101" || report.Status != "checked_in_surface" {
		t.Fatalf("unexpected leader election report metadata: %+v", report)
	}
	if report.Summary.BackendCount != 4 || report.Summary.LiveProvenBackends != 1 || report.Summary.HarnessProvenBackends != 1 || report.Summary.ContractOnlyBackends != 2 || report.Summary.CurrentProofBackend != "shared_sqlite_subscriber_lease" {
		t.Fatalf("unexpected leader election summary: %+v", report.Summary)
	}
	if len(report.Backends) != 4 {
		t.Fatalf("expected 4 backend rows, got %+v", report.Backends)
	}
	if report.Backends[0].Backend != "shared_sqlite_subscriber_lease" || report.Backends[0].RuntimeReadiness != "live_proven" || len(report.Backends[0].EndpointSurfaces) != 3 {
		t.Fatalf("unexpected shared sqlite row: %+v", report.Backends[0])
	}
	if report.Backends[1].Backend != "shared_store_takeover_hardening" || report.Backends[1].RuntimeReadiness != "harness_proven" {
		t.Fatalf("unexpected shared-store row: %+v", report.Backends[1])
	}
	if report.Backends[3].Backend != "quorum_replicated_leader_election" || report.Backends[3].RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected quorum row: %+v", report.Backends[3])
	}

	for _, relative := range []string{
		"docs/reports/multi-node-coordination-report.md",
		"docs/reports/review-readiness.md",
		"docs/reports/issue-coverage.md",
	} {
		contents := readRepoFile(t, repoRoot, relative)
		for _, needle := range []string{"leader-election-capability-surface.json", "shared SQLite", "harness_proven", "contract_only"} {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", relative, needle)
			}
		}
	}
}
