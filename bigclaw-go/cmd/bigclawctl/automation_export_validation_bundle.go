package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	automationBrokerSummary        = "docs/reports/broker-validation-summary.json"
	automationBrokerBootstrap      = "docs/reports/broker-bootstrap-review-summary.json"
	automationBrokerValidationPack = "docs/reports/broker-failover-fault-injection-validation-pack.md"
	automationSharedQueueReport    = "docs/reports/multi-node-shared-queue-report.json"
	automationSharedQueueSummary   = "docs/reports/shared-queue-companion-summary.json"
)

var automationLatestReports = map[string]string{
	"local":      "docs/reports/sqlite-smoke-report.json",
	"kubernetes": "docs/reports/kubernetes-live-smoke-report.json",
	"ray":        "docs/reports/ray-live-smoke-report.json",
}

var automationContinuationArtifacts = []struct {
	Path        string
	Description string
}{
	{
		Path:        "docs/reports/validation-bundle-continuation-scorecard.json",
		Description: "summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof.",
	},
	{
		Path:        "docs/reports/validation-bundle-continuation-policy-gate.json",
		Description: "records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability.",
	},
}

var automationFollowupDigests = []struct {
	Path        string
	Description string
}{
	{
		Path:        "docs/reports/validation-bundle-continuation-digest.md",
		Description: "Validation bundle continuation caveats are consolidated here.",
	},
}

var automationLaneAliases = map[string]string{
	"local":      "local",
	"kubernetes": "k8s",
	"ray":        "ray",
}

var automationFailureEventTypes = map[string]struct{}{
	"task.cancelled":   {},
	"task.dead_letter": {},
	"task.failed":      {},
	"task.retried":     {},
}

type automationExportValidationBundleOptions struct {
	GoRoot                     string
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
	Now                        func() time.Time
}

func runAutomationExportValidationBundleCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e export-validation-bundle", flag.ContinueOnError)
	goRoot := flags.String("go-root", "", "bigclaw-go repo root")
	runID := flags.String("run-id", "", "bundle run id")
	bundleDir := flags.String("bundle-dir", "", "bundle output directory relative to --go-root")
	summaryPath := flags.String("summary-path", "docs/reports/live-validation-summary.json", "canonical summary JSON path")
	indexPath := flags.String("index-path", "docs/reports/live-validation-index.md", "canonical index markdown path")
	manifestPath := flags.String("manifest-path", "docs/reports/live-validation-index.json", "manifest JSON path")
	runLocal := flags.String("run-local", "1", "whether the local lane was enabled (1|0)")
	runKubernetes := flags.String("run-kubernetes", "1", "whether the kubernetes lane was enabled (1|0)")
	runRay := flags.String("run-ray", "1", "whether the ray lane was enabled (1|0)")
	validationStatus := flags.Int("validation-status", 0, "overall validation exit status")
	runBroker := flags.String("run-broker", "0", "whether broker validation was enabled (1|0)")
	brokerBackend := flags.String("broker-backend", "", "broker backend name")
	brokerReportPath := flags.String("broker-report-path", "", "broker report path relative to --go-root")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", "", "broker bootstrap summary path relative to --go-root")
	localReportPath := flags.String("local-report-path", "", "local lane report path")
	localStdoutPath := flags.String("local-stdout-path", "", "local lane stdout log path")
	localStderrPath := flags.String("local-stderr-path", "", "local lane stderr log path")
	k8sReportPath := flags.String("kubernetes-report-path", "", "kubernetes lane report path")
	k8sStdoutPath := flags.String("kubernetes-stdout-path", "", "kubernetes lane stdout log path")
	k8sStderrPath := flags.String("kubernetes-stderr-path", "", "kubernetes lane stderr log path")
	rayReportPath := flags.String("ray-report-path", "", "ray lane report path")
	rayStdoutPath := flags.String("ray-stdout-path", "", "ray lane stdout log path")
	rayStderrPath := flags.String("ray-stderr-path", "", "ray lane stderr log path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e export-validation-bundle [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	required := map[string]string{
		"--go-root":                trim(*goRoot),
		"--run-id":                 trim(*runID),
		"--bundle-dir":             trim(*bundleDir),
		"--local-report-path":      trim(*localReportPath),
		"--local-stdout-path":      trim(*localStdoutPath),
		"--local-stderr-path":      trim(*localStderrPath),
		"--kubernetes-report-path": trim(*k8sReportPath),
		"--kubernetes-stdout-path": trim(*k8sStdoutPath),
		"--kubernetes-stderr-path": trim(*k8sStderrPath),
		"--ray-report-path":        trim(*rayReportPath),
		"--ray-stdout-path":        trim(*rayStdoutPath),
		"--ray-stderr-path":        trim(*rayStderrPath),
	}
	for name, value := range required {
		if value == "" {
			return fmt.Errorf("%s is required", name)
		}
	}
	report, exitCode, err := automationExportValidationBundle(automationExportValidationBundleOptions{
		GoRoot:                     absPath(*goRoot),
		RunID:                      trim(*runID),
		BundleDir:                  trim(*bundleDir),
		SummaryPath:                trim(*summaryPath),
		IndexPath:                  trim(*indexPath),
		ManifestPath:               trim(*manifestPath),
		RunLocal:                   *runLocal == "1",
		RunKubernetes:              *runKubernetes == "1",
		RunRay:                     *runRay == "1",
		ValidationStatus:           *validationStatus,
		RunBroker:                  *runBroker == "1",
		BrokerBackend:              trim(*brokerBackend),
		BrokerReportPath:           trim(*brokerReportPath),
		BrokerBootstrapSummaryPath: trim(*brokerBootstrapSummaryPath),
		LocalReportPath:            trim(*localReportPath),
		LocalStdoutPath:            trim(*localStdoutPath),
		LocalStderrPath:            trim(*localStderrPath),
		KubernetesReportPath:       trim(*k8sReportPath),
		KubernetesStdoutPath:       trim(*k8sStdoutPath),
		KubernetesStderrPath:       trim(*k8sStderrPath),
		RayReportPath:              trim(*rayReportPath),
		RayStdoutPath:              trim(*rayStdoutPath),
		RayStderrPath:              trim(*rayStderrPath),
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	return err
}

func automationExportValidationBundle(opts automationExportValidationBundleOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	root := opts.GoRoot
	bundleDir := filepath.Join(root, opts.BundleDir)
	if filepath.IsAbs(opts.BundleDir) {
		bundleDir = opts.BundleDir
	}
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return nil, 0, err
	}

	summary := map[string]any{
		"run_id":       opts.RunID,
		"generated_at": now().UTC().Format(time.RFC3339Nano),
		"status":       automationStatusFromExitCode(opts.ValidationStatus),
		"bundle_path":  automationRelPath(bundleDir, root),
		"closeout_commands": []any{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}

	localSection, err := automationBuildComponentSection("local", opts.RunLocal, root, bundleDir, filepath.Join(root, opts.LocalReportPath), opts.LocalStdoutPath, opts.LocalStderrPath)
	if err != nil {
		return nil, 0, err
	}
	k8sSection, err := automationBuildComponentSection("kubernetes", opts.RunKubernetes, root, bundleDir, filepath.Join(root, opts.KubernetesReportPath), opts.KubernetesStdoutPath, opts.KubernetesStderrPath)
	if err != nil {
		return nil, 0, err
	}
	raySection, err := automationBuildComponentSection("ray", opts.RunRay, root, bundleDir, filepath.Join(root, opts.RayReportPath), opts.RayStdoutPath, opts.RayStderrPath)
	if err != nil {
		return nil, 0, err
	}
	brokerSection, err := automationBuildBrokerSection(opts.RunBroker, opts.BrokerBackend, root, bundleDir, opts.BrokerBootstrapSummaryPath, opts.BrokerReportPath)
	if err != nil {
		return nil, 0, err
	}
	sharedQueue, err := automationBuildSharedQueueCompanion(root, bundleDir)
	if err != nil {
		return nil, 0, err
	}

	summary["local"] = localSection
	summary["kubernetes"] = k8sSection
	summary["ray"] = raySection
	summary["broker"] = brokerSection
	summary["shared_queue_companion"] = sharedQueue
	summary["validation_matrix"] = automationBuildValidationMatrix(summary)

	continuationGate := automationBuildContinuationGateSummary(root)
	if continuationGate != nil {
		summary["continuation_gate"] = continuationGate
	}

	bundleSummaryPath := filepath.Join(bundleDir, "summary.json")
	summary["summary_path"] = automationRelPath(bundleSummaryPath, root)
	if err := automationWriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, 0, err
	}
	if err := automationWriteJSON(filepath.Join(root, opts.SummaryPath), summary); err != nil {
		return nil, 0, err
	}

	recentRuns := automationBuildRecentRuns(filepath.Dir(bundleDir))
	manifest := map[string]any{
		"latest":      summary,
		"recent_runs": recentRuns,
	}
	if continuationGate != nil {
		manifest["continuation_gate"] = continuationGate
	}
	if err := automationWriteJSON(filepath.Join(root, opts.ManifestPath), manifest); err != nil {
		return nil, 0, err
	}

	indexText := automationRenderIndex(summary, recentRuns, continuationGate, automationExistingArtifacts(root, automationContinuationArtifacts), automationExistingArtifacts(root, automationFollowupDigests))
	if err := automationWriteText(filepath.Join(root, opts.IndexPath), indexText); err != nil {
		return nil, 0, err
	}
	if err := automationWriteText(filepath.Join(bundleDir, "README.md"), indexText); err != nil {
		return nil, 0, err
	}
	return summary, opts.ValidationStatus, nil
}

