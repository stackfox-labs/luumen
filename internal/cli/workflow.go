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
		loadConfig:      config.Load,
		rokitInstaller:  tools.NewRokit(nil, ""),
	}
}

func ensureWorkflowDeps(deps workflowCommandDeps) workflowCommandDeps {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.Load
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

			commands, resolveErr := resolveBuiltInCommandSequence(commandName, cfg, state)
			if len(commands) > 0 {
				printWorkflowPlan(cmd, workflowWorkspaceName(state, cfg), commandName, commands)
			}
			if resolveErr != nil {
				return resolveErr
			}

			runner := workflowTaskRunner(cmd, deps, commandName, state, commands)
			syntheticCfg := &config.Config{
				Tasks: map[string]config.TaskValue{
					"__builtin_" + commandName: config.NewTaskValue(commands...),
				},
			}
			if err := runner.RunNamedTask(cmd.Context(), "__builtin_"+commandName, syntheticCfg, workflowRunOptions(cmd, state.RootPath)); err != nil {
				return fmt.Errorf("command %q failed: %w", commandName, err)
			}
			return nil
		},
	}

	return cmd
}

func workflowTaskRunner(cmd *cobra.Command, deps workflowCommandDeps, commandName string, state workspace.Workspace, commands []string) taskRunner {
	if deps.taskRunner != nil {
		return deps.taskRunner
	}

	baseRunner := newSelfHealingShellRunner(cmd, commandName, state, deps.rokitInstaller)
	shellRunner := newWorkflowStepShellRunner(cmd, baseRunner, len(commands))
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

func resolveBuiltInCommandSequence(commandName string, cfg *config.Config, state workspace.Workspace) ([]string, error) {
	if cfg != nil {
		if commandValue, ok := cfg.Commands[commandName]; ok {
			if len(commandValue.Commands) > 0 {
				return append([]string(nil), commandValue.Commands...), nil
			}
		}
	}

	switch commandName {
	case "dev":
		projectPath, usedFallback, err := resolveDefaultRojoProjectPathForPlan(state)
		if err != nil {
			return nil, err
		}
		commands := []string{
			fmt.Sprintf("rojo sourcemap %s --output sourcemap.json", projectPath),
			fmt.Sprintf("rojo serve %s", projectPath),
		}
		if usedFallback {
			return commands, missingDefaultRojoProjectError(commandName, state)
		}
		return commands, nil
	case "build":
		projectPath, usedFallback, err := resolveDefaultRojoProjectPathForPlan(state)
		if err != nil {
			return nil, err
		}
		commands := []string{fmt.Sprintf("rojo build %s --output build.rbxl", projectPath)}
		if usedFallback {
			return commands, missingDefaultRojoProjectError(commandName, state)
		}
		return commands, nil
	default:
		return nil, fmt.Errorf("command %q is not configured. Next: define [commands].%s in %s", commandName, commandName, workspace.LuumenConfigFile)
	}
}

func printWorkflowPlan(cmd *cobra.Command, workspaceName string, commandName string, commands []string) {
	if isQuiet(cmd) {
		return
	}

	writer := cmd.OutOrStdout()
	prefix := styleAccent(writer, "[luu]")
	fmt.Fprintf(writer, "%s %s %s\n", prefix, styleMuted(writer, "workspace:"), workspaceName)
	fmt.Fprintf(writer, "%s %s %s\n", prefix, styleMuted(writer, "command:"), styleAccent(writer, commandName))

	if len(commands) == 1 {
		fmt.Fprintf(writer, "%s %s %s\n", prefix, styleMuted(writer, "running:"), styleCommand(writer, shellStyleCommand(commands[0])))
		fmt.Fprintln(writer)
		return
	}

	fmt.Fprintf(writer, "%s %s %d %s\n", prefix, styleMuted(writer, "resolved:"), len(commands), styleMuted(writer, "steps"))
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

func resolveDefaultRojoProjectPath(state workspace.Workspace) (string, error) {
	if !state.HasRojoProject || len(state.RojoProjectPaths) == 0 {
		return "", fmt.Errorf("default implementation for this command requires a project file (*.project.json) in %s. Next: add default.project.json or define [commands] override", state.RootPath)
	}
	path, err := toRelativeConfigPath(state.RootPath, state.RojoProjectPaths[0])
	if err != nil {
		return "", fmt.Errorf("failed to resolve Rojo project path: %w", err)
	}
	return path, nil
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
	return fmt.Errorf("no Rojo project file (*.project.json) was found in %s, so the default %q implementation cannot run. Next: add default.project.json or define [commands].%s in %s", state.RootPath, commandName, commandName, workspace.LuumenConfigFile)
}

func runSyntheticCommandTask(ctx context.Context, runner taskRunner, commandName string, commands []string, options tasks.RunOptions) error {
	syntheticCfg := &config.Config{
		Tasks: map[string]config.TaskValue{
			"__builtin_" + commandName: config.NewTaskValue(commands...),
		},
	}
	return runner.RunNamedTask(ctx, "__builtin_"+commandName, syntheticCfg, options)
}
