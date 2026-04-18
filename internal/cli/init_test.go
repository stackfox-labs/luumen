package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luumen/internal/config"
	"luumen/internal/workspace"
)

type capturedConfigWrite struct {
	calls int
	path  string
	cfg   *config.Config
	err   error
}

func (c *capturedConfigWrite) Write(path string, cfg *config.Config) error {
	c.calls++
	c.path = path
	c.cfg = cfg
	return c.err
}

func TestInitHelpFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"init", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected init help to render, got: %v", err)
	}
	if !strings.Contains(output.String(), "luu init") {
		t.Fatalf("expected init help text, got: %q", output.String())
	}
}

func TestInitAdoptionRokitWallyRojo(t *testing.T) {
	t.Parallel()

	writer := &capturedConfigWrite{}
	root := filepath.Clean("repo")
	state := workspace.Workspace{
		RootPath:         root,
		LuumenConfigPath: filepath.Join(root, workspace.LuumenConfigFile),
		HasRokitConfig:   true,
		HasWallyConfig:   true,
		HasRojoProject:   true,
		RojoProjectPaths: []string{filepath.Join(root, "default.project.json")},
	}

	err := executeInitCommand(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig: writer.Write,
	})
	if err != nil {
		t.Fatalf("expected init success, got: %v", err)
	}
	if writer.calls != 1 {
		t.Fatalf("expected one config write, got %d", writer.calls)
	}
	if writer.path != state.LuumenConfigPath {
		t.Fatalf("expected write path %q, got %q", state.LuumenConfigPath, writer.path)
	}
	if writer.cfg == nil {
		t.Fatal("expected generated config")
	}

	if writer.cfg.Project.Name != filepath.Base(root) {
		t.Fatalf("expected project name %q, got %q", filepath.Base(root), writer.cfg.Project.Name)
	}
	if !writer.cfg.Install.Tools || !writer.cfg.Install.Packages {
		t.Fatalf("expected both install categories true, got %+v", writer.cfg.Install)
	}

	assertTask(t, writer.cfg, "dev", []string{"rojo sourcemap default.project.json --output sourcemap.json", "rojo serve default.project.json"})
	assertTask(t, writer.cfg, "build", []string{"rojo build default.project.json --output build.rbxl"})
	assertTask(t, writer.cfg, "lint", []string{"selene src"})
	assertTask(t, writer.cfg, "format", []string{"stylua src"})
	assertTask(t, writer.cfg, "test", []string{"lune run test"})
}

func TestInitAdoptionPartialSetup(t *testing.T) {
	t.Parallel()

	writer := &capturedConfigWrite{}
	root := filepath.Clean("repo")
	state := workspace.Workspace{
		RootPath:         root,
		LuumenConfigPath: filepath.Join(root, workspace.LuumenConfigFile),
		HasRokitConfig:   true,
		HasWallyConfig:   false,
		HasRojoProject:   true,
		RojoProjectPaths: []string{filepath.Join(root, "games", "default.project.json")},
	}

	err := executeInitCommand(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig: writer.Write,
	})
	if err != nil {
		t.Fatalf("expected init success for partial setup, got: %v", err)
	}
	if writer.calls != 1 {
		t.Fatalf("expected one config write, got %d", writer.calls)
	}
	if !writer.cfg.Install.Tools || writer.cfg.Install.Packages {
		t.Fatalf("expected tools-only install settings, got %+v", writer.cfg.Install)
	}
	assertTask(t, writer.cfg, "dev", []string{"rojo sourcemap games/default.project.json --output sourcemap.json", "rojo serve games/default.project.json"})
}

func TestInitRefusesExistingLuumenConfig(t *testing.T) {
	t.Parallel()

	writer := &capturedConfigWrite{}
	state := workspace.Workspace{
		RootPath:         "repo",
		LuumenConfigPath: filepath.Join("repo", workspace.LuumenConfigFile),
		HasLuumenConfig:  true,
	}

	err := executeInitCommand(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig: writer.Write,
	})
	if err == nil {
		t.Fatal("expected existing config error")
	}
	if !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected overwrite-protection error, got: %v", err)
	}
	if writer.calls != 0 {
		t.Fatalf("expected no writes when config already exists, got %d", writer.calls)
	}
}

func TestInitFallsBackToBasicConfigWhenRojoInfoMissing(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "keep.txt"), []byte("data\n"), 0o644); err != nil {
		t.Fatalf("failed to seed existing repo file: %v", err)
	}

	writer := &capturedConfigWrite{}
	cmd := newInitCmd(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{
				RootPath:         repo,
				LuumenConfigPath: filepath.Join(repo, workspace.LuumenConfigFile),
				HasRokitConfig:   true,
				HasWallyConfig:   true,
			}, nil
		},
		writeConfig: writer.Write,
	})
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected basic config fallback success, got: %v", err)
	}
	if writer.calls != 1 {
		t.Fatalf("expected one config write, got %d", writer.calls)
	}
	if writer.cfg == nil {
		t.Fatal("expected basic config to be written")
	}
	if !writer.cfg.Install.Tools || !writer.cfg.Install.Packages {
		t.Fatalf("expected install settings to mirror detected repo config, got %+v", writer.cfg.Install)
	}
	if len(writer.cfg.Tasks) != 0 {
		t.Fatalf("expected basic config without generated tasks, got %+v", writer.cfg.Tasks)
	}
}

