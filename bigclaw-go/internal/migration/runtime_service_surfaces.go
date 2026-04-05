package migration

type LegacyRuntimeServiceSurfaceReplacement struct {
	RetiredPythonPath string
	SurfaceKind       string
	GoReplacements    []string
	EvidencePaths     []string
	Status            string
}

func LegacyRuntimeServiceSurfaceReplacements() []LegacyRuntimeServiceSurfaceReplacement {
	return []LegacyRuntimeServiceSurfaceReplacement{
		{
			RetiredPythonPath: "src/bigclaw/service.py",
			SurfaceKind:       "python-service-entrypoint",
			GoReplacements: []string{
				"bigclaw-go/internal/service/server.go",
				"bigclaw-go/internal/service/server_test.go",
			},
			EvidencePaths: []string{
				"reports/OPE-148-150-validation.md",
				"reports/OPE-151-153-validation.md",
				"reports/BIG-GO-948-validation.md",
			},
			Status: "retired Python service entry and monitoring endpoints are owned by the Go service package",
		},
		{
			RetiredPythonPath: "tests/test_service.py",
			SurfaceKind:       "python-service-regression-tests",
			GoReplacements: []string{
				"bigclaw-go/internal/service/server_test.go",
			},
			EvidencePaths: []string{
				"reports/OPE-148-150-validation.md",
				"reports/OPE-151-153-validation.md",
				"reports/BIG-GO-948-validation.md",
			},
			Status: "retired Python service tests are covered by Go-native service regression tests",
		},
	}
}
