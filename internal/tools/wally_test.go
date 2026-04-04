package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"luumen/internal/process"
)

func TestWallyInstallBuildsExpectedCommand(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	wally := NewWally(runner, "")

	if _, err := wally.Install(context.Background(), RunOptions{}); err != nil {
		t.Fatalf("expected install success, got: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one run call, got %d", len(runner.calls))
	}
	if runner.calls[0].command.String() != "wally install" {
		t.Fatalf("expected wally install, got %q", runner.calls[0].command.String())
	}
}

func TestWallyAddPackageBuildsExpectedCommand(t *testing.T) {
	t.Parallel()

	runner := &fakeRunner{}
	wally := NewWally(runner, "")

	if _, err := wally.AddPackage(context.Background(), "sleitnick/knit", RunOptions{}); err != nil {
		t.Fatalf("expected add package success, got: %v", err)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected one run call, got %d", len(runner.calls))
	}
	if runner.calls[0].command.String() != "wally add sleitnick/knit" {
		t.Fatalf("expected wally add command, got %q", runner.calls[0].command.String())
	}
}

func TestWallyAddPackageRequiresReference(t *testing.T) {
	t.Parallel()

	wally := NewWally(&fakeRunner{}, "")
	_, err := wally.AddPackage(context.Background(), " ", RunOptions{})
	if err == nil {
		t.Fatal("expected validation error for empty package reference")
	}
}

func TestWallyHasConfig(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	wally := NewWally(nil, "")

	hasConfig, err := wally.HasConfig(root)
	if err != nil {
		t.Fatalf("expected no error on missing config, got: %v", err)
	}
	if hasConfig {
		t.Fatal("expected missing wally.toml to report false")
	}

	path := filepath.Join(root, WallyConfigFile)
	if writeErr := os.WriteFile(path, []byte("# wally\n"), 0o644); writeErr != nil {
		t.Fatalf("failed to write wally.toml: %v", writeErr)
	}

	hasConfig, err = wally.HasConfig(root)
	if err != nil {
		t.Fatalf("expected no error when wally.toml exists, got: %v", err)
	}
	if !hasConfig {
		t.Fatal("expected wally.toml to report true")
	}
}

func TestWallyWrapsRunnerErrorsConsistently(t *testing.T) {
	t.Parallel()

	underlying := &process.ExecError{Kind: process.ErrorKindExitCode, Command: "wally install", ExitCode: 1}
	runner := &fakeRunner{result: process.Result{ExitCode: 1}, err: underlying}
	wally := NewWally(runner, "")

	_, err := wally.Install(context.Background(), RunOptions{})
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if !strings.Contains(err.Error(), "wally command \"wally install\" failed") {
		t.Fatalf("expected standardized wrapper prefix, got: %v", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("expected wrapper to preserve underlying error, got: %v", err)
	}
}
