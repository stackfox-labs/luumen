package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"luumen/internal/config"
)

func TestScaffoldProjectFromTemplateAllowsNoFiles(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "minimal")
	template := createTemplate{
		Name:        "minimal",
		Description: "minimal",
		Project: createTemplateProject{
			Install: createTemplateInstall{Tools: false, Packages: false},
		},
	}

	if err := scaffoldProjectFromTemplate(target, "minimal", template, nil, config.Write); err != nil {
		t.Fatalf("expected minimal no-file scaffold to succeed, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, config.FileName)); err != nil {
		t.Fatalf("expected %s to exist, got: %v", config.FileName, err)
	}
}

func TestScaffoldProjectFromTemplateRunsScaffoldCommands(t *testing.T) {
	t.Parallel()

	target := filepath.Join(t.TempDir(), "scaffold")
	template := createTemplate{
		Name:             "minimal",
		Description:      "minimal",
		ScaffoldCommands: []string{"echo {{project_name}}", "echo {{package_name}}"},
		Project: createTemplateProject{
			Install: createTemplateInstall{Tools: false, Packages: false},
		},
	}

	calls := make([]string, 0, 2)
	runner := func(command string, workingDir string) error {
		calls = append(calls, workingDir+"::"+command)
		return nil
	}

	if err := scaffoldProjectFromTemplate(target, "My Game", template, runner, config.Write); err != nil {
		t.Fatalf("expected scaffold commands to run, got: %v", err)
	}

	expected := []string{
		target + "::echo My Game",
		target + "::echo my-game",
	}
	if !reflect.DeepEqual(calls, expected) {
		t.Fatalf("expected scaffold command calls %#v, got %#v", expected, calls)
	}
}
