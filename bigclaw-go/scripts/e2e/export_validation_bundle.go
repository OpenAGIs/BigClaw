package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var latestReports = map[string]string{
	"local":      "docs/reports/sqlite-smoke-report.json",
	"kubernetes": "docs/reports/kubernetes-live-smoke-report.json",
	"ray":        "docs/reports/ray-live-smoke-report.json",
}

const (
	brokerSummaryPath            = "docs/reports/broker-validation-summary.json"
	brokerBootstrapSummaryPath   = "docs/reports/broker-bootstrap-review-summary.json"
	brokerValidationPackPath     = "docs/reports/broker-failover-fault-injection-validation-pack.md"
	sharedQueueReportPath        = "docs/reports/multi-node-shared-queue-report.json"
	sharedQueueSummaryPath       = "docs/reports/shared-queue-companion-summary.json"
	legacyExportValidationPath   = "scripts/e2e/export_validation_bundle.py"
	goExportValidationPath       = "scripts/e2e/export_validation_bundle.go"
)

var continuationArtifacts = []artifactDescription{
	{
		Path:        "docs/reports/validation-bundle-continuation-scorecard.json",
		Description: "summarizes the rolling readiness view across recent bundled local, Kubernetes, and Ray runs plus the shared-queue companion proof.",
	},
	{
		Path:        "docs/reports/validation-bundle-continuation-policy-gate.json",
		Description: "records the current policy decision for bundle freshness, repeated lane coverage, and shared-queue companion availability.",
	},
}

var followupDigests = []artifactDescription{
	{
		Path:        "docs/reports/validation-bundle-continuation-digest.md",
		Description: "Validation bundle continuation caveats are consolidated here.",
	},
}

var laneAliases = map[string]string{
	"local":      "local",
	"kubernetes": "k8s",
	"ray":        "ray",
}

var failureEventTypes = map[string]struct{}{
	"task.cancelled":   {},
	"task.dead_letter": {},
	"task.failed":      {},
	"task.retried":     {},
}

type artifactDescription struct {
	Path        string
	Description string
}

type componentSectionInput struct {
	Name       string
	Enabled    bool
	Root       string
	BundleDir  string
	ReportPath string
	StdoutPath string
	StderrPath string
}

type brokerSectionInput struct {
	Enabled              bool
	Backend              string
	Root                 string
	BundleDir            string
	BootstrapSummaryPath string
	ReportPath           string
}

type exportValidationArgs struct {
	GoRoot                     string
	RunID                      string
	BundleDir                  string
	SummaryPath                string
	IndexPath                  string
	ManifestPath               string
	RunLocal                   string
	RunKubernetes              string
	RunRay                     string
	ValidationStatus           string
	RunBroker                  string
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
}

func main() {
	args := parseExportValidationArgs()
	code, err := runExportValidation(args)
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}

func parseExportValidationArgs() exportValidationArgs {
	var args exportValidationArgs
	flag.StringVar(&args.GoRoot, "go-root", "", "repo root")
	flag.StringVar(&args.RunID, "run-id", "", "run id")
	flag.StringVar(&args.BundleDir, "bundle-dir", "", "bundle dir")
	flag.StringVar(&args.SummaryPath, "summary-path", "docs/reports/live-validation-summary.json", "summary path")
	flag.StringVar(&args.IndexPath, "index-path", "docs/reports/live-validation-index.md", "index path")
	flag.StringVar(&args.ManifestPath, "manifest-path", "docs/reports/live-validation-index.json", "manifest path")
	flag.StringVar(&args.RunLocal, "run-local", "1", "run local")
	flag.StringVar(&args.RunKubernetes, "run-kubernetes", "1", "run kubernetes")
	flag.StringVar(&args.RunRay, "run-ray", "1", "run ray")
	flag.StringVar(&args.ValidationStatus, "validation-status", "0", "validation status")
	flag.StringVar(&args.RunBroker, "run-broker", "0", "run broker")
	flag.StringVar(&args.BrokerBackend, "broker-backend", "", "broker backend")
	flag.StringVar(&args.BrokerReportPath, "broker-report-path", "", "broker report path")
	flag.StringVar(&args.BrokerBootstrapSummaryPath, "broker-bootstrap-summary-path", "", "broker bootstrap summary path")
	flag.StringVar(&args.LocalReportPath, "local-report-path", "", "local report path")
	flag.StringVar(&args.LocalStdoutPath, "local-stdout-path", "", "local stdout path")
	flag.StringVar(&args.LocalStderrPath, "local-stderr-path", "", "local stderr path")
	flag.StringVar(&args.KubernetesReportPath, "kubernetes-report-path", "", "kubernetes report path")
	flag.StringVar(&args.KubernetesStdoutPath, "kubernetes-stdout-path", "", "kubernetes stdout path")
	flag.StringVar(&args.KubernetesStderrPath, "kubernetes-stderr-path", "", "kubernetes stderr path")
	flag.StringVar(&args.RayReportPath, "ray-report-path", "", "ray report path")
	flag.StringVar(&args.RayStdoutPath, "ray-stdout-path", "", "ray stdout path")
	flag.StringVar(&args.RayStderrPath, "ray-stderr-path", "", "ray stderr path")
	flag.Parse()
	return args
}

