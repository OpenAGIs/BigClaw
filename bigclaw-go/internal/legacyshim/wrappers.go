package legacyshim

import (
	"path/filepath"
	"strings"
)

func BuildBigclawctlExecArgs(repoRoot string, command []string, forwarded []string) []string {
	argv := []string{"bash", filepath.Join(repoRoot, "scripts", "ops", "bigclawctl")}
	argv = append(argv, command...)
	argv = append(argv, forwarded...)
	return argv
}

func RepoRootFromScript(scriptPath string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(scriptPath)))
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
