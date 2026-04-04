package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"luumen/internal/process"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type fakeRecoveryRokit struct {
	installCalls int
	addCalls     int
	installErr   error
	addErr       error
}

func (f *fakeRecoveryRokit) Install(_ context.Context, _ tools.RunOptions) (process.Result, error) {
	f.installCalls++
	if f.installErr != nil {
		return process.Result{ExitCode: 1}, f.installErr
	}
	return process.Result{ExitCode: 0}, nil
}

func (f *fakeRecoveryRokit) Add(_ context.Context, tool string, _ string, options tools.RunOptions) (process.Result, error) {
	f.addCalls++
	if f.addErr != nil {
		return process.Result{ExitCode: 1}, f.addErr
	}
	path := filepath.Join(options.WorkingDir, workspace.RokitConfigFile)
	if _, err := addToolToRokitConfig(path, tool); err != nil {
		return process.Result{ExitCode: 1}, err
	}
	return process.Result{ExitCode: 0}, nil
}

func TestEnsureExecutableAddsAndInstallsKnownUndeclaredTool(t *testing.T) {
	repo := t.TempDir()
	binDir := filepath.Join(repo, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	writeFakeExecutable(t, binDir, "rokit")
	writeFakeExecutable(t, binDir, "selene")

	pathSep := ":"
	if runtime.GOOS == "windows" {
		pathSep = ";"
	}
	t.Setenv("PATH", binDir+pathSep+os.Getenv("PATH"))

	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	if err := os.WriteFile(rokitPath, []byte("[tools]\nrojo = \"rojo-rbx/rojo@7.6.1\"\n"), 0o644); err != nil {
		t.Fatalf("failed to write rokit config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(withYesMode(context.Background(), true))
	installer := &fakeRecoveryRokit{}
	runner := newSelfHealingShellRunner(cmd, "lint", workspace.Workspace{
		RootPath:        repo,
		HasRokitConfig:  true,
		RokitConfigPath: rokitPath,
	}, installer)

	_, err := runner.ensureExecutable(context.Background(), "selene", process.Options{
		WorkingDir: repo,
		Stdout:     io.Discard,
		Stderr:     io.Discard,
	})
	if err != nil {
		t.Fatalf("expected undeclared known tool recovery success, got: %v", err)
	}

	if installer.addCalls != 1 {
		t.Fatalf("expected one rokit add call, got %d", installer.addCalls)
	}
	if installer.installCalls != 0 {
		t.Fatalf("expected no rokit install call for undeclared tool, got %d", installer.installCalls)
	}

	contents, err := os.ReadFile(rokitPath)
	if err != nil {
		t.Fatalf("failed to read rokit config: %v", err)
	}
	text := strings.ToLower(string(contents))
	if !strings.Contains(text, "selene =") {
		t.Fatalf("expected rokit config to include selene after recovery, got: %s", string(contents))
	}
}

func TestEnsureExecutableRepairsLegacyToolKey(t *testing.T) {
	repo := t.TempDir()
	binDir := filepath.Join(repo, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	writeFakeExecutable(t, binDir, "rokit")
	writeFakeExecutable(t, binDir, "selene")

	pathSep := ":"
	if runtime.GOOS == "windows" {
		pathSep = ";"
	}
	t.Setenv("PATH", binDir+pathSep+os.Getenv("PATH"))

	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	legacy := "[tools]\nselene-0-30-1 = \"Kampfkarren/selene@0.30.1\"\n"
	if err := os.WriteFile(rokitPath, []byte(legacy), 0o644); err != nil {
		t.Fatalf("failed to write rokit config: %v", err)
	}

	cmd := &cobra.Command{}
	cmd.SetContext(withYesMode(context.Background(), true))
	installer := &fakeRecoveryRokit{}
	runner := newSelfHealingShellRunner(cmd, "lint", workspace.Workspace{
		RootPath:        repo,
		HasRokitConfig:  true,
		RokitConfigPath: rokitPath,
	}, installer)

	_, err := runner.ensureExecutable(context.Background(), "selene", process.Options{WorkingDir: repo, Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		t.Fatalf("expected recovery success, got: %v", err)
	}

	contents, err := os.ReadFile(rokitPath)
	if err != nil {
		t.Fatalf("failed to read rokit config: %v", err)
	}
	text := strings.ToLower(string(contents))
	if !strings.Contains(text, "selene =") || strings.Contains(text, "selene-0-30-1") {
		t.Fatalf("expected legacy key to be repaired, got: %s", string(contents))
	}
}

func TestRunShellPrintsInstallAndRetryCommands(t *testing.T) {
	repo := t.TempDir()
	binDir := filepath.Join(repo, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	writeFakeExecutable(t, binDir, "rokit")
	writeFakeExecutable(t, binDir, "selene")

	pathSep := ":"
	if runtime.GOOS == "windows" {
		pathSep = ";"
	}
	t.Setenv("PATH", binDir+pathSep+os.Getenv("PATH"))

	rokitPath := filepath.Join(repo, workspace.RokitConfigFile)
	if err := os.WriteFile(rokitPath, []byte("[tools]\nrojo = \"rojo-rbx/rojo@7.6.1\"\n"), 0o644); err != nil {
		t.Fatalf("failed to write rokit config: %v", err)
	}

	output := bytes.NewBuffer(nil)
	cmd := &cobra.Command{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetContext(withYesMode(context.Background(), true))
	installer := &fakeRecoveryRokit{}

	runner := newSelfHealingShellRunner(cmd, "lint", workspace.Workspace{
		RootPath:        repo,
		HasRokitConfig:  true,
		RokitConfigPath: rokitPath,
	}, installer)

	if _, err := runner.RunShell(context.Background(), "selene src", process.Options{WorkingDir: repo, Stdout: io.Discard, Stderr: io.Discard}); err != nil {
		t.Fatalf("expected run shell success, got: %v", err)
	}

	text := strings.ToLower(output.String())
	if !strings.Contains(text, "running: rokit add") {
		t.Fatalf("expected add command output, got: %q", output.String())
	}
	if strings.Contains(text, "rokit add kampfkarren/selene selene") {
		t.Fatalf("expected add output to omit redundant alias, got: %q", output.String())
	}
	if !strings.Contains(text, "running: selene src") {
		t.Fatalf("expected retry command output, got: %q", output.String())
	}
	if installer.addCalls != 1 {
		t.Fatalf("expected one add call, got %d", installer.addCalls)
	}
}

func writeFakeExecutable(t *testing.T, dir string, name string) {
	t.Helper()

	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".cmd")
		content := "@echo off\r\nexit /b 0\r\n"
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatalf("failed to write fake executable %s: %v", path, err)
		}
		return
	}

	path := filepath.Join(dir, name)
	content := "#!/usr/bin/env sh\nexit 0\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("failed to write fake executable %s: %v", path, err)
	}
}
