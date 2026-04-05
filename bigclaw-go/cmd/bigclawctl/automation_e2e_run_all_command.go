package main

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/config"
	"bigclaw-go/internal/events"
)

type automationRunAllOptions struct {
	GoRoot                     string
	SummaryReportPath          string
	IndexReportPath            string
	ManifestReportPath         string
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
	ExecCommand                func(name string, args ...string) *exec.Cmd
	MakeTempFile               func(pattern string) (*os.File, error)
}

func runAutomationRunAllCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e run-all", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	summaryPath := flags.String("summary-path", envOrDefault("BIGCLAW_E2E_SUMMARY_REPORT_PATH", "docs/reports/live-validation-summary.json"), "summary output path")
	indexPath := flags.String("index-path", envOrDefault("BIGCLAW_E2E_INDEX_PATH", "docs/reports/live-validation-index.md"), "index output path")
	manifestPath := flags.String("manifest-path", envOrDefault("BIGCLAW_E2E_MANIFEST_PATH", "docs/reports/live-validation-index.json"), "manifest output path")
	artifactRoot := flags.String("artifact-root", envOrDefault("BIGCLAW_E2E_ARTIFACT_ROOT", "docs/reports/live-validation-runs"), "bundle artifact root")
	runID := flags.String("run-id", envOrDefault("BIGCLAW_E2E_RUN_ID", time.Now().UTC().Format("20060102T150405Z")), "bundle run id")
	runLocal := flags.Bool("run-local", envBoolOrDefault("BIGCLAW_E2E_RUN_LOCAL", true), "run local lane")
	runKubernetes := flags.Bool("run-kubernetes", envBoolOrDefault("BIGCLAW_E2E_RUN_KUBERNETES", true), "run kubernetes lane")
	runRay := flags.Bool("run-ray", envBoolOrDefault("BIGCLAW_E2E_RUN_RAY", true), "run ray lane")
	runBroker := flags.Bool("run-broker", envBoolOrDefault("BIGCLAW_E2E_RUN_BROKER", false), "include broker evidence")
	brokerBackend := flags.String("broker-backend", envOrDefault("BIGCLAW_E2E_BROKER_BACKEND", ""), "broker backend")
	brokerReportPath := flags.String("broker-report-path", envOrDefault("BIGCLAW_E2E_BROKER_REPORT_PATH", ""), "broker report path")
	brokerBootstrapSummaryPath := flags.String("broker-bootstrap-summary-path", envOrDefault("BIGCLAW_E2E_BROKER_BOOTSTRAP_SUMMARY_PATH", "docs/reports/broker-bootstrap-review-summary.json"), "broker bootstrap summary path")
	refreshContinuation := flags.Bool("refresh-continuation", envBoolOrDefault("BIGCLAW_E2E_REFRESH_CONTINUATION", true), "refresh continuation artifacts")
	enforceContinuationGate := flags.Bool("enforce-continuation-gate", envBoolOrDefault("BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE", false), "legacy alias for fail-mode continuation gate")
	continuationGateMode := flags.String("continuation-gate-mode", envOrDefault("BIGCLAW_E2E_CONTINUATION_GATE_MODE", ""), "continuation gate mode: review|hold|fail")
	continuationScorecardPath := flags.String("continuation-scorecard-path", envOrDefault("BIGCLAW_E2E_CONTINUATION_SCORECARD_PATH", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json"), "continuation scorecard output path")
	continuationPolicyGatePath := flags.String("continuation-policy-gate-path", envOrDefault("BIGCLAW_E2E_CONTINUATION_POLICY_GATE_PATH", "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json"), "continuation gate output path")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e run-all [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}

	return automationRunAll(automationRunAllOptions{
		GoRoot:                     absPath(*goRoot),
		SummaryReportPath:          *summaryPath,
		IndexReportPath:            *indexPath,
		ManifestReportPath:         *manifestPath,
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
	})
}

func runAutomationBrokerBootstrapSummaryCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e broker-bootstrap-summary", flag.ContinueOnError)
	outputPath := flags.String("output", "docs/reports/broker-bootstrap-review-summary.json", "output path for the broker bootstrap review summary")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e broker-bootstrap-summary [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationBrokerBootstrapSummary(*outputPath)
	if err != nil {
		return err
	}
	return emit(report, *asJSON, 0)
}

func automationRunAll(opts automationRunAllOptions) error {
	root := absPath(opts.GoRoot)
	if root == "" {
		root = absPath(".")
	}
	execCommand := opts.ExecCommand
	if execCommand == nil {
		execCommand = exec.Command
	}
	makeTempFile := opts.MakeTempFile
	if makeTempFile == nil {
		makeTempFile = func(pattern string) (*os.File, error) {
			return os.CreateTemp("", pattern)
		}
	}

	gateMode := trim(opts.ContinuationGateMode)
	if gateMode == "" {
		if opts.EnforceContinuationGate {
			gateMode = "fail"
		} else {
			gateMode = "hold"
		}
	}

	bundleDirRel := filepath.ToSlash(filepath.Join(opts.ArtifactRootRel, opts.RunID))
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(bundleDirRel)), 0o755); err != nil {
		return err
	}

	localReportRel := filepath.ToSlash(filepath.Join(bundleDirRel, "sqlite-smoke-report.json"))
	k8sReportRel := filepath.ToSlash(filepath.Join(bundleDirRel, "kubernetes-live-smoke-report.json"))
	rayReportRel := filepath.ToSlash(filepath.Join(bundleDirRel, "ray-live-smoke-report.json"))

	localOut, err := makeTempFile("bigclaw-local-e2e-out.*")
	if err != nil {
		return err
	}
	defer os.Remove(localOut.Name())
	defer localOut.Close()
	localErr, err := makeTempFile("bigclaw-local-e2e-err.*")
	if err != nil {
		return err
	}
	defer os.Remove(localErr.Name())
	defer localErr.Close()
	k8sOut, err := makeTempFile("bigclaw-k8s-e2e-out.*")
	if err != nil {
		return err
	}
	defer os.Remove(k8sOut.Name())
	defer k8sOut.Close()
	k8sErr, err := makeTempFile("bigclaw-k8s-e2e-err.*")
	if err != nil {
		return err
	}
	defer os.Remove(k8sErr.Name())
	defer k8sErr.Close()
	rayOut, err := makeTempFile("bigclaw-ray-e2e-out.*")
	if err != nil {
		return err
	}
	defer os.Remove(rayOut.Name())
	defer rayOut.Close()
	rayErr, err := makeTempFile("bigclaw-ray-e2e-err.*")
	if err != nil {
		return err
	}
	defer os.Remove(rayErr.Name())
	defer rayErr.Close()

	status := 0
	var waiters []func() error

	if opts.RunLocal {
		cmd := execCommand("go", "run", filepath.Join(root, "cmd/bigclawctl"), "automation", "e2e", "run-task-smoke",
			"--autostart",
			"--go-root", root,
			"--executor", "local",
			"--title", "SQLite smoke",
			"--entrypoint", "echo hello from sqlite",
			"--report-path", localReportRel,
		)
		cmd.Stdout = localOut
		cmd.Stderr = localErr
		cmd.Dir = root
		if err := cmd.Start(); err != nil {
			return err
		}
		waiters = append(waiters, cmd.Wait)
	}

	if opts.RunKubernetes {
		cmd := execCommand(filepath.Join(root, "scripts/e2e/kubernetes_smoke.sh"))
		cmd.Stdout = k8sOut
		cmd.Stderr = k8sErr
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "BIGCLAW_KUBERNETES_SMOKE_REPORT_PATH="+k8sReportRel)
		if err := cmd.Start(); err != nil {
			return err
		}
		waiters = append(waiters, cmd.Wait)
	}

	if opts.RunRay {
		cmd := execCommand(filepath.Join(root, "scripts/e2e/ray_smoke.sh"))
		cmd.Stdout = rayOut
		cmd.Stderr = rayErr
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "BIGCLAW_RAY_SMOKE_REPORT_PATH="+rayReportRel)
		if err := cmd.Start(); err != nil {
			return err
		}
		waiters = append(waiters, cmd.Wait)
	}

	for _, wait := range waiters {
		if err := wait(); err != nil {
			status = 1
		}
	}

	exportBundle := func() error {
		brokerBootstrapOutput := opts.BrokerBootstrapSummaryPath
		if !filepath.IsAbs(brokerBootstrapOutput) {
			brokerBootstrapOutput = filepath.Join(root, filepath.FromSlash(brokerBootstrapOutput))
		}
		if _, err := automationBrokerBootstrapSummary(brokerBootstrapOutput); err != nil {
			return err
		}
		cmd := execCommand("go", "run", filepath.Join(root, "cmd/bigclawctl"), "automation", "e2e", "export-validation-bundle",
			"--go-root", root,
			"--run-id", opts.RunID,
			"--bundle-dir", bundleDirRel,
			"--summary-path", opts.SummaryReportPath,
			"--index-path", opts.IndexReportPath,
			"--manifest-path", opts.ManifestReportPath,
			"--run-local", strconv.FormatBool(opts.RunLocal),
			"--run-kubernetes", strconv.FormatBool(opts.RunKubernetes),
			"--run-ray", strconv.FormatBool(opts.RunRay),
			"--run-broker", strconv.FormatBool(opts.RunBroker),
			"--broker-backend", opts.BrokerBackend,
			"--broker-report-path", opts.BrokerReportPath,
			"--broker-bootstrap-summary-path", opts.BrokerBootstrapSummaryPath,
			"--validation-status", strconv.Itoa(status),
			"--local-report-path", localReportRel,
			"--local-stdout-path", localOut.Name(),
			"--local-stderr-path", localErr.Name(),
			"--kubernetes-report-path", k8sReportRel,
			"--kubernetes-stdout-path", k8sOut.Name(),
			"--kubernetes-stderr-path", k8sErr.Name(),
			"--ray-report-path", rayReportRel,
			"--ray-stdout-path", rayOut.Name(),
			"--ray-stderr-path", rayErr.Name(),
		)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	exportStatus := 0
	if err := exportBundle(); err != nil {
		exportStatus = commandExitCode(err)
		if exportStatus == 0 {
			return err
		}
	}

	if opts.RefreshContinuation {
		scorecardCmd := execCommand("go", "run", filepath.Join(root, "cmd/bigclawctl"), "automation", "e2e", "continuation-scorecard",
			"--go-root", root,
			"--output", opts.ContinuationScorecardPath,
		)
		scorecardCmd.Dir = root
		scorecardCmd.Stdout = os.Stdout
		scorecardCmd.Stderr = os.Stderr
		if err := scorecardCmd.Run(); err != nil {
			return err
		}

		gateStatus := 0
		gateCmd := execCommand("go", "run", filepath.Join(root, "cmd/bigclawctl"), "automation", "e2e", "continuation-policy-gate",
			"--go-root", root,
			"--scorecard", opts.ContinuationScorecardPath,
			"--enforcement-mode", gateMode,
			"--output", opts.ContinuationPolicyGatePath,
		)
		gateCmd.Dir = root
		gateCmd.Stdout = os.Stdout
		gateCmd.Stderr = os.Stderr
		if err := gateCmd.Run(); err != nil {
			gateStatus = commandExitCode(err)
			if gateStatus == 0 {
				return err
			}
		}

		rerenderStatus := 0
		if err := exportBundle(); err != nil {
			rerenderStatus = commandExitCode(err)
			if rerenderStatus == 0 {
				return err
			}
		}
		if exportStatus == 0 && rerenderStatus != 0 {
			exportStatus = rerenderStatus
		}
		if gateStatus != 0 {
			return exitError(gateStatus)
		}
	}

	if exportStatus != 0 {
		return exitError(exportStatus)
	}
	return nil
}

