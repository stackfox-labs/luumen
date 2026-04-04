package cli

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"luumen/internal/config"
	"luumen/internal/process"
	"luumen/internal/tasks"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type rojoWorkflowRunner interface {
	Serve(ctx context.Context, args []string, options tools.RunOptions) (process.Result, error)
	Build(ctx context.Context, args []string, options tools.RunOptions) (process.Result, error)
	Sourcemap(ctx context.Context, args []string, options tools.RunOptions) (process.Result, error)
}

type workflowCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	loadConfig      func(path string) (*config.Config, error)
	taskRunner      taskRunner
	rojoRunner      rojoWorkflowRunner
}

func defaultWorkflowCommandDeps() workflowCommandDeps {
	return workflowCommandDeps{
		detectWorkspace: workspace.Detect,
		loadConfig:      config.Load,
		taskRunner:      tasks.NewEngine(nil, "luu"),
		rojoRunner:      tools.NewRojo(nil, ""),
	}
}

func ensureWorkflowDeps(deps workflowCommandDeps) workflowCommandDeps {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.Load
	}
	if deps.taskRunner == nil {
		deps.taskRunner = tasks.NewEngine(nil, "luu")
	}
	if deps.rojoRunner == nil {
		deps.rojoRunner = tools.NewRojo(nil, "")
	}
	return deps
}

func newServeCmd(deps workflowCommandDeps) *cobra.Command {
	deps = ensureWorkflowDeps(deps)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run Rojo serve or configured override",
		Example: "luu serve\n" +
			"luu serve --quiet",
		Args: requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			statusf(cmd, "Starting Rojo server...")
			state, cfg, err := loadWorkflowContext(deps)
			if err != nil {
				return err
			}
			return executeServe(cmd.Context(), deps, state, cfg, nil, serveRunOptions(cmd, state.RootPath), true)
		},
	}

	return cmd
}

func newSourcemapCmd(deps workflowCommandDeps) *cobra.Command {
	deps = ensureWorkflowDeps(deps)

	cmd := &cobra.Command{
		Use:     "sourcemap",
		Short:   "Run Rojo sourcemap or configured override",
		Example: "luu sourcemap",
		Args:    requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			statusf(cmd, "Generating sourcemap...")
			state, cfg, err := loadWorkflowContext(deps)
			if err != nil {
				return err
			}
			if err := executeSourcemap(cmd.Context(), deps, state, cfg, nil, commandRunOptions(cmd, state.RootPath), true); err != nil {
				return err
			}
			successf(cmd, "Sourcemap generated")
			return nil
		},
	}

	return cmd
}

func newBuildCmd(deps workflowCommandDeps) *cobra.Command {
	deps = ensureWorkflowDeps(deps)

	var plugin bool
	var watch bool

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Run Rojo build or configured override",
		Example: "luu build\n" +
			"luu build --plugin\n" +
			"luu build --watch",
		Args: requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			statusf(cmd, "Building project...")
			state, cfg, err := loadWorkflowContext(deps)
			if err != nil {
				return err
			}

			passThrough := make([]string, 0, 2)
			if plugin {
				passThrough = append(passThrough, "--plugin")
			}
			if watch {
				passThrough = append(passThrough, "--watch")
			}

			if err := executeBuild(cmd.Context(), deps, state, cfg, passThrough, commandRunOptions(cmd, state.RootPath), true); err != nil {
				return err
			}
			successf(cmd, "Build completed")
			return nil
		},
	}

	cmd.Flags().BoolVar(&plugin, "plugin", false, "pass through --plugin to rojo build")
	cmd.Flags().BoolVar(&watch, "watch", false, "pass through --watch to rojo build")
	return cmd
}

func newDevCmd(deps workflowCommandDeps) *cobra.Command {
	deps = ensureWorkflowDeps(deps)

	cmd := &cobra.Command{
		Use:     "dev",
		Short:   "Run development workflow (sourcemap then serve) or configured override",
		Example: "luu dev",
		Args:    requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			statusf(cmd, "Starting development workflow...")
			state, cfg, err := loadWorkflowContext(deps)
			if err != nil {
				return err
			}

			runOptions := commandRunOptions(cmd, state.RootPath)
			overridden, err := executeOverrideIfPresent(cmd.Context(), deps, cfg, "dev", runOptions, state, true)
			if err != nil {
				return err
			}
			if overridden {
				successf(cmd, "Development workflow completed")
				return nil
			}

			if err := executeSourcemap(cmd.Context(), deps, state, cfg, nil, runOptions, false); err != nil {
				return err
			}
			successf(cmd, "Sourcemap generated")
			statusf(cmd, "Starting Rojo server...")
			return executeServe(cmd.Context(), deps, state, cfg, nil, serveRunOptions(cmd, state.RootPath), false)
		},
	}

	return cmd
}

