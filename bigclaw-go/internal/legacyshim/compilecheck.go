package legacyshim

import (
	"os/exec"
)

type CompileCheckResult struct {
	Python string   `json:"python"`
	Files  []string `json:"files"`
	Output string   `json:"output,omitempty"`
}

type runner func(name string, args ...string) ([]byte, error)

func FrozenCompileCheckFiles(repoRoot string) []string {
	return []string{}
}

func CompileCheck(repoRoot string, pythonBin string) (CompileCheckResult, error) {
	return compileCheck(repoRoot, pythonBin, func(name string, args ...string) ([]byte, error) {
		return exec.Command(name, args...).CombinedOutput()
	})
}

func compileCheck(repoRoot string, pythonBin string, run runner) (CompileCheckResult, error) {
	files := FrozenCompileCheckFiles(repoRoot)
	if len(files) == 0 {
		return CompileCheckResult{
			Python: pythonBin,
			Files:  files,
		}, nil
	}
	args := make([]string, 0, len(files)+2)
	args = append(args, "-m", "py_compile")
	args = append(args, files...)
	output, err := run(pythonBin, args...)
	return CompileCheckResult{
		Python: pythonBin,
		Files:  files,
		Output: string(output),
	}, err
}
