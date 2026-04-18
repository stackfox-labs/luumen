package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luumen/internal/workspace"
)

func TestRunnerHealthyRepo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, workspace.LuumenConfigFile), "return {\n    project = {\n        name = \"game\",\n    },\n}\n")
	writeFile(t, filepath.Join(root, workspace.RokitConfigFile), "[tools]\nrojo = \"rojo-rbx/rojo\"\n")
	writeFile(t, filepath.Join(root, workspace.WallyConfigFile), "[dependencies]\n")
	writeFile(t, filepath.Join(root, "default.project.json"), "{\"tree\":{}}\n")
	mustMkdir(t, filepath.Join(root, "Packages"))

	runner := NewRunner(func(_ string) (string, error) { return "ok", nil })
	report := runner.Run(workspace.Workspace{
		RootPath:         root,
		HasLuumenConfig:  true,
		LuumenConfigPath: filepath.Join(root, workspace.LuumenConfigFile),
		HasRokitConfig:   true,
		RokitConfigPath:  filepath.Join(root, workspace.RokitConfigFile),
		HasWallyConfig:   true,
		WallyConfigPath:  filepath.Join(root, workspace.WallyConfigFile),
		HasRojoProject:   true,
		RojoProjectPaths: []string{filepath.Join(root, "default.project.json")},
	})

	if report.Errors != 0 || report.Warnings != 0 {
		t.Fatalf("expected healthy report, got errors=%d warnings=%d results=%#v", report.Errors, report.Warnings, report.Results)
	}
	if report.Passes == 0 {
		t.Fatal("expected pass results for healthy report")
	}
}

func TestRunnerMalformedConfigsAndRojo(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, workspace.LuumenConfigFile), "return {\n    tasks = {\n        bad = 1,\n    },\n}\n")
	writeFile(t, filepath.Join(root, workspace.RokitConfigFile), "[tools\n")
	writeFile(t, filepath.Join(root, workspace.WallyConfigFile), "[dependencies\n")
	writeFile(t, filepath.Join(root, "default.project.json"), "{bad json")

	runner := NewRunner(func(_ string) (string, error) { return "ok", nil })
	report := runner.Run(workspace.Workspace{
		RootPath:         root,
		HasLuumenConfig:  true,
		LuumenConfigPath: filepath.Join(root, workspace.LuumenConfigFile),
		HasRokitConfig:   true,
		RokitConfigPath:  filepath.Join(root, workspace.RokitConfigFile),
		HasWallyConfig:   true,
		WallyConfigPath:  filepath.Join(root, workspace.WallyConfigFile),
		HasRojoProject:   true,
		RojoProjectPaths: []string{filepath.Join(root, "default.project.json")},
	})

	if report.Errors < 4 {
		t.Fatalf("expected multiple config errors, got %d (%#v)", report.Errors, report.Results)
	}
}

func TestRunnerMissingToolsAndPackages(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, workspace.WallyConfigFile), "[dependencies]\n")

	runner := NewRunner(func(binary string) (string, error) {
		return "", &binaryNotFoundError{binary: binary}
	})
	report := runner.Run(workspace.Workspace{
		RootPath:        root,
		HasWallyConfig:  true,
		WallyConfigPath: filepath.Join(root, workspace.WallyConfigFile),
	})

	if report.Errors == 0 {
		t.Fatalf("expected binary missing error, got %#v", report.Results)
	}
	if report.Warnings == 0 {
		t.Fatalf("expected missing Packages warning, got %#v", report.Results)
	}
}

func TestRunnerMissingRojoProjectWarning(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeFile(t, filepath.Join(root, workspace.LuumenConfigFile), "return {\n    tasks = {\n        dev = \"rojo serve default.project.json\",\n    },\n}\n")

	runner := NewRunner(func(_ string) (string, error) { return "ok", nil })
	report := runner.Run(workspace.Workspace{
		RootPath:         root,
		HasLuumenConfig:  true,
		LuumenConfigPath: filepath.Join(root, workspace.LuumenConfigFile),
		HasRojoProject:   false,
	})

	if report.Warnings == 0 {
		t.Fatalf("expected warning for missing rojo project, got %#v", report.Results)
	}
	found := false
	for _, result := range report.Results {
		if result.ID == "rojo-config" && strings.Contains(strings.ToLower(result.Message), "rojo project file not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected rojo-config warning, got %#v", report.Results)
	}
}

func TestRunnerNoRojoWarningForNonRojoProject(t *testing.T) {
	t.Parallel()

	runner := NewRunner(func(_ string) (string, error) { return "ok", nil })
	report := runner.Run(workspace.Workspace{RootPath: t.TempDir(), HasRojoProject: false})

	for _, result := range report.Results {
		if result.ID == "rojo-config" {
			t.Fatalf("expected no rojo-config result for non-rojo project, got %#v", result)
		}
	}
}

type binaryNotFoundError struct {
	binary string
}

func (e *binaryNotFoundError) Error() string {
	return e.binary + " not found"
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("failed to create directory %s: %v", path, err)
	}
}
