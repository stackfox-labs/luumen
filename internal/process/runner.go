package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type ErrorKind string

const (
	ErrorKindNotFound          ErrorKind = "not_found"
	ErrorKindExitCode          ErrorKind = "exit_code"
	ErrorKindInvalidWorkingDir ErrorKind = "invalid_working_dir"
	ErrorKindInterrupted       ErrorKind = "interrupted"
	ErrorKindInvalidCommand    ErrorKind = "invalid_command"
)

type ExecError struct {
	Kind       ErrorKind
	Command    string
	WorkingDir string
	ExitCode   int
	Err        error
}

func (e *ExecError) Error() string {
	switch e.Kind {
	case ErrorKindNotFound:
		return fmt.Sprintf("executable for %q was not found in PATH", e.Command)
	case ErrorKindExitCode:
		return fmt.Sprintf("command %q exited with code %d", e.Command, e.ExitCode)
	case ErrorKindInvalidWorkingDir:
		return fmt.Sprintf("working directory %q is invalid", e.WorkingDir)
	case ErrorKindInterrupted:
		return fmt.Sprintf("command %q was interrupted", e.Command)
	case ErrorKindInvalidCommand:
		return "command name is required"
	default:
		return fmt.Sprintf("failed to run command %q", e.Command)
	}
}

func (e *ExecError) Unwrap() error {
	return e.Err
}

func IsKind(err error, kind ErrorKind) bool {
	var execErr *ExecError
	if !errors.As(err, &execErr) {
		return false
	}
	return execErr.Kind == kind
}

type Command struct {
	Name string
	Args []string
}

func (c Command) String() string {
	if len(c.Args) == 0 {
		return c.Name
	}
	return strings.TrimSpace(c.Name + " " + strings.Join(c.Args, " "))
}

type Options struct {
	WorkingDir string
	Env        map[string]string
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
}

type Result struct {
	ExitCode int
}

func Run(ctx context.Context, command Command, options Options) (Result, error) {
	if strings.TrimSpace(command.Name) == "" {
		return Result{ExitCode: -1}, &ExecError{Kind: ErrorKindInvalidCommand}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return Result{ExitCode: -1}, &ExecError{
			Kind:    ErrorKindInterrupted,
			Command: command.String(),
			Err:     err,
		}
	}

	if err := validateWorkingDirectory(options.WorkingDir); err != nil {
		return Result{ExitCode: -1}, &ExecError{
			Kind:       ErrorKindInvalidWorkingDir,
			WorkingDir: options.WorkingDir,
			Command:    command.String(),
			Err:        err,
		}
	}

	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = options.WorkingDir
	cmd.Env = mergedEnvironment(options.Env)
	cmd.Stdin = options.Stdin
	cmd.Stdout = options.Stdout
	cmd.Stderr = options.Stderr
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	err := cmd.Run()
	if err == nil {
		return Result{ExitCode: 0}, nil
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
		return Result{ExitCode: -1}, &ExecError{
			Kind:    ErrorKindInterrupted,
			Command: command.String(),
			Err:     err,
		}
	}

	if executableNotFound(err) {
		return Result{ExitCode: -1}, &ExecError{
			Kind:    ErrorKindNotFound,
			Command: command.String(),
			Err:     err,
		}
	}

	exitCode := extractExitCode(err)
	if exitCode >= 0 {
		return Result{ExitCode: exitCode}, &ExecError{
			Kind:     ErrorKindExitCode,
			Command:  command.String(),
			ExitCode: exitCode,
			Err:      err,
		}
	}

	return Result{ExitCode: -1}, fmt.Errorf("failed to run command %q: %w", command.String(), err)
}

func RunShell(ctx context.Context, command string, options Options) (Result, error) {
	if strings.TrimSpace(command) == "" {
		return Result{ExitCode: -1}, &ExecError{Kind: ErrorKindInvalidCommand}
	}

	shellCommand := buildShellCommand(command)
	result, err := Run(ctx, shellCommand, options)
	if err != nil {
		return result, fmt.Errorf("shell command %q failed: %w", command, err)
	}

	return result, nil
}

func buildShellCommand(command string) Command {
	if runtime.GOOS == "windows" {
		return Command{Name: "cmd", Args: []string{"/C", command}}
	}
	return Command{Name: "sh", Args: []string{"-c", command}}
}

func validateWorkingDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(resolvedPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}

	return nil
}

func mergedEnvironment(overrides map[string]string) []string {
	base := os.Environ()
	if len(overrides) == 0 {
		return base
	}

	values := make(map[string]string, len(base)+len(overrides))
	for _, item := range base {
		parts := strings.SplitN(item, "=", 2)
		key := normalizeEnvKey(parts[0])
		if len(parts) == 2 {
			values[key] = parts[1]
			continue
		}
		values[key] = ""
	}

	for key, value := range overrides {
		values[normalizeEnvKey(key)] = value
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	merged := make([]string, 0, len(values))
	for _, key := range keys {
		merged = append(merged, key+"="+values[key])
	}
	return merged
}

func normalizeEnvKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

func executableNotFound(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}

	var execError *exec.Error
	if errors.As(err, &execError) {
		return errors.Is(execError.Err, exec.ErrNotFound)
	}

	return false
}

func extractExitCode(err error) int {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
