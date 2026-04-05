package reporting

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ValidationBundleExporterGenerator = "bigclaw-go/scripts/e2e/export_validation_bundle/main.go"

var (
	liveValidationLatestReports = map[string]string{
		"local":      "docs/reports/sqlite-smoke-report.json",
		"kubernetes": "docs/reports/kubernetes-live-smoke-report.json",
		"ray":        "docs/reports/ray-live-smoke-report.json",
	}
	liveValidationContinuationArtifacts = []liveValidationArtifact{
		{
			Path:        "docs/reports/validation-bundle-continuation-scorecard.json",
			Description: "summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof.",
		},
		{
			Path:        "docs/reports/validation-bundle-continuation-policy-gate.json",
			Description: "records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability.",
		},
	}
	liveValidationFollowupDigests = []liveValidationArtifact{
		{
			Path:        "docs/reports/validation-bundle-continuation-digest.md",
			Description: "Validation bundle continuation caveats are consolidated here.",
		},
	}
	liveValidationLaneAliases = map[string]string{
		"local":      "local",
		"kubernetes": "k8s",
		"ray":        "ray",
	}
	liveValidationFailureEventTypes = map[string]struct{}{
		"task.cancelled":   {},
		"task.dead_letter": {},
		"task.failed":      {},
		"task.retried":     {},
	}
)

const (
	liveValidationBrokerSummary          = "docs/reports/broker-validation-summary.json"
	liveValidationBrokerBootstrapSummary = "docs/reports/broker-bootstrap-review-summary.json"
	liveValidationBrokerValidationPack   = "docs/reports/broker-failover-fault-injection-validation-pack.md"
	liveValidationSharedQueueReport      = "docs/reports/multi-node-shared-queue-report.json"
	liveValidationSharedQueueSummary     = "docs/reports/shared-queue-companion-summary.json"
)

type LiveValidationBundleOptions struct {
	RunID                      string
	BundleDir                  string
	SummaryPath                string
	IndexPath                  string
	ManifestPath               string
	RunLocal                   bool
	RunKubernetes              bool
	RunRay                     bool
	ValidationStatus           int
	RunBroker                  bool
	BrokerBackend              string
	BrokerReportPath           string
	BrokerBootstrapSummaryPath string
	LocalReportPath            string
	LocalStdoutPath            string
	LocalStderrPath            string
	KubernetesReportPath       string
	KubernetesStdoutPath       string
	KubernetesStderrPath       string
	RayReportPath              string
	RayStdoutPath              string
	RayStderrPath              string
	Now                        time.Time
}

type liveValidationArtifact struct {
	Path        string
	Description string
}

