package tools

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luumen/internal/process"
)

type sequenceRunner struct {
	calls []runCall
	step  int
}

func (s *sequenceRunner) Run(_ context.Context, command process.Command, options process.Options) (process.Result, error) {
	s.calls = append(s.calls, runCall{command: command, options: options})
	if s.step == 0 {
		s.step++
		return process.Result{ExitCode: 1}, &process.ExecError{Kind: process.ErrorKindExitCode, Command: command.String(), ExitCode: 1}
	}
	return process.Result{ExitCode: 0}, nil
}

func TestRokitInstallBuildsExpectedCommand(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	rokit := NewRokit(runner, "")
	root := t.TempDir()

	var loggerTool string
	var loggerCommand string

	result, err := rokit.Install(context.Background(), RunOptions{
		WorkingDir: root,
		Env: map[string]string{
			"LUUMEN_TEST": "true",
		},
		Logger: func(toolName string, command process.Command) {
			loggerTool = toolName
			loggerCommand = command.String()
		},
	})
	if err != nil {
		t.Fatalf("expected install success, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one command run, got %d", len(runner.calls))
	}

	call := runner.calls[0]
	if call.command.Name != DefaultRokitExecutable {
		t.Fatalf("expected executable %q, got %q", DefaultRokitExecutable, call.command.Name)
	}
	if len(call.command.Args) != 1 || call.command.Args[0] != "install" {
		t.Fatalf("expected rokit install args, got %#v", call.command.Args)
	}
	if call.options.WorkingDir != root {
		t.Fatalf("expected working dir %q, got %q", root, call.options.WorkingDir)
	}
	if loggerTool != "rokit" {
		t.Fatalf("expected logger tool to be rokit, got %q", loggerTool)
	}
	if loggerCommand != "rokit install" {
		t.Fatalf("expected logger command rokit install, got %q", loggerCommand)
	}
}

func TestRokitSyncUsesInstallInvocation(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	rokit := NewRokit(runner, "")

	if _, err := rokit.Sync(context.Background(), RunOptions{}); err != nil {
		t.Fatalf("expected sync success, got: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one run call, got %d", len(runner.calls))
	}
	if runner.calls[0].command.String() != "rokit install" {
		t.Fatalf("expected sync to use rokit install, got %q", runner.calls[0].command.String())
	}
}

func TestRokitHasConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	rokit := NewRokit(nil, "")

	hasConfig, err := rokit.HasConfig(root)
	if err != nil {
		t.Fatalf("expected no error on missing config, got: %v", err)
	}
	if hasConfig {
		t.Fatal("expected missing rokit.toml to report false")
	}

	path := filepath.Join(root, RokitConfigFile)
	if writeErr := os.WriteFile(path, []byte("# rokit\n"), 0o644); writeErr != nil {
		t.Fatalf("failed to write rokit.toml: %v", writeErr)
	}

	hasConfig, err = rokit.HasConfig(root)
	if err != nil {
		t.Fatalf("expected no error when rokit.toml exists, got: %v", err)
	}
	if !hasConfig {
		t.Fatal("expected rokit.toml to report true")
	}
}

func TestRokitWrapsRunnerErrorsConsistently(t *testing.T) {
	t.Parallel()

	underlying := &process.ExecError{Kind: process.ErrorKindExitCode, Command: "rokit install", ExitCode: 1}
	runner := &fakeRunner{result: process.Result{ExitCode: 1}, err: underlying}
	rokit := NewRokit(runner, "")

	_, err := rokit.Install(context.Background(), RunOptions{})
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if !strings.Contains(err.Error(), "rokit command \"rokit install\" failed") {
		t.Fatalf("expected standardized wrapper prefix, got: %v", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("expected wrapper to preserve underlying error, got: %v", err)
	}
}

func TestRokitInstallRetriesWithTerminalIOOnExitCodeFailure(t *testing.T) {
	t.Parallel()

	originalReaderIsTerminal := readerIsTerminal
	readerIsTerminal = func(_ io.Reader) bool { return true }
	defer func() { readerIsTerminal = originalReaderIsTerminal }()

	runner := &sequenceRunner{}
	rokit := NewRokit(runner, "")

	result, err := rokit.Install(context.Background(), RunOptions{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Stdin:  os.Stdin,
	})
	if err != nil {
		t.Fatalf("expected retry path to succeed, got: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected successful retry exit code, got: %d", result.ExitCode)
	}

	if len(runner.calls) != 2 {
		t.Fatalf("expected two run attempts, got %d", len(runner.calls))
	}
	if runner.calls[0].options.Stdout != io.Discard || runner.calls[0].options.Stderr != io.Discard {
		t.Fatalf("expected first attempt to use quiet outputs")
	}
	if runner.calls[1].options.Stdout != os.Stdout || runner.calls[1].options.Stderr != os.Stderr {
		t.Fatalf("expected second attempt to use terminal outputs")
	}
}

func TestRokitInstallUsesNoTrustCheckInCI(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	rokit := NewRokit(runner, "")

	if _, err := rokit.Install(context.Background(), RunOptions{Env: map[string]string{"CI": "true"}}); err != nil {
		t.Fatalf("expected install success, got: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("expected one run call, got %d", len(runner.calls))
	}

	got := runner.calls[0].command.Args
	expected := []string{"install", "--no-trust-check"}
	if len(got) != len(expected) || got[0] != expected[0] || got[1] != expected[1] {
		t.Fatalf("expected args %#v, got %#v", expected, got)
	}
}

func TestRokitInstallUsesNoTrustCheckWhenExplicitlyEnabled(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	rokit := NewRokit(runner, "")

	if _, err := rokit.Install(context.Background(), RunOptions{Env: map[string]string{"LUU_ROKIT_NO_TRUST_CHECK": "1"}}); err != nil {
		t.Fatalf("expected install success, got: %v", err)
	}

	if len(runner.calls) != 1 {
		t.Fatalf("expected one run call, got %d", len(runner.calls))
	}

	got := runner.calls[0].command.Args
	expected := []string{"install", "--no-trust-check"}
	if len(got) != len(expected) || got[0] != expected[0] || got[1] != expected[1] {
		t.Fatalf("expected args %#v, got %#v", expected, got)
	}
}
