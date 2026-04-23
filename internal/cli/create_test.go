package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luumen/internal/config"
)

func TestCreateFreshProject(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := filepath.Join(parent, "my-game")
	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeCreateCommand(createCommandDeps{
		getwd:          func() (string, error) { return parent, nil },
		writeConfig:    config.Write,
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "--template", "rojo-wally", "my-game")
	if err != nil {
		t.Fatalf("expected create success, got: %v", err)
	}

	requiredFiles := []string{
		filepath.Join(target, config.FileName),
		filepath.Join(target, "rokit.toml"),
		filepath.Join(target, "wally.toml"),
		filepath.Join(target, "default.project.json"),
		filepath.Join(target, "src", "server", "init.server.luau"),
		filepath.Join(target, "src", "shared", "init.luau"),
	}
	for _, required := range requiredFiles {
		if _, statErr := os.Stat(required); statErr != nil {
			t.Fatalf("expected scaffolded file %s to exist, got: %v", required, statErr)
		}
	}

	cfg, err := config.Load(filepath.Join(target, config.FileName))
	if err != nil {
		t.Fatalf("expected %s to load, got: %v", config.FileName, err)
	}
	if cfg.Project.Name != "my-game" {
		t.Fatalf("expected project name my-game, got %q", cfg.Project.Name)
	}
	if !cfg.Luu.Install.Tools || !cfg.Luu.Install.Packages {
		t.Fatalf("expected install defaults enabled, got %+v", cfg.Luu.Install)
	}

	rokitContents, readErr := os.ReadFile(filepath.Join(target, "rokit.toml"))
	if readErr != nil {
		t.Fatalf("failed to read scaffolded rokit.toml: %v", readErr)
	}
	rokitText := string(rokitContents)
	if !strings.Contains(rokitText, "rojo-rbx/rojo@") || !strings.Contains(rokitText, "UpliftGames/wally@") {
		t.Fatalf("expected versioned rokit tool specs, got: %s", rokitText)
	}

	if rokit.calls != 1 || wally.calls != 1 {
		t.Fatalf("expected both installers to run once, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
	if rokit.lastOption.WorkingDir != target || wally.lastOption.WorkingDir != target {
		t.Fatalf("expected installers to run in %s, got rokit=%s wally=%s", target, rokit.lastOption.WorkingDir, wally.lastOption.WorkingDir)
	}
}

func TestCreateNoInstall(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := filepath.Join(parent, "my-game")
	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeCreateCommand(createCommandDeps{
		getwd:          func() (string, error) { return parent, nil },
		writeConfig:    config.Write,
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "--template", "rojo-wally", "--no-install", "my-game")
	if err != nil {
		t.Fatalf("expected create --no-install success, got: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(target, config.FileName)); statErr != nil {
		t.Fatalf("expected %s to exist, got: %v", config.FileName, statErr)
	}
	if rokit.calls != 0 || wally.calls != 0 {
		t.Fatalf("expected installers to be skipped, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestCreateInteractiveNoFlags(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := filepath.Join(parent, "interactive-game")
	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	cmd := newCreateCmd(createCommandDeps{
		getwd:          func() (string, error) { return parent, nil },
		writeConfig:    config.Write,
		rokitInstaller: rokit,
		wallyInstaller: wally,
	})
	output := bytes.NewBuffer(nil)
	cmd.SetIn(strings.NewReader("interactive-game\nrojo-wally\nn\n"))
	cmd.SetOut(output)
	cmd.SetErr(bytes.NewBuffer(nil))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected interactive create success, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, config.FileName)); err != nil {
		t.Fatalf("expected %s in interactive scaffold, got: %v", config.FileName, err)
	}
	if rokit.calls != 0 || wally.calls != 0 {
		t.Fatalf("expected installers skipped after interactive no choice, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}

	text := output.String()
	if !strings.Contains(text, "◇ Project name:") {
		t.Fatalf("expected styled project prompt prefix, got: %q", text)
	}
}

func TestCreateRojoWallyTemplateScaffoldsClient(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := filepath.Join(parent, "rojo-wally-game")
	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeCreateCommand(createCommandDeps{
		getwd:          func() (string, error) { return parent, nil },
		writeConfig:    config.Write,
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "--template", "rojo-wally", "--no-install", "rojo-wally-game")
	if err != nil {
		t.Fatalf("expected create with rojo-wally template success, got: %v", err)
	}

	if _, err := os.Stat(filepath.Join(target, "src", "client", "init.client.luau")); err != nil {
		t.Fatalf("expected rojo-wally template client entrypoint, got: %v", err)
	}

	cfg, err := config.Load(filepath.Join(target, config.FileName))
	if err != nil {
		t.Fatalf("expected %s to load, got: %v", config.FileName, err)
	}
	if _, ok := cfg.Tasks["check"]; !ok {
		t.Fatalf("expected rojo-wally template check task, got %+v", cfg.Tasks)
	}
}

func TestCreateExistingDirectoryFails(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	target := filepath.Join(parent, "my-game")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("failed to create existing target: %v", err)
	}

	err := executeCreateCommand(createCommandDeps{
		getwd:          func() (string, error) { return parent, nil },
		writeConfig:    config.Write,
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	}, "--template", "rojo-wally", "my-game")
	if err == nil {
		t.Fatal("expected existing directory failure")
	}
	if !strings.Contains(err.Error(), "destination already exists") {
		t.Fatalf("expected destination exists message, got: %v", err)
	}
}

func TestCreateValidatesInputAndHelp(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"create", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected create help success, got: %v", err)
	}
	if !strings.Contains(output.String(), "create") {
		t.Fatalf("expected create help output, got: %q", output.String())
	}

	err := executeCreateCommand(createCommandDeps{
		getwd:          func() (string, error) { return t.TempDir(), nil },
		writeConfig:    config.Write,
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	}, "--template", "minimal")
	if err == nil {
		t.Fatal("expected missing name validation failure")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "project name is required") {
		t.Fatalf("expected project-name guidance, got: %v", err)
	}
}

func executeCreateCommand(deps createCommandDeps, args ...string) error {
	cmd := newCreateCmd(deps)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}