func automationStatusFromExitCode(code int) string {
	if code == 0 {
		return "succeeded"
	}
	return "failed"
}

func automationWriteJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func automationWriteText(path string, body string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}

func automationReadJSON(path string) any {
	body, err := os.ReadFile(path)
	if err != nil || len(body) == 0 {
		return nil
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return payload
}

func automationRelPath(path string, root string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return filepath.ToSlash(rel)
}

func automationCopyTextArtifact(source string, destination string) string {
	body, err := os.ReadFile(source)
	if err != nil {
		return ""
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return destination
	}
	if err := automationWriteText(destination, string(body)); err != nil {
		return ""
	}
	return destination
}

func automationCopyJSONArtifact(source string, destination string) string {
	payload := automationReadJSON(source)
	if payload == nil {
		return ""
	}
	if filepath.Clean(source) == filepath.Clean(destination) {
		return destination
	}
	if err := automationWriteJSON(destination, payload); err != nil {
		return ""
	}
	return destination
}

func automationFirstText(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok && trim(text) != "" {
			return trim(text)
		}
	}
	return ""
}

func automationCollectReportEvents(report map[string]any) []map[string]any {
	events := []map[string]any{}
	status, _ := report["status"].(map[string]any)
	if status != nil {
		if statusEvents, ok := status["events"].([]any); ok {
			for _, item := range statusEvents {
				if event, ok := item.(map[string]any); ok {
					events = append(events, event)
				}
			}
		}
		if latest, ok := status["latest_event"].(map[string]any); ok {
			id, _ := latest["id"].(string)
			if id == "" || !automationHasEventID(events, id) {
				events = append(events, latest)
			}
		}
	}
	if reportEvents, ok := report["events"].([]any); ok {
		for _, item := range reportEvents {
			event, ok := item.(map[string]any)
			if !ok {
				continue
			}
			id, _ := event["id"].(string)
			if id != "" && automationHasEventID(events, id) {
				continue
			}
			events = append(events, event)
		}
	}
	return events
}

func automationHasEventID(events []map[string]any, id string) bool {
	for _, event := range events {
		if eventID, _ := event["id"].(string); eventID == id {
			return true
		}
	}
	return false
}

