package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"luumen/internal/config"
	"luumen/internal/process"
	"luumen/internal/tasks"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type workflowCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	loadConfig      func(path string) (*config.Config, error)
	taskRunner      taskRunner
	rokitInstaller  rokitInstaller
}

func defaultWorkflowCommandDeps() workflowCommandDeps {
	return workflowCommandDeps{
		detectWorkspace: workspace.Detect,
		loadConfig:      config.LoadTasks,
		rokitInstaller:  tools.NewRokit(nil, ""),
	}
}

func ensureWorkflowDeps(deps workflowCommandDeps) workflowCommandDeps {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.LoadTasks
	}
	if deps.rokitInstaller == nil {
		deps.rokitInstaller = tools.NewRokit(nil, "")
	}
	return deps
}

func newDevCmd(deps workflowCommandDeps) *cobra.Command {
	return newBuiltInWorkflowCmd("dev", "Run the main development workflow", "luu dev", deps)
}

func newBuildCmd(deps workflowCommandDeps) *cobra.Command {
	return newBuiltInWorkflowCmd("build", "Produce a build output", "luu build", deps)
}

func newLintCmd(deps workflowCommandDeps) *cobra.Command {
	return newBuiltInWorkflowCmd("lint", "Run static analysis", "luu lint", deps)
}

func newFormatCmd(deps workflowCommandDeps) *cobra.Command {
	return newBuiltInWorkflowCmd("format", "Format code", "luu format", deps)
}

func newTestCmd(deps workflowCommandDeps) *cobra.Command {
	return newBuiltInWorkflowCmd("test", "Run tests", "luu test", deps)
}

func newBuiltInWorkflowCmd(commandName string, short string, example string, deps workflowCommandDeps) *cobra.Command {
	deps = ensureWorkflowDeps(deps)

	cmd := &cobra.Command{
		Use:     commandName,
		Short:   short,
		Example: example,
		Args:    requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			state, cfg, err := loadWorkflowContext(deps)
			if err != nil {
				return err
			}

			taskName, taskCfg, steps, resolveErr := resolveBuiltInTask(commandName, cfg, state)
			if len(steps) > 0 {
				printWorkflowPlan(cmd, workflowWorkspaceName(state, cfg), taskName, steps)
			}
			if resolveErr != nil {
				return resolveErr
			}

			runner := workflowTaskRunner(cmd, deps, commandName, state, steps)
			if err := runner.RunNamedTask(cmd.Context(), taskName, taskCfg, workflowRunOptions(cmd, state.RootPath)); err != nil {
				return fmt.Errorf("task %q failed: %w", taskName, err)
			}
			return nil
		},
	}

	return cmd
}

func workflowTaskRunner(cmd *cobra.Command, deps workflowCommandDeps, commandName string, state workspace.Workspace, steps []string) taskRunner {
	if deps.taskRunner != nil {
		return deps.taskRunner
	}

	baseRunner := newSelfHealingShellRunner(cmd, commandName, state, deps.rokitInstaller)
	shellRunner := newWorkflowStepShellRunner(cmd, baseRunner, len(steps))
	return tasks.NewEngine(shellRunner, "luu")
}

type workflowStepShellRunner struct {
	cmd        *cobra.Command
	inner      tasks.ShellRunner
	totalSteps int
	stepIndex  int
}

func newWorkflowStepShellRunner(cmd *cobra.Command, inner tasks.ShellRunner, totalSteps int) *workflowStepShellRunner {
	if totalSteps < 0 {
		totalSteps = 0
	}
	return &workflowStepShellRunner{cmd: cmd, inner: inner, totalSteps: totalSteps}
}

func (r *workflowStepShellRunner) RunShell(ctx context.Context, command string, options process.Options) (process.Result, error) {
	if r.cmd != nil && !isQuiet(r.cmd) && r.totalSteps > 1 {
		writer := r.cmd.OutOrStdout()
		prefix := styleAccent(writer, "[luu]")
		currentStep := r.stepIndex + 1
		if currentStep > r.totalSteps {
			currentStep = r.totalSteps
		}
		fmt.Fprintf(
			writer,
			"%s %s %d/%d: %s\n\n",
			prefix,
			styleMuted(writer, "step"),
			currentStep,
			r.totalSteps,
			styleCommand(writer, shellStyleCommand(command)),
		)
	}

	r.stepIndex++
	return r.inner.RunShell(ctx, command, options)
}

func workflowRunOptions(cmd *cobra.Command, workingDir string) tasks.RunOptions {
	stdout, stderr := commandOutputWriters(cmd)
	return tasks.RunOptions{
		WorkingDir: workingDir,
		Stdout:     stdout,
		Stderr:     stderr,
		Stdin:      cmd.InOrStdin(),
	}
}