func runExportValidation(args exportValidationArgs) (int, error) {
	root, err := filepath.Abs(args.GoRoot)
	if err != nil {
		return 1, err
	}
	bundleDir := filepath.Join(root, filepath.FromSlash(args.BundleDir))
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return 1, err
	}

	summary := map[string]any{
		"run_id":       args.RunID,
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"status":       exportValidationStatus(args.ValidationStatus),
		"bundle_path":  relExportPath(bundleDir, root),
		"closeout_commands": []any{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
			"cd bigclaw-go && go test ./...",
			"git push origin <branch> && git log -1 --stat",
		},
	}

	localSection, err := buildComponentSection(componentSectionInput{
		Name:       "local",
		Enabled:    args.RunLocal == "1",
		Root:       root,
		BundleDir:  bundleDir,
		ReportPath: filepath.Join(root, filepath.FromSlash(args.LocalReportPath)),
		StdoutPath: args.LocalStdoutPath,
		StderrPath: args.LocalStderrPath,
	})
	if err != nil {
		return 1, err
	}
	summary["local"] = localSection

	k8sSection, err := buildComponentSection(componentSectionInput{
		Name:       "kubernetes",
		Enabled:    args.RunKubernetes == "1",
		Root:       root,
		BundleDir:  bundleDir,
		ReportPath: filepath.Join(root, filepath.FromSlash(args.KubernetesReportPath)),
		StdoutPath: args.KubernetesStdoutPath,
		StderrPath: args.KubernetesStderrPath,
	})
	if err != nil {
		return 1, err
	}
	summary["kubernetes"] = k8sSection

	raySection, err := buildComponentSection(componentSectionInput{
		Name:       "ray",
		Enabled:    args.RunRay == "1",
		Root:       root,
		BundleDir:  bundleDir,
		ReportPath: filepath.Join(root, filepath.FromSlash(args.RayReportPath)),
		StdoutPath: args.RayStdoutPath,
		StderrPath: args.RayStderrPath,
	})
	if err != nil {
		return 1, err
	}
	summary["ray"] = raySection

	brokerSection, err := buildBrokerSection(brokerSectionInput{
		Enabled:              args.RunBroker == "1",
		Backend:              strings.TrimSpace(args.BrokerBackend),
		Root:                 root,
		BundleDir:            bundleDir,
		BootstrapSummaryPath: joinIfPresent(root, args.BrokerBootstrapSummaryPath),
		ReportPath:           joinIfPresent(root, args.BrokerReportPath),
	})
	if err != nil {
		return 1, err
	}
	summary["broker"] = brokerSection

	sharedQueue, err := buildSharedQueueCompanion(root, bundleDir)
	if err != nil {
		return 1, err
	}
	summary["shared_queue_companion"] = sharedQueue
	summary["validation_matrix"] = buildValidationMatrix(summary)

	continuationGate, err := buildContinuationGateSummary(root)
	if err != nil {
		return 1, err
	}
	if continuationGate != nil {
		summary["continuation_gate"] = continuationGate
	}

	bundleSummaryPath := filepath.Join(bundleDir, "summary.json")
	canonicalSummaryPath := filepath.Join(root, filepath.FromSlash(args.SummaryPath))
	summary["summary_path"] = relExportPath(bundleSummaryPath, root)
	if err := writeExportJSON(bundleSummaryPath, summary); err != nil {
		return 1, err
	}
	if err := writeExportJSON(canonicalSummaryPath, summary); err != nil {
		return 1, err
	}

	recentRuns, err := buildRecentRuns(filepath.Dir(bundleDir))
	if err != nil {
		return 1, err
	}
	manifest := map[string]any{
		"latest":      summary,
		"recent_runs": recentRuns,
	}
	if continuationGate != nil {
		manifest["continuation_gate"] = continuationGate
	}
	if err := writeExportJSON(filepath.Join(root, filepath.FromSlash(args.ManifestPath)), manifest); err != nil {
		return 1, err
	}

	continuationItems := buildArtifactList(root, continuationArtifacts)
	followupItems := buildArtifactList(root, followupDigests)
	indexText := renderIndex(summary, recentRuns, continuationGate, continuationItems, followupItems)
	indexPath := filepath.Join(root, filepath.FromSlash(args.IndexPath))
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return 1, err
	}
	if err := os.WriteFile(indexPath, []byte(indexText), 0o644); err != nil {
		return 1, err
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "README.md"), []byte(indexText), 0o644); err != nil {
		return 1, err
	}

	body, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return 1, err
	}
	if _, err := fmt.Println(string(body)); err != nil {
		return 1, err
	}
	if summary["status"] == "succeeded" {
		return 0, nil
	}
	return 1, nil
}

func exportValidationStatus(raw string) string {
	if raw == "0" {
		return "succeeded"
	}
	return "failed"
}

