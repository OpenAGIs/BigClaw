package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bigclaw-go/internal/reporting"
)

func main() {
	goRoot := flag.String("go-root", "", "BigClaw repo or bigclaw-go root")
	runID := flag.String("run-id", "", "bundle run id")
	bundleDir := flag.String("bundle-dir", "", "bundle directory")
	summaryPath := flag.String("summary-path", "docs/reports/live-validation-summary.json", "summary path")
	indexPath := flag.String("index-path", "docs/reports/live-validation-index.md", "index path")
	manifestPath := flag.String("manifest-path", "docs/reports/live-validation-index.json", "manifest path")
	runLocal := flag.String("run-local", "1", "whether local validation ran")
	runKubernetes := flag.String("run-kubernetes", "1", "whether kubernetes validation ran")
	runRay := flag.String("run-ray", "1", "whether ray validation ran")
	validationStatus := flag.Int("validation-status", 0, "validation exit status")
	runBroker := flag.String("run-broker", "0", "whether broker validation ran")
	brokerBackend := flag.String("broker-backend", "", "broker backend")
	brokerReportPath := flag.String("broker-report-path", "", "broker report path")
	brokerBootstrapSummaryPath := flag.String("broker-bootstrap-summary-path", "", "broker bootstrap summary path")
	localReportPath := flag.String("local-report-path", "", "local report path")
	localStdoutPath := flag.String("local-stdout-path", "", "local stdout path")
	localStderrPath := flag.String("local-stderr-path", "", "local stderr path")
	kubernetesReportPath := flag.String("kubernetes-report-path", "", "kubernetes report path")
	kubernetesStdoutPath := flag.String("kubernetes-stdout-path", "", "kubernetes stdout path")
	kubernetesStderrPath := flag.String("kubernetes-stderr-path", "", "kubernetes stderr path")
	rayReportPath := flag.String("ray-report-path", "", "ray report path")
	rayStdoutPath := flag.String("ray-stdout-path", "", "ray stdout path")
	rayStderrPath := flag.String("ray-stderr-path", "", "ray stderr path")
	flag.Parse()

	root, err := resolveRoot(*goRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	summary, _, _, err := reporting.ExportLiveValidationBundle(root, reporting.LiveValidationBundleOptions{
		RunID:                      *runID,
		BundleDir:                  *bundleDir,
		SummaryPath:                *summaryPath,
		IndexPath:                  *indexPath,
		ManifestPath:               *manifestPath,
		RunLocal:                   flagEnabled(*runLocal),
		RunKubernetes:              flagEnabled(*runKubernetes),
		RunRay:                     flagEnabled(*runRay),
		ValidationStatus:           *validationStatus,
		RunBroker:                  flagEnabled(*runBroker),
		BrokerBackend:              *brokerBackend,
		BrokerReportPath:           *brokerReportPath,
		BrokerBootstrapSummaryPath: *brokerBootstrapSummaryPath,
		LocalReportPath:            *localReportPath,
		LocalStdoutPath:            *localStdoutPath,
		LocalStderrPath:            *localStderrPath,
		KubernetesReportPath:       *kubernetesReportPath,
		KubernetesStdoutPath:       *kubernetesStdoutPath,
		KubernetesStderrPath:       *kubernetesStderrPath,
		RayReportPath:              *rayReportPath,
		RayStdoutPath:              *rayStdoutPath,
		RayStderrPath:              *rayStderrPath,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	contents, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(contents))
	if *validationStatus != 0 {
		os.Exit(1)
	}
}

func resolveRoot(value string) (string, error) {
	if value != "" {
		if filepath.IsAbs(value) {
			return value, nil
		}
		return filepath.Abs(value)
	}
	root, err := reporting.FindRepoRoot(".")
	if err == nil {
		return root, nil
	}
	return reporting.FindRepoRoot(filepath.Dir(os.Args[0]))
}

func flagEnabled(value string) bool {
	return value == "1"
}
