package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"luumen/internal/process"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type fakeInstaller struct {
	calls      int
	addCalls   int
	lastOption tools.RunOptions
	lastTool   string
	lastAlias  string
	err        error
}

func (f *fakeInstaller) Install(_ context.Context, options tools.RunOptions) (process.Result, error) {
	f.calls++
	f.lastOption = options
	if f.err != nil {
		return process.Result{ExitCode: 1}, f.err
	}
	return process.Result{ExitCode: 0}, nil
}

func (f *fakeInstaller) Add(_ context.Context, tool string, alias string, options tools.RunOptions) (process.Result, error) {
	f.addCalls++
	f.lastOption = options
	f.lastTool = tool
	f.lastAlias = alias

	rokitPath := filepath.Join(options.WorkingDir, workspace.RokitConfigFile)
	if _, err := addToolToRokitConfig(rokitPath, tool); err != nil {
		return process.Result{ExitCode: 1}, err
	}

	if f.err != nil {
		return process.Result{ExitCode: 1}, f.err
	}
	return process.Result{ExitCode: 0}, nil
}

func TestInstallDefaultRunsToolsAndPackages(t *testing.T) {
	t.Parallel()

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true, HasWallyConfig: true}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	})
	if err != nil {
		t.Fatalf("expected install success, got: %v", err)
	}
	if rokit.calls != 1 || wally.calls != 1 {
		t.Fatalf("expected both installers to run once, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInstallToolsOnly(t *testing.T) {
	t.Parallel()

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true, HasWallyConfig: true}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "--tools")
	if err != nil {
		t.Fatalf("expected tools-only install success, got: %v", err)
	}
	if rokit.calls != 1 || wally.calls != 0 {
		t.Fatalf("expected only rokit install, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInstallPackagesOnly(t *testing.T) {
	t.Parallel()

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true, HasWallyConfig: true}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "--packages")
	if err != nil {
		t.Fatalf("expected packages-only install success, got: %v", err)
	}
	if rokit.calls != 0 || wally.calls != 1 {
		t.Fatalf("expected only wally install, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInstallFlagPrecedenceNoToolsOverTools(t *testing.T) {
	t.Parallel()

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true, HasWallyConfig: true}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	}, "--tools", "--packages", "--no-tools")
	if err != nil {
		t.Fatalf("expected precedence scenario success, got: %v", err)
	}
	if rokit.calls != 0 || wally.calls != 1 {
		t.Fatalf("expected no-tools to disable rokit while packages runs, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInstallMissingConfigDefaultErrors(t *testing.T) {
	t.Parallel()

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo"}, nil
		},
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	})
	if err == nil {
		t.Fatal("expected missing-config error")
	}
	if !strings.Contains(err.Error(), "no installable configuration found") {
		t.Fatalf("expected actionable missing-config message, got: %v", err)
	}
}

func TestInstallPartialRepoSetupRokitOnly(t *testing.T) {
	t.Parallel()

	rokit := &fakeInstaller{}
	wally := &fakeInstaller{}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true, HasWallyConfig: false}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: wally,
	})
	if err != nil {
		t.Fatalf("expected partial repo install success, got: %v", err)
	}
	if rokit.calls != 1 || wally.calls != 0 {
		t.Fatalf("expected only rokit to run, got rokit=%d wally=%d", rokit.calls, wally.calls)
	}
}

func TestInstallExplicitToolsWithoutRokitConfigErrors(t *testing.T) {
	t.Parallel()

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasWallyConfig: true}, nil
		},
		rokitInstaller: &fakeInstaller{},
		wallyInstaller: &fakeInstaller{},
	}, "--tools")
	if err == nil {
		t.Fatal("expected explicit-tools missing-config error")
	}
	if !strings.Contains(err.Error(), workspace.RokitConfigFile) {
		t.Fatalf("expected error to mention %s, got: %v", workspace.RokitConfigFile, err)
	}
}

func TestInstallMissingExecutableError(t *testing.T) {
	t.Parallel()

	rokit := &fakeInstaller{
		err: fmt.Errorf("tool missing: %w", &process.ExecError{Kind: process.ErrorKindNotFound, Command: "rokit install"}),
	}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: &fakeInstaller{},
	})
	if err == nil {
		t.Fatal("expected executable-not-found error")
	}
	if !strings.Contains(err.Error(), "Rokit executable was not found in PATH") {
		t.Fatalf("expected missing PATH guidance, got: %v", err)
	}
}

func TestInstallUnderlyingFailureIsActionable(t *testing.T) {
	t.Parallel()

	rokitErr := errors.New("install failed")
	rokit := &fakeInstaller{err: rokitErr}

	err := executeInstallCommand(installCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRokitConfig: true}, nil
		},
		rokitInstaller: rokit,
		wallyInstaller: &fakeInstaller{},
	})
	if err == nil {
		t.Fatal("expected underlying install failure")
	}
	if !strings.Contains(err.Error(), "failed to install tools via Rokit") {
		t.Fatalf("expected actionable install failure message, got: %v", err)
	}
	if !errors.Is(err, rokitErr) {
		t.Fatalf("expected wrapped underlying error, got: %v", err)
	}
}

func TestInstallHelpIsAvailableFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"install", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected install help to render, got: %v", err)
	}

	if !strings.Contains(output.String(), "--tools") || !strings.Contains(output.String(), "--packages") {
		t.Fatalf("expected install help to include flags, got: %q", output.String())
	}
}

func TestInstallAliasHelpIsAvailableFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"i", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected install alias help to render, got: %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "Usage:\n  luu install [flags]") || !strings.Contains(text, "Aliases:\n  install, i") {
		t.Fatalf("expected install alias to resolve to install help, got: %q", text)
	}
}

func executeInstallCommand(deps installCommandDeps, args ...string) error {
	cmd := newInstallCmd(deps)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}