func buildContinuationGateSummary(root string) (map[string]any, error) {
	gatePath := filepath.Join(root, "docs", "reports", "validation-bundle-continuation-policy-gate.json")
	value, err := readExportJSON(gatePath)
	if err != nil || value == nil {
		return nil, nil
	}
	gate := asExportMap(value)
	if len(gate) == 0 {
		return nil, nil
	}
	return map[string]any{
		"path":          relExportPath(gatePath, root),
		"status":        asExportStringOr(gate["status"], "unknown"),
		"recommendation": asExportStringOr(gate["recommendation"], "unknown"),
		"failing_checks": asExportSlice(gate["failing_checks"]),
		"enforcement":   asExportMap(gate["enforcement"]),
		"summary":       asExportMap(gate["summary"]),
		"reviewer_path": asExportMap(gate["reviewer_path"]),
		"next_actions":  asExportSlice(gate["next_actions"]),
	}, nil
}

func readExportJSON(path string) (any, error) {
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
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return normalizeExportValue(payload), nil
}

func writeExportJSON(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
}

func relExportPath(path, root string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func copyTextArtifact(source, destination string) (string, error) {
	if source == "" {
		return "", nil
	}
	if _, err := os.Stat(source); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	sourceAbs, _ := filepath.Abs(source)
	destAbs, _ := filepath.Abs(destination)
	if sourceAbs == destAbs {
		return destination, nil
	}
	body, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(destination, body, 0o644); err != nil {
		return "", err
	}
	return destination, nil
}

func copyJSONArtifact(source, destination string) (string, error) {
	value, err := readExportJSON(source)
	if err != nil || value == nil {
		return "", err
	}
	sourceAbs, _ := filepath.Abs(source)
	destAbs, _ := filepath.Abs(destination)
	if sourceAbs == destAbs {
		return destination, nil
	}
	if err := writeExportJSON(destination, value); err != nil {
		return "", err
	}
	return destination, nil
}

func firstExportText(values ...any) string {
	for _, value := range values {
		if text, ok := value.(string); ok {
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func collectReportEvents(report map[string]any) []map[string]any {
	events := []map[string]any{}
	status := asExportMap(report["status"])
	if len(status) > 0 {
		for _, item := range asExportSlice(status["events"]) {
			if event := asExportMap(item); len(event) > 0 {
				events = append(events, event)
			}
		}
		if latest := asExportMap(status["latest_event"]); len(latest) > 0 {
			events = append(events, latest)
		}
	}
	for _, item := range asExportSlice(report["events"]) {
		if event := asExportMap(item); len(event) > 0 {
			events = append(events, event)
		}
	}
	return events
}

func eventPayloadText(event map[string]any, key string) string {
	payload := asExportMap(event["payload"])
	return firstExportText(payload[key])
}

func latestReportEvent(report map[string]any) map[string]any {
	events := collectReportEvents(report)
	if len(events) == 0 {
		return nil
	}
	return events[len(events)-1]
}

func findRoutingReason(report map[string]any) string {
	events := collectReportEvents(report)
	for i := len(events) - 1; i >= 0; i-- {
		if firstExportText(events[i]["type"]) == "scheduler.routed" {
			return firstExportText(eventPayloadText(events[i], "reason"))
		}
	}
	return ""
}

func buildFailureRootCause(section map[string]any, report map[string]any) map[string]any {
	events := collectReportEvents(report)
	latestEvent := latestReportEvent(report)
	latestStatus := firstExportText(
		asExportMap(report["status"])["state"],
		asExportMap(report["task"])["state"],
		componentStatus(report),
	)

	var causeEvent map[string]any
	for i := len(events) - 1; i >= 0; i-- {
		if _, ok := failureEventTypes[firstExportText(events[i]["type"])]; ok {
			causeEvent = events[i]
			break
		}
	}
	if causeEvent == nil && latestStatus != "" && latestStatus != "succeeded" {
		causeEvent = latestEvent
	}

	location := firstExportText(
		section["stderr_path"],
		section["service_log_path"],
		section["audit_log_path"],
		section["bundle_report_path"],
	)
	if causeEvent == nil {
		return map[string]any{
			"status":     "not_triggered",
			"event_type": firstExportText(asExportMap(latestEvent)["type"]),
			"message":    "",
			"location":   location,
			"event_id":   "",
			"timestamp":  "",
		}
	}
	return map[string]any{
		"status":     "captured",
		"event_type": firstExportText(causeEvent["type"]),
		"message": firstExportText(
			eventPayloadText(causeEvent, "message"),
			eventPayloadText(causeEvent, "reason"),
			report["error"],
			report["failure_reason"],
		),
		"location":  location,
		"event_id":  firstExportText(causeEvent["id"]),
		"timestamp": firstExportText(causeEvent["timestamp"]),
	}
}

func buildValidationMatrixEntry(name string, section map[string]any, report map[string]any) map[string]any {
	task := asExportMap(report["task"])
	taskID := task["id"]
	if taskID == nil {
		taskID = section["task_id"]
	}
	executor := firstExportText(task["required_executor"])
	if executor == "" {
		executor = name
	}
	rootCause := asExportMap(section["failure_root_cause"])
	return map[string]any{
		"lane":                  laneAliases[name],
		"executor":              executor,
		"enabled":               section["enabled"],
		"status":                asExportStringOr(section["status"], "unknown"),
		"task_id":               taskID,
		"canonical_report_path": asExportStringOr(section["canonical_report_path"], ""),
		"bundle_report_path":    asExportStringOr(section["bundle_report_path"], ""),
		"latest_event_type":     asExportStringOr(section["latest_event_type"], ""),
		"routing_reason":        asExportStringOr(section["routing_reason"], ""),
		"root_cause_event_type": asExportStringOr(rootCause["event_type"], ""),
		"root_cause_location":   asExportStringOr(rootCause["location"], ""),
		"root_cause_message":    asExportStringOr(rootCause["message"], ""),
	}
}

func buildValidationMatrix(summary map[string]any) []any {
	rows := []any{}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section := asExportMap(summary[name])
		if len(section) == 0 {
			continue
		}
		if row := asExportMap(section["validation_matrix"]); len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows
}

func componentStatus(report map[string]any) string {
	if len(report) == 0 {
		return "missing_report"
	}
	status := report["status"]
	if statusMap := asExportMap(status); len(statusMap) > 0 {
		return asExportStringOr(statusMap["state"], "unknown")
	}
	if statusText, ok := status.(string); ok {
		return statusText
	}
	if value, ok := report["all_ok"].(bool); ok {
		if value {
			return "succeeded"
		}
		return "failed"
	}
	return "unknown"
}

func buildSharedQueueCompanion(root, bundleDir string) (map[string]any, error) {
	canonicalReportPath := filepath.Join(root, filepath.FromSlash(sharedQueueReportPath))
	canonicalSummaryPath := filepath.Join(root, filepath.FromSlash(sharedQueueSummaryPath))
	bundleReportPath := filepath.Join(bundleDir, "multi-node-shared-queue-report.json")
	bundleSummaryPath := filepath.Join(bundleDir, "shared-queue-companion-summary.json")
	value, err := readExportJSON(canonicalReportPath)
	if err != nil {
		return nil, err
	}
	report := asExportMap(value)

	summary := map[string]any{
		"available":              len(report) > 0,
		"canonical_report_path":  sharedQueueReportPath,
		"canonical_summary_path": sharedQueueSummaryPath,
		"bundle_report_path":     relExportPath(bundleReportPath, root),
		"bundle_summary_path":    relExportPath(bundleSummaryPath, root),
	}
	if len(report) == 0 {
		summary["status"] = "missing_report"
		return summary, nil
	}

	if copied, err := copyJSONArtifact(canonicalReportPath, bundleReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		summary["bundle_report_path"] = relExportPath(copied, root)
	}

	summary["status"] = ternaryStatus(asExportBool(report["all_ok"]))
	summary["generated_at"] = report["generated_at"]
	summary["count"] = report["count"]
	summary["cross_node_completions"] = report["cross_node_completions"]
	summary["duplicate_started_tasks"] = len(asExportSlice(report["duplicate_started_tasks"]))
	summary["duplicate_completed_tasks"] = len(asExportSlice(report["duplicate_completed_tasks"]))
	summary["missing_completed_tasks"] = len(asExportSlice(report["missing_completed_tasks"]))
	summary["submitted_by_node"] = asExportMap(report["submitted_by_node"])
	summary["completed_by_node"] = asExportMap(report["completed_by_node"])
	nodes := []any{}
	for _, item := range asExportSlice(report["nodes"]) {
		node := asExportMap(item)
		if name := firstExportText(node["name"]); name != "" {
			nodes = append(nodes, name)
		}
	}
	summary["nodes"] = nodes
	if err := writeExportJSON(bundleSummaryPath, summary); err != nil {
		return nil, err
	}
	if err := writeExportJSON(canonicalSummaryPath, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func ternaryStatus(ok bool) string {
	if ok {
		return "succeeded"
	}
	return "failed"
}

func buildComponentSection(input componentSectionInput) (map[string]any, error) {
	latestReportPath := filepath.Join(input.Root, filepath.FromSlash(latestReports[input.Name]))
	section := map[string]any{
		"enabled":               input.Enabled,
		"bundle_report_path":    relExportPath(input.ReportPath, input.Root),
		"canonical_report_path": latestReports[input.Name],
	}
	if !input.Enabled {
		section["status"] = "skipped"
		return section, nil
	}

	value, err := readExportJSON(input.ReportPath)
	if err != nil {
		return nil, err
	}
	report := asExportMap(value)
	section["report"] = report
	section["status"] = componentStatus(report)

	if copied, err := copyJSONArtifact(input.ReportPath, latestReportPath); err != nil {
		return nil, err
	} else if copied != "" {
		section["canonical_report_path"] = relExportPath(copied, input.Root)
	}
	if copied, err := copyTextArtifact(input.StdoutPath, filepath.Join(input.BundleDir, input.Name+".stdout.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stdout_path"] = relExportPath(copied, input.Root)
	}
	if copied, err := copyTextArtifact(input.StderrPath, filepath.Join(input.BundleDir, input.Name+".stderr.log")); err != nil {
		return nil, err
	} else if copied != "" {
		section["stderr_path"] = relExportPath(copied, input.Root)
	}

	if len(report) > 0 {
		task := asExportMap(report["task"])
		if taskID := firstExportText(task["id"]); taskID != "" {
			section["task_id"] = taskID
		}
		if baseURL := firstExportText(report["base_url"]); baseURL != "" {
			section["base_url"] = baseURL
		}
		if stateDir := firstExportText(report["state_dir"]); stateDir != "" {
			section["state_dir"] = stateDir
			if copied, err := copyTextArtifact(filepath.Join(stateDir, "audit.jsonl"), filepath.Join(input.BundleDir, input.Name+".audit.jsonl")); err != nil {
				return nil, err
			} else if copied != "" {
				section["audit_log_path"] = relExportPath(copied, input.Root)
			}
		}
		if serviceLog := firstExportText(report["service_log"]); serviceLog != "" {
			if copied, err := copyTextArtifact(serviceLog, filepath.Join(input.BundleDir, input.Name+".service.log")); err != nil {
				return nil, err
			} else if copied != "" {
				section["service_log_path"] = relExportPath(copied, input.Root)
			}
		}
		if latestEvent := latestReportEvent(report); len(latestEvent) > 0 {
			section["latest_event_type"] = firstExportText(latestEvent["type"])
			section["latest_event_timestamp"] = firstExportText(latestEvent["timestamp"])
			payload := asExportMap(latestEvent["payload"])
			if artifacts := asExportSlice(payload["artifacts"]); len(artifacts) > 0 {
				section["artifact_paths"] = artifacts
			}
		}
		if routingReason := findRoutingReason(report); routingReason != "" {
			section["routing_reason"] = routingReason
		}
		section["failure_root_cause"] = buildFailureRootCause(section, report)
		section["validation_matrix"] = buildValidationMatrixEntry(input.Name, section, report)
	} else {
		section["failure_root_cause"] = map[string]any{
			"status":     "missing_report",
			"event_type": "",
			"message":    "",
			"location":   section["bundle_report_path"],
		}
		section["validation_matrix"] = buildValidationMatrixEntry(input.Name, section, map[string]any{})
	}
	return section, nil
}

func buildBrokerSection(input brokerSectionInput) (map[string]any, error) {
	bundleSummaryPath := filepath.Join(input.BundleDir, "broker-validation-summary.json")
	bundleBootstrapSummaryPath := filepath.Join(input.BundleDir, "broker-bootstrap-review-summary.json")
	section := map[string]any{
		"enabled":                         input.Enabled,
		"backend":                         input.Backend,
		"bundle_summary_path":             relExportPath(bundleSummaryPath, input.Root),
		"canonical_summary_path":          brokerSummaryPath,
		"bundle_bootstrap_summary_path":   relExportPath(bundleBootstrapSummaryPath, input.Root),
		"canonical_bootstrap_summary_path": brokerBootstrapSummaryPath,
		"validation_pack_path":            brokerValidationPackPath,
	}
	configurationState := "not_configured"
	if input.Enabled && input.Backend != "" {
		configurationState = "configured"
	}
	section["configuration_state"] = configurationState

	if input.BootstrapSummaryPath != "" {
		if value, err := readExportJSON(input.BootstrapSummaryPath); err != nil {
			return nil, err
		} else if bootstrap := asExportMap(value); len(bootstrap) > 0 {
			if copied, err := copyJSONArtifact(input.BootstrapSummaryPath, bundleBootstrapSummaryPath); err != nil {
				return nil, err
			} else if copied != "" {
				section["bundle_bootstrap_summary_path"] = relExportPath(copied, input.Root)
			}
			if copied, err := copyJSONArtifact(input.BootstrapSummaryPath, filepath.Join(input.Root, filepath.FromSlash(brokerBootstrapSummaryPath))); err != nil {
				return nil, err
			} else if copied != "" {
				section["canonical_bootstrap_summary_path"] = relExportPath(copied, input.Root)
			}
			section["bootstrap_summary"] = bootstrap
			section["bootstrap_ready"] = asExportBool(bootstrap["ready"])
			section["runtime_posture"] = bootstrap["runtime_posture"]
			section["live_adapter_implemented"] = asExportBool(bootstrap["live_adapter_implemented"])
			section["proof_boundary"] = bootstrap["proof_boundary"]
			if validationErrors := asExportSlice(bootstrap["validation_errors"]); len(validationErrors) > 0 {
				section["validation_errors"] = validationErrors
			}
			if completeness := asExportMap(bootstrap["config_completeness"]); len(completeness) > 0 {
				section["config_completeness"] = completeness
			}
		}
	}

	if !input.Enabled || input.Backend == "" {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := writeExportJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := writeExportJSON(filepath.Join(input.Root, filepath.FromSlash(brokerSummaryPath)), section); err != nil {
			return nil, err
		}
		return section, nil
	}

	if input.ReportPath == "" {
		section["status"] = "skipped"
		section["reason"] = "missing_report_path"
		if err := writeExportJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := writeExportJSON(filepath.Join(input.Root, filepath.FromSlash(brokerSummaryPath)), section); err != nil {
			return nil, err
		}
		return section, nil
	}

	value, err := readExportJSON(input.ReportPath)
	if err != nil {
		return nil, err
	}
	report := asExportMap(value)
	section["canonical_report_path"] = relExportPath(input.ReportPath, input.Root)
	section["bundle_report_path"] = relExportPath(filepath.Join(input.BundleDir, filepath.Base(input.ReportPath)), input.Root)
	if len(report) == 0 {
		section["status"] = "skipped"
		section["reason"] = "not_configured"
		if err := writeExportJSON(bundleSummaryPath, section); err != nil {
			return nil, err
		}
		if err := writeExportJSON(filepath.Join(input.Root, filepath.FromSlash(brokerSummaryPath)), section); err != nil {
			return nil, err
		}
		return section, nil
	}
	if copied, err := copyJSONArtifact(input.ReportPath, filepath.Join(input.BundleDir, filepath.Base(input.ReportPath))); err != nil {
		return nil, err
	} else if copied != "" {
		section["bundle_report_path"] = relExportPath(copied, input.Root)
	}
	section["report"] = report
	section["status"] = componentStatus(report)
	if err := writeExportJSON(bundleSummaryPath, section); err != nil {
		return nil, err
	}
	if err := writeExportJSON(filepath.Join(input.Root, filepath.FromSlash(brokerSummaryPath)), section); err != nil {
		return nil, err
	}
	return section, nil
}

func buildRecentRuns(bundleRoot string) ([]any, error) {
	entries, err := os.ReadDir(bundleRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	type runItem struct {
		GeneratedAt string
		Summary     map[string]any
	}
	runs := []runItem{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		value, err := readExportJSON(filepath.Join(bundleRoot, entry.Name(), "summary.json"))
		if err != nil {
			return nil, err
		}
		summary := asExportMap(value)
		if len(summary) == 0 {
			continue
		}
		runs = append(runs, runItem{
			GeneratedAt: asExportStringOr(summary["generated_at"], ""),
			Summary:     summary,
		})
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].GeneratedAt > runs[j].GeneratedAt })
	if len(runs) > 8 {
		runs = runs[:8]
	}
	items := make([]any, 0, len(runs))
	for _, run := range runs {
		items = append(items, map[string]any{
			"run_id":       run.Summary["run_id"],
			"generated_at": run.Summary["generated_at"],
			"status":       asExportStringOr(run.Summary["status"], "unknown"),
			"bundle_path":  asExportStringOr(run.Summary["bundle_path"], ""),
			"summary_path": asExportStringOr(run.Summary["summary_path"], ""),
		})
	}
	return items, nil
}

func buildArtifactList(root string, candidates []artifactDescription) []artifactDescription {
	items := []artifactDescription{}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(candidate.Path))); err == nil {
			items = append(items, candidate)
		}
	}
	return items
}

func renderIndex(summary map[string]any, recentRuns []any, continuationGate map[string]any, continuationArtifacts []artifactDescription, followupDigests []artifactDescription) string {
	lines := []string{
		"# Live Validation Index",
		"",
		fmt.Sprintf("- Latest run: `%v`", summary["run_id"]),
		fmt.Sprintf("- Generated at: `%v`", summary["generated_at"]),
		fmt.Sprintf("- Status: `%v`", summary["status"]),
		fmt.Sprintf("- Bundle: `%v`", summary["bundle_path"]),
		fmt.Sprintf("- Summary JSON: `%v`", summary["summary_path"]),
		"",
		"## Latest bundle artifacts",
		"",
	}
	for _, name := range []string{"local", "kubernetes", "ray"} {
		section := asExportMap(summary[name])
		matrix := asExportMap(section["validation_matrix"])
		lines = append(lines, fmt.Sprintf("### %s", name))
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", section["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", section["status"]))
		if lane := firstExportText(matrix["lane"]); lane != "" {
			lines = append(lines, fmt.Sprintf("- Validation lane: `%s`", lane))
		}
		lines = append(lines, fmt.Sprintf("- Bundle report: `%v`", section["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Latest report: `%v`", section["canonical_report_path"]))
		for _, field := range []struct {
			Key   string
			Label string
		}{
			{"stdout_path", "Stdout log"},
			{"stderr_path", "Stderr log"},
			{"service_log_path", "Service log"},
			{"audit_log_path", "Audit log"},
			{"task_id", "Task ID"},
			{"latest_event_type", "Latest event"},
			{"routing_reason", "Routing reason"},
		} {
			if value := firstExportText(section[field.Key]); value != "" {
				lines = append(lines, fmt.Sprintf("- %s: `%s`", field.Label, value))
			}
		}
		rootCause := asExportMap(section["failure_root_cause"])
		if len(rootCause) > 0 {
			lines = append(lines,
				fmt.Sprintf("- Failure root cause: status=`%s` event=`%s` location=`%s`",
					asExportStringOr(rootCause["status"], "unknown"),
					asExportStringOr(rootCause["event_type"], "unknown"),
					asExportStringOr(rootCause["location"], "n/a"),
				),
			)
			if message := firstExportText(rootCause["message"]); message != "" {
				lines = append(lines, fmt.Sprintf("- Failure detail: `%s`", message))
			}
		}
		lines = append(lines, "")
	}

	if matrix := asExportSlice(summary["validation_matrix"]); len(matrix) > 0 {
		lines = append(lines, "## Validation matrix", "")
		for _, rowItem := range matrix {
			row := asExportMap(rowItem)
			lines = append(lines, fmt.Sprintf(
				"- Lane `%s` executor=`%s` status=`%s` enabled=`%v` report=`%s`",
				asExportStringOr(row["lane"], "unknown"),
				asExportStringOr(row["executor"], "unknown"),
				asExportStringOr(row["status"], "unknown"),
				row["enabled"],
				asExportStringOr(row["bundle_report_path"], ""),
			))
			if firstExportText(row["root_cause_event_type"], row["root_cause_message"]) != "" {
				lines = append(lines, fmt.Sprintf(
					"- Lane `%s` root cause: event=`%s` location=`%s` message=`%s`",
					asExportStringOr(row["lane"], "unknown"),
					asExportStringOr(row["root_cause_event_type"], "unknown"),
					asExportStringOr(row["root_cause_location"], "n/a"),
					asExportStringOr(row["root_cause_message"], ""),
				))
			}
		}
		lines = append(lines, "")
	}

	if broker := asExportMap(summary["broker"]); len(broker) > 0 {
		lines = append(lines, "### broker")
		lines = append(lines, fmt.Sprintf("- Enabled: `%v`", broker["enabled"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", broker["status"]))
		lines = append(lines, fmt.Sprintf("- Configuration state: `%v`", broker["configuration_state"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%v`", broker["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%v`", broker["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle bootstrap summary: `%v`", broker["bundle_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical bootstrap summary: `%v`", broker["canonical_bootstrap_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Validation pack: `%v`", broker["validation_pack_path"]))
		if backend := firstExportText(broker["backend"]); backend != "" {
			lines = append(lines, fmt.Sprintf("- Backend: `%s`", backend))
		}
		if _, ok := broker["bootstrap_ready"]; ok {
			lines = append(lines, fmt.Sprintf("- Bootstrap ready: `%v`", broker["bootstrap_ready"]))
		}
		if value := firstExportText(broker["runtime_posture"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Runtime posture: `%s`", value))
		}
		if _, ok := broker["live_adapter_implemented"]; ok {
			lines = append(lines, fmt.Sprintf("- Live adapter implemented: `%v`", broker["live_adapter_implemented"]))
		}
		if completeness := asExportMap(broker["config_completeness"]); len(completeness) > 0 {
			lines = append(lines, fmt.Sprintf(
				"- Config completeness: driver=`%v` urls=`%v` topic=`%v` consumer_group=`%v`",
				completeness["driver"], completeness["urls"], completeness["topic"], completeness["consumer_group"],
			))
		}
		if value := firstExportText(broker["proof_boundary"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Proof boundary: `%s`", value))
		}
		for _, item := range asExportSlice(broker["validation_errors"]) {
			if text := firstExportText(item); text != "" {
				lines = append(lines, fmt.Sprintf("- Validation error: `%s`", text))
			}
		}
		if value := firstExportText(broker["bundle_report_path"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Bundle report: `%s`", value))
		}
		if value := firstExportText(broker["canonical_report_path"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Canonical report: `%s`", value))
		}
		if value := firstExportText(broker["reason"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Reason: `%s`", value))
		}
		lines = append(lines, "")
	}

	if sharedQueue := asExportMap(summary["shared_queue_companion"]); len(sharedQueue) > 0 {
		lines = append(lines, "### shared-queue companion")
		lines = append(lines, fmt.Sprintf("- Available: `%v`", sharedQueue["available"]))
		lines = append(lines, fmt.Sprintf("- Status: `%v`", sharedQueue["status"]))
		lines = append(lines, fmt.Sprintf("- Bundle summary: `%v`", sharedQueue["bundle_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical summary: `%v`", sharedQueue["canonical_summary_path"]))
		lines = append(lines, fmt.Sprintf("- Bundle report: `%v`", sharedQueue["bundle_report_path"]))
		lines = append(lines, fmt.Sprintf("- Canonical report: `%v`", sharedQueue["canonical_report_path"]))
		for _, field := range []struct {
			Key   string
			Label string
		}{
			{"cross_node_completions", "Cross-node completions"},
			{"duplicate_started_tasks", "Duplicate `task.started`"},
			{"duplicate_completed_tasks", "Duplicate `task.completed`"},
			{"missing_completed_tasks", "Missing `task.completed`"},
		} {
			if _, ok := sharedQueue[field.Key]; ok {
				lines = append(lines, fmt.Sprintf("- %s: `%v`", field.Label, sharedQueue[field.Key]))
			}
		}
		if nodes := asExportSlice(sharedQueue["nodes"]); len(nodes) > 0 {
			names := []string{}
			for _, node := range nodes {
				names = append(names, firstExportText(node))
			}
			lines = append(lines, fmt.Sprintf("- Nodes: `%s`", strings.Join(names, ", ")))
		}
		lines = append(lines, "")
	}

	if len(recentRuns) > 0 {
		lines = append(lines, "## Recent runs", "")
		for _, item := range recentRuns {
			run := asExportMap(item)
			lines = append(lines, fmt.Sprintf(
				"- `%s` status=`%s` generated_at=`%s` summary=`%s`",
				asExportStringOr(run["run_id"], ""),
				asExportStringOr(run["status"], "unknown"),
				asExportStringOr(run["generated_at"], ""),
				asExportStringOr(run["summary_path"], ""),
			))
		}
		lines = append(lines, "")
	}

	if continuationGate != nil {
		lines = append(lines, "## Continuation workflow", "")
		lines = append(lines, fmt.Sprintf("- Policy gate: `%s`", asExportStringOr(continuationGate["path"], "")))
		lines = append(lines, fmt.Sprintf("- Policy status: `%s`", asExportStringOr(continuationGate["status"], "unknown")))
		lines = append(lines, fmt.Sprintf("- Recommendation: `%s`", asExportStringOr(continuationGate["recommendation"], "unknown")))
		enforcement := asExportMap(continuationGate["enforcement"])
		if len(enforcement) > 0 {
			lines = append(lines, fmt.Sprintf("- Workflow mode: `%s`", asExportStringOr(enforcement["mode"], "unknown")))
			lines = append(lines, fmt.Sprintf("- Workflow outcome: `%s`", asExportStringOr(enforcement["outcome"], "unknown")))
			if _, ok := enforcement["exit_code"]; ok {
				lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", enforcement["exit_code"]))
			}
		}
		gateSummary := asExportMap(continuationGate["summary"])
		if latestRunID := firstExportText(gateSummary["latest_run_id"]); latestRunID != "" {
			lines = append(lines, fmt.Sprintf("- Gate latest run: `%s`", latestRunID))
		}
		if _, ok := gateSummary["failing_check_count"]; ok {
			lines = append(lines, fmt.Sprintf("- Failing checks: `%v`", gateSummary["failing_check_count"]))
		}
		if _, ok := gateSummary["workflow_exit_code"]; ok {
			lines = append(lines, fmt.Sprintf("- Workflow exit code on current evidence: `%v`", gateSummary["workflow_exit_code"]))
		}
		reviewerPath := asExportMap(continuationGate["reviewer_path"])
		if value := firstExportText(reviewerPath["digest_path"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer digest: `%s`", value))
		}
		if value := firstExportText(reviewerPath["index_path"]); value != "" {
			lines = append(lines, fmt.Sprintf("- Reviewer index: `%s`", value))
		}
		for _, item := range asExportSlice(continuationGate["next_actions"]) {
			if text := firstExportText(item); text != "" {
				lines = append(lines, fmt.Sprintf("- Next action: `%s`", text))
			}
		}
		lines = append(lines, "")
	}

	if len(continuationArtifacts) > 0 {
		lines = append(lines, "## Continuation artifacts", "")
		for _, item := range continuationArtifacts {
			lines = append(lines, fmt.Sprintf("- `%s` %s", item.Path, item.Description))
		}
		lines = append(lines, "")
	}

	if len(followupDigests) > 0 {
		lines = append(lines, "## Parallel follow-up digests", "")
		for _, item := range followupDigests {
			lines = append(lines, fmt.Sprintf("- `%s` %s", item.Path, item.Description))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func normalizeExportValue(value any) any {
	switch cast := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(cast))
		for key, item := range cast {
			out[key] = normalizeExportValue(item)
		}
		return out
	case []any:
		out := make([]any, len(cast))
		for index, item := range cast {
			out[index] = normalizeExportValue(item)
		}
		return out
	case string:
		if cast == legacyExportValidationPath {
			return goExportValidationPath
		}
		return cast
	default:
		return value
	}
}

func joinIfPresent(root, rel string) string {
	if strings.TrimSpace(rel) == "" {
		return ""
	}
	return filepath.Join(root, filepath.FromSlash(rel))
}

func asExportMap(value any) map[string]any {
	if cast, ok := value.(map[string]any); ok {
		return cast
	}
	return map[string]any{}
}

func asExportSlice(value any) []any {
	if cast, ok := value.([]any); ok {
		return cast
	}
	return nil
}

func asExportBool(value any) bool {
	cast, ok := value.(bool)
	return ok && cast
}

func asExportStringOr(value any, fallback string) string {
	if cast, ok := value.(string); ok {
		return cast
	}
	return fallback
}
