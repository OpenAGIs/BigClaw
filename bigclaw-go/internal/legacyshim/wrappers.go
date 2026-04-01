package legacyshim

import (
	"path/filepath"
	"strings"
)

const LegacyPythonWrapperNotice = "Legacy Python operator wrapper: use scripts/ops/bigclawctl for the Go mainline. This Python path remains only as a compatibility shim during migration."

func AppendMissingFlag(args []string, flag string, value string) []string {
	flagPrefix := flag + "="
	for _, arg := range args {
		if arg == flag || strings.HasPrefix(arg, flagPrefix) {
			return append([]string{}, args...)
		}
	}
	return append(append([]string{}, args...), flag, value)
}

func BuildBigclawctlExecArgs(repoRoot string, command []string, forwarded []string) []string {
	argv := []string{"bash", filepath.Join(repoRoot, "scripts", "ops", "bigclawctl")}
	argv = append(argv, command...)
	argv = append(argv, forwarded...)
	return argv
}

func RepoRootFromScript(scriptPath string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(scriptPath)))
}

func BuildRefillArgs(repoRoot string, forwarded []string) []string {
	return BuildBigclawctlExecArgs(repoRoot, []string{"refill"}, forwarded)
}
