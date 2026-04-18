package tasks

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"luumen/internal/config"
	"luumen/internal/process"
)

type fakeShellRunner struct {
	steps         []string
	failOnCommand map[string]error
}

func (f *fakeShellRunner) RunShell(_ context.Context, command string, _ process.Options) (process.Result, error) {
	f.steps = append(f.steps, command)
	if err, ok := f.failOnCommand[command]; ok {
		return process.Result{ExitCode: 1}, err
	}
	return process.Result{ExitCode: 0}, nil
}

func TestRunNamedTaskSingleCommand(t *testing.T) {
	t.Parallel()

	runner := &fakeShellRunner{}
	engine := NewEngine(runner, "luu")
	cfg := &config.Config{Tasks: map[string]config.TaskValue{"fmt": config.NewTaskValue("stylua src")}}

	if err := engine.RunNamedTask(context.Background(), "fmt", cfg, RunOptions{}); err != nil {
		t.Fatalf("expected single task success, got: %v", err)
	}
	if !reflect.DeepEqual(runner.steps, []string{"stylua src"}) {
		t.Fatalf("expected one step, got %#v", runner.steps)
	}
}

func TestRunNamedTaskSequentialCommands(t *testing.T) {
	t.Parallel()

	runner := &fakeShellRunner{}
	engine := NewEngine(runner, "luu")
	cfg := &config.Config{Tasks: map[string]config.TaskValue{"build-all": config.NewTaskValue("luu sourcemap", "rojo build default.project.json --output build.rbxl")}}

	if err := engine.RunNamedTask(context.Background(), "build-all", cfg, RunOptions{}); err != nil {
		t.Fatalf("expected sequential task success, got: %v", err)
	}

	expected := []string{"luu sourcemap", "rojo build default.project.json --output build.rbxl"}
	if !reflect.DeepEqual(runner.steps, expected) {
		t.Fatalf("expected sequential steps %#v, got %#v", expected, runner.steps)
	}
}

func TestRunNamedTaskStopsOnFailure(t *testing.T) {
	t.Parallel()

	failure := errors.New("command failed")
	runner := &fakeShellRunner{failOnCommand: map[string]error{"fail": failure}}
	engine := NewEngine(runner, "luu")
	cfg := &config.Config{Tasks: map[string]config.TaskValue{"ci": config.NewTaskValue("ok-one", "fail", "ok-two")}}

	err := engine.RunNamedTask(context.Background(), "ci", cfg, RunOptions{})
	if err == nil {
		t.Fatal("expected task failure")
	}
	if !errors.Is(err, failure) {
		t.Fatalf("expected wrapped failure, got: %v", err)
	}
	expected := []string{"ok-one", "fail"}
	if !reflect.DeepEqual(runner.steps, expected) {
		t.Fatalf("expected halt after failure %#v, got %#v", expected, runner.steps)
	}
}

func TestRunNamedTaskRecursiveAndCycleDetection(t *testing.T) {
	t.Parallel()

	runner := &fakeShellRunner{}
	engine := NewEngine(runner, "luu")
	cfg := &config.Config{
		Tasks: map[string]config.TaskValue{
			"dev":       config.NewTaskValue("luu run sourcemap", "luu run serve"),
			"sourcemap": config.NewTaskValue("rojo sourcemap default.project.json --output sourcemap.json"),
			"serve":     config.NewTaskValue("rojo serve default.project.json"),
		},
	}

	if err := engine.RunNamedTask(context.Background(), "dev", cfg, RunOptions{}); err != nil {
		t.Fatalf("expected recursive task success, got: %v", err)
	}
	expected := []string{
		"rojo sourcemap default.project.json --output sourcemap.json",
		"rojo serve default.project.json",
	}
	if !reflect.DeepEqual(runner.steps, expected) {
		t.Fatalf("expected nested steps %#v, got %#v", expected, runner.steps)
	}

	cycleCfg := &config.Config{
		Tasks: map[string]config.TaskValue{
			"a": config.NewTaskValue("luu run b"),
			"b": config.NewTaskValue("luu run a"),
		},
	}
	cycleErr := engine.RunNamedTask(context.Background(), "a", cycleCfg, RunOptions{})
	if cycleErr == nil {
		t.Fatal("expected cycle detection error")
	}
	if !errors.Is(cycleErr, ErrTaskCycle) {
		t.Fatalf("expected ErrTaskCycle, got: %v", cycleErr)
	}
	if !strings.Contains(cycleErr.Error(), "a -> b -> a") {
		t.Fatalf("expected cycle path in error, got: %v", cycleErr)
	}
}

func TestNormalizeTaskValueRejectsEmpty(t *testing.T) {
	t.Parallel()

	_, err := NormalizeTaskValue(config.TaskValue{})
	if err == nil {
		t.Fatal("expected empty task validation error")
	}
}

func TestParseNestedRunCommand(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		command string
		task    string
		ok      bool
	}{
		"simple":       {command: "luu run dev", task: "dev", ok: true},
		"windows-exe":  {command: "luu.exe run ci", task: "ci", ok: true},
		"relative":     {command: "./luu run build", task: "build", ok: true},
		"non-matching": {command: "rojo serve", ok: false},
		"extra-args":   {command: "luu run dev --watch", ok: false},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			task, ok := parseNestedRunCommand(tc.command, "luu")
			if ok != tc.ok {
				t.Fatalf("expected ok=%v, got %v", tc.ok, ok)
			}
			if task != tc.task {
				t.Fatalf("expected task %q, got %q", tc.task, task)
			}
		})
	}
}

func TestRunNamedTaskMissingTask(t *testing.T) {
	t.Parallel()

	engine := NewEngine(&fakeShellRunner{}, "luu")
	cfg := &config.Config{Tasks: map[string]config.TaskValue{"fmt": config.NewTaskValue("stylua src")}}

	err := engine.RunNamedTask(context.Background(), "lint", cfg, RunOptions{})
	if err == nil {
		t.Fatal("expected missing task error")
	}
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%s", "lint")) {
		t.Fatalf("expected missing task name in error, got: %v", err)
	}
}