func automationLatestReportEvent(report map[string]any) map[string]any {
	events := automationCollectReportEvents(report)
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func automationEventPayloadText(event map[string]any, key string) string {
	payload, _ := event["payload"].(map[string]any)
	if payload == nil {
		return ""
	}
	return automationFirstText(payload[key])
}

func automationFindRoutingReason(report map[string]any) string {
	events := automationCollectReportEvents(report)
	for i := len(events) - 1; i >= 0; i-- {
		if eventType, _ := events[i]["type"].(string); eventType == "scheduler.routed" {
			return automationEventPayloadText(events[i], "reason")
		}
	}
	return ""
}

func automationComponentStatus(report map[string]any) string {
	if report == nil {
		return "missing_report"
	}
	if status, ok := report["status"].(map[string]any); ok {
		return automationFirstText(status["state"])
	}
	if status, ok := report["status"].(string); ok && trim(status) != "" {
		return trim(status)
	}
	if allOK, ok := report["all_ok"].(bool); ok {
		if allOK {
			return "succeeded"
		}
		return "failed"
	}
	return "unknown"
}

func automationBuildFailureRootCause(section map[string]any, report map[string]any) map[string]any {
	events := automationCollectReportEvents(report)
	latestEvent := automationLatestReportEvent(report)
	latestStatus := automationFirstText(
		func() any {
			if status, ok := report["status"].(map[string]any); ok {
				return status["state"]
			}
			return nil
		}(),
		func() any {
			if task, ok := report["task"].(map[string]any); ok {
				return task["state"]
			}
			return nil
		}(),
		automationComponentStatus(report),
	)

	var causeEvent map[string]any
	for i := len(events) - 1; i >= 0; i-- {
		if eventType, _ := events[i]["type"].(string); eventType != "" {
			if _, found := automationFailureEventTypes[eventType]; found {
				causeEvent = events[i]
				break
			}
		}
	}
	if causeEvent == nil && latestStatus != "" && latestStatus != "succeeded" {
		causeEvent = latestEvent
	}

	location := automationFirstText(section["stderr_path"], section["service_log_path"], section["audit_log_path"], section["bundle_report_path"])
	if causeEvent == nil {
		return map[string]any{
			"status":     "not_triggered",
			"event_type": automationFirstText(latestEvent["type"]),
			"message":    "",
			"location":   location,
			"event_id":   "",
			"timestamp":  "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": automationFirstText(causeEvent["type"]),
		"message": automationFirstText(
			automationEventPayloadText(causeEvent, "message"),
			automationEventPayloadText(causeEvent, "reason"),
			report["error"],
			report["failure_reason"],
		),
		"location":  location,
		"event_id":  automationFirstText(causeEvent["id"]),
		"timestamp": automationFirstText(causeEvent["timestamp"]),
	}
}

func automationBuildValidationMatrixEntry(name string, section map[string]any, report map[string]any) map[string]any {
	var taskID string
	var executor string
	if task, ok := report["task"].(map[string]any); ok {
		taskID = automationFirstText(task["id"])
		executor = automationFirstText(task["required_executor"])
	}
	if executor == "" {
		executor = name
	}
	rootCause, _ := section["failure_root_cause"].(map[string]any)
	return map[string]any{
		"lane":                  automationLaneAliases[name],
		"executor":              executor,
		"enabled":               section["enabled"],
		"status":                section["status"],
		"task_id":               taskID,
		"canonical_report_path": section["canonical_report_path"],
		"bundle_report_path":    section["bundle_report_path"],
		"latest_event_type":     section["latest_event_type"],
		"routing_reason":        section["routing_reason"],
		"root_cause_event_type": rootCause["event_type"],
		"root_cause_location":   rootCause["location"],
		"root_cause_message":    rootCause["message"],
	}
}

func automationBuildValidationMatrix(summary map[string]any) []any {
	rows := []any{}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		if row, ok := section["validation_matrix"].(map[string]any); ok {
			rows = append(rows, row)
		}
	}
	return rows
}

func automationBuildSharedQueueCompanion(root string, bundleDir string) (map[string]any, error) {
	canonicalReportPath := filepath.Join(root, automationSharedQueueReport)
	canonicalSummaryPath := filepath.Join(root, automationSharedQueueSummary)
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")
	report, _ := automationReadJSON(canonicalReportPath).(map[string]any)
	summary := map[string]any{
		"available":              report != nil,
		"canonical_report_path":  automationSharedQueueReport,
		"canonical_summary_path": automationSharedQueueSummary,
		"bundle_report_path":     automationRelPath(bundleReportPath, root),
		"bundle_summary_path":    automationRelPath(bundleSummaryPath, root),
	}
	if report == nil {
		summary["status"] = "missing_report"
		return summary, nil
	}
	if copied := automationCopyJSONArtifact(canonicalReportPath, bundleReportPath); copied != "" {
		summary["bundle_report_path"] = automationRelPath(copied, root)
	}
	summary["status"] = "failed"
	if allOK, _ := report["all_ok"].(bool); allOK {
		summary["status"] = "succeeded"
	}
	summary["generated_at"] = report["generated_at"]
	summary["count"] = report["count"]
	summary["cross_node_completions"] = report["cross_node_completions"]
	summary["duplicate_started_tasks"] = automationListLen(report["duplicate_started_tasks"])
	summary["duplicate_completed_tasks"] = automationListLen(report["duplicate_completed_tasks"])
	summary["missing_completed_tasks"] = automationListLen(report["missing_completed_tasks"])
	summary["submitted_by_node"] = report["submitted_by_node"]
	summary["completed_by_node"] = report["completed_by_node"]
	summary["nodes"] = automationNodeNames(report["nodes"])
	if err := automationWriteJSON(bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := automationWriteJSON(canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func automationListLen(value any) int {
	items, _ := value.([]any)
	return len(items)
}

func automationNodeNames(value any) []any {
	items, _ := value.([]any)
	names := make([]any, 0, len(items))
	for _, item := range items {
		node, _ := item.(map[string]any)
		if name, ok := node["name"].(string); ok && trim(name) != "" {
			names = append(names, name)
		}
	}
	return names
}

func automationBuildComponentSection(name string, enabled bool, root string, bundleDir string, reportPath string, stdoutPath string, stderrPath string) (map[string]any, error) {
	latestReportPath := filepath.Join(root, automationLatestReports[name])
	section := map[string]any{
		"enabled":               enabled,
		"bundle_report_path":    automationRelPath(reportPath, root),
		"canonical_report_path": automationLatestReports[name],
	}
	if !enabled {
		section["status"] = "skipped"
		return section, nil
	}
	report, _ := automationReadJSON(reportPath).(map[string]any)
	section["report"] = report
	section["status"] = automationComponentStatus(report)
	if copied := automationCopyJSONArtifact(reportPath, latestReportPath); copied != "" {
		section["canonical_report_path"] = automationRelPath(copied, root)
	}
	if copied := automationCopyTextArtifact(stdoutPath, filepath.Join(bundleDir, name+".stdout.log")); copied != "" {
		section["stdout_path"] = automationRelPath(copied, root)
	}
	if copied := automationCopyTextArtifact(stderrPath, filepath.Join(bundleDir, name+".stderr.log")); copied != "" {
		section["stderr_path"] = automationRelPath(copied, root)
	}
	if report == nil {
		section["failure_root_cause"] = map[string]any{
			"status":     "missing_report",
			"event_type": "",
			"message":    "",
			"location":   section["bundle_report_path"],
			"event_id":   "",
			"timestamp":  "",
		}
		section["validation_matrix"] = automationBuildValidationMatrixEntry(name, section, map[string]any{})
		return section, nil
	}
	task, _ := report["task"].(map[string]any)
	if taskID := automationFirstText(task["id"]); taskID != "" {
		section["task_id"] = taskID
	}
	if baseURL := automationFirstText(report["base_url"]); baseURL != "" {
		section["base_url"] = baseURL
	}
	if stateDir := automationFirstText(report["state_dir"]); stateDir != "" {
		section["state_dir"] = stateDir
		if copied := automationCopyTextArtifact(filepath.Join(stateDir, "audit.jsonl"), filepath.Join(bundleDir, name+".audit.jsonl")); copied != "" {
			section["audit_log_path"] = automationRelPath(copied, root)
		}
	}
	if serviceLog := automationFirstText(report["service_log"]); serviceLog != "" {
		if copied := automationCopyTextArtifact(serviceLog, filepath.Join(bundleDir, name+".service.log")); copied != "" {
			section["service_log_path"] = automationRelPath(copied, root)
		}
	}
	if latestEvent := automationLatestReportEvent(report); latestEvent != nil {
		section["latest_event_type"] = automationFirstText(latestEvent["type"])
		section["latest_event_timestamp"] = automationFirstText(latestEvent["timestamp"])
		if payload, ok := latestEvent["payload"].(map[string]any); ok {
			if artifacts, ok := payload["artifacts"].([]any); ok {
				filtered := make([]any, 0, len(artifacts))
				for _, artifact := range artifacts {
					if path, ok := artifact.(string); ok {
						filtered = append(filtered, path)
					}
				}
				section["artifact_paths"] = filtered
			}
		}
	}
	if reason := automationFindRoutingReason(report); reason != "" {
		section["routing_reason"] = reason
	}
	section["failure_root_cause"] = automationBuildFailureRootCause(section, report)
	section["validation_matrix"] = automationBuildValidationMatrixEntry(name, section, report)
	return section, nil
}

func automationBuildBrokerSection(enabled bool, backend string, root string, bundleDir string, bootstrapSummaryRelPath string, reportRelPath string) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(bundleDir, "broker-validation-summary.json")
	bundleBootstrapPath := filepath.Join(bundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                          enabled,
		"backend":                          nil,
		"bundle_summary_path":              automationRelPath(bundleSummaryPath, root),
		"canonical_summary_path":           automationBrokerSummary,
		"bundle_bootstrap_summary_path":    automationRelPath(bundleBootstrapPath, root),
		"canonical_bootstrap_summary_path": automationBrokerBootstrap,
		"validation_pack_path":             automationBrokerValidationPack,
	}
	if trim(backend) != "" {
		section["backend"] = backend
	}
	if enabled && trim(backend) != "" {
		section["configuration_state"] = "configured"
	} else {
		section["configuration_state"] = "not_configured"
	}
	var bootstrapSummary map[string]any
	if trim(bootstrapSummaryRelPath) != "" {
		bootstrapSummary, _ = automationReadJSON(filepath.Join(root, bootstrapSummaryRelPath)).(map[string]any)
	}
	if bootstrapSummary != nil {
		if copied := automationCopyJSONArtifact(filepath.Join(root, bootstrapSummaryRelPath), bundleBootstrapPath); copied != "" {
			section["bundle_bootstrap_summary_path"] = automationRelPath(copied, root)
		}
		if copied := automationCopyJSONArtifact(filepath.Join(root, bootstrapSummaryRelPath), filepath.Join(root, automationBrokerBootstrap)); copied != "" {
			section["canonical_bootstrap_summary_path"] = automationRelPath(copied, root)
		}
		section["bootstrap_summary"] = bootstrapSummary
		section["bootstrap_ready"] = bootstrapSummary["ready"]
		section["runtime_posture"] = bootstrapSummary["runtime_posture"]
		section["live_adapter_implemented"] = bootstrapSummary["live_adapter_implemented"]
		section["proof_boundary"] = bootstrapSummary["proof_boundary"]
		if validationErrors, ok := bootstrapSummary["validation_errors"].([]any); ok {
			section["validation_errors"] = validationErrors
		}
		if completeness, ok := bootstrapSummary["config_completeness"].(map[string]any); ok {
			section["config_completeness"] = completeness
		}
	}
	if !enabled || trim(backend) == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := automationWriteJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := automationWriteJSON(filepath.Join(root, automationBrokerSummary), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if trim(reportRelPath) == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		if err := automationWriteJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := automationWriteJSON(filepath.Join(root, automationBrokerSummary), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	reportPath := filepath.Join(root, reportRelPath)
	report, _ := automationReadJSON(reportPath).(map[string]any)
	section["canonical_report_path"] = automationRelPath(reportPath, root)
	section["bundle_report_path"] = automationRelPath(filepath.Join(bundleDir, filepath.Base(reportPath)), root)
	if report == nil {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := automationWriteJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := automationWriteJSON(filepath.Join(root, automationBrokerSummary), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if copied := automationCopyJSONArtifact(reportPath, filepath.Join(bundleDir, filepath.Base(reportPath))); copied != "" {
		section["bundle_report_path"] = automationRelPath(copied, root)
	}
	section["report"] = report
	section["status"] = automationComponentStatus(report)
	if err := automationWriteJSON(bundleSummaryPath, section); err != nil {
		return nil, err
	}
	if err := automationWriteJSON(filepath.Join(root, automationBrokerSummary), section); err != nil {
		return nil, err
	}
	return section, nil
}

func automationBuildRecentRuns(bundleRoot string) []any {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		return nil
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
		summary, _ := automationReadJSON(filepath.Join(bundleRoot, entry.Name(), "summary.json")).(map[string]any)
		if summary == nil {
			continue
		}
		runs = append(runs, runSummary{GeneratedAt: automationFirstText(summary["generated_at"]), Summary: summary})
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].GeneratedAt > runs[j].GeneratedAt
	})
	items := []any{}
	for i, run := range runs {
		if i >= 8 {
			break
		}
		items = append(items, map[string]any{
			"run_id":       run.Summary["run_id"],
			"generated_at": run.Summary["generated_at"],
			"status":       run.Summary["status"],
			"bundle_path":  run.Summary["bundle_path"],
			"summary_path": run.Summary["summary_path"],
		})
	}
	return items
}

func automationBuildContinuationGateSummary(root string) map[string]any {
	gate, _ := automationReadJSON(filepath.Join(root, "docs/reports/validation-bundle-continuation-policy-gate.json")).(map[string]any)
	if gate == nil {
		return nil
	}
	summary, _ := gate["summary"].(map[string]any)
	enforcement, _ := gate["enforcement"].(map[string]any)
	reviewerPath, _ := gate["reviewer_path"].(map[string]any)
	nextActions, _ := gate["next_actions"].([]any)
	failingChecks, _ := gate["failing_checks"].([]any)
	return map[string]any{
		"path":           "docs/reports/validation-bundle-continuation-policy-gate.json",
		"status":         automationFirstText(gate["status"]),
		"recommendation": automationFirstText(gate["recommendation"]),
		"failing_checks": failingChecks,
		"enforcement":    enforcement,
		"summary":        summary,
		"reviewer_path":  reviewerPath,
		"next_actions":   nextActions,
	}
}

func automationExistingArtifacts(root string, items []struct {
	Path        string
	Description string
}) []any {
	artifacts := []any{}
	for _, item := range items {
		if _, err := os.Stat(filepath.Join(root, item.Path)); err == nil {
			artifacts = append(artifacts, map[string]any{
				"path":        item.Path,
				"description": item.Description,
			})
		}
	}
	return artifacts
}

func automationRenderIndex(summary map[string]any, recentRuns []any, continuationGate map[string]any, continuationArtifacts []any, followupDigests []any) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%s`", automationFirstText(summary["run_id"])),
		fmt.Sprintf("- Generated at: `%s`", automationFirstText(summary["generated_at"])),
		fmt.Sprintf("- Status: `%s`", automationFirstText(summary["status"])),
		fmt.Sprintf("- Bundle: `%s`", automationFirstText(summary["bundle_path"])),
		fmt.Sprintf("- Summary JSON: `%s`", automationFirstText(summary["summary_path"])),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section, _ := summary[name].(map[string]any)
		lines = append(lines, "### "+name)
		matrix, _ := section["validation_matrix"].(map[string]any)
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", section["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", section["status"]))
		if lane := automationFirstText(matrix["lane"]); lane != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", lane))
		}
		lines = append(lines, fmt.Sprintf("- Bundle report: `%v`", section["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Latest report: `%v`", section["canonical_report_path"]))
		if path := automationFirstText(section["stdout_path"]); path != "" {
			lines = append(lines, fmt.Sprintf("- Stdout log: `%s`", path))
		}
		if path := automationFirstText(section["stderr_path"]); path != "" {
			lines = append(lines, fmt.Sprintf("- Stderr log: `%s`", path))
		}
		if path := automationFirstText(section["service_log_path"]); path != "" {
			lines = append(lines, fmt.Sprintf("- Service log: `%s`", path))
		}
		if path := automationFirstText(section["audit_log_path"]); path != "" {
			lines = append(lines, fmt.Sprintf("- Audit log: `%s`", path))
		}
		if taskID := automationFirstText(section["task_id"]); taskID != "" {
			lines = append(lines, fmt.Sprintf("- Task ID: `%s`", taskID))
		}
		if eventType := automationFirstText(section["latest_event_type"]); eventType != "" {
			lines = append(lines, fmt.Sprintf("- Latest event: `%s`", eventType))
		}
		if reason := automationFirstText(section["routing_reason"]); reason != "" {
			lines = append(lines, fmt.Sprintf("- Routing reason: `%s`", reason))
		}
		if rootCause, ok := section["failure_root_cause"].(map[string]any); ok {
			lines = append(lines, fmt.Sprintf("- Failure root cause: status=`%v` event=`%v` location=`%v`", rootCause["status"], rootCause["event_type"], rootCause["location"]))
			if message := automationFirstText(rootCause["message"]); message != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", message))
			}
		}
		lines = append(lines, "")
	}
	if len(recentRuns) >= 0 {
		if validationMatrix, ok := summary["validation_matrix"].([]any); ok && len(validationMatrix) > 0 {
			lines = append(lines, "## Validation matrix", "")
			for _, item := range validationMatrix {
				row, _ := item.(map[string]any)
				lines = append(lines, fmt.Sprintf("- Lane `%v` executor=`%v` status=`%v` enabled=`%v` report=`%v`", row["lane"], row["executor"], row["status"], row["enabled"], row["bundle_report_path"]))
				if automationFirstText(row["root_cause_event_type"], row["root_cause_message"]) != "" {
					lines = append(lines, fmt.Sprintf("- Lane `%v` root cause: event=`%v` location=`%v` message=`%v`", row["lane"], row["root_cause_event_type"], row["root_cause_location"], row["root_cause_message"]))
				}
			}
			lines = append(lines, "")
		}
	}
	if broker, ok := summary["broker"].(map[string]any); ok {
		lines = append(lines, "### broker")
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", broker["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", broker["status"]))
		lines = append(lines, fmt.Sprintf("- Configuration state: `%v`", broker["configuration_state"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%v`", broker["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%v`", broker["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle bootstrap summary: `%v`", broker["bundle_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical bootstrap summary: `%v`", broker["canonical_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Validation pack: `%v`", broker["validation_pack_path"]))
		if backend := automationFirstText(broker["backend"]); backend != "" {
			lines = append(lines, fmt.Sprintf("- Backend: `%s`", backend))
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%v`", broker["bootstrap_ready"]))
		}
		if posture := automationFirstText(broker["runtime_posture"]); posture != "" {
			lines = append(lines, fmt.Sprintf("- Runtime posture: `%s`", posture))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%v`", broker["live_adapter_implemented"]))
		}
		if completeness, ok := broker["config_completeness"].(map[string]any); ok {
			lines = append(lines, fmt.Sprintf("- Config completeness: driver=`%v` urls=`%v` topic=`%v` consumer_group=`%v`", completeness["driver"], completeness["urls"], completeness["topic"], completeness["consumer_group"]))
		}
		if proofBoundary := automationFirstText(broker["proof_boundary"]); proofBoundary != "" {
			lines = append(lines, fmt.Sprintf("- Proof boundary: `%s`", proofBoundary))
		}
		if validationErrors, ok := broker["validation_errors"].([]any); ok {
			for _, item := range validationErrors {
				lines = append(lines, fmt.Sprintf("- Validation error: `%v`", item))
			}
		}
		if path := automationFirstText(broker["bundle_report_path"]); path != "" {
			lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", path))
		}
		if path := automationFirstText(broker["canonical_report_path"]); path != "" {
			lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", path))
		}
		if reason := automationFirstText(broker["reason"]); reason != "" {
			lines = append(lines, fmt.Sprintf("- Reason: `%s`", reason))
		}
		lines = append(lines, "")
	}
	if sharedQueue, ok := summary["shared_queue_companion"].(map[string]any); ok {
		lines = append(lines, "### shared-queue companion")
		lines = append(lines, fmt.Sprintf("- Available: `%v`", sharedQueue["available"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", sharedQueue["status"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%v`", sharedQueue["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%v`", sharedQueue["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle report: `%v`", sharedQueue["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical report: `%v`", sharedQueue["canonical_report_path"]))
		if _, ok := sharedQueue["cross_node_completions"]; ok {
			lines = append(lines, fmt.Sprintf("- Cross-node completions: `%v`", sharedQueue["cross_node_completions"]))
		}
		if _, ok := sharedQueue["duplicate_started_tasks"]; ok {
			lines = append(lines, fmt.Sprintf("- Duplicate `task.started`: `%v`", sharedQueue["duplicate_started_tasks"]))
		}
		if _, ok := sharedQueue["duplicate_completed_tasks"]; ok {
			lines = append(lines, fmt.Sprintf("- Duplicate `task.completed`: `%v`", sharedQueue["duplicate_completed_tasks"]))
		}
		if _, ok := sharedQueue["missing_completed_tasks"]; ok {
			lines = append(lines, fmt.Sprintf("- Missing terminal completions: `%v`", sharedQueue["missing_completed_tasks"]))
		}
		lines = append(lines, "")
	}
	lines = append(lines, "## Workflow closeout commands", "")
	if commands, ok := summary["closeout_commands"].([]any); ok {
		for _, command := range commands {
			lines = append(lines, fmt.Sprintf("- `%v`", command))
		}
	}
	lines = append(lines, "", "## Recent bundles", "")
	if len(recentRuns) == 0 {
		lines = append(lines, "- No previous bundles found")
	} else {
		for _, item := range recentRuns {
			run, _ := item.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%v` · `%v` · `%v` · `%v`", run["run_id"], run["status"], run["generated_at"], run["bundle_path"]))
		}
	}
	lines = append(lines, "")
	if continuationGate != nil {
		lines = append(lines, "## Continuation gate", "")
		lines = append(lines, fmt.Sprintf("- Status: `%v`", continuationGate["status"]))
		lines = append(lines, fmt.Sprintf("- Recommendation: `%v`", continuationGate["recommendation"]))
		lines = append(lines, fmt.Sprintf("- Report: `%v`", continuationGate["path"]))
		if enforcement, ok := continuationGate["enforcement"].(map[string]any); ok {
			if mode := automationFirstText(enforcement["mode"]); mode != "" {
				lines = append(lines, fmt.Sprintf("- Workflow mode: `%s`", mode))
			}
			if outcome := automationFirstText(enforcement["outcome"]); outcome != "" {
				lines = append(lines, fmt.Sprintf("- Workflow outcome: `%s`", outcome))
			}
		}
		if gateSummary, ok := continuationGate["summary"].(map[string]any); ok {
			if latestRunID := automationFirstText(gateSummary["latest_run_id"]); latestRunID != "" {
				lines = append(lines, fmt.Sprintf("- Latest reviewed run: `%s`", latestRunID))
			}
			if value, ok := gateSummary["failing_check_count"]; ok {
				lines = append(lines, fmt.Sprintf("- Failing checks: `%v`", value))
			}
			if value, ok := gateSummary["workflow_exit_code"]; ok {
				lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", value))
			}
		}
		if reviewerPath, ok := continuationGate["reviewer_path"].(map[string]any); ok {
			if digestPath := automationFirstText(reviewerPath["digest_path"]); digestPath != "" {
				lines = append(lines, fmt.Sprintf("- Reviewer digest: `%s`", digestPath))
			}
			if indexPath := automationFirstText(reviewerPath["index_path"]); indexPath != "" {
				lines = append(lines, fmt.Sprintf("- Reviewer index: `%s`", indexPath))
			}
		}
		if nextActions, ok := continuationGate["next_actions"].([]any); ok {
			for _, action := range nextActions {
				lines = append(lines, fmt.Sprintf("- Next action: `%v`", action))
			}
		}
		lines = append(lines, "")
	}
	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, item := range continuationArtifacts {
			artifact, _ := item.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%v` %v", artifact["path"], artifact["description"]))
		}
		lines = append(lines, "")
	}
	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, item := range followupDigests {
			digest, _ := item.(map[string]any)
			lines = append(lines, fmt.Sprintf("- `%v` %v", digest["path"], digest["description"]))
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
