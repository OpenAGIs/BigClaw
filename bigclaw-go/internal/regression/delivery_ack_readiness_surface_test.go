package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDeliveryAckReadinessSurfaceReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "delivery-ack-readiness-surface.json")

	var report struct {
		Ticket  string `json:"ticket"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Summary struct {
			BackendCount         int `json:"backend_count"`
			ExplicitAckBackends  int `json:"explicit_ack_backends"`
			DurableAckBackends   int `json:"durable_ack_backends"`
			BestEffortBackends   int `json:"best_effort_backends"`
			ContractOnlyBackends int `json:"contract_only_backends"`
		} `json:"summary"`
		Backends []struct {
			Backend                 string `json:"backend"`
			AcknowledgementClass    string `json:"acknowledgement_class"`
			ExplicitAcknowledgement bool   `json:"explicit_acknowledgement"`
			DurableAcknowledgement  bool   `json:"durable_acknowledgement"`
			RuntimeReadiness        string `json:"runtime_readiness"`
		} `json:"backends"`
	}
	readJSONFile(t, reportPath, &report)
	if report.Ticket != "OPE-264" || report.Status != "checked_in_surface" {
		t.Fatalf("unexpected delivery ack report metadata: %+v", report)
	}
	if report.Summary.BackendCount != 5 || report.Summary.ExplicitAckBackends != 3 || report.Summary.DurableAckBackends != 2 || report.Summary.BestEffortBackends != 1 || report.Summary.ContractOnlyBackends != 1 {
		t.Fatalf("unexpected delivery ack summary: %+v", report.Summary)
	}
	if len(report.Backends) != 5 {
		t.Fatalf("expected 5 backend rows, got %+v", report.Backends)
	}
	if report.Backends[0].Backend != "memory" || report.Backends[0].AcknowledgementClass != "best_effort_only" || report.Backends[0].ExplicitAcknowledgement || report.Backends[0].DurableAcknowledgement {
		t.Fatalf("unexpected memory delivery ack row: %+v", report.Backends[0])
	}
	if report.Backends[1].Backend != "sqlite" || !report.Backends[1].ExplicitAcknowledgement || !report.Backends[1].DurableAcknowledgement {
		t.Fatalf("unexpected sqlite delivery ack row: %+v", report.Backends[1])
	}
	if report.Backends[4].Backend != "broker_replicated" || report.Backends[4].RuntimeReadiness != "contract_only" {
		t.Fatalf("unexpected broker replicated delivery ack row: %+v", report.Backends[4])
	}

	contents := readRepoFile(t, repoRoot, "docs/reports/event-bus-reliability-report.md")
	for _, needle := range []string{"delivery-ack-readiness-surface.json", "Memory bus delivery acknowledgements remain sink-level best effort"} {
		if !strings.Contains(contents, needle) {
			t.Fatalf("event-bus reliability report missing substring %q", needle)
		}
	}

	for _, relative := range []string{
		"docs/reports/issue-coverage.md",
		"docs/reports/review-readiness.md",
	} {
		body := readRepoFile(t, repoRoot, relative)
		if !strings.Contains(body, "delivery-ack-readiness-surface.json") {
			t.Fatalf("expected %s to reference delivery ack readiness surface", relative)
		}
	}
}