func ExportLiveValidationBundle(root string, options LiveValidationBundleOptions) (map[string]any, string, map[string]any, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, "", nil, errors.New("go root is required")
	}
	if options.RunID == "" {
		return nil, "", nil, errors.New("run id is required")
	}
	if options.BundleDir == "" {
		return nil, "", nil, errors.New("bundle dir is required")
	}
	if options.SummaryPath == "" {
		options.SummaryPath = "docs/reports/live-validation-summary.json"
	}
	if options.IndexPath == "" {
		options.IndexPath = "docs/reports/live-validation-index.md"
	}
	if options.ManifestPath == "" {
		options.ManifestPath = "docs/reports/live-validation-index.json"
	}

	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	bigclawGoRoot := filepath.Join(root, "bigclaw-go")
	bundleDir := resolveLiveValidationPath(root, options.BundleDir)
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, "", nil, err
	}

	summary := map[string]any{
		"run_id":       options.RunID,
		"generated_at": formatOffsetTimestamp(now),
		"status":       liveValidationSummaryStatus(options.ValidationStatus),
		"bundle_path":  liveValidationRelPath(root, bundleDir),
		"closeout_commands": []string{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}

	localSection, err := buildLiveValidationComponentSection(root, bigclawGoRoot, bundleDir, liveValidationComponentInput{
		Name:       "local",
		Enabled:    options.RunLocal,
		ReportPath: options.LocalReportPath,
		StdoutPath: options.LocalStdoutPath,
		StderrPath: options.LocalStderrPath,
	})
	if err != nil {
		return nil, "", nil, err
	}
	summary["local"] = localSection

	kubernetesSection, err := buildLiveValidationComponentSection(root, bigclawGoRoot, bundleDir, liveValidationComponentInput{
		Name:       "kubernetes",
		Enabled:    options.RunKubernetes,
		ReportPath: options.KubernetesReportPath,
		StdoutPath: options.KubernetesStdoutPath,
		StderrPath: options.KubernetesStderrPath,
	})
	if err != nil {
		return nil, "", nil, err
	}
	summary["kubernetes"] = kubernetesSection

	raySection, err := buildLiveValidationComponentSection(root, bigclawGoRoot, bundleDir, liveValidationComponentInput{
		Name:       "ray",
		Enabled:    options.RunRay,
		ReportPath: options.RayReportPath,
		StdoutPath: options.RayStdoutPath,
		StderrPath: options.RayStderrPath,
	})
	if err != nil {
		return nil, "", nil, err
	}
	summary["ray"] = raySection

	brokerSection, err := buildLiveValidationBrokerSection(root, bigclawGoRoot, bundleDir, liveValidationBrokerInput{
		Enabled:              options.RunBroker,
		Backend:              strings.TrimSpace(options.BrokerBackend),
		ReportPath:           options.BrokerReportPath,
		BootstrapSummaryPath: options.BrokerBootstrapSummaryPath,
	})
	if err != nil {
		return nil, "", nil, err
	}
	summary["broker"] = brokerSection

	sharedQueueSection, err := buildLiveValidationSharedQueueCompanion(root, bundleDir)
	if err != nil {
		return nil, "", nil, err
	}
	summary["shared_queue_companion"] = sharedQueueSection
	summary["validation_matrix"] = buildLiveValidationMatrix(summary)

	continuationGate, err := buildLiveValidationContinuationGate(root)
	if err != nil {
		return nil, "", nil, err
	}
	if len(continuationGate) > 0 {
		summary["continuation_gate"] = continuationGate
	}

	bundleSummaryPath := filepath.Join(bundleDir, "summary.json")
	summary["summary_path"] = liveValidationRelPath(root, bundleSummaryPath)
	if err := WriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, "", nil, err
	}
	if err := WriteJSON(resolveLiveValidationPath(root, options.SummaryPath), summary); err != nil {
		return nil, "", nil, err
	}

	recentRuns, err := buildLiveValidationRecentRuns(filepath.Dir(bundleDir), root)
	if err != nil {
		return nil, "", nil, err
	}
	manifest := map[string]any{
		"latest":      summary,
		"recent_runs": recentRuns,
	}
	if len(continuationGate) > 0 {
		manifest["continuation_gate"] = continuationGate
	}
	if err := WriteJSON(resolveLiveValidationPath(root, options.ManifestPath), manifest); err != nil {
		return nil, "", nil, err
	}

	continuationArtifacts := collectLiveValidationArtifacts(root, liveValidationContinuationArtifacts)
	followupDigests := collectLiveValidationArtifacts(root, liveValidationFollowupDigests)
	indexText := renderLiveValidationIndex(summary, recentRuns, continuationGate, continuationArtifacts, followupDigests)
	indexPath := resolveLiveValidationPath(root, options.IndexPath)
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return nil, "", nil, err
	}
	if err := os.WriteFile(indexPath, []byte(indexText), 0o644); err != nil {
		return nil, "", nil, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "README.md"), []byte(indexText), 0o644); err != nil {
		return nil, "", nil, err
	}

	return summary, indexText, manifest, nil
}

type liveValidationComponentInput struct {
	Name       string
	Enabled    bool
	ReportPath string
	StdoutPath string
	StderrPath string
}