func executeServe(ctx context.Context, deps workflowCommandDeps, state workspace.Workspace, cfg *config.Config, args []string, options tasks.RunOptions, allowDefaultOverrideSkip bool) error {
	overridden, err := executeOverrideIfPresent(ctx, deps, cfg, "serve", options, state, allowDefaultOverrideSkip)
	if err != nil {
		return err
	}
	if overridden {
		return nil
	}

	projectPath, err := resolveDefaultRojoProjectPath(state)
	if err != nil {
		return err
	}

	finalArgs := append([]string{projectPath}, args...)
	if _, err := deps.rojoRunner.Serve(ctx, finalArgs, toolsRunOptions(options)); err != nil {
		return formatRojoError("serve", err)
	}
	return nil
}

func executeSourcemap(ctx context.Context, deps workflowCommandDeps, state workspace.Workspace, cfg *config.Config, args []string, options tasks.RunOptions, allowDefaultOverrideSkip bool) error {
	overridden, err := executeOverrideIfPresent(ctx, deps, cfg, "sourcemap", options, state, allowDefaultOverrideSkip)
	if err != nil {
		return err
	}
	if overridden {
		return nil
	}

	projectPath, err := resolveDefaultRojoProjectPath(state)
	if err != nil {
		return err
	}

	finalArgs := append([]string{projectPath}, args...)
	if _, err := deps.rojoRunner.Sourcemap(ctx, finalArgs, toolsRunOptions(options)); err != nil {
		return formatRojoError("sourcemap", err)
	}
	return nil
}

func executeBuild(ctx context.Context, deps workflowCommandDeps, state workspace.Workspace, cfg *config.Config, args []string, options tasks.RunOptions, allowDefaultOverrideSkip bool) error {
	overridden, err := executeOverrideIfPresent(ctx, deps, cfg, "build", options, state, allowDefaultOverrideSkip)
	if err != nil {
		return err
	}
	if overridden {
		return nil
	}

	projectPath, err := resolveDefaultRojoProjectPath(state)
	if err != nil {
		return err
	}

	finalArgs := append([]string{projectPath, "--output", "build.rbxl"}, args...)
	if _, err := deps.rojoRunner.Build(ctx, finalArgs, toolsRunOptions(options)); err != nil {
		return formatRojoError("build", err)
	}
	return nil
}

func executeOverrideIfPresent(ctx context.Context, deps workflowCommandDeps, cfg *config.Config, commandName string, options tasks.RunOptions, state workspace.Workspace, allowDefaultOverrideSkip bool) (bool, error) {
	if cfg == nil || len(cfg.Commands) == 0 {
		return false, nil
	}

	commandValue, ok := cfg.Commands[commandName]
	if !ok {
		return false, nil
	}

	if allowDefaultOverrideSkip && isSkippableDefaultOverride(commandName, commandValue, state) {
		return false, nil
	}

	syntheticName := "__builtin_" + commandName
	taskMap := make(map[string]config.TaskValue, len(cfg.Tasks)+1)
	for name, taskValue := range cfg.Tasks {
		taskMap[name] = taskValue
	}
	taskMap[syntheticName] = commandValue

	syntheticCfg := &config.Config{Tasks: taskMap}
	if err := deps.taskRunner.RunNamedTask(ctx, syntheticName, syntheticCfg, options); err != nil {
		return true, fmt.Errorf("%s command override failed: %w", commandName, err)
	}
	return true, nil
}

func isSkippableDefaultOverride(commandName string, commandValue config.TaskValue, state workspace.Workspace) bool {
	projectPath, err := resolveDefaultRojoProjectPath(state)
	if err != nil {
		return false
	}

	switch commandName {
	case "serve":
		return matchesCommandSequence(commandValue.Commands, []string{fmt.Sprintf("rojo serve %s", projectPath)})
	case "build":
		return matchesCommandSequence(commandValue.Commands, []string{fmt.Sprintf("rojo build %s --output build.rbxl", projectPath)})
	case "sourcemap":
		return matchesCommandSequence(commandValue.Commands, []string{fmt.Sprintf("rojo sourcemap %s --output sourcemap.json", projectPath)})
	case "dev":
		return matchesCommandSequence(commandValue.Commands, []string{"luu sourcemap", fmt.Sprintf("rojo serve %s", projectPath)}) ||
			matchesCommandSequence(commandValue.Commands, []string{"luu sourcemap", "luu serve"})
	default:
		return false
	}
}

