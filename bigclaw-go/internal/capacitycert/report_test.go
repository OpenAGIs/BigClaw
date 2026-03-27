package capacitycert

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildReportPassesCheckedInEvidence(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	report, markdown, err := BuildReport(BuildOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	summary := nestedMap(report, "summary")
	if got := stringValue(summary["overall_status"], ""); got != "pass" {
		t.Fatalf("overall_status = %q, want pass", got)
	}

	if got := report["generated_at"]; got != "2026-03-13T09:44:42.458392Z" {
		t.Fatalf("generated_at = %v, want 2026-03-13T09:44:42.458392Z", got)
	}

	if got := stringValue(nestedMap(report, "saturation_indicator")["status"], ""); got != "pass" {
		t.Fatalf("saturation_indicator.status = %q, want pass", got)
	}

	if got := stringValue(nestedMap(report, "mixed_workload")["status"], ""); got != "pass" {
		t.Fatalf("mixed_workload.status = %q, want pass", got)
	}

	failedLanes := stringSliceAt(summary, "failed_lanes")
	if len(failedLanes) != 0 {
		t.Fatalf("failed_lanes = %v, want empty", failedLanes)
	}

	soakMatrix := mapSliceAt(report, "soak_matrix")
	found1000x24 := false
	for _, lane := range soakMatrix {
		if stringValue(lane["lane"], "") == "1000x24" {
			found1000x24 = true
			break
		}
	}
	if !found1000x24 {
		t.Fatalf("soak_matrix missing 1000x24 lane")
	}

	if want := "## Admission Policy Summary"; !strings.Contains(markdown, want) {
		t.Fatalf("markdown missing %q", want)
	}
	if want := "Runtime enforcement: `none`"; !strings.Contains(markdown, want) {
		t.Fatalf("markdown missing %q", want)
	}
}