func buildLiveValidationComponentSection(root string, bigclawGoRoot string, bundleDir string, input liveValidationComponentInput) (map[string]any, error) {
	reportPath := resolveEvidencePath(root, bigclawGoRoot, input.ReportPath)
	latestReportPath := resolveLiveValidationPath(root, liveValidationLatestReports[input.Name])
	section := map[string]any{
		"enabled":               input.Enabled,
		"bundle_report_path":    liveValidationRelPath(root, reportPath),
		"canonical_report_path": liveValidationLatestReports[input.Name],
	}
	if !input.Enabled {
		section["status"] = "skipped"
		return section, nil
	}

	report, err := readOptionalJSON(reportPath)
	if err != nil {
		return nil, err
	}
	section["report"] = report
	section["status"] = liveValidationComponentStatus(report)

	if copied, err := copyJSONArtifact(reportPath, latestReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		section["canonical_report_path"] = liveValidationRelPath(root, copied)
	}

	if copied, err := copyTextArtifact(resolveEvidencePath(root, bigclawGoRoot, input.StdoutPath), filepath.Join(bundleDir, input.Name+".stdout.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stdout_path"] = liveValidationRelPath(root, copied)
	}
	if copied, err := copyTextArtifact(resolveEvidencePath(root, bigclawGoRoot, input.StderrPath), filepath.Join(bundleDir, input.Name+".stderr.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stderr_path"] = liveValidationRelPath(root, copied)
	}

	if len(report) == 0 {
		section["failure_root_cause"] = map[string]any{
			"status":     "missing_report",
			"event_type": "",
			"message":    "",
			"location":   asString(section["bundle_report_path"]),
			"event_id":   "",
			"timestamp":  "",
		}
		section["validation_matrix"] = buildLiveValidationMatrixEntry(input.Name, section, nil)
		return section, nil
	}

	task := asMap(report["task"])
	if taskID := strings.TrimSpace(asString(task["id"])); taskID != "" {
		section["task_id"] = taskID
	}
	if baseURL := strings.TrimSpace(asString(report["base_url"])); baseURL != "" {
		section["base_url"] = baseURL
	}
	if stateDir := strings.TrimSpace(asString(report["state_dir"])); stateDir != "" {
		section["state_dir"] = stateDir
		auditSource := filepath.Join(stateDir, "audit.jsonl")
		if copied, err := copyTextArtifact(auditSource, filepath.Join(bundleDir, input.Name+".audit.jsonl")); err != nil {
			return nil, err
		} else if copied != "" {
			section["audit_log_path"] = liveValidationRelPath(root, copied)
		}
	}
	if serviceLog := strings.TrimSpace(asString(report["service_log"])); serviceLog != "" {
		if copied, err := copyTextArtifact(serviceLog, filepath.Join(bundleDir, input.Name+".service.log")); err != nil {
			return nil, err
		} else if copied != "" {
			section["service_log_path"] = liveValidationRelPath(root, copied)
		}
	}
	if latestEvent := latestLiveValidationEvent(report); len(latestEvent) > 0 {
		section["latest_event_type"] = firstNonEmptyString(asString(latestEvent["type"]))
		section["latest_event_timestamp"] = firstNonEmptyString(asString(latestEvent["timestamp"]))
		payload := asMap(latestEvent["payload"])
		artifacts := asSlice(payload["artifacts"])
		if len(artifacts) > 0 {
			paths := make([]string, 0, len(artifacts))
			for _, artifact := range artifacts {
				if text := strings.TrimSpace(asString(artifact)); text != "" {
					paths = append(paths, text)
				}
			}
			if len(paths) > 0 {
				section["artifact_paths"] = paths
			}
		}
	}
	if routingReason := findLiveValidationRoutingReason(report); routingReason != "" {
		section["routing_reason"] = routingReason
	}
	section["failure_root_cause"] = buildLiveValidationFailureRootCause(section, report)
	section["validation_matrix"] = buildLiveValidationMatrixEntry(input.Name, section, report)
	return section, nil
}

type liveValidationBrokerInput struct {
	Enabled              bool
	Backend              string
	ReportPath           string
	BootstrapSummaryPath string
}

func buildLiveValidationBrokerSection(root string, bigclawGoRoot string, bundleDir string, input liveValidationBrokerInput) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(bundleDir, "broker-validation-summary.json")
	bundleBootstrapSummaryPath := filepath.Join(bundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                          input.Enabled,
		"backend":                          liveValidationNilIfEmpty(input.Backend),
		"bundle_summary_path":              liveValidationRelPath(root, bundleSummaryPath),
		"canonical_summary_path":           liveValidationBrokerSummary,
		"bundle_bootstrap_summary_path":    liveValidationRelPath(root, bundleBootstrapSummaryPath),
		"canonical_bootstrap_summary_path": liveValidationBrokerBootstrapSummary,
		"validation_pack_path":             liveValidationBrokerValidationPack,
	}
	configurationState := "not_configured"
	if input.Enabled && input.Backend != "" {
		configurationState = "configured"
	}
	section["configuration_state"] = configurationState

	if bootstrapSummaryPath := strings.TrimSpace(input.BootstrapSummaryPath); bootstrapSummaryPath != "" {
		bootstrapPath := resolveEvidencePath(root, bigclawGoRoot, bootstrapSummaryPath)
		bootstrapSummary, err := readOptionalJSON(bootstrapPath)
		if err != nil {
			return nil, err
		}
		if len(bootstrapSummary) > 0 {
			if copied, err := copyJSONArtifact(bootstrapPath, bundleBootstrapSummaryPath); err != nil {
				return nil, err
			} else if copied != "" {
				section["bundle_bootstrap_summary_path"] = liveValidationRelPath(root, copied)
			}
			if copied, err := copyJSONArtifact(bootstrapPath, resolveLiveValidationPath(root, liveValidationBrokerBootstrapSummary)); err != nil {
				return nil, err
			} else if copied != "" {
				section["canonical_bootstrap_summary_path"] = liveValidationRelPath(root, copied)
			}
			section["bootstrap_summary"] = bootstrapSummary
			section["bootstrap_ready"] = asBool(bootstrapSummary["ready"])
			if runtimePosture := bootstrapSummary["runtime_posture"]; runtimePosture != nil {
				section["runtime_posture"] = runtimePosture
			}
			section["live_adapter_implemented"] = asBool(bootstrapSummary["live_adapter_implemented"])
			if proofBoundary := bootstrapSummary["proof_boundary"]; proofBoundary != nil {
				section["proof_boundary"] = proofBoundary
			}
			if validationErrors := asSlice(bootstrapSummary["validation_errors"]); len(validationErrors) > 0 {
				errorsOut := make([]string, 0, len(validationErrors))
				for _, item := range validationErrors {
					if text := strings.TrimSpace(asString(item)); text != "" {
						errorsOut = append(errorsOut, text)
					}
				}
				section["validation_errors"] = errorsOut
			}
			if completeness := asMap(bootstrapSummary["config_completeness"]); len(completeness) > 0 {
				section["config_completeness"] = completeness
			}
		}
	}

	if !input.Enabled || input.Backend == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		return section, writeLiveValidationBrokerSummaries(root, bundleSummaryPath, section)
	}
	if strings.TrimSpace(input.ReportPath) == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		return section, writeLiveValidationBrokerSummaries(root, bundleSummaryPath, section)
	}

	reportPath := resolveEvidencePath(root, bigclawGoRoot, input.ReportPath)
	report, err := readOptionalJSON(reportPath)
	if err != nil {
		return nil, err
	}
	section["canonical_report_path"] = liveValidationRelPath(root, reportPath)
	section["bundle_report_path"] = liveValidationRelPath(root, filepath.Join(bundleDir, filepath.Base(reportPath)))
	if len(report) == 0 {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		return section, writeLiveValidationBrokerSummaries(root, bundleSummaryPath, section)
	}
	if copied, err := copyJSONArtifact(reportPath, filepath.Join(bundleDir, filepath.Base(reportPath))); err != nil {
		return nil, err
	} else if copied != "" {
		section["bundle_report_path"] = liveValidationRelPath(root, copied)
	}
	section["report"] = report
	section["status"] = liveValidationComponentStatus(report)
	return section, writeLiveValidationBrokerSummaries(root, bundleSummaryPath, section)
}

