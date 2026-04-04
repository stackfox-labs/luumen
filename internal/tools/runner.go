package tools

import (
	"context"
	"fmt"
	"io"

	"luumen/internal/process"
)

type CommandRunner interface {
	Run(ctx context.Context, command process.Command, options process.Options) (process.Result, error)
}

type InvocationLogger func(toolName string, command process.Command)

type RunOptions struct {
	WorkingDir string
	Env        map[string]string
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	Logger     InvocationLogger
}

type processRunner struct{}

func (processRunner) Run(ctx context.Context, command process.Command, options process.Options) (process.Result, error) {
	return process.Run(ctx, command, options)
}

func withDefaultRunner(runner CommandRunner) CommandRunner {
	if runner != nil {
		return runner
	}
	return processRunner{}
}

func executeTool(ctx context.Context, runner CommandRunner, toolName string, command process.Command, options RunOptions) (process.Result, error) {
	if options.Logger != nil {
		options.Logger(toolName, command)
	}

	result, err := runner.Run(ctx, command, process.Options{
		WorkingDir: options.WorkingDir,
		Env:        options.Env,
		Stdout:     options.Stdout,
		Stderr:     options.Stderr,
		Stdin:      options.Stdin,
	})
	if err != nil {
		return result, fmt.Errorf("%s command %q failed: %w", toolName, command.String(), err)
	}

	return result, nil
}
