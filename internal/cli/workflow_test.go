package cli

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"luumen/internal/config"
	"luumen/internal/process"
	"luumen/internal/tasks"
	"luumen/internal/workspace"
)

type fakeWorkflowRunner struct {
	calls    int
	lastTask string
	lastCfg  *config.Config
}

func (f *fakeWorkflowRunner) RunNamedTask(_ context.Context, taskName string, cfg *config.Config, _ tasks.RunOptions) error {
	f.calls++
	f.lastTask = taskName
	f.lastCfg = cfg
	return nil
}

func TestDevDefaultResolvesRojoPlan(t *testing.T) {
	t.Parallel()

	runner := &fakeWorkflowRunner{}
	err := executeWorkflowCommand(t, newDevCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRojoProject: true, RojoProjectPaths: []string{"repo/default.project.json"}}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: runner,
	}))
	if err != nil {
		t.Fatalf("expected dev success, got: %v", err)
	}
	if runner.calls != 1 || runner.lastTask != "__builtin_dev" {
		t.Fatalf("expected one synthetic dev task invocation, got calls=%d task=%q", runner.calls, runner.lastTask)
	}

	task := runner.lastCfg.Tasks["__builtin_dev"]
	expected := []string{"rojo sourcemap default.project.json --output sourcemap.json", "rojo serve default.project.json"}
	if !reflect.DeepEqual(task.Commands, expected) {
		t.Fatalf("expected dev plan %#v, got %#v", expected, task.Commands)
	}
}

func TestBuildDefaultResolvesRojoPlan(t *testing.T) {
	t.Parallel()

	runner := &fakeWorkflowRunner{}
	err := executeWorkflowCommand(t, newBuildCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRojoProject: true, RojoProjectPaths: []string{"repo/default.project.json"}}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: runner,
	}))
	if err != nil {
		t.Fatalf("expected build success, got: %v", err)
	}

	task := runner.lastCfg.Tasks["__builtin_build"]
	expected := []string{"rojo build default.project.json --output build.rbxl"}
	if !reflect.DeepEqual(task.Commands, expected) {
		t.Fatalf("expected build plan %#v, got %#v", expected, task.Commands)
	}
}

func TestLintUsesCommandOverride(t *testing.T) {
	t.Parallel()

	runner := &fakeWorkflowRunner{}
	err := executeWorkflowCommand(t, newLintCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasLuumenConfig: true, LuumenConfigPath: "repo/luumen.toml"}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{Commands: map[string]config.TaskValue{"lint": config.NewTaskValue("selene src")}}, nil
		},
		taskRunner: runner,
	}))
	if err != nil {
		t.Fatalf("expected lint override success, got: %v", err)
	}

	task := runner.lastCfg.Tasks["__builtin_lint"]
	expected := []string{"selene src"}
	if !reflect.DeepEqual(task.Commands, expected) {
		t.Fatalf("expected lint override %#v, got %#v", expected, task.Commands)
	}
}

func TestLintRequiresConfiguredCommand(t *testing.T) {
	t.Parallel()

	err := executeWorkflowCommand(t, newLintCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasLuumenConfig: true, LuumenConfigPath: "repo/luumen.toml"}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: &fakeWorkflowRunner{},
	}))
	if err == nil {
		t.Fatal("expected lint configuration error")
	}
	if !strings.Contains(err.Error(), "[commands].lint") {
		t.Fatalf("expected actionable lint configuration guidance, got: %v", err)
	}
}

func TestWorkflowPlanOutputIncludesResolvedSteps(t *testing.T) {
	t.Parallel()

	output := bytes.NewBuffer(nil)
	cmd := newDevCmd(defaultWorkflowCommandDeps())
	cmd.SetOut(output)
	printWorkflowPlan(cmd, "my-game", "dev", []string{"one", "two"})

	text := output.String()
	if !strings.Contains(text, "[luu] workspace: my-game") || !strings.Contains(text, "[luu] resolved: 2 steps") {
		t.Fatalf("expected workflow plan output, got: %q", text)
	}
	if strings.Contains(text, "step 1/2") {
		t.Fatalf("expected step lines to be streamed at execution time, got: %q", text)
	}
}

func TestWorkflowBuiltInsExposedFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"lint", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected lint help success, got: %v", err)
	}
	if !strings.Contains(output.String(), "Run static analysis") {
		t.Fatalf("expected lint help content, got: %q", output.String())
	}
}

func TestRootHelpShowsCommandGroups(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"--help"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected root help success, got: %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "Workflow Commands") || !strings.Contains(text, "Dependency Commands") {
		t.Fatalf("expected categorized command sections in help, got: %q", text)
	}
}

func TestRootVersionFlagPrintsVersion(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"--version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("expected root version success, got: %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "luu version") || !strings.Contains(text, "dev") {
		t.Fatalf("expected root version output, got: %q", text)
	}
}

func TestDevMissingRojoProjectPrintsPlannedStepsBeforeFailing(t *testing.T) {
	t.Parallel()

	cmd := newDevCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRojoProject: false}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: &fakeWorkflowRunner{},
	})

	output := bytes.NewBuffer(nil)
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing project error")
	}

	text := output.String()
	if !strings.Contains(text, "[luu] command: dev") || !strings.Contains(text, "[luu] resolved: 2 steps") {
		t.Fatalf("expected planned command output before failure, got: %q", text)
	}
}

func TestWorkflowStepRunnerPrintsStepBeforeExecution(t *testing.T) {
	t.Parallel()

	output := bytes.NewBuffer(nil)
	cmd := newDevCmd(defaultWorkflowCommandDeps())
	cmd.SetOut(output)

	inner := &fakeWorkflowShellRunner{}
	runner := newWorkflowStepShellRunner(cmd, inner, 2)
	if _, err := runner.RunShell(context.Background(), "echo one", process.Options{}); err != nil {
		t.Fatalf("expected first step success, got: %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "step 1/2") || !strings.Contains(text, "echo one") {
		t.Fatalf("expected streamed step output, got: %q", text)
	}
}

type fakeWorkflowShellRunner struct{}

func (f *fakeWorkflowShellRunner) RunShell(_ context.Context, _ string, _ process.Options) (process.Result, error) {
	return process.Result{ExitCode: 0}, nil
}

func executeWorkflowCommand(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}