func writeLiveValidationBrokerSummaries(root string, bundleSummaryPath string, section map[string]any) error {
	if err := WriteJSON(bundleSummaryPath, section); err != nil {
		return err
	}
	return WriteJSON(resolveLiveValidationPath(root, liveValidationBrokerSummary), section)
}

func buildLiveValidationSharedQueueCompanion(root string, bundleDir string) (map[string]any, error) {
	canonicalReportPath := resolveLiveValidationPath(root, liveValidationSharedQueueReport)
	canonicalSummaryPath := resolveLiveValidationPath(root, liveValidationSharedQueueSummary)
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")

	report, err := readOptionalJSON(canonicalReportPath)
	if err != nil {
		return nil, err
	}

	summary := map[string]any{
		"available":              len(report) > 0,
		"canonical_report_path":  liveValidationSharedQueueReport,
		"canonical_summary_path": liveValidationSharedQueueSummary,
		"bundle_report_path":     liveValidationRelPath(root, bundleReportPath),
		"bundle_summary_path":    liveValidationRelPath(root, bundleSummaryPath),
	}
	if len(report) == 0 {
		summary["status"] = "missing_report"
		return summary, nil
	}
	if copied, err := copyJSONArtifact(canonicalReportPath, bundleReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		summary["bundle_report_path"] = liveValidationRelPath(root, copied)
	}

	summary["status"] = "failed"
	if asBool(report["all_ok"]) {
		summary["status"] = "succeeded"
	}
	for _, key := range []string{
		"generated_at",
		"count",
		"cross_node_completions",
		"submitted_by_node",
		"completed_by_node",
	} {
		if value, ok := report[key]; ok {
			summary[key] = value
		}
	}
	summary["duplicate_started_tasks"] = len(asSlice(report["duplicate_started_tasks"]))
	summary["duplicate_completed_tasks"] = len(asSlice(report["duplicate_completed_tasks"]))
	summary["missing_completed_tasks"] = len(asSlice(report["missing_completed_tasks"]))
	nodes := make([]string, 0)
	for _, item := range asSlice(report["nodes"]) {
		node := asMap(item)
		if name := strings.TrimSpace(asString(node["name"])); name != "" {
			nodes = append(nodes, name)
		}
	}
	summary["nodes"] = nodes
	if err := WriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := WriteJSON(canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func buildLiveValidationContinuationGate(root string) (map[string]any, error) {
	gatePath := resolveLiveValidationPath(root, "docs/reports/validation-bundle-continuation-policy-gate.json")
	gate, err := readOptionalJSON(gatePath)
	if err != nil {
		return nil, err
	}
	if len(gate) == 0 {
		return nil, nil
	}
	result := map[string]any{
		"path":           liveValidationRelPath(root, gatePath),
		"status":         firstNonEmptyString(asString(gate["status"]), "unknown"),
		"recommendation": firstNonEmptyString(asString(gate["recommendation"]), "unknown"),
		"failing_checks": stringSliceFromAny(gate["failing_checks"]),
		"enforcement":    mapOrEmpty(gate["enforcement"]),
		"summary":        mapOrEmpty(gate["summary"]),
		"reviewer_path":  mapOrEmpty(gate["reviewer_path"]),
		"next_actions":   stringSliceFromAny(gate["next_actions"]),
	}
	return result, nil
}

func buildLiveValidationRecentRuns(bundleRoot string, root string) ([]map[string]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type runSummary struct {
		GeneratedAt string
		Summary     map[string]any
	}
	runs := make([]runSummary, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		summaryPath := filepath.Join(bundleRoot, entry.Name(), "summary.json")
		summary, err := readOptionalJSON(summaryPath)
		if err != nil {
			return nil, err
		}
		if len(summary) == 0 {
			continue
		}
		runs = append(runs, runSummary{GeneratedAt: asString(summary["generated_at"]), Summary: summary})
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].GeneratedAt > runs[j].GeneratedAt
	})
	items := make([]map[string]any, 0, len(runs))
	for idx, run := range runs {
		if idx >= 8 {
			break
		}
		items = append(items, map[string]any{
			"run_id":       asString(run.Summary["run_id"]),
			"generated_at": asString(run.Summary["generated_at"]),
			"status":       firstNonEmptyString(asString(run.Summary["status"]), "unknown"),
			"bundle_path":  asString(run.Summary["bundle_path"]),
			"summary_path": asString(run.Summary["summary_path"]),
		})
	}
	return items, nil
}

