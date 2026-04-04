package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"luumen/internal/process"
)

func TestRojoServeBuildAndSourcemapCommands(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	rojo := NewRojo(runner, "")

	if _, err := rojo.Serve(context.Background(), []string{"default.project.json", "--port", "34872"}, RunOptions{WorkingDir: t.TempDir()}); err != nil {
		t.Fatalf("expected serve success, got: %v", err)
	}
	if _, err := rojo.Build(context.Background(), []string{"default.project.json", "--output", "build.rbxl"}, RunOptions{}); err != nil {
		t.Fatalf("expected build success, got: %v", err)
	}
	if _, err := rojo.Sourcemap(context.Background(), []string{"default.project.json", "--output", "sourcemap.json"}, RunOptions{}); err != nil {
		t.Fatalf("expected sourcemap success, got: %v", err)
	}

	if len(runner.calls) != 3 {
		t.Fatalf("expected three command calls, got %d", len(runner.calls))
	}

	assertArgs(t, runner.calls[0].command.Args, []string{"serve", "default.project.json", "--port", "34872"})
	assertArgs(t, runner.calls[1].command.Args, []string{"build", "default.project.json", "--output", "build.rbxl"})
	assertArgs(t, runner.calls[2].command.Args, []string{"sourcemap", "default.project.json", "--output", "sourcemap.json"})
}

func TestRojoProjectFilesDetection(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.project.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("failed to write b.project.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.project.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("failed to write a.project.json: %v", err)
	}

	rojo := NewRojo(nil, "")
	files, err := rojo.ProjectFiles(root)
	if err != nil {
		t.Fatalf("expected project file detection success, got: %v", err)
	}

	expected := []string{
		filepath.Join(root, "a.project.json"),
		filepath.Join(root, "b.project.json"),
	}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("expected sorted project files %#v, got %#v", expected, files)
	}
}

func TestRojoWrapsRunnerErrorsConsistently(t *testing.T) {
	t.Parallel()

	underlying := &process.ExecError{Kind: process.ErrorKindExitCode, Command: "rojo serve", ExitCode: 1}
	runner := &fakeRunner{result: process.Result{ExitCode: 1}, err: underlying}
	rojo := NewRojo(runner, "")

	_, err := rojo.Serve(context.Background(), []string{"default.project.json"}, RunOptions{})
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if !strings.Contains(err.Error(), "rojo command \"rojo serve default.project.json\" failed") {
		t.Fatalf("expected standardized wrapper prefix, got: %v", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("expected wrapper to preserve underlying error, got: %v", err)
	}
}

func assertArgs(t *testing.T, got []string, expected []string) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected args %#v, got %#v", expected, got)
	}
}
