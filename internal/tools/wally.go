package tools

import (
	"context"
	"fmt"
	"strings"

	"luumen/internal/process"
)

const (
	DefaultWallyExecutable = "wally"
	WallyConfigFile        = "wally.toml"
)

type Wally struct {
	runner     CommandRunner
	executable string
}

func NewWally(runner CommandRunner, executable string) *Wally {
	if strings.TrimSpace(executable) == "" {
		executable = DefaultWallyExecutable
	}

	return &Wally{
		runner:     withDefaultRunner(runner),
		executable: executable,
	}
}

func (w *Wally) HasConfig(rootPath string) (bool, error) {
	return hasConfigFile(rootPath, WallyConfigFile)
}

func (w *Wally) Install(ctx context.Context, options RunOptions) (process.Result, error) {
	command := process.Command{Name: w.executable, Args: []string{"install"}}
	return executeTool(ctx, w.runner, "wally", command, options)
}

func (w *Wally) AddPackage(ctx context.Context, packageRef string, options RunOptions) (process.Result, error) {
	if strings.TrimSpace(packageRef) == "" {
		return process.Result{ExitCode: -1}, fmt.Errorf("package reference is required")
	}

	command := process.Command{Name: w.executable, Args: []string{"add", packageRef}}
	return executeTool(ctx, w.runner, "wally", command, options)
}
