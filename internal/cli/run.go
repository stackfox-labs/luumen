package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"luumen/internal/config"
	"luumen/internal/tasks"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type taskRunner interface {
	RunNamedTask(ctx context.Context, taskName string, cfg *config.Config, options tasks.RunOptions) error
}

type runCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	loadConfig      func(path string) (*config.Config, error)
	taskRunner      taskRunner
}

func defaultRunCommandDeps() runCommandDeps {
	return runCommandDeps{
		detectWorkspace: workspace.Detect,
		loadConfig:      config.LoadTasks,
		taskRunner:      nil,
	}
}

func newRunCmd(deps runCommandDeps) *cobra.Command {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.LoadTasks
	}
	cmd := &cobra.Command{
		Use:   "run <task>",
		Short: "Run a named Luumen task",
		Long:  "Run executes a task from tasks in project.config.luau. Task values can be a string or an array of strings.",
		Example: "luu run test\n" +
			"luu run ci --quiet",
		Args: requireExactlyOneArg("task"),
		RunE: func(cmd *cobra.Command, args []string) error {
			statusf(cmd, "Running task: %s", args[0])

			state, err := deps.detectWorkspace("")
			if err != nil {
				return fmt.Errorf("failed to detect workspace: %w. Next: run the command from a repository directory", err)
			}
			if !state.HasLuumenConfig {
				return fmt.Errorf("cannot run task: %s was not found in %s. Next: create project.config.luau or run luu init", workspace.LuumenConfigFile, state.RootPath)
			}

			cfg, err := deps.loadConfig(state.LuumenConfigPath)
			if err != nil {
				return fmt.Errorf("failed to load %s: %w", workspace.LuumenConfigFile, err)
			}

			runner := deps.taskRunner
			if runner == nil {
				runner = tasks.NewEngine(newSelfHealingShellRunner(cmd, "run", state, tools.NewRokit(nil, "")), "luu")
			}

			stdout, stderr := commandOutputWriters(cmd)

			if err := runner.RunNamedTask(cmd.Context(), args[0], cfg, tasks.RunOptions{
				WorkingDir: state.RootPath,
				Stdout:     stdout,
				Stderr:     stderr,
				Stdin:      cmd.InOrStdin(),
			}); err != nil {
				return fmt.Errorf("failed to run task %q: %w", args[0], err)
			}

			statusf(cmd, "Task completed: %s", args[0])
			return nil
		},
	}

	return cmd
}