func automationBrokerBootstrapSummary(outputPath string) (map[string]any, error) {
	cfg := config.LoadFromEnv()
	summary := events.BrokerBootstrapReviewSummaryFromConfig(
		cfg.EventLogBackend,
		cfg.EventLogTargetBackend,
		events.BrokerRuntimeConfig{
			Driver:             cfg.EventLogBrokerDriver,
			URLs:               cfg.EventLogBrokerURLs,
			Topic:              cfg.EventLogBrokerTopic,
			ConsumerGroup:      cfg.EventLogConsumerGroup,
			PublishTimeout:     cfg.EventLogPublishTimeout,
			ReplayLimit:        cfg.EventLogReplayLimit,
			CheckpointInterval: cfg.EventLogCheckpointInterval,
		},
	)
	body, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return nil, err
	}
	resolvedOutput := outputPath
	if !filepath.IsAbs(resolvedOutput) {
		resolvedOutput = filepath.Join(".", resolvedOutput)
	}
	if err := os.MkdirAll(filepath.Dir(resolvedOutput), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(resolvedOutput, append(body, '\n'), 0o644); err != nil {
		return nil, err
	}
	report := map[string]any{}
	if err := json.Unmarshal(body, &report); err != nil {
		return nil, err
	}
	report["output_path"] = outputPath
	return report, nil
}

func envBoolOrDefault(key string, fallback bool) bool {
	value := trim(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.ToLower(value))
	if err == nil {
		return parsed
	}
	return value == "1"
}

func commandExitCode(err error) int {
	if err == nil {
		return 0
	}
	var code exitError
	if errors.As(err, &code) {
		return int(code)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 0
}
