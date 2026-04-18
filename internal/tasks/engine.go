package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"luumen/internal/config"
	"luumen/internal/process"
)

var (
	ErrTaskNotFound = errors.New("task not found")
	ErrTaskCycle    = errors.New("task cycle detected")
)

type ShellRunner interface {
	RunShell(ctx context.Context, command string, options process.Options) (process.Result, error)
}

type RunOptions struct {
	WorkingDir string
	Env        map[string]string
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
}

type Engine struct {
	shellRunner ShellRunner
	cliName     string
}

type processShellRunner struct{}

func (processShellRunner) RunShell(ctx context.Context, command string, options process.Options) (process.Result, error) {
	return process.RunShell(ctx, command, options)
}

func NewEngine(shellRunner ShellRunner, cliName string) *Engine {
	if shellRunner == nil {
		shellRunner = processShellRunner{}
	}
	if strings.TrimSpace(cliName) == "" {
		cliName = "luu"
	}
	return &Engine{shellRunner: shellRunner, cliName: strings.ToLower(strings.TrimSpace(cliName))}
}

func (e *Engine) RunNamedTask(ctx context.Context, taskName string, cfg *config.Config, options RunOptions) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	if strings.TrimSpace(taskName) == "" {
		return errors.New("task name is required")
	}
	if len(cfg.Tasks) == 0 {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskName)
	}

	return e.runTask(ctx, cfg, strings.TrimSpace(taskName), options, nil)
}

func (e *Engine) runTask(ctx context.Context, cfg *config.Config, taskName string, options RunOptions, stack []string) error {
	for _, current := range stack {
		if current == taskName {
			cycle := append(append([]string(nil), stack...), taskName)
			return fmt.Errorf("%w: %s", ErrTaskCycle, strings.Join(cycle, " -> "))
		}
	}

	taskValue, ok := cfg.Tasks[taskName]
	if !ok {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, taskName)
	}

	plan, err := NormalizeTaskValue(taskValue)
	if err != nil {
		return fmt.Errorf("task %q is invalid: %w", taskName, err)
	}

	nextStack := append(append([]string(nil), stack...), taskName)
	for _, step := range plan.Steps {
		nestedTask, isNested := parseNestedRunCommand(step, e.cliName)
		if isNested {
			if err := e.runTask(ctx, cfg, nestedTask, options, nextStack); err != nil {
				return err
			}
			continue
		}

		if _, err := e.shellRunner.RunShell(ctx, step, process.Options{
			WorkingDir: options.WorkingDir,
			Env:        options.Env,
			Stdout:     options.Stdout,
			Stderr:     options.Stderr,
			Stdin:      options.Stdin,
		}); err != nil {
			return fmt.Errorf("task %q step %q failed: %w", taskName, step, err)
		}
	}

	return nil
}

func parseNestedRunCommand(command string, cliName string) (string, bool) {
	fields := strings.Fields(command)
	if len(fields) != 3 {
		return "", false
	}

	name := strings.ToLower(strings.TrimSpace(fields[0]))
	name = strings.TrimPrefix(name, "./")
	name = strings.TrimPrefix(name, ".\\")
	name = strings.TrimSuffix(name, ".exe")
	if name != cliName {
		return "", false
	}
	if strings.ToLower(strings.TrimSpace(fields[1])) != "run" {
		return "", false
	}
	nested := strings.TrimSpace(fields[2])
	if nested == "" {
		return "", false
	}
	return nested, true
}
