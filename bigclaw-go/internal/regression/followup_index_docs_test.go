package regression

import (
	"strings"
	"testing"
)

func TestParallelFollowUpIndexStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/reports/parallel-follow-up-index.md")

	requiredSubstrings := []string{
		"# Parallel Follow-up Index",
		"canonical repo-native index",
		"planning a new `BIG-PAR-*` slice",
		"| `OPE-264` / `BIG-PAR-075` | tracing backend and span propagation | `docs/reports/tracing-backend-follow-up-digest.md` | `GET /v2/reports/distributed/export`; `docs/reports/ambiguous-publish-outcome-proof-summary.json`; `docs/reports/publish-ack-outcome-surface.json` |",
		"| `OPE-265` / `BIG-PAR-076` | telemetry pipeline, sampling policy, and high-cardinality controls | `docs/reports/telemetry-pipeline-controls-follow-up-digest.md` | `docs/reports/telemetry-sampling-cardinality-evidence-pack.json` |",
		"| `OPE-266` / `BIG-PAR-092` | live shadow traffic comparison and parity drift | `docs/reports/live-shadow-comparison-follow-up-digest.md` | `docs/reports/live-shadow-mirror-scorecard.json`; `docs/reports/live-shadow-index.md`; `GET /debug/status`; `GET /v2/control-center` |",
		"| `OPE-254` / `BIG-PAR-088` | rollback safeguard trigger surface | `docs/reports/rollback-safeguard-follow-up-digest.md` | `docs/reports/rollback-trigger-surface.json`; `GET /debug/status`; `GET /v2/control-center` |",
		"| `OPE-268` / `BIG-PAR-079` | production corpus coverage | `docs/reports/production-corpus-migration-coverage-digest.md` | `docs/reports/shadow-matrix-report.json`; `docs/reports/live-shadow-drift-rollup.json` |",
		"| `OPE-269` / `BIG-PAR-080` | subscriber takeover executability | `docs/reports/subscriber-takeover-executability-follow-up-digest.md` | `docs/reports/multi-subscriber-takeover-validation-report.md`; `docs/reports/live-multi-node-subscriber-takeover-report.json` |",
		"| `OPE-261` / `BIG-PAR-085` | cross-process coordination boundary | `docs/reports/cross-process-coordination-boundary-digest.md` | `docs/reports/cross-process-coordination-capability-surface.json`; `docs/reports/multi-node-coordination-report.md`; `docs/reports/external-store-validation-report.json` |",
		"| `OPE-257` / `BIG-PAR-095` | contract-only coordination targets | `docs/reports/cross-process-coordination-boundary-digest.md` | `docs/reports/cross-process-coordination-capability-surface.json` |",
		"| `OPE-271` / `BIG-PAR-082` | validation bundle continuation | `docs/reports/validation-bundle-continuation-digest.md` | `docs/reports/validation-bundle-continuation-scorecard.json`; `docs/reports/validation-bundle-continuation-policy-gate.json`; `docs/reports/shared-queue-companion-summary.json` |",
		"| `OPE-222` | replicated event-log durability rollout gate | `docs/reports/replicated-event-log-durability-rollout-contract.md` | `docs/reports/replicated-broker-durability-rollout-spike.md`; `docs/reports/broker-durability-rollout-scorecard.json`; `internal/events/durability.go` |",
		"`docs/reports/parallel-validation-matrix.md`",
		"prefer linking here instead of repeating the full",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/reports/parallel-follow-up-index.md missing substring %q", needle)
		}
	}
}