func matchesCommandSequence(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}

	for index := range expected {
		if strings.TrimSpace(actual[index]) != expected[index] {
			return false
		}
	}

	return true
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
		return "", fmt.Errorf("default Rojo command requires a project file (*.project.json) in %s. Next: add default.project.json or configure [commands] override", state.RootPath)
	}
	path, err := toRelativeConfigPath(state.RootPath, state.RojoProjectPaths[0])
	if err != nil {
		return "", fmt.Errorf("failed to resolve Rojo project path: %w", err)
	}
	return path, nil
}

func formatRojoError(command string, err error) error {
	if process.IsKind(err, process.ErrorKindNotFound) {
		return fmt.Errorf("failed to run rojo %s: Rojo executable was not found in PATH: %w", command, err)
	}
	return fmt.Errorf("failed to run rojo %s: %w", command, err)
}

func commandRunOptions(cmd *cobra.Command, workingDir string) tasks.RunOptions {
	stdout, stderr := commandOutputWriters(cmd)
	return tasks.RunOptions{
		WorkingDir: workingDir,
		Stdout:     stdout,
		Stderr:     stderr,
		Stdin:      cmd.InOrStdin(),
	}
}

func serveRunOptions(cmd *cobra.Command, workingDir string) tasks.RunOptions {
	if isVerbose(cmd) || isQuiet(cmd) {
		return commandRunOptions(cmd, workingDir)
	}

	readyWriter := newRojoServeReadyWriter(cmd.OutOrStdout())
	return tasks.RunOptions{
		WorkingDir: workingDir,
		Stdout:     readyWriter,
		Stderr:     readyWriter,
		Stdin:      cmd.InOrStdin(),
	}
}

type rojoServeReadyWriter struct {
	writer    io.Writer
	buffer    strings.Builder
	announced bool
	mu        sync.Mutex
}

var serveURLPattern = regexp.MustCompile(`https?://[^\s]+`)
var servePortPattern = regexp.MustCompile(`(?i)\bport:\s*([0-9]{2,5})\b`)
var serveAddressPattern = regexp.MustCompile(`(?i)\baddress:\s*([^\r\n]+)`)

func newRojoServeReadyWriter(writer io.Writer) *rojoServeReadyWriter {
	return &rojoServeReadyWriter{writer: writer}
}

func (w *rojoServeReadyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.announced {
		return len(p), nil
	}

	_, _ = w.buffer.Write(p)
	content := w.buffer.String()
	lower := strings.ToLower(content)

	if url := extractServeURL(content); url != "" {
		fmt.Fprintf(w.writer, "%s Rojo server started at %s (Ctrl+C to stop)\n", successPrefix(w.writer), url)
		w.announced = true
		w.buffer.Reset()
		return len(p), nil
	}

	if strings.Contains(lower, "rojo server listening:") {
		if host, port, ok := extractServeHostAndPort(content); ok {
			fmt.Fprintf(w.writer, "%s Rojo server started at http://%s:%s/ (Ctrl+C to stop)\n", successPrefix(w.writer), host, port)
			w.announced = true
			w.buffer.Reset()
			return len(p), nil
		}

		if strings.Contains(lower, "visit http://") {
			fmt.Fprintf(w.writer, "%s Rojo server started (Ctrl+C to stop)\n", successPrefix(w.writer))
			w.announced = true
			w.buffer.Reset()
			return len(p), nil
		}
	}

	if strings.Contains(lower, "visit http://") {
		fmt.Fprintf(w.writer, "%s Rojo server started (Ctrl+C to stop)\n", successPrefix(w.writer))
		w.announced = true
		w.buffer.Reset()
		return len(p), nil
	}

	if w.buffer.Len() > 8192 {
		content := w.buffer.String()
		tail := content[len(content)-2048:]
		w.buffer.Reset()
		_, _ = w.buffer.WriteString(tail)
	}

	return len(p), nil
}

func extractServeURL(content string) string {
	match := serveURLPattern.FindString(content)
	return strings.TrimSpace(match)
}

func extractServeHostAndPort(content string) (string, string, bool) {
	portMatch := servePortPattern.FindStringSubmatch(content)
	if len(portMatch) < 2 {
		return "", "", false
	}

	host := "localhost"
	addressMatch := serveAddressPattern.FindStringSubmatch(content)
	if len(addressMatch) >= 2 {
		candidate := strings.TrimSpace(addressMatch[1])
		if candidate != "" {
			host = candidate
		}
	}

	return host, strings.TrimSpace(portMatch[1]), true
}

func toolsRunOptions(options tasks.RunOptions) tools.RunOptions {
	return tools.RunOptions{
		WorkingDir: options.WorkingDir,
		Env:        options.Env,
		Stdout:     options.Stdout,
		Stderr:     options.Stderr,
		Stdin:      options.Stdin,
	}
}
