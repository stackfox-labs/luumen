package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"luumen/internal/config"
	"luumen/internal/process"
	"luumen/internal/tasks"
	"luumen/internal/tools"
	"luumen/internal/workspace"
)

type installCommandDeps struct {
	detectWorkspace func(path string) (workspace.Workspace, error)
	loadConfig      func(path string) (*config.Config, error)
	taskRunner      taskRunner
	rokitInstaller  rokitInstaller
	wallyInstaller  wallyInstaller
}

type rokitInstaller interface {
	Install(ctx context.Context, options tools.RunOptions) (process.Result, error)
}

type wallyInstaller interface {
	Install(ctx context.Context, options tools.RunOptions) (process.Result, error)
}

func defaultInstallCommandDeps() installCommandDeps {
	return installCommandDeps{
		detectWorkspace: workspace.Detect,
		loadConfig:      config.LoadTasks,
		rokitInstaller:  tools.NewRokit(nil, ""),
		wallyInstaller:  tools.NewWally(nil, ""),
	}
}

func newInstallCmd(deps installCommandDeps) *cobra.Command {
	if deps.detectWorkspace == nil {
		deps.detectWorkspace = workspace.Detect
	}
	if deps.loadConfig == nil {
		deps.loadConfig = config.LoadTasks
	}
	if deps.rokitInstaller == nil {
		deps.rokitInstaller = tools.NewRokit(nil, "")
	}
	if deps.wallyInstaller == nil {
		deps.wallyInstaller = tools.NewWally(nil, "")
	}

	var toolsOnly bool
	var packagesOnly bool
	var noTools bool
	var noPackages bool

	cmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"i"},
		Short:   "Install repo tools and packages",
		Long: "Install orchestrates Rokit tools and Wally packages based on workspace files. " +
			"By default it installs both when configuration files exist.",
		Example: "luu install\n" +
			"luu install --tools\n" +
			"luu install --packages\n" +
			"luu install --tools --no-tools --packages",
		Args: requireNoPositionalArgs(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			state, err := deps.detectWorkspace("")
			if err != nil {
				return fmt.Errorf("failed to detect workspace: %w. Next: run the command from a repository directory", err)
			}

			if state.HasLuumenConfig {
				cfg, err := deps.loadConfig(state.LuumenConfigPath)
				if err != nil {
					return fmt.Errorf("failed to load %s: %w", workspace.LuumenConfigFile, err)
				}

				if _, ok := cfg.Tasks["install"]; ok {
					statusf(cmd, "Running task: install")

					runner := deps.taskRunner
					if runner == nil {
						runner = tasks.NewEngine(newSelfHealingShellRunner(cmd, "install", state, deps.rokitInstaller), "luu")
					}

					stdout, stderr := commandOutputWriters(cmd)
					if err := runner.RunNamedTask(cmd.Context(), "install", cfg, tasks.RunOptions{
						WorkingDir: state.RootPath,
						Stdout:     stdout,
						Stderr:     stderr,
						Stdin:      cmd.InOrStdin(),
					}); err != nil {
						if errors.Is(err, tasks.ErrTaskNotFound) {
							return err
						}
						return fmt.Errorf("task %q failed: %w", "install", err)
					}
					statusf(cmd, "Task completed: install")
					return nil
				}
			}

			mode := resolveInstallMode(installModeInput{
				ToolsOnly:    toolsOnly,
				PackagesOnly: packagesOnly,
				NoTools:      noTools,
				NoPackages:   noPackages,
			})

			if !mode.InstallTools && !mode.InstallPackages {
				return fmt.Errorf("nothing to install: requested scope disables both tools and packages. Next: remove one of --no-tools or --no-packages")
			}

			statusf(cmd, "Resolving install scope...")

			toolsAvailable := state.HasRokitConfig
			packagesAvailable := state.HasWallyConfig

			if mode.InstallTools && !toolsAvailable && mode.ToolsExplicit {
				return fmt.Errorf("cannot install tools: %s was not found in %s. Next: add rokit.toml or run without --tools", workspace.RokitConfigFile, state.RootPath)
			}
			if mode.InstallPackages && !packagesAvailable && mode.PackagesExplicit {
				return fmt.Errorf("cannot install packages: %s was not found in %s. Next: add wally.toml or run without --packages", workspace.WallyConfigFile, state.RootPath)
			}

			ranTools := false
			ranPackages := false

			if mode.InstallTools && toolsAvailable {
				statusf(cmd, "Installing tools with Rokit...")
				if _, err := deps.rokitInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, state.RootPath)); err != nil {
					if process.IsKind(err, process.ErrorKindNotFound) {
						return fmt.Errorf("failed to install tools: Rokit executable was not found in PATH: %w", err)
					}
					return fmt.Errorf("failed to install tools via Rokit: %w", err)
				}
				successf(cmd, "Tools installed")
				ranTools = true
			}

			if mode.InstallPackages && packagesAvailable {
				statusf(cmd, "Installing packages with Wally...")
				if _, err := deps.wallyInstaller.Install(cmd.Context(), defaultToolRunOptions(cmd, state.RootPath)); err != nil {
					if process.IsKind(err, process.ErrorKindNotFound) {
						return fmt.Errorf("failed to install packages: Wally executable was not found in PATH: %w", err)
					}
					return fmt.Errorf("failed to install packages via Wally: %w", err)
				}
				successf(cmd, "Packages installed")
				ranPackages = true
			}

			if !ranTools && !ranPackages {
				return fmt.Errorf("no installable configuration found: expected %s and/or %s. Next: add config files or run luu init", workspace.RokitConfigFile, workspace.WallyConfigFile)
			}

			successf(cmd, "Install completed")

			return nil
		},
	}

	cmd.Flags().BoolVar(&toolsOnly, "tools", false, "install tools only (unless overridden by --packages or --no-tools)")
	cmd.Flags().BoolVar(&packagesOnly, "packages", false, "install packages only (unless overridden by --tools or --no-packages)")
	cmd.Flags().BoolVar(&noTools, "no-tools", false, "disable tool installation")
	cmd.Flags().BoolVar(&noPackages, "no-packages", false, "disable package installation")

	return cmd
}

type installModeInput struct {
	ToolsOnly    bool
	PackagesOnly bool
	NoTools      bool
	NoPackages   bool
}

type installMode struct {
	InstallTools     bool
	InstallPackages  bool
	ToolsExplicit    bool
	PackagesExplicit bool
}

func resolveInstallMode(input installModeInput) installMode {
	mode := installMode{
		InstallTools:     true,
		InstallPackages:  true,
		ToolsExplicit:    input.ToolsOnly,
		PackagesExplicit: input.PackagesOnly,
	}

	if input.ToolsOnly && !input.PackagesOnly {
		mode.InstallTools = true
		mode.InstallPackages = false
	}
	if input.PackagesOnly && !input.ToolsOnly {
		mode.InstallTools = false
		mode.InstallPackages = true
	}
	if input.ToolsOnly && input.PackagesOnly {
		mode.InstallTools = true
		mode.InstallPackages = true
	}

	if input.NoTools {
		mode.InstallTools = false
	}
	if input.NoPackages {
		mode.InstallPackages = false
	}

	return mode
}
