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
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type fakeRojoWorkflowRunner struct {
	serveCalls     [][]string
	buildCalls     [][]string
	sourcemapCalls [][]string
	serveErr       error
	buildErr       error
	sourcemapErr   error
}

func (f *fakeRojoWorkflowRunner) Serve(_ context.Context, args []string, _ tools.RunOptions) (process.Result, error) {
	f.serveCalls = append(f.serveCalls, append([]string(nil), args...))
	if f.serveErr != nil {
		return process.Result{ExitCode: 1}, f.serveErr
	}
	return process.Result{ExitCode: 0}, nil
}

func (f *fakeRojoWorkflowRunner) Build(_ context.Context, args []string, _ tools.RunOptions) (process.Result, error) {
	f.buildCalls = append(f.buildCalls, append([]string(nil), args...))
	if f.buildErr != nil {
		return process.Result{ExitCode: 1}, f.buildErr
	}
	return process.Result{ExitCode: 0}, nil
}

func (f *fakeRojoWorkflowRunner) Sourcemap(_ context.Context, args []string, _ tools.RunOptions) (process.Result, error) {
	f.sourcemapCalls = append(f.sourcemapCalls, append([]string(nil), args...))
	if f.sourcemapErr != nil {
		return process.Result{ExitCode: 1}, f.sourcemapErr
	}
	return process.Result{ExitCode: 0}, nil
}

type fakeWorkflowTaskRunner struct {
	calls    int
	lastTask string
	lastCfg  *config.Config
	err      error
}

func (f *fakeWorkflowTaskRunner) RunNamedTask(_ context.Context, taskName string, cfg *config.Config, _ tasks.RunOptions) error {
	f.calls++
	f.lastTask = taskName
	f.lastCfg = cfg
	return f.err
}

func TestServeFallbackDefault(t *testing.T) {
	t.Parallel()

	rojo := &fakeRojoWorkflowRunner{}
	err := executeWorkflowCommand(t, newServeCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRojoProject: true, RojoProjectPaths: []string{"repo/default.project.json"}}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: &fakeWorkflowTaskRunner{},
		rojoRunner: rojo,
	}))
	if err != nil {
		t.Fatalf("expected serve default success, got: %v", err)
	}
	if len(rojo.serveCalls) != 1 {
		t.Fatalf("expected one rojo serve call, got %d", len(rojo.serveCalls))
	}
	expected := []string{"default.project.json"}
	if !reflect.DeepEqual(rojo.serveCalls[0], expected) {
		t.Fatalf("expected serve args %#v, got %#v", expected, rojo.serveCalls[0])
	}
}

func TestServeUsesCommandOverride(t *testing.T) {
	t.Parallel()

	rojo := &fakeRojoWorkflowRunner{}
	runner := &fakeWorkflowTaskRunner{}
	err := executeWorkflowCommand(t, newServeCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasLuumenConfig: true, LuumenConfigPath: "repo/luumen.toml"}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{Commands: map[string]config.TaskValue{"serve": config.NewTaskValue("echo custom")}}, nil
		},
		taskRunner: runner,
		rojoRunner: rojo,
	}))
	if err != nil {
		t.Fatalf("expected serve override success, got: %v", err)
	}
	if runner.calls != 1 || runner.lastTask != "__builtin_serve" {
		t.Fatalf("expected override runner call __builtin_serve, got calls=%d task=%q", runner.calls, runner.lastTask)
	}
	if len(rojo.serveCalls) != 0 {
		t.Fatalf("expected no default rojo serve when override exists, got %d", len(rojo.serveCalls))
	}
}

func TestBuildPassesThroughFlags(t *testing.T) {
	t.Parallel()

	rojo := &fakeRojoWorkflowRunner{}
	err := executeWorkflowCommand(t, newBuildCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRojoProject: true, RojoProjectPaths: []string{"repo/default.project.json"}}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: &fakeWorkflowTaskRunner{},
		rojoRunner: rojo,
	}), "--plugin", "--watch")
	if err != nil {
		t.Fatalf("expected build success, got: %v", err)
	}
	if len(rojo.buildCalls) != 1 {
		t.Fatalf("expected one rojo build call, got %d", len(rojo.buildCalls))
	}
	expected := []string{"default.project.json", "--output", "build.rbxl", "--plugin", "--watch"}
	if !reflect.DeepEqual(rojo.buildCalls[0], expected) {
		t.Fatalf("expected build args %#v, got %#v", expected, rojo.buildCalls[0])
	}
}

func TestDevFallbackRunsSourcemapThenServe(t *testing.T) {
	t.Parallel()

	rojo := &fakeRojoWorkflowRunner{}
	err := executeWorkflowCommand(t, newDevCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{RootPath: "repo", HasRojoProject: true, RojoProjectPaths: []string{"repo/default.project.json"}}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{}, nil
		},
		taskRunner: &fakeWorkflowTaskRunner{},
		rojoRunner: rojo,
	}))
	if err != nil {
		t.Fatalf("expected dev fallback success, got: %v", err)
	}
	if len(rojo.sourcemapCalls) != 1 || len(rojo.serveCalls) != 1 {
		t.Fatalf("expected sourcemap+serve default calls, got sourcemap=%d serve=%d", len(rojo.sourcemapCalls), len(rojo.serveCalls))
	}
}

