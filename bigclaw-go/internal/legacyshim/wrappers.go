package legacyshim

import (
	"path/filepath"
)

const LegacyPythonWrapperNotice = "Legacy Python operator wrapper: use scripts/ops/bigclawctl for the Go mainline. This Python path remains only as a compatibility shim during migration."

func BuildBigclawctlExecArgs(repoRoot string, command []string, forwarded []string) []string {
	argv := []string{"bash", filepath.Join(repoRoot, "scripts", "ops", "bigclawctl")}
	argv = append(argv, command...)
	argv = append(argv, forwarded...)
	return argv
}

func BuildRefillArgs(repoRoot string, forwarded []string) []string {
	return BuildBigclawctlExecArgs(repoRoot, []string{"refill"}, forwarded)
}
