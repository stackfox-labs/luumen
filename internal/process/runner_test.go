package process

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestRunCommandSuccess(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	var stderr strings.Builder

	result, err := Run(context.Background(), successCommandSpec(), Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(stdout.String(), "process-ok") {
		t.Fatalf("expected streamed stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "process-err") {
		t.Fatalf("expected streamed stderr output, got %q", stderr.String())
	}
}

func TestRunCommandFailureExitCode(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), failureCommandSpec(7), Options{})
	if err == nil {
		t.Fatal("expected an error for failing command")
	}
	if result.ExitCode != 7 {
		t.Fatalf("expected exit code 7, got %d", result.ExitCode)
	}
	if !IsKind(err, ErrorKindExitCode) {
		t.Fatalf("expected exit-code error kind, got: %v", err)
	}
}

func TestRunMissingExecutable(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Command{Name: "__luumen_missing_binary__"}, Options{})
	if err == nil {
		t.Fatal("expected error for missing executable")
	}
	if result.ExitCode != -1 {
		t.Fatalf("expected -1 exit code, got %d", result.ExitCode)
	}
	if !IsKind(err, ErrorKindNotFound) {
		t.Fatalf("expected not-found error kind, got: %v", err)
	}
}

func TestRunInvalidWorkingDirectory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist")

	result, err := Run(context.Background(), successCommandSpec(), Options{WorkingDir: missing})
	if err == nil {
		t.Fatal("expected invalid working directory error")
	}
	if result.ExitCode != -1 {
		t.Fatalf("expected -1 exit code, got %d", result.ExitCode)
	}
	if !IsKind(err, ErrorKindInvalidWorkingDir) {
		t.Fatalf("expected invalid working directory error kind, got: %v", err)
	}
}

func TestRunInterrupted(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := Run(ctx, successCommandSpec(), Options{})
	if err == nil {
		t.Fatal("expected interrupted error")
	}
	if result.ExitCode != -1 {
		t.Fatalf("expected -1 exit code, got %d", result.ExitCode)
	}
	if !IsKind(err, ErrorKindInterrupted) {
		t.Fatalf("expected interrupted error kind, got: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation to be wrapped, got: %v", err)
	}
}

func TestRunShellCommand(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	result, err := RunShell(context.Background(), shellEchoCommand(), Options{Stdout: &stdout})
	if err != nil {
		t.Fatalf("expected shell command success, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected shell exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(stdout.String(), "shell-ok") {
		t.Fatalf("expected shell output to include shell-ok, got %q", stdout.String())
	}
}

func TestRunCommandWithEnvironmentOverride(t *testing.T) {
	t.Parallel()

	var stdout strings.Builder
	result, err := Run(context.Background(), envEchoCommandSpec("LUUMEN_PROCESS_TEST_VAR"), Options{
		Stdout: &stdout,
		Env: map[string]string{
			"LUUMEN_PROCESS_TEST_VAR": "env-ok",
		},
	})
	if err != nil {
		t.Fatalf("expected env command success, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(stdout.String(), "env-ok") {
		t.Fatalf("expected env output to contain env-ok, got %q", stdout.String())
	}
}

func successCommandSpec() Command {
	if runtime.GOOS == "windows" {
		return Command{Name: "cmd", Args: []string{"/C", "echo process-ok & echo process-err 1>&2"}}
	}
	return Command{Name: "sh", Args: []string{"-c", "echo process-ok; echo process-err 1>&2"}}
}

func failureCommandSpec(code int) Command {
	if runtime.GOOS == "windows" {
		return Command{Name: "cmd", Args: []string{"/C", "exit " + intToString(code)}}
	}
	return Command{Name: "sh", Args: []string{"-c", "exit " + intToString(code)}}
}

func envEchoCommandSpec(name string) Command {
	if runtime.GOOS == "windows" {
		return Command{Name: "cmd", Args: []string{"/C", "echo %" + name + "%"}}
	}
	return Command{Name: "sh", Args: []string{"-c", "printf \"%s\" \"$" + name + "\""}}
}

func shellEchoCommand() string {
	if runtime.GOOS == "windows" {
		return "echo shell-ok"
	}
	return "echo shell-ok"
}

func intToString(value int) string {
	return strconv.Itoa(value)
}