func TestInitDetectWorkspaceFailure(t *testing.T) {
	t.Parallel()

	detectErr := errors.New("detect failed")
	err := executeInitCommand(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{}, detectErr
		},
		writeConfig: (&capturedConfigWrite{}).Write,
	})
	if err == nil {
		t.Fatal("expected detect failure")
	}
	if !errors.Is(err, detectErr) {
		t.Fatalf("expected wrapped detect error, got: %v", err)
	}
}

func TestInitCreateInPlaceOnConfirmation(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	state := workspace.Workspace{
		RootPath:         repo,
		LuumenConfigPath: filepath.Join(repo, workspace.LuumenConfigFile),
	}
	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	cmd := newInitCmd(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig:    config.Write,
		rokitInstaller: rokit,
		wallyInstaller: wally,
	})
	cmd.SetIn(strings.NewReader("y\n"))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected init create-in-place success, got: %v", err)
	}

	required := []string{
		filepath.Join(repo, workspace.LuumenConfigFile),
	}
	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected scaffolded file %s, got: %v", path, err)
		}
	}

	if rokit.calls != 0 || wally.calls != 0 {
		t.Fatalf("expected installers to follow minimal template settings, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInitCreateInPlaceDeclined(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	state := workspace.Workspace{
		RootPath:         repo,
		LuumenConfigPath: filepath.Join(repo, workspace.LuumenConfigFile),
	}
	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	cmd := newInitCmd(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig:    config.Write,
		rokitInstaller: rokit,
		wallyInstaller: wally,
	})
	cmd.SetIn(strings.NewReader("n\n"))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected init cancellation when prompt is declined")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cancelled") {
		t.Fatalf("expected cancellation message, got: %v", err)
	}
	if rokit.calls != 0 || wally.calls != 0 {
		t.Fatalf("expected installers not to run, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInitCreateInPlaceNonEmptyDirectoryOffersBasicConfig(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	state := workspace.Workspace{
		RootPath:         repo,
		LuumenConfigPath: filepath.Join(repo, workspace.LuumenConfigFile),
	}
	if err := os.WriteFile(filepath.Join(repo, "keep.txt"), []byte("data\n"), 0o644); err != nil {
		t.Fatalf("failed to seed non-empty directory: %v", err)
	}

	writer := &capturedConfigWrite{}
	cmd := newInitCmd(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig:    writer.Write,
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	})
	cmd.SetIn(strings.NewReader("y\ny\n"))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected basic config fallback success, got: %v", err)
	}
	if writer.calls != 1 {
		t.Fatalf("expected one config write, got %d", writer.calls)
	}
	if writer.cfg == nil {
		t.Fatal("expected basic config to be generated")
	}
	if writer.cfg.Project.Name != filepath.Base(repo) {
		t.Fatalf("expected basic config project name %q, got %q", filepath.Base(repo), writer.cfg.Project.Name)
	}
	if writer.cfg.Install.Tools || writer.cfg.Install.Packages {
		t.Fatalf("expected empty install settings for plain fallback, got %+v", writer.cfg.Install)
	}
	if len(writer.cfg.Tasks) != 0 {
		t.Fatalf("expected no generated tasks for basic config, got %+v", writer.cfg.Tasks)
	}
}

func TestInitCreateInPlaceNonEmptyDirectoryCanDeclineBasicConfigFallback(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	state := workspace.Workspace{
		RootPath:         repo,
		LuumenConfigPath: filepath.Join(repo, workspace.LuumenConfigFile),
	}
	if err := os.WriteFile(filepath.Join(repo, "keep.txt"), []byte("data\n"), 0o644); err != nil {
		t.Fatalf("failed to seed non-empty directory: %v", err)
	}

	writer := &capturedConfigWrite{}
	cmd := newInitCmd(initCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return state, nil
		},
		writeConfig:    writer.Write,
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	})
	cmd.SetIn(strings.NewReader("y\nn\n"))
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected cancellation when basic config fallback is declined")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "cancelled") {
		t.Fatalf("expected cancellation guidance, got: %v", err)
	}
	if writer.calls != 0 {
		t.Fatalf("expected no config write when fallback is declined, got %d", writer.calls)
	}
}

func assertTask(t *testing.T, cfg *config.Config, name string, expected []string) {
	t.Helper()

	task, ok := cfg.Tasks[name]
	if !ok {
		t.Fatalf("expected task %q", name)
	}
	if len(task.Steps) != len(expected) {
		t.Fatalf("expected %d steps for %q, got %#v", len(expected), name, task.Steps)
	}
	for index := range expected {
		if task.Steps[index] != expected[index] {
			t.Fatalf("expected task %q entry %d to be %q, got %q", name, index, expected[index], task.Steps[index])
		}
	}
}

func executeInitCommand(deps initCommandDeps, args ...string) error {
	cmd := newInitCmd(deps)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}