func resolveBuiltInTask(commandName string, cfg *config.Config, state workspace.Workspace) (string, *config.Config, []string, error) {
	if cfg != nil {
		if taskValue, ok := cfg.Tasks[commandName]; ok {
			return commandName, cfg, append([]string(nil), taskValue.Steps...), nil
		}
	}

	steps, err := resolveBuiltInTaskFallback(commandName, state)
	if err != nil {
		return commandName, nil, steps, err
	}

	return commandName, &config.Config{
		Tasks: map[string]config.TaskValue{
			commandName: config.NewTaskValue(steps...),
		},
	}, steps, nil
}

func resolveBuiltInTaskFallback(commandName string, state workspace.Workspace) ([]string, error) {
	switch commandName {
	case "dev":
		projectPath, usedFallback, err := resolveDefaultRojoProjectPathForPlan(state)
		if err != nil {
			return nil, err
		}
		steps := []string{
			fmt.Sprintf("rojo sourcemap %s --output sourcemap.json", projectPath),
			fmt.Sprintf("rojo serve %s", projectPath),
		}
		if usedFallback {
			return steps, missingDefaultRojoProjectError(commandName, state)
		}
		return steps, nil
	case "build":
		projectPath, usedFallback, err := resolveDefaultRojoProjectPathForPlan(state)
		if err != nil {
			return nil, err
		}
		steps := []string{fmt.Sprintf("rojo build %s --output build.rbxl", projectPath)}
		if usedFallback {
			return steps, missingDefaultRojoProjectError(commandName, state)
		}
		return steps, nil
	case "lint", "format", "test":
		return nil, missingWorkflowTaskError(commandName)
	default:
		return nil, missingWorkflowTaskError(commandName)
	}
}

func printWorkflowPlan(cmd *cobra.Command, workspaceName string, taskName string, steps []string) {
	if isQuiet(cmd) {
		return
	}

	writer := cmd.OutOrStdout()
	prefix := styleAccent(writer, "[luu]")
	fmt.Fprintf(writer, "%s %s %s\n", prefix, styleMuted(writer, "workspace:"), workspaceName)
	fmt.Fprintf(writer, "%s %s %s\n", prefix, styleMuted(writer, "task:"), styleAccent(writer, taskName))

	if len(steps) == 1 {
		fmt.Fprintf(writer, "%s %s %s\n", prefix, styleMuted(writer, "running:"), styleCommand(writer, shellStyleCommand(steps[0])))
		fmt.Fprintln(writer)
		return
	}

	fmt.Fprintf(writer, "%s %s %d %s\n", prefix, styleMuted(writer, "resolved:"), len(steps), styleMuted(writer, "steps"))
	fmt.Fprintln(writer)
}

func workflowWorkspaceName(state workspace.Workspace, cfg *config.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.Project.Name) != "" {
		return strings.TrimSpace(cfg.Project.Name)
	}
	return filepath.Base(state.RootPath)
}

func loadWorkflowContext(deps workflowCommandDeps) (workspace.Workspace, *config.Config, error) {
	state, err := deps.detectWorkspace("")
	if err != nil {
		return workspace.Workspace{}, nil, fmt.Errorf("failed to detect workspace: %w. Next: run the command from a repository directory", err)
	}

	if !state.HasLuumenConfig {
		return state, &config.Config{}, nil
	}

	cfg, err := deps.loadConfig(state.LuumenConfigPath)
	if err != nil {
		return workspace.Workspace{}, nil, fmt.Errorf("failed to load %s: %w", workspace.LuumenConfigFile, err)
	}
	return state, cfg, nil
}

func resolveDefaultRojoProjectPathForPlan(state workspace.Workspace) (string, bool, error) {
	if !state.HasRojoProject || len(state.RojoProjectPaths) == 0 {
		return "default.project.json", true, nil
	}

	path, err := toRelativeConfigPath(state.RootPath, state.RojoProjectPaths[0])
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve Rojo project path: %w", err)
	}

	return path, false, nil
}

func missingDefaultRojoProjectError(commandName string, state workspace.Workspace) error {
	return fmt.Errorf("no Rojo project file (*.project.json) was found in %s, so the default %q task cannot run. Next: add default.project.json or define tasks.%s in %s", state.RootPath, commandName, commandName, workspace.LuumenConfigFile)
}

func missingWorkflowTaskError(taskName string) error {
	return fmt.Errorf("task %q is not defined in tasks\n[next] Add tasks.%s to %s", taskName, taskName, workspace.LuumenConfigFile)
}
