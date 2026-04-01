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

func BuildWorkspaceBootstrapArgs(repoRoot string, forwarded []string) []string {
	args := AppendMissingFlag(forwarded, "--repo-url", "git@github.com:OpenAGIs/BigClaw.git")
	args = AppendMissingFlag(args, "--cache-key", "openagis-bigclaw")
	return BuildBigclawctlExecArgs(repoRoot, []string{"workspace", "bootstrap"}, args)
}

func TranslateWorkspaceValidateArgs(forwarded []string) []string {
	translated := make([]string, 0, len(forwarded))
	for i := 0; i < len(forwarded); {
		arg := forwarded[i]
		switch {
		case arg == "--report-file" && i+1 < len(forwarded):
			translated = append(translated, "--report", forwarded[i+1])
			i += 2
		case strings.HasPrefix(arg, "--report-file="):
			translated = append(translated, "--report="+strings.SplitN(arg, "=", 2)[1])
			i++
		case arg == "--no-cleanup":
			translated = append(translated, "--cleanup=false")
			i++
		case arg == "--issues":
			issues := []string{}
			i++
			for i < len(forwarded) && !strings.HasPrefix(forwarded[i], "-") {
				issues = append(issues, forwarded[i])
				i++
			}
			translated = append(translated, "--issues", strings.Join(issues, ","))
		default:
			translated = append(translated, arg)
			i++
		}
	}
	return translated
}

func BuildWorkspaceValidateArgs(repoRoot string, forwarded []string) []string {
	return BuildBigclawctlExecArgs(repoRoot, []string{"workspace", "validate"}, TranslateWorkspaceValidateArgs(forwarded))
}

func BuildRefillArgs(repoRoot string, forwarded []string) []string {
	return BuildBigclawctlExecArgs(repoRoot, []string{"refill"}, forwarded)
}

func BuildWorkspaceRuntimeBootstrapArgs(repoRoot string, forwarded []string) []string {
	return BuildBigclawctlExecArgs(repoRoot, []string{"workspace"}, forwarded)
}
