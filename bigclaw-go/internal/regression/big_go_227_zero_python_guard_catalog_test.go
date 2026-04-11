package regression

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type zeroPythonGuardCase struct {
	ticket        string
	repoScan      bool
	pythonDirs    []string
	existingPaths []string
	absentPaths   []string
	reportPath    string
	reportNeedles []string
}

var zeroPythonGuardCatalog = []zeroPythonGuardCase{
	{
		ticket:        "100",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-100-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-100", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO100"},
	},
	{
		ticket:        "109",
		repoScan:      true,
		pythonDirs:    []string{".githooks", ".github", ".symphony", "bigclaw-go/examples", "bigclaw-go/docs/reports/live-validation-runs", "reports"},
		existingPaths: []string{".githooks/post-commit", ".githooks/post-rewrite", ".github/workflows/ci.yml", "bigclaw-go/examples/shadow-task.json", "bigclaw-go/examples/shadow-corpus-manifest.json"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-109-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-109", "Repository-wide Python file count: `0`.", "`.githooks`: `0` Python files", "`.github`: `0` Python files", "`.symphony`: `0` Python files", "`bigclaw-go/examples`: `0` Python files", "`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files", "`reports`: `0` Python files", "`.githooks/post-commit`", "`.githooks/post-rewrite`", "`.github/workflows/ci.yml`", "`bigclaw-go/examples/shadow-task.json`", "`bigclaw-go/examples/shadow-corpus-manifest.json`", "`find . -path '*/.git' -prune -o -type f \\\\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\\\) -print | sort`", "`find .githooks .github .symphony bigclaw-go/examples bigclaw-go/docs/reports/live-validation-runs reports -type f \\\\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\\\) 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO109"},
	},
	{
		ticket:        "113",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-113-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-113", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO113(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "1174",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1183",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1185",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1188",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1193",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1194",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1197",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1199",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "119",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts", ".github", ".githooks", ".symphony", "docs", "bigclaw-go/docs", "bigclaw-go/examples", "scripts/ops"},
		existingPaths: []string{".github/workflows/ci.yml", ".githooks/post-commit", "scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/docs/go-cli-script-migration.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-119-python-asset-sweep.md",
		reportNeedles: []string{"# BIG-GO-119 Python Asset Sweep", "Remaining physical Python asset inventory: `0` files.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Hidden and lower-priority directories audited in this lane:", "`.github`: `0` Python files", "`.githooks`: `0` Python files", "`.symphony`: `0` Python files", "`docs`: `0` Python files", "`bigclaw-go/docs`: `0` Python files", "`bigclaw-go/examples`: `0` Python files", "`scripts/ops`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/internal/bootstrap/bootstrap.go`", "`bigclaw-go/docs/go-cli-script-migration.md`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find .github .githooks .symphony docs bigclaw-go/docs bigclaw-go/examples scripts/ops -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO119(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|HiddenAndAuxiliaryDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`"},
	},
	{
		ticket:        "11",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-11-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-11", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO11"},
	},
	{
		ticket:        "1202",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1203",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1204",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1205",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1208",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1210",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1212",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1213",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1214",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1215",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1217",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1218",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1220",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1221",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1223",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1224",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1225",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1226",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1228",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1229",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1231",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1232",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1233",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1236",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1238",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1239",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1240",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1241",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/docs/go-cli-script-migration.md"},
		absentPaths:   []string{},
		reportPath:    "docs/reports/big-go-1241-python-asset-sweep.md",
		reportNeedles: []string{"# BIG-GO-1241 Python Asset Sweep", "Remaining physical Python asset inventory: `0` files.", "`src/bigclaw`: directory not present, so residual Python files = `0`", "`tests`: directory not present, so residual Python files = `0`", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/internal/bootstrap/bootstrap.go`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | wc -l`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1241(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`", "Result: `0`", "bigclaw-go/internal/regression"},
	},
	{
		ticket:        "1242",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1243",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1244",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1245",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1246",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1247",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1248",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1249",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1249-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1249", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -type f -name '*.py' | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1249"},
	},
	{
		ticket:        "1250",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/cmd/bigclawctl/main.go", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1252",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1252-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1252", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/internal/bootstrap/bootstrap.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -type f -name '*.py' | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1252", "bigclaw-go/internal/regression"},
	},
	{
		ticket:        "1253",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1253-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1253", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -type f -name '*.py' | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1253"},
	},
	{
		ticket:        "1254",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1254-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1254", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -type f -name '*.py' | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1254"},
	},
	{
		ticket:        "1256",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/internal/bootstrap/bootstrap.go"},
		absentPaths:   []string{},
		reportPath:    "",
		reportNeedles: []string{},
	},
	{
		ticket:        "1260",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1260-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1260", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1260"},
	},
	{
		ticket:        "1262",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1262-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1262", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -type f -name '*.py' | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1262"},
	},
	{
		ticket:        "1264",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1264-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1264", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1264"},
	},
	{
		ticket:        "1265",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1265-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1265", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1265"},
	},
	{
		ticket:        "1271",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1271-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1271", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1271"},
	},
	{
		ticket:        "1272",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1272-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1272", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1272"},
	},
	{
		ticket:        "1273",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1273-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1273", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1273"},
	},
	{
		ticket:        "1274",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1274-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1274", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1274"},
	},
	{
		ticket:        "1275",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1275-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1275", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1275"},
	},
	{
		ticket:        "1278",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1278-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1278", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1278"},
	},
	{
		ticket:        "1279",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1279-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1279", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1279"},
	},
	{
		ticket:        "127",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-127-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-127", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO127"},
	},
	{
		ticket:        "1280",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1280-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1280", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1280"},
	},
	{
		ticket:        "1281",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1281-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1281", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1281"},
	},
	{
		ticket:        "1282",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1282-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1282", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1282"},
	},
	{
		ticket:        "1283",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1283-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1283", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1283"},
	},
	{
		ticket:        "1284",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1284-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1284", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1284"},
	},
	{
		ticket:        "1286",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1286-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1286", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1286"},
	},
	{
		ticket:        "1288",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1288-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1288", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1288"},
	},
	{
		ticket:        "1290",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1290-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1290", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1290"},
	},
	{
		ticket:        "1291",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1291-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1291", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1291"},
	},
	{
		ticket:        "1292",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1292-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1292", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1292"},
	},
	{
		ticket:        "1293",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1293-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1293", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1293"},
	},
	{
		ticket:        "1295",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1295-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1295", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1295"},
	},
	{
		ticket:        "1297",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1297-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1297", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1297"},
	},
	{
		ticket:        "1298",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1298-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1298", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1298"},
	},
	{
		ticket:        "1299",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1299-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1299", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1299"},
	},
	{
		ticket:        "129",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-129-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-129", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO129"},
	},
	{
		ticket:        "1302",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1302-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1302", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1302"},
	},
	{
		ticket:        "1303",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1303-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1303", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1303"},
	},
	{
		ticket:        "1307",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1307-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1307", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1307"},
	},
	{
		ticket:        "1308",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1308-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1308", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1308"},
	},
	{
		ticket:        "1311",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1311-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1311", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1311"},
	},
	{
		ticket:        "1312",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1312-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1312", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1312"},
	},
	{
		ticket:        "1314",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1314-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1314", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1314"},
	},
	{
		ticket:        "1316",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1316-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1316", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1316"},
	},
	{
		ticket:        "1317",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1317-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1317", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1317"},
	},
	{
		ticket:        "1319",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1319-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1319", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1319"},
	},
	{
		ticket:        "1320",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1320-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1320", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1320"},
	},
	{
		ticket:        "1321",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1321-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1321", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1321"},
	},
	{
		ticket:        "1324",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1324-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1324", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1324"},
	},
	{
		ticket:        "1326",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1326-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1326", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1326"},
	},
	{
		ticket:        "1327",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1327-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1327", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1327"},
	},
	{
		ticket:        "1328",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1328-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1328", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1328"},
	},
	{
		ticket:        "1330",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1330-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1330", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1330"},
	},
	{
		ticket:        "1333",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1333-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1333", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1333"},
	},
	{
		ticket:        "1334",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1334-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1334", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1334"},
	},
	{
		ticket:        "1336",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1336-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1336", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1336"},
	},
	{
		ticket:        "1339",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1339-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1339", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1339"},
	},
	{
		ticket:        "1340",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1340-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1340", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1340"},
	},
	{
		ticket:        "1341",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1341-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1341", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1341"},
	},
	{
		ticket:        "1343",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1343-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1343", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1343"},
	},
	{
		ticket:        "1344",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1344-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1344", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1344"},
	},
	{
		ticket:        "1345",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1345-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1345", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1345"},
	},
	{
		ticket:        "1347",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1347-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1347", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1347"},
	},
	{
		ticket:        "1348",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1348-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1348", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1348"},
	},
	{
		ticket:        "1349",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1349-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1349", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1349"},
	},
	{
		ticket:        "1350",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1350-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1350", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1350"},
	},
	{
		ticket:        "1351",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1351-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1351", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1351"},
	},
	{
		ticket:        "1353",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh", "bigclaw-go/scripts/benchmark/run_suite.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1353-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1353", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`bigclaw-go/scripts/benchmark/run_suite.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1353"},
	},
	{
		ticket:        "1359",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"bigclaw-go/scripts/e2e/ray_smoke.sh", "bigclaw-go/docs/e2e-validation.md", "bigclaw-go/cmd/bigclawctl/main.go"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1359-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1359", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`bigclaw-go/scripts/e2e/ray_smoke.sh` now defaults `BIGCLAW_RAY_SMOKE_ENTRYPOINT` to `sh -c 'echo hello from ray'`", "`bigclaw-go/docs/e2e-validation.md` no longer lists `python3` as a prerequisite for the active smoke path", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1359"},
	},
	{
		ticket:        "1360",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1360-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1360", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1360"},
	},
	{
		ticket:        "1370",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1370-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1370", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1370"},
	},
	{
		ticket:        "1371",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1371-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1371", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1371"},
	},
	{
		ticket:        "1374",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1374-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1374", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1374"},
	},
	{
		ticket:        "1376",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1376-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1376", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1376"},
	},
	{
		ticket:        "1377",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1377-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1377", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1377"},
	},
	{
		ticket:        "1380",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1380-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1380", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1380"},
	},
	{
		ticket:        "1381",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1381-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1381", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1381"},
	},
	{
		ticket:        "1382",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1382-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1382", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1382"},
	},
	{
		ticket:        "1383",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1383-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1383", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1383"},
	},
	{
		ticket:        "1385",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1385-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1385", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1385"},
	},
	{
		ticket:        "1388",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1388-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1388", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1388"},
	},
	{
		ticket:        "1389",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1389-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1389", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1389"},
	},
	{
		ticket:        "1391",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1391-python-asset-sweep.md",
		reportNeedles: []string{},
	},
	{
		ticket:        "1392",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1392-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1392", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1392"},
	},
	{
		ticket:        "1393",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1393-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1393", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1393"},
	},
	{
		ticket:        "1395",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1395-python-asset-sweep.md",
		reportNeedles: []string{},
	},
	{
		ticket:        "1396",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1396-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1396", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1396"},
	},
	{
		ticket:        "1398",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1398-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1398", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1398"},
	},
	{
		ticket:        "139",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts", "reports", "docs/reports", "bigclaw-go/docs/reports", "bigclaw-go/docs/reports/live-shadow-runs", "bigclaw-go/docs/reports/live-validation-runs"},
		existingPaths: []string{"reports/BIG-GO-1274-validation.md", "docs/reports/bootstrap-cache-validation.md", "bigclaw-go/docs/reports/live-shadow-index.md", "bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json", "bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-139-python-asset-sweep.md",
		reportNeedles: []string{"# BIG-GO-139 Python Asset Sweep", "Remaining physical Python asset inventory: `0` files.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Report-heavy auxiliary directories audited in this lane:", "`reports`: `0` Python files", "`docs/reports`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python files", "`bigclaw-go/docs/reports/live-validation-runs`: `0` Python files", "`reports/BIG-GO-1274-validation.md`", "`docs/reports/bootstrap-cache-validation.md`", "`bigclaw-go/docs/reports/live-shadow-index.md`", "`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`", "`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find reports docs/reports bigclaw-go/docs/reports bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO139(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReportHeavyAuxiliaryDirectoriesStayPythonFree|RetainedNativeReportAssetsRemainAvailable|LaneReportDocumentsPythonAssetSweep)$'`"},
	},
	{
		ticket:        "1400",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1400-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1400", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1400"},
	},
	{
		ticket:        "1401",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1401-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1401", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1401"},
	},
	{
		ticket:        "1402",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1402-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1402", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1402"},
	},
	{
		ticket:        "1406",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1406-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1406", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1406"},
	},
	{
		ticket:        "1407",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1407-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1407", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1407"},
	},
	{
		ticket:        "1409",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1409-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1409", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1409"},
	},
	{
		ticket:        "1410",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1410-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1410", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1410"},
	},
	{
		ticket:        "1412",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1412-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1412", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1412"},
	},
	{
		ticket:        "1419",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1419-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1419", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1419"},
	},
	{
		ticket:        "1422",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1422-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1422", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1422"},
	},
	{
		ticket:        "1424",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1424-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1424", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1424"},
	},
	{
		ticket:        "1425",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1425-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1425", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1425"},
	},
	{
		ticket:        "1426",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1426-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1426", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1426"},
	},
	{
		ticket:        "1427",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1427-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1427", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1427"},
	},
	{
		ticket:        "142",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-142-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-142", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO142(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "1430",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1430-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1430", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1430"},
	},
	{
		ticket:        "1433",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1433-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1433", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1433"},
	},
	{
		ticket:        "1436",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1436-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1436", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1436"},
	},
	{
		ticket:        "1439",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1439-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1439", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1439"},
	},
	{
		ticket:        "143",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-143-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-143", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO143(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "1454",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1454-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1454", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1454"},
	},
	{
		ticket:        "1516",
		repoScan:      true,
		pythonDirs:    []string{"workspace", "bootstrap", "planning", "bigclaw-go/internal/bootstrap", "bigclaw-go/internal/planning"},
		existingPaths: []string{"docs/symphony-repo-bootstrap-template.md", "scripts/dev_bootstrap.sh", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/internal/planning/planning.go", "bigclaw-go/internal/api/broker_bootstrap_surface.go"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1516-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1516", "Repository-wide physical Python file count before lane changes: `0`", "Repository-wide physical Python file count after lane changes: `0`", "Focused `workspace/bootstrap/planning` physical Python file count before lane", "Focused `workspace/bootstrap/planning` physical Python file count after lane", "Deleted files in this lane: `[]`", "Focused ledger for `workspace/bootstrap/planning`: `[]`", "`workspace`: directory not present, so residual Python files = `0`", "`bootstrap`: directory not present, so residual Python files = `0`", "`planning`: directory not present, so residual Python files = `0`", "`bigclaw-go/internal/bootstrap`: `0` Python files", "`bigclaw-go/internal/planning`: `0` Python files", "`docs/symphony-repo-bootstrap-template.md`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/internal/bootstrap/bootstrap.go`", "`bigclaw-go/internal/planning/planning.go`", "`bigclaw-go/internal/api/broker_bootstrap_surface.go`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find workspace bootstrap planning bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1516"},
	},
	{
		ticket:        "152",
		repoScan:      true,
		pythonDirs:    []string{"tests", "bigclaw-go/scripts", "bigclaw-go/internal/migration", "bigclaw-go/internal/regression", "bigclaw-go/docs/reports"},
		existingPaths: []string{"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go", "bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go", "bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go", "bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md", "bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md", "bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md", "reports/BIG-GO-948-validation.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-152-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-152", "Repository-wide Python file count: `0`.", "`tests`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`bigclaw-go/internal/migration`: `0` Python files", "`bigclaw-go/internal/regression`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`reports/BIG-GO-948-validation.md`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go`", "`bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`", "`bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md`", "`bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md`", "`bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find tests bigclaw-go/scripts bigclaw-go/internal/migration bigclaw-go/internal/regression bigclaw-go/docs/reports -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO152(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "1562",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/scheduler/scheduler.go", "bigclaw-go/internal/worker/runtime.go", "bigclaw-go/internal/orchestrator/loop.go", "bigclaw-go/internal/queue/queue.go", "bigclaw-go/internal/control/controller.go", "docs/go-mainline-cutover-issue-pack.md"},
		absentPaths:   []string{"src/bigclaw/runtime.py", "src/bigclaw/scheduler.py", "src/bigclaw/orchestration.py", "src/bigclaw/workflow.py", "src/bigclaw/queue.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-1562-src-bigclaw-tranche-b.md",
		reportNeedles: []string{"BIG-GO-1562", "Repository-wide physical Python file count before lane changes: `0`", "Repository-wide physical Python file count after lane changes: `0`", "Focused `src/bigclaw` tranche-B physical Python file count before lane changes: `0`", "Focused `src/bigclaw` tranche-B physical Python file count after lane changes: `0`", "Deleted files in this lane: `[]`", "Focused tranche-B ledger: `[]`", "`src/bigclaw`: directory not present, so residual Python files = `0`", "`src/bigclaw/runtime.py`", "`src/bigclaw/scheduler.py`", "`src/bigclaw/orchestration.py`", "`src/bigclaw/workflow.py`", "`src/bigclaw/queue.py`", "`bigclaw-go/internal/scheduler/scheduler.go`", "`bigclaw-go/internal/worker/runtime.go`", "`bigclaw-go/internal/orchestrator/loop.go`", "`bigclaw-go/internal/queue/queue.go`", "`bigclaw-go/internal/control/controller.go`", "`docs/go-mainline-cutover-issue-pack.md`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1562"},
	},
	{
		ticket:        "1577",
		repoScan:      false,
		pythonDirs:    []string{},
		existingPaths: []string{"scripts/ops/bigclawctl", "bigclaw-go/internal/intake/mapping.go", "bigclaw-go/internal/repo/board.go", "bigclaw-go/internal/repo/triage.go", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go", "bigclaw-go/cmd/bigclawctl/automation_commands.go", "bigclaw-go/cmd/bigclawctl/automation_commands_test.go"},
		absentPaths:   []string{"src/bigclaw/cost_control.py", "src/bigclaw/mapping.py", "src/bigclaw/repo_board.py", "src/bigclaw/roadmap.py", "src/bigclaw/workspace_bootstrap_cli.py", "tests/test_design_system.py", "tests/test_live_shadow_bundle.py", "tests/test_pilot.py", "tests/test_repo_triage.py", "tests/test_subscriber_takeover_harness.py", "scripts/ops/symphony_workspace_bootstrap.py", "bigclaw-go/scripts/e2e/export_validation_bundle_test.py", "bigclaw-go/scripts/migration/export_live_shadow_bundle.py"},
		reportPath:    "docs/reports/big-go-1577-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1577", "`src/bigclaw/cost_control.py`", "`src/bigclaw/mapping.py`", "`src/bigclaw/repo_board.py`", "`src/bigclaw/roadmap.py`", "`src/bigclaw/workspace_bootstrap_cli.py`", "`scripts/ops/symphony_workspace_bootstrap.py`", "`bigclaw-go/scripts/e2e/export_validation_bundle_test.py`", "`bigclaw-go/scripts/migration/export_live_shadow_bundle.py`", "`scripts/ops/bigclawctl`", "`bigclaw-go/internal/intake/mapping.go`", "`bigclaw-go/internal/repo/board.go`", "`bigclaw-go/internal/repo/triage.go`", "`bigclaw-go/internal/bootstrap/bootstrap.go`", "`bigclaw-go/cmd/bigclawctl/automation_commands.go`", "`bigclaw-go/cmd/bigclawctl/automation_commands_test.go`", "`bigclawctl automation migration export-live-shadow-bundle`", "`find src/bigclaw tests scripts/ops bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1577"},
	},
	{
		ticket:        "157",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts", "docs", "docs/reports", "reports", "scripts", "bigclaw-go/scripts", "bigclaw-go/docs/reports", "bigclaw-go/examples"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh", "bigclaw-go/scripts/benchmark/run_suite.sh", "bigclaw-go/docs/reports/review-readiness.md", "docs/reports"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-157-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-157", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`docs`: `0` Python files", "`docs/reports`: `0` Python files", "`reports`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`bigclaw-go/examples`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`bigclaw-go/scripts/benchmark/run_suite.sh`", "`bigclaw-go/docs/reports/review-readiness.md`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO157"},
	},
	{
		ticket:        "1586",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/scripts/benchmark/run_suite.sh", "bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go", "bigclaw-go/cmd/bigclawctl/automation_commands_test.go", "bigclaw-go/internal/queue/benchmark_test.go", "bigclaw-go/internal/scheduler/benchmark_test.go", "bigclaw-go/docs/reports/benchmark-report.md", "bigclaw-go/docs/reports/benchmark-matrix-report.json", "bigclaw-go/docs/reports/long-duration-soak-report.md", "bigclaw-go/docs/reports/soak-local-50x8.json"},
		absentPaths:   []string{"bigclaw-go/scripts/benchmark/capacity_certification.py", "bigclaw-go/scripts/benchmark/capacity_certification_test.py", "bigclaw-go/scripts/benchmark/run_matrix.py", "bigclaw-go/scripts/benchmark/soak_local.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-1586-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1586", "Repository-wide Python file count: `0`.", "`bigclaw-go/scripts/benchmark`: `0` Python files", "`bigclaw-go/scripts/benchmark/capacity_certification.py`", "`bigclaw-go/scripts/benchmark/capacity_certification_test.py`", "`bigclaw-go/scripts/benchmark/run_matrix.py`", "`bigclaw-go/scripts/benchmark/soak_local.py`", "`bigclaw-go/scripts/benchmark/run_suite.sh`", "`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`", "`bigclaw-go/internal/queue/benchmark_test.go`", "`bigclaw-go/internal/scheduler/benchmark_test.go`", "`bigclaw-go/docs/reports/benchmark-report.md`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find bigclaw-go/scripts/benchmark -type f -name '*.py' -print | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1586(RepositoryHasNoPythonFiles|BenchmarkBucketStaysPythonFree|RetiredBenchmarkPythonHelpersRemainAbsent|BenchmarkReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "1587",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawctl/automation_commands.go", "bigclaw-go/docs/go-cli-script-migration.md", "docs/go-cli-script-migration-plan.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-1587-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1587", "Repository-wide Python file count: `0`.", "`bigclaw-go`: `0` Python files", "`bigclaw-go/scripts/migration`: `0` Python files", "Directory absent on disk: `yes`.", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawctl/automation_commands.go`", "`bigclaw-go/docs/go-cli-script-migration.md`", "`docs/go-cli-script-migration-plan.md`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`", "`test ! -d bigclaw-go/scripts/migration`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1587"},
	},
	{
		ticket:        "1594",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/collaboration/thread.go", "bigclaw-go/internal/githubsync/sync.go", "bigclaw-go/internal/pilot/report.go", "bigclaw-go/internal/repo/triage.go", "bigclaw-go/internal/policy/validation.go", "bigclaw-go/internal/costcontrol/controller_test.go", "bigclaw-go/internal/githubsync/sync_test.go", "bigclaw-go/internal/workflow/orchestration_test.go", "scripts/ops/bigclawctl"},
		absentPaths:   []string{"src/bigclaw/collaboration.py", "src/bigclaw/github_sync.py", "src/bigclaw/pilot.py", "src/bigclaw/repo_triage.py", "src/bigclaw/validation_policy.py", "tests/test_cost_control.py", "tests/test_github_sync.py", "tests/test_orchestration.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-1594-go-only-sweep-refill.md",
		reportNeedles: []string{"BIG-GO-1594", "Repository-wide Python file count before lane changes: `0`.", "Repository-wide Python file count after lane changes: `0`.", "Explicit remaining Python asset list: none.", "`src/bigclaw/collaboration.py` -> `bigclaw-go/internal/collaboration/thread.go`", "`src/bigclaw/github_sync.py` -> `bigclaw-go/internal/githubsync/sync.go`", "`src/bigclaw/pilot.py` -> `bigclaw-go/internal/pilot/report.go`", "`src/bigclaw/repo_triage.py` -> `bigclaw-go/internal/repo/triage.go`", "`src/bigclaw/validation_policy.py` -> `bigclaw-go/internal/policy/validation.go`", "`tests/test_cost_control.py` -> `bigclaw-go/internal/costcontrol/controller_test.go`", "`tests/test_github_sync.py` -> `bigclaw-go/internal/githubsync/sync_test.go`", "`tests/test_orchestration.py` -> `bigclaw-go/internal/workflow/orchestration_test.go`", "`scripts/ops/bigclawctl`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1594(RepositoryHasNoPythonFiles|AssignedPythonAssetsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`", "Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1594 hardens that baseline rather than lowering the numeric file count further."},
	},
	{
		ticket:        "1596",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/consoleia/consoleia.go", "bigclaw-go/internal/consoleia/consoleia_test.go", "bigclaw-go/internal/issuearchive/archive.go", "bigclaw-go/internal/issuearchive/archive_test.go", "bigclaw-go/internal/queue/queue.go", "bigclaw-go/internal/risk/risk.go", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/internal/product/dashboard_run_contract_test.go", "bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go", "scripts/ops/bigclawctl"},
		absentPaths:   []string{"src/bigclaw/console_ia.py", "src/bigclaw/issue_archive.py", "src/bigclaw/queue.py", "src/bigclaw/risk.py", "src/bigclaw/workspace_bootstrap.py", "tests/test_dashboard_run_contract.py", "tests/test_issue_archive.py", "tests/test_parallel_validation_bundle.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-1596-go-only-sweep-refill.md",
		reportNeedles: []string{"BIG-GO-1596", "Repository-wide Python file count before lane changes: `0`.", "Repository-wide Python file count after lane changes: `0`.", "Explicit remaining Python asset list: none.", "`src/bigclaw/console_ia.py` -> `bigclaw-go/internal/consoleia/consoleia.go`", "`src/bigclaw/issue_archive.py` -> `bigclaw-go/internal/issuearchive/archive.go`", "`src/bigclaw/queue.py` -> `bigclaw-go/internal/queue/queue.go`", "`src/bigclaw/risk.py` -> `bigclaw-go/internal/risk/risk.go`", "`src/bigclaw/workspace_bootstrap.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go`", "`tests/test_dashboard_run_contract.py` -> `bigclaw-go/internal/product/dashboard_run_contract_test.go`", "`tests/test_issue_archive.py` -> `bigclaw-go/internal/issuearchive/archive_test.go`", "`tests/test_parallel_validation_bundle.py` -> `bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`", "`scripts/ops/bigclawctl`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1596(RepositoryHasNoPythonFiles|AssignedPythonAssetsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`", "Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1596 hardens that baseline rather than lowering the numeric file count further."},
	},
	{
		ticket:        "1598",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/product/dashboard_run_contract.go", "bigclaw-go/internal/product/dashboard_run_contract_test.go", "bigclaw-go/internal/queue/memory_queue.go", "bigclaw-go/internal/triage/repo.go", "bigclaw-go/internal/api/server.go", "bigclaw-go/internal/bootstrap/bootstrap.go", "bigclaw-go/internal/planning/planning.go", "bigclaw-go/internal/workflow/definition.go", "bigclaw-go/internal/api/live_shadow_surface.go"},
		absentPaths:   []string{"src/bigclaw/dashboard_run_contract.py", "src/bigclaw/memory.py", "src/bigclaw/repo_commits.py", "src/bigclaw/run_detail.py", "src/bigclaw/workspace_bootstrap_validation.py", "tests/test_dsl.py", "tests/test_live_shadow_scorecard.py", "tests/test_planning.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-1598-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1598", "Repository-wide Python file count: `0`.", "`src/bigclaw/dashboard_run_contract.py`", "`src/bigclaw/memory.py`", "`src/bigclaw/repo_commits.py`", "`src/bigclaw/run_detail.py`", "`src/bigclaw/workspace_bootstrap_validation.py`", "`tests/test_dsl.py`", "`tests/test_live_shadow_scorecard.py`", "`tests/test_planning.py`", "`bigclaw-go/internal/product/dashboard_run_contract.go`", "`bigclaw-go/internal/queue/memory_queue.go`", "`bigclaw-go/internal/triage/repo.go`", "`bigclaw-go/internal/api/server.go`", "`bigclaw-go/internal/bootstrap/bootstrap.go`", "`bigclaw-go/internal/planning/planning.go`", "`bigclaw-go/internal/workflow/definition.go`", "`bigclaw-go/internal/api/live_shadow_surface.go`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1598"},
	},
	{
		ticket:        "1600",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"bigclaw-go/internal/workflow/definition.go", "bigclaw-go/internal/workflow/definition_test.go", "bigclaw-go/internal/observability/audit.go", "bigclaw-go/internal/observability/recorder.go", "bigclaw-go/internal/repo/governance.go", "bigclaw-go/internal/product/saved_views.go", "bigclaw-go/internal/observability/audit_test.go", "bigclaw-go/internal/events/bus_test.go", "bigclaw-go/internal/policy/memory_test.go", "bigclaw-go/internal/repo/repo_surfaces_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go"},
		absentPaths:   []string{"src/bigclaw/dsl.py", "src/bigclaw/observability.py", "src/bigclaw/repo_governance.py", "src/bigclaw/saved_views.py", "tests/test_audit_events.py", "tests/test_event_bus.py", "tests/test_memory.py", "tests/test_repo_board.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-1600-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-1600", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Explicit assigned Python asset list:", "`src/bigclaw/dsl.py`", "`src/bigclaw/observability.py`", "`src/bigclaw/repo_governance.py`", "`src/bigclaw/saved_views.py`", "`tests/test_audit_events.py`", "`tests/test_event_bus.py`", "`tests/test_memory.py`", "`tests/test_repo_board.py`", "`bigclaw-go/internal/workflow/definition.go`", "`bigclaw-go/internal/observability/audit.go`", "`bigclaw-go/internal/observability/recorder.go`", "`bigclaw-go/internal/repo/governance.go`", "`bigclaw-go/internal/product/saved_views.go`", "`bigclaw-go/internal/observability/audit_test.go`", "`bigclaw-go/internal/events/bus_test.go`", "`bigclaw-go/internal/policy/memory_test.go`", "`bigclaw-go/internal/repo/repo_surfaces_test.go`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1600"},
	},
	{
		ticket:        "161",
		repoScan:      true,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/events/transition_bus.go", "bigclaw-go/internal/events/transition_bus_test.go", "bigclaw-go/internal/regression/top_level_module_purge_tranche13_test.go"},
		absentPaths:   []string{"src/bigclaw/event_bus.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-161-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-161", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`src/bigclaw/event_bus.py`", "`bigclaw-go/internal/events/transition_bus.go`", "`bigclaw-go/internal/events/transition_bus_test.go`", "`bigclaw-go/internal/regression/top_level_module_purge_tranche13_test.go`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw bigclaw-go/internal/events -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO161(RepositoryHasNoPythonFiles|SrcBigclawStaysPythonFree|RemovedEventBusModuleStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche13$'`"},
	},
	{
		ticket:        "162",
		repoScan:      false,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/control/controller_test.go", "bigclaw-go/internal/product/dashboard_run_contract_test.go", "bigclaw-go/internal/uireview/uireview_test.go", "bigclaw-go/internal/designsystem/designsystem_test.go", "bigclaw-go/internal/workflow/definition_test.go", "bigclaw-go/internal/evaluation/evaluation_test.go", "bigclaw-go/internal/refill/queue_repo_fixture_test.go", "bigclaw-go/internal/costcontrol/controller_test.go", "bigclaw-go/internal/issuearchive/archive_test.go", "bigclaw-go/internal/pilot/report_test.go", "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "bigclaw-go/docs/reports/shared-queue-companion-summary.json"},
		absentPaths:   []string{"tests/test_control_center.py", "tests/test_operations.py", "tests/test_ui_review.py", "tests/test_design_system.py", "tests/test_dsl.py", "tests/test_evaluation.py", "tests/test_parallel_validation_bundle.py", "tests/test_followup_digests.py", "tests/test_live_shadow_scorecard.py", "tests/test_parallel_refill.py", "tests/test_reports.py"},
		reportPath:    "docs/reports/big-go-162-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-162", "Repository-wide Python file count: `0`.", "`tests`: absent", "`tests/test_control_center.py`", "`tests/test_parallel_validation_bundle.py`", "`tests/test_followup_digests.py`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/control/controller_test.go`", "`bigclaw-go/internal/designsystem/designsystem_test.go`", "`bigclaw-go/internal/refill/queue_repo_fixture_test.go`", "`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`", "`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find tests bigclaw-go/internal bigclaw-go/docs/reports -type f \\\\( -name '*.py' -o -name 'validation-bundle-continuation-scorecard.json' -o -name 'shared-queue-companion-summary.json' -o -name 'cross-process-coordination-capability-surface.json' \\\\) 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO162(ResidualPythonTestTreeStaysAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "167",
		repoScan:      true,
		pythonDirs:    []string{"bigclaw-go/internal/regression", "bigclaw-go/internal/migration", "bigclaw-go/docs/reports", "reports"},
		existingPaths: []string{"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/internal/regression/root_script_residual_sweep_test.go", "bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go", "bigclaw-go/internal/migration/legacy_model_runtime_modules.go", "bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go", "bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go", "bigclaw-go/docs/reports/review-readiness.md", "bigclaw-go/docs/reports/big-go-152-python-asset-sweep.md", "bigclaw-go/docs/reports/big-go-157-python-asset-sweep.md", "bigclaw-go/docs/reports/big-go-162-python-asset-sweep.md", "reports/BIG-GO-152-validation.md", "reports/BIG-GO-157-validation.md", "reports/BIG-GO-162-validation.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-167-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-167", "Repository-wide Python file count: `0`.", "`bigclaw-go/internal/regression`: `0` Python files", "`bigclaw-go/internal/migration`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`reports`: `0` Python files", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/root_script_residual_sweep_test.go`", "`bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`", "`bigclaw-go/internal/migration/legacy_model_runtime_modules.go`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`", "`bigclaw-go/docs/reports/review-readiness.md`", "`bigclaw-go/docs/reports/big-go-152-python-asset-sweep.md`", "`bigclaw-go/docs/reports/big-go-157-python-asset-sweep.md`", "`bigclaw-go/docs/reports/big-go-162-python-asset-sweep.md`", "`reports/BIG-GO-152-validation.md`", "`reports/BIG-GO-157-validation.md`", "`reports/BIG-GO-162-validation.md`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find bigclaw-go/internal/regression bigclaw-go/internal/migration bigclaw-go/docs/reports reports -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO167(RepositoryHasNoPythonFiles|ReferenceDenseGoOwnedDirectoriesStayPythonFree|NativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "168",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts", "docs", "docs/reports", "reports", "scripts", "bigclaw-go/scripts", "bigclaw-go/docs/reports", "bigclaw-go/examples"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh", "bigclaw-go/scripts/benchmark/run_suite.sh", "bigclaw-go/docs/reports/review-readiness.md", "docs/reports"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-168-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-168", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`docs`: `0` Python files", "`docs/reports`: `0` Python files", "`reports`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`bigclaw-go/examples`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`bigclaw-go/scripts/benchmark/run_suite.sh`", "`bigclaw-go/docs/reports/review-readiness.md`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find docs docs/reports reports scripts bigclaw-go/scripts bigclaw-go/docs/reports bigclaw-go/examples -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO168"},
	},
	{
		ticket:        "170",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-170-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-170", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO170"},
	},
	{
		ticket:        "171",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-171-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-171", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO171(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "172",
		repoScan:      true,
		pythonDirs:    []string{"bigclaw-go/internal/api", "bigclaw-go/internal/contract", "bigclaw-go/internal/events", "bigclaw-go/internal/githubsync", "bigclaw-go/internal/governance", "bigclaw-go/internal/observability", "bigclaw-go/internal/orchestrator", "bigclaw-go/internal/planning", "bigclaw-go/internal/policy", "bigclaw-go/internal/product", "bigclaw-go/internal/queue", "bigclaw-go/internal/repo", "bigclaw-go/internal/workflow"},
		existingPaths: []string{"bigclaw-go/internal/api/coordination_surface.go", "bigclaw-go/internal/contract/execution_test.go", "bigclaw-go/internal/events/bus_test.go", "bigclaw-go/internal/githubsync/sync_test.go", "bigclaw-go/internal/governance/freeze_test.go", "bigclaw-go/internal/observability/recorder_test.go", "bigclaw-go/internal/orchestrator/loop_test.go", "bigclaw-go/internal/planning/planning_test.go", "bigclaw-go/internal/policy/memory_test.go", "bigclaw-go/internal/product/clawhost_rollout_test.go", "bigclaw-go/internal/queue/sqlite_queue_test.go", "bigclaw-go/internal/repo/gateway.go", "bigclaw-go/internal/repo/governance.go", "bigclaw-go/internal/repo/links.go", "bigclaw-go/internal/repo/registry.go", "bigclaw-go/internal/workflow/model_test.go", "bigclaw-go/internal/workflow/orchestration_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "reports/BIG-GO-948-validation.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-172-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-172", "Repository-wide Python file count: `0`.", "`bigclaw-go/internal/api`: `0` Python files", "`bigclaw-go/internal/contract`: `0` Python files", "`bigclaw-go/internal/events`: `0` Python files", "`bigclaw-go/internal/githubsync`: `0` Python files", "`bigclaw-go/internal/governance`: `0` Python files", "`bigclaw-go/internal/observability`: `0` Python files", "`bigclaw-go/internal/orchestrator`: `0` Python files", "`bigclaw-go/internal/planning`: `0` Python files", "`bigclaw-go/internal/policy`: `0` Python files", "`bigclaw-go/internal/product`: `0` Python files", "`bigclaw-go/internal/queue`: `0` Python files", "`bigclaw-go/internal/repo`: `0` Python files", "`bigclaw-go/internal/workflow`: `0` Python files", "Explicit remaining Python asset list: none.", "`tests/test_cross_process_coordination_surface.py`", "`tests/test_event_bus.py`", "`tests/test_execution_contract.py`", "`tests/test_execution_flow.py`", "`tests/test_github_sync.py`", "`tests/test_governance.py`", "`tests/test_memory.py`", "`tests/test_models.py`", "`tests/test_observability.py`", "`tests/test_orchestration.py`", "`tests/test_planning.py`", "`tests/test_queue.py`", "`tests/test_repo_gateway.py`", "`tests/test_repo_governance.py`", "`tests/test_repo_links.py`", "`tests/test_repo_registry.py`", "`tests/test_repo_rollout.py`", "`bigclaw-go/internal/api/coordination_surface.go`", "`bigclaw-go/internal/events/bus_test.go`", "`bigclaw-go/internal/contract/execution_test.go`", "`bigclaw-go/internal/workflow/orchestration_test.go`", "`bigclaw-go/internal/githubsync/sync_test.go`", "`bigclaw-go/internal/governance/freeze_test.go`", "`bigclaw-go/internal/policy/memory_test.go`", "`bigclaw-go/internal/observability/recorder_test.go`", "`bigclaw-go/internal/planning/planning_test.go`", "`bigclaw-go/internal/queue/sqlite_queue_test.go`", "`bigclaw-go/internal/repo/gateway.go`", "`bigclaw-go/internal/repo/governance.go`", "`bigclaw-go/internal/repo/links.go`", "`bigclaw-go/internal/repo/registry.go`", "`bigclaw-go/internal/product/clawhost_rollout_test.go`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find bigclaw-go/internal/api bigclaw-go/internal/contract bigclaw-go/internal/events bigclaw-go/internal/githubsync bigclaw-go/internal/governance bigclaw-go/internal/observability bigclaw-go/internal/orchestrator bigclaw-go/internal/planning bigclaw-go/internal/policy bigclaw-go/internal/product bigclaw-go/internal/queue bigclaw-go/internal/repo bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO172(RepositoryHasNoPythonFiles|RemainingTestHeavyReplacementDirectoriesStayPythonFree|RepresentativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "177",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-177-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-177", "Broad repo Python reduction sweep Y", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO177"},
	},
	{
		ticket:        "178",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-178-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-178", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO178"},
	},
	{
		ticket:        "181",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "bigclaw-go/internal/governance", "bigclaw-go/internal/domain", "bigclaw-go/internal/observability", "bigclaw-go/internal/product", "bigclaw-go/internal/workflow"},
		existingPaths: []string{"bigclaw-go/internal/governance/freeze.go", "bigclaw-go/internal/domain/task.go", "bigclaw-go/internal/observability/recorder.go", "bigclaw-go/internal/product/dashboard_run_contract.go", "bigclaw-go/internal/workflow/orchestration.go", "bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go"},
		absentPaths:   []string{"src/bigclaw/governance.py", "src/bigclaw/models.py", "src/bigclaw/observability.py", "src/bigclaw/operations.py", "src/bigclaw/orchestration.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-181-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-181", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`src/bigclaw/governance.py`", "`src/bigclaw/models.py`", "`src/bigclaw/observability.py`", "`src/bigclaw/operations.py`", "`src/bigclaw/orchestration.py`", "`bigclaw-go/internal/governance/freeze.go`", "`bigclaw-go/internal/domain/task.go`", "`bigclaw-go/internal/observability/recorder.go`", "`bigclaw-go/internal/product/dashboard_run_contract.go`", "`bigclaw-go/internal/workflow/orchestration.go`", "`bigclaw-go/internal/regression/top_level_module_purge_tranche15_test.go`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw bigclaw-go/internal/governance bigclaw-go/internal/domain bigclaw-go/internal/observability bigclaw-go/internal/product bigclaw-go/internal/workflow -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO181(RepositoryHasNoPythonFiles|SrcBigclawTranche15StaysPythonFree|RetiredTranche15PythonPathsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche15$'`"},
	},
	{
		ticket:        "183",
		repoScan:      false,
		pythonDirs:    []string{},
		existingPaths: []string{"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/api/coordination_surface.go", "bigclaw-go/internal/contract/execution_test.go", "bigclaw-go/internal/workflow/orchestration_test.go", "bigclaw-go/internal/planning/planning_test.go", "bigclaw-go/internal/queue/sqlite_queue_test.go", "bigclaw-go/internal/repo/repo_surfaces_test.go", "bigclaw-go/internal/collaboration/thread_test.go", "bigclaw-go/internal/repo/gateway.go", "bigclaw-go/internal/repo/governance.go", "bigclaw-go/internal/repo/links.go", "bigclaw-go/internal/repo/registry.go", "bigclaw-go/internal/product/clawhost_rollout_test.go", "bigclaw-go/internal/triage/repo_test.go", "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json", "bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json", "bigclaw-go/docs/reports/shared-queue-companion-summary.json", "bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json", "bigclaw-go/docs/reports/shadow-matrix-report.json"},
		absentPaths:   []string{"tests/conftest.py", "tests/test_cross_process_coordination_surface.py", "tests/test_execution_contract.py", "tests/test_followup_digests.py", "tests/test_live_shadow_bundle.py", "tests/test_live_shadow_scorecard.py", "tests/test_orchestration.py", "tests/test_parallel_refill.py", "tests/test_parallel_validation_bundle.py", "tests/test_planning.py", "tests/test_queue.py", "tests/test_repo_board.py", "tests/test_repo_collaboration.py", "tests/test_repo_gateway.py", "tests/test_repo_governance.py", "tests/test_repo_links.py", "tests/test_repo_registry.py", "tests/test_repo_rollout.py", "tests/test_repo_triage.py"},
		reportPath:    "bigclaw-go/docs/reports/big-go-183-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-183", "Repository-wide Python file count: `0`.", "`tests`: absent", "`bigclaw-go/internal`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files and retained Go-owned report fixtures", "`tests/conftest.py`", "`tests/test_cross_process_coordination_surface.py`", "`tests/test_parallel_validation_bundle.py`", "`tests/test_repo_registry.py`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/api/coordination_surface.go`", "`bigclaw-go/internal/contract/execution_test.go`", "`bigclaw-go/internal/workflow/orchestration_test.go`", "`bigclaw-go/internal/planning/planning_test.go`", "`bigclaw-go/internal/queue/sqlite_queue_test.go`", "`bigclaw-go/internal/repo/repo_surfaces_test.go`", "`bigclaw-go/internal/collaboration/thread_test.go`", "`bigclaw-go/internal/product/clawhost_rollout_test.go`", "`bigclaw-go/internal/triage/repo_test.go`", "`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`", "`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`", "`bigclaw-go/docs/reports/shared-queue-companion-summary.json`", "`bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`", "`bigclaw-go/docs/reports/shadow-matrix-report.json`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find tests bigclaw-go/internal bigclaw-go/docs/reports -type f \\\\( -name '*.py' -o -name 'validation-bundle-continuation-scorecard.json' -o -name 'shared-queue-companion-summary.json' -o -name 'cross-process-coordination-capability-surface.json' -o -name 'live-shadow-mirror-scorecard.json' -o -name 'shadow-matrix-report.json' \\\\) 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO183(ResidualPythonTestTreeStaysAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "186",
		repoScan:      true,
		pythonDirs:    []string{"bigclaw-go/examples", "bigclaw-go/docs/reports/broker-failover-stub-artifacts", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts", "scripts/ops"},
		existingPaths: []string{"bigclaw-go/examples/shadow-task.json", "bigclaw-go/examples/shadow-corpus-manifest.json", "bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json", "bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-a-audit.jsonl", "scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-186-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-186", "Repository-wide Python file count: `0`.", "`bigclaw-go/examples`: `0` Python files", "`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python files", "`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python files", "`scripts/ops`: `0` Python files", "Explicit remaining Python asset list: none.", "`bigclaw-go/examples/shadow-task.json`", "`bigclaw-go/examples/shadow-corpus-manifest.json`", "`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-01/backend-health.json`", "`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/replay-capture.json`", "`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/contention-then-takeover-live/node-a-audit.jsonl`", "`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl`", "`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/lease-expiry-stale-writer-rejected-live/node-a-audit.jsonl`", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`", "`find bigclaw-go/examples bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts scripts/ops -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO186(RepositoryHasNoPythonFiles|SupportAssetDirectoriesStayPythonFree|RetainedSupportAssetsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "191",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-191-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-191", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO191(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "192",
		repoScan:      true,
		pythonDirs:    []string{"tests", "bigclaw-go/scripts", "bigclaw-go/internal/migration", "bigclaw-go/internal/regression", "bigclaw-go/docs/reports"},
		existingPaths: []string{"reports/BIG-GO-948-validation.md", "bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go", "bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go", "bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go", "bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md", "bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md", "bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-192-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-192", "Repository-wide Python file count: `0`.", "`tests`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`bigclaw-go/internal/migration`: `0` Python files", "`bigclaw-go/internal/regression`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`reports/BIG-GO-948-validation.md`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go`", "`bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`", "`bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md`", "`bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md`", "`bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find tests bigclaw-go/scripts bigclaw-go/internal/migration bigclaw-go/internal/regression bigclaw-go/docs/reports -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO192(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "198",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-198-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-198", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO198"},
	},
	{
		ticket:        "19",
		repoScan:      true,
		pythonDirs:    []string{"src/bigclaw", "tests", "scripts", "bigclaw-go/scripts"},
		existingPaths: []string{"scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "scripts/dev_bootstrap.sh", "bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go", "bigclaw-go/scripts/e2e/run_all.sh"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-19-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-19", "Repository-wide Python file count: `0`.", "`src/bigclaw`: `0` Python files", "`tests`: `0` Python files", "`scripts`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "Explicit remaining Python asset list: none.", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`scripts/dev_bootstrap.sh`", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`bigclaw-go/scripts/e2e/run_all.sh`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO19"},
	},
	{
		ticket:        "200",
		repoScan:      true,
		pythonDirs:    []string{"bigclaw-go/cmd", "scripts/ops", "bigclaw-go/docs/reports"},
		existingPaths: []string{"bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawctl/automation_commands.go", "bigclaw-go/cmd/bigclawctl/migration_commands.go", "bigclaw-go/cmd/bigclawd/main.go", "scripts/ops/bigclawctl", "scripts/ops/bigclaw-issue", "scripts/ops/bigclaw-panel", "scripts/ops/bigclaw-symphony", "bigclaw-go/docs/reports/issue-coverage.md", "bigclaw-go/docs/reports/parallel-follow-up-index.md", "bigclaw-go/docs/reports/parallel-validation-matrix.md", "bigclaw-go/docs/reports/review-readiness.md", "bigclaw-go/docs/reports/linear-project-sync-summary.md", "bigclaw-go/docs/reports/epic-closure-readiness-report.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-200-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-200", "Repository-wide Python file count: `0`.", "`bigclaw-go/cmd`: `0` Python files", "`scripts/ops`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`bigclaw-go/cmd/bigclawctl/main.go`", "`bigclaw-go/cmd/bigclawctl/automation_commands.go`", "`bigclaw-go/cmd/bigclawctl/migration_commands.go`", "`bigclaw-go/cmd/bigclawd/main.go`", "`scripts/ops/bigclawctl`", "`scripts/ops/bigclaw-issue`", "`scripts/ops/bigclaw-panel`", "`scripts/ops/bigclaw-symphony`", "`bigclaw-go/docs/reports/issue-coverage.md`", "`bigclaw-go/docs/reports/parallel-follow-up-index.md`", "`bigclaw-go/docs/reports/parallel-validation-matrix.md`", "`bigclaw-go/docs/reports/review-readiness.md`", "`bigclaw-go/docs/reports/linear-project-sync-summary.md`", "`bigclaw-go/docs/reports/epic-closure-readiness-report.md`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find bigclaw-go/cmd scripts/ops bigclaw-go/docs/reports -maxdepth 2 -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO200(RepositoryHasNoPythonFiles|CommandAndReportIndexSurfacesStayPythonFree|GoNativeEntryPointsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "202",
		repoScan:      true,
		pythonDirs:    []string{"tests", "bigclaw-go/scripts", "bigclaw-go/internal/migration", "bigclaw-go/internal/regression", "bigclaw-go/docs/reports"},
		existingPaths: []string{"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go", "bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go", "bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go", "bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md", "bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md", "bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md", "reports/BIG-GO-948-validation.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-202-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-202", "Repository-wide Python file count: `0`.", "`tests`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`bigclaw-go/internal/migration`: `0` Python files", "`bigclaw-go/internal/regression`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`reports/BIG-GO-948-validation.md`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go`", "`bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`", "`bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md`", "`bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md`", "`bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find tests bigclaw-go/scripts bigclaw-go/internal/migration bigclaw-go/internal/regression bigclaw-go/docs/reports -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO202(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
	{
		ticket:        "208",
		repoScan:      true,
		pythonDirs:    []string{"reports", "bigclaw-go/docs/reports", "bigclaw-go/internal/regression", "bigclaw-go/internal/migration", "bigclaw-go/scripts"},
		existingPaths: []string{"reports/BIG-GO-948-validation.md", "bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go", "bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go", "bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go", "bigclaw-go/internal/regression/python_test_tranche17_removal_test.go", "bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md", "bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md", "bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md", "bigclaw-go/docs/reports/big-go-192-python-asset-sweep.md"},
		absentPaths:   []string{},
		reportPath:    "bigclaw-go/docs/reports/big-go-208-python-asset-sweep.md",
		reportNeedles: []string{"BIG-GO-208", "Repository-wide Python file count: `0`.", "`reports`: `0` Python files", "`bigclaw-go/docs/reports`: `0` Python files", "`bigclaw-go/internal/regression`: `0` Python files", "`bigclaw-go/internal/migration`: `0` Python files", "`bigclaw-go/scripts`: `0` Python files", "`reports/BIG-GO-948-validation.md`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go`", "`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`", "`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`", "`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`", "`bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`", "`bigclaw-go/internal/regression/big_go_192_zero_python_guard_test.go`", "`bigclaw-go/docs/reports/big-go-13-legacy-test-contract-sweep-d.md`", "`bigclaw-go/docs/reports/big-go-1365-legacy-test-contract-sweep-b.md`", "`bigclaw-go/docs/reports/big-go-1577-python-asset-sweep.md`", "`bigclaw-go/docs/reports/big-go-192-python-asset-sweep.md`", "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`", "`find reports bigclaw-go/docs/reports bigclaw-go/internal/regression bigclaw-go/internal/migration bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`", "`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO208(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`"},
	},
}

func TestZeroPythonGuardCatalog(t *testing.T) {
	if len(zeroPythonGuardCatalog) != 186 {
		t.Fatalf("expected 186 consolidated zero-Python guard cases, found %d", len(zeroPythonGuardCatalog))
	}

	seen := make(map[string]struct{}, len(zeroPythonGuardCatalog))
	for _, guardCase := range zeroPythonGuardCatalog {
		if guardCase.ticket == "" {
			t.Fatal("catalog contains an empty ticket id")
		}
		if _, ok := seen[guardCase.ticket]; ok {
			t.Fatalf("catalog contains duplicate ticket %s", guardCase.ticket)
		}
		seen[guardCase.ticket] = struct{}{}
	}
}

func TestZeroPythonGuardRepositoryAndAuditedDirectories(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	for _, guardCase := range zeroPythonGuardCatalog {
		guardCase := guardCase
		t.Run("BIGGO"+guardCase.ticket, func(t *testing.T) {
			if guardCase.repoScan {
				pythonFiles := collectPythonFiles(t, rootRepo)
				if len(pythonFiles) != 0 {
					t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
				}
			}

			for _, relativeDir := range guardCase.pythonDirs {
				pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
				if len(pythonFiles) != 0 {
					t.Fatalf("expected audited directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
				}
			}
		})
	}
}

func TestZeroPythonGuardReplacementAndDeletedPaths(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	for _, guardCase := range zeroPythonGuardCatalog {
		guardCase := guardCase
		t.Run("BIGGO"+guardCase.ticket, func(t *testing.T) {
			for _, relativePath := range guardCase.absentPaths {
				if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
					t.Fatalf("expected retired Python path to remain absent: %s", relativePath)
				}
			}
			for _, relativePath := range guardCase.existingPaths {
				if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
					t.Fatalf("expected replacement path to exist: %s (%v)", relativePath, err)
				}
			}
		})
	}
}

func TestZeroPythonGuardLaneReports(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	for _, guardCase := range zeroPythonGuardCatalog {
		guardCase := guardCase
		if guardCase.reportPath == "" {
			continue
		}
		t.Run("BIGGO"+guardCase.ticket, func(t *testing.T) {
			report := readRepoFile(t, rootRepo, resolveZeroPythonGuardReportPath(guardCase.reportPath))
			for _, needle := range guardCase.reportNeedles {
				if !strings.Contains(report, normalizeZeroPythonGuardNeedle(needle)) {
					t.Fatalf("lane report %s missing substring %q", guardCase.reportPath, needle)
				}
			}
		})
	}
}

func resolveZeroPythonGuardReportPath(reportPath string) string {
	if strings.HasPrefix(reportPath, "docs/reports/") {
		return filepath.ToSlash(filepath.Join("bigclaw-go", reportPath))
	}
	return reportPath
}

func normalizeZeroPythonGuardNeedle(needle string) string {
	return strings.ReplaceAll(needle, "\\\\", "\\")
}

func collectPythonFiles(t *testing.T, root string) []string {
	t.Helper()

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".py" {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}

func TestBIGGO227ConsolidatedZeroPythonGuardFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	regressionDir := filepath.Join(rootRepo, "bigclaw-go", "internal", "regression")

	files, err := filepath.Glob(filepath.Join(regressionDir, "big_go_*_zero_python_guard_test.go"))
	if err != nil {
		t.Fatalf("glob consolidated zero-Python guard files: %v", err)
	}

	allowed := map[string]struct{}{
		"big_go_1235_zero_python_guard_test.go": {},
		"big_go_124_zero_python_guard_test.go":  {},
		"big_go_154_zero_python_guard_test.go":  {},
		"big_go_176_zero_python_guard_test.go":  {},
		"big_go_205_zero_python_guard_test.go":  {},
	}
	if len(files) != len(allowed) {
		t.Fatalf("expected exactly %d specialized zero-Python guard files to remain, found %d: %v", len(allowed), len(files), files)
	}
	for _, file := range files {
		name := filepath.Base(file)
		if _, ok := allowed[name]; !ok {
			t.Fatalf("unexpected zero-Python guard file remained after consolidation: %s", name)
		}
	}

	if _, err := os.Stat(filepath.Join(regressionDir, "big_go_227_zero_python_guard_catalog_test.go")); err != nil {
		t.Fatalf("expected consolidated catalog test to exist: %v", err)
	}

	consolidatedReport := readRepoFile(t, rootRepo, "docs/reports/big-go-227-zero-python-guard-consolidation.md")
	for _, needle := range []string{
		"BIG-GO-227",
		"186",
		"191",
		"big_go_227_zero_python_guard_catalog_test.go",
		"big_go_1235_zero_python_guard_test.go",
		"big_go_205_zero_python_guard_test.go",
		"go test -count=1 ./internal/regression -run 'TestZeroPythonGuardCatalog|TestZeroPythonGuardRepositoryAndAuditedDirectories|TestZeroPythonGuardReplacementAndDeletedPaths|TestZeroPythonGuardLaneReports|TestBIGGO227ConsolidatedZeroPythonGuardFiles|TestBIGGO1235ReadmeStaysGoOnly|TestBIGGO124TargetResidualPythonPathsAbsent|TestBIGGO154|TestBIGGO176|TestBIGGO205'",
	} {
		if !strings.Contains(consolidatedReport, needle) {
			t.Fatalf("BIG-GO-227 consolidation report missing substring %q", needle)
		}
	}

	statusSummary := readRepoFile(t, rootRepo, "reports/BIG-GO-227-status.json")
	for _, needle := range []string{"\"identifier\": \"BIG-GO-227\"", "\"consolidated_case_count\": 186", "\"retained_specialized_files\": 5"} {
		if !strings.Contains(statusSummary, needle) {
			t.Fatalf("BIG-GO-227 status summary missing substring %q", needle)
		}
	}

	validation := readRepoFile(t, rootRepo, "reports/BIG-GO-227-validation.md")
	for _, needle := range []string{"BIG-GO-227 validation", "PASS", "TestZeroPythonGuardCatalog", "TestBIGGO227ConsolidatedZeroPythonGuardFiles"} {
		if !strings.Contains(validation, needle) {
			t.Fatalf("BIG-GO-227 validation report missing substring %q", needle)
		}
	}
}

func Example_zeroPythonGuardCatalogSummary() {
	fmt.Println(len(zeroPythonGuardCatalog))
	fmt.Println(zeroPythonGuardCatalog[0].ticket)
	fmt.Println(zeroPythonGuardCatalog[len(zeroPythonGuardCatalog)-1].ticket)
	// Output:
	// 186
	// 100
	// 208
}