func buildLiveValidationMatrix(summary map[string]any) []map[string]any {
	rows := make([]map[string]any, 0, 3)
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		section := asMap(summary[lane])
		row := asMap(section["validation_matrix"])
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func buildLiveValidationMatrixEntry(name string, section map[string]any, report map[string]any) map[string]any {
	task := asMap(report["task"])
	taskID := asString(task["id"])
	if taskID == "" {
		taskID = asString(section["task_id"])
	}
	executor := strings.TrimSpace(asString(task["required_executor"]))
	if executor == "" {
		executor = name
	}
	rootCause := asMap(section["failure_root_cause"])
	return map[string]any{
		"lane":                  firstNonEmptyString(liveValidationLaneAliases[name], name),
		"executor":              executor,
		"enabled":               asBoolWithFallback(section["enabled"], false),
		"status":                firstNonEmptyString(asString(section["status"]), "unknown"),
		"task_id":               taskID,
		"canonical_report_path": asString(section["canonical_report_path"]),
		"bundle_report_path":    asString(section["bundle_report_path"]),
		"latest_event_type":     asString(section["latest_event_type"]),
		"routing_reason":        asString(section["routing_reason"]),
		"root_cause_event_type": asString(rootCause["event_type"]),
		"root_cause_location":   asString(rootCause["location"]),
		"root_cause_message":    asString(rootCause["message"]),
	}
}

func buildLiveValidationFailureRootCause(section map[string]any, report map[string]any) map[string]any {
	events := collectLiveValidationEvents(report)
	latestEvent := latestLiveValidationEvent(report)
	latestStatus := firstNonEmptyString(
		asString(asMap(report["status"])["state"]),
		asString(asMap(report["task"])["state"]),
		liveValidationComponentStatus(report),
	)

	causeEvent := map[string]any{}
	for idx := len(events) - 1; idx >= 0; idx-- {
		eventType := firstNonEmptyString(asString(events[idx]["type"]))
		if _, ok := liveValidationFailureEventTypes[eventType]; ok {
			causeEvent = events[idx]
			break
		}
	}
	if len(causeEvent) == 0 && latestStatus != "" && latestStatus != "succeeded" {
		causeEvent = latestEvent
	}

	location := firstNonEmptyString(
		asString(section["stderr_path"]),
		asString(section["service_log_path"]),
		asString(section["audit_log_path"]),
		asString(section["bundle_report_path"]),
	)
	if len(causeEvent) == 0 {
		return map[string]any{
			"status":     "not_triggered",
			"event_type": asString(latestEvent["type"]),
			"message":    "",
			"location":   location,
			"event_id":   "",
			"timestamp":  "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": firstNonEmptyString(asString(causeEvent["type"])),
		"message": firstNonEmptyString(
			liveValidationEventPayloadText(causeEvent, "message"),
			liveValidationEventPayloadText(causeEvent, "reason"),
			asString(report["error"]),
			asString(report["failure_reason"]),
		),
		"location":  location,
		"event_id":  firstNonEmptyString(asString(causeEvent["id"])),
		"timestamp": firstNonEmptyString(asString(causeEvent["timestamp"])),
	}
}

func collectLiveValidationEvents(report map[string]any) []map[string]any {
	events := make([]map[string]any, 0)
	status := asMap(report["status"])
	if statusEvents := asSlice(status["events"]); len(statusEvents) > 0 {
		for _, item := range statusEvents {
			event := asMap(item)
			if len(event) > 0 {
				events = append(events, event)
			}
		}
	}
	latestEvent := asMap(status["latest_event"])
	if len(latestEvent) > 0 {
		latestID := asString(latestEvent["id"])
		if latestID == "" || !liveValidationEventIDExists(events, latestID) {
			events = append(events, latestEvent)
		}
	}
	for _, item := range asSlice(report["events"]) {
		event := asMap(item)
		if len(event) == 0 {
			continue
		}
		eventID := asString(event["id"])
		if eventID != "" && liveValidationEventIDExists(events, eventID) {
			continue
		}
		events = append(events, event)
	}
	return events
}

func liveValidationEventIDExists(events []map[string]any, eventID string) bool {
	for _, event := range events {
		if asString(event["id"]) == eventID {
			return true
		}
	}
	return false
}

func latestLiveValidationEvent(report map[string]any) map[string]any {
	events := collectLiveValidationEvents(report)
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func findLiveValidationRoutingReason(report map[string]any) string {
	events := collectLiveValidationEvents(report)
	for idx := len(events) - 1; idx >= 0; idx-- {
		if asString(events[idx]["type"]) == "scheduler.routed" {
			return liveValidationEventPayloadText(events[idx], "reason")
		}
	}
	return ""
}

func liveValidationEventPayloadText(event map[string]any, key string) string {
	payload := asMap(event["payload"])
	return firstNonEmptyString(asString(payload[key]))
}

func liveValidationComponentStatus(report map[string]any) string {
	if len(report) == 0 {
		return "missing_report"
	}
	status := report["status"]
	if typed, ok := status.(map[string]any); ok {
		return asString(typed["state"])
	}
	if text := strings.TrimSpace(asString(status)); text != "" {
		return text
	}
	if value, ok := report["all_ok"].(bool); ok {
		if value {
			return "succeeded"
		}
		return "failed"
	}
	return "unknown"
}

func renderLiveValidationIndex(summary map[string]any, recentRuns []map[string]any, continuationGate map[string]any, continuationArtifacts []liveValidationArtifact, followupDigests []liveValidationArtifact) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", asString(summary["run_id"])),
		fmt.Sprintf("- Generated at: `%s`", asString(summary["generated_at"])),
		fmt.Sprintf("- Status: `%s`", asString(summary["status"])),
		fmt.Sprintf("- Bundle: `%s`", asString(summary["bundle_path"])),
		fmt.Sprintf("- Summary JSON: `%s`", asString(summary["summary_path"])),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, lane := range []string{"local", "kubernetes", "ray"} {
		section := asMap(summary[lane])
		matrix := asMap(section["validation_matrix"])
		lines = append(lines, "### "+lane)
		lines = append(lines, fmt.Sprintf("- Enabled: `%t`", asBoolWithFallback(section["enabled"], false)))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", asString(section["status"])))
		if laneValue := asString(matrix["lane"]); laneValue != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", laneValue))
		}
		lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", asString(section["bundle_report_path"])))
		lines = append(lines, fmt.Sprintf("- Latest report: `%s`", asString(section["canonical_report_path"])))
		if text := asString(section["stdout_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Stdout log: `%s`", text))
		}
		if text := asString(section["stderr_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Stderr log: `%s`", text))
		}
		if text := asString(section["service_log_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Service log: `%s`", text))
		}
		if text := asString(section["audit_log_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Audit log: `%s`", text))
		}
		if text := asString(section["task_id"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Task ID: `%s`", text))
		}
		if text := asString(section["latest_event_type"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Latest event: `%s`", text))
		}
		if text := asString(section["routing_reason"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Routing reason: `%s`", text))
		}
		rootCause := asMap(section["failure_root_cause"])
		if len(rootCause) > 0 {
			lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%s` event=`%s` location=`%s`", asString(rootCause["status"]), asString(rootCause["event_type"]), asString(rootCause["location"])))
			if message := asString(rootCause["message"]); message != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", message))
			}
		}
		lines = append(lines, "")
	}

	if matrix := anyToMapSlice(summary["validation_matrix"]); len(matrix) > 0 {
		lines = append(lines, "## Validation matrix", "")
		for _, row := range matrix {
			lines = append(lines, fmt.Sprintf("- Lane `%s` executor=`%s` status=`%s` enabled=`%t` report=`%s`", asString(row["lane"]), asString(row["executor"]), asString(row["status"]), asBoolWithFallback(row["enabled"], false), asString(row["bundle_report_path"])))
			if asString(row["root_cause_event_type"]) != "" || asString(row["root_cause_message"]) != "" {
				lines = append(lines, fmt.Sprintf("- Lane `%s` root cause: event=`%s` location=`%s` message=`%s`", asString(row["lane"]), asString(row["root_cause_event_type"]), asString(row["root_cause_location"]), asString(row["root_cause_message"])))
			}
		}
		lines = append(lines, "")
	}

	if broker := asMap(summary["broker"]); len(broker) > 0 {
		lines = append(lines, "### broker")
		lines = append(lines, fmt.Sprintf("- Enabled: `%t`", asBoolWithFallback(broker["enabled"], false)))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", asString(broker["status"])))
		lines = append(lines, fmt.Sprintf("- Configuration state: `%s`", asString(broker["configuration_state"])))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%s`", asString(broker["bundle_summary_path"])))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%s`", asString(broker["canonical_summary_path"])))
		lines = append(lines, fmt.Sprintf("- Bundle bootstrap summary: `%s`", asString(broker["bundle_bootstrap_summary_path"])))
		lines = append(lines, fmt.Sprintf("- Canonical bootstrap summary: `%s`", asString(broker["canonical_bootstrap_summary_path"])))
		lines = append(lines, fmt.Sprintf("- Validation pack: `%s`", asString(broker["validation_pack_path"])))
		if text := asString(broker["backend"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Backend: `%s`", text))
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%t`", asBoolWithFallback(broker["bootstrap_ready"], false)))
		}
		if text := asString(broker["runtime_posture"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Runtime posture: `%s`", text))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%t`", asBoolWithFallback(broker["live_adapter_implemented"], false)))
		}
		if completeness := asMap(broker["config_completeness"]); len(completeness) > 0 {
			lines = append(lines, fmt.Sprintf("- Config completeness: driver=`%t` urls=`%t` topic=`%t` consumer_group=`%t`", asBoolWithFallback(completeness["driver"], false), asBoolWithFallback(completeness["urls"], false), asBoolWithFallback(completeness["topic"], false), asBoolWithFallback(completeness["consumer_group"], false)))
		}
		if text := asString(broker["proof_boundary"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Proof boundary: `%s`", text))
		}
		for _, item := range stringSliceFromAny(broker["validation_errors"]) {
			lines = append(lines, fmt.Sprintf("- Validation error: `%s`", item))
		}
		if text := asString(broker["bundle_report_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", text))
		}
		if text := asString(broker["canonical_report_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", text))
		}
		if text := asString(broker["reason"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Reason: `%s`", text))
		}
		lines = append(lines, "")
	}

	if sharedQueue := asMap(summary["shared_queue_companion"]); len(sharedQueue) > 0 {
		lines = append(lines, "### shared-queue companion")
		lines = append(lines, fmt.Sprintf("- Available: `%t`", asBoolWithFallback(sharedQueue["available"], false)))
		lines = append(lines, fmt.Sprintf("- Status: `%s`", asString(sharedQueue["status"])))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%s`", asString(sharedQueue["bundle_summary_path"])))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%s`", asString(sharedQueue["canonical_summary_path"])))
		lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", asString(sharedQueue["bundle_report_path"])))
		lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", asString(sharedQueue["canonical_report_path"])))
		if value, ok := sharedQueue["cross_node_completions"]; ok {
			lines = append(lines, fmt.Sprintf("- Cross-node completions: `%d`", asInt(value)))
		}
		if value, ok := sharedQueue["duplicate_started_tasks"]; ok {
			lines = append(lines, fmt.Sprintf("- Duplicate `task.started`: `%d`", asInt(value)))
		}
		if value, ok := sharedQueue["duplicate_completed_tasks"]; ok {
			lines = append(lines, fmt.Sprintf("- Duplicate `task.completed`: `%d`", asInt(value)))
		}
		if value, ok := sharedQueue["missing_completed_tasks"]; ok {
			lines = append(lines, fmt.Sprintf("- Missing terminal completions: `%d`", asInt(value)))
		}
		lines = append(lines, "")
	}

	lines = append(lines, "## Workflow closeout commands", "")
	for _, command := range stringSliceFromAny(summary["closeout_commands"]) {
		lines = append(lines, fmt.Sprintf("- `%s`", command))
	}
	lines = append(lines, "", "## Recent bundles", "")
	if len(recentRuns) == 0 {
		lines = append(lines, "- No previous bundles found")
	} else {
		for _, run := range recentRuns {
			lines = append(lines, fmt.Sprintf("- `%s` · `%s` · `%s` · `%s`", asString(run["run_id"]), asString(run["status"]), asString(run["generated_at"]), asString(run["bundle_path"])))
		}
	}
	lines = append(lines, "")

	if len(continuationGate) > 0 {
		lines = append(lines, "## Continuation gate", "")
		lines = append(lines, fmt.Sprintf("- Status: `%s`", asString(continuationGate["status"])))
		lines = append(lines, fmt.Sprintf("- Recommendation: `%s`", asString(continuationGate["recommendation"])))
		lines = append(lines, fmt.Sprintf("- Report: `%s`", asString(continuationGate["path"])))
		enforcement := asMap(continuationGate["enforcement"])
		if text := asString(enforcement["mode"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Workflow mode: `%s`", text))
		}
		if text := asString(enforcement["outcome"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Workflow outcome: `%s`", text))
		}
		gateSummary := asMap(continuationGate["summary"])
		if text := asString(gateSummary["latest_run_id"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Latest reviewed run: `%s`", text))
		}
		if _, ok := gateSummary["failing_check_count"]; ok {
			lines = append(lines, fmt.Sprintf("- Failing checks: `%d`", asInt(gateSummary["failing_check_count"])))
		}
		if _, ok := gateSummary["workflow_exit_code"]; ok {
			lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%d`", asInt(gateSummary["workflow_exit_code"])))
		}
		reviewerPath := asMap(continuationGate["reviewer_path"])
		if text := asString(reviewerPath["digest_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer digest: `%s`", text))
		}
		if text := asString(reviewerPath["index_path"]); text != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer index: `%s`", text))
		}
		for _, action := range stringSliceFromAny(continuationGate["next_actions"]) {
			lines = append(lines, fmt.Sprintf("- Next action: `%s`", action))
		}
		lines = append(lines, "")
	}
	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, artifact := range continuationArtifacts {
			lines = append(lines, fmt.Sprintf("- `%s` %s", artifact.Path, artifact.Description))
		}
		lines = append(lines, "")
	}
	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, digest := range followupDigests {
			lines = append(lines, fmt.Sprintf("- `%s` %s", digest.Path, digest.Description))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func collectLiveValidationArtifacts(root string, candidates []liveValidationArtifact) []liveValidationArtifact {
	items := make([]liveValidationArtifact, 0, len(candidates))
	for _, item := range candidates {
		if pathExists(resolveLiveValidationPath(root, item.Path)) {
			items = append(items, item)
		}
	}
	return items
}

func copyTextArtifact(source string, destination string) (string, error) {
	if !pathExists(source) {
		return "", nil
	}
	sourceResolved, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	destinationResolved, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if sourceResolved == destinationResolved {
		return destinationResolved, nil
	}
	contents, err := os.ReadFile(sourceResolved)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destinationResolved), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(destinationResolved, contents, 0o644); err != nil {
		return "", err
	}
	return destinationResolved, nil
}

func copyJSONArtifact(source string, destination string) (string, error) {
	if !pathExists(source) {
		return "", nil
	}
	sourceResolved, err := filepath.Abs(source)
	if err != nil {
		return "", err
	}
	destinationResolved, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	if sourceResolved == destinationResolved {
		return destinationResolved, nil
	}
	payload, err := readOptionalJSON(sourceResolved)
	if err != nil {
		return "", err
	}
	if len(payload) == 0 {
		return "", nil
	}
	if err := WriteJSON(destinationResolved, payload); err != nil {
		return "", err
	}
	return destinationResolved, nil
}

func readOptionalJSON(path string) (map[string]any, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if info.Size() == 0 {
		return nil, nil
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(contents, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func resolveLiveValidationPath(root string, value string) string {
	if filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(root, normalizeRepoRelativePath(root, value))
}

func liveValidationRelPath(root string, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return filepath.ToSlash(target)
	}
	return filepath.ToSlash(rel)
}

func formatOffsetTimestamp(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.999999999-07:00")
}

func liveValidationSummaryStatus(status int) string {
	if status == 0 {
		return "succeeded"
	}
	return "failed"
}

func liveValidationNilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func stringSliceFromAny(value any) []string {
	if typed, ok := value.([]string); ok {
		return typed
	}
	items := asSlice(value)
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text := strings.TrimSpace(asString(item)); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func mapOrEmpty(value any) map[string]any {
	return asMap(value)
}

func anyToMapSlice(value any) []map[string]any {
	if typed, ok := value.([]map[string]any); ok {
		return typed
	}
	items := asSlice(value)
	if len(items) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		entry := asMap(item)
		if len(entry) > 0 {
			out = append(out, entry)
		}
	}
	return out
}
