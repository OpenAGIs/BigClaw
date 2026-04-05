package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type automationE2ERunAllOptions struct {
	GoRoot                     string
	SummaryPath                string
	IndexPath                  string
	ManifestPath               string
	ArtifactRootRel            string
	RunID                      string
	RunLocal                   bool
	RunKubernetes              bool
	RunRay                     bool
	RunBroker                  bool
	BrokerBackend              string
	BrokerReportPath           string
	BrokerBootstrapSummaryPath string
	RefreshContinuation        bool
	EnforceContinuationGate    bool
	ContinuationGateMode       string
	ContinuationScorecardPath  string
	ContinuationPolicyGatePath string
	KubernetesImage            string
	KubernetesEntrypoint       string
	RayEntrypoint              string
	RayRuntimeEnvJSON          string
}

func runAutomationE2ERunAllCommand(args []string) error {
	now := time.Now().UTC()
	flags := flag.NewFlagSet("automation e2e run-all", flag.ContinueOnError)
	goRoot := flags.String("go-root", defaultRepoRoot(), "bigclaw-go repo root")
	summaryPath := flags.String("summary-path", envOrDefault("BIGCLAW_E2E_SUMMARY_REPORT_PATH", "docs/reports/live-validation-summary.json"), "summary output path")
	indexPath := flags.String("index-path", envOrDefault("BIGCLAW_E2E_INDEX_PATH", "docs/reports/live-validation-index.md"), "index markdown path")
	manifestPath := flags.String("manifest-path", envOrDefault("BIGCLAW_E2E_MANIFEST_PATH", "docs/reports/live-validation-index.json"), "manifest json path")
	artifactRoot := flags.String("artifact-root", envOrDefault("BIGCLAW_E2E_ARTIFACT_ROOT", "docs/reports/live-validation-runs"), "bundle artifact root")
	runID := flags.String("run-id", envOrDefault("BIGCLAW_E2E_RUN_ID", now.Format("20060102T150405Z")), "bundle run id")
	runLocal := flags.Bool("run-local", envBoolOrDefault("BIGCLAW_E2E_RUN_LOCAL", true), "run local lane")
	runKubernetes := flags.Bool("run-kubernetes", envBoolOrDefault("BIGCLAW_E2E_RUN_KUBERNETES", true), "run kubernetes lane")
	runRay := flags.Bool("run-ray", envBoolOrDefault("BIGCLAW_E2E_RUN_RAY", true), "run ray lane")
	runBroker := flags.Bool("run-broker", envBoolOrDefault("BIGCLAW_E2E_RUN_BROKER", false), "run broker lane")
	brokerBackend := flags.String("broker-backend", envOrDefault("BIGCLAW_E2E_BROKER_BACKEND", ""), "broker backend")
	brokerReportPath := flags.String("broker-report-path", envOrDefault("BIGCLAW_E2E_BROKER_REPORT_PATH", ""), "broker report path")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", envOrDefault("BIGCLAW_E2E_BROKER_BOOTSTRAP_SUMMARY_PATH", "docs/reports/broker-bootstrap-review-summary.json"), "broker bootstrap summary path")
	refreshContinuation := flags.Bool("refresh-continuation", envBoolOrDefault("BIGCLAW_E2E_REFRESH_CONTINUATION", true), "refresh continuation scorecard and gate")
	enforceContinuationGate := flags.Bool("enforce-continuation-gate", envBoolOrDefault("BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE", false), "legacy continuation gate fail alias")
	continuationGateMode := flags.String("continuation-gate-mode", envOrDefault("BIGCLAW_E2E_CONTINUATION_GATE_MODE", ""), "continuation gate mode: review|hold|fail")
	continuationScorecardPath := flags.String("continuation-scorecard-path", envOrDefault("BIGCLAW_E2E_CONTINUATION_SCORECARD_PATH", "docs/reports/validation-bundle-continuation-scorecard.json"), "continuation scorecard path")
	continuationPolicyGatePath := flags.String("continuation-policy-gate-path", envOrDefault("BIGCLAW_E2E_CONTINUATION_POLICY_GATE_PATH", "docs/reports/validation-bundle-continuation-policy-gate.json"), "continuation policy gate path")
	kubernetesImage := flags.String("kubernetes-image", envOrDefault("BIGCLAW_KUBERNETES_SMOKE_IMAGE", envOrDefault("BIGCLAW_KUBERNETES_IMAGE", "alpine:3.20")), "kubernetes smoke image")
	kubernetesEntrypoint := flags.String("kubernetes-entrypoint", envOrDefault("BIGCLAW_KUBERNETES_SMOKE_ENTRYPOINT", "echo hello from kubernetes"), "kubernetes smoke entrypoint")
	rayEntrypoint := flags.String("ray-entrypoint", envOrDefault("BIGCLAW_RAY_SMOKE_ENTRYPOINT", "python -c \"print('hello from ray')\""), "ray smoke entrypoint")
	rayRuntimeEnvJSON := flags.String("ray-runtime-env-json", envOrDefault("BIGCLAW_RAY_RUNTIME_ENV_JSON", ""), "ray runtime env json")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e run-all [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	report, exitCode, err := automationE2ERunAll(automationE2ERunAllOptions{
		GoRoot:                     absPath(*goRoot),
		SummaryPath:                *summaryPath,
		IndexPath:                  *indexPath,
		ManifestPath:               *manifestPath,
		ArtifactRootRel:            *artifactRoot,
		RunID:                      *runID,
		RunLocal:                   *runLocal,
		RunKubernetes:              *runKubernetes,
		RunRay:                     *runRay,
		RunBroker:                  *runBroker,
		BrokerBackend:              *brokerBackend,
		BrokerReportPath:           *brokerReportPath,
		BrokerBootstrapSummaryPath: *brokerBootstrapSummaryPath,
		RefreshContinuation:        *refreshContinuation,
		EnforceContinuationGate:    *enforceContinuationGate,
		ContinuationGateMode:       *continuationGateMode,
		ContinuationScorecardPath:  *continuationScorecardPath,
		ContinuationPolicyGatePath: *continuationPolicyGatePath,
		KubernetesImage:            *kubernetesImage,
		KubernetesEntrypoint:       *kubernetesEntrypoint,
		RayEntrypoint:              *rayEntrypoint,
		RayRuntimeEnvJSON:          *rayRuntimeEnvJSON,
	})
	if err != nil {
		return err
	}
	return emit(report, *asJSON, exitCode)
}

func automationE2ERunAll(opts automationE2ERunAllOptions) (map[string]any, int, error) {
	root := absPath(opts.GoRoot)
	if trim(opts.SummaryPath) == "" {
		opts.SummaryPath = "docs/reports/live-validation-summary.json"
	}
	if trim(opts.IndexPath) == "" {
		opts.IndexPath = "docs/reports/live-validation-index.md"
	}
	if trim(opts.ManifestPath) == "" {
		opts.ManifestPath = "docs/reports/live-validation-index.json"
	}
	if trim(opts.ArtifactRootRel) == "" {
		opts.ArtifactRootRel = "docs/reports/live-validation-runs"
	}
	if trim(opts.RunID) == "" {
		opts.RunID = time.Now().UTC().Format("20060102T150405Z")
	}
	if trim(opts.BrokerBootstrapSummaryPath) == "" {
		opts.BrokerBootstrapSummaryPath = "docs/reports/broker-bootstrap-review-summary.json"
	}
	if trim(opts.ContinuationScorecardPath) == "" {
		opts.ContinuationScorecardPath = "docs/reports/validation-bundle-continuation-scorecard.json"
	}
	if trim(opts.ContinuationPolicyGatePath) == "" {
		opts.ContinuationPolicyGatePath = "docs/reports/validation-bundle-continuation-policy-gate.json"
	}
	continuationMode := trim(opts.ContinuationGateMode)
	if continuationMode == "" {
		if opts.EnforceContinuationGate {
			continuationMode = "fail"
		} else {
			continuationMode = "hold"
		}
	}

	scorecardPath := normalizeLegacyGoRootRelativePath(root, opts.ContinuationScorecardPath)
	policyGatePath := normalizeLegacyGoRootRelativePath(root, opts.ContinuationPolicyGatePath)
	bundleDirRel := filepath.ToSlash(filepath.Join(opts.ArtifactRootRel, opts.RunID))
	if err := os.MkdirAll(filepath.Join(root, bundleDirRel), 0o755); err != nil {
		return nil, 0, err
	}

	localReportRel := filepath.ToSlash(filepath.Join(bundleDirRel, "sqlite-smoke-report.json"))
	k8sReportRel := filepath.ToSlash(filepath.Join(bundleDirRel, "kubernetes-live-smoke-report.json"))
	rayReportRel := filepath.ToSlash(filepath.Join(bundleDirRel, "ray-live-smoke-report.json"))

	localStdout, err := os.CreateTemp("", "bigclaw-local-e2e-out.*")
	if err != nil {
		return nil, 0, err
	}
	localStderr, err := os.CreateTemp("", "bigclaw-local-e2e-err.*")
	if err != nil {
		return nil, 0, err
	}
	k8sStdout, err := os.CreateTemp("", "bigclaw-k8s-e2e-out.*")
	if err != nil {
		return nil, 0, err
	}
	k8sStderr, err := os.CreateTemp("", "bigclaw-k8s-e2e-err.*")
	if err != nil {
		return nil, 0, err
	}
	rayStdout, err := os.CreateTemp("", "bigclaw-ray-e2e-out.*")
	if err != nil {
		return nil, 0, err
	}
	rayStderr, err := os.CreateTemp("", "bigclaw-ray-e2e-err.*")
	if err != nil {
		return nil, 0, err
	}
	tempFiles := []*os.File{localStdout, localStderr, k8sStdout, k8sStderr, rayStdout, rayStderr}
	defer func() {
		for _, file := range tempFiles {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
	}()

	type laneResult struct {
		err      error
		exitCode int
	}
	results := make(chan laneResult, 3)
	var wg sync.WaitGroup

	runLane := func(enabled bool, env map[string]string, stdoutFile, stderrFile *os.File, args []string) {
		if !enabled {
			return
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			exitCode, err := automationRunExternalCommand(root, env, stdoutFile, stderrFile, args...)
			results <- laneResult{err: err, exitCode: exitCode}
		}()
	}

	runLane(opts.RunLocal, map[string]string{"BIGCLAW_QUEUE_BACKEND": "sqlite"}, localStdout, localStderr,
		[]string{
			"run", filepath.Join(root, "cmd", "bigclawctl"), "automation", "e2e", "run-task-smoke",
			"--autostart",
			"--go-root", root,
			"--executor", "local",
			"--title", "SQLite smoke",
			"--entrypoint", "echo hello from sqlite",
			"--report-path", localReportRel,
		},
	)
	runLane(opts.RunKubernetes, nil, k8sStdout, k8sStderr,
		[]string{
			"run", filepath.Join(root, "cmd", "bigclawctl"), "automation", "e2e", "run-task-smoke",
			"--autostart",
			"--go-root", root,
			"--executor", "kubernetes",
			"--title", "Kubernetes smoke test",
			"--image", opts.KubernetesImage,
			"--entrypoint", opts.KubernetesEntrypoint,
			"--report-path", k8sReportRel,
		},
	)
	rayArgs := []string{
		"run", filepath.Join(root, "cmd", "bigclawctl"), "automation", "e2e", "run-task-smoke",
		"--autostart",
		"--go-root", root,
		"--executor", "ray",
		"--title", "Ray smoke test",
		"--entrypoint", opts.RayEntrypoint,
		"--report-path", rayReportRel,
	}
	if trim(opts.RayRuntimeEnvJSON) != "" {
		rayArgs = append(rayArgs, "--runtime-env-json", opts.RayRuntimeEnvJSON)
	}
	runLane(opts.RunRay, nil, rayStdout, rayStderr, rayArgs)

	go func() {
		wg.Wait()
		close(results)
	}()

	validationStatus := 0
	for result := range results {
		if result.exitCode != 0 || result.err != nil {
			validationStatus = 1
		}
	}

	bootstrapSummaryPath := normalizeLegacyGoRootRelativePath(root, opts.BrokerBootstrapSummaryPath)
	if _, _, err := automationRunJSONCommand(root, nil,
		"run", filepath.Join(root, "scripts", "e2e", "broker_bootstrap_summary.go"),
		"--output", filepath.Join(root, bootstrapSummaryPath),
	); err != nil {
		return nil, 0, err
	}

	exportArgs := []string{
		"run", filepath.Join(root, "cmd", "bigclawctl"), "automation", "e2e", "export-validation-bundle",
		"--go-root", root,
		"--run-id", opts.RunID,
		"--bundle-dir", bundleDirRel,
		"--summary-path", opts.SummaryPath,
		"--index-path", opts.IndexPath,
		"--manifest-path", opts.ManifestPath,
		"--run-local", strconv.FormatBool(opts.RunLocal),
		"--run-kubernetes", strconv.FormatBool(opts.RunKubernetes),
		"--run-ray", strconv.FormatBool(opts.RunRay),
		"--run-broker", strconv.FormatBool(opts.RunBroker),
		"--broker-backend", opts.BrokerBackend,
		"--broker-report-path", opts.BrokerReportPath,
		"--broker-bootstrap-summary-path", bootstrapSummaryPath,
		"--validation-status", strconv.Itoa(validationStatus),
		"--local-report-path", localReportRel,
		"--local-stdout-path", localStdout.Name(),
		"--local-stderr-path", localStderr.Name(),
		"--kubernetes-report-path", k8sReportRel,
		"--kubernetes-stdout-path", k8sStdout.Name(),
		"--kubernetes-stderr-path", k8sStderr.Name(),
		"--ray-report-path", rayReportRel,
		"--ray-stdout-path", rayStdout.Name(),
		"--ray-stderr-path", rayStderr.Name(),
	}
	report, exportStatus, err := automationRunJSONCommand(root, nil, exportArgs...)
	if err != nil {
		return nil, 0, err
	}

	if !opts.RefreshContinuation {
		return report, exportStatus, nil
	}

	if _, _, err := automationRunJSONCommand(root, nil,
		"run", filepath.Join(root, "cmd", "bigclawctl"), "automation", "e2e", "continuation-scorecard",
		"--go-root", root,
		"--output", scorecardPath,
	); err != nil {
		return nil, 0, err
	}

	_, gateStatus, err := automationRunJSONCommand(root, nil,
		"run", filepath.Join(root, "cmd", "bigclawctl"), "automation", "e2e", "continuation-policy-gate",
		"--go-root", root,
		"--scorecard", scorecardPath,
		"--enforcement-mode", continuationMode,
		"--output", policyGatePath,
	)
	if err != nil {
		return nil, 0, err
	}

	rerenderReport, rerenderStatus, err := automationRunJSONCommand(root, nil, exportArgs...)
	if err != nil {
		return nil, 0, err
	}
	if exportStatus == 0 && rerenderStatus != 0 {
		exportStatus = rerenderStatus
	}
	report = rerenderReport

	if gateStatus != 0 {
		return report, gateStatus, nil
	}
	return report, exportStatus, nil
}

func automationRunExternalCommand(root string, env map[string]string, stdoutFile, stderrFile *os.File, args ...string) (int, error) {
	command := exec.Command("go", args...)
	command.Dir = root
	command.Stdout = stdoutFile
	command.Stderr = stderrFile
	command.Env = automationCommandEnv(env)
	if err := command.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func automationRunJSONCommand(root string, env map[string]string, args ...string) (map[string]any, int, error) {
	command := exec.Command("go", args...)
	command.Dir = root
	command.Env = automationCommandEnv(env)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, 0, err
		}
	}
	report := map[string]any{}
	if trim(stdout.String()) != "" {
		if decodeErr := json.Unmarshal(stdout.Bytes(), &report); decodeErr != nil {
			return nil, 0, fmt.Errorf("decode command output: %w", decodeErr)
		}
	}
	if exitCode != 0 && len(report) == 0 {
		message := trim(stderr.String())
		if message == "" {
			message = "external command failed"
		}
		return nil, 0, errors.New(message)
	}
	return report, exitCode, nil
}

func automationCommandEnv(overrides map[string]string) []string {
	env := os.Environ()
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := strings.ToLower(trim(os.Getenv(key)))
	switch value {
	case "", "default":
		return fallback
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func normalizeLegacyGoRootRelativePath(root, value string) string {
	trimmed := trim(value)
	if trimmed == "" {
		return ""
	}
	prefix := filepath.Base(root) + string(filepath.Separator)
	if strings.HasPrefix(trimmed, prefix) {
		return strings.TrimPrefix(trimmed, prefix)
	}
	return trimmed
}