func TestDevSkipsCanonicalDefaultOverride(t *testing.T) {
	t.Parallel()

	rojo := &fakeRojoWorkflowRunner{}
	runner := &fakeWorkflowTaskRunner{}
	err := executeWorkflowCommand(t, newDevCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{
				RootPath:         "repo",
				HasLuumenConfig:  true,
				LuumenConfigPath: "repo/luumen.toml",
				HasRojoProject:   true,
				RojoProjectPaths: []string{"repo/default.project.json"},
			}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{Commands: map[string]config.TaskValue{
				"dev": config.NewTaskValue("luu sourcemap", "rojo serve default.project.json"),
			}}, nil
		},
		taskRunner: runner,
		rojoRunner: rojo,
	}))
	if err != nil {
		t.Fatalf("expected canonical default override to run as built-in workflow, got: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("expected no task-runner override call for canonical default, got %d", runner.calls)
	}
	if len(rojo.sourcemapCalls) != 1 || len(rojo.serveCalls) != 1 {
		t.Fatalf("expected default sourcemap+serve path, got sourcemap=%d serve=%d", len(rojo.sourcemapCalls), len(rojo.serveCalls))
	}
}

func TestDevSkipsLuuServeAliasOverride(t *testing.T) {
	t.Parallel()

	rojo := &fakeRojoWorkflowRunner{}
	runner := &fakeWorkflowTaskRunner{}
	err := executeWorkflowCommand(t, newDevCmd(workflowCommandDeps{
		detectWorkspace: func(_ string) (workspace.Workspace, error) {
			return workspace.Workspace{
				RootPath:         "repo",
				HasLuumenConfig:  true,
				LuumenConfigPath: "repo/luumen.toml",
				HasRojoProject:   true,
				RojoProjectPaths: []string{"repo/default.project.json"},
			}, nil
		},
		loadConfig: func(_ string) (*config.Config, error) {
			return &config.Config{Commands: map[string]config.TaskValue{
				"dev": config.NewTaskValue("luu sourcemap", "luu serve"),
			}}, nil
		},
		taskRunner: runner,
		rojoRunner: rojo,
	}))
	if err != nil {
		t.Fatalf("expected luu serve alias override to run as built-in workflow, got: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("expected no task-runner override call for luu serve alias, got %d", runner.calls)
	}
	if len(rojo.sourcemapCalls) != 1 || len(rojo.serveCalls) != 1 {
		t.Fatalf("expected default sourcemap+serve path, got sourcemap=%d serve=%d", len(rojo.sourcemapCalls), len(rojo.serveCalls))
	}
}

func TestWorkflowHelpFromRoot(t *testing.T) {
	t.Parallel()

	root := NewRootCmd()
	output := bytes.NewBuffer(nil)
	root.SetOut(output)
	root.SetErr(output)
	root.SetArgs([]string{"build", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected build help success, got: %v", err)
	}
	if !strings.Contains(output.String(), "--plugin") || !strings.Contains(output.String(), "--watch") {
		t.Fatalf("expected build help to include pass-through flags, got: %q", output.String())
	}
}

func TestRojoServeReadyWriterAnnouncesOnce(t *testing.T) {
	t.Parallel()

	output := bytes.NewBuffer(nil)
	writer := newRojoServeReadyWriter(output)

	if _, err := writer.Write([]byte("Rojo server listening:\n")); err != nil {
		t.Fatalf("expected first write success, got: %v", err)
	}
	if _, err := writer.Write([]byte("Visit http://localhost:34872/ in your browser\n")); err != nil {
		t.Fatalf("expected second write success, got: %v", err)
	}

	text := output.String()
	if strings.Count(text, "Rojo server started") != 1 {
		t.Fatalf("expected one readiness announcement, got: %q", text)
	}
}

func TestRojoServeReadyWriterIncludesURLWhenAvailable(t *testing.T) {
	t.Parallel()

	output := bytes.NewBuffer(nil)
	writer := newRojoServeReadyWriter(output)

	if _, err := writer.Write([]byte("Rojo server listening:\n  Address: localhost\n  Port:    34872\n")); err != nil {
		t.Fatalf("expected write success, got: %v", err)
	}

	text := output.String()
	if !strings.Contains(text, "http://localhost:34872/") {
		t.Fatalf("expected readiness announcement to include URL, got: %q", text)
	}
}

func executeWorkflowCommand(t *testing.T, cmd *cobra.Command, args ...string) error {
	t.Helper()
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetErr(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	return cmd.Execute()
}
