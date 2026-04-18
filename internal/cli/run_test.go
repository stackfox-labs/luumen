package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"luumen/internal/config"
	"luumen/internal/tasks"
	"luumen/internal/workspace"
)

type fakeTaskRunner struct {
	calls    int
	lastTask string
	lastCfg  *config.Config
	lastOpts tasks.RunOptions
	err      error
}

func (f *fakeTaskRunner) RunNamedTask(_ context.Context, taskName string, cfg *config.Config, options tasks.RunOptions) error {
	f.calls++
	f.lastTask = taskName
	f.lastCfg = cfg
	f.lastOpts = options
	return f.err
}

func TestRunCommandExecutesNamedTask(t *testing.T) {
	t.Parallel()

	runner := &fakeTaskRunner{}
	cfg := &config.Config{Tasks: map[string]config.TaskValue{"build": config.NewTaskValue("rojo build")}}

	err := executeRunCommand(runCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasLuumenConfig: true, LuumenConfigPath: "repo/" + workspace.LuumenConfigFile}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return cfg, nil
		},
		taskRunner: runner,
	}, "build")
	if err != nil {
		t.Fatalf("expected run command success, got: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("expected one task invocation, got %d", runner.calls)
	}
	if runner.lastTask != "build" {
		t.Fatalf("expected task build, got %q", runner.lastTask)
	}
	if runner.lastCfg != cfg {
		t.Fatal("expected loaded config to be passed to task runner")
	}
	if runner.lastOpts.WorkingDir != "repo" {
		t.Fatalf("expected working dir repo, got %q", runner.lastOpts.WorkingDir)
	}
}

func TestRunCommandRequiresLuumenConfig(t *testing.T) {
	t.Parallel()

	err := executeRunCommand(runCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasLuumenConfig: false}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: &fakeTaskRunner{},
	}, "build")
	if err == nil {
		t.Fatal("expected missing config error")
	}
	if !strings.Contains(err.Error(), workspace.LuumenConfigFile) {
		t.Fatalf("expected missing config guidance, got: %v", err)
	}
}

func TestRunCommandTaskFailurePropagates(t *testing.T) {
	t.Parallel()

	taskErr := errors.New("task failed")
	runner := &fakeTaskRunner{err: taskErr}

	err := executeRunCommand(runCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasLuumenConfig: true, LuumenConfigPath: "repo/" + workspace.LuumenConfigFile}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{Tasks: map[string]config.TaskValue{"build": config.NewTaskValue("rojo build")}}, nil
		},
		taskRunner: runner,
	}, "build")
	if err == nil {
		t.Fatal("expected task failure")
	}
	if !errors.Is(err, taskErr) {
		t.Fatalf("expected wrapped task error, got: %v", err)
	}
}

func TestRunHelpAvailableFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"run", "--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected run help success, got: %v", err)
	}
	if !strings.Contains(output.String(), "Run executes a task") {
		t.Fatalf("expected run help output, got: %q", output.String())
	}
}

func executeRunCommand(deps runCommandDeps, args ...string) error {
	cmd := newRunCmd(deps)
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}
